package plugin

import (
	"bard/bard"
	"bard/bard-plugin/base"
)

/**
	这里应该快速获取子包中定义好的客户端插件
 */
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



