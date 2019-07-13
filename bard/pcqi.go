package bard

import (
	"bufio"
	"errors"
	"net"
	"strconv"
	"sync"
)

var ExceptionTurnOffRemoteTCP = errors.New("Turn off remote host TCP connection exception:")
var ExceptionTurnOffClientTCP = errors.New("Turn off client host TCP connection exception:")
var ExceptionUDPChannelOpen = errors.New("UDP channel failed to open:")

// 主要用于socks5请求问题
// Proxy connection request information
type PCQInfo struct {
	Ver byte  		// version
	Cmd byte		// command
	Frag byte		// 仅用于Cmd=0x03时的udp传输
	Rsv byte		// reserve
	Dst *Address
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
	var resp []byte
	if p.Cmd == REQUEST_TCP {
		// 请求tcp代理
		// 请求代理阶段，服务器返回成功标记
		resp = []byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	} else if p.Cmd == REQUEST_UDP {
		// 主要响应socks5最后的请求 cmd 为udp
		// 服务器端server 一般只有一个ip todo 先别管多IP吧 而且还只支持ipv4
		var ip = net.ParseIP(config.GetServers()[0])
		// fmt.Println("ip is .............", ip.To4())
		// 遵照回应udp的写法
		resp = append([]byte{0x05, 0x00, 0x00, 0x01}, ip.To4()...)
		resp = append(resp, p.Dst.Port[0], p.Dst.Port[1]+2)
	}

	_, err := conn.Write(resp)
	return err


}


func (p *PCQInfo) HandleConn(conn net.Conn, config *Config) (e error) {
	r := bufio.NewReaderSize(conn, 6*1024)

	if p.Cmd == REQUEST_TCP {
		e = p.Response(conn, config)


		remote, e := net.Dial("tcp", p.Dst.String())
		//remote.SetTimeout(config.Timeout)

		if e != nil {
			return e
		}
		defer func() {
			e = remote.Close()
			//if e != nil {
			//	//Logff(ExceptionTurnOffRemoteTCP.Error()+"%v", LOG_WARNING, e)
			//}
		}()

		wg := new(sync.WaitGroup)
		wg.Add(2)
		//fmt.Println("xxxxx")
		go func() {
			defer wg.Done()
			written, e := Pipe(remote, r, nil)
			if e != nil {
				Deb.Printf("从r中写入到remote失败: %v", e)
			} else {
				Deb.Printf("client -> remote 复制了%dB信息", written)
			}
			// todo
			e = remote.Close()
			if e != nil {
				Logff(ExceptionTurnOffRemoteTCP.Error()+"%v", LOG_WARNING, e)
			}
		}()

		go func() {
			defer wg.Done()
			written, e := Pipe(conn, remote, nil)
			if e != nil {
				Deb.Printf("从remote中写入到r失败: %v", e)
			} else {
				Deb.Printf("remote->client 复制了%dB信息", written)
			}
			//e = conn.Close()
			//if e != nil {
			//	Logff(ExceptionTurnOffClientTCP.Error()+"%v", LOG_WARNING, e)
			//}

			// conn 返回后有父级函数关闭 还有其timeout关闭
		}()

		wg.Wait()
		return nil

	} else if p.Cmd == REQUEST_UDP {
		//fmt.Println("打个标记")

		// todo 最终监听的udp端口还没定，暂且是client端口+2
		udpaddr, e := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(p.Dst.PortToInt()+2))

		if e != nil {
			return e
		}

		udpPacket, e := net.ListenUDP("udp", udpaddr)
		//fmt.Println("p.dst.port", p.Dst.PortToInt())
		//fmt.Println(udpPacket.LocalAddr())

		if e != nil {
			Logff(ExceptionUDPChannelOpen.Error()+"%v", LOG_WARNING, e)
			return e
		}

		// 对客户端回应
		e = p.Response(conn, config)

		if e != nil {
			Deb.Println("pcqi.go response error:", e)
			return e
		}

		packet, e := NewPacket(conn, udpPacket, p.Dst.PortToInt())
		if e != nil {
			Deb.Println("pcqi.go newPacket error:", e)
			return e
		}
		packet.SetTimeout(config.Timeout)

		wg := new(sync.WaitGroup)
		wg. Add(2)

		go func() {
			defer wg.Done()
			for {
				err := packet.Listen()
				if err != nil {
					// 记录到日志 可能以后会出现其他错误 如果只是udp关闭的话就是正确的逻辑
					Slog.Println("packet.listen close:", err)
					close(packet.message)
					break
				}
			}
		}()

		go func() {
			defer wg.Done()
			for {
				_, err := packet.Request()
				if err != nil {
					Slog.Println("packet.request close:", err)
					break
				}
			}
		}()

		wg.Wait()

	}
	return e
}
