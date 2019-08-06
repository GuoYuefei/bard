package plugin

import (
	"bard/bard"
	"bard/bard-plugin/base"
)

var plugins = []bard.IPlugin{
	base.V,

}

func WinPlugins() *bard.Plugins {
	ps := &bard.Plugins{}

	ps.Init()

	for _, v := range plugins {
		ps.Register(v)
	}

	return ps
}



