# logservice：结构化错误记录与持久事件处理

第一版用于项目试接入。保留 `base/log` 原有技术日志，不自动订阅全部日志；业务在明确的失败边界调用 `Record`。包内默认 PostgreSQL 实现负责 schema、记录、聚合、outbox、投递和重试，接入方不需要实现 Store。

## 最小接入

```go
builder, err := logservice.New(logservice.Options{
    DataSource: db, // 已打开的 commons/dbsource.Service，必须是 PostgreSQL
    Config: logservice.Config{
        Scope: logservice.Scope{
            Application: "argus", Environment: "production", Tenant: "default",
        },
        AllowedAttributes: []string{"region", "provider"},
    },
    Schema: logservice.SchemaOptions{Namespace: "argus_logservice"},
})
if err != nil { return err }

err = builder.RegisterHandler(logservice.HandlerRegistration{
    ID: "collection-observer-v1",
    Filter: logservice.EventFilter{
        Kinds: []logservice.EventKind{logservice.EventErrorRecorded},
        Sources: []string{"collection"},
    },
    Delivery: logservice.DeliveryPolicy{
        Timeout: 5 * time.Second, MaxAttempts: 5, Concurrency: 1,
    },
}, logservice.HandlerFunc(func(ctx context.Context, d logservice.Delivery) error {
    // 稳定的 d.IdempotencyKey 供目标系统去重；错误信息正文不写回 logservice。
    // 第一轮试用建议只接观察/计数 handler，不直接执行恢复业务动作。
    return observer.Observe(ctx, d.IdempotencyKey, d.Event)
}))
if err != nil { return err }
svc, err := builder.Start(startupCtx)
if err != nil { return err }

receipt, err := svc.Record(ctx, logservice.RecordInput{
    ID: logservice.RecordID(observationID), // 同一次提交重试复用；新观察使用新 ID
    GroupKey: "provider-fetch-timeout",
    Facts: logservice.Facts{
        Source: "collection", Nature: logservice.NatureExternalDependency,
        Category: "network", Severity: logservice.SeverityError,
        Component: "collector", Operation: "fetch", Code: "provider_timeout",
        Summary: "Collection provider request timed out",
        Subject: logservice.Ref{Kind: "source", ID: sourceID},
    },
})
```

调用方需要根据业务决定如何处理写入失败。采集等主流程可用短调用 deadline 并继续原有错误返回；金融交易等流程不能把日志服务当作业务事务或可靠消息事务的替代品。`Record` 没有内存接收成功的中间态。

### 生命周期与现有基础设施

- `Start(ctx)` 的 context 只限制初始化，成功后由 `Close(ctx)` 停止 worker；取消 startup context 不会关闭服务。
- 在现有 `boot` 插件 Start 中构建/注册/启动，Stop 中先关闭 logservice，再关闭业务拥有的 dbsource。无需替换应用 boot 框架。
- `Close` 停止新操作，取消 handler context 并等待在途操作；超时返回 context 错误，清理仍继续。再次 `Close` 可继续等待。
- handler 和同步 hook 必须配合 context。无法强制终止忽略取消的 Go 函数，不为此无限启动替代 goroutine。
- 服务使用有界数量的持久投递 worker；未把 durable 队列塞进 `asyncservice` 的进程内等待队列，重启后以 PostgreSQL 为准。
- DataSource 是 `dbsource.Service` 的 `Ping` / `BeginTx` 子集。服务总是拥有独立数据库事务，外层业务事务回滚不会撤销已返回 Stored 的记录；两者不是一个原子业务事务。

## 记录、分类与查询

`Facts` 分开描述 Source、Nature、Category、Severity、组件、操作、错误码、关联对象及可重试/用户可修复属性。nil 布尔值代表未知。Normalizer → 第一个匹配的 Classifier → 自定义 Redactor → 内置脱敏。Classifier 只填调用者未显式提供的分类字段。hook 在 Start 前注册，注册后不可动态变更。

