package bard

import (
	"io"
)

type FunOrnament = func([]byte) ([]byte, int)


func Pipe(dst io.Writer, src io.Reader, ornament FunOrnament) (written int64, err error) {
	return PipeBuffer(dst, src, nil, ornament)
}



// 参照io.copyBuffer
// ornament 用于将来插件注册使用
func PipeBuffer(dst io.Writer, src io.Reader, buf []byte, ornament FunOrnament) (written int64, err error) {

	if ornament == nil {
		// 点缀函数如果不存在的话
		ornament = func(bytes []byte) ([]byte, int) {
			// nothing to do
			return bytes, len(bytes)
		}
	}

	if buf == nil {
		size := 32 * 1024
		if l, ok := src.(*io.LimitedReader); ok && int64(size) > l.N {
			if l.N < 1 {
				size = 1
			} else {
				size = int(l.N)
			}
		}
		buf = make([]byte, size)
	}
	for {
		nr, er := src.Read(buf)
		//fmt.Println(nr)
		// 数据处理
		buf, n := ornament(buf[0:nr])

		//if nr != n {
		//	fmt.Printf("长度： %d\nadd udp proxy head %v\n", n, buf[0:n])
		//}

		if nr > 0 {
			nw, ew := dst.Write(buf[0:n])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if n != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
}
