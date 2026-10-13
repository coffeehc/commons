package webfacade

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"unicode/utf8"
)

// BaseResponse 定义 Web/API 共用的 JSON 响应外壳。
type BaseResponse interface {
	IsSuccess() bool
	GetMessage() string
	GetPayload() any
	GetRequestId() string
	GetCode() int64
}

// AjaxResponse 是 Web/API 原生响应外壳；仅 Success 决定业务成功，Code 由业务定义。
// 列表分页信息放在 Payload 内，不提升到响应外壳顶层。
type AjaxResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Payload   any    `json:"payload"`
	RequestID string `json:"request_id"`
	Code      int64  `json:"code"`
	Redirect  string `json:"redirect,omitempty"`
}

// IsSuccess 返回业务成功标记，不推导应用状态码。
func (a *AjaxResponse) IsSuccess() bool { return a.Success }

// GetMessage 返回公开响应消息。
func (a *AjaxResponse) GetMessage() string { return a.Message }

// GetPayload 返回业务载荷。
func (a *AjaxResponse) GetPayload() any { return a.Payload }

// GetRequestId 返回请求标识。
func (a *AjaxResponse) GetRequestId() string { return a.RequestID }

// GetCode 返回业务自定义状态码。
func (a *AjaxResponse) GetCode() int64 { return a.Code }

const (
	// MaxSafeInteger 是 JSON/JavaScript 客户端可精确表示的最大整数。
	MaxSafeInteger int64 = 9007199254740991
	// MaxCursorBytes 与既有 bizlog 游标传输预算一致，按 UTF-8 字节计数。
	MaxCursorBytes = 4096
)

var (
	// ErrInvalidListData 表示列表载荷缺字段、类型错误或总数超出安全整数范围。
	ErrInvalidListData = errors.New("invalid list response")
	// ErrInvalidCursor 表示游标为空、编码非法或超出长度预算。
	ErrInvalidCursor = errors.New("invalid pagination cursor")
)

// Cursor 表示服务明确提供的游标结果。nil *Cursor 表示没有提供游标状态，
// EndCursor 表示明确末页；令牌不透明，不裁剪、不解码，也不猜测其 ID 含义。
type Cursor struct{ token *string }

// NextPageCursor 校验并创建续页游标；空串、非法 UTF-8 或超长令牌返回错误。
func NextPageCursor(token string) (*Cursor, error) {
	if !validCursor(token) {
		return nil, ErrInvalidCursor
	}
	return &Cursor{token: &token}, nil
}

// EndCursor 创建明确末页，序列化为 JSON null；不会省略字段。
func EndCursor() *Cursor { return &Cursor{} }

// Token 返回续页令牌；false 表示明确末页。
// 调用前先检查 ListData.NextCursor 非 nil，以区分未提供游标状态。
func (c Cursor) Token() (string, bool) {
	if c.token == nil {
		return "", false
	}
	return *c.token, true
}

func validCursor(token string) bool {
	return token != "" && len(token) <= MaxCursorBytes && utf8.ValidString(token)
}

// MarshalJSON 将明确末页写为 null，将续页令牌原样写为字符串。
func (c Cursor) MarshalJSON() ([]byte, error) {
	if c.token == nil {
		return []byte("null"), nil
	}
	if !validCursor(*c.token) {
		return nil, ErrInvalidCursor
	}
	return json.Marshal(*c.token)
}

// UnmarshalJSON 读取明确末页或有效续页令牌；字段缺失由 ListData 保留。
func (c *Cursor) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		c.token = nil
		return nil
	}
	var token string
	if json.Unmarshal(data, &token) != nil || !validCursor(token) {
		return ErrInvalidCursor
	}
	c.token = &token
	return nil
}

// ListData 是标准列表载荷。Total 为 nil 表示未计算，指向零表示已知总数为零。
// NextCursor 为 nil 表示未提供游标状态，不代表末页；总数与游标可以同时存在。
// 输出的 list 恒为数组；为兼容已发布 httpx，读取 list:null 时归一为空切片。
type ListData[T any] struct {
	// List 输出恒为数组，nil 切片输出为空数组。
	List []T `json:"list"`
	// Total 仅在实际计算总数后赋值；nil 不表示零。
	Total *int64 `json:"total,omitempty"`
	// NextCursor 为 nil 时省略；明确末页必须使用 EndCursor。
	NextCursor *Cursor `json:"next_cursor,omitempty"`
}

// MarshalJSON 校验总数并保证列表输出为数组，未知总数与未提供游标均省略。
func (d ListData[T]) MarshalJSON() ([]byte, error) {
	if d.Total != nil && (*d.Total < 0 || *d.Total > MaxSafeInteger) {
		return nil, ErrInvalidListData
	}
	list := d.List
	if list == nil {
		list = []T{}
	}
	type wire struct {
		// List 输出恒为数组，nil 切片输出为空数组。
		List []T `json:"list"`
		// Total 仅在实际计算总数后赋值；nil 不表示零。
		Total      *int64  `json:"total,omitempty"`
		NextCursor *Cursor `json:"next_cursor,omitempty"`
	}
	return json.Marshal(wire{list, d.Total, d.NextCursor})
}

// UnmarshalJSON 严格校验列表字段，并保留游标缺失、续页和明确末页三态。
func (d *ListData[T]) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if json.Unmarshal(data, &fields) != nil || fields == nil {
		return ErrInvalidListData
	}
	list, ok := fields["list"]
	if !ok {
		return ErrInvalidListData
	}
	var decoded ListData[T]
	if json.Unmarshal(list, &decoded.List) != nil {
		return ErrInvalidListData
	}
	if decoded.List == nil {
		decoded.List = []T{}
	}
	if total, ok := fields["total"]; ok {
		if bytes.Equal(bytes.TrimSpace(total), []byte("null")) {
			return ErrInvalidListData
		}
		var number float64
		if json.Unmarshal(total, &number) != nil || number < 0 || number > float64(MaxSafeInteger) || math.Trunc(number) != number {
			return ErrInvalidListData
		}
		value := int64(number)
		decoded.Total = &value
	}
	if cursor, ok := fields["next_cursor"]; ok {
		// 直接解码 **Cursor 会把显式 null 与字段缺失混为一谈，故先保留存在性。
		decoded.NextCursor = EndCursor()
		if err := decoded.NextCursor.UnmarshalJSON(cursor); err != nil {
			return err
		}
	}
	*d = decoded
	return nil
}
