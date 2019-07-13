package bard

import "bufio"

const (
	SEND = true
	RECEIVE = false
)

// 处理插件 c函数 De取自Decipher前缀
func dealDeCamouflage(r *bufio.Reader, iPlugin IPlugin) {
	if iPlugin == nil {
		return
	}
	end := iPlugin.EndCam()
	if end == END_FLAG {
		// 表示没有混淆协议加入，不做前处理
		return
	}

	for {
		b, e := r.ReadByte()
		if e != nil {
			// todo
			return
		}
		if b == end {
			break
		}
	}
}


func dealEnCamouflage(bs []byte, plugin IPlugin) ([]byte, int) {
	return 	plugin.Camouflage(bs, SEND)
}
