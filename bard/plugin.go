package bard

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"sort"
	"strconv"
	"strings"
)

// 每个插件实现必须在服务器端和客户端都实现一遍 加码和解码需要对称

// !!!强制规定，所有插件都必须以V作为Symbol Name暴露出来
const (
	SYMBOL_NAME = "V"
)

var END_FLAG = []byte{0xff}				// 默认bigplugin的EndCam返回内容。表示不作处理


const (
	SEND = true
	RECEIVE = false
)

var PLUGIN_ZERO = errors.New("No valid plugins under the folder")

type IPluFun func([]byte, bool) ([]byte, int)

type Send = bool

// 应该要设置热插拔
// 对io接口需要结合Pipe.go中的函数使用
type IPlugin interface {

	// node EndCam 废弃
	// todo 每个函数都有两个状态，一个是接收时怎么做一个是发送时怎么做
	// todo 所以函数签名还是要改

	// 其中为了实现Camouflage还需要一个函数，表示混淆协议的结束符号
	// !!!! 函数废弃，正好做保留字段
	EndCam() []byte

	// 下面三函数的bool都表示是否是send消息，是执行send消息处理部分，否就执行get消息之后处理部分
	// node 注意 三个函数 在Send为false的情况下都不应该改变[]byte参数的引用，否则将不起作用. send=false此时返回值[]byte应该和传入形参的引用相同 return原实参引用
	// node 注意 当Send为true时返回值的[]byte可以与传入形参的实参的引用不同
	// node 所以 此时我们A函数不提倡使用压缩算法 否则在Receive的时候传入切片将不够用，会出现错误 经过C A函数处理后，长度应该要小于实参cap
	// node 注意 我这边一直强调是引用，而非内容

	// 伪装、混淆， 在socks5协议之前伪装协议头
	// 也就是在socks5之前加一个啥协议什么的
	// 仅在socks5握手时有用 在接收时第一个返回数据为各分块的长度，一个长度占两字节，大端存取
	Camouflage([]byte, Send) ([]byte, int)

	// 防嗅探，
	// socks5握手阶段开始每次socks5连接的io都要经过这个函数。 可以操作传输层之上的所有内容
	AntiSniffing([]byte, Send) ([]byte, int)

	// 操作传输内容
	// 这个主要是用于操作远程服务器和客户端主机之间传送的内容 不包括socks5本身
	// node 如果启用了A函数，请不要再启用O函数				A函数会将socks5协议加密，更加安全
	Ornament([]byte, Send) ([]byte, int)

	// 优先级，越是优先越后运行	0是最高优先级
	// !!! 一个重要的解释：前面三位是保留位
	// 当十六位0001,xxxx,xxxx,xxxx这样格式的，为在socks5协议之前的混淆协议，这时启用Camouflage函数，
	// 当十六位0010,xxxx,xxxx,xxxx格式的，为加密或操作socks协议本身，这个主要防止在建立socks5连接阶段被嗅探,启用AntiSniffing
	// 当十六位0100,xxxx,xxxx,xxxx格式的，为加密或操作传输内容的 启用Ornament函数
	// 每四位表示对应函数的优先级001->最右四位，以此类推
	// 优先级相同时会随机在前在后 这不利于客户端解码 所以请各个插件各自权衡好优先级,也可以配置文件形式给出留给用户设定
	Priority() uint16

	// 判定是是否是同一个插件 在Plugins中用作key以保证同一款插件只被加载一次
	GetID() string

	// 判定插件的Version 如果有多款插件在插件文件夹下面就启用高版本的插件
	// 插件版本形式0.0.0 一般大版本号.小版本号.补丁版本号
	Version() string
}

// TODO 需要一个分析优先级的函数
// 先定义函数返回值类型 一个uint8类型，来说明插件的作用函数是哪些
// 再返回会一个长度是3的byte切片 说明C A O三函数的优先级
func pluginPriority(iPlugin IPlugin) (uint8, []*sortPriFun) {
	pris := make([]*sortPriFun, 3)
	var prs = uint8(iPlugin.Priority()>>12)

	// 下面的元素放入切片是没有做check， 所以需要外层函数对返回的prs做check
	// 插件中的Camouflage函数可用
	pris[0] = &sortPriFun{uint8(0x000f & iPlugin.Priority()), iPlugin.Camouflage, iPlugin.GetID()}
	// A
	pris[1] = &sortPriFun{uint8((0x00f0 & iPlugin.Priority()) >> 4), iPlugin.AntiSniffing, iPlugin.GetID()}
	// O
	pris[2] = &sortPriFun{uint8((0x0f00 & iPlugin.Priority()) >> 8), iPlugin.Ornament, iPlugin.GetID()}

	return prs, pris
}

