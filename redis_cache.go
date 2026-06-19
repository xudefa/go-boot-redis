// Package redis 基于 Redis 提供缓存实现。
//
// 该包将 Redis 与 go-boot 缓存接口集成，
// 支持内存缓存和 Redis 分布式缓存。
//
// 定义：
//
//   - RedisCache: Redis 缓存实现
//   - ClientOption: Redis 客户端配置选项函数
//   - CacheOption: 缓存配置选项函数
//
// 快速开始:
//
//	// 使用函数式选项创建客户端
//	client := redis.NewClient(
//	    redis.WithAddress("localhost:6379"),
//	    redis.WithPassword(""),
//	    redis.WithDB(0),
//	)
//
//	// 创建缓存实例
//	redisCache, _ := redis.NewCache(client,
//	    redis.WithPrefix("prefix:"),
//	)
//
//	// 或一次性创建带配置的缓存
//	redisCache, _ := redis.NewCacheWithConfig(
//	    []redis.ClientOption{
//	        redis.WithAddress("localhost:6379"),
//	    },
//	    redis.WithPrefix("cache:"),
//	)
package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/xudefa/go-boot/cache"
)

// Logger 定义日志接口
type Logger interface {
	Info(msg string, keysAndValues ...interface{})
	Error(err error, msg string, keysAndValues ...interface{})
}

// RedisCache 是基于 Redis 的缓存实现。
//
// 字段说明:
//   - client: Redis 客户端（支持普通客户端和集群客户端）
//   - prefix: 键前缀
//   - defaultTTL: 默认过期时间
//   - logger: 日志记录器
type RedisCache struct {
	client     redis.UniversalClient
	prefix     string
	defaultTTL time.Duration
	logger     Logger
}

// NewRedisCache 创建新的 Redis 缓存实例。
//
// 参数:
//   - prefix: 键前缀
//   - defaultTTL: 默认过期时间
//   - client: Redis 客户端（支持普通客户端和集群客户端）
//
// 返回值:
//   - *RedisCache: 缓存实例
//   - error: 创建错误
func NewRedisCache(prefix string, defaultTTL time.Duration, client redis.UniversalClient) (*RedisCache, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTTL) // 使用defaultTTL作为连接测试超时
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}
	return &RedisCache{
		client:     client,
		prefix:     prefix,
		defaultTTL: defaultTTL,
		logger:     nil, // 默认无日志
	}, nil
}

// fullKey 返回带前缀的完整键名。
//
// 参数:
//   - key: 原始键名
//
// 返回值:
//   - string: 带前缀的键名
func (c *RedisCache) fullKey(key string) string {
	return c.prefix + key
}

// Get 获取缓存值。
//
// 参数:
//   - ctx: 上下文
//   - key: 缓存键
//
// 返回值:
//   - any: 缓存值
//   - error: 错误
func (c *RedisCache) Get(ctx context.Context, key string) (any, error) {
	start := time.Now()
	val, err := c.client.Get(ctx, c.fullKey(key)).Result()
	duration := time.Since(start)

	if err != nil {
		if errors.Is(err, redis.Nil) {
			if c.logger != nil {
				c.logger.Info("cache miss", "key", key, "duration", duration)
			}
			return nil, cache.ErrNotFound
		}
		if c.logger != nil {
			c.logger.Error(err, "get cache failed", "key", key, "duration", duration)
		}
		return nil, err
	}

	if c.logger != nil {
		c.logger.Info("cache hit", "key", key, "duration", duration)
	}

	var result any
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		if c.logger != nil {
			c.logger.Error(err, "unmarshal cached value failed", "key", key)
		}
		return nil, err
	}
	return result, nil
}

// Set 设置缓存值。
//
// 参数:
//   - ctx: 上下文
//   - key: 缓存键
//   - value: 缓存值
//   - ttl: 过期时间
//
// 返回值:
//   - error: 错误
func (c *RedisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	start := time.Now()
	data, err := json.Marshal(value)
	if err != nil {
		if c.logger != nil {
			c.logger.Error(err, "marshal value failed", "key", key)
		}
		return err
	}

	var setResult error
	if ttl > 0 {
		setResult = c.client.Set(ctx, c.fullKey(key), data, ttl).Err()
	} else {
		setResult = c.client.Set(ctx, c.fullKey(key), data, 0).Err()
	}

	duration := time.Since(start)

	if setResult != nil {
		if c.logger != nil {
			c.logger.Error(setResult, "set cache failed", "key", key, "duration", duration)
		}
		return setResult
	}

	if c.logger != nil {
		c.logger.Info("set cache success", "key", key, "duration", duration, "ttl", ttl)
	}

	return nil
}

