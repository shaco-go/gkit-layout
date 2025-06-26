package configs

import "strings"

type Config struct {
	Env      string `mapstructure:"env"`       // 环境
	LogLevel string `mapstructure:"log_level"` // 日志等级
	Database Mysql  `mapstructure:"database"`  // 数据库
}

type Mysql struct {
	Host     string `mapstructure:"host"`     // 主机
	Port     int    `mapstructure:"port"`     // 端口
	Username string `mapstructure:"username"` // 用户名
	Password string `mapstructure:"password"` // 密码
	DBName   string `mapstructure:"db_name"`  // 数据库
}

func (c *Config) IsDev() bool {
	return strings.Index(strings.ToLower(c.Env), "dev") == 0
}
