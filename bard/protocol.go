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
	Atyp byte			// Atyp address type
	Addr []byte
	Port int			// 16 bit
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
func ParseReq(requset []byte) *PCQInfo {
	var pcqi = &PCQInfo{}
	pcqi.Ver = requset[0]
	pcqi.Cmd = requset[1]
	pcqi.Rsv = requset[2]
	dst := Address{}
	dst.Atyp = requset[3]

	// 地址可能存在几种可能变长的域名，定常的ipv4和ipv6
	switch dst.Atyp {
	case uint8(0x01):
		// ipv4
		dst.Addr = append([]byte{}, requset[4: 8]...)				//取四五六七
		dst.Port = int(requset[8: 9][0]) * 256 + int(requset[9: 10][0])
	case 0x03:
		// domain
		l := requset[4]
		dst.Addr = append([]byte{}, requset[5: 5+l]...)
		dst.Port = int(requset[5+l: 6+l][0]) * 256 + int(requset[6+l: 7+l][0])
	case 0x04:
		// ipv6
		dst.Addr = append([]byte{}, requset[4: 20]...)
		dst.Port = int(requset[20: 21][0]) * 256 + int(requset[21: 22][0])
	}

	pcqi.Dst = dst
	return pcqi
}

func HandShake(r *bufio.Reader, conn net.Conn) error {
	version, _ := r.ReadByte()
	log.Printf("socks's version is %d", version)

	if version != 5 {
		return errors.New("该协议不是socks5协议")
	}

	nmethods, _ := r.ReadByte()
	log.Printf("Methods' lenght is %d", nmethods)

	buf := make([]byte, nmethods)

	io.ReadFull(r, buf)
	log.Printf("验证方式为： %v", buf)

	resp := []byte{5, 0}
	conn.Write(resp)
	return nil
}

func ReadAddr(r *bufio.Reader) (string, error) {
	version, _ := r.ReadByte()
	log.Printf("socks's version is %d", version)
	if version != 5 {
		return "", errors.New("该协议不是socks5协议")
	}

	cmd, _ := r.ReadByte()

	if cmd != 1 {
		return "", errors.New("客户端请求类型不为1， 暂且只支持代理连接")
	}

	r.ReadByte()		//保留字段

	addrtype, _ := r.ReadByte()
	log.Printf("客户端请求为远程服务器地址类型为: %d", addrtype)


	if addrtype != 3 {
		return "", errors.New("请求的远程服务器地址类型部位3， 暂时不支持其他地址类型")
	}

	addrlen, _ := r.ReadByte()
	addr := make([]byte, addrlen)
	io.ReadFull(r, addr)

	log.Printf("域名为: %s", addr)

	var port int16

	binary.Read(r, binary.BigEndian, &port)

	return fmt.Sprintf("%s:%d", addr, port), nil

}

func HandleConn(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	HandShake(r, conn)
	addr, err := ReadAddr(r)
	if err != nil {
		log.Println(err)
	}
	log.Printf("得到的完整的地址是：%s", addr)
	resp := []byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	conn.Write(resp)

	var (
		remote net.Conn
	)

	remote, err = net.Dial("tcp", addr)
	if err != nil {
		log.Println(err)
		conn.Close()
		return
	}
	defer remote.Close()

	wg := new(sync.WaitGroup)
	wg.Add(2)
	fmt.Println("xxxxx")
	go func() {
		defer wg.Done()
		written, e := io.Copy(remote, r)
		if e != nil {
			log.Printf("从r中写入到remote失败: %v", e)
		} else {
			log.Printf("复制了%d信息", written)
		}

	}()

	go func() {
		defer wg.Done()
		written, e := io.Copy(conn, remote)
		if e != nil {
			log.Printf("从remote中写入到r失败: %v", e)
		} else {
			log.Printf("复制了%d信息", written)
		}

	}()

	wg.Wait()
	//remote.Close()
	//conn.Close()
}