// Del 删除指定的缓存键。
//
// 参数:
//   - ctx: 上下文
//   - keys: 缓存键列表
//
// 返回值:
//   - error: 错误
func (c *RedisCache) Del(ctx context.Context, keys ...string) error {
	start := time.Now()
	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = c.fullKey(key)
	}
	err := c.client.Del(ctx, fullKeys...).Err()
	duration := time.Since(start)

	if err != nil {
		if c.logger != nil {
			c.logger.Error(err, "delete cache failed", "keys", keys, "duration", duration)
		}
		return err
	}

	if c.logger != nil {
		c.logger.Info("delete cache success", "keys", keys, "duration", duration)
	}

	return nil
}

// HGet 获取哈希字段的值
func (c *RedisCache) HGet(ctx context.Context, key, field string) (any, error) {
	start := time.Now()
	val, err := c.client.HGet(ctx, c.fullKey(key), field).Result()
	duration := time.Since(start)

	if err != nil {
		if errors.Is(err, redis.Nil) {
			if c.logger != nil {
				c.logger.Info("hash field not found", "key", key, "field", field, "duration", duration)
			}
			return nil, cache.ErrNotFound
		}
		if c.logger != nil {
			c.logger.Error(err, "get hash field failed", "key", key, "field", field, "duration", duration)
		}
		return nil, err
	}

	if c.logger != nil {
		c.logger.Info("hash field found", "key", key, "field", field, "duration", duration)
	}

	var result any
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		if c.logger != nil {
			c.logger.Error(err, "unmarshal hash field value failed", "key", key, "field", field)
		}
		return nil, err
	}
	return result, nil
}

// HSet 设置哈希字段的值
func (c *RedisCache) HSet(ctx context.Context, key, field string, value any, ttl time.Duration) error {
	start := time.Now()
	data, err := json.Marshal(value)
	if err != nil {
		if c.logger != nil {
			c.logger.Error(err, "marshal hash field value failed", "key", key, "field", field)
		}
		return err
	}

	pipe := c.client.Pipeline()
	pipe.HSet(ctx, c.fullKey(key), field, data)

	if ttl > 0 {
		pipe.Expire(ctx, c.fullKey(key), ttl)
	}

	_, err = pipe.Exec(ctx)
	duration := time.Since(start)

	if err != nil {
		if c.logger != nil {
			c.logger.Error(err, "set hash field failed", "key", key, "field", field, "duration", duration)
		}
		return err
	}

	if c.logger != nil {
		c.logger.Info("set hash field success", "key", key, "field", field, "duration", duration, "ttl", ttl)
	}

	return nil
}

// HDel 删除哈希字段
func (c *RedisCache) HDel(ctx context.Context, key string, fields ...string) error {
	start := time.Now()
	fullKey := c.fullKey(key)
	err := c.client.HDel(ctx, fullKey, fields...).Err()
	duration := time.Since(start)

	if err != nil {
		if c.logger != nil {
			c.logger.Error(err, "delete hash fields failed", "key", key, "fields", fields, "duration", duration)
		}
		return err
	}

	if c.logger != nil {
		c.logger.Info("delete hash fields success", "key", key, "fields", fields, "duration", duration)
	}

	return nil
}

// HMGet 批量获取哈希字段的值
func (c *RedisCache) HMGet(ctx context.Context, key string, fields ...string) ([]any, error) {
	start := time.Now()
	values, err := c.client.HMGet(ctx, c.fullKey(key), fields...).Result()
	duration := time.Since(start)

	if err != nil {
		if c.logger != nil {
			c.logger.Error(err, "batch get hash fields failed", "key", key, "fields", fields, "duration", duration)
		}
		return nil, err
	}

	result := make([]any, len(values))
	for i, val := range values {
		if val == nil {
			continue
		}
		str, ok := val.(string)
		if !ok {
			continue
		}
		var item any
		if err := json.Unmarshal([]byte(str), &item); err != nil {
			continue
		}
		result[i] = item
	}

	if c.logger != nil {
		c.logger.Info("batch get hash fields success", "key", key, "fields", fields, "duration", duration)
	}

	return result, nil
}

// SAdd 向集合添加元素
func (c *RedisCache) SAdd(ctx context.Context, key string, members ...any) error {
	start := time.Now()
	args := make([]interface{}, len(members))
	for i, member := range members {
		data, err := json.Marshal(member)
		if err != nil {
			if c.logger != nil {
				c.logger.Error(err, "marshal set member failed", "key", key, "index", i)
			}
			return err
		}
		args[i] = data
	}

	err := c.client.SAdd(ctx, c.fullKey(key), args...).Err()
	duration := time.Since(start)

	if err != nil {
		if c.logger != nil {
			c.logger.Error(err, "add to set failed", "key", key, "members", len(members), "duration", duration)
		}
		return err
	}

	if c.logger != nil {
		c.logger.Info("add to set success", "key", key, "members", len(members), "duration", duration)
	}

	return nil
}

