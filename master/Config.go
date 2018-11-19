package master

import (
	"encoding/json"
	"io/ioutil"
)

//程序配置
type Config struct {
	ApiPort         int `json:"airPort"`
	ApiReadTimeout  int `json:"apiReadTimeout"`
	ApiWriteTimeout int `json:"apiWriteTimeout"`
}

var (
	G_config *Config
)

func InitConfig(filename string) (err error) {

	var (
		content []byte
		conf    Config
	)

	//把配置文件读进来
	if content, err = ioutil.ReadFile(filename); err != nil {
		return
	}

	json.Unmarshal(content, &conf)

	G_config = &conf
	return
}
