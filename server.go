package main

import (
	"bard/bard"
	"net"
)

const (
	ConfigPath = "./debug/config/config.yml"
	PulginDir = "./debug/plugins"
)


func main() {
	config, err := bard.ParseConfig(ConfigPath)

	bard.Deb.Open = config.Debug
	bard.Slog.Open = config.Slog

	if err != nil {
		bard.Logf("path error: %s is error", ConfigPath)
		return
	}

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

		go bard.ServerHandleConn(conn, config)
	}
}

// 初始化插件
// 把所有的插件总结成一个插件使用
func initPlugin(plugindir string) (bard.IPlugin, error) {
	ps, e := bard.PluginsFromDir(plugindir)
	if e != nil {
		// 要么出问题了，要么插件集合为空
		return nil, e
	}

	var iplugin bard.IPlugin

	// 接下来整合插件
	for k,v := range ps.Pmap {

	}
}
