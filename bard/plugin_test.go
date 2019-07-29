package bard

import (
	"fmt"
	"testing"
)

const (
	PLUGIN_DIR_S = "../server/debug/plugins/"
	PLUGIN_DIR_C = "../client/debug/plugins/"
)

func TestPluginsFromDir(t *testing.T) {
	ps, e := PluginsFromDir(PLUGIN_DIR_S)
	if e != nil || ps == nil {
		t.Fatal("error")
		return
	}
	if ps.Pmap == nil {
		fmt.Printf("ps.Pmap is nil\n")
		return
	}
	fmt.Printf("key\tvalue\tpriority\n")
	for k, v := range ps.Pmap {
		fmt.Printf("%s\t%v\t",k, v)
		fmt.Printf("%x\n",v.Priority())
	}
}

func TestPlugins_SortPriority(t *testing.T) {
	ps, _ := PluginsFromDir(PLUGIN_DIR_S)
	EC, Cs, As, Os := ps.SortPriority()
	fmt.Println(EC(), len(Cs), len(As), len(Os))
}

func TestPlugins_GetCAO(t *testing.T) {
	ps, _ := PluginsFromDir(PLUGIN_DIR_S)
	EC, C, A, O := ps.GetCAO()
	// 正确优先级数字大的先执行，也就是优先级低
	fmt.Println(EC())
	C([]byte{},true)
	A([]byte{}, true)
	O([]byte{}, true)
}

func TestPlugins_ToBigIPlugin(t *testing.T) {
	ps, _ := PluginsFromDir(PLUGIN_DIR_S)
	plugin := ps.ToBigIPlugin()
	fmt.Println(plugin.EndCam())
	plugin.Camouflage([]byte{}, true)
	plugin.AntiSniffing([]byte{}, true)
	plugin.Ornament([]byte{}, true)
}