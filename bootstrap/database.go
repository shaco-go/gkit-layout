package bootstrap

import (
	"fmt"
	"github.com/rs/zerolog"
	"github.com/shaco-go/gkit-layout/configs"
	"github.com/shaco-go/gkit-layout/global"
	gkit_gorm "github.com/shaco-go/gkit-layout/pkg/gorm"
	gkit_zerolog "github.com/shaco-go/gkit-layout/pkg/zerolog"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"time"
)

func InitMysql(conf configs.Mysql, z zerolog.Logger) *gorm.DB {
	dsn := gkit_gorm.DefaultDSN()
	dsn.Host = conf.Host
	dsn.Port = conf.Port
	dsn.Username = conf.Username
	dsn.Password = conf.Password
	dsn.DBName = conf.DBName

	level, err := zerolog.ParseLevel(global.Conf.Log.LogLevel)
	if err != nil {
		global.Log.Warn().Err(err).Send()
	}
	db, err := gorm.Open(mysql.Open(dsn.String()), &gorm.Config{
		Logger: gkit_zerolog.NewGormLogger(z.With().Timestamp().Logger(), logger.Config{
			SlowThreshold:             3 * time.Second,
			Colorful:                  global.Conf.Log.HumanReadable,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      false,
			LogLevel:                  gkit_zerolog.ZeroToGormLevel(level),
		}),
	})
	if err != nil && !global.Conf.IsDev() {
		panic(fmt.Errorf("初始化数据库失败:%w", err))
	}
	return db
}
