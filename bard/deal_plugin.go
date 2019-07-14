package bard

// node 原本想废弃，现留一有用函数

// 处理插件 c函数 De取自Decipher前缀
//func dealDeCamouflage(r *bufio.Reader, iPlugin IPlugin) {
//	if iPlugin == nil {
//		return
//	}
//	end := iPlugin.EndCam()
//	if end == END_FLAG {
//		// 表示没有混淆协议加入，不做前处理
//		return
//	}
//
//	for {
//		b, e := r.ReadByte()
//		if e != nil {
//			// todo
//			return
//		}
//		if b == end {
//			break
//		}
//	}
//}


func dealEnCamouflage(bs []byte, plugin IPlugin) ([]byte, int) {
	return 	plugin.Camouflage(bs, SEND)
}

func dealEnAntiSniffing(bs []byte, plugin IPlugin) ([]byte, int) {
	return plugin.AntiSniffing(bs, SEND)
}

func dealDeAntiSniffing(bs []byte, plugin IPlugin) ([]byte, int) {
	return plugin.AntiSniffing(bs, RECEIVE)
}

func dealOrnament(send Send, plugin IPlugin) FunOrnament {
	if plugin == nil {
		return nil
	}
	return func(bs []byte) (bytes []byte, i int) {
		return plugin.Ornament(bs, send)
	}
}