// SMembers 获取集合的所有成员
func (c *RedisCache) SMembers(ctx context.Context, key string) ([]any, error) {
	start := time.Now()
	values, err := c.client.SMembers(ctx, c.fullKey(key)).Result()
	duration := time.Since(start)

	if err != nil {
		if errors.Is(err, redis.Nil) {
			if c.logger != nil {
				c.logger.Info("set is empty", "key", key, "duration", duration)
			}
			return []any{}, nil
		}
		if c.logger != nil {
			c.logger.Error(err, "get set members failed", "key", key, "duration", duration)
		}
		return nil, err
	}

	result := make([]any, len(values))
	for i, val := range values {
		var item any
		if err := json.Unmarshal([]byte(val), &item); err != nil {
			continue
		}
		result[i] = item
	}

	if c.logger != nil {
		c.logger.Info("get set members success", "key", key, "members", len(result), "duration", duration)
	}

	return result, nil
}

// ZAdd 向有序集合添加成员
func (c *RedisCache) ZAdd(ctx context.Context, key string, members ...redis.Z) error {
	start := time.Now()
	for i := range members {
		data, err := json.Marshal(members[i].Member)
		if err != nil {
			if c.logger != nil {
				c.logger.Error(err, "marshal zset member failed", "key", key, "index", i)
			}
			return err
		}
		members[i].Member = data
	}

	err := c.client.ZAdd(ctx, c.fullKey(key), members...).Err()
	duration := time.Since(start)

	if err != nil {
		if c.logger != nil {
			c.logger.Error(err, "add to zset failed", "key", key, "members", len(members), "duration", duration)
		}
		return err
	}

	if c.logger != nil {
		c.logger.Info("add to zset success", "key", key, "members", len(members), "duration", duration)
	}

	return nil
}

// ZRange 获取有序集合指定范围的成员
func (c *RedisCache) ZRange(ctx context.Context, key string, start, stop int64) ([]any, error) {
	startTime := time.Now()
	values, err := c.client.ZRange(ctx, c.fullKey(key), start, stop).Result()
	duration := time.Since(startTime)

	if err != nil {
		if errors.Is(err, redis.Nil) {
			if c.logger != nil {
				c.logger.Info("zset is empty", "key", key, "start", start, "stop", stop, "duration", duration)
			}
			return []any{}, nil
		}
		if c.logger != nil {
			c.logger.Error(err, "get zset range failed", "key", key, "start", start, "stop", stop, "duration", duration)
		}
		return nil, err
	}

	result := make([]any, len(values))
	for i, val := range values {
		var item any
		if err := json.Unmarshal([]byte(val), &item); err != nil {
			continue
		}
		result[i] = item
	}

	if c.logger != nil {
		c.logger.Info("get zset range success", "key", key, "start", start, "stop", stop, "members", len(result), "duration", duration)
	}

	return result, nil
}

// Increment 原子递增计数器
func (c *RedisCache) Increment(ctx context.Context, key string, increment int64) (int64, error) {
	start := time.Now()
	result, err := c.client.IncrBy(ctx, c.fullKey(key), increment).Result()
	duration := time.Since(start)

	if err != nil {
		if c.logger != nil {
			c.logger.Error(err, "increment failed", "key", key, "increment", increment, "duration", duration)
		}
		return 0, err
	}

	if c.logger != nil {
		c.logger.Info("increment success", "key", key, "increment", increment, "result", result, "duration", duration)
	}

	return result, nil
}

// Decrement 原子递减计数器
func (c *RedisCache) Decrement(ctx context.Context, key string, decrement int64) (int64, error) {
	start := time.Now()
	result, err := c.client.DecrBy(ctx, c.fullKey(key), decrement).Result()
	duration := time.Since(start)

	if err != nil {
		if c.logger != nil {
			c.logger.Error(err, "decrement failed", "key", key, "decrement", decrement, "duration", duration)
		}
		return 0, err
	}

	if c.logger != nil {
		c.logger.Info("decrement success", "key", key, "decrement", decrement, "result", result, "duration", duration)
	}

	return result, nil
}

// Expire 设置键的过期时间
func (c *RedisCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	start := time.Now()
	err := c.client.Expire(ctx, c.fullKey(key), ttl).Err()
	duration := time.Since(start)

	if err != nil {
		if c.logger != nil {
			c.logger.Error(err, "set expire failed", "key", key, "ttl", ttl, "duration", duration)
		}
		return err
	}

	if c.logger != nil {
		c.logger.Info("set expire success", "key", key, "ttl", ttl, "duration", duration)
	}

	return nil
}

