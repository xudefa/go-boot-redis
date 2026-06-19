// Package redis 提供高级 Redis 客户端配置选项。
//
// 包含连接超时、TLS、重试策略等高级配置，
// 以及构建 Redis Options 和 Cluster Options 的内部函数。
package redis

import (
	"crypto/tls"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// AdvancedClientOption 高级 Redis 客户端配置选项
type AdvancedClientOption func(*advancedClientConfig)

// advancedClientConfig 高级 Redis 客户端配置
type advancedClientConfig struct {
	clientConfig
	// 连接相关
	dialTimeout        time.Duration // 拨号超时时间
	readTimeout        time.Duration // 读取超时时间
	writeTimeout       time.Duration // 写入超时时间
	minRetryBackoff    time.Duration // 最小重试退避时间
	maxRetryBackoff    time.Duration // 最大重试退避时间
	maxRetries         int           // 最大重试次数
	poolTimeout        time.Duration // 连接池超时时间
	idleTimeout        time.Duration // 空闲连接超时时间
	idleCheckFrequency time.Duration // 空闲连接检查频率

	// TLS 相关
	tlsConfig *tls.Config // TLS 配置
	enableTLS bool        // 是否启用 TLS

	// 命令相关
	disableIndentity bool   // 是否禁用客户端身份标识
	identitySuffix   string // 客户端身份后缀
}

// WithDialTimeout 设置拨号超时时间
func WithDialTimeout(timeout time.Duration) AdvancedClientOption {
	return func(c *advancedClientConfig) {
		c.dialTimeout = timeout
	}
}

// WithReadTimeout 设置读取超时时间
func WithReadTimeout(timeout time.Duration) AdvancedClientOption {
	return func(c *advancedClientConfig) {
		c.readTimeout = timeout
	}
}

// WithWriteTimeout 设置写入超时时间
func WithWriteTimeout(timeout time.Duration) AdvancedClientOption {
	return func(c *advancedClientConfig) {
		c.writeTimeout = timeout
	}
}

// WithMaxRetries 设置最大重试次数
func WithMaxRetries(retries int) AdvancedClientOption {
	return func(c *advancedClientConfig) {
		c.maxRetries = retries
	}
}

// WithPoolTimeout 设置连接池超时时间
func WithPoolTimeout(timeout time.Duration) AdvancedClientOption {
	return func(c *advancedClientConfig) {
		c.poolTimeout = timeout
	}
}

// WithIdleTimeout 设置连接空闲超时时间
func WithIdleTimeout(timeout time.Duration) AdvancedClientOption {
	return func(c *advancedClientConfig) {
		c.idleTimeout = timeout
	}
}

// WithIdleCheckFrequency 设置空闲连接检查频率
func WithIdleCheckFrequency(freq time.Duration) AdvancedClientOption {
	return func(c *advancedClientConfig) {
		c.idleCheckFrequency = freq
	}
}

// WithMinRetryBackoff 设置最小重试退避时间
func WithMinRetryBackoff(backoff time.Duration) AdvancedClientOption {
	return func(c *advancedClientConfig) {
		c.minRetryBackoff = backoff
	}
}

// WithMaxRetryBackoff 设置最大重试退避时间
func WithMaxRetryBackoff(backoff time.Duration) AdvancedClientOption {
	return func(c *advancedClientConfig) {
		c.maxRetryBackoff = backoff
	}
}

// WithTLS 启用 TLS 连接
func WithTLS(config *tls.Config) AdvancedClientOption {
	return func(c *advancedClientConfig) {
		c.tlsConfig = config
		c.enableTLS = true
	}
}

// WithDisableIdentity 禁用客户端身份标识
func WithDisableIdentity(disable bool) AdvancedClientOption {
	return func(c *advancedClientConfig) {
		c.disableIndentity = disable
	}
}

// WithAddressForAdvanced 设置 Redis 地址（用于高级选项）
func WithAddressForAdvanced(addr string) AdvancedClientOption {
	return func(c *advancedClientConfig) {
		c.address = addr
	}
}

// WithIdentitySuffix 设置客户端身份后缀
func WithIdentitySuffix(suffix string) AdvancedClientOption {
	return func(c *advancedClientConfig) {
		c.identitySuffix = suffix
	}
}

// WithLoggerToAdvanced 设置日志记录器（用于高级选项）
func WithLoggerToAdvanced(logger Logger) AdvancedClientOption {
	// 这个函数只是为了让高级客户端支持logger，但实际logger由缓存层处理
	return func(c *advancedClientConfig) {
		// 日志记录器由缓存层处理
	}
}

// mergeAdvancedConfig 合并基础配置和高级配置，返回带默认值的高级配置
func mergeAdvancedConfig(opts ...AdvancedClientOption) *advancedClientConfig {
	cfg := &advancedClientConfig{
		clientConfig: clientConfig{
			address: "localhost:6379",
		},
		// 默认值
		dialTimeout:        5 * time.Second,
		readTimeout:        3 * time.Second,
		writeTimeout:       3 * time.Second,
		maxRetries:         -1, // 默认不重试
		poolTimeout:        4 * time.Second,
		idleTimeout:        5 * time.Minute,
		idleCheckFrequency: time.Minute,
		minRetryBackoff:    8 * time.Millisecond,
		maxRetryBackoff:    1 * time.Second,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

// buildAdvancedRedisOptions 构建增强版 Redis 单机客户端选项
func buildAdvancedRedisOptions(cfg *advancedClientConfig) *redis.Options {
	opts := &redis.Options{
		Addr:     cfg.address,
		Username: cfg.username,
		Password: cfg.password,
		DB:       cfg.db,

		// 连接相关
		DialTimeout:     cfg.dialTimeout,
		ReadTimeout:     cfg.readTimeout,
		WriteTimeout:    cfg.writeTimeout,
		MaxRetries:      cfg.maxRetries,
		MinRetryBackoff: cfg.minRetryBackoff,
		MaxRetryBackoff: cfg.maxRetryBackoff,
		PoolSize:        cfg.poolSize,
		PoolTimeout:     cfg.poolTimeout,
		MinIdleConns:    cfg.minIdleConnections,
		MaxActiveConns:  cfg.maxActiveConnections,

		// TLS
		TLSConfig: cfg.tlsConfig,
	}

	if cfg.enableTLS && opts.TLSConfig == nil {
		opts.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}

	if !cfg.disableIndentity {
		opts.IdentitySuffix = cfg.identitySuffix
	}

	return opts
}

// buildAdvancedRedisClusterOptions 构建增强版 Redis 集群客户端选项
func buildAdvancedRedisClusterOptions(cfg *advancedClientConfig) *redis.ClusterOptions {
	opts := &redis.ClusterOptions{
		Addrs:    strings.Split(cfg.address, ","),
		Username: cfg.username,
		Password: cfg.password,

		// 连接相关
		DialTimeout:     cfg.dialTimeout,
		ReadTimeout:     cfg.readTimeout,
		WriteTimeout:    cfg.writeTimeout,
		MaxRetries:      cfg.maxRetries,
		MinRetryBackoff: cfg.minRetryBackoff,
		MaxRetryBackoff: cfg.maxRetryBackoff,
		PoolSize:        cfg.poolSize,
		PoolTimeout:     cfg.poolTimeout,
		MinIdleConns:    cfg.minIdleConnections,
		MaxActiveConns:  cfg.maxActiveConnections,

		// TLS
		TLSConfig: cfg.tlsConfig,
	}

	if cfg.enableTLS && opts.TLSConfig == nil {
		opts.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}

	if !cfg.disableIndentity {
		opts.IdentitySuffix = cfg.identitySuffix
	}

	return opts
}
