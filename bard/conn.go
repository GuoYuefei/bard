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
}

func NewConn(conn net.Conn) *Conn {
	c := &Conn{conn, TIMEOUT}
	return c
}

func NewConnTimeout(conn net.Conn, timeout int) *Conn {
	c := &Conn{conn, 0}
	c.SetTimeout(timeout)
	return c
}

func (c *Conn) SetTimeout(second int) {
	c.timeout = time.Duration(second) * time.Second
	_ = c.SetDeadline(time.Now().Add(c.timeout))
}


func (c *Conn) SetDeadline(t time.Time) error {
	err := c.Conn.SetDeadline(t)
	if err != nil {
		Slog.Printf("Conn set deadline error: %v", err)
	}
	return err
}

func (c *Conn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)
	_ = c.SetDeadline(time.Now().Add(c.timeout))
	return
}

func (c *Conn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	_ = c.SetDeadline(time.Now().Add(c.timeout))
	return
}

