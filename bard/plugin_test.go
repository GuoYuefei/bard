package bard

import "testing"

const (
	PLUGIN_DIR = "../debug/plugins"
)

func TestPluginsFromDir(t *testing.T) {
	PluginsFromDir(PLUGIN_DIR)

}
