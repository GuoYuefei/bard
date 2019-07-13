package main

import (
	"bard/bard"
	"net"
)

const (
	ConfigPath = "./debug/config/config.yml"
	PluginDir = "./debug/plugins"
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
	bard.Logf("Open the proxy service with the address port of %s:%d", config.GetServers()[0], config.ServerPort)
	for {
		netconn, err := listener.Accept()

		if err != nil {
			bard.Logln(err)
			continue
		}

		// 为了timeout重写了一个类型
		conn := bard.NewConnTimeout(netconn, config.Timeout)
		conn.Register(plugin)

		go bard.ServerHandleConn(conn, config)
	}
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

