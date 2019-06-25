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
	Src net.Addr
	Rsv []byte				//保留字节 2B
	Frag byte
	Dst *Address
	Data *bytes.Buffer				// 请求携带的数据
}

func (u *UDPReqS) String() string {
	if u.Dst.Atyp != 0x003 {
		return u.Dst.String()
	}

	ips, e := net.LookupIP(u.Dst.AddrString())

	if e != nil {
		log.Println("-------------dns解析失败---------------", e)
		return ""
	}
	fmt.Println(ips)

	return fmt.Sprintf("%s:%d", ips[0], u.Dst.PortToInt())
}

//Network
func (u *UDPReqS)Network() string {
	return "udp"
}

func NewUDPReqS(packet net.PacketConn) (*UDPReqS, error) {

	var buf [4096]byte

	n, addr, err := packet.ReadFrom(buf[0:])

	reader := bufio.NewReader(bytes.NewReader(buf[0:n]))

	u := &UDPReqS{}
	u.udpPacket = packet
	u.Src = addr

	u.Rsv = make([]byte, 2)

	_, err = io.ReadFull(reader, u.Rsv)
	if err != nil {
		return nil, err
	}

	u.Frag, _ = reader.ReadByte()

	u.Dst, _ = ReadRemoteHost(reader)

	u.Data = new(bytes.Buffer)
	_, err = io.Copy(u.Data, reader)
	//time.Sleep(10000*time.Millisecond)
	fmt.Println("len", len(u.Data.Bytes()),"u.Data", u.Data.Bytes())
	if err != nil {
		return nil, err
	}

	return u, nil
}

// 代理服务器对外发出udp请求
func (u *UDPReqS)ReqRemote() (res *bytes.Buffer ,err error) {


	//fmt.Println(net.LookupIP(u.Dst.AddrString()))

	addr, err := net.ResolveUDPAddr(u.Network(), u.Dst.String())

	if err != nil {
		log.Println("-------------udp addr error is %v ", err)
		return
	}

	_, err = u.udpPacket.WriteTo(u.Data.Bytes(), addr)
	//fmt.Println(u.Data.Bytes())
	//fmt.Println(u, u.Frag)

	//time.Sleep(3000*time.Millisecond)

	if err != nil {
		log.Println("reqremote:---------------------- ", err)
		return nil, err
	}
	var buf [4096]byte
	//time.Sleep(100*time.Millisecond)
	n, _, err := u.udpPacket.ReadFrom(buf[0:])

	res = bytes.NewBuffer(buf[0:n])
	fmt.Println("............",len(res.Bytes()), res.Bytes())
	return
}
