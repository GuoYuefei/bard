// socks5 协议的一些方法集合
package bard

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
)

/**
	这个文件是server程序独占。 考虑是否移除到包外
 */

const (
	SocksVersion = 0x05
	REFUSE = 0xff
	NOAUTH = 0x00
	AuthUserPassword = 0x02				//RFC1929
	UPSubProtocolVer = 0x01			// 子协议版本
)

var ErrorAuth = errors.New("Authentication failed")

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

func ServerHandShake(r *bufio.Reader, conn net.Conn, config *Config) error {
	version, err := r.ReadByte()

	if err != nil {
		return err
	}

	if version != SocksVersion {
		return fmt.Errorf("socks's version is %d", version)
	}

	nmethods, _ := r.ReadByte()
	Deb.Printf("Methods' Lenght is %d", nmethods)

	buf := make([]byte, nmethods)

	_, err = io.ReadFull(r, buf)
	if err != nil {
		return err
	}
	Deb.Printf("验证方式为： %v", buf)

	resp, ok := Auth(buf, r, conn, config)
	if !ok {
		Logf("Connection request from client IP %v, permission authentication failed", conn.RemoteAddr())
	}

	_, err = conn.Write(resp)
	if err != nil {
		return err
	}
	if !ok {
		return ErrorAuth
	}

	return nil
}

/**
	暂且支持最基本的两种方式 0x00, 0x02
	其中0x02 使用0x01子协议	RFC1929
	@param authMethods 		客户端支持的验证方式
	@param config 				authMethod		服务器选择的验证方式和账户密码的配置信息
	@param r conn				都是代表那个连接
	@return []byte 			返回能最后一步需要回复应该要发送的认证回复的代码
 */
func Auth(authMethods []byte, r *bufio.Reader, conn net.Conn, config *Config) ([]byte, bool) {
	for _, v := range authMethods {
		if v == config.AuthMethod {
			switch v {
			case NOAUTH: 					// 无需认证
				return []byte{SocksVersion, NOAUTH}, true
			case AuthUserPassword:
				// 用户名和密码 认证
				return UserPassWD(r, conn, config.Users)
			default:					// 无验证方式 就拒绝连接
				goto Refuse
			}
		}
	}
	// should do something here
	Refuse:
		return []byte{SocksVersion, REFUSE}, false
}

func UserPassWD(r *bufio.Reader, conn net.Conn, users []*User) ([]byte, bool) {
	var (
		subProtocolVer byte
		ulen byte
		uname []byte
		plen byte
		passwd []byte
	)
	_, err := conn.Write([]byte{SocksVersion, AuthUserPassword})
	if err != nil {
		Logln("write auth method error:", err)
		goto Refuse
	}

	subProtocolVer, err = r.ReadByte()


	if subProtocolVer != UPSubProtocolVer {
		Logf("The User/Password sub-protocol version is %d, not %d", subProtocolVer, UPSubProtocolVer)
		goto Refuse		// 0x01代表拒绝  协议版本都对不上，小样还想连接
	}

	ulen, err = r.ReadByte()

	if err != nil {
		Logln("read len of username error:", err)
		goto Refuse
	}

	uname = make([]byte, ulen)
	_, err = io.ReadFull(r, uname)
	if err != nil {
		Logln("read uname error:", err)
		goto Refuse
	}

	plen, err = r.ReadByte()
	if err != nil {
		Logln("read len of passwd error:", err)
		goto Refuse
	}

	passwd = make([]byte, plen)
	_, err = io.ReadFull(r, passwd)
	if err != nil {
		Logln("read passwd error:", err)
		goto Refuse
	}
	Deb.Println(string(uname), string(passwd))

	for _, v := range users {
		Deb.Println(v.Username, v.Password)
		if v.Username == string(uname) && v.Password == string(passwd) {
			return []byte{UPSubProtocolVer, 0x00}, true				// 认证成功
		}
	}
	// 账号密码不正确 就执行Refuse

Refuse :
		return []byte{UPSubProtocolVer, 0x01}, false

}

func ReadRemoteHost(r *bufio.Reader) (*Address, error) {
	var err error
	address := &Address{}
	addrType, err := r.ReadByte()
	var port [2]byte

	if err != nil {
		goto ErrorReturn
	}

	address.Atyp = addrType

	switch addrType {
	case 0x01:
		var ip net.IP = make([]byte, 4)
		_, err = io.ReadFull(r, ip)
		address.Addr = ip

	case 0x03:
		domainLen, _ := r.ReadByte()
		var domain []byte = make([]byte, domainLen)
		_, err = io.ReadFull(r, domain)
		address.Addr = domain

	case 0x04:
		var ip net.IP = make([]byte, 16)
		_, err = io.ReadFull(r, ip)
		address.Addr = ip
	}
	if err != nil {
		goto ErrorReturn
	}


	_, err = io.ReadFull(r, port[0:])
	if err != nil {
		goto ErrorReturn
	}
	address.Port = port[0:]

	return address, nil

	ErrorReturn:
		return nil, err

}

func ReadPCQInfo(r *bufio.Reader) (*PCQInfo, error) {
	version, _ := r.ReadByte()

	if version != SocksVersion {
		return nil, fmt.Errorf("This is not the Socks5 protocol, version is %d", version)
	}

	cmd, _ := r.ReadByte()

	Deb.Println("socks' cmd:\t",cmd)
	if cmd&0x01 != 0x01 {
		// todo 现在仅支持0x03 and 0x01 即非bind请求
		return nil, errors.New("客户端请求类型不为1或者3， 暂且只支持代理连接和udp")
	}


	rsv, err := r.ReadByte()		//保留字段

	if err != nil {
		return nil, err
	}

	// address应该能传出去的
	address, err := ReadRemoteHost(r)
	if err != nil {
		return nil, err
	}

	return &PCQInfo{version, cmd,0x00,rsv, address}, nil
}



func dealCamouflage(r *bufio.Reader, iPlugin IPlugin) {
	end := iPlugin.EndCam()

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


func ServerHandleConn(conn net.Conn, config *Config, plugin IPlugin) {
	defer func() {
		err := conn.Close()
		// timeout 可能会应发错误，原因此时conn已关闭
		if err != nil {
			Logff("Close socks5 connection error, the error is %v", LOG_WARNING, err)
		}
	}()

	r := bufio.NewReader(conn)

	// 移除混淆和伪装
	dealCamouflage(r, plugin)

	err := ServerHandShake(r, conn, config)

	if err != nil {			// 认证失败也会返回错误哦
		return
	}

	// TODO 权限认证

	pcq, err := ReadPCQInfo(r)
	if err != nil {
		Deb.Println(err)
		// 拒绝请求处理
		resp := []byte{0x05, 0x05, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
		_, err := conn.Write(resp)
		if err != nil {
			Deb.Printf("refuse connect error:\t", err)
		}
		return
	}
	Deb.Printf("得到的完整的地址是：%s", pcq)
	err = pcq.HandleConn(conn, r, config)
	if err != nil {
		Deb.Println(err)
		return
	}

	//remote.Close()
	//conn.Close()
}






