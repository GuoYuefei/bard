package main

import (
	"bard/bard"
	cPlugin "bard/client/plugin"
	"bufio"
	"fmt"
	"net"
	"runtime"
)

const (
	ConfigPath = "./client/debug/config/config.yml"
	PluginDir = "./client/debug/plugins"
)

func main() {
	config := doConfig()
	plugin := doPlugin()

	socksSever(config, plugin)

}

// socks5协议接受消息，但是不进行转发，而是交由另个协程发给我们自己的远程代理主机
func socksSever(config *bard.Config, plugin bard.IPlugin) {
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
		go localServerHandleConn(conn, config, plugin)

	}
}

// 设定的时候权限认证只保留不认证一种方式
func localServerHandleConn(localConn *bard.Conn, config *bard.Config, plugin bard.IPlugin) {

	// 默认是4k，调高到6k
	r := bufio.NewReaderSize(localConn, bard.BUFSIZE)

	// 握手可以复用 包括Auth过程
	err := bard.ServerHandShake(r, localConn, config)

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

	// todo 请求成功的回复由远程服务器端给结果 由本地服务器修改部分内容发送  这个部分的回复应该由client的DealLocalConn负责

	client, err := bard.NewClient(localConn, pcq, config, plugin)
	if err != nil || client.PCRsp.Rep != 0x00 {
		if err != nil {
			bard.Deb.Println(err)
		} else {
			bard.Deb.Println("refused by remote server")
		}
		bard.RefuseRequest(localConn)
		return
	}

	// todo udp通道部分应该是由哪里负责？  			给client负责

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

func doPlugin() bard.IPlugin {
	if runtime.GOOS == "windows" {
		return winPlugin()
	}
	return otherPlugin()
}

func otherPlugin() bard.IPlugin {
	ps, err := bard.PluginsFromDir(PluginDir)
	if err != nil {
		// 上面函数已有错误处理
		return nil
	}
	plugin := ps.ToBigIPlugin()
	return plugin
}

// todo 以后记得解决
//  windows go语言还不支持插件编译，不知道以后支不支持，暂行方案，直接一起编译把
func winPlugin() bard.IPlugin {
	ps := cPlugin.WinPlugins()
	return ps.ToBigIPlugin()
}


