package bootstrap

import (
	"github.com/rs/zerolog"
	"github.com/shaco-go/gkit-layout/global"
	gkit_zerolog "github.com/shaco-go/gkit-layout/pkg/zerolog"
)

func InitLog() zerolog.Logger {
	logConf := gkit_zerolog.NewDevLogConfig()
	logConf.Channel = parseLogChannel(global.Conf.Log.Channel)
	logConf.Level, _ = zerolog.ParseLevel(global.Conf.Log.LogLevel)
	logConf.HumanReadable = global.Conf.Log.HumanReadable
	logConf.LogFileName = global.Conf.AppName + ".log"

	// 初始化日志,分开因为gorm会自动生成错误行
	return gkit_zerolog.New(logConf)
}

func parseLogChannel(channel []string) []gkit_zerolog.ChannelType {
	var res []gkit_zerolog.ChannelType
	for _, item := range channel {
		val, err := gkit_zerolog.ParseChannelType(item)
		if err != nil {
			panic(err)
		}
		res = append(res, val)
	}
	return res
}
