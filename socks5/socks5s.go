package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
)
const (
	socksVersion = uint8(0x05)
)

func main() {
	In, err := net.Listen("tcp", ":1081")


	if err != nil {
		// handle error
	}

	for {
		conn, err := In.Accept()
		defer conn.Close()
		if err != nil {
			// handle error
			continue
		}

		go func() {
			ctx, err := readfully(conn)
			if err != nil {
				log.Println("da wo ya")
				return
			}
			tmp := <-ctx
			fmt.Println(tmp)
			// 暂时简单的判断
			if tmp[1] == byte(1) {

				n, err := conn.Write([]byte{socksVersion, 0x00})
				//conn.Write([]byte{socksVersion, 0x00})
				if err != nil {
					log.Fatalln(err)
				} else {
					fmt.Println("发送", []byte{socksVersion, 0x00})
					fmt.Println("发送了", n,"个数据")

					//ctx, _ := readFully(conn)
					tmp = <-ctx
					fmt.Println("bytes: \t", tmp, "\n对应的字符串: \t", string(tmp))
					_, _ = conn.Write([]byte{5, 0, 0, 3, 11, 101, 120, 97, 109, 112, 108, 101, 46, 99, 111, 109, 0, 80})


					tmp = <-ctx
					fmt.Println("bytes: \t", tmp, "\n对应的字符串: \n", string(tmp))
					file, _ := os.Create("./log.txt")
					defer file.Close()

					_, _ = file.Write(tmp)

					resp, _ := http.Get("http://example.com")
					//defer resp.Body.Close()
					var body [102400]byte
					n, _ = resp.Body.Read(body[0:])
					//tmp := []byte{0, 0, 1, 3, 11, 101, 120, 97, 109, 112, 108, 101, 46, 99, 111, 109, 0, 80}			//这个是udp代理才需要的
					//tmp = append(tmp, body[0:n]...)
					//conn.Write(tmp)
					//fmt.Println(string(body[0:n]))
					conn.Write(body[0:n])
					conn.Close()
				}
			}
		}()
	}
}

// 弃用， 当连接不断时for循环会卡住
func readFully(conn net.Conn) ([]byte, error) {
	//defer conn.Close()
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

func readfully(conn io.ReadCloser)  (<-chan []byte, error) {

	rst := make(chan []byte, 1024)
	var err error
	//result := bytes.NewBuffer(nil)
	var buf [1024]byte					// 这个数字让人纠结
	go func() {
		for {
			var n int
			n, err = conn.Read(buf[0:])
			rst <- buf[0:n]
			if err != nil {
				if err == io.EOF {
					break
				}
				return
			}
		}
	}()
	if err != nil && err != io.EOF {
		return nil, err
	}

	return rst, nil
}


