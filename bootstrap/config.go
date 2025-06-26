package bootstrap

import (
	"fmt"
	"github.com/shaco-go/gkit-layout/configs"
	"github.com/spf13/viper"
)

// InitConfig 初始化viper并返回config
func InitConfig(path string) *configs.Config {
	v := viper.New()
	v.SetConfigFile(path)
	err := v.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("初始化viper失败: %w", err))
	}
	var conf configs.Config
	err = v.Unmarshal(&conf)
	if err != nil {
		panic(fmt.Errorf("解析配置文件失败: %w", err))
	}
	return &conf
}
