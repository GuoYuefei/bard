package bard

import (
	"bufio"
	"bytes"
	"errors"
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

// endtodo !!!!! first
// LOCAL -> CSM -> REMOTE
// REMOTE -> SCM -> LOCAL
// PCQI 有请求的所有信息
// 如果是udp需要生成Packet类型，应该说要组合Packet之后重写Listen
type Client struct {
	config *Config

	LocalConn *Conn
	CSMessage chan *Message

	PCQI *PCQInfo					// node 这是LocalConn得到的请求 这个内容其实只要原封不动传给远程代理就行
	// todo 以下addr需要改成PCRspInfo类型
	PCRsp *PCRspInfo			// node 这是RemoteConn远程服务器发回的响应	服务器返回响应，仅udp代理时有用,先不考虑udp

	SCMessage chan *Message
	RemoteConn *Conn
}

func (c *Client)CheckUDP() {
	// todo 检查是不是udp连接 如果是，就为Client添加udp通道所需要的属性
}

func (c *Client)Pipe() {
	wg := &sync.WaitGroup{}
	e := c.PCQI.Response(c.LocalConn, c.config.GetLocalString())
	if e != nil {
		Deb.Println(e)
		wg.Done(); wg.Done()			// 发生错误还需要解锁的
		return
	}
	go func() {
		defer wg.Done()
		// 转发给远程主机，此时应该将客户端拿来的东西给解密，解密是在read之后，所以该过程是最后处理的函数
		written, e := Pipe(c.RemoteConn, c.LocalConn, nil)
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
		// 转发给远程主机，此时应该将客户端拿来的东西给解密，解密是在read之后，所以该过程是最后处理的函数
		written, e := Pipe(c.LocalConn, c.RemoteConn, nil)
		if e != nil {
			Deb.Printf("LocalConn -> RemoteConn失败: %v", e)
		} else {
			Deb.Printf("LocalConn -> RemoteConn 复制了%dB信息", written)
		}
		// todo
		e = c.LocalConn.Close()
		if e != nil {
			Logff(ExceptionTurnOffRemoteTCP.Error()+"%v", LOG_WARNING, e)
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
	remoteConn, pcrsp, err := NewRemoteConn(config, pcqi)
	if err != nil {
		return
	}
	c.config = config
	c.RemoteConn = remoteConn
	c.LocalConn = localConn
	c.PCQI = pcqi

	// 和udp通道不同，udp通道两个出口或入口都是相同协议的。 这边是双协议所以需要两个通道
	c.CSMessage = make(chan *Message, MESSAGESIZE)
	c.SCMessage = make(chan *Message, MESSAGESIZE)

	c.PCRsp = pcrsp

	return
}

// 这个就是想
func NewRemoteConn(config *Config, pcqi *PCQInfo) (remoteConn *Conn, pcrsp *PCRspInfo, err error) {
	conn, err := net.Dial("tcp", config.GetServers()[0]+":"+config.ServerPortString())
	if err != nil {
		return
	}
	remoteConn = NewConnTimeout(conn, config.Timeout)

	r := bufio.NewReader(conn)

	pcrsp, err = ClientHandleShakeWithRemote(r, remoteConn, pcqi)

	return
}

// 与远程代理服务器握手
func ClientHandleShakeWithRemote(r *bufio.Reader, conn *Conn, pcqi *PCQInfo) (pcrsp *PCRspInfo,e error) {
	conn.Write([]byte{SocksVersion, 0x02, NOAUTH, AuthUserPassword})
	b, e := r.ReadByte()
	if e != nil {
		return
	}
	if b != SocksVersion {
		e = ErrorSocksVersion
		return
	}
	method, e := r.ReadByte()
	if e != nil {
		return
	}
	if method == AuthUserPassword {
		// todo 进行密码验证环节
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




