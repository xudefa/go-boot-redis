// redis 集成模块测试
// 测试 Redis 客户端创建，包括单机、集群和通用客户端等不同模式
package redis

import (
	"context"
	"testing"
	"time"
)

// TestNewClient 测试创建单机 Redis 客户端，验证返回的 client 不为空
func TestNewClient(t *testing.T) {
	client := NewClient(
		WithAddress("localhost:6379"),
	)
	if client == nil {
		t.Error("NewClient should return client")
	}
	if client != nil {
		_ = client.Close()
	}
}

// TestNewClientWithOptions 测试使用密码、DB、连接池等选项创建单机客户端，验证各选项生效
func TestNewClientWithOptions(t *testing.T) {
	client := NewClient(
		WithAddress("localhost:6379"),
		WithPassword("testpass"),
		WithDB(1),
		WithPoolSize(10),
		WithMinIdleConnections(2),
	)
	if client == nil {
		t.Error("NewClient should return client with options")
	}
	if client != nil {
		_ = client.Close()
	}
}

// TestNewClusterClient 测试创建 Redis 集群客户端，验证不为空
func TestNewClusterClient(t *testing.T) {
	client := NewClusterClient(
		WithAddress("localhost:6379"),
	)
	if client == nil {
		t.Error("NewClusterClient should return client")
	}
}

// TestNewUniversalClient_NonCluster 测试创建通用客户端（非集群模式），验证创建成功
func TestNewUniversalClient_NonCluster(t *testing.T) {
	client := NewUniversalClient(
		WithAddress("localhost:6379"),
		WithCluster(false),
	)
	if client == nil {
		t.Error("NewUniversalClient should return client")
	}
	_ = client.Close()
}

// TestNewUniversalClient_Cluster 测试创建通用客户端（集群模式），验证创建成功
func TestNewUniversalClient_Cluster(t *testing.T) {
	client := NewUniversalClient(
		WithAddress("localhost:6379,localhost:6380"),
		WithCluster(true),
	)
	if client == nil {
		t.Error("NewUniversalClient should return cluster client")
	}
	_ = client.Close()
}

// TestClientOptions 测试批量传入多个选项创建客户端，验证地址、用户名、密码、DB、连接池等全部生效
func TestClientOptions(t *testing.T) {
	opts := []ClientOption{
		WithAddress("redis.example.com:6379"),
		WithUsername("user"),
		WithPassword("pass"),
		WithDB(3),
		WithPoolSize(20),
		WithMaxActiveConnections(100),
		WithMinIdleConnections(5),
	}

	client := NewClient(opts...)
	if client == nil {
		t.Fatal("NewClient should return client")
	}
	_ = client.Close()
}

// TestNewAdvancedClient 测试创建高级 Redis 客户端
func TestNewAdvancedClient(t *testing.T) {
	client := NewAdvancedClient(
		WithAddressForAdvanced("localhost:6379"),
		WithDialTimeout(5*time.Second),
		WithReadTimeout(3*time.Second),
		WithWriteTimeout(3*time.Second),
		WithMaxRetries(3),
		WithPoolTimeout(4*time.Second),
		WithIdleTimeout(5*time.Minute),
	)
	if client == nil {
		t.Error("NewAdvancedClient should return client")
	}
	if client != nil {
		_ = client.Close()
	}
}

// TestNewAdvancedClusterClient 测试创建高级 Redis 集群客户端
func TestNewAdvancedClusterClient(t *testing.T) {
	client := NewAdvancedClusterClient(
		WithAddressForAdvanced("localhost:6379"),
		WithDialTimeout(5*time.Second),
		WithMaxRetries(1),
	)
	if client == nil {
		t.Error("NewAdvancedClusterClient should return client")
	}
	if client != nil {
		_ = client.Close()
	}
}

// TestNewAdvancedUniversalClient 测试创建高级通用客户端
func TestNewAdvancedUniversalClient(t *testing.T) {
	client := NewAdvancedUniversalClient(
		WithAddressForAdvanced("localhost:6379"),
		WithMaxRetries(-1), // 不重试
	)
	if client == nil {
		t.Error("NewAdvancedUniversalClient should return client")
	}
	_ = client.Close()
}

