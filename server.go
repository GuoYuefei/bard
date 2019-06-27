package main

import (
	"bard/bard"
	"net"
)

const (
	ConfigPath = "./debug/config/config.yml"
)

func main() {
	// 开启debug
	bard.Deb.Open = true

	// 暂且关闭日志
	//bard.Slog.Open = false


	config, err := bard.ParseConfig(ConfigPath)
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
	for {
		conn, err := listener.Accept()
		if err != nil {
			bard.Logln(err)
			continue
		}

		go bard.HandleConn(conn, config)
	}
}
