// socks5 协议的一些方法集合
package bard

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
)

/**
	这个文件是server程序独占。 考虑是否移除到包外
 */

const (
	SocksVersion uint8 = 0x05



	// UDP ASSOCIATE  udp 协议请求代理
	REQUEST_UDP uint8 = 0X03
	// connect  请求TCP连接
	REQUEST_TCP uint8 = 0X01
	// bind  特殊的tcp连接   据我所知只有ftp需要这个 暂时未实现   没用到过
	REQUEST_BIND uint8 = 0X02

	IPV4 uint8 = 0X01
	DOMAIN uint8 = 0X03
	IPV6 uint8 = 0X04

)
var ErrorSocksVersion = errors.New("is not socks5")
var ErrorAuth = errors.New("Authentication failed")

/**
客户端发送要建立的代理连接的地址及端口 地址可能是域名、ipv4、ipv6
+----------+------------+---------+-----------+-----------------------+------------+
|协议版本号  | 请求的类型  |保留字段   |  地址类型  |  地址数据              |  地址端口    |
+----------+------------+---------+-----------+-----------------------+------------+
|1个字节    | 1个字节     |1个字节   |  1个字节   |  变长                  |  2个字节    |
+----------+------------+---------+-----------+-----------------------+------------+
|0x05      | 0x01		|0x00     |  0x01     |  0x0a,0x00,0x01,0x0a  |  0x00,0x50 |
+----------+------------+---------+-----------+-----------------------+------------+

 */

func ServerHandShake(r *bufio.Reader, conn net.Conn, config *Config) error {
	version, err := r.ReadByte()

	if err != nil {
		return err
	}

	if version != SocksVersion {
		return fmt.Errorf("socks's version is %d", version)
	}

	nmethods, _ := r.ReadByte()
	Deb.Printf("Methods' Lenght is %d", nmethods)

	buf := make([]byte, nmethods)

	_, err = io.ReadFull(r, buf)
	if err != nil {
		return err
	}
	Deb.Printf("验证方式为： %v", buf)

	resp, ok := Auth(buf, r, conn, config)
	if !ok {
		Logf("Connection request from client IP %v, permission authentication failed", conn.RemoteAddr())
	}

	// endtodo 丢弃缓冲区中所有内容，准备下轮对话
	r.Reset(conn)
	_, err = conn.Write(resp)
	if err != nil {
		return err
	}
	if !ok {
		return ErrorAuth
	}

	return nil
}


func ReadRemoteHost(r *bufio.Reader) (*Address, error) {
	var err error
	address := &Address{}
	addrType, err := r.ReadByte()
	var port [2]byte

	if err != nil {
		goto ErrorReturn
	}

	address.Atyp = addrType

	switch addrType {
	case 0x01:
		var ip net.IP = make([]byte, 4)
		_, err = io.ReadFull(r, ip)
		address.Addr = ip

	case 0x03:
		domainLen, _ := r.ReadByte()
		var domain []byte = make([]byte, domainLen)
		_, err = io.ReadFull(r, domain)
		address.Addr = domain

	case 0x04:
		var ip net.IP = make([]byte, 16)
		_, err = io.ReadFull(r, ip)
		address.Addr = ip
	}
	if err != nil {
		goto ErrorReturn
	}


	_, err = io.ReadFull(r, port[0:])
	if err != nil {
		goto ErrorReturn
	}
	address.Port = port[0:]

	return address, nil

	ErrorReturn:
		return nil, err

}

func ReadPCQInfo(r *bufio.Reader) (*PCQInfo, error) {
	version, _ := r.ReadByte()

	if version != SocksVersion {
		return nil, fmt.Errorf("This is not the Socks5 protocol, version is %d", version)
	}

	cmd, _ := r.ReadByte()

	Deb.Println("socks' cmd:\t",cmd)
	if cmd&0x01 != 0x01 {
		// todo 现在仅支持0x03 and 0x01 即非bind请求
		return nil, errors.New("客户端请求类型不为1或者3， 暂且只支持代理连接和udp")
	}


	rsv, err := r.ReadByte()		//保留字段

	if err != nil {
		return nil, err
	}

	// address应该能传出去的
	address, err := ReadRemoteHost(r)
	if err != nil {
		return nil, err
	}

	return &PCQInfo{version, cmd,0x00,rsv, address}, nil
}

// 记录服务器端的相应请求的信息
func ReadPCRspInfo(r *bufio.Reader) (pcrsp *PCRspInfo, e error) {
	ver, e := r.ReadByte()
	if e != nil {
		return
	}
	if ver != SocksVersion {
		e = ErrorSocksVersion
		return
	}
	response , e := r.ReadByte()
	if e != nil {
		return
	}
	// 成功 不细分错误代码
	if response != 0x00 {
		return nil, fmt.Errorf("server reponse code is %x", response)
	}
	rsv, e := r.ReadByte() //保留字节
	if e != nil {
		return
	}
	addr, e := ReadRemoteHost(r)
	if e != nil {
		return
	}
	pcrsp = &PCRspInfo{
		Ver: ver,
		Rep: response,
		RSV: rsv,
		SAddr: addr,
	}
	return pcrsp, e
}

// 请求是的拒绝请求回应
func RefuseRequest(conn net.Conn) {
	resp := []byte{0x05, 0x05, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	_, err := conn.Write(resp)
	if err != nil {
		Deb.Printf("refuse connect error:\t", err)
	}
}





