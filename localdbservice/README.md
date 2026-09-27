# localdbservice

`localdbservice` 为调用方独占的 Pebble 数据目录提供基础读写、批量提交和生命周期管理。
包内默认配置以 Pebble 当前版本维护的默认值为基础，不为任一具体业务预设超大 SST、缓存、
压缩、Bloom Filter 或 Compaction 策略。

## 打开数据库

同步持久化使用便捷入口：

```go
store, err := localdbservice.Open(dataDir)
```

允许最多一个同步周期的数据尚未落盘时使用：

```go
store, err := localdbservice.OpenWithPeriodicSync(dataDir, time.Second)
```

需要针对调用方负载调整 Pebble 时，先取得独立默认配置，再只覆盖已经确认需要调整的字段：

```go
config := localdbservice.DefaultOpenConfig()
config.SyncInterval = time.Second
config.PebbleOptions.CacheSize = 128 << 20
config.PebbleOptions.MemTableSize = 64 << 20
config.PebbleOptions.MemTableStopWritesThreshold = 4
config.PebbleOptions.L0CompactionThreshold = 8
config.PebbleOptions.L0StopWritesThreshold = 32

store, err := localdbservice.OpenWithConfig(dataDir, config)
```

`OpenWithConfig` 会克隆传入的 `pebble.Options` 顶层字段，补齐 Pebble 默认值并执行完整校验。
调用方可以在打开后修改原配置中的标量字段；显式共享的 Cache 等引用资源仍遵循 Pebble 自身的
生命周期规则。数据库目录、Pebble 实例和周期同步协程由返回的 Service 独占，调用方必须在结束时
执行 `Close()`。

## 默认配置边界

`DefaultOpenConfig` 保留 localdbservice 历史数据库使用的 whole-key comparer split 行为，其余参数
交给 Pebble `EnsureDefaults()` 维护。当前 Pebble v2 默认配置包括：

- L0 Compaction Threshold 为 4；
- L0 Stop Writes Threshold 为 12；
- Base Level 容量目标为 64 MiB；
- L0 SST 目标大小为 2 MiB，后续 Level 逐层翻倍；
- 同步写入为默认持久化策略。

这些值是通用起点，不代表所有业务的最佳参数。大吞吐、超大 Value、点查询密集或磁盘受限的调用方
应根据自身写入速率、Value 大小、磁盘带宽、内存预算和读写比例显式配置。

## 参数含义

- `CacheSize`：Pebble Block Cache 的内存预算，不等同于操作系统页缓存。
- `MemTableSize`：单个内存写缓冲区的目标大小，写满后生成 L0 SST。
- `MemTableStopWritesThreshold`：等待刷盘的 MemTable 达到该数量时暂停新写入。
- `L0CompactionThreshold`：L0 积压达到阈值后开始 Compaction。
- `L0StopWritesThreshold`：L0 严重积压后暂停新写入。
- `LBaseMaxBytes`：动态 Base Level 的容量目标，不是数据库容量上限。
- `TargetFileSizes`：各 Level Compaction 输出 SST 的目标大小，不是严格文件上限。
- `MaxConcurrentCompactions`：允许同时执行的后台 Compaction 数量。
- `BytesPerSync`：控制 SST 写入时操作系统脏页回写节奏，不等同于 WAL 持久化周期。
- `SyncInterval`：localdbservice 的 WAL 周期同步窗口；零表示每次写入同步持久化。

L1 以后不直接限制 SST 文件数量。Pebble 根据 Level 容量目标和 `TargetFileSizes` 自动决定文件数；
L0 则通过 Compaction 和 Stop Writes 阈值控制积压。