- `RecordStored`：记录、去重身份、分组计数、事件和匹配的投递已在同一事务提交。
- `RecordDuplicate`：同一 scope、ID 和规范化提交身份已存在，不增加计数、不发新事件。即使更改 hook/default 或关闭采集，也会先识别已持久化的重试。
- `RecordSuppressed` / `RecordDisabled`：没有创建新记录。
- ID 被其他提交内容复用返回 `ErrConflict`。数据库失败或提交结果不明返回 `ErrUnavailable`，不声称未提交；应复用原 ID 重试。
- producer 身份使用版本化的固定内置脱敏结果，在 mutable hook/default 之前计算；被移除的秘密、被截断的部分视作等价。不会持久化原始敏感输入或其裸哈希。
- ID 必须是 1–256 字节的字母数字开头标识符，其后允许 `_.:/-`；handler ID 最长128字节。识别出的 token/JWT 等凭据格式会被拒绝，不静默改写身份。
- `GroupKey` 推荐使用稳定模板标识，不能放用户输入或秘密。省略时按分类/来源/组件/操作/错误码聚合，无错误码时再加入脱敏 Summary。GroupingVersion 隔离分组算法调整。
- `ErrorGroup.EntryCount` 是去重后的错误观察条数，不是未解决故障数、业务实体数或 handler 尝试次数。本版没有 Incident/Resolve 功能，也不把 handler 成功解释为业务恢复。

`Query` 返回分组及同过滤条件的 MatchedGroups / MatchedEntries / MatchedEntryCount；`QueryRecords` 查询单条观察；`Detail` 给出分组和分页记录；`GetRecord` 查精确 ID。支持发生时间 `[Since, Until)`、来源、性质、类别、等级、组件、错误码、对象和脱敏摘要文本。

Scope 只能由可信应用装配绑定，不接受请求参数更换 tenant。所有查询和投递操作都带 scope。提供 HTTP 接口时，宿主仍需身份认证和授权。

分页默认50、最多200，按稳定 ID 升序，cursor 绑定 scope、查询类型与过滤条件。记录/分组页使用 scope 内事务序号高水位和数据库时间，后来提交的记录不会插入已有分页结果；过期 Detail/Stack 始终按读取时刻隐藏。`QueryDeliveries` 是实时运维状态，状态可能在两页之间变化，不是历史状态快照。

## 事件与 handler

第一版支持 `error.recorded` 和 `group.created`，事件是独立不可变的提交时脱敏摘要，不含 Detail/Stack。Filter 必须指定 Kinds 或显式 AllKinds；不填不能误订阅全部事件。每个匹配 handler 有独立持久投递、状态、尝试次数、重试策略和错误码。

- 至少一次投递；进程在副作用成功后、ACK 前退出可能重复调用。handler 使用 `IdempotencyKey` 做业务目标端幂等，执行前检查当前业务条件。
- `nil` 只确认消费或明确跳过。普通 error 重试；包装 `ErrPermanentHandler` 停止重试；panic 被隔离为安全错误码。错误正文不持久化，也不会递归写回错误记录。
- 默认 timeout30秒、最多5次、退避1秒至1分钟、最长投递年龄24小时、单 handler 并发1。并发上限32/实例；多个进程的总并发相加。
- SQL 使用 `FOR UPDATE SKIP LOCKED` 领取，lease 为该投递 Timeout+1秒，token/state/有效期共同限制 ACK。旧 worker 不能覆盖新尝试；副作用仍需目标端幂等。
- handler 通过传入的 ctx 派生调用 `Record`，保留 Root/Parent/Depth/HandlerPath。同一 handler 的因果链再次匹配会留下 `blocked/causation_loop`；超过深度上限返回 `ErrCausationLimit`。主动丢弃 ctx 无法保留这项保护。
- 默认最多10000条 pending/running 投递。新记录需要超过容量时整笔返回 `ErrBackpressure`，记录/计数/事件一起回滚；不会先返回成功后静默丢弃。
- `QueryDeliveries` 可看 pending/running/succeeded/failed/blocked、Attempts 和 LastErrorCode。没有手动 redrive API；不要直接修改表来伪造重试状态。

