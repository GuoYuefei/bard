package bard

import (
	"bufio"
	"bytes"
	"fmt"
	"errors"
	"io"
	"net"
	"time"
)

const (
	BUFSIZE = 32 * 1024
	MESSAGESIZE = 10

)
var ErrorChanelClose = errors.New("chanel is closed")

type UdpMessage struct {
	dst *net.UDPAddr
	//Data []byte
	*Message
}

//// 不知道对不对还未验证
//func (u *UdpMessage) Read(b []byte) (n int, err error) {
//	return bytes.NewReader(u.Data).Read(b)
//}
//
//func (u *UdpMessage) Write(b []byte) (n int, err error) {
//	write := new(bytes.Buffer)
//	// todo 可能有错
//	i, err := write.Write(b)
//	u.Data = write.Bytes()
//
//	return i, err
//}

func (u *UdpMessage) GetDst() *net.UDPAddr {
	return u.dst
}

// 用于记录一对udp通道
type Packet struct {
	Packet *net.UDPConn
	timeout time.Duration
	Client *net.UDPAddr
	Servers map[string] *net.UDPAddr			// 远程主机应该有一个列表 客户端第一次发给远程主机的时候将其记录进Servers列表
	Socks *Conn
	message chan *UdpMessage
	Frag uint8									// udp分段
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
	caddr := conn.RemoteAddr()				// socks5远程连接地址就是客户端地址
	packet := &Packet{timeout: 10}
	packet.Frag = 0
	packet.Socks = conn
	packet.Packet = p
	packet.Servers = make(map[string] *net.UDPAddr)
	packet.message = make(chan *UdpMessage, MESSAGESIZE)

	if addr, ok := caddr.(*net.TCPAddr); ok {
		packet.Client, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", addr.IP, cport))
	} else {
		err = fmt.Errorf("net.Addr must be tcpaddr ")
	}

	return packet, err
}

// 将chan message的消息按指定位子请求出去
func (p *Packet) Request() (n int, err error) {
	message, ok := <- p.message
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
		err = nil			// io不完全的话就当没错
	}

	Deb.Printf("len of data:%d\tForwarding data： %v\nThe forwarding address is %v\n", i, message.Data, message.dst)

	return i, err
}

func (p *Packet) Listen() error {

	var message = &UdpMessage{}
	var buf = make([]byte, BUFSIZE)

	nr, addr, err := p.ReadFrom(buf)
	if err != nil {
		Deb.Println(err)
		return err
	}

	var uaddr *net.UDPAddr

	uaddr, _ = addr.(*net.UDPAddr)

	Deb.Println("the udp message send from the remote:", addr.String())
	Deb.Printf("p.client.string=%s\n",p.Client.String())
	Deb.Println("the len of p.servers:", len(p.Servers))

	// todo 当代理服务器在远程主机上时，QQ需要只会验证客户端IP。而无需验证端口。也就是说请求是客户端发来的端口信息也并无软用 不过这样写之后可以兼容正规socks5协议
	if p.Client.IP.String() == uaddr.IP.String()  {
		if p.Client.String() != uaddr.String() {
			p.Client = uaddr			//改变p.client的port
		}
		// endtodo 消息来自客户端就需要进行解密
		if p.Socks.plugin != nil {
			_, nr = p.Socks.plugin.Ornament(buf[0:nr], RECEIVE)
		}
		reader := bufio.NewReader(bytes.NewReader(buf[0:nr]))
		// 客户端发来的消息
		udpreqs, err := NewUDPReqSFromReader(reader, addr)
		if err != nil {
			Deb.Println(err)
			return err
		}
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
				Deb.Println("this is Listen() ",err)
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
			srcip := src.IP.To4()
			srcipType := IPV4
			if srcip == nil {
				srcip = src.IP.To16()
				srcipType = IPV6
				if srcip == nil {
					Deb.Println("Address error IP cannot be parsed into version 4 or 6")
					return err
				}
			}

			message.dst = p.Client
			Deb.Printf("Processing UDP messages from remote host %s", src)
			// 如果发送的消息来自ip和port记录在servers中了，那么就执行转发.否则丢弃
			_, err := Pipe(message, reader, func(data []byte) ([]byte, int) {
				head := append([]byte{0x00, 0x00, p.Frag, srcipType}, srcip...)
				head = append(head, uint8(src.Port>>8), uint8(src.Port))
				data = append(head, data...)

				// endtodo 数据要加密 对于远程主机发来的消息就应该加密发送给客户端 虽然可能插件不一定存在加密过程
				if p.Socks.plugin != nil {
					data, _ = p.Socks.plugin.Ornament(data, SEND)
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


func (p *Packet) SetServers(key string, server) {

}

func (p *Packet) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	n, addr, err = p.Packet.ReadFrom(b)

	if err != nil {
		return
	}


	// endtodo 不加入处理混淆  udp设计时不加入混淆，但是加入加密
	// todo 处理加密
	plugin := p.Socks.plugin

	if plugin != nil {
		uaddr, _ := addr.(*net.UDPAddr)
		fmt.Println("ip: -------- ", uaddr)
		if uaddr.IP.String() == p.Client.IP.String() {
			// 如果是客户端发来的消息就做处理
			//_, n = plugin.Camouflage(b[0:n], RECEIVE)
			_, n = plugin.AntiSniffing(b[0:n], RECEIVE)
		}
	}

	_ = p.SetDeadline(p.GetDeadline())
	return
}

func (p *Packet) WriteTo(b []byte, addr net.Addr) (n int, err error) {

	// todo 处理加密
	plugin := p.Socks.plugin
	if plugin != nil {
		uaddr, _ := addr.(*net.UDPAddr)

		if uaddr.IP.String() == p.Client.IP.String() {
			// 如果是发送给客户端的就进行加密等动作
			b, n = plugin.AntiSniffing(b, SEND)
		}
	}

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




