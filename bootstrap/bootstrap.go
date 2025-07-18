package bootstrap

import (
	"github.com/shaco-go/gkit-layout/global"
)

// Init 初始化顺序很重要,有的模块依赖其他模块
func Init(path string) {
	// 初始数据库
	global.Conf = InitConfig(path)

	// 初始化日志配置
	log := InitLog()
	global.Log = log.With().Stack().Caller().Timestamp().Logger()
	global.DB = InitMysql(global.Conf.Database, log)
	global.Redis = InitRedis()
	global.Cache = InitCache()
}