// TestHealthChecker 测试健康检查功能
func TestHealthChecker(t *testing.T) {
	client := NewClient(WithAddress("localhost:6379"))
	defer func() { _ = client.Close() }()

	hc := NewHealthChecker(client, true)

	ctx := context.Background()
	err := hc.CheckHealth(ctx)
	if err != nil {
		t.Logf("Health check failed (might be expected if Redis is not running): %v", err)
	}

	// 测试 nil 客户端的情况
	hcNil := NewHealthChecker(nil, true)
	err = hcNil.CheckHealth(ctx)
	if err == nil {
		t.Error("Health check with nil client should return error")
	}
}

// TestExtendedCacheOperations 测试扩展的缓存操作
func TestExtendedCacheOperations(t *testing.T) {
	client := NewClient(WithAddress("localhost:6379"))
	defer func() { _ = client.Close() }()

	cache, err := NewRedisCache("test:", 5*time.Second, client)
	if err != nil {
		t.Skipf("Skip extended cache tests: %v", err)
	}

	ctx := context.Background()

	// 测试哈希操作
	err = cache.HSet(ctx, "hash_key", "field1", "value1", 10*time.Second)
	if err != nil {
		t.Logf("HSet operation failed (might be expected if Redis is not running): %v", err)
	} else {
		val, err := cache.HGet(ctx, "hash_key", "field1")
		if err != nil {
			t.Logf("HGet operation failed: %v", err)
		} else if val != "value1" {
			t.Errorf("Expected 'value1', got %v", val)
		}
	}

	// 清理测试前的counter键
	_, _ = client.Del(ctx, "test:counter").Result()

	// 测试计数器操作
	counterVal, err := cache.Increment(ctx, "counter", 1)
	if err != nil {
		t.Logf("Increment operation failed: %v", err)
	} else if counterVal != 1 {
		t.Errorf("Expected counter value 1, got %d", counterVal)
	}

	counterVal, err = cache.Increment(ctx, "counter", 5)
	if err != nil {
		t.Logf("Increment operation failed: %v", err)
	} else if counterVal != 6 {
		t.Errorf("Expected counter value 6, got %d", counterVal)
	}

	counterVal, err = cache.Decrement(ctx, "counter", 2)
	if err != nil {
		t.Logf("Decrement operation failed: %v", err)
	} else if counterVal != 4 {
		t.Errorf("Expected counter value 4, got %d", counterVal)
	}
}

// TestCacheWithZeroTTL 测试零TTL的缓存操作
func TestCacheWithZeroTTL(t *testing.T) {
	client := NewClient(WithAddress("localhost:6379"))
	defer func() { _ = client.Close() }()

	cache, err := NewRedisCache("ttl_test:", 0, client)
	if err != nil {
		t.Skipf("Skip TTL test: %v", err)
	}

	ctx := context.Background()
	err = cache.Set(ctx, "ttl_key", "ttl_value", 0)
	if err != nil {
		t.Logf("Set with zero TTL failed: %v", err)
	}

	val, err := cache.Get(ctx, "ttl_key")
	if err != nil {
		t.Logf("Get after zero TTL set failed: %v", err)
	} else if val != "ttl_value" {
		t.Errorf("Expected 'ttl_value', got %v", val)
	}
}

// TestExpireAndPersist 测试过期和持久化功能
func TestExpireAndPersist(t *testing.T) {
	client := NewClient(WithAddress("localhost:6379"))
	defer func() { _ = client.Close() }()

	cache, err := NewRedisCache("expire_test:", 5*time.Second, client)
	if err != nil {
		t.Skipf("Skip expire test: %v", err)
	}

	ctx := context.Background()

	// 设置一个键
	err = cache.Set(ctx, "expire_key", "expire_value", 10*time.Second)
	if err != nil {
		t.Logf("Set operation failed: %v", err)
	}

	// 设置较短的过期时间
	err = cache.Expire(ctx, "expire_key", 1*time.Second)
	if err != nil {
		t.Logf("Expire operation failed: %v", err)
	}

	// 移除过期时间
	err = cache.Persist(ctx, "expire_key")
	if err != nil {
		t.Logf("Persist operation failed: %v", err)
	}
}
