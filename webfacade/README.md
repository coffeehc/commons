# Web/API 响应与分页合同

`github.com/coffeehc/commons/webfacade` 原生拥有 Web/API 响应模型、发送/错误处理、查询与分页传输合同。消费方处理这些职责只需引入该包。原 `httpx/httpxcommons` 的响应 API 已删除，不保留 alias 或 wrapper；该包中未迁移的证书加载功能不受影响。

## 响应外壳

`AjaxResponse` 保留 `success`、`message`、`payload`、`request_id`、`code`、可选 `redirect` 及 `BaseResponse` 方法。只有 `success: true` 决定业务成功，`code: 0` 也是合法成功码。业务码与 HTTP 状态仍由端点决定；分页字段位于 `payload` 内。

`SendSuccess`、`SendError`、`SendErrorWithRedirect`、`SendErrors` 保留既有外壳与 protobuf 协商。`SendError` 接收明确可公开的文本；`SendErrors` 只公开明确分类的 `base/errors.MessageError` 消息。未知、系统、数据库或混合组合错误统一使用“系统内部错误”。包装层文本不会外泄；组合中的系统/未知错误优先于公开业务消息。响应不附带底层错误或原始请求数据。

既有 protobuf 字段编号和宽松 `ParseQuery` 不变。统一错误 helper 使用 `SendPBError` 时，修正其原先忽略 `statusCode` 的问题；`SendPBErrors` 与 JSON 使用相同安全分类。有效 PB 响应字节由固定 golden 数据验证。PB 序列化失败仍遵循 commons 既有 `SendPBSuccess` 行为；已删除的旧 httpx helper 不再提供独立异常状态语义。

## 列表载荷

```go
// 未请求总数：不执行 COUNT，Total 保持 nil。
page := webfacade.ListData[Row]{List: rows}

// 只有实际计算过总数才赋值；零表示真实零。
total := int64(0)
page.Total = &total

// 游标模式必须明确给出续页令牌或末页。
next, err := webfacade.NextPageCursor(opaqueToken)
if err != nil { /* 处理生产方无效游标 */ }
page.NextCursor = next
page.NextCursor = webfacade.EndCursor() // 输出 next_cursor: null
```

- `list` 必填，生产方恒输出数组；nil 切片输出 `[]`。
- 读取兼容已发布 httpx 的 `list: null`，并归一为 `[]`；缺失或其他错误类型不接受。这是历史读取兼容，标准生产方不输出 null。
- 缺失 `total` 表示未计算，`total: 0` 表示已知零。存在时必须是 0 至 9007199254740991 的 JavaScript 安全整数；拒绝 null、字符串、负数、小数与不安全整数。生产方不输出 null。
- 缺失 `next_cursor` 表示未提供游标状态，例如 offset 模式；不能据此判断游标末页。非空字符串表示可续扫，显式 null 表示末页。`ListData` 解码保留这三种状态。
- `total` 与 `next_cursor` 可共存，互不推导。逐项权限过滤后允许空列表仍带非空续扫令牌；有令牌不保证下页有数据，也不说明剩余条数。游标模式只能以显式 null 终结，不能看列表长度猜末页。
- 游标是不透明且原样传输的 UTF-8 字符串，最多 4096 字节，与既有 bizlog 传输预算一致。不裁剪、不解 base64、不猜数字 ID、不自动生成 SQL 条件。空串非法；末页显式使用 `EndCursor()`。
- `Cursor.Token()` 在有续页令牌时返回 `(token, true)`，明确末页返回 `("", false)`；调用前先检查指针是否为 nil，以区分字段缺失。
- 旧 `httpxcommons.ListData[T]` Go API 已删除。生产方必须改用本包的 `ListData[T]` 和可选 `*int64` 总数；没有执行 COUNT 时保持 nil。

## 严格分页解析

`ParsePaginationQuery(c, OffsetPagination, maximumSize)` 与 `ParsePaginationQuery(c, CursorPagination, maximumSize)` 是独立的严格入口，不改变旧 `ParsePageQuery`。

- `page_size` 必填，必须是规范十进制正整数，且不超过端点明确指定的上限。
- Offset 模式要求规范非负 `page_index`；页码、页大小及其乘积必须在 JavaScript 安全整数范围内。禁止同时带 cursor。
- Cursor 模式缺失 `cursor` 表示首屏；后续请求携带不透明非空令牌。禁止同时带 page_index。令牌与范围、排序的绑定校验由业务端点负责。
- `return_total` 缺省 false；仅显式 true 请求计算，false 表示不计算。空、重复或格式错误的分页/总数参数被拒绝。其他过滤参数仍由业务端点验证。
- 解析器不访问数据库。服务/存储层必须在 `ReturnTotal == false` 时跳过 COUNT，并在结果中省略 Total；仅隐藏已计算的总数并不能减少数据库开销。
- 新严格入口是显式选用的协议，不要求既有业务端点自动收紧它们已经发布的输入范围。

## 共同 fixture 与检查

`testdata/webfacade-list.v1.json` 是后端与 xui 逐字节共用的合同样本，覆盖成功码零、缺省/零/最大安全总数、游标缺失/续页/末页/共存、UTF-8 字节边界、历史 null 列表以及坏值。同步时复制原字节，不独立分叉语义。

```sh
unset DBSOURCE_TEST_POSTGRES LOGSERVICE_TEST_POSTGRES
go test ./webfacade ./dbsource/dbmonitor
go test -race ./webfacade ./dbsource/dbmonitor
go vet ./webfacade ./dbsource/dbmonitor
```

本次不引入服务生命周期、DI 容器、新数据库抽象或游标编码方案。普通单测不构成真实 PostgreSQL COUNT 行为验证；业务存储需另行验证不计数路径。
