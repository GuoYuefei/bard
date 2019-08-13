package bard

import (
	"bytes"
	"errors"
	"net"
	"runtime"
	"time"
)

const (
	TIMEOUT = 180			// 这个包默认的timeout取3分钟
	//ReadFullMaxTimes = 100000000
	WaitTimes = 10
)

// 该类型实现net.conn接口
type Conn struct {
	net.Conn
	timeout time.Duration
	plugin IPlugin
	protocol TCSubProtocol
}

func NewConn(conn net.Conn) *Conn {
	c := &Conn{conn, 0, nil, nil}
	return c
}

func NewConnTimeout(conn net.Conn, timeout int) *Conn {
	c := &Conn{conn, 0, nil, nil}
	c.SetTimeout(timeout)
	return c
}

// 该注册方式是覆盖型的
func (c *Conn) Register(plugin IPlugin, protocol TCSubProtocol) {
	c.plugin = plugin
	c.protocol = protocol
}

func (c *Conn) Protocol() TCSubProtocol {
	return c.protocol
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

	var writeDo = func(blen int, resp []byte) ([]byte, int) {
		// 在加密和混淆之间加入自定义的控制信息，主要需要知道加密数据块的长度
		resp, n = c.protocol.WriteDo(resp)
		addlen := n - blen
		return resp, addlen
	}


	var resp []byte = b
	var addlen = 0
	blen := len(b)
	p := c.plugin
	if p == nil {
		if c.protocol != nil {
			resp, addlen = writeDo(blen, resp)
		}
		goto Write
	}
	/**
	******************************有插件处理部分*************************************
	 */

	// 处理tcp负载数据内容
	resp, n = p.AntiSniffing(resp, SEND)
	addlen = n - blen

	resp, addlen = writeDo(blen, resp[0:n])

	// 处理添加混淆内容
	resp, n = p.Camouflage(resp, SEND)
	addlen = n - blen

Write:
	n, err = c.Conn.Write(resp)
	n = n - addlen			// 减去增加的内容才算是写入数据的长度
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

	/**
	******************************有插件处理部分*************************************
	 */


	// 切片是引用类型 如果在函数内重新赋值（引用本身赋值），那么函数外无改变
	// 为了能够实现net.Conn接口 这里处理负载内容只能是压缩算法

	p := c.plugin
	// 处理摘除混淆
	sep := p.EndCam()
	// 如果为默认EndCam就不做处理
	if len(sep)!=1 || sep[0] != END_FLAG[0] {
		_, err := getWriterBlock(c.Conn, sep)

		if err != nil {
			return 0, err
		}
	}

	// io.EOF会返回[0,0], 0. 其他错误nil， 0
	_, n = c.protocol.ReadDo(c.Conn)

	if n == 0 {
		return 0, errors.New("read package len error or io.EOF")
	}

	n, err = ReadFull(c.Conn, b[:n])

	_ = c.SetDeadline(c.GetDeadline())

	// 处理tcp上的数据负载
	_, n = p.AntiSniffing(b[0:n], RECEIVE)

	if err != nil {
		Logf("ReadFull err: %s\n", err)
	}

	return
}

// 读取时需要还原原发送块
func getWriterBlock(conn net.Conn, sp []byte) ([]byte, error) {
	//       方案二：conn第一个读到的双字节代表混淆长度   因为网络包反正都是分包发的，防火墙无法识别这个双字节是上次的携带数据还是这次的信息数据
	// 56*len(sp)+1
	source := make([]byte, 1)
	var err error
	//conn.Read(source)				// fixed 可能读不完全的情况， 应该修复
	_, err = ReadFull(conn, source)
	if err != nil {
		return source, err
	}
	for {
		if bytes.Index(source, sp) > -1 {
			break
		}
		source, err = ReadByteAppend(conn, source)
		if err != nil {
			return source, err
		}
	}
	// 此时得到的是混淆的头部，还需要根绝头部读取

	return source, nil
}

// 另一种解决方案 返回的是混淆头部 双字节 大端字节序   有特征 
func getWriterBlock1(conn net.Conn) ([]byte, error) {
	headlen := make([]byte, 2)
	//n, err := conn.Read(headlen)
	_, err := ReadFull(conn, headlen)
	if err != nil {
		return headlen, err
	}
	var hlh uint = uint(headlen[0])
	var hll uint = uint(headlen[1])
	// headlenght
	hl := hlh << 8 + hll
	head := make([]byte, hl)

	_, err = ReadFull(conn, head)
	if err != nil {
		return head, err
	}
	return head, nil

}

// 出错 or 读满bs结束
func ReadFull(conn net.Conn, bs []byte) (n int, err error) {
	//var times = 0
	lens := len(bs)
	var i = 0
	//  && times < ReadFullMaxTimes
	for n != lens {
		//times++
		i, err = conn.Read(bs[n:])
		n += i
		if err != nil {
			break
		}

		if n == lens {
			break
		}

		// 没读取完就让出时间片等下读
		for i := 0; i < WaitTimes; i++ {
			runtime.Gosched()
		}
	}
	return n, err
}

func ReadByteAppend(conn net.Conn, source []byte) ([]byte, error) {
	temp := make([]byte, 1)
	_, err := ReadFull(conn, temp)
	if err != nil {
		return source, err
	}

	bs := append(source, temp...)
	return bs, nil
}