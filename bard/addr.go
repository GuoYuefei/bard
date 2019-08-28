package bard

import (
	"errors"
	"fmt"
	"net"
)

// @param ip
// @return []byte ip代表的字节数组
// @return int IPV4(0x01) or IPV6(0x04),代表该ip是什么类型 0x00错误时返回
// @return error 错误返回
func IpToBytes(ip net.IP) ([]byte, byte, error){
	srcip := ip.To4()
	srcipType := IPV4
	if srcip == nil {
		srcip = ip.To16()
		srcipType = IPV6
		if srcip == nil {
			return nil, 0x00 ,errors.New("Address error IP cannot be parsed into version 4 or 6 ")
		}
	}
	return srcip, srcipType, nil
}

type UDPAddress struct {
	*Address
}

func (u *UDPAddress) Network() string {
	return "udp"
}

type Address struct {
	Atyp byte			// Atyp address type 0x01, 0x03, 0x04 => ipv4 domain ipv6
	Addr []byte
	Port []byte			// 16 bit
}

func (a *Address) PortToInt() int {
	p := a.Port
	return 256*int(p[0])+int(p[1])
}

// 因为域名这里没有记录下长度，如果用于协议的话，前面需要加域名的长度， 如果是ip则不用加工
func (a *Address) ToProtocolAddr() []byte {
	if a.Atyp&0x02!=0x02 {
		// ip就返回原本的bytes
		return a.Addr
	}

	var domainLen byte = byte(len(a.Addr))
	return append([]byte{domainLen}, a.Addr...)
}

// 这是常规协议回应可能用的bytes结构
func (a *Address) ToProtocol() []byte {
	return append(append([]byte{a.Atyp},  a.ToProtocolAddr()...), a.Port...)
}

func (a *Address) AddrString() string {
	var hostname string
	if !(a.Atyp&0x02==0x02) {
		// 就说明是非域名
		var ip net.IP = a.Addr
		hostname = ip.String()
	} else {
		hostname = string(a.Addr)
	}

	return hostname
}

func (a *Address) String() string {

	if a.Atyp == IPV6 {
		// ipv6 需要写成这样[ipv6]:port
		return fmt.Sprintf("[%s]:%d", a.AddrString(), a.PortToInt())
	} else {
		return fmt.Sprintf("%s:%d", a.AddrString(), a.PortToInt())
	}
}
