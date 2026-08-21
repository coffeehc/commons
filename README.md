# CoffeeHC Commons

一个基于 Go 1.27 的通用工具包，以服务方式封装了各种基础设施组件，依赖于 [boot engine](https://github.com/coffeehc/boot) 框架。

## 核心服务模块

### 1. asyncservice - 异步执行服务
- 基于 Go 标准库实现动态并发上限和任务生命周期管理
- 支持延迟任务、定时任务调度
- 默认并发上限 100,000，不预分配 goroutine
- 提供 `Submit`、`Schedule`、`AfterFunc`、`ChangePoolSize`、`PoolStatus` 等方法
- 内置 `timingwheel` 包统一管理 timer 注册与停止

### 2. dbsource - 数据库服务
- 支持 MySQL、PostgreSQL、SQLite3
- 对外提供统一的执行、查询、动态行、预处理语句和事务接口
- SQLite/MySQL 由无插件的 `sqlxdialect` 子包实现
- PostgreSQL 由无插件的 `pgdialect` 子包基于原生 `pgxpool` 实现
- 只有父包 `dbsource` 负责插件注册、监控和实现选择，不暴露底层连接池
- `dbsource.Open` 可创建并关闭独立实例，支持迁移时同时连接源库和目标库
- 业务 SQL 统一使用 `?` 占位符，PostgreSQL 标识符和占位符由方言改写；原生 JSON `?` 操作符写作 `??`
- PostgreSQL TLS 通过 `Config.SSLMode`、`SSLRootCert`、`SSLCert` 和 `SSLKey` 配置；空模式保留 pgx 默认行为
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
- 内置纯 Go 磁盘 FIFO 实现，兼容原有 segment 和 metadata 文件格式
- 同名队列具备进程内和跨进程独占保护
- 单个文件最大 512MB
- 最大消息 4MB
- 每 100 次读写操作或 5 秒同步

### 11. embeddedmqservice - 嵌入式 MQ 服务
- 提供持久化入队、领取、确认、指数退避重试和死信能力
- 支持消息去重、稳定分片和同分片顺序消费
- 支持租约超时恢复、业务 payload 清理和死信保留策略
- 保证 at-least-once 投递，业务处理方负责幂等
- reader 与 worker 统一由 `asyncservice` 执行

### 12. refcache - 引用缓存服务
- 用于缓存引用对象

### 13. bufpool - 缓冲区池
- byte buffer 复用，减少 GC 压力

### 14. coder - 编解码服务
- 支持 JSON（json-iterator）
- 支持 Protobuf
- 自动平台适配

### 15. cryptos - 加密服务
- AES 加密（CBC 模式，支持 PKCS5/PKCS7/Zeros 填充）
- 哈希函数
- 数据脱敏
- 随机数生成

### 16. ipcreator - IP 生成器
- 内置各省 IP 段数据
- 支持按省份随机生成 IP
- 支持全国范围随机生成

### 17. utils - 工具类
- 时间处理
- 字符串处理
- 类型转换
- 文件操作
- 数据脱敏
- 上下文工具
- 任务限流

### 18. models - 基础数据模型
- 基于 Protobuf 的数据结构定义

## 设计特点

1. **统一的服务模式**：所有服务都通过 `EnablePlugin` 初始化，通过 `GetService` 获取
2. **插件化架构**：基于 boot engine 的插件系统
3. **上下文传递**：大多数操作都支持 context.Context
4. **可配置性**：通过 viper 支持配置覆盖
5. **监控支持**：支持操作监控和埋点
6. **高性能**：使用连接池、动态并发控制和缓存优化

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
