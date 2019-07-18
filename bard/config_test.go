package bard

import (
	"fmt"
	"testing"
)

func TestParseConfig(t *testing.T) {
	config, err := ParseConfig("../server/debug/config/config.yml")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(config)
}

func TestConfig_Users(t *testing.T) {
	config, err := ParseConfig("../server/debug/config/config.yml")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(config)
	for i, v := range config.Users {
		fmt.Printf("%d\t\t username: %s \t\t passwd: %s\n", i, v.Username, v.Password)
	}
}