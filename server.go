package main

import (
	"io"
	"log"
	"net"
	"bard/bard"
	"strconv"
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
			//fmt.Println(tmp)
			// 暂时简单的判断
			if tmp[1] == byte(1) {

				n, err := conn.Write([]byte{bard.SocksVersion, 0x00})
				//conn.Write([]byte{socksVersion, 0x00})
				if err != nil {
					log.Fatalln(err)
				} else {
					//fmt.Println("发送", []byte{bard.SocksVersion, 0x00})
					//fmt.Println("发送了", n,"个数据")

					//ctx, _ := readFully(conn)
					tmp = <-ctx
					pcri := bard.ParseReq(tmp)
					//fmt.Println("bytes: \t", tmp, "\n对应的字符串: \t", string(tmp))
					temp := append([]byte{5,0}, tmp[2:]...)
					_, _ = conn.Write(temp)
					//fmt.Println("回应了请求: ", temp)


					tmp = <-ctx
					//fmt.Println("bytes: \t", tmp, "\n对应的字符串: \n", string(tmp))

					//fmt.Println(string(pcri.Dst.Addr))

					conn2, err := net.Dial("tcp", string(pcri.Dst.Addr)+":"+strconv.Itoa(pcri.Dst.Port))
					defer conn2.Close()
					if err != nil {
						// todo
					}

					var body [102400]byte
					n, err = conn2.Write(tmp)

					n, err = conn2.Read(body[0:])

					conn.Write(body[0: n])

				}
			}
			conn.Close()
		}()
	}
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
