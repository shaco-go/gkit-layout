package bootstrap

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/shaco-go/gkit-layout/global"
)

func InitRedis() *redis.Client {
	// 创建缓存实例
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", global.Conf.Redis.Host, global.Conf.Redis.Port),
		Password: global.Conf.Redis.Password,
		DB:       global.Conf.Redis.DB,
	})

	_, err := client.Ping(context.Background()).Result()
	if err != nil && !global.Conf.IsDev() {
		panic(fmt.Errorf("redis conn fail :%w", err))
	}
	return nil
}
