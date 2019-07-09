package bard

import (
	"fmt"
	"testing"
)

const (
	PLUGIN_DIR = "../debug/plugins/"
)

func TestPluginsFromDir(t *testing.T) {
	ps, e := PluginsFromDir(PLUGIN_DIR)
	if e != nil || ps == nil {
		t.Fatal("error")
		return
	}
	if ps.Pmap == nil {
		fmt.Printf("ps.Pmap is nil\n")
		return
	}
	fmt.Printf("key\tvalue\n")
	for k, v := range ps.Pmap {
		fmt.Printf("%s\t%v\n",k, v)
		fmt.Printf("%x\n",v.Priority())
	}
}

func TestPlugins_SortPriority(t *testing.T) {
	ps, _ := PluginsFromDir(PLUGIN_DIR)
	Cs, As, Os := ps.SortPriority()
	fmt.Println(len(Cs), len(As), len(Os))
}

func TestPlugins_GetCAO(t *testing.T) {
	ps, _ := PluginsFromDir(PLUGIN_DIR)
	C, A, O := ps.GetCAO()
	// 正确优先级数字大的先执行，也就是优先级低
	C([]byte{})
	A([]byte{})
	O([]byte{})
}
