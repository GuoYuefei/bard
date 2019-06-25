package bard

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"sync"
	"time"
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


func (p *PCQInfo) HandleConn(conn net.Conn, r *bufio.Reader) (e error) {
	if p.Cmd == 0x01 {
		resp := []byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

		conn.Write(resp)

		var (
			remote net.Conn
			err error
		)

		remote, err = net.Dial("tcp", p.Dst.String())
		if err != nil {
			log.Println(err)
			conn.Close()
			return err
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
		fmt.Println("打个标记")

		udpaddr, e := net.ResolveUDPAddr("udp4", ":"+strconv.Itoa(p.Dst.PortToInt()+1))

		if e != nil {
			log.Println(e)
			return e
		}

		udpPacket, e := net.ListenUDP("udp4", udpaddr)
		//fmt.Println("p.dst.port", p.Dst.PortToInt())
		//fmt.Println(udpPacket.LocalAddr())

		if e != nil {
			log.Printf("udp代理服务器端开启监听失败: %v", e)
			return e
		}

		resp := append([]byte{0x05, 0x00, 0x00, 0x01, 192, 168, 1, 16}, p.Dst.Port[0], p.Dst.Port[1]+1)
		conn.Write(resp)

		for {
			udpReqS, e := NewUDPReqS(udpPacket)
			if e != nil {
				log.Println(e)
				return e
			}


			//// todo 此时其实还是不知道信息到底是远程服务器发来的还是客户端发来的， 这边默认是客户端导致了服务器端主动发来的数据无法穿透代理
			// 根据请求信息向客户端真的想要请求的服务器请求
			res, e := udpReqS.ReqRemote()
			fmt.Println("打个标记1")
			if e != nil {
				log.Println(e)
				return e
			}

			temp := new(bytes.Buffer)
			// 这个是代理服务器返回客户端
			written, e := Pipe(temp, res, func(data []byte) ([]byte,int) {
				head := append([]byte{0x00, 0x00, 0x00}, udpReqS.Dst.ToProtocol()...)
				data = append(head, data...)
				return data, len(data)
			})
			if e != nil && e != io.ErrShortWrite {
				log.Println(e)
				return e
			}

			clientAddr, e := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", "127.0.0.1", p.Dst.PortToInt()))
			fmt.Println("客户端的udp监听地址：", clientAddr)
			n, e := udpPacket.WriteTo(temp.Bytes()[0: written], clientAddr)


			if e != nil {
				log.Println(e)
				return e
			}

			//if written != int64(n) {
			//	fmt.Printf("读写问题 written = %d, n = %d", written, n)
			//}



			log.Printf("---------------通过udp传输了%dB的数据\n这些数据是%v", n, temp.Bytes())
			time.Sleep(100*time.Millisecond)

		}
	}
	return e
}
