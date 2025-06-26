package bootstrap

import (
	"fmt"
	"github.com/rs/zerolog"
	"github.com/shaco-go/gkit-layout/configs"
	gkit_gorm "github.com/shaco-go/gkit-layout/pkg/gorm"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitMysql(dev bool, conf configs.Mysql, zl zerolog.Logger) *gorm.DB {
	dsn := gkit_gorm.DefaultDSN()
	dsn.Host = conf.Host
	dsn.Port = conf.Port
	dsn.Username = conf.Username
	dsn.Password = conf.Password
	dsn.DBName = conf.DBName

	var logConf logger.Config
	if dev {
		logConf = gkit_gorm.DevConfigLog
	} else {
		logConf = gkit_gorm.ProConfigLog
	}

	db, err := gorm.Open(mysql.Open(dsn.String()), &gorm.Config{
		Logger: gkit_gorm.NewLog(logConf, zl),
	})
	if err != nil {
		panic(fmt.Errorf("初始化数据库失败:%w", err))
	}
	return db
}
