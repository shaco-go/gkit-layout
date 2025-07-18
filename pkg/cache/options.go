package cache

import (
	"github.com/redis/go-redis/v9"
)

// CacheType 缓存类型
type CacheType int

const (
	// MemoryCache 内存缓存
	MemoryCache CacheType = iota
	// RedisCache Redis缓存
	RedisCache
)

// Options 缓存配置选项
type Options struct {
	// Type 缓存类型
	Type CacheType

	// Redis Redis客户端实例
	Redis redis.UniversalClient

	// KeyPrefix 键前缀
	KeyPrefix string

	// LockPrefix 锁前缀
	LockPrefix string

	// CacheSize 内存缓存大小(字节)
	CacheSize int

	// SetGCPercent 是否设置GC百分比
	SetGCPercent bool
}

// Option 配置函数类型
type Option func(*Options)

// WithRedis 使用Redis缓存
func WithRedis(client redis.UniversalClient) Option {
	return func(o *Options) {
		o.Type = RedisCache
		o.Redis = client
	}
}

// WithMemory 使用内存缓存
func WithMemory() Option {
	return func(o *Options) {
		o.Type = MemoryCache
	}
}

// WithKeyPrefix 设置键前缀
func WithKeyPrefix(prefix string) Option {
	return func(o *Options) {
		o.KeyPrefix = prefix
	}
}

// WithLockPrefix 设置锁前缀
func WithLockPrefix(prefix string) Option {
	return func(o *Options) {
		o.LockPrefix = prefix
	}
}

// WithCacheSize 设置内存缓存大小(字节)
func WithCacheSize(size int) Option {
	return func(o *Options) {
		o.CacheSize = size
	}
}

// WithSetGCPercent 设置是否调整GC百分比
func WithSetGCPercent(set bool) Option {
	return func(o *Options) {
		o.SetGCPercent = set
	}
}
