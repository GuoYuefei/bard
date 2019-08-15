package main

import (
	"bard/bard"
	WP "bard/client/win_plugin"
	WT "bard/client/win_sub_protocol"
	"bufio"
	"fmt"
	"net"
	"runtime"
)

const (
	ConfigPath = "./client/debug/config/config.yml"
	PluginDir = "./client/debug/plugins"
	SubProtocolDir = "./client/debug/sub_protocols"
)

func main() {
	config := doConfig()
	plugins := doPlugin()
	TCSubProtocols := doTCSubProtocol()
	socksSever(config, plugins, TCSubProtocols)

}

// socks5协议接受消息，但是不进行转发，而是交由另个协程发给我们自己的远程代理主机
func socksSever(config *bard.Config, plugins *bard.Plugins, protocols *bard.TCSubProtocols) {
	listener, err := net.Listen("tcp", config.LocalAddress+":"+config.LocalPortString())
	if err != nil {
		//log.Fatalln(err)
		bard.Deb.Printf("Failed to open the proxy server with the following error: %v", err)
		return
	}
	fmt.Printf("Open the local proxy service with the address port of %s:%d\n", config.GetLocalString(), config.LocalPort)
	fmt.Printf("remote proxy service address port is %s:%d\n", config.GetServers()[0], config.ServerPort)
	for {
		netconn, err := listener.Accept()

		if err != nil {
			bard.Logln(err)
			continue
		}

		conn := bard.NewConnTimeout(netconn, config.Timeout)
		go localServerHandleConn(conn, config, plugins, protocols)

	}
}

// 设定的时候权限认证只保留不认证一种方式
func localServerHandleConn(localConn *bard.Conn, config *bard.Config, plugins *bard.Plugins, protocols *bard.TCSubProtocols) {

	// 默认是4k，调高到6k
	r := bufio.NewReaderSize(localConn, bard.BUFSIZE)

	// 握手可以复用 包括Auth过程
	// 客户端不需要在本地连接时无需加入通讯配置，而应该是一个正常的socks5客户端
	err, _ := bard.ServerHandShake(r, localConn, config)

	if err != nil { // 认证失败也会返回错误哦
		return
	}

	pcq, err := bard.ReadPCQInfo(r)

	if err != nil {
		bard.Deb.Println(err)
		// 拒绝请求处理 				// 接受连接处理因为各自连接的不同需要分辨cmd字段之后分辨处理
		bard.RefuseRequest(localConn)
		return
	}
	bard.Deb.Printf("客户端得到的完整的地址是：%s", pcq)

	client, err := bard.NewClient(localConn, pcq, config, plugins, protocols)
	if err != nil || client.PCRsp.Rep != 0x00 {
		if err != nil {
			bard.Logln(err)
		} else {
			bard.Logln("refused by remote server")
		}
		bard.RefuseRequest(localConn)
		return
	}

	defer client.Close()

	client.Pipe()

}

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

func doPlugin() *bard.Plugins {
	if runtime.GOOS == "windows" {
		return winPlugin()
	}
	return otherPlugin()
}

func otherPlugin() *bard.Plugins {
	ps, err := bard.PluginsFromDir(PluginDir)
	if err != nil {
		// 上面函数已有错误处理
		return nil
	}
	//plugin := ps.ToBigIPlugin()
	return ps
}

// todo 以后记得解决
//  windows go语言还不支持插件编译，不知道以后支不支持，暂行方案，直接一起编译把
func winPlugin() *bard.Plugins {
	ps := WP.WinPlugins()
	return ps
}

func doTCSubProtocol() *bard.TCSubProtocols {
	var protocols *bard.TCSubProtocols
	if runtime.GOOS == "windows" {
		// windows
		protocols = WT.WinSubProtocols()
	} else {
		var e error
		// 这是unix-like
		protocols, e = bard.SubProtocolsFromDir(SubProtocolDir)
		if e == bard.SubProtocol_ZERO {
			bard.Deb.Println(e)
		} else if e != nil {
			bard.Logln(e)
		}
	}
	// 整合Default
	protocols.Register(bard.DefaultTCSP)

	return protocols

}


