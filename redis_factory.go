// Package redis 基于 Redis 提供缓存实现。
//
// 该包将 Redis 与 go-boot 缓存接口集成，
// 支持内存缓存和 Redis 分布式缓存。
package redis

import (
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/xudefa/go-boot/cache"
)

// clientConfig Redis 客户端基础配置
type clientConfig struct {
	address              string // Redis 服务器地址
	username             string // 用户名
	password             string // 密码
	db                   int    // 数据库编号
	poolSize             int    // 连接池大小
	maxActiveConnections int    // 最大活跃连接数
	minIdleConnections   int    // 最小空闲连接数
	useCluster           bool   // 是否使用集群模式
}

// ClientOption Redis 客户端配置选项
type ClientOption func(*clientConfig)

// WithAddress 设置 Redis 地址
func WithAddress(addr string) ClientOption {
	return func(c *clientConfig) {
		c.address = addr
	}
}

// WithUsername 设置 Redis 用户名
func WithUsername(username string) ClientOption {
	return func(c *clientConfig) {
		c.username = username
	}
}

// WithPassword 设置 Redis 密码
func WithPassword(password string) ClientOption {
	return func(c *clientConfig) {
		c.password = password
	}
}

// WithDB 设置 Redis 数据库编号
func WithDB(db int) ClientOption {
	return func(c *clientConfig) {
		c.db = db
	}
}

// WithPoolSize 设置连接池大小
func WithPoolSize(size int) ClientOption {
	return func(c *clientConfig) {
		c.poolSize = size
	}
}

// WithMaxActiveConnections 设置最大活跃连接数
func WithMaxActiveConnections(n int) ClientOption {
	return func(c *clientConfig) {
		c.maxActiveConnections = n
	}
}

// WithMinIdleConnections 设置最小空闲连接数
func WithMinIdleConnections(n int) ClientOption {
	return func(c *clientConfig) {
		c.minIdleConnections = n
	}
}

// WithCluster 启用集群模式
func WithCluster(enabled bool) ClientOption {
	return func(c *clientConfig) {
		c.useCluster = enabled
	}
}

// CacheOption 缓存配置选项函数
type CacheOption func(*cacheOptions)

// cacheOptions 缓存配置
type cacheOptions struct {
	prefix     string        // 键前缀
	defaultTTL time.Duration // 默认过期时间
	logger     Logger        // 日志记录器
}

// WithPrefix 设置键前缀
func WithPrefix(prefix string) CacheOption {
	return func(o *cacheOptions) {
		o.prefix = prefix
	}
}

// WithDefaultTTL 设置默认过期时间
func WithDefaultTTL(ttl time.Duration) CacheOption {
	return func(o *cacheOptions) {
		o.defaultTTL = ttl
	}
}

// WithLogger 设置日志记录器
func WithLogger(logger Logger) CacheOption {
	return func(o *cacheOptions) {
		o.logger = logger
	}
}

