package bard

import (
	"net"
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
	var resp []byte
	p := c.plugin
	// 处理tcp负载数据内容
	resp, n = p.AntiSniffing(b, SEND)

	// 处理添加混淆内容
	resp, n = p.Camouflage(resp, SEND)

	n, err = c.Conn.Write(resp)
	_ = c.SetDeadline(c.GetDeadline())
	return
}

func (c *Conn) Read(b []byte) (n int, err error) {
	// 切片是引用类型 如果在函数内重新赋值（引用本身赋值），那么函数外无改变
	// 为了能够实现net.Conn接口 这里处理负载内容只能是压缩算法

	//var req []byte
	n, err = c.Conn.Read(b)
	//fmt.Println("111111    ", n)
	p := c.plugin
	// todo 可能b会存在太小而无法容下处理后的数据 这里不考虑压缩算法
	// 处理摘除混淆
	_, n = p.Camouflage(b[0:n], RECEIVE)

	//fmt.Println("-----1-------",n)

	// 处理tcp上的数据负载
	_, n = p.AntiSniffing(b[0:n], RECEIVE)
	// 可能会出现len(b) > n的情况，具体看插件实现
	//fmt.Println("----2-----",n)
	_ = c.SetDeadline(c.GetDeadline())
	return
}

