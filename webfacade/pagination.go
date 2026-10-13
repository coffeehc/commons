package webfacade

import (
	"errors"
	"net/url"
	"strconv"

	"github.com/gofiber/fiber/v3"
)

// PaginationMode 指定分页模式，禁止根据请求中的游标内容自动猜测。
type PaginationMode string

const (
	// OffsetPagination 使用页码与页大小。
	OffsetPagination PaginationMode = "offset"
	// CursorPagination 使用不透明续页令牌。
	CursorPagination PaginationMode = "cursor"
)

// ErrInvalidPagination 表示分页模式、参数或边界不合法。
var ErrInvalidPagination = errors.New("invalid pagination query")

// PaginationQuery 保存明确的分页模式与总数查询选择。解析器不执行 COUNT，
// 存储层必须遵守 ReturnTotal=false；Cursor 保持不透明，不解释为 ID 或 SQL 条件。
type PaginationQuery struct {
	Mode        PaginationMode
	PageIndex   int64
	PageSize    int64
	Cursor      string
	ReturnTotal bool
}

// ParsePaginationQuery 是独立的严格分页解析入口，不改变旧 ParsePageQuery。
// page_size 必填；offset 模式要求 page_index，cursor 模式缺省游标表示首屏。
// 显式空游标非法；return_total 缺省为 false，仅接受 true/false。
// 其他过滤参数仍由业务端点校验。
func ParsePaginationQuery(c fiber.Ctx, mode PaginationMode, maxPageSize int64) (PaginationQuery, error) {
	var out PaginationQuery
	if (mode != OffsetPagination && mode != CursorPagination) || maxPageSize < 1 || maxPageSize > MaxSafeInteger {
		return out, ErrInvalidPagination
	}
	params, err := url.ParseQuery(string(c.Request().URI().QueryString()))
	if err != nil {
		return out, ErrInvalidPagination
	}
	for _, key := range []string{"page_index", "page_size", "cursor", "return_total"} {
		if values, exists := params[key]; exists && len(values) != 1 {
			return out, ErrInvalidPagination
		}
	}
	size, err := canonicalNonnegative(params.Get("page_size"))
	if err != nil || size < 1 || size > maxPageSize {
		return out, ErrInvalidPagination
	}
	out = PaginationQuery{Mode: mode, PageSize: size}
	if values, exists := params["return_total"]; exists {
		switch values[0] {
		case "true":
			out.ReturnTotal = true
		case "false":
		default:
			return PaginationQuery{}, ErrInvalidPagination
		}
	}
	if mode == OffsetPagination {
		if _, exists := params["cursor"]; exists {
			return PaginationQuery{}, ErrInvalidPagination
		}
		index, err := canonicalNonnegative(params.Get("page_index"))
		if err != nil || index > MaxSafeInteger/size {
			return PaginationQuery{}, ErrInvalidPagination
		}
		out.PageIndex = index
	} else {
		if _, exists := params["page_index"]; exists {
			return PaginationQuery{}, ErrInvalidPagination
		}
		if values, exists := params["cursor"]; exists {
			if !validCursor(values[0]) {
				return PaginationQuery{}, ErrInvalidPagination
			}
			out.Cursor = values[0]
		}
	}
	return out, nil
}

func canonicalNonnegative(value string) (int64, error) {
	number, err := strconv.ParseInt(value, 10, 64)
	if err != nil || number < 0 || number > MaxSafeInteger || strconv.FormatInt(number, 10) != value {
		return 0, ErrInvalidPagination
	}
	return number, nil
}
