package bard

import (
	"fmt"
	"io/ioutil"
	"net"
	"testing"
)

func TestConn_Write(t *testing.T) {
	plugin := doPlugin(PLUGIN_DIR_C)
	// 开客户端连续发送数据
	tcpconn, e := net.Dial("tcp", ":9999")
	if e != nil {
		return
	}
	conn := NewConn(tcpconn)
	conn.Register(plugin)
	for i := 0; i < 6; i++ {
		bytes, _ := ioutil.ReadFile("../client/debug/config/config.yml")
		fmt.Println(conn.Write(bytes))
	}
}

func doPlugin(PluginDir string) IPlugin {
	ps, err := PluginsFromDir(PluginDir)
	if err != nil {
		// 上面函数已有错误处理
		return nil
	}
	plugin := ps.ToBigIPlugin()
	return plugin
}

func TestConn_Read(t *testing.T) {
	// 开服务器读取数据
	plugin := doPlugin(PLUGIN_DIR_S)
	listener, e := net.Listen("tcp", ":9999")
	if e != nil {
		fmt.Println(e)
		return
	}
	tcpconn, e := listener.Accept()
	conn := NewConn(tcpconn)
	conn.Register(plugin)
	if e != nil {
		fmt.Println(e)
		return
	}
	message := &UdpMessage{}
	_, e = Pipe(message, conn, nil)
	if e != nil {
		fmt.Println(e)
		return
	}
	//fmt.Println(string(message.Data))

}
