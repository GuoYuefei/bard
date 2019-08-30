package bard

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

const (
	BUFSIZE     = 61 * 1024
	MESSAGESIZE = 10
	WriteBUFSIZE = 32 * 1024
	ReadBUFSIZE = 63 * 1024
)

var ErrorChanelClose = errors.New("chanel is closed")

type UdpMessage struct {
	dst *net.UDPAddr
	Data []byte
	//*Message
}

// 不知道对不对还未验证
func (u *UdpMessage) Read(b []byte) (n int, err error) {
	return bytes.NewReader(u.Data).Read(b)
}

func (u *UdpMessage) Write(b []byte) (n int, err error) {
	write := new(bytes.Buffer)
	// todo 可能有错
	i, err := write.Write(b)
	u.Data = write.Bytes()

	return i, err
}

func (u *UdpMessage) GetDst() *net.UDPAddr {
	return u.dst
}

// 用于记录一对udp通道
type Packet struct {
	Packet  *net.UDPConn
	timeout time.Duration
	Client  *net.UDPAddr
	Servers map[string]*net.UDPAddr // 远程主机应该有一个列表 客户端第一次发给远程主机的时候将其记录进Servers列表
	Socks   *Conn						// 插件类型一般是由Socks带入Packet的
	message chan *UdpMessage
	Frag    uint8 // udp分段
	buf map[string] []byte
}

func (p *Packet) GetDeadline() time.Time {
	deadline := time.Time{}
	if p.timeout > 0 {
		deadline = time.Now().Add(p.timeout)
	}
	return deadline
}

func (p *Packet) SetTimeout(second int) {
	p.timeout = time.Duration(second) * time.Second
	_ = p.SetDeadline(p.GetDeadline())
}

func NewPacket(conn *Conn, p *net.UDPConn, cport int) (*Packet, error) {
	var err error
	caddr := conn.RemoteAddr() // socks5远程连接地址就是客户端地址
	packet := &Packet{timeout: 10}
	packet.Frag = 0
	packet.Socks = conn
	packet.Packet = p
	packet.Servers = make(map[string]*net.UDPAddr)
	packet.message = make(chan *UdpMessage, MESSAGESIZE)
	packet.buf = make(map[string] []byte)

	if addr, ok := caddr.(*net.TCPAddr); ok {
		packet.Client, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", addr.IP, cport))
	} else {
		err = fmt.Errorf("net.Addr must be tcpaddr ")
	}

	return packet, err
}

// 将chan message的消息按指定位子请求出去
func (p *Packet) Request() (n int, err error) {
	message, ok := <-p.message
	if !ok {
		// 如果chanel已经关闭
		return 0, ErrorChanelClose
	}
	i, err := p.WriteTo(message.Data, message.dst)

	if err != nil {
		Deb.Println(err)
	}

	if i != len(message.Data) {
		Deb.Println("io 不完全")
		err = nil // io不完全的话就当没错
	}

	Deb.Printf("len of data:%d\tForwarding data： %v\nThe forwarding address is %v\n", i, message.Data, message.dst)

	return i, err
}

