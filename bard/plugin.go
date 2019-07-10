package bard

import (
	"errors"
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

var PLUGIN_ZERO = errors.New("No valid plugins under the folder")

type IPluFun func([]byte) ([]byte, int)

// 应该要设置热插拔
type IPlugin interface {

	// 伪装， 在socks5协议之前伪装协议头
	Camouflage([]byte) ([]byte, int)

	// 防嗅探，连接建立过程或udp传输时使用 这里内容比较少可能使用非对称加密
	AntiSniffing([]byte) ([]byte, int)

	// 操作传输内容
	Ornament([]byte) ([]byte, int)

	// 优先级，越是优先越后运行	0是最高优先级
	// !!! 一个重要的解释：前面三位是保留位
	// 当八位0001,xxxx,xxxx,xxxx这样格式的，为在socks5协议之前的混淆协议，这时启用Camouflage函数，
	// 当八位0010,xxxx,xxxx,xxxx格式的，为加密或操作socks协议本身，这个主要防止在建立socks5连接阶段被嗅探,启用AntiSniffing
	// 当八位0100,xxxx,xxxx,xxxx格式的，为加密或操作传输内容的 启用Ornament函数
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
	var prs uint8 = uint8(iPlugin.Priority()>>12)

	// 下面的元素放入切片是没有做check， 所以需要外层函数对返回的prs做check
	// 插件中的Camouflage函数可用
	pris[0] = &sortPriFun{uint8(0x000f & iPlugin.Priority()), iPlugin.Camouflage}
	// A
	pris[1] = &sortPriFun{uint8((0x00f0 & iPlugin.Priority()) >> 4), iPlugin.AntiSniffing}
	// O
	pris[2] = &sortPriFun{uint8((0x0f00 & iPlugin.Priority()) >> 8), iPlugin.Ornament}

	return prs, pris
}

// 排序时的中间类型不暴露
type sortPriFun struct {
	pri uint8			// 表示下面函数的优先级
	fun IPluFun			// 一个插件中的优先函数
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
func (p *Plugins)SortPriority() (Cs []IPluFun, As []IPluFun, Os []IPluFun){

	var Cfuns sortPriFuns = make([]*sortPriFun, 0, len(p.Pmap))
	var Afuns sortPriFuns = make([]*sortPriFun, 0, len(p.Pmap))
	var Ofuns sortPriFuns = make([]*sortPriFun, 0, len(p.Pmap))
	for _, v := range p.Pmap {
		u, fun := pluginPriority(v)
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

	return Cs, As, Os
}

// 最后返回根据是否生效，以及各插件指定函数的优先级分别返回三个总函数
func (p *Plugins)GetCAO() (C IPluFun, A IPluFun, O IPluFun) {
	Cs, As, Os := p.SortPriority()

	var genCAO = func(ss []IPluFun) (s IPluFun) {
		s = func(in []byte) (out []byte, l int) {
			out = in
			for _, v := range Cs {
				out, l = v(out)
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
	C IPluFun
	A IPluFun
	O IPluFun
}

func (b *bigIPlugin)Camouflage(bs []byte) ([]byte, int) {
	return b.C(bs)
}
func (b *bigIPlugin)AntiSniffing(bs []byte) ([]byte, int) {
	return b.A(bs)
}
func (b *bigIPlugin)Ornament(bs []byte) ([]byte, int) {
	return b.O(bs)
}
// 以下三函数只为实现接口
func (b *bigIPlugin)Priority() uint16 {
	return 0x0000
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
	C, A, O := p.GetCAO()
	BP := &bigIPlugin{C, A, O}
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
func PluginsFromDir(plugin_dir string) (ps *Plugins, e error) {
	ps = &Plugins{}
	plugindir, e := os.Open(plugin_dir)

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
		filep := filepath.Join(plugin_dir, name)
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
	} else if ipv1 == nil {
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
