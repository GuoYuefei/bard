package bard

import (
	"bytes"
	"fmt"
	"net"
	"runtime"
	"time"
)

const (
	TIMEOUT = 180			// 这个包默认的timeout取3分钟
)

// 该类型实现net.conn接口
type Conn struct {
	net.Conn
	timeout time.Duration
	plugin IPlugin
}

func NewConn(conn net.Conn) *Conn {
	c := &Conn{conn, 0, nil}
	return c
}

func NewConnTimeout(conn net.Conn, timeout int) *Conn {
	c := &Conn{conn, 0, nil}
	c.SetTimeout(timeout)
	return c
}

func (c *Conn) Register(plugin IPlugin) {
	c.plugin = plugin
}

func (c *Conn) Plugin() IPlugin {
	return c.plugin
}

func (c *Conn) SetTimeout(second int) {
	c.timeout = time.Duration(second) * time.Second

	_ = c.SetDeadline(c.GetDeadline())
}

func (c *Conn) GetDeadline() time.Time {
	Deadline := time.Time{}
	if c.timeout > 0 {
		Deadline = time.Now().Add(c.timeout)
	}
	return Deadline
}


func (c *Conn) SetDeadline(t time.Time) error {
	err := c.Conn.SetDeadline(t)
	if err != nil {
		Slog.Printf("Conn set deadline error: %v", err)
	}
	return err
}

func (c *Conn) Write(b []byte) (n int, err error) {
	var resp []byte = b
	var addlen = 0
	blen := len(b)
	p := c.plugin
	if p == nil {
		goto Write
	}

	// 处理tcp负载数据内容
	resp, n = p.AntiSniffing(resp, SEND)
	addlen = n - blen

	// 处理添加混淆内容
	resp, n = p.Camouflage(resp, SEND)
	addlen = n - blen

Write:
	n, err = c.Conn.Write(resp)
	n = n - addlen			// 减去增加的内容才是真实的内容   // node
	_ = c.SetDeadline(c.GetDeadline())
	return
}

func (c *Conn) Read(b []byte) (n int, err error) {
	// 如果没插件那就正常读取
	if c.plugin == nil {
		n, err = c.Conn.Read(b)
		_ = c.SetDeadline(c.GetDeadline())
		return
	}


	// 切片是引用类型 如果在函数内重新赋值（引用本身赋值），那么函数外无改变
	// 为了能够实现net.Conn接口 这里处理负载内容只能是压缩算法

	p := c.plugin
	// 处理摘除混淆
	sep := p.EndCam()
	temp := getWriterBlock(c.Conn, sep)

	//fmt.Printf("混淆报头%d：%s\n", len(temp), temp)
	_, n = p.Camouflage(temp, RECEIVE)
	fmt.Println("数据块大小：", n)

	nr, err := c.Conn.Read(b[:n])
	for nr != n {
		// 如果数据还没完全到达   先让本协程让出时间片 等待一会再读取
		runtime.Gosched()
		i, err := c.Conn.Read(b[nr:n])

		if err != nil {
			return  nr, err
		}
		nr += i
	}

	//fmt.Println(nr)
	_ = c.SetDeadline(c.GetDeadline())

	//fmt.Println("get c:")
	//fmt.Printf("%s\n", b[:n])
	//fmt.Println("-----1-------",n)

	// 处理tcp上的数据负载
	_, n = p.AntiSniffing(b[0:n], RECEIVE)
	//fmt.Println("-------收到ca：\t"+string(b[0:n]))

	return
}

// FIXME first

// 读取时需要还原原发送块
func getWriterBlock(conn net.Conn, sp []byte) []byte {
	// s 77 c 56 todo 解决这里的效率问题
	source := make([]byte, 56*len(sp))
	conn.Read(source)
	for {
		if bytes.Index(source, sp) > -1 {
			break
		}
		source = ReadByteAppend(conn, source)
	}
	// 此时得到的是混淆的头部，还需要根绝头部读取




	return source
}

func ReadByteAppend(conn net.Conn, source []byte) []byte {
	temp := make([]byte, 1)
	conn.Read(temp)
	bs := append(source, temp...)
	return bs
}