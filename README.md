自用的工具包，基本上都是以服务方式封装，依赖于 [boot engine](https://github.com/coffeehc/boot) 使用的,包含基础的一些service，项目可以直接引用

1. asyncservice 异步执行服务
2. coder 编码处理 json，pb
3. cryptos 加解密处理
4. dbsource 数据库处理，支持mysql，pg
5. httpc httpClient封装
6. keylockservice 本地全局锁服务
7. 嵌入式本地kv存储 (基于[pebble](https://github.com/cockroachdb/pebble))
8. memcache 内存缓存服务
9. models 基础的pb数据结构定义
10. redisservice redis 服务封装
11. sequences 全局序列号生成器
12. utils 一些基础的utill类
13. webfacade 对web service 处理的一些封装，主要是配合 [httpx](https://github.com/coffeehc/httpx)使用，本质是对gin的扩展