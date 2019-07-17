package bard

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
)

// 客户端的大部分内容 请求过程

//// client -> server传的message
//type CToSMessage struct {
//	*Message
//	ProxyRequest []byte			// 其实就是将本地服务器接受到的请求原封不动的记录下来			至于在请求之前的过程---验证，其实同一客户端下都是相同的
//
//}

// 基础的Message结构体
type Message struct {
	Data []byte
}

// 不知道对不对还未验证
func (c *Message) Read(b []byte) (n int, err error) {
	return bytes.NewReader(c.Data).Read(b)
}

func (c *Message) Write(b []byte) (n int, err error) {
	write := new(bytes.Buffer)
	// todo 可能有错
	i, err := write.Write(b)
	c.Data = write.Bytes()

	return i, err
}

// todo !!!!! first
// LOCAL -> CSM -> REMOTE
// REMOTE -> SCM -> LOCAL
// PCQI 有请求的所有信息
// 如果是udp需要生成Packet类型，应该说要组合Packet之后重写Listen
type Client struct {
	LocalConn *Conn
	CSMessage chan *Message

	PCQI *PCQInfo
	Addr *Address			//	服务器返回地址，仅udp代理时有用,先不考虑udp

	SCMessage chan *Message
	RemoteConn *Conn
}

func (c *Client)DealLocal() {
	// Local -> CSM
	// Local <- SCM
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		for {
			csmessage := &Message{}
			_, err := Pipe(csmessage, c.LocalConn, nil)
			if err != nil && err != io.ErrShortWrite {
				break
			}
			c.CSMessage<-csmessage
		}
	}()

	go func() {
		defer wg.Done()
		for {
			scmessage := <- c.SCMessage
			_, err := Pipe(c.LocalConn, scmessage, nil)
			if err != nil && err != io.ErrShortWrite {
				break
			}
		}
	}()

	wg.Wait()
}

func (c *Client)DealRemote() {
	// Remote -> SCM
	// Remote <- CSM
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		for {
			scmessage := &Message{}
			_, err := Pipe(scmessage, c.RemoteConn, nil)
			if err != nil && err != io.ErrShortWrite {
				break
			}
			c.SCMessage <- scmessage
		}

	}()


	go func() {
		defer wg.Done()
		for {
			csmessage := <- c.CSMessage
			_, err := Pipe(c.RemoteConn, csmessage, nil)
			if err != nil && err != io.ErrShortWrite {
				break
			}
		}
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
func NewClient(localConn *Conn, pcqi *PCQInfo, config *Config) (c *Client, err error){
	c = &Client{}
	remoteConn, addr, err := NewRemoteConn(config, pcqi)
	if err != nil {
		return
	}
	c.RemoteConn = remoteConn
	c.LocalConn = localConn
	c.PCQI = pcqi

	// 和udp通道不同，udp通道两个出口或入口都是相同协议的。 这边是双协议所以需要两个通道
	c.CSMessage = make(chan *Message, MESSAGESIZE)
	c.SCMessage = make(chan *Message, MESSAGESIZE)

	c.Addr = addr

	return
}

// 这个就是想
func NewRemoteConn(config *Config, pcqi *PCQInfo) (remoteConn *Conn, addr *Address, err error) {
	conn, err := net.Dial("tcp", config.GetServers()[0]+":"+config.ServerPortString())
	if err != nil {
		return
	}
	remoteConn = NewConnTimeout(conn, config.Timeout)

	r := bufio.NewReader(conn)

	addr, err = ClientHandleShakeWithRemote(r, remoteConn, pcqi)

	return
}

// 与远程代理服务器握手
func ClientHandleShakeWithRemote(r *bufio.Reader, conn *Conn, pcqi *PCQInfo) (addr *Address,e error) {
	errversion := errors.New("is not socks5 server")
	conn.Write([]byte{SocksVersion, 0x02, NOAUTH, AuthUserPassword})
	b, e := r.ReadByte()
	if e != nil {
		return
	}
	if b != SocksVersion {
		e = errversion
		return
	}
	method, e := r.ReadByte()
	if method == AuthUserPassword {
		// 进行密码验证环节
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

	b, e = r.ReadByte()
	if e != nil {
		return
	}
	if b != SocksVersion {
		e = errversion
		return
	}
	response , e := r.ReadByte()
	if e != nil {
		return
	}
	// 成功 不细分错误代码
	if response != 0x00 {
		return nil, fmt.Errorf("server reponse code is %x", response)
	}
	_, e = r.ReadByte() //保留字节
	if e != nil {
		return
	}
	addr, e = ReadRemoteHost(r)

	return
}




