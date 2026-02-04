# CoffeeHC Commons

一个基于 Go 的通用工具包，以服务方式封装了各种基础设施组件，依赖于 [boot engine](https://github.com/coffeehc/boot) 框架。

## 核心服务模块

### 1. asyncservice - 异步执行服务
- 基于协程池（ants）和时间轮实现
- 支持延迟任务、定时任务调度
- 默认协程池大小 100,000
- 提供 `Submit`、`Schedule`、`AfterFunc` 等方法

### 2. dbsource - 数据库服务
- 支持 MySQL、PostgreSQL、SQLite3
- 基于 sqlx 和 pgx 实现
- 内置 SQL 构建器（sqlbuilder）
- 支持事务、监控、分页查询
- 支持 sharding 分库分表

### 3. httpc - HTTP 客户端封装
- 基于 resty/v2 封装
- 内置 DNS 缓存（5分钟缓存时间，4分钟刷新）
- 支持连接池（MaxIdleConnsPerHost: 5000）
- 支持重试机制（默认 3 次）
- 支持 HTTP/2

### 4. redisservice - Redis 服务
- 基于 go-redis/v9 实现
- 支持单机和集群模式
- 支持虚拟 Redis 模式（用于测试）
- 内置分布式锁支持
- 提供监控和插件扩展

### 5. sequences - 全局序列号生成器
- 类似 Snowflake 算法
- 18位 ID（3 bits DC + 5 bits Node + 10 bits Sequence + 时间戳）
- 支持反向解析序列号
- 自定义纪元时间（Epoch: 1444281954363）

### 6. memcache - 内存缓存服务
- 基于 freecache 实现
- 默认缓存大小 8MB
- 支持 JSON/Protobuf 编码
- 可配置禁用缓存

### 7. webfacade - Web 服务门面
- 基于 Fiber 框架封装
- 支持查询参数解析
- 支持分页、排序、条件过滤
- 与 httpx 配合使用

### 8. keylockservice - 键锁服务
- 本地全局锁服务
- 用于防止并发重复操作

### 9. localdbservice - 本地嵌入式数据库
- 基于 cockroachdb/pebble 实现
- 提供嵌入式 KV 存储

### 10. localqueue - 本地持久化队列
- 基于 nsqio/go-diskqueue 实现
- 单个文件最大 512MB
- 最大消息 4MB
- 每 100 次写入或 5 秒同步

### 11. refcache - 引用缓存服务
- 用于缓存引用对象

### 12. bufpool - 缓冲区池
- byte buffer 复用，减少 GC 压力

### 13. coder - 编解码服务
- 支持 JSON（json-iterator）
- 支持 Protobuf
- 自动平台适配

### 14. cryptos - 加密服务
- AES 加密（CBC 模式，支持 PKCS5/PKCS7/Zeros 填充）
- 哈希函数
- 数据脱敏
- 随机数生成

### 15. ipcreator - IP 生成器
- 内置各省 IP 段数据
- 支持按省份随机生成 IP
- 支持全国范围随机生成

### 16. utils - 工具类
- 时间处理
- 字符串处理
- 类型转换
- 文件操作
- 数据脱敏
- 上下文工具
- 任务限流

### 17. models - 基础数据模型
- 基于 Protobuf 的数据结构定义

## 设计特点

1. **统一的服务模式**：所有服务都通过 `EnablePlugin` 初始化，通过 `GetService` 获取
2. **插件化架构**：基于 boot engine 的插件系统
3. **上下文传递**：大多数操作都支持 context.Context
4. **可配置性**：通过 viper 支持配置覆盖
5. **监控支持**：支持操作监控和埋点
6. **高性能**：大量使用连接池、协程池、缓存优化

## 技术栈

主要依赖：
- `coffeehc/boot` - 插件框架
- `coffeehc/httpx` - HTTP 框架
- `coffeehc/base` - 基础库
- `go-redis/v9` - Redis 客户端
- `resty/v2` - HTTP 客户端
- `sqlx` / `pgx` - 数据库驱动
- `freecache` - 内存缓存
- `pebble/v2` - 嵌入式数据库
- `ants/v2` - 协程池
- `fiber/v2` - Web 框架
- `viper` - 配置管理
- `zap` - 日志
- `google.golang.org/protobuf` - Protobuf

## 使用方式

### 引入包

```go
import _ "github.com/coffeehc/commons"
```

### 获取服务

```go
// 获取数据库服务
dbService := dbsource.GetService()

// 获取 Redis 服务
redisService := redisservice.GetService()

// 获取异步服务
asyncService := asyncservice.GetService()
```

## License

MIT
