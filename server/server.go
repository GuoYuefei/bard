package main

import (
	"bard/bard"
	"bufio"
	"fmt"
	"net"
)

const (
	ConfigPath = "./server/debug/config/config.yml"
	PluginDir = "./server/debug/plugins"
)


func main() {
	config := doConfig()

	plugin := doPlugin()

	listener, err := net.Listen("tcp", ":"+config.ServerPortString())
	if err != nil {
		//log.Fatalln(err)
		bard.Logf("Failed to open the proxy server with the following error: %v", err)
		return
	}
	fmt.Printf("Open the proxy service with the address port of %s:%d\n", config.GetServers()[0], config.ServerPort)
	for {
		netconn, err := listener.Accept()

		if err != nil {
			bard.Logln(err)
			continue
		}

		// 为了timeout重写了一个类型
		conn := bard.NewConnTimeout(netconn, config.Timeout)
		conn.Register(plugin)

		go remoteServerHandleConn(conn, config)
	}
}

func remoteServerHandleConn(conn *bard.Conn, config *bard.Config) {
	defer func() {
		err := conn.Close()
		// timeout 可能会应发错误，原因此时conn已关闭
		if err != nil {
			bard.Logff("Close socks5 connection error, the error is %v", bard.LOG_WARNING, err)
		}
	}()

	// 默认是4k，调高到6k
	r := bufio.NewReaderSize(conn, 6*1024)

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
	err = pcq.HandleConn(conn, config)
	if err != nil {
		bard.Deb.Println(err)
		return
	}

	//remote.Close()
	//conn.Close()
}

//------------------ 初始化函数------------------

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

func doPlugin() bard.IPlugin {
	ps, err := bard.PluginsFromDir(PluginDir)
	if err != nil {
		// 上面函数已有错误处理
		return nil
	}
	plugin := ps.ToBigIPlugin()
	return plugin
}

