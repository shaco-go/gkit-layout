package bootstrap

import (
	"flag"
	"github.com/shaco-go/gkit-layout/global"
	gkit_zerolog "github.com/shaco-go/gkit-layout/pkg/zerolog"
)

func Init() {
	path := flag.String("c", "configs/development.yaml", "config file path")
	flag.Parse()
	// 初始数据库
	global.Conf = InitConfig(*path)
	// 初始化日志,分开因为gorm会自动生成错误行
	zl := gkit_zerolog.New(global.Conf.IsDev(), global.Conf.LogLevel)
	global.Log = zl.With().Stack().Caller().Timestamp().Logger()
	global.DB = InitMysql(global.Conf.IsDev(), global.Conf.Database, zl.With().Stack().Timestamp().Logger())
}
