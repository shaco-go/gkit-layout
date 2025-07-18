package global

import (
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/shaco-go/gkit-layout/configs"
	"github.com/shaco-go/gkit-layout/pkg/cache"
	"gorm.io/gorm"
)

var (
	Conf  *configs.Config
	DB    *gorm.DB
	Log   zerolog.Logger
	Cache *cache.Cache
	Redis *redis.Client
)
