package bard

import (
	"fmt"
	"testing"
)

func TestParseConfig(t *testing.T) {
	config, err := ParseConfig("../debug/config/config.yml")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(config)
}
