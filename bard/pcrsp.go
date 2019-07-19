package bard

// proxy connect response information
// 与pcqi类似
type PCRspInfo struct {
	Ver byte
	Rep byte
	RSV byte
	SAddr *Address			//cmd=0x03时为服务器端udp的监听地址， cmd=0x01时
}

// 方法？？？
