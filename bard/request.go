package bard

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
)

// 客户端的大部分内容 请求过程

// LOCAL -> CSM -> REMOTE
// REMOTE -> SCM -> LOCAL
// PCQI 有请求的所有信息
// 如果是udp需要生成Packet类型，应该说要组合Packet之后重写Listen
type Client struct {
	config *Config

	LocalConn *Conn
	//CSMessage chan *Message

	PCQI *PCQInfo					// node 这是LocalConn得到的请求 这个内容其实只要原封不动传给远程代理就行
	// todo 以下addr需要改成PCRspInfo类型
	PCRsp *PCRspInfo			// node 这是RemoteConn远程服务器发回的响应	服务器返回响应，仅udp代理时有用,先不考虑udp

	//SCMessage chan *Message
	RemoteConn *Conn
}

func (c *Client)Close() error {
	e1 := c.RemoteConn.Close()
	e2 := c.LocalConn.Close()
	if e1 != nil && e2 != nil {
		return fmt.Errorf("RemoteConn Close error: %v,\nLocalConn Close error: %v\n", e1, e2)
	}
	if e1 != nil {
		return fmt.Errorf("RemoteConn Close error: %v\n", e1)
	}
	if e2 != nil {
		return fmt.Errorf("LocalConn Close error: %v\n", e2)
	}
	return nil
}

func (c *Client)Pipe() {
	defer c.LocalConn.Close()
	if c.PCQI.Cmd == REQUEST_TCP {
		c.PipeTcp()
	} else if c.PCQI.Cmd == REQUEST_UDP {
		c.PipeUdp()
	} else {
		// 前期就已经检查，不会出现这种情况
		return
	}
}

func (c *Client)PipeUdp() {
	// do something with udp channel
	//remoteUdpAddr, err := net.ResolveUDPAddr("udp", c.PCRsp.SAddr.AddrString())
	localUdpAddr, err := net.ResolveUDPAddr("udp", c.config.GetLocalString()+":"+
														strconv.Itoa(c.PCQI.Dst.PortToInt()+2))
	if err != nil {
		Deb.Println("UDP parse error,", err)
		return
	}
	// node 这个udpPacket是客户端处理一个udp连接时的唯一通道，包括与客户端的客户端通讯和客户端的服务器端
	localPacket, err := net.ListenUDP("udp", localUdpAddr)
	if err != nil {
		Deb.Println(err)
		RefuseRequest(c.LocalConn)
		return
	}
	err = c.PCQI.Response(c.LocalConn, c.config.GetLocalString())

	if err != nil {
		Deb.Println(err)
		//RefuseRequest(c.LocalConn)			// 没回复成功成功，不知要不要回复失败，因为可能回复失败也失败
		return
	}

	// c.RemoteConn 主要是把含有plugin的一个连接传入 此时Packet类型中的client就是远程代理服务器的监听地址了。 因为udp交流是双方是平等的，也可以将远程服务器理解成本udp连接的客户端
	packet, e := NewPacket(/*c.LocalConn*/c.RemoteConn, localPacket, c.PCRsp.SAddr.PortToInt())
	if e != nil {
		return
	}
	packet.SetTimeout(c.config.Timeout)
	if addr, ok := c.LocalConn.RemoteAddr().(*net.TCPAddr); ok {
		udpAddr, err := net.ResolveUDPAddr("udp", addr.IP.String()+":"+strconv.Itoa(c.PCQI.Dst.PortToInt()))
		if err != nil {
			return
		}
		packet.AddServer("local", udpAddr)
	} else {
		Deb.Println("c.LocalConn.RemoteAddr() is not a TCPAddr")
	}
	wg := new(sync.WaitGroup)
	wg. Add(2)

	go func() {
		defer wg.Done()
		for {
			err := c.LocalConn.SetDeadline(c.LocalConn.GetDeadline()) //维持下本地的socks5连接
			if err != nil {
				Deb.Println("Local socks5 protocol setting timeout error, may have been disconnected")
				break
			}
			err = packet.ListenToFixedTarget("local")
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
			err := c.LocalConn.SetDeadline(c.LocalConn.GetDeadline()) //维持下本地的socks5连接
			if err != nil {
				Deb.Println("Local socks5 protocol setting timeout error, may have been disconnected")
				break
			}
			_, err = packet.Request()
			if err != nil {
				Slog.Println("packet.request close:", err)
				break
			}
		}
	}()

	wg.Wait()
}

