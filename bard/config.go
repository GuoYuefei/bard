package bard

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
)

// 配置内容 配置文件使用json or yml， 简单嘛，易用嘛
// 使用yml吧，哈哈哈， 就是为了学一种新的配置文件。 个人感觉json现在大多用于传输， yml适合做配置 因为还是需要有注释的
// 其实json注释也能行，自己先做预处理

type Config struct {
	Server interface{} 		`yaml:"server"`
	ServerPort int			`yaml:"server_port"`
	LocalPort int 			`yaml:"local_port"`
	LocalAddress string 	`yaml:"local_address"`

	// 以上是基础信息
	// server 是server配置项     客户端四者都需要

}

func (config *Config) String() string {
	return fmt.Sprintf("Server=%v\nServerPort=%v\nLocalPort=%v\nLocalAddress=%v\n",
		config.GetServers(), config.ServerPort, config.LocalPort, config.LocalAddress)
}

func ParseConfig(path string) (config *Config, err error) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return
	}
	config = &Config{}
	if err = yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}

	return
}

func (c *Config) ServerPortString() string {
	return strconv.Itoa(c.ServerPort)
}

func (c *Config) LocalPortString() string {
	return strconv.Itoa(c.LocalPort)
}

func (config *Config) GetServers() []string {
	if config.Server == nil {
		return nil
	}

	if s, ok := config.Server.(string); ok {
		return []string{s}
	}

	if arr, ok := config.Server.([]interface{}); ok {
		serverArr := make([]string, len(arr))
		for i, s := range arr {
			if serverArr[i], ok = s.(string); !ok {
				goto typeError
			}
		}
		return serverArr
	}
typeError:
	panic(fmt.Sprintf("Config.Server type error %v", reflect.TypeOf(config.Server)))
}



