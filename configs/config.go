package configs

import "strings"

type Config struct {
	Env      string `mapstructure:"env"`      // 环境
	AppName  string `mapstructure:"app_name"` // 应用名称
	Database Mysql  `mapstructure:"database"` // 数据库
	Cache    string `mapstructure:"cache"`    // 缓存类型
	Redis    Redis  `mapstructure:"redis"`    // redis
	Log      Log    `mapstructure:"log"`      // 日志配置
}

type Log struct {
	Channel       []string `mapstructure:"channel"`
	LogLevel      string   `mapstructure:"log_level"`      // 日志等级 info一下模式可打印sql
	HumanReadable bool     `mapstructure:"human_readable"` // 是否使用可读格式
}

type Mysql struct {
	Host     string `mapstructure:"host"`     // 主机
	Port     int    `mapstructure:"port"`     // 端口
	Username string `mapstructure:"username"` // 用户名
	Password string `mapstructure:"password"` // 密码
	DBName   string `mapstructure:"db_name"`  // 数据库
}

type Redis struct {
	Host     string `mapstructure:"host"`     // 主机
	Port     int    `mapstructure:"port"`     // 端口
	Password string `mapstructure:"password"` // 密码
	DB       int    `mapstructure:"db"`       // 数据库
}

func (c *Config) IsDev() bool {
	return strings.Index(strings.ToLower(c.Env), "dev") == 0
}
