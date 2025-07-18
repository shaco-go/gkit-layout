package bootstrap

import (
	"fmt"
	"github.com/shaco-go/gkit-layout/global"
	"github.com/shaco-go/gkit-layout/pkg/cache"
)

// InitCache 初始化缓存
func InitCache() *cache.Cache {
	var op = []cache.Option{
		cache.WithKeyPrefix(global.Conf.AppName + ":"),
		cache.WithLockPrefix(global.Conf.AppName + ":lock:"),
	}
	if global.Conf.Cache == "redis" {
		op = append(op, cache.WithRedis(global.Redis))
	} else {
		op = append(op, cache.WithMemory())
	}
	c, err := cache.New(op...)
	if err != nil {
		panic(fmt.Errorf("cache init fail :%w", err))
	}
	return &c
}
