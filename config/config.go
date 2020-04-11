package config

import (
	"fmt"
	"github.com/Unknwon/goconfig"
)

func LoadConfigFile() *goconfig.ConfigFile {
	configFile, err := goconfig.LoadConfigFile("config.ini")
	if err != nil {
		fmt.Println("文件加载错误..." + err.Error())
		return nil
	}
	return configFile
}
