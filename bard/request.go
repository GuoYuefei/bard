package bard

import "bytes"

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

	SCMessage chan *Message
	RemoteConn *Conn
}