func (p *Packet) Listen() error {

	var message = &UdpMessage{}
	var buf = make([]byte, ReadBUFSIZE)

	nr, addr, err := p.ReadFrom(buf)
	if err != nil {
		Deb.Println(err)
		return err
	}

	var uaddr *net.UDPAddr

	uaddr, _ = addr.(*net.UDPAddr)

	Deb.Println("the udp message send from the remote:", addr.String())
	Deb.Printf("p.client.string=%s\n", p.Client.String())
	Deb.Println("the len of p.servers:", len(p.Servers))

	// todo 当代理服务器在远程主机上时，QQ需要只会验证客户端IP。而无需验证端口。也就是说请求是客户端发来的端口信息也并无软用 不过这样写之后可以兼容正规socks5协议
	if p.Client.IP.String() == uaddr.IP.String() {
		if p.Client.String() != uaddr.String() {
			p.Client = uaddr //改变p.client的port
		}
		// endtodo 1 消息来自客户端就需要进行解密 解密就可能存在多余数据或者少数据的情况，这时候就需要用p.buf将其存起来
		if p.Socks.plugin != nil {
			//_, nr = p.Socks.plugin.Ornament(buf[0:nr], RECEIVE)
			buf, nr = p.Decode(buf[0:nr], addr)
			if nr == 0 {
				return nil
			}
		}
		reader := bufio.NewReader(bytes.NewReader(buf[0:nr]))
		// 客户端发来的消息
		udpreqs, err := NewUDPReqSFromReader(reader, addr)
		if err != nil {
			Deb.Println(err)
			return err
		}
		//fmt.Println(udpreqs)
		// 如果原本远程servers列表存在该远程主机，就直接提取
		if dst, ok := p.Servers[udpreqs.String()]; ok {
			message.dst = dst
			message.Data = udpreqs.Data.Bytes()
			p.message <- message
			return err
		} else {
			// 原本列表中不存在
			dst, err := net.ResolveUDPAddr(udpreqs.Network(), udpreqs.String())
			//fmt.Println(dst)
			if err != nil {
				Deb.Println("this is Listen() ", err)
				return err
			}
			p.Servers[udpreqs.String()] = dst
			message.dst = dst
			message.Data = udpreqs.Data.Bytes()
			p.message <- message
		}
		// 客户端发来的消息 end
	} else {
		// 远程主机or其他

		if src, ok := p.Servers[addr.String()]; ok {
			reader := bufio.NewReader(bytes.NewReader(buf[0:nr]))
			// 当 远程主机
			//srcip := src.IP.To4()
			//srcipType := IPV4
			//if srcip == nil {
			//	srcip = src.IP.To16()
			//	srcipType = IPV6
			//	if srcip == nil {
			//		return errors.New("Address error IP cannot be parsed into version 4 or 6 ")
			//	}
			//}

			srcip, srcipType, err := IpToBytes(src.IP)
			if err != nil {
				return err
			}

			message.dst = p.Client
			Deb.Printf("Processing UDP messages from remote host %s", src)
			// 如果发送的消息来自ip和port记录在servers中了，那么就执行转发.否则丢弃
			_, err = Pipe(message, reader, func(data []byte) ([]byte, int) {
				head := append([]byte{0x00, 0x00, p.Frag, srcipType}, srcip...)
				head = append(head, uint8(src.Port>>8), uint8(src.Port))
				data = append(head, data...)

				// endtodo 2 数据要加密 对于远程主机发来的消息就应该加密发送给客户端 虽然可能插件不一定存在加密过程
				if p.Socks.plugin != nil {
					//data, _ = p.Socks.plugin.Ornament(data, SEND)
					data, _ = p.Encode(data)
				}

				return data, len(data)
			})

			if err != nil && err != io.ErrShortWrite {
				Deb.Println(err)
				return err
			}

			p.message <- message
		} else {
			// 若是无记录主机就丢弃信息
			Deb.Printf("Discard UDP messages from remote host %s\n", addr.String())
		}

	}
	return err

}

func (p *Packet) AddServer(key string, server *net.UDPAddr) {
	p.Servers[key] = server
}

// 监听 但是固定双方通道的两头的主机地址
// 客户端使用
// 因为是客户端使用，所以Packet中的p.Servers其实是本地的服务器， p.Client客户端是指远程代理服务器（固定）
func (p *Packet) ListenToFixedTarget(serverKey string) error {
	var message = &UdpMessage{}
	var buf = make([]byte, BUFSIZE)
	nr, addr, err := p.ReadFrom(buf)

	if err != nil {
		Deb.Println(err)
		return err
	}
	var uaddr *net.UDPAddr
	uaddr, _ = addr.(*net.UDPAddr)
	if uaddr.String() == p.Client.String() {
		// 是远程代理服务器发来的消息 此时远程代理服务器在客户端看来它就是一个客户端
		// endtodo 3 远程代理服务器发来的消息需要解密
		if p.Socks.plugin != nil {
			// 这里的RECEIVE都是本软件的客户端和服务器端的相对关系，与其他软件不相关
			//_, nr = p.Socks.plugin.Ornament(buf[0:nr], RECEIVE)
			buf, nr = p.Decode(buf[0:nr], addr)
			if nr == 0 {
				return nil
			}
		}
		if src, ok := p.Servers[serverKey]; ok {
			message.dst = src
			message.Data = buf[0:nr]
			p.message <- message
		} else {
			Deb.SetPrefix(LOG_EXCEPTION)
			Deb.Printf("不存在%s这个key值的server", serverKey)
			Deb.SetPrefix(LOG_INFO)
		}

	} else if src, ok := p.Servers[serverKey]; ok {
		// 这里的server中应该只有一个值，且是客户端的客户端的地址
		if src.IP.String() == uaddr.IP.String() {
			// 如果serverKey存在且与读取到的udp消息IP相同。！！。
			// 以下if 为了兼容QQudp会更换port来进行udp连接
			if src.String() != uaddr.String() {
				p.Servers[serverKey] = uaddr
			}

			message.dst = p.Client
			Deb.Printf("Processing UDP messages from client host %s", src)

			// endtodo 4 客户端来的消息应该要加密
			if p.Socks.plugin != nil {
				// 这里的RECEIVE都是本软件的客户端和服务器端的相对关系，与其他软件不相关
				//_, nr = p.Socks.plugin.Ornament(buf[0:nr], SEND)
				buf, nr = p.Encode(buf[0:nr])
			}
			message.Data = buf[0:nr]

			p.message <- message
			// 要发送给远程代理服务器，需要加密
		} else {
			// 若是无记录主机就丢弃信息
			Deb.Printf("Discard UDP messages from client host %s\n", addr.String())
		}
	} else {
		Deb.SetPrefix(LOG_EXCEPTION)
		err = fmt.Errorf("不存在%s这个key值的server", serverKey)
		Deb.Println(err)
		Deb.SetPrefix(LOG_INFO)
	}
	return err
}