func (c *Client)PipeTcp() {
	wg := new(sync.WaitGroup)
	wg.Add(2)
	e := c.PCQI.Response(c.LocalConn, c.config.GetLocalString())
	if e != nil {
		Deb.Println(e)
		c.Close()
		wg.Done(); wg.Done()			// 发生错误还需要解锁的
		return
	}
	go func() {
		defer wg.Done()
		written, e := Pipe(c.RemoteConn, c.LocalConn, dealOrnament(SEND, c.RemoteConn.Plugin()))
		if e != nil {
			Deb.Printf("LocalConn -> RemoteConn失败: %v", e)
		} else {
			Deb.Printf("LocalConn -> RemoteConn 复制了%dB信息", written)
		}
		// todo
		e = c.RemoteConn.Close()
		if e != nil {
			Logff(ExceptionTurnOffRemoteTCP.Error()+"%v", LOG_WARNING, e)
		}
	}()

	go func() {
		defer wg.Done()
		// 远程服务器发来的消息可能超过BUFSIZE，因为加过修饰
		Readbuf := make([]byte, ReadBUFSIZE)

		written, e := PipeBuffer(c.LocalConn, c.RemoteConn, Readbuf, dealOrnament(RECEIVE, c.RemoteConn.Plugin()))
		if e != nil {
			Deb.Printf("RemoteConn -> LocalConn失败: %v", e)
		} else {
			Deb.Printf("RemoteConn -> LocalConn 复制了%dB信息", written)
		}
		// todo
		//e = c.LocalConn.Close()
		//if e != nil {
		//	Logff(ExceptionTurnOffRemoteTCP.Error()+"%v", LOG_WARNING, e)
		//}
	}()
	wg.Wait()


}



/**
	定义Client的能力
	1、可以根据配置文件连接远程代理服务器，做到第一次握手
	2、初始化两个Message的通道
	-------------以上应该在初始化时完成----------------
	3、两个conn各自对自己的通道操作。 每个函数2协程
	4、外层调用，开两个协程。 over 一共一次连接7协程


	@param localConn 一定要是已经建立起本地socks5连接的conn
	@param pcqi 是localConn接收到的请求信息
	@param config 配置文件信息
*/
func NewClient(localConn *Conn, pcqi *PCQInfo, config *Config, plugin IPlugin) (c *Client, err error){
	c = &Client{}
	remoteConn, pcrsp, err := NewRemoteConn(config, pcqi, plugin)

	if err != nil {
		return
	}
	c.config = config
	c.RemoteConn = remoteConn
	c.LocalConn = localConn
	c.PCQI = pcqi

	// 和udp通道不同，udp通道两个出口或入口都是相同协议的。 这边是双协议所以需要两个通道
	//c.CSMessage = make(chan *Message, MESSAGESIZE)
	//c.SCMessage = make(chan *Message, MESSAGESIZE)

	c.PCRsp = pcrsp

	return
}

// 这个就是想
func NewRemoteConn(config *Config, pcqi *PCQInfo, plugin IPlugin) (remoteConn *Conn, pcrsp *PCRspInfo, err error) {
	conn, err := net.Dial("tcp", config.GetServers()[0]+":"+config.ServerPortString())
	if err != nil {
		return
	}

	remoteConn = NewConnTimeout(conn, config.Timeout)
	if plugin != nil {
		remoteConn.Register(plugin)
	}

	r := bufio.NewReader(remoteConn)

	pcrsp, err = ClientHandleShakeWithRemote(r, remoteConn, pcqi, config)

	return
}

// 与远程代理服务器握手
func ClientHandleShakeWithRemote(r *bufio.Reader, conn *Conn, pcqi *PCQInfo, config *Config) (pcrsp *PCRspInfo,e error) {
	conn.Write([]byte{SocksVersion, 0x02, NOAUTH, AuthUserPassword})

	b, e := r.ReadByte()
	if e != nil {
		return
	}

	if b != SocksVersion {
		e = ErrorSocksVersion
		Deb.Println("version byte is", b)
		return
	}
	method, e := r.ReadByte()
	if e != nil {
		return
	}

	if method == AuthUserPassword {
		// endtodo 进行密码验证环节
		if !UserPassWDClient(r, conn, config.Users[0]) {
			e = errors.New("user password: server auth refused")
			return
		}
	} else if method != NOAUTH {
		// 不是账号密码验证和不需要验证两种方式，就返回错误
		e = errors.New("server return Auth method error")
		return
	}
	r.Reset(conn)			// 清空缓存
	// 验证通过之后处理第二次握手----请求建立
	_, e = conn.Write(pcqi.ToBytes())
	if e != nil {
		return
	}

	pcrsp, e = ReadPCRspInfo(r)

	return
}




