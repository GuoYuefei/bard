package bard

import (
	"bufio"
	"io"
	"net"
)

const (
	REFUSE uint8 = 0xff


	// 认证方式
	NOAUTH uint8 = 0x00
	AuthUserPassword uint8 = 0x02				//RFC1929
	UPSubProtocolVer uint8 = 0x01			// 上面子协议版本
)

/**
@description 	暂且支持最基本的两种方式 0x00, 0x02
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


/**
@description 0x02 使用用户密码验证的子协议 协议版本0x01   对应RFC1929
@param r是经由conn包装而来
@param conn 连接接口类型
@param users是记录所有用户对象的指针的集合
@return []byte 是认证之后应该返回客户端的代码  可以分成拒绝和接受两种
@return bool 返回接受连接与否
 */
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