// @describe 根据p配置内容决定要不要加密， 如果不加密就原样输出
// @param []byte 原文
// @return res []byte 密文
// @return n int 长度
func (p *Packet) Encode(src []byte) (res []byte, n int) {
	res, n = src, len(src)
	if p.Socks.plugin == nil {
		if p.Socks.protocol != nil {
			res, n = p.Socks.protocol.WriteDo(res)
		}
		return
	}
	// 当plugin存在

	if p.Socks.protocol == nil {
		panic("packet encode: Subprotocols must be configured in the presence of plug-ins")
	}
	// 这时候是正常情况
	res, n = p.Socks.plugin.AntiSniffing(res, SEND)
	res, n = p.Socks.protocol.WriteDo(res[0:n])
	return
}

// endtodo 解密 解密就可能存在多余数据或者少数据的情况，这时候就需要用p.buf将其存起来 当没有plugin时，就不会用到p.buf
// @describe 根据p配置内容决定要不要解密， 如果不解密就原样输出
// @param []byte 密文
// @param net.Addr 密文来源
// @return res []byte 原文
// @return n int 长度
func (p *Packet) Decode(src []byte, addr net.Addr) (res []byte, n int) {
	var temp []byte
	temp, n = src, len(src)

	if p.Socks.plugin == nil {
		reader := bytes.NewReader(temp)
		if p.Socks.protocol != nil {
			// 在无plugin下子协议并无软用
			_, _ = p.Socks.protocol.ReadDo(reader)
		}
		return
	}

	if p.Socks.protocol == nil {
		// 在有plugin下，必须配置子协议
		panic("packet decode: Subprotocols must be configured in the presence of plug-ins")
	}

	if v, ok := p.buf[addr.String()]; ok {
		temp = append(v, temp...)
	}
	reader := bytes.NewReader(temp)

	do, n := p.Socks.protocol.ReadDo(reader)
	//fmt.Println(do, n)
	dolen := len(do)

	if dolen+n > len(temp) {
		// temp 如果超过长度了应该不返回等下次返回
		p.buf[addr.String()] = temp
		return nil, 0
	}

	// 如果没超过长度就正常返回
	p.buf[addr.String()] = temp[dolen+n:]

	res, n = p.Socks.plugin.AntiSniffing(temp[dolen:dolen+n], RECEIVE)

	return res, n
}

func (p *Packet) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	n, addr, err = p.Packet.ReadFrom(b)

	if err != nil {
		return
	}

	_ = p.SetDeadline(p.GetDeadline())
	return
}

func (p *Packet) WriteTo(b []byte, addr net.Addr) (n int, err error) {

	n, err = p.Packet.WriteTo(b, addr)
	_ = p.SetDeadline(p.GetDeadline())
	return
}

func (p *Packet) Close() error {
	var err error
	err1 := p.Packet.Close()

	err2 := p.Socks.Close()

	if err1 != nil {
		err = err1
	}
	if err2 != nil {
		err = err2
	}
	// todo chan 应该要关闭
	//close(p.message)

	return err
}

func (p *Packet) LocalAddr() net.Addr {
	return p.Packet.LocalAddr()
}

func (p *Packet) SetDeadline(t time.Time) error {
	err := p.Packet.SetDeadline(t)
	e := p.Socks.SetDeadline(t)

	if err != nil && e != nil {
		err = fmt.Errorf("packet set deadline error: %v, and conn set deadline error: %v", err, e)
	} else if e != nil && err == nil {
		err = e
	}
	if err != nil {
		Slog.Printf("Packet set deadline error: %v", err)
	}
	return err
}

func (p *Packet) SetReadDeadline(t time.Time) error {
	return p.Packet.SetReadDeadline(t)
}

func (p *Packet) SetWriteDeadline(t time.Time) error {
	return p.Packet.SetWriteDeadline(t)
}
