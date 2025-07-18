package cache

import (
	"context"
	"github.com/google/uuid"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/redis/go-redis/v9"
)

type redisCache struct {
	client    redis.UniversalClient
	prefix    string
	lockKey   string
	lockValue string
}

func newRedisCache(opts *Options) (Cache, error) {
	if opts.Redis == nil {
		return nil, errors.New("cache: redis client is required")
	}

	return &redisCache{
		client:  opts.Redis,
		prefix:  opts.KeyPrefix,
		lockKey: opts.LockPrefix,
	}, nil
}

func (c *redisCache) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	fullKey := c.prefix + key

	// 序列化值
	var data []byte
	var err error

	// 如果值已经是[]byte类型，直接使用
	if byteData, ok := value.([]byte); ok {
		data = byteData
	} else {
		// 否则序列化为JSON
		data, err = Marshal(value)
		if err != nil {
			return errors.Wrap(err, "cache: failed to marshal value")
		}
	}

	return c.client.Set(ctx, fullKey, data, expiration).Err()
}

func (c *redisCache) GetRaw(ctx context.Context, key string) ([]byte, error) {
	fullKey := c.prefix + key

	data, err := c.client.Get(ctx, fullKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrNotFound
		}
		return nil, errors.Wrap(err, "cache: failed to get value from redis")
	}

	return data, nil
}

func (c *redisCache) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := c.prefix + key

	count, err := c.client.Exists(ctx, fullKey).Result()
	if err != nil {
		return false, errors.Wrap(err, "cache: failed to check key existence")
	}

	return count > 0, nil
}

func (c *redisCache) SaveRaw(ctx context.Context, key string, fn func() ([]byte, error), expiration time.Duration, options ...SaveOption) ([]byte, error) {
	opts := &saveOptions{}
	for _, opt := range options {
		opt(opts)
	}

	// 如果不是强制刷新，先尝试从缓存获取
	if !opts.ForceRefresh {
		data, err := c.GetRaw(ctx, key)
		if err == nil {
			return data, nil
		}
		if err != ErrNotFound {
			return nil, err
		}
	}

	// 使用分布式锁防止缓存击穿（多个请求同时获取不存在的缓存）
	lockKey := "lock:" + key

	// 尝试获取锁，防止缓存击穿
	locked := false
	lockValue, err := c.Lock(ctx, lockKey, 5*time.Second)
	if err == nil {
		locked = true
		defer c.Unlock(ctx, lockKey, lockValue)
	} else if err != ErrLockAcquired {
		// 如果是其他错误，则直接返回
		return nil, err
	}

	// 如果获取到锁或者锁已被其他请求获取但尝试再次从缓存获取
	if locked || err == ErrLockAcquired {
		// 再次尝试从缓存获取，可能其他持有锁的请求已经设置了缓存
		data, err := c.GetRaw(ctx, key)
		if err == nil {
			return data, nil
		}
		if err != ErrNotFound {
			return nil, err
		}

		// 如果没有获取到锁，等待一段时间后再重试
		if !locked {
			time.Sleep(100 * time.Millisecond)
			return c.SaveRaw(ctx, key, fn, expiration, options...)
		}
	}

	// 缓存未命中或强制刷新，调用函数获取数据
	result, err := fn()
	if err != nil {
		return nil, err
	}

	// 处理缓存穿透 - 即使结果为空值，仍然缓存
	if (result == nil || len(result) == 0) && opts.PreventCacheMiss {
		exp := expiration
		if opts.NilExpiration > 0 {
			exp = opts.NilExpiration
		}
		err = c.Set(ctx, key, result, exp)
	} else {
		err = c.Set(ctx, key, result, expiration)
	}

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *redisCache) Lock(ctx context.Context, key string, expiration time.Duration) (string, error) {
	fullKey := c.lockKey + key

	// 生成唯一的锁标识符
	u, err := uuid.NewUUID()
	if err != nil {
		return "", errors.WithStack(err)
	}

	// 使用SET NX命令（只在键不存在时设置）来实现分布式锁
	// 相当于执行 SET key value NX PX expiration
	success, err := c.client.SetNX(ctx, fullKey, u.String(), expiration).Result()
	if err != nil {
		return "", errors.Wrap(err, "cache: failed to acquire lock")
	}

	if !success {
		return "", ErrLockAcquired
	}

	return u.String(), nil
}

func (c *redisCache) Unlock(ctx context.Context, key string, value string) error {
	fullKey := c.lockKey + key

	// 使用Lua脚本确保只删除由当前持有者设置的锁
	// 这防止了一个客户端意外删除另一个客户端的锁
	const luaScript = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
else
    return 0
end`

	result, err := c.client.Eval(ctx, luaScript, []string{fullKey}, value).Result()
	if err != nil {
		return errors.Wrap(err, "cache: failed to release lock")
	}

	if result.(int64) == 0 {
		return ErrLockNotOwned
	}

	return nil
}

func (c *redisCache) Close() error {
	return c.client.Close()
}
