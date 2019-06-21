package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"golang.org/x/net/proxy"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
)

func main() {
	dialer, err := proxy.SOCKS5("tcp", "127.0.0.1:1080", nil, proxy.Direct)
	dealerr(err, "can not connect to")

	perhost := proxy.NewPerHost(proxy.Direct, dialer)
	perhost.AddHost("www.google.com")
	//perhost.AddHost("www.163.com")
	//perhost.AddHost("www.baidu.com")
	perhost.AddHost("www.facebook.com")
	//perhost.AddHost("es6.ruanyifeng.com")
	//perhost.AddHost("www.zmz2019.com")

	//conn, err := perhost.Dial("tcp", "www.facebook.com:80")
	//
	//defer conn.Close()
	//
	//dealerr(err, "2")

	//_, err = conn.Write([]byte("GET / HTTP/1.0\r\n\r\n"))
	transport := &http.Transport{}
	client := &http.Client{Transport: transport}
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return perhost.Dial(network,addr)
	}
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	//transport.DialTLS = perhost.Dial


	// TODO 当将百度addHost后通过https协议也能获取到完整的网页。 也就是就tls连接而言是由服务器端控制的
	if resp, err := client.Get("https://www.baidu.com"); err != nil {
		log.Fatal(err)
	} else {
		defer resp.Body.Close()
		all, err := ioutil.ReadAll(resp.Body)
		dealerr(err, "5")
		fmt.Println(string(all))

	}



}

func dealerr(err2 error, mess string) {
	if err2 != nil {
		fmt.Fprintln(os.Stderr, mess, err2)
		//os.Exit(1)
	}
}

func readFully(conn net.Conn) ([]byte, error) {
	defer conn.Close()

	result := bytes.NewBuffer(nil)
	var buf [1024]byte
	for {
		n, err := conn.Read(buf[0:])
		result.Write(buf[0:n])
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
	}

	return result.Bytes(), nil

}