// NewClient 创建新的 Redis 客户端（普通模式）。
//
// 参数:
//   - opts: 配置选项
//
// 返回值:
//   - *redis.Client: Redis 客户端实例
func NewClient(opts ...ClientOption) *redis.Client {
	cfg := &clientConfig{
		address: "localhost:6379",
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return redis.NewClient(buildRedisOptions(cfg))
}

// NewClusterClient 创建新的 Redis 集群客户端。
//
// 参数:
//   - opts: 配置选项
//
// 返回值:
//   - *redis.ClusterClient: Redis 集群客户端实例
func NewClusterClient(opts ...ClientOption) *redis.ClusterClient {
	cfg := &clientConfig{
		address: "localhost:6379",
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return redis.NewClusterClient(buildRedisClusterOptions(cfg))
}

// NewUniversalClient 创建通用 Redis 客户端（自动判断集群模式）。
//
// 参数:
//   - opts: 配置选项
//
// 返回值:
//   - redis.UniversalClient: Redis 通用客户端接口
func NewUniversalClient(opts ...ClientOption) redis.UniversalClient {
	cfg := &clientConfig{
		address: "localhost:6379",
	}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.useCluster {
		return redis.NewClusterClient(buildRedisClusterOptions(cfg))
	}
	return redis.NewClient(buildRedisOptions(cfg))
}

// NewCache 创建缓存实例。
//
// 参数:
//   - client: Redis 客户端
//   - opts: 可选缓存配置项（如键前缀、默认过期时间等）
//
// 返回值:
//   - cache.Cache: 缓存接口实例
//   - error: 创建错误
func NewCache(client redis.UniversalClient, opts ...CacheOption) (cache.Cache, error) {
	if client == nil {
		return nil, fmt.Errorf("redis client must not be nil")
	}

	options := &cacheOptions{
		prefix:     "",
		defaultTTL: 5 * time.Second,
		logger:     nil,
	}
	for _, opt := range opts {
		opt(options)
	}

	redisCache, err := NewRedisCache(options.prefix, options.defaultTTL, client)
	if err != nil {
		return nil, err
	}
	// 设置日志记录器
	redisCache.logger = options.logger
	return redisCache, nil
}

// NewCacheWithConfig 从配置选项创建缓存实例。
//
// 参数:
//   - clientOpts: Redis 客户端配置选项
//   - cacheOpts: 缓存配置选项
//
// 返回值:
//   - cache.Cache: 缓存接口实例
//   - error: 创建错误
func NewCacheWithConfig(clientOpts []ClientOption, cacheOpts ...CacheOption) (cache.Cache, error) {
	client := NewUniversalClient(clientOpts...)
	return NewCache(client, cacheOpts...)
}

// NewAdvancedClient 创建高级 Redis 客户端
func NewAdvancedClient(opts ...AdvancedClientOption) *redis.Client {
	cfg := mergeAdvancedConfig(opts...)
	return redis.NewClient(buildAdvancedRedisOptions(cfg))
}

// NewAdvancedClusterClient 创建高级 Redis 集群客户端
func NewAdvancedClusterClient(opts ...AdvancedClientOption) *redis.ClusterClient {
	cfg := mergeAdvancedConfig(opts...)
	return redis.NewClusterClient(buildAdvancedRedisClusterOptions(cfg))
}

// NewAdvancedUniversalClient 创建高级通用 Redis 客户端
func NewAdvancedUniversalClient(opts ...AdvancedClientOption) redis.UniversalClient {
	cfg := mergeAdvancedConfig(opts...)
	if cfg.useCluster {
		return redis.NewClusterClient(buildAdvancedRedisClusterOptions(cfg))
	}
	return redis.NewClient(buildAdvancedRedisOptions(cfg))
}

func buildRedisOptions(cfg *clientConfig) *redis.Options {
	advancedCfg := &advancedClientConfig{
		clientConfig: *cfg,
		// 使用默认高级配置
		dialTimeout:        5 * time.Second,
		readTimeout:        3 * time.Second,
		writeTimeout:       3 * time.Second,
		maxRetries:         -1,
		poolTimeout:        4 * time.Second,
		idleTimeout:        5 * time.Minute,
		idleCheckFrequency: time.Minute,
		minRetryBackoff:    8 * time.Millisecond,
		maxRetryBackoff:    1 * time.Second,
	}
	return buildAdvancedRedisOptions(advancedCfg)
}

func buildRedisClusterOptions(cfg *clientConfig) *redis.ClusterOptions {
	advancedCfg := &advancedClientConfig{
		clientConfig: *cfg,
		// 使用默认高级配置
		dialTimeout:        5 * time.Second,
		readTimeout:        3 * time.Second,
		writeTimeout:       3 * time.Second,
		maxRetries:         -1,
		poolTimeout:        4 * time.Second,
		idleTimeout:        5 * time.Minute,
		idleCheckFrequency: time.Minute,
		minRetryBackoff:    8 * time.Millisecond,
		maxRetryBackoff:    1 * time.Second,
	}
	return buildAdvancedRedisClusterOptions(advancedCfg)
}
