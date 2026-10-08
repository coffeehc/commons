# CoffeeHC Commons

一个基于 Go 1.27 的通用工具包，以服务方式封装了各种基础设施组件，依赖于 [boot engine](https://github.com/coffeehc/boot) 框架。

## 核心服务模块

### logservice - 结构化错误服务（第一版）
- 内置 PostgreSQL 记录、去重聚合、持久事件与独立 handler 重试
- 直接复用 `dbsource.Service`；不要求业务实现 Store，不改变 `base/log`
- 提供分类/脱敏扩展、范围隔离查询、详情过期和非破坏性的注册升级
- [接入说明、边界与真实 PostgreSQL 验证](logservice/README.md)

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
- PostgreSQL TLS 通过 `Config.SSLMode`、`SSLRootCert`、`SSLCert` 和 `SSLKey` 配置；空模式固定为 `prefer`，不读取 `PGSSLMODE`
- 内置 SQL 构建器（sqlbuilder）
- 支持事务、监控、分页查询
- 支持 sharding 分库分表

#### PostgreSQL 配置来源

PostgreSQL 连接只使用调用方提供的配置和明确默认值，不让 `PG*` 环境变量、当前操作系统用户名、`.pgpass`、`.pg_service.conf` 或默认客户端证书覆盖配置或导致解析失败。`host`、`user`、`database` 必须明确填写；密码可以为空；端口默认 `5432`，连接超时默认 5 秒。`Config.SearchPath` 继续把一个 schema 作为完整标识符传递。

独立连接探测可使用 `pgdialect.ParsePoolConfig(dsn)`，返回原生 `*pgxpool.Config`，其 `ConnConfig` 可交给 `pgx.ConnectConfig`。支持 URL 和 keyword/value DSN、显式 TLS 证书路径及运行时/连接池参数；拒绝 `service`、`servicefile`、`passfile` 这些额外配置源。TLS 验证仍使用显式 CA 或 Go/操作系统正常的系统信任行为。

实现不修改进程环境，也不 fork/vendor pgx。pgx 内部仍会读取环境，但解析前已经覆盖所有对应设置。为隔离 pgx 的 service 选择机制，解析期间创建仅含固定空 section 的临时描述文件；不写入任何用户配置或密码，返回前删除。升级 pgx 时须复核新增的环境参数并运行 `go test -race ./dbsource/...`。

#### Shutdown cancellation 回归

调用 context 已取消或到期，且操作错误链仅包含对应 `context.Canceled` / `context.DeadlineExceeded` 时，dbsource 保留标准 `errors.Is` 身份，不记录错误日志或 DPanic。SQL、约束、连接故障、服务端超时以及混合真实错误仍走原有错误策略；不会仅凭 `ctx.Err()` 或 PostgreSQL `57014` 静默处理。

基线：commons `896d1e7`、pgx `v5.9.1`。真实 PostgreSQL 查询取消时，旧代码记录 DPanic，且 `base/errors.ConverError` 隐藏取消身份。pgx 的超时包装支持 `Unwrap`；标量扫描须优先返回延迟到 `Rows.Next` 才出现的终止错误。事务取消后若物理连接已关闭，回滚仍释放池租约，但不附加该关闭造成的重复错误；独立回滚失败仍保留。

普通回归不需要数据库。真实 PostgreSQL 回归须显式提供测试 DSN，每轮创建独立 `dbsource_cancel_*` schema 并验证删除，不读取应用配置：

```sh
# 仅指向专用测试数据库；凭据通过运行时环境提供。
export DBSOURCE_TEST_POSTGRES='host=127.0.0.1 port=5432 user=test dbname=dbsource_test sslmode=disable'
go test ./...
go test -race ./...
go vet ./...
go build ./...
```

测试通过 `pg_stat_activity` 确认查询已执行后才取消，覆盖 query/row/rows/exec/insert/statement、显式事务、HandleTx、deadline、真实 SQL/约束/连接错误与服务端 timeout。日志断言使用现有同步日志订阅，包含连接标记以避免未接入 logger 时误判为零日志。

真实 SIGTERM 测试通过 `bootintegration` 标签启用。使用包含当前 commons 和 boot checkout 的临时 workspace（boot 启动回滚基线为 `875fb40`），避免改变 commons 的依赖版本：

```sh
# 从 commons 根目录运行，boot checkout 位于相邻目录。
validation_dir=$(mktemp -d)
go -C "$validation_dir" work init "$PWD" "$PWD/../boot"
GOWORK="$validation_dir/go.work" go test -race -tags bootintegration ./dbsource -run '^TestPostgresShutdown' -count=1 -v
# 单独检查 boot 的启动回滚、非零退出与逆序关闭。
go -C ../boot test -race ./engine ./plugin
```

SIGTERM 测试在独立子进程中调用真实 `engine.WaitServiceStop`，模拟 worker Stop 的 cancel/join，再关闭数据库；同时覆盖 active 与 idle shutdown。2026-09-20 在 PostgreSQL 18.6 验证取消错误可识别、DPanic 为零、子进程退出码为零。boot 未作修改；其全模块检查仍有既存 Fiber API、`RegisterPluginByFast`、gRPC 示例及 vet 问题，engine/plugin 生命周期回归单独验证。

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
