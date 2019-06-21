// socks5 协议的一些方法集合
package bard

const (
	SocksVersion = 5

)

type Address struct {
	Atyp byte			// Atyp address type
	Addr []byte
	Port int			// 16 bit
}


/**
客户端发送要建立的代理连接的地址及端口 地址可能是域名、ipv4、ipv6
+----------+------------+---------+-----------+-----------------------+------------+
|协议版本号  | 请求的类型  |保留字段   |  地址类型  |  地址数据              |  地址端口    |
+----------+------------+---------+-----------+-----------------------+------------+
|1个字节    | 1个字节     |1个字节   |  1个字节   |  变长                  |  2个字节    |
+----------+------------+---------+-----------+-----------------------+------------+
|0x05      | 0x01		|0x00     |  0x01     |  0x0a,0x00,0x01,0x0a  |  0x00,0x50 |
+----------+------------+---------+-----------+-----------------------+------------+

 */

// Proxy connection request information
type PCQInfo struct {
	Ver byte  		// version
	Cmd byte		// command
	Rsv byte		// reserve
	Dst Address
}

// 解析请求信息 这个函数在程序中会往复使用，在其后的试探中应该加强其效率
func ParseReq(requset []byte) *PCQInfo {
	var pcqi = &PCQInfo{}
	pcqi.Ver = requset[0]
	pcqi.Cmd = requset[1]
	pcqi.Rsv = requset[2]
	dst := Address{}
	dst.Atyp = requset[3]

	// 地址可能存在几种可能变长的域名，定常的ipv4和ipv6
	switch dst.Atyp {
	case uint8(0x01):
		// ipv4
		dst.Addr = append([]byte{}, requset[4: 8]...)				//取四五六七
		dst.Port = int(requset[8: 9][0]<<8 + requset[9: 10][0])
	case 0x03:
		// domain
		l := requset[4]
		dst.Addr = append([]byte{}, requset[5: 5+l]...)
		dst.Port = int(requset[5+l: 6+l][0]<<8 + requset[6+l: 7+l][0])
	case 0x04:
		// ipv6
		dst.Addr = append([]byte{}, requset[4: 20]...)
		dst.Port = int(requset[20: 21][0]<<8 + requset[21: 22][0])
	}

	pcqi.Dst = dst
	return pcqi
}





