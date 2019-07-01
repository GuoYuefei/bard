package bard

import (
	"fmt"
	"net"
)

const (
	IPV4 uint8 = 0X01
	DOMAIN = 0X03
	IPV6 = 0X04
)


type UDPAddress struct {
	*Address
}

func (u *UDPAddress) Network() string {
	return "udp"
}

type Address struct {
	Atyp byte			// Atyp address type 0x01, 0x02, 0x04
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

	return fmt.Sprintf("%s:%d", a.AddrString(), a.PortToInt())
}
