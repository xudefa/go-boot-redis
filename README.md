# go-boot-redis

[![Go Version](https://img.shields.io/github/go-mod/go-version/xudefa/go-boot-redis)](https://go.dev/) [![License](https://img.shields.io/github/license/xudefa/go-boot-redis)](./LICENSE) [![Build Status](https://img.shields.io/github/actions/workflow/status/xudefa/go-boot-redis/test.yml?branch=master)](https://github.com/xudefa/go-boot-redis/actions) [![Go Reference](https://pkg.go.dev/badge/github.com/xudefa/go-boot-redis.svg)](https://pkg.go.dev/github.com/xudefa/go-boot-redis) [![Go Report Card](https://goreportcard.com/badge/github.com/xudefa/go-boot-redis)](https://goreportcard.com/report/github.com/xudefa/go-boot-redis)

基于 [go-boot](https://github.com/xudefa/go-boot) 的 Redis 缓存集成模块。将 go-redis 无缝集成到 go-boot 的 IoC 容器和自动配置体系中，提供高级 Redis 客户端配置、分布式缓存实现和健康检查能力。

> 设计理念：遵循 go-boot 的开发规范，通过函数式选项模式和自动配置实现零代码启动 Redis 缓存服务。

## 整体架构

```
┌───────────────────────────────────────────────────────────────────────┐
│                    go-boot ApplicationContext                         │
│  ┌───────────┐ ┌──────────────┐ ┌───────────┐ ┌───────────┐           │
│  │ Container │ │  Environment │ │ Lifecycle │ │ EventBus  │           │
│  └───────────┘ └──────────────┘ └───────────┘ └───────────┘           │
│                       ┌─────────────────────┐                         │
│                       │ AutoConfig Registry │                         │
│                       └─────────────────────┘                         │
└───────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
                    ┌───────────────────────────────┐
                    │    go-boot-redis Starter      │
                    │  ┌─────────────────────────┐  │
                    │  │ RedisCache Bean         │  │
                    │  │ Health Indicator        │  │
                    │  │ Client Factory          │  │
                    │  │ Cache Implementation    │  │
                    │  └─────────────────────────┘  │
                    └───────────────────────────────┘
```

## 目录

- [快速开始](#快速开始)
- [功能特性](#功能特性)
- [缓存操作](#缓存操作)
- [高级配置](#高级配置)
- [配置选项](#配置选项)
- [项目结构](#项目结构)
- [开发指南](#开发指南)
- [贡献](#贡献)
- [许可证](#许可证)

## 快速开始

### 安装

```bash
# 安装核心框架
go get github.com/xudefa/go-boot

# 安装 Redis 集成模块
go get github.com/xudefa/go-boot-redis
```

### 最小示例

```go
package main

import (
    "context"
    "fmt"

    "github.com/xudefa/go-boot/boot"
    "github.com/xudefa/go-boot/cache"
)

func main() {
    app, err := boot.NewApplication(
        boot.WithAppName("my-cache-app"),
        boot.WithVersion("1.0.0"),
    )
    if err != nil {
        panic(err)
    }
    defer app.Stop()

    // 启动应用（自动配置 Redis 缓存）
    app.Start()

    // 获取缓存实例并操作
    redisCache := app.Container().Get("redisCache").(cache.Cache)
    
    ctx := context.Background()
    redisCache.Set(ctx, "user:1001", `{"name":"Alice","age":30}`, 0)
    
    var value string
    redisCache.Get(ctx, "user:1001", &value)
    fmt.Println(value)

    // 等待终止信号
    app.WaitForSignal()
}
```

## 功能特性

| 特性 | 说明 |
|------|------|
| Redis 客户端工厂 | 支持普通客户端、集群客户端和通用客户端 |
| 自动配置 | 通过环境变量自动配置 Redis 连接 |
| 缓存实现 | 实现 go-boot `cache.Cache` 接口 |
| 健康检查 | 内置 Redis 健康指示器 |
| 函数式选项 | 灵活的连接配置（TLS、重试、连接池等） |
| 集群支持 | 自动识别并支持 Redis Cluster 模式 |
| 优雅启停 | 支持连接池优雅关闭 |

## 缓存操作

### 基本操作

```go
cache := app.Container().Get("redisCache").(cache.Cache)

// 设置缓存
cache.Set(ctx, "key", "value", 5*time.Minute)

// 获取缓存
var result string
err := cache.Get(ctx, "key", &result)

// 删除缓存
cache.Delete(ctx, "key")

// 检查是否存在
exists := cache.Contains(ctx, "key")
```

### JSON 序列化

```go
type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

user := User{ID: 1001, Name: "Alice"}
cache.Set(ctx, "user:1001", user, 10*time.Minute)

var retrieved User
cache.Get(ctx, "user:1001", &retrieved)
```

### 带前缀的缓存

```go
// 通过自动配置设置前缀
// redis.cache.prefix=app1:
cache.Set(ctx, "session:abc", data, 30*time.Minute)
// 实际 Redis 键：app1:session:abc
```

## 高级配置

### 创建 Redis 客户端

```go
import "github.com/xudefa/go-boot-redis/redis"

// 创建普通客户端
client := redis.NewClient(
    redis.WithAddress("localhost:6379"),
    redis.WithPassword("secret"),
    redis.WithDB(0),
)

// 创建集群客户端
clusterClient := redis.NewClusterClient(
    redis.WithClusterAddresses([]string{
        "node1:6379",
        "node2:6379",
        "node3:6379",
    }),
    redis.WithPassword("secret"),
)

// 创建通用客户端（自动判断集群模式）
universalClient := redis.NewUniversalClient(
    redis.WithAddress("localhost:6379"),
    redis.WithCluster(true),
)
```

### 高级连接选项

```go
client := redis.NewClient(
    redis.WithAddress("localhost:6379"),
    redis.WithPassword("secret"),
    redis.WithPoolSize(10),
    redis.WithMaxActiveConnections(20),
    redis.WithMinIdleConnections(5),
    redis.WithDialTimeout(5*time.Second),
    redis.WithReadTimeout(3*time.Second),
    redis.WithWriteTimeout(3*time.Second),
    redis.WithMaxRetries(3),
    redis.WithIdleTimeout(5*time.Minute),
)
```

### 创建缓存实例

```go
// 方式一：使用已有客户端
cache, err := redis.NewCache(client,
    redis.WithPrefix("app:"),
    redis.WithDefaultTTL(10*time.Minute),
)

// 方式二：一次性创建
cache, err := redis.NewCacheWithConfig(
    []redis.ClientOption{
        redis.WithAddress("localhost:6379"),
        redis.WithPassword("secret"),
    },
    redis.WithPrefix("app:"),
    redis.WithDefaultTTL(10*time.Minute),
)
```

## 配置选项

通过 `boot.WithProperty()` 或配置文件设置：

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `redis.enabled` | `true` | 是否启用 Redis 缓存 |
| `redis.address` | `localhost:6379` | Redis 服务器地址 |
| `redis.username` | `` | Redis 用户名 |
| `redis.password` | `` | Redis 密码 |
| `redis.db` | `0` | Redis 数据库编号 |
| `redis.pool-size` | `10` | 连接池大小 |
| `redis.max-active` | `20` | 最大活跃连接数 |
| `redis.min-idle` | `5` | 最小空闲连接数 |
| `redis.use-cluster` | `false` | 是否使用集群模式 |
| `redis.prefix` | `` | 缓存键前缀 |
| `redis.default-ttl` | `5s` | 默认过期时间 |

### 示例配置

```yaml
# application.yml
redis:
  enabled: true
  address: localhost:6379
  password: secret
  db: 0
  pool-size: 20
  max-active: 50
  min-idle: 10
  use-cluster: false
  prefix: "myapp:"
  default-ttl: 10m
```

## 项目结构

```
go-boot-redis/
├── redis_factory.go        # Redis 客户端工厂
├── redis_cache.go          # Redis 缓存实现
├── options.go              # 函数式选项配置
├── autoconfig.go           # 自动配置注册
├── redis_starter.go        # Redis 启动器
├── redis_test.go           # 单元测试
├── README.md
├── LICENSE
└── go.mod
```

## 开发指南

### 构建

```bash
go build ./...
```

### 测试

```bash
go test ./...
go test -cover ./...       # 带覆盖率
go test -race ./...        # 数据竞争检测
```

### 代码规范

```bash
go fmt ./...
golangci-lint run
```

## 贡献

欢迎提交 Issue 和 Pull Request！详细贡献指南请参阅 [CONTRIBUTING.md](./CONTRIBUTING.md)。

## 许可证

本项目采用 MIT 许可证 — 详情请参阅 [LICENSE](./LICENSE) 文件。