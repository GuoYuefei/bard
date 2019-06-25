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
	BUFSIZE = 4096
)

type UdpMessage struct {
	dst *net.UDPAddr
	Data []byte
}

// 不知道对不对还未验证
func (u *UdpMessage) Read(b []byte) (n int, err error) {
	return bytes.NewBuffer(b).Write(u.Data)
}

func (u *UdpMessage) Write(b []byte) (n int, err error) {
	write := new(bytes.Buffer)
	// todo 可能有错
	u.Data = write.Bytes()
	return write.Write(b)
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
}

// todo CADDR 应该有socks给出
func NewPacket(p net.PacketConn, caddr net.Addr,cport int) (*Packet, error) {
	var err error
	packet := &Packet{}
	packet.Servers = make(map[string] *net.UDPAddr)

	packet.message = make(chan *UdpMessage, 1)			//暂且一个
	packet.Packet = p
	if addr, ok := caddr.(*net.TCPAddr); ok {
		packet.Client, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", addr.IP, cport))
	} else {
		err = fmt.Errorf("net.Addr must be tcpaddr ")
	}

	return packet, err
}

func Request() (n int, err error) {
	return 1, fmt.Errorf("")
}

func (p *Packet) Listen() {

	var message = &UdpMessage{}
	var buf = make([]byte, BUFSIZE)

	nr, addr, err := p.Packet.ReadFrom(buf)

	if err != nil {
		log.Println(err)
		return
	}
	reader := bufio.NewReader(bytes.NewReader(buf[0:nr]))

	if p.Client.String() == addr.String()  {
		// 客户端发来的消息
		udpreqs, err := NewUDPReqSFromReader(reader, addr)
		if err != nil {
			log.Println(err)
			return
		}
		// 如果原本远程servers列表存在该远程主机，就直接提取
		if dst, ok := p.Servers[udpreqs.String()]; ok {
			message.dst = dst
			message.Data = udpreqs.Data.Bytes()
			p.message <- message
			return
		} else {
			// 原本列表中不存在
			dst, err := net.ResolveUDPAddr("udp", udpreqs.String())
			if err != nil {
				log.Println(err)
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
		if src, ok := p.Servers[addr.String()]; ok {
			message.dst = p.Client
			// 如果发送的消息来自ip和port记录在servers中了，那么就执行转发.否则丢弃
			_, err := Pipe(message, reader, func(data []byte) ([]byte, int) {
				head := append([]byte{0x00, 0x00, 0x00, 0x01}, src.IP...)
				head = append(head, uint8(src.Port>>8), uint8(src.Port))
				data = append(head, data...)
				return data, len(data)
			})

			if err != nil && err != io.ErrShortWrite {
				log.Println(err)
				return
			}

			p.message <- message
			return
		} else {
			// 若是无记录主机就丢弃信息
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




