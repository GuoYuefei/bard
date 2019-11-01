package bard

// proxy connect response information
// 与pcqi类似
type PCRspInfo struct {
	Ver byte					// 协议号
	Rep byte					// 状态码
	RSV byte					// 保留字段
	SAddr *Address			// 地址信息 请求时cmd=0x03时为服务器端udp的监听地址， cmd=0x01时无用
}

// 方法？？？
