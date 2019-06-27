package main

import (
	"bard/bard"
	"fmt"
	"log"
	"net"
)

const (
	ConfigPath = "./debug/config/config.yml"
)

func main() {

	config, err := bard.ParseConfig(ConfigPath)
	if err != nil {
		panic(fmt.Sprintf("path error: %s is error", ConfigPath))
	}
	listener, err := net.Listen("tcp", ":"+config.ServerPortString())
	if err != nil {
		log.Fatalln(err)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatalln(err)
		}

		go bard.HandleConn(conn, config)
	}
}
