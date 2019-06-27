// socks5 协议的一些方法集合
package bard

import (
	"bufio"
	"errors"
	"io"
	"log"
	"net"
)

const (
	SocksVersion = 5

)




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

func HandShake(r *bufio.Reader, conn net.Conn) error {
	version, err := r.ReadByte()

	if err != nil {
		log.Println(err)
		return err
	}

	log.Printf("socks's version is %d", version)

	if version != SocksVersion {
		return errors.New("该协议不是socks5协议")
	}

	nmethods, _ := r.ReadByte()
	log.Printf("Methods' Lenght is %d", nmethods)

	buf := make([]byte, nmethods)

	io.ReadFull(r, buf)
	log.Printf("验证方式为： %v", buf)

	resp := []byte{5, 0}
	conn.Write(resp)
	return nil
}

func ReadRemoteHost(r *bufio.Reader) (*Address, error) {
	address := &Address{}
	addrType, _ := r.ReadByte()

	address.Atyp = addrType

	switch addrType {
	case 0x01:
		var ip net.IP = make([]byte, 4)
		io.ReadFull(r, ip)
		address.Addr = ip

	case 0x03:
		domainLen, _ := r.ReadByte()
		var domain []byte = make([]byte, domainLen)
		io.ReadFull(r, domain)
		address.Addr = domain

	case 0x04:
		var ip net.IP = make([]byte, 16)
		io.ReadFull(r, ip)
		address.Addr = ip
	}


	var port [2]byte
	io.ReadFull(r, port[0:])
	address.Port = port[0:]

	return address, nil

}

func ReadPCQInfo(r *bufio.Reader) (*PCQInfo, error) {
	version, _ := r.ReadByte()
	log.Printf("socks's version is %d", version)
	if version != SocksVersion {
		return nil, errors.New("该协议不是socks5协议")
	}

	cmd, _ := r.ReadByte()

	log.Println("socks' cmd:\t",cmd)
	if cmd&0x01 != 0x01 {
		// todo 现在仅支持0x03 and 0x01 即非bind请求
		return nil, errors.New("客户端请求类型不为1或者3， 暂且只支持代理连接和udp")
	}


	rsv, _ := r.ReadByte()		//保留字段

	// address应该能传出去的
	address, _ := ReadRemoteHost(r)
	//log.Println("连接具体地址为：",address)

	return &PCQInfo{version, cmd,0x00,rsv, address}, nil

}






func HandleConn(conn net.Conn, config *Config) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	err := HandShake(r, conn)

	if err != nil {
		log.Println(err)
		return
	}

	pcq, err := ReadPCQInfo(r)
	if err != nil {
		log.Println(err)
		// 拒绝请求处理
		resp := []byte{0x05, 0x05, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
		conn.Write(resp)
		return
	}
	log.Printf("得到的完整的地址是：%s", pcq)
	err = pcq.HandleConn(conn, r, config)
	if err != nil {
		log.Println(err)
		return
	}

	//remote.Close()
	//conn.Close()
}