// 排序时的中间类型不暴露
type sortPriFun struct {
	pri uint8			// 表示下面函数的优先级
	fun IPluFun			// 一个插件中的函数
	id string			// 表示该插件的id
}


type sortPriFuns []*sortPriFun

// 用于sort必须实现接口
func (s sortPriFuns) Len() int {
	return len(s)
}

func (s sortPriFuns) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// 从大到小
func (s sortPriFuns) Less(i, j int) bool {
	return s[i].pri > s[j].pri
}



// 根据优先级排序
// 返回的是三个函数根据优先级排好序的函数数组
func (p *Plugins)SortPriority() (EC func() []byte,Cs []IPluFun, As []IPluFun, Os []IPluFun){

	var Cfuns sortPriFuns = make([]*sortPriFun, 0, len(p.Pmap))
	var Afuns sortPriFuns = make([]*sortPriFun, 0, len(p.Pmap))
	var Ofuns sortPriFuns = make([]*sortPriFun, 0, len(p.Pmap))
	for _, v := range p.Pmap {
		u, fun := pluginPriority(v)
		// 三if可以将无效函数排除在排序之外
		if u & 0x01 == 0x01 {
			Cfuns = append(Cfuns, fun[0])
		}

		if u & 0x02 == 0x02 {
			Afuns = append(Afuns, fun[1])
		}

		if u & 0x04 == 0x04 {
			Ofuns = append(Ofuns, fun[2])
		}
	}
	sort.Sort(Cfuns)
	sort.Sort(Afuns)
	sort.Sort(Ofuns)

	Cs = make([]IPluFun, len(Cfuns))
	As = make([]IPluFun, len(Afuns))
	Os = make([]IPluFun, len(Ofuns))
	for k, v := range Cfuns {
		Cs[k] = v.fun
	}
	for k, v := range Afuns {
		As[k] = v.fun
	}
	for k, v := range Ofuns {
		Os[k] = v.fun
	}

	if  Cfuns.Len() != 0 {
		// 如果存在可用的伪装函数，那么EndCam可用
		EC = p.Pmap[Cfuns[len(Cfuns)-1].id].EndCam
	} else {
		// 如果伪装函数不存在，就返回一个不是标记的标记
		EC = func() []byte {
			return END_FLAG
		}
	}


	return EC, Cs, As, Os
}

// 最后返回根据是否生效，以及各插件指定函数的优先级分别返回三个总函数
func (p *Plugins)GetCAO() (EC func() []byte, C IPluFun, A IPluFun, O IPluFun) {
	EC, Cs, As, Os := p.SortPriority()

	var genCAO = func(ss []IPluFun) (s IPluFun) {
		s = func(in []byte, send bool) (out []byte, l int) {
			out = in
			l = len(in)
			for _, v := range ss {
				out, l = v(out, send)
			}
			return
		}
		return
	}

	C = genCAO(Cs)
	A = genCAO(As)
	O = genCAO(Os)
	return
}

type bigIPlugin struct {
	EC func() []byte
	C IPluFun
	A IPluFun
	O IPluFun
}

func (b *bigIPlugin)EndCam() []byte {
	return b.EC()
}

func (b *bigIPlugin)Camouflage(bs []byte, send bool) ([]byte, int) {
	return b.C(bs, send)
}
func (b *bigIPlugin)AntiSniffing(bs []byte, send bool) ([]byte, int) {
	return b.A(bs, send)
}
func (b *bigIPlugin)Ornament(bs []byte, send bool) ([]byte, int) {
	return b.O(bs, send)
}
// 以下三函数只为实现接口
func (b *bigIPlugin)Priority() uint16 {
	return 0x7000
}
func (b *bigIPlugin)GetID() string {
	return "v"
}
func (b *bigIPlugin)Version() string {
	return "1.0.0"
}


