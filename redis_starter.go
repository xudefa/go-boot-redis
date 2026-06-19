package redis

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	src "github.com/xudefa/go-boot"

	"github.com/redis/go-redis/v9"
)

// HealthChecker 实现 Redis 健康检查
type HealthChecker struct {
	client      redis.UniversalClient
	lastChecked atomic.Int64
	lastStatus  atomic.Bool
	testPing    bool
}

// NewHealthChecker 创建新的健康检查器
func NewHealthChecker(client redis.UniversalClient, testPing bool) *HealthChecker {
	hc := &HealthChecker{
		client:   client,
		testPing: testPing,
	}
	// 初始化状态为健康
	hc.lastStatus.Store(true)
	return hc
}

// CheckHealth 检查 Redis 连接健康状态
func (h *HealthChecker) CheckHealth(ctx context.Context) error {
	if h.client == nil {
		return fmt.Errorf("redis client is nil")
	}

	if !h.testPing {
		return nil
	}

	start := time.Now()
	err := h.client.Ping(ctx).Err()
	duration := time.Since(start)

	// 更新最后检查时间和状态
	h.lastChecked.Store(time.Now().Unix())
	h.lastStatus.Store(err == nil)

	if err != nil {
		return fmt.Errorf("redis health check failed: %w, response time: %v", err, duration)
	}

	return nil
}

// IsHealthy 返回上次健康检查的状态
func (h *HealthChecker) IsHealthy() bool {
	return h.lastStatus.Load()
}

// LastChecked 返回上次检查的时间戳
func (h *HealthChecker) LastChecked() int64 {
	return h.lastChecked.Load()
}

// Starter 实现 Redis 连接启动器。
//
// 用于在应用启动时初始化 Redis 连接并测试可用性。
type Starter struct {
	client        redis.UniversalClient
	testPing      bool
	healthChecker *HealthChecker
}

// NewStarter 创建新的 Redis 启动器。
//
// 参数:
//   - client: Redis 客户端（支持 *redis.Client 和 *redis.ClusterClient）
//   - testPing: 是否测试连接
//
// 返回值:
//   - *Starter: 启动器实例
func NewStarter(client redis.UniversalClient, testPing bool) *Starter {
	return &Starter{
		client:        client,
		testPing:      testPing,
		healthChecker: NewHealthChecker(client, testPing),
	}
}

// Starter 实现 boot.Starter 接口。
func (s *Starter) Starter() error {
	if s.client == nil {
		return nil
	}
	if s.testPing {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.healthChecker.CheckHealth(ctx)
	}
	return nil
}

// NewRedisStarter 创建 Redis 启动器。
func NewRedisStarter(client redis.UniversalClient, testPing bool) src.Starter {
	return NewStarter(client, testPing)
}

var _ src.Starter = (*Starter)(nil)

// ContextStarter 实现带上下文的 Redis 启动器。
type ContextStarter struct {
	client        redis.UniversalClient
	testPing      bool
	healthChecker *HealthChecker
}

// NewContextStarter 创建带上下文的 Redis 启动器。
func NewContextStarter(client redis.UniversalClient, testPing bool) *ContextStarter {
	return &ContextStarter{
		client:        client,
		testPing:      testPing,
		healthChecker: NewHealthChecker(client, testPing),
	}
}

// Start 实现 boot.ContextStarter 接口。
func (s *ContextStarter) Start(ctx context.Context) error {
	if s.client == nil {
		return nil
	}
	if s.testPing {
		return s.healthChecker.CheckHealth(ctx)
	}
	return nil
}

// GetHealthChecker 返回健康检查器
func (s *Starter) GetHealthChecker() *HealthChecker {
	return s.healthChecker
}

// GetHealthChecker 返回健康检查器
func (s *ContextStarter) GetHealthChecker() *HealthChecker {
	return s.healthChecker
}
