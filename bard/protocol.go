// socks5 协议的一些方法集合
package bard

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

const (
	SocksVersion = 5

)

type Address struct {
	Atyp byte			// Atyp address type 0x01, 0x02, 0x04
	Addr []byte
	Port int16			// 16 bit
}

func (a *Address) AddrString() string {
	var hostname string
	if !(a.Atyp&0x02==0x02) {
		// 就说明是非域名
		var ip net.IP = a.Addr
		hostname = ip.String()
	} else {
		hostname = string(a.Addr)
	}

	return hostname
}

func (a *Address) String() string {

	return fmt.Sprintf("%s:%d", a.AddrString(), a.Port)
}


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

// Proxy connection request information
type PCQInfo struct {
	Ver byte  		// version
	Cmd byte		// command
	Rsv byte		// reserve
	Dst Address
}

// 解析请求信息 这个函数在程序中会往复使用，在其后的试探中应该加强其效率
// 这个方法废弃 错误， 并且在之前的server中使用，现在的server未用
func ParseReq(requset []byte) *PCQInfo {
	var pcqi = &PCQInfo{}
	pcqi.Ver = requset[0]
	pcqi.Cmd = requset[1]
	pcqi.Rsv = requset[2]
	dst := Address{}
	dst.Atyp = requset[3]

	// 地址可能存在几种可能变长的域名，定常的ipv4和ipv6 ip那边有问题。。。。
	switch dst.Atyp {
	case uint8(0x01):
		// ipv4
		dst.Addr = append([]byte{}, requset[4: 8]...)				//取四五六七
		dst.Port = int16(requset[8: 9][0]) * 256 + int16(requset[9: 10][0])
	case 0x03:
		// domain
		l := requset[4]
		dst.Addr = append([]byte{}, requset[5: 5+l]...)
		dst.Port = int16(requset[5+l: 6+l][0]) * 256 + int16(requset[6+l: 7+l][0])
	case 0x04:
		// ipv6
		dst.Addr = append([]byte{}, requset[4: 20]...)
		dst.Port = int16(requset[20: 21][0]) * 256 + int16(requset[21: 22][0])
	}

	pcqi.Dst = dst
	return pcqi
}

func HandShake(r *bufio.Reader, conn net.Conn) error {
	version, _ := r.ReadByte()
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


	var port int16
	binary.Read(r, binary.BigEndian, &port)
	address.Port = port

	return address, nil

}

func ReadAddr(r *bufio.Reader) (*Address, error) {
	version, _ := r.ReadByte()
	log.Printf("socks's version is %d", version)
	if version != SocksVersion {
		return nil, errors.New("该协议不是socks5协议")
	}

	cmd, _ := r.ReadByte()

	if cmd != 1 {
		return nil, errors.New("客户端请求类型不为1， 暂且只支持代理连接")
	}

	r.ReadByte()		//保留字段

	// address应该能传出去的
	address, _ := ReadRemoteHost(r)
	log.Println("连接具体地址为：",address)

	return address, nil

}

func HandleConn(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	err := HandShake(r, conn)

	if err != nil {
		log.Println(err)
		return
	}

	addr, err := ReadAddr(r)
	if err != nil {
		log.Println(err)
		// 拒绝请求处理
		resp := []byte{0x05, 0x05, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
		conn.Write(resp)
		return
	}
	log.Printf("得到的完整的地址是：%s", addr)
	resp := []byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	conn.Write(resp)

	var (
		remote net.Conn
	)

	remote, err = net.Dial("tcp", addr.String())
	if err != nil {
		log.Println(err)
		conn.Close()
		return
	}
	defer remote.Close()

	wg := new(sync.WaitGroup)
	wg.Add(2)
	//fmt.Println("xxxxx")
	go func() {
		defer wg.Done()
		written, e := io.Copy(remote, r)
		if e != nil {
			log.Printf("从r中写入到remote失败: %v", e)
		} else {
			log.Printf("r -> remote 复制了%dB信息", written)
		}
		remote.Close()
	}()

	go func() {
		defer wg.Done()
		written, e := io.Copy(conn, remote)
		if e != nil {
			log.Printf("从remote中写入到r失败: %v", e)
		} else {
			log.Printf("remote->r 复制了%dB信息", written)
		}
		conn.Close()
	}()

	wg.Wait()
	//remote.Close()
	//conn.Close()
}






