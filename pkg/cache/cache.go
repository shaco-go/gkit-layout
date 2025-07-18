package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/cockroachdb/errors"
)

var (
	// 定义错误类型
	ErrNotFound      = errors.New("cache: key not found")
	ErrLockAcquired  = errors.New("cache: lock already acquired")
	ErrLockNotOwned  = errors.New("cache: lock not owned by caller")
	ErrInvalidParams = errors.New("cache: invalid parameters")
)

// Cache 定义缓存接口
type Cache interface {
	// Set 设置缓存，带过期时间
	Set(ctx context.Context, key string, value any, expiration time.Duration) error

	// GetRaw 获取原始缓存数据
	GetRaw(ctx context.Context, key string) ([]byte, error)

	// Exists 检查键是否存在
	Exists(ctx context.Context, key string) (bool, error)

	// SaveRaw 获取或设置原始缓存数据
	SaveRaw(ctx context.Context, key string, fn func() ([]byte, error), expiration time.Duration, options ...SaveOption) ([]byte, error)

	// Lock 获取分布式锁，返回锁的唯一标识符
	Lock(ctx context.Context, key string, expiration time.Duration) (string, error)

	// Unlock 释放分布式锁
	Unlock(ctx context.Context, key string, value string) error

	// Close 关闭缓存
	Close() error
}

// SaveOption 定义Save方法的可选参数
type SaveOption func(*saveOptions)

type saveOptions struct {
	// ForceRefresh 是否强制刷新缓存
	ForceRefresh bool

	// PreventCacheMiss 防止缓存穿透，当fn返回nil时仍缓存一个空值
	PreventCacheMiss bool

	// NilExpiration 空值的过期时间(防止缓存穿透时使用)
	NilExpiration time.Duration
}

// WithForceRefresh 强制刷新缓存，不管是否存在都会调用fn
func WithForceRefresh() SaveOption {
	return func(o *saveOptions) {
		o.ForceRefresh = true
	}
}

// WithPreventCacheMiss 防止缓存穿透，当数据不存在时也缓存一个空占位符
func WithPreventCacheMiss(expiration time.Duration) SaveOption {
	return func(o *saveOptions) {
		o.PreventCacheMiss = true
		o.NilExpiration = expiration
	}
}

// New 创建一个新的缓存实例
func New(opts ...Option) (Cache, error) {
	options := &Options{
		Type: MemoryCache,
	}

	for _, opt := range opts {
		opt(options)
	}

	switch options.Type {
	case MemoryCache:
		return newMemoryCache(options)
	case RedisCache:
		return newRedisCache(options)
	default:
		return nil, errors.New("cache: unsupported cache type")
	}
}

// 泛型辅助函数

// Get 获取并反序列化缓存数据
func Get[T any](ctx context.Context, cache Cache, key string) (T, error) {
	var value T

	data, err := cache.GetRaw(ctx, key)
	if err != nil {
		return value, err
	}

	// 如果数据为空，直接返回零值
	if len(data) == 0 {
		return value, nil
	}

	// 反序列化数据
	err = Unmarshal(data, &value)
	if err != nil {
		return value, errors.Wrap(err, "cache: failed to unmarshal value")
	}

	return value, nil
}

// Save 获取或设置缓存数据
func Save[T any](ctx context.Context, cache Cache, key string, fn func() (T, error), expiration time.Duration, options ...SaveOption) (T, error) {
	var value T

	// 使用一个适配器函数，将泛型函数转换为返回[]byte的函数
	rawFn := func() ([]byte, error) {
		result, err := fn()
		if err != nil {
			return nil, err
		}

		// 序列化结果
		data, err := Marshal(result)
		if err != nil {
			return nil, errors.Wrap(err, "cache: failed to marshal value")
		}

		return data, nil
	}

	// 调用原始的SaveRaw方法
	rawData, err := cache.SaveRaw(ctx, key, rawFn, expiration, options...)
	if err != nil {
		return value, err
	}

	// 如果数据为空，直接返回零值
	if len(rawData) == 0 {
		return value, nil
	}

	// 反序列化数据
	err = Unmarshal(rawData, &value)
	if err != nil {
		return value, errors.Wrap(err, "cache: failed to unmarshal value")
	}

	return value, nil
}

// Marshal 序列化数据
func Marshal(v interface{}) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}

// Unmarshal 反序列化数据
func Unmarshal(data []byte, v interface{}) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, v)
}
