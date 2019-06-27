package bard

import (
	"bufio"
	"errors"
	"log"
	"net"
	"strconv"
	"sync"
)

// 主要用于socks5请求问题
// Proxy connection request information
type PCQInfo struct {
	Ver byte  		// version
	Cmd byte		// command
	Frag byte		// 仅用于Cmd=0x03时的udp传输
	Rsv byte		// reserve
	Dst *Address
}

func (p *PCQInfo) Remote() (net.Conn, error){
	if p.Cmd == 0x01 {
		return net.Dial("tcp", p.String())
	} else if p.Cmd == 0x03 {
		return net.Dial("udp", p.String())
	} else {
		return nil, errors.New("cmd为" + strconv.Itoa(int(p.Cmd)) +",暂不支持该命令")
	}
}

func (p *PCQInfo) Network() string {
	if p.Cmd != 0x03 {
		return "tcp"
	} else {
		return "udp"
	}
}

func (p *PCQInfo) String() string {
	return p.Dst.String()
}

func (p *PCQInfo) Response(conn net.Conn, config *Config) error {

	if p.Cmd == 0x01 {
		resp := []byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
		_, err := conn.Write(resp)
		return err
	}

	// 主要响应socks5最后的请求
	// 服务器端server 一般只有一个ip todo 先别管多IP吧
	var ip = net.ParseIP(config.GetServers()[0])
	//fmt.Println("ip is .............", ip.To4())
	// 因为tcp情况下后面的ip端口都是无效信息， 所以不会影响什么。 这边是遵照回应udp的写法
	resp := append([]byte{0x05, 0x00, 0x00, 0x01}, ip.To4()...)
	resp = append(resp, p.Dst.Port[0], p.Dst.Port[1]+2)
	_, err := conn.Write(resp)
	return err
}


func (p *PCQInfo) HandleConn(conn net.Conn, r *bufio.Reader, config *Config) (e error) {
	if p.Cmd == 0x01 {
		e = p.Response(conn, config)

		var (
			remote net.Conn
		)

		remote, e = net.Dial("tcp", p.Dst.String())
		if e != nil {
			log.Println(e)
			conn.Close()
			return e
		}
		defer remote.Close()

		wg := new(sync.WaitGroup)
		wg.Add(2)
		//fmt.Println("xxxxx")
		go func() {
			defer wg.Done()
			written, e := Pipe(remote, r, nil)
			if e != nil {
				log.Printf("从r中写入到remote失败: %v", e)
			} else {
				log.Printf("r -> remote 复制了%dB信息", written)
			}
			remote.Close()
		}()

		go func() {
			defer wg.Done()
			written, e := Pipe(conn, remote, nil)
			if e != nil {
				log.Printf("从remote中写入到r失败: %v", e)
			} else {
				log.Printf("remote->r 复制了%dB信息", written)
			}
			conn.Close()
		}()

		wg.Wait()
		return nil

	} else if p.Cmd == 0x03 {
		//fmt.Println("打个标记")

		udpaddr, e := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(p.Dst.PortToInt()+2))

		if e != nil {
			log.Println(e)
			return e
		}

		udpPacket, e := net.ListenUDP("udp", udpaddr)
		//fmt.Println("p.dst.port", p.Dst.PortToInt())
		//fmt.Println(udpPacket.LocalAddr())

		if e != nil {
			log.Printf("udp代理服务器端开启监听失败: %v", e)
			return e
		}

		// 对客户端回应
		e = p.Response(conn, config)

		packet, e := NewPacket(conn, udpPacket, p.Dst.PortToInt())

		wg := new(sync.WaitGroup)
		wg. Add(2)

		go func() {
			defer wg.Done()
			for {
				packet.Listen()
			}
		}()

		go func() {
			defer wg.Done()
			for {
				_, err := packet.Request()
				if err != nil {
					continue
				}
			}
		}()

		wg.Wait()

	}
	return e
}