// 将所有的插件按照各自的各自三函数优先级重组成一个IPlugin返回 这是除三函数以外接口中的其他方法无意义
func (p *Plugins)ToBigIPlugin() IPlugin {
	// 内部类型， 不暴露
	EC, C, A, O := p.GetCAO()
	BP := &bigIPlugin{EC, C, A, O}
	return BP
}



type Plugins struct {
	Pmap map[string]IPlugin
}

func (p *Plugins) Init() {
	p.Pmap = make(map[string]IPlugin)
}

func (p *Plugins) Register(plugin IPlugin) {
	if inplugin, ok := p.Pmap[plugin.GetID()]; !ok {
		// 如果还没有存在这个插件，那么久直接添加
		p.Pmap[plugin.GetID()] = plugin
	} else {
		// 如果已经存在了，那么就比较版本号, 将新插件放入，无论新旧重新放入
		newPlugin := whoNewPlugin(inplugin, plugin)
		p.Pmap[newPlugin.GetID()] = newPlugin
	}
}

// 如果文件夹下没有有效插件，也会返回错误，该错误为PLUGIN_ZERO
func PluginsFromDir(pluginDir string) (ps *Plugins, e error) {
	ps = &Plugins{}
	plugindir, e := os.Open(pluginDir)

	if e != nil {
		Logff("Failed to open folder for plugin: %v", LOG_EXCEPTION, e)
		return
	}

	ps.Init()

	// names 是文件夹下面所有文件的名字，这时候还要判断是不是.so后缀
	names, e := plugindir.Readdirnames(0)

	for _, name := range names {
		if !isPluginFile(name) {
			// 不是插件文件就跳过
			continue
		}
		filep := filepath.Join(pluginDir, name)
		pfile, e := plugin.Open(filep)
		if e != nil {
			Logff("Filename: %s,Failed to open plugin: %v", LOG_WARNING, name, e)
			continue
		}
		symbol, e := pfile.Lookup(SYMBOL_NAME)
		if e != nil {
			Logff("Filename: %s, Failed to Lookup symbol: %v", LOG_WARNING, name, e)
			continue
		}
		// 这时拿到插件要告诉我们的信息了
		if IP, ok := symbol.(IPlugin); ok {
			ps.Register(IP)
			fmt.Printf("load plugin %s\n", name)
			continue
		} else {
			Logff("Filename: %s, Failed to register plugin", LOG_WARNING, name)
		}
	}

	if len(ps.Pmap) == 0 {
		e = PLUGIN_ZERO
	} else {
		e = nil
	}
	return
}

//  检查两个插件新旧，返回版本新的插件
func whoNewPlugin(iPlugin1 IPlugin, iPlugin2 IPlugin) IPlugin {
	ipv1 := ParseVersion(iPlugin1.Version())
	ipv2 := ParseVersion(iPlugin2.Version())
	if ipv1 != nil && ipv2 != nil {
		goto Normal
	}

	if ipv1 == nil && ipv2 == nil {
		return nil
	} else if ipv1 == nil {
		return iPlugin2
	} else if ipv2 == nil {
		return iPlugin1
	}

Normal:

	for i := 0; i < 3; i++ {
		// 大版本号开始比较
		if ipv1[i] > ipv2[i] {
			return iPlugin1
		}
		if ipv1[i] < ipv2[i] {
			return iPlugin2
		}
	}
	// 版本完全相同 return 随便一个就行了
	return iPlugin1
}

// 对外暴露的原因，适合三级分类的版本号
// @param version 三级版本号
// @return []byte 2->补丁版本号， 1->小版本号， 0->大版本号
// 若发生错误则返回nil
func ParseVersion(version string) (ver []byte) {
	split := strings.Split(version, ".")
	ver = make([]byte, 3)
	for i, v := range split {
		atoi, e := strconv.Atoi(v)
		if e != nil {
			return nil
		}
		ver[i] = uint8(atoi)
	}
	return
}

// 判定是否是plugin文件，简单的认为.so结尾的是插件文件
func isPluginFile(name string) bool {
	t := strings.Split(name, ".")
	if t[len(t)-1] == "so" {
		return true
	}
	return false
}
