package bootstrap

import (
	"github.com/shaco-go/gkit-layout/global"
	gkit_zerolog "github.com/shaco-go/gkit-layout/pkg/zerolog"
)

// Init 初始化顺序很重要,有的模块依赖其他模块
func Init(path string) {
	// 初始数据库
	global.Conf = InitConfig(path)
	// 初始化日志,分开因为gorm会自动生成错误行
	zl := gkit_zerolog.New(global.Conf.IsDev(), global.Conf.LogLevel)
	global.Log = zl.With().Stack().Caller().Timestamp().Logger()
	global.DB = InitMysql(global.Conf.IsDev(), global.Conf.Database, zl.With().Stack().Timestamp().Logger())
	global.Redis = InitRedis()
	global.Cache = InitCache()
}
