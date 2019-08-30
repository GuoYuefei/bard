package bard

import (
	"bufio"
	"bytes"
	"io"
	"net"
)

// udp

// 不论是UDP请求还是相应都是这个格式
type UDPReqS struct {
	// udppacket 交给该类型管理
	//udpPacket net.PacketConn
	Src net.Addr
	Rsv []byte				//保留字节 2B
	Frag byte
	Dst *Address
	Data *bytes.Buffer				// 请求携带的数据
}

// ip:port 域名全部dns成ip
func (u *UDPReqS) String() string {
	if u.Dst.Atyp != 0x03 {
		return u.Dst.String()
	}

	ips, e := net.LookupIP(u.Dst.AddrString())

	if e != nil {
		Deb.Println("-------------dns解析失败---------------", e)
		return ""
	}
	// 解析出来可能只有ipv6，所以得处理ipv6
	ipbytes, iptype, _ := IpToBytes(ips[0])

	addr := Address{
		Atyp: iptype,
		Addr: ipbytes,
		Port: u.Dst.Port,
	}

	return addr.String()
}

//Network
func (u *UDPReqS)Network() string {
	return "udp"
}

func NewUDPReqS(packet net.PacketConn) (*UDPReqS, error) {

	var buf = make([]byte, BUFSIZE)


	n, addr, err := packet.ReadFrom(buf[0:])
	if err != nil {
		Deb.Println(err)
		return nil, err
	}


	reader := bufio.NewReader(bytes.NewReader(buf[0:n]))


	Deb.Printf("udp的消息来源 %s\n", addr)				//来源和我们要发送的地址相同

	return NewUDPReqSFromReader(reader, addr)
}

func NewUDPReqSFromReader(reader *bufio.Reader, addr net.Addr) (*UDPReqS, error) {
	u := &UDPReqS{}
	//u.udpPacket = packet
	u.Src = addr			// todo 记录下消息来源

	u.Rsv = make([]byte, 2)

	_, err := io.ReadFull(reader, u.Rsv)
	if err != nil {
		return nil, err
	}

	u.Frag, _ = reader.ReadByte()

	u.Dst, err = ReadRemoteHost(reader)
	if err != nil {
		return nil, err
	}

	u.Data = new(bytes.Buffer)
	_, err = io.Copy(u.Data, reader)
	//time.Sleep(10000*time.Millisecond)
	//fmt.Println("len", len(u.Data.Bytes()),"u.Data", u.Data.Bytes())
	if err != nil {
		return nil, err
	}

	return u, nil
}

// 代理服务器对外发出udp请求
//func (u *UDPReqS)ReqRemote() (res *bytes.Buffer ,err error) {
//
//
//	//fmt.Println(net.LookupIP(u.Dst.AddrString()))
//
//	addr, err := net.ResolveUDPAddr(u.Network(), u.Dst.String())
//
//	if err != nil {
//		Deb.Println("-------------udp addr error is %v ", err)
//		return
//	}
//
//	_, err = u.udpPacket.WriteTo(u.Data.Bytes(), addr)
//	//fmt.Println(u.Data.Bytes())
//	//fmt.Println(u, u.Frag)
//
//	//time.Sleep(3000*time.Millisecond)
//
//	if err != nil {
//		Deb.Println("reqremote:---------------------- ", err)
//		return nil, err
//	}
//	var buf [4096]byte
//	//time.Sleep(100*time.Millisecond)
//	n, _, err := u.udpPacket.ReadFrom(buf[0:])
//
//	res = bytes.NewBuffer(buf[0:n])
//	fmt.Println("............",len(res.Bytes()), res.Bytes())
//	return
//}
