// socks5 协议的一些方法集合
package bard

import (
	"bufio"
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
	Port []byte			// 16 bit
}

func (a *Address) PortToInt() int {
	p := a.Port
	return 256*int(p[0])+int(p[1])
}

// 因为域名这里没有记录下长度，如果用于协议的话，前面需要加域名的长度， 如果是ip则不用加工
func (a *Address) ToProtocolAddr() []byte {
	if a.Atyp&0x02==0x22 {
		// ip就返回原本的bytes
		return a.Addr
	}

	var domainLen byte = byte(len(a.Addr))
	return append([]byte{domainLen}, a.Addr...)
}

// 这是常规协议回应可能用的bytes结构
func (a *Address) ToProtocol() []byte {
	return append(append([]byte{a.Atyp},  a.ToProtocolAddr()...), a.Port...)
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

	return fmt.Sprintf("%s:%d", a.AddrString(), a.PortToInt())
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
	Frag byte		// 仅用于Cmd=0x03时的udp传输
	Rsv byte		// reserve
	Dst *Address
}

func (p *PCQInfo) String() string {
	return p.Dst.String()
}

// 参照io.copyBuffer
// ornament 用于将来插件注册使用
func (p *PCQInfo) Copy(dst io.Writer, src io.Reader, buf []byte, ornament func([]byte)) (written int64, err error) {
	// If the reader has a WriteTo method, use it to do the copy.
	// Avoids an allocation and a copy.
	if wt, ok := src.(io.WriterTo); ok {
		return wt.WriteTo(dst)
	}
	// Similarly, if the writer has a ReadFrom method, use it to do the copy.
	if rt, ok := dst.(io.ReaderFrom); ok {
		return rt.ReadFrom(src)
	}

	var udpHandle = func(bytes []byte) {
		if !(p.Cmd == 0x03) {
			return 				// 非udp连接无需处理
		}
		// 请求udp连接
		temp := append([]byte{0x00, 0x00, p.Frag},
			p.Dst.ToProtocol()...)
		bytes = append(temp, bytes...)
	}

	if ornament == nil {
		// 点缀函数如果不存在的话
		ornament = udpHandle
	} else {
		ornament = func(bytes []byte) {
			ornament(bytes)

			// udp处理是socks5协议的一部分，属于会话层协议，应该放在加密(表示层)或者其他高于会话层协议的数据处理的后面
			udpHandle(bytes)
		}
	}

	if buf == nil {
		size := 32 * 1024
		if l, ok := src.(*io.LimitedReader); ok && int64(size) > l.N {
			if l.N < 1 {
				size = 1
			} else {
				size = int(l.N)
			}
		}
		buf = make([]byte, size)
	}
	for {
		nr, er := src.Read(buf)
		// 数据处理
		ornament(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
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

	fmt.Println(cmd)
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




func HandleConn(conn net.Conn) {
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
	resp := []byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	conn.Write(resp)

	var (
		remote net.Conn
	)

	// todo 这个要改
	remote, err = net.Dial("tcp", pcq.String())
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






