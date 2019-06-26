package bard

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

const (
	BUFSIZE = 32 * 1024
	MESSAGESIZE = 10
)

type UdpMessage struct {
	dst *net.UDPAddr
	Data []byte
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

// 用于记录一对udp通道（自个儿取的名）
type Packet struct {
	Packet net.PacketConn
	Client *net.UDPAddr
	Servers map[string] *net.UDPAddr			// 远程主机应该有一个列表 客户端第一次发给远程主机的时候将其记录进Servers列表
	Socks net.Conn
	message chan *UdpMessage
	Frag uint8
}

func NewPacket(conn net.Conn, p net.PacketConn,cport int) (*Packet, error) {
	var err error
	caddr := conn.RemoteAddr()
	packet := &Packet{}
	packet.Frag = 0
	packet.Socks = conn
	packet.Packet = p
	packet.Servers = make(map[string] *net.UDPAddr)
	packet.message = make(chan *UdpMessage, MESSAGESIZE)

	if addr, ok := caddr.(*net.TCPAddr); ok {
		packet.Client, err = net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", addr.IP, cport))
	} else {
		err = fmt.Errorf("net.Addr must be tcpaddr ")
	}

	return packet, err
}

// 将chan message的消息按指定位子请求出去
func (p *Packet) Request() (n int, err error) {
	message := <- p.message
	i, err := p.WriteTo(message.Data, message.dst)

	if err != nil {
		log.Println(err)
	}

	if i != len(message.Data) {
		log.Println("io 不完全")
	}

		fmt.Printf("len of data:%d Forwarding data： %v\nThe forwarding address is %v\n", i, message.Data, message.dst)

	return i, err
}

func (p *Packet) Listen() {

	var message = &UdpMessage{}
	var buf = make([]byte, BUFSIZE)

	nr, addr, err := p.ReadFrom(buf)

	//var uaddr *net.UDPAddr
	//
	//uaddr, _ = addr.(*net.UDPAddr)



	if err != nil {
		log.Println(err)
		return
	}
	reader := bufio.NewReader(bytes.NewReader(buf[0:nr]))
	fmt.Println("the message send from the remote:", addr.String())
	fmt.Println("the len of p.servers:", len(p.Servers))
	fmt.Printf("p.client.string=%s\n",p.Client.String())
	if p.Client.String() == addr.String()  {
		//if p.Client.String() != uaddr.String() {
		//	p.Client = uaddr			//改变p.client的port
		//}
		// 客户端发来的消息
		udpreqs, err := NewUDPReqSFromReader(reader, addr)
		if err != nil {
			log.Println(err)
			return
		}
		//fmt.Printf("qq发来的frag: %v", udpreqs.Frag)
		// 如果原本远程servers列表存在该远程主机，就直接提取
		if dst, ok := p.Servers[udpreqs.String()]; ok {
			message.dst = dst
			message.Data = udpreqs.Data.Bytes()
			p.message <- message
			return
		} else {
			// 原本列表中不存在
			dst, err := net.ResolveUDPAddr(udpreqs.Network(), udpreqs.String())
			//fmt.Println(dst)
			if err != nil {
				log.Println("这里是Listen() ",err)
				return
			}
			p.Servers[udpreqs.String()] = dst
			message.dst = dst
			message.Data = udpreqs.Data.Bytes()
			p.message <- message
			return
		}
		// 客户端发来的消息 end
	} else {
		// 远程主机or其他

		if src, ok := p.Servers[addr.String()]; ok {
			// 当 远程主机

			message.dst = p.Client
			fmt.Println(src)
			// 如果发送的消息来自ip和port记录在servers中了，那么就执行转发.否则丢弃
			_, err := Pipe(message, reader, func(data []byte) ([]byte, int) {
				head := append([]byte{0x00, 0x00, p.Frag, 0x01}, src.IP.To4()...)
				head = append(head, uint8(src.Port>>8), uint8(src.Port))
				data = append(head, data...)
				return data, len(data)
			})

			if err != nil && err != io.ErrShortWrite {
				log.Println(err)
				return
			}

			//fmt.Println(len(message.Data),message.Data)

			p.message <- message
			return
		} else {
			// 若是无记录主机就丢弃信息
			fmt.Println("丢弃来自主机", addr.String(), "的信息")
			return
		}

	}


}




func (p *Packet) ReadFrom(b []byte) (n int, addr net.Addr, err error) {

	return p.Packet.ReadFrom(b)
}

func (p *Packet) WriteTo(b []byte, addr net.Addr) (n int, err error) {
	return p.Packet.WriteTo(b, addr)
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

	return err
}

func (p *Packet) LocalAddr() net.Addr {
	return p.Packet.LocalAddr()
}

func (p *Packet) SetDeadline(t time.Time) error {
	return p.Packet.SetDeadline(t)
}

func (p *Packet) SetReadDeadline(t time.Time) error {
	return p.Packet.SetReadDeadline(t)
}

func (p *Packet) SetWriteDeadline(t time.Time) error {
	return p.Packet.SetWriteDeadline(t)
}




