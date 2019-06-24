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

func (p *PCQInfo) Copy(dst io.Writer, src io.Reader, ornament func([]byte)) (written int64, err error) {
	return p.CopyBuffer(dst, src, nil, ornament)
}

// 参照io.copyBuffer
// ornament 用于将来插件注册使用
func (p *PCQInfo) CopyBuffer(dst io.Writer, src io.Reader, buf []byte, ornament func([]byte)) (written int64, err error) {
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
		if !(p.Cmd != 0x03) {
			return 				// 非udp连接无需处理
		}
		// 请求udp连接
		temp := append([]byte{0x00, 0x00, p.Frag},
			p.Dst.ToProtocol()...)
		bytes = append(temp, bytes...)
		p.Frag++
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
			written, e := p.Copy(remote, r, nil)
			if e != nil {
				log.Printf("从r中写入到remote失败: %v", e)
			} else {
				log.Printf("r -> remote 复制了%dB信息", written)
			}
			remote.Close()
		}()

		go func() {
			defer wg.Done()
			written, e := p.Copy(conn, remote, nil)
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

		udpPacket, e := net.ListenPacket("udp", "47.100.167.83:"+ strconv.Itoa(p.Dst.PortToInt()))

		if e != nil {
			log.Printf("udp代理服务器端开启监听失败: %v", e)
			return e
		}

		resp := append([]byte{0x05, 0x00, 0x00, 0x01, 47, 100, 167, 83}, p.Dst.Port...)
		conn.Write(resp)


		udpReqS, e := NewUDPReqS(udpPacket)
		if e != nil {
			log.Println(e)
			return e
		}
		// 根据请求信息向客户端真的想要请求的服务器请求
		res, e := udpReqS.Req()
		fmt.Println("打个标记1")
		if e != nil {
			log.Println(e)
			return e
		}

		temp := new(bytes.Buffer)
		// 这个是代理服务器返回客户端
		written, e := p.Copy(temp, res, nil)
		if e != nil {
			log.Println(e)
			return e
		}

		n, e := udpPacket.WriteTo(temp.Bytes(), p)


		if e != nil {
			log.Println(e)
			return e
		}

		if written != int64(n) {
			fmt.Println("读写问题")
		}

		log.Printf("---------------通过udp传输了%dB的数据", written)
	}
	return e
}
