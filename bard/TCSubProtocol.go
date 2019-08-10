package bard

import (
	"bard/bard-plugin/sub_protocol"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"plugin"
)

/**
	本文件为传输控制子协议 依旧为插件形式
	Transmission Control SubProtocol
 */

const DEFAULTTCSPID = "Default"
// T 作为子协议的symbol name
const SUBP_SYMBOL_NAME = "T"

var SubProtocol_ZERO = errors.New("No valid SubProtocol Plugin under the folder ")

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
	// @return []byte 添加控制信息后的切片
	// @return int 添加控制信息后切片的长度
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

func (t *TCSubProtocols)FindByID(id string) TCSubProtocol {
	if v, ok := t.Tmap[id]; ok {
		return v
	}
	return nil
}

func SubProtocolsFromDir(subProtocolsPath string) (ts *TCSubProtocols, e error) {
	ts = &TCSubProtocols{}
	ts.Init()
	subprotocolsdir, e := os.Open(subProtocolsPath)
	if e != nil {
		Logff("Failed to open folder for sub_protocol plugin: %v", LOG_EXCEPTION, e)
		return
	}

	// names 是文件夹下面所有文件的名字，这时候还要判断是不是.so后缀
	names, e := subprotocolsdir.Readdirnames(0)

	for _, name := range names {
		if !isPluginFile(name) {
			// 不是插件文件就跳过
			continue
		}
		filep := filepath.Join(subProtocolsPath, name)
		pfile, e := plugin.Open(filep)
		if e != nil {
			Logff("Filename: %s,Failed to open sub_protocol plugin: %v", LOG_WARNING, name, e)
			continue
		}
		symbol, e := pfile.Lookup(SUBP_SYMBOL_NAME)
		if e != nil {
			Logff("Filename: %s, Failed to Lookup symbol: %v", LOG_WARNING, name, e)
			continue
		}
		// 这时拿到插件要告诉我们的信息了
		if IP, ok := symbol.(TCSubProtocol); ok {
			ts.Register(IP)
			fmt.Printf("load sub_protocol plugin %s\n", name)
			continue
		} else {
			Logff("Filename: %s, Failed to register sub_protocol plugin", LOG_WARNING, name)
		}
	}

	if len(ts.Tmap) == 0 {
		e = SubProtocol_ZERO
	} else {
		e = nil
	}
	return

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
var DefaultTCSP = sub_protocol.NewAssembleTCSP(
	DEFAULTTCSPID,
	DefaultTCSPReadDo,
	DefaultTCSPWriteDo,
)

