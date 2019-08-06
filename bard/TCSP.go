package bard

import (
	"net"
)

/**
	本文件为传输控制子协议 依旧为插件形式
	Transmission Control SubProtocol
 */

type ReadDoFuncType = func(net.Conn) ([]byte, int)
type WriteDoFunType = func([]byte) ([]byte, int)

type TCSPReadDo interface {
	// @describe 根据协议从conn中读取控制信息
	// @param conn net.Conn 可以读取的连接
	// @return []byte 本函数从conn中读取的内容
	// @return uint 得到接下来数据包的长度
	ReadDo(conn net.Conn) ([]byte, int)
}

type TCSPWriteDo interface {
	// @describe 根据协议添加控制信息
	// @param	[]byte 原自己切片
	// @return []byte 添加控制信息的切片
	// @return uint 添加控制信息后切片的长度
	WriteDo([]byte) ([]byte, int)
}

/**
	传输控制子协议
 */
type TCSubProtocol interface {
	TCSPReadDo
	TCSPWriteDo
	ID() string
}

type TCSubProtocols struct {
	Tmap map[string] TCSubProtocol
}

func (t *TCSubProtocols)Init() {
	t.Tmap = make(map[string] TCSubProtocol)
}

func (t *TCSubProtocols)Register(protocol TCSubProtocol) {
	t.Tmap[protocol.ID()] = protocol
}

// 可以通过组合AssembleTCSP来拓展它
type AssembleTCSP struct {
	readDo ReadDoFuncType
	writeDo WriteDoFunType
}

func (a *AssembleTCSP)ID() string {
	return "Default"
}

// 将do注册如AssembleTCSP
func (a *AssembleTCSP)RegisterDo(do1 ReadDoFuncType, do2 WriteDoFunType) {
	a.readDo = do1
	a.writeDo = do2
}

func (a *AssembleTCSP)ReadDo(conn net.Conn) ([]byte, int) {
	return a.readDo(conn)
}

func (a *AssembleTCSP)WriteDo(bs []byte) ([]byte, int) {
	return a.writeDo(bs)
}

// 一个默认的TCSubProtocol的Do函数
func DefaultTCSPReadDo(conn net.Conn) ([]byte, int) {
	// default len is two byte
	lslice := make([]byte, 2)
	_, err := ReadFull(conn, lslice)
	if err != nil {
		return nil, 0
	}
	// 大端
	lenh, lenl := int(lslice[0]), int(lslice[1])
	l := lenh<<8+lenl
	//fmt.Println(lenh, lenl, l)
	return lslice, l
}

func DefaultTCSPWriteDo(bs []byte) ([]byte, int) {
	l := len(bs)
	lenh, lenl := byte(l>>8), byte(l)
	//fmt.Println(lenh, lenl)
	lslice := []byte{lenh, lenl}
	bs = append(lslice, bs...)
	return bs, len(bs)
}

// 提供默认的控制子协议
var DefaultTCSP = &AssembleTCSP{
	readDo:  DefaultTCSPReadDo,
	writeDo: DefaultTCSPWriteDo,
}


