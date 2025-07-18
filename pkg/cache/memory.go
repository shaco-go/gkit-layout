package cache

import (
	"context"
	"github.com/google/uuid"
	"runtime/debug"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/coocood/freecache"
)

type memoryCache struct {
	cache   *freecache.Cache
	mu      sync.RWMutex
	prefix  string
	lockKey string
	locks   map[string]string // key -> identifier
	lockMu  sync.Mutex
}

func newMemoryCache(opts *Options) (Cache, error) {
	// 默认缓存大小为100MB
	cacheSize := 100 * 1024 * 1024
	if opts.CacheSize > 0 {
		cacheSize = opts.CacheSize
	}

	// 创建freecache实例
	cache := freecache.NewCache(cacheSize)

	// 设置GC百分比为20%
	if opts.SetGCPercent {
		debug.SetGCPercent(20)
	}

	c := &memoryCache{
		cache:   cache,
		locks:   make(map[string]string),
		prefix:  opts.KeyPrefix,
		lockKey: opts.LockPrefix,
	}

	return c, nil
}

func (c *memoryCache) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	fullKey := c.prefix + key

	// 计算过期时间（秒）
	var expireSeconds int
	if expiration > 0 {
		expireSeconds = int(expiration.Seconds())
	}

	// 序列化值
	var data []byte
	var err error

	if value == nil {
		data = nil
	} else if rawData, ok := value.([]byte); ok {
		data = rawData
	} else {
		data, err = Marshal(value)
		if err != nil {
			return errors.Wrap(err, "cache: failed to marshal value")
		}
	}

	// 设置到freecache
	err = c.cache.Set([]byte(fullKey), data, expireSeconds)
	if err != nil {
		return errors.Wrap(err, "cache: failed to set value in freecache")
	}

	return nil
}

func (c *memoryCache) GetRaw(ctx context.Context, key string) ([]byte, error) {
	fullKey := c.prefix + key

	// 从freecache获取数据
	data, err := c.cache.Get([]byte(fullKey))
	if err == freecache.ErrNotFound {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, errors.Wrap(err, "cache: failed to get value from freecache")
	}

	return data, nil
}

func (c *memoryCache) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := c.prefix + key

	// 检查键是否存在
	_, err := c.cache.Get([]byte(fullKey))
	if errors.Is(err, freecache.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, errors.Wrap(err, "cache: failed to check key existence")
	}

	return true, nil
}

func (c *memoryCache) SaveRaw(ctx context.Context, key string, fn func() ([]byte, error), expiration time.Duration, options ...SaveOption) ([]byte, error) {
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
		if !errors.Is(err, ErrNotFound) {
			return nil, err
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

func (c *memoryCache) Lock(ctx context.Context, key string, expiration time.Duration) (string, error) {
	c.lockMu.Lock()
	defer c.lockMu.Unlock()

	lockKey := c.lockKey + key
	if _, exists := c.locks[lockKey]; exists {
		return "", ErrLockAcquired
	}

	// 生成唯一标识符作为锁的值
	u, err := uuid.NewUUID()
	if err != nil {
		return "", errors.WithStack(err)
	}

	// 设置锁
	c.locks[lockKey] = u.String()

	// 设置自动过期
	if expiration > 0 {
		go func(key string, value string, d time.Duration) {
			select {
			case <-time.After(d):
				c.lockMu.Lock()
				defer c.lockMu.Unlock()
				// 确保锁还是被同一个值持有
				if v, exists := c.locks[key]; exists && v == value {
					delete(c.locks, key)
				}
			case <-ctx.Done():
				return
			}
		}(lockKey, u.String(), expiration)
	}

	return u.String(), nil
}

func (c *memoryCache) Unlock(ctx context.Context, key string, value string) error {
	c.lockMu.Lock()
	defer c.lockMu.Unlock()

	lockKey := c.lockKey + key
	if val, exists := c.locks[lockKey]; !exists || val != value {
		return ErrLockNotOwned
	}

	delete(c.locks, lockKey)
	return nil
}

func (c *memoryCache) Close() error {
	// freecache没有显式的Close方法
	return nil
}
