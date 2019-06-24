package bard

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
)

// udp

// 不论是UDP请求还是相应都是这个格式
type UDPReqS struct {
	// udppacket 交给该类型管理
	udpPacket net.PacketConn
	Rsv []byte				//保留字节 2B
	Frag byte
	Dst *Address
	Data *bytes.Buffer				// 请求携带的数据
}

func (u *UDPReqS) String() string {
	return u.Dst.String()
}

//Network
func (u *UDPReqS)Network() string {
	return "udp"
}

func NewUDPReqS(packet net.PacketConn) (*UDPReqS, error) {

	var buf [4096]byte

	n, _, err := packet.ReadFrom(buf[0:])

	reader := bufio.NewReader(bytes.NewReader(buf[0:n]))

	u := &UDPReqS{}
	u.udpPacket = packet

	u.Rsv = make([]byte, 2)

	_, err = io.ReadFull(reader, u.Rsv)
	if err != nil {
		return nil, err
	}

	u.Frag, _ = reader.ReadByte()

	u.Dst, _ = ReadRemoteHost(reader)

	u.Data = new(bytes.Buffer)
	_, err = io.Copy(u.Data, reader)
	fmt.Println("u.Data", u.Data.Bytes())
	if err != nil {
		return nil, err
	}

	return u, nil
}

// 代理服务器对外发出udp请求
func (u *UDPReqS)Req() (res *bytes.Buffer ,err error) {

	_, err = u.udpPacket.WriteTo(u.Data.Bytes(), u)
	fmt.Println(u.Data.Bytes())
	fmt.Println(u)

	if err != nil {
		log.Println(".......", err)
		return nil, err
	}
	res = new(bytes.Buffer)

	_, _, err = u.udpPacket.ReadFrom(res.Bytes())

	fmt.Println(res.Bytes())
	return
}
