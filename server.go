package main

import (
	"bard/bard"
	"log"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", ":1081")
	if err != nil {
		log.Fatalln(err)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatalln(err)
		}

		go bard.HandleConn(conn)
	}
}
