// Package redis 提供 Redis 缓存客户端的自动配置。
//
// 当 redis.enabled=true 时自动启用，从 Environment 中读取 redis.address、redis.username、
// redis.password、redis.db、redis.pool-size、redis.use-cluster 等配置项，
// 创建并注册 Redis Cache Bean 到 IoC 容器中（Bean ID: redisCache），实现 cache.Cache 接口。
//
// 同时会自动注册 Redis 健康指标（Bean ID: redisHealthIndicator），
// 使用 PING 命令进行 Redis 连接检查。
package redis

import (
	"context"

	rediscore "github.com/xudefa/go-boot-redis"

	"github.com/xudefa/go-boot/actuator"
	"github.com/xudefa/go-boot/boot"
	"github.com/xudefa/go-boot/cache"
	"github.com/xudefa/go-boot/condition"
	"github.com/xudefa/go-boot/constants"
	"github.com/xudefa/go-boot/core"
)

// init 注册 Redis 自动配置，由 redis.enabled=true 条件控制。
func init() {
	boot.RegisterAutoConfig(&RedisAutoConfiguration{},
		condition.OnProperty(constants.RedisEnabled, constants.ConditionTrue),
	)
}

// RedisAutoConfiguration Redis 缓存客户端的自动配置。
//
// 从 Environment 中读取 redis.address、redis.password、redis.pool-size 等配置项，
// 创建 Redis Cache 实例并注册到 IoC 容器中，实现 cache.Cache 接口。
// 启用条件：redis.enabled=true
type RedisAutoConfiguration struct{}

// Configure 执行自动配置逻辑，创建 Redis Cache 并注册为 Bean。
//
// 同时注册 Redis 健康指标，用于监控 Redis 连接状态。
func (r *RedisAutoConfiguration) Configure(ctx boot.ApplicationContext) error {
	env := ctx.Environment()

	clientOpts := []rediscore.ClientOption{
		rediscore.WithAddress(env.GetString(constants.RedisAddress, constants.DefaultRedisAddress)),
		rediscore.WithUsername(env.GetString(constants.RedisUsername, "")),
		rediscore.WithPassword(env.GetString(constants.RedisPassword, "")),
		rediscore.WithDB(env.GetInt(constants.RedisDB, constants.DefaultRedisDB)),
		rediscore.WithPoolSize(env.GetInt(constants.RedisPoolSize, constants.DefaultRedisPoolSize)),
		rediscore.WithMaxActiveConnections(env.GetInt(constants.RedisMaxActiveConnections, constants.DefaultRedisMaxActiveConnections)),
		rediscore.WithMinIdleConnections(env.GetInt(constants.RedisMinIdleConnections, constants.DefaultRedisMinIdleConnections)),
		rediscore.WithCluster(env.GetBool(constants.RedisUseCluster, constants.DefaultRedisUseCluster)),
	}

	cacheInst, err := rediscore.NewCacheWithConfig(clientOpts)
	if err != nil {
		return err
	}

	if err := ctx.Register(constants.RedisCacheBeanID,
		core.Bean(cacheInst),
		core.Singleton(),
	); err != nil {
		return err
	}

	redisHealthIndicator := actuator.NewRedisHealthIndicator(func(ctx context.Context) error {
		if rc, ok := cacheInst.(*rediscore.RedisCache); ok {
			return rc.Client().Ping(ctx).Err()
		}
		return nil
	})

	if err := ctx.Register(constants.RedisHealthIndicatorBeanID,
		core.Bean(redisHealthIndicator),
		core.Singleton(),
	); err != nil {
		return err
	}

	return nil
}

var _ cache.Cache = (*rediscore.RedisCache)(nil)