### 非破坏性的注册升级

同一 scope 的所有进程必须使用相同注册版本。首次 `RegistryVersion` 默认1；同版本改动 handler ID、filter 或策略会在 Start 返回 `ErrConflict`。

调整时：

1. 停止旧实例，等待 `Close` 排空在途 handler。
2. 新配置设置 `RegistryVersion: 2, PreviousRegistryVersion: 1`，注册新 handler 集合，启动。后续使用连续版本。
3. 成功后同版本实例可正常启动。旧版本不能继续写新记录/领取新投递；两个不同的 v2 升级只能一个成功。

升级通过数据库 CAS 并检查没有有效在途 lease。旧未消费事件保持创建时的 handler 目标、策略、幂等键和尝试次数；filter/策略更改影响新事件。移除 handler 时旧 pending 投递变为 `blocked/registry_missing_handler`；之后重新加入同一 ID 可恢复这些投递，但不会恢复 causation_loop 阻断。恢复后仍遵守原 MaxAge/MaxAttempts。新增 handler 不补发历史事件。当前注册的 Concurrency 是新实例 worker 数。

## 数据安全、schema 与保留

默认使用独立 `logservice` schema，建议项目指定自己的命名空间。仅允许小写安全 SQL 标识符。`MigrationAutoSafe` 自动创建自有 schema；已存在但不归本服务所有的 namespace 会拒绝接管。`MigrationVerifyOnly` 不创建 schema。启动使用迁移锁、版本检查和结构指纹（列、约束、索引）；缺表、改列或不兼容版本会拒绝启动，不做破坏性自动修复。不要向服务专属 schema 填入业务表。

内置脱敏覆盖已识别的 password/token/API key/Authorization、Cookie/Set-Cookie、URL 用户密码、JWT、PEM 私钥等格式；支持 JSON 转义字符串。Attributes 默认不接收，必须显式 allowlist，敏感键即使 allowlist 也会被丢弃。所有输入有字节/总量限制，输出有截断上限。任意自由文本中的未知秘密不可能靠通用规则全部识别；宿主应使用稳定错误码/摘要，避免采集请求体、连接串及用户私密内容，领域特有格式注册 Redactor。

默认 Detail/Stack 保留7天。读取到期后立即隐藏；每分钟清理和显式 `PruneDetails` 物理删除这两个字段。Summary、路由元数据、记录身份、去重、聚合、事件和投递历史本版持续保留；尚未实现整体归档/删除或成本控制。试用期应监控数据库容量，不能把细节过期当作全量数据删除承诺。

原始数据库异常不向 API 返回，保留 ErrUnavailable 和可识别的 context 错误。宿主自己的 dbsource/数据库诊断配置仍由宿主管理。

## 验证

```sh
go test ./...
go test -race ./...
go vet ./...
go build ./...

# 专用测试库，不读取应用配置。
LOGSERVICE_TEST_POSTGRES='postgres://test@127.0.0.1:5432/logservice_test?sslmode=disable' \
  ./logservice/test-postgres.sh -count=3

# 或使用已安装 PostgreSQL 创建/销毁临时 loopback-only 集群。
LOGSERVICE_PG_BIN=/usr/lib/postgresql/17/bin ./logservice/test-postgres.sh -count=3
```

真实 PostgreSQL 测试覆盖并发去重/计数、持久 payload 脱敏、隔离/分页/过期、事务失败回滚与提交 ACK 丢失、重启/独立重试/lease fencing、因果环、backpressure、注册 CAS 升级、结构漂移及宿主外层事务隔离。没有设置测试 DSN 时普通测试明确跳过 PostgreSQL 用例；模拟对象不能替代这些验证。

本版无 UI。后续项目 UI 可展示按来源/性质/类别过滤的分组列表、EntryCount、最近发生、记录详情和投递状态；恢复操作应依赖业务明确提供的命令与条件。
