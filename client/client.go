package main

import (
	"bard/bard"
	"bufio"
	"fmt"
	"io"
	"net"
	"sync"
)

const (
	ConfigPath = "./server/debug/config/config.yml"
)

func main() {
	fun()

}


// socks5协议接受消息，但是不进行转发，而是交由另个协程发给我们自己的远程代理主机
func socksSever() {
	config := doConfig()
	listener, err := net.Listen("tcp", ":"+config.LocalPortString())
	if err != nil {
		//log.Fatalln(err)
		bard.Logf("Failed to open the proxy server with the following error: %v", err)
		return
	}
	for {
		netconn, err := listener.Accept()

		if err != nil {
			bard.Logln(err)
			continue
		}

		// 为了timeout重写了一个类型
		conn := bard.NewConnTimeout(netconn,  config.Timeout)
		go localServerHandleConn(conn, config)

	}
}

// 设定的时候权限认证只保留不认证一种方式
func localServerHandleConn(conn *bard.Conn, config *bard.Config) {
	defer func() {
		err := conn.Close()
		// timeout 可能会应发错误，原因此时conn已关闭
		if err != nil {
			bard.Logff("Close socks5 connection error, the error is %v", bard.LOG_WARNING, err)
		}
	}()

	// 默认是4k，调高到6k
	r := bufio.NewReaderSize(conn, 6*1024)

	// 握手可以复用 包括Auth过程
	err := bard.ServerHandShake(r, conn, config)

	if err != nil {			// 认证失败也会返回错误哦
		return
	}

	pcq, err := bard.ReadPCQInfo(r)

	if err != nil {
		bard.Deb.Println(err)
		// 拒绝请求处理 				// 接受连接处理因为各自连接的不同需要分辨cmd字段之后分辨处理
		resp := []byte{0x05, 0x05, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
		_, err := conn.Write(resp)
		if err != nil {
			bard.Deb.Printf("refuse connect error:\t", err)
		}
		return
	}
	bard.Deb.Printf("得到的完整的地址是：%s", pcq)

	wg := new(sync.WaitGroup)
	// todo 2 处理返回消息还需要一个协程
	wg.Add(2)



	wg.Wait()
}

// todo 客户端协程独立于是否连接，而取决于是否在message chan中存在消息。并且能根据message的多少自由的扩展协程数量







func doConfig() (config *bard.Config) {
	config, err := bard.ParseConfig(ConfigPath)

	if err != nil {
		bard.Logf("path error: %s is error", ConfigPath)
		return
	}

	bard.Deb.Open = config.Debug
	bard.Slog.Open = config.Slog

	return
}




// client easy example
func fun() {
	netconn, err := net.Dial("tcp", "127.0.0.1:1081")
	if err != nil {
		bard.Deb.Println(err)
		return
	}
	conn := bard.NewConnTimeout(netconn, 300)
	defer conn.Close()
	conn.Write([]byte{bard.SocksVersion, 0x01, 0x00})
	r := bufio.NewReader(conn)
	b, _ := r.ReadByte()
	if b == bard.SocksVersion {
		if b, err = r.ReadByte(); err != nil {
			return
		}else {
			if b == 0xff {
				return
			}
		}
	}
	req := append([]byte{bard.SocksVersion, 0x01, 0x00, 0x03,
		11}, []byte("example.com")...)
	_, _ = conn.Write(append(req, 0, 80))

	readByte, err := r.ReadByte()
	if readByte == bard.SocksVersion {
		i, _ := r.ReadByte()
		if i != 0x00 {
			return
		}
	} else {
		return
	}
	r.Reset(conn)
	_, err = conn.Write([]byte("GET / HTTP/1.1\r\nHost: example.com\r\nUser-Agent: Mozilla/5.0 (Windows; U; Windows NT 5.1; en-US; rv:1.7.6)\r\n"+
		"Gecko/20050225 Firefox/1.0.1\r\nConnection: Keep-Alive\r\n\r\n"))
	if err != nil {
		bard.Deb.Println(err)
		return
	}
	fmt.Println("write cg")
	var result = make([]byte, 1*500)
	n, err := io.ReadFull(r, result)
	if err != nil {
		bard.Deb.Println(err)
		return
	}
	fmt.Println(string(result[0:n]))
	conn.Close()
}
