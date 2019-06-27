// socks5 协议的一些方法集合
package bard

import (
	"bufio"
	"errors"
	"fmt"
	"io"
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

	resp := []byte{5, 0}
	_, err = conn.Write(resp)
	if err != nil {
		return err
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
		return nil, errors.New("This is not the Socks5 protocol")
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






func HandleConn(conn net.Conn, config *Config) {
	defer func() {
		err := conn.Close()
		// timeout 可能会应发错误，原因此时conn已关闭
		if err != nil {
			Logff("Close socks5 connection error, the error is %v", LOG_WARNING, err)
		}
	}()
	r := bufio.NewReader(conn)
	err := HandShake(r, conn)

	if err != nil {
		Deb.Println(err)
		return
	}

	pcq, err := ReadPCQInfo(r)
	if err != nil {
		Deb.Println(err)
		// 拒绝请求处理
		resp := []byte{0x05, 0x05, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
		_, err := conn.Write(resp)
		if err != nil {
			Deb.Printf("refuse connect error:\t", err)
		}
		return
	}
	Deb.Printf("得到的完整的地址是：%s", pcq)
	err = pcq.HandleConn(conn, r, config)
	if err != nil {
		Deb.Println(err)
		return
	}

	//remote.Close()
	//conn.Close()
}






