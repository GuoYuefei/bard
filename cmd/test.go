package main

import (
	"fmt"
	"net"
)

func main() {
	var a = []byte{0,1,2,3,4,5}
	b := a[1:3]		//证明切片这种方式运算只是指针赋值
	a[1] = 2
	fmt.Println(b)
	fmt.Println(a)
	conn, err := net.Dial("tcp", "zhihan.me:80")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(conn)
}