// Persist 移除键的过期时间
func (c *RedisCache) Persist(ctx context.Context, key string) error {
	start := time.Now()
	err := c.client.Persist(ctx, c.fullKey(key)).Err()
	duration := time.Since(start)

	if err != nil {
		if c.logger != nil {
			c.logger.Error(err, "remove expire failed", "key", key, "duration", duration)
		}
		return err
	}

	if c.logger != nil {
		c.logger.Info("remove expire success", "key", key, "duration", duration)
	}

	return nil
}

// Exists 检查缓存键是否存在。
//
// 参数:
//   - ctx: 上下文
//   - key: 缓存键
//
// 返回值:
//   - bool: 是否存在
//   - error: 错误
func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.client.Exists(ctx, c.fullKey(key)).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// TTL 获取键的剩余过期时间。
//
// 参数:
//   - ctx: 上下文
//   - key: 缓存键
//
// 返回值:
//   - time.Duration: 剩余过期时间
//   - error: 错误
func (c *RedisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.client.TTL(ctx, c.fullKey(key)).Result()
}

// GetMulti 批量获取缓存值。
//
// 参数:
//   - ctx: 上下文
//   - keys: 缓存键列表
//
// 返回值:
//   - map[string]any: 缓存值映射
//   - error: 错误
func (c *RedisCache) GetMulti(ctx context.Context, keys []string) (map[string]any, error) {
	if len(keys) == 0 {
		return make(map[string]any), nil
	}
	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = c.fullKey(key)
	}
	values, err := c.client.MGet(ctx, fullKeys...).Result()
	if err != nil {
		return nil, err
	}
	result := make(map[string]any)
	for i, val := range values {
		if val == nil {
			continue
		}
		str, ok := val.(string)
		if !ok {
			continue
		}
		var item any
		if err := json.Unmarshal([]byte(str), &item); err != nil {
			continue
		}
		result[keys[i]] = item
	}
	return result, nil
}

// SetMulti 批量设置缓存值。
//
// 参数:
//   - ctx: 上下文
//   - items: 缓存项映射
//   - ttl: 过期时间
//
// 返回值:
//   - error: 错误
func (c *RedisCache) SetMulti(ctx context.Context, items map[string]any, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}
	pipe := c.client.Pipeline()
	for key, value := range items {
		data, err := json.Marshal(value)
		if err != nil {
			pipe.Discard()
			return err
		}
		if ttl > 0 {
			pipe.Set(ctx, c.fullKey(key), data, ttl)
		} else {
			pipe.Set(ctx, c.fullKey(key), data, 0)
		}
	}
	_, err := pipe.Exec(ctx)
	return err
}

// DeleteMulti 批量删除缓存值。
//
// 参数:
//   - ctx: 上下文
//   - keys: 缓存键列表
//
// 返回值:
//   - error: 错误
func (c *RedisCache) DeleteMulti(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = c.fullKey(key)
	}
	return c.client.Del(ctx, fullKeys...).Err()
}

// Clear 清除所有缓存。
//
// 参数:
//   - ctx: 上下文
//
// 返回值:
//   - error: 错误
func (c *RedisCache) Clear(ctx context.Context) error {
	keys, err := c.client.Keys(ctx, c.prefix+"*").Result()
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	return c.client.Del(ctx, keys...).Err()
}

// GetWithGetter 获取缓存值，如果不存在则调用回调函数获取并缓存。
//
// 参数:
//   - ctx: 上下文
//   - key: 缓存键
//   - fn: 获取数据的回调函数
//
// 返回值:
//   - any: 缓存值
//   - error: 错误
func (c *RedisCache) GetWithGetter(ctx context.Context, key string, fn cache.Getter) (any, error) {
	val, err := c.Get(ctx, key)
	if err == nil {
		return val, nil
	}
	if !errors.Is(err, cache.ErrNotFound) {
		return nil, err
	}

	val, err = fn(ctx, key)
	if err != nil {
		return nil, err
	}
	if val != nil {
		if cacheErr := c.Set(ctx, key, val, 0); cacheErr != nil {
			if c.logger != nil {
				c.logger.Error(cacheErr, "failed to cache value after getter",
					"key", key,
				)
			}
		}
	}
	return val, nil
}

// Close 关闭 Redis 连接。
func (c *RedisCache) Close() error {
	if c.logger != nil {
		c.logger.Info("closing redis connection")
	}
	return c.client.Close()
}

// Client 返回底层 Redis 客户端。
func (c *RedisCache) Client() redis.UniversalClient {
	return c.client
}
