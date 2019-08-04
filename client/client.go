package main

import (
	"bard/bard"
	"bard/client/plugin"
	"bufio"
	"fmt"
	"io"
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
	listener, err := net.Listen("tcp", ":"+config.LocalPortString())
	if err != nil {
		//log.Fatalln(err)
		bard.Deb.Printf("Failed to open the proxy server with the following error: %v", err)
		return
	}
	bard.Deb.Printf("Open the local proxy service with the address port of %s:%d\n", config.GetLocalString(), config.LocalPort)
	bard.Deb.Printf("remote proxy service address port is %s:%d\n", config.GetServers()[0], config.ServerPort)
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
	//defer func() {
	//	err := localConn.Close()
	//	// timeout 可能会应发错误，原因此时conn已关闭
	//	if err != nil {
	//		bard.Logff("Close socks5 connection error, the error is %v", bard.LOG_WARNING, err)
	//	}
	//}()

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
	return plugin.V
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
		} else {
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
	_, err = conn.Write([]byte("GET / HTTP/1.1\r\nHost: example.com\r\nUser-Agent: Mozilla/5.0 (Windows; U; Windows NT 5.1; en-US; rv:1.7.6)\r\n" +
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
