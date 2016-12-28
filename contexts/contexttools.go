package contexts

import (
	"context"
	"time"
)

//AsInt 从上下文获取 Key 对应的值,并转换为 int
func AsInt(cxt context.Context, key string) int {
	v := cxt.Value(key)
	if i, ok := v.(int); ok {
		return i
	}
	return 0
}

//AsUint 从上下文获取 Key 对应的值,并转换为 uint
func AsUint(cxt context.Context, key string) uint {
	v := cxt.Value(key)
	if i, ok := v.(uint); ok {
		return i
	}
	return 0
}

//AsInt32 从上下文获取 Key 对应的值,并转换为 int32
func AsInt32(cxt context.Context, key string) int32 {
	v := cxt.Value(key)
	if i, ok := v.(int32); ok {
		return i
	}
	return 0
}

//AsUint32 从上下文获取 Key 对应的值,并转换为 uint32
func AsUint32(cxt context.Context, key string) uint32 {
	v := cxt.Value(key)
	if i, ok := v.(uint32); ok {
		return i
	}
	return 0
}

//AsUint32Point 从上下文获取 Key 对应的值,并转换为 *uint32
func AsUint32Point(cxt context.Context, key string) *uint32 {
	v := cxt.Value(key)
	if i, ok := v.(*uint32); ok {
		return i
	}
	return nil
}

//AsInt32Point 从上下文获取 Key 对应的值,并转换为 int32指针
func AsInt32Point(cxt context.Context, key string) *int32 {
	v := cxt.Value(key)
	if i, ok := v.(*int32); ok {
		return i
	}
	return nil
}

//AsInt64 从上下文获取 Key 对应的值,并转换为 int64
func AsInt64(cxt context.Context, key string) int64 {
	v := cxt.Value(key)
	if i, ok := v.(int64); ok {
		return i
	}
	return 0
}

//AsUint64 从上下文获取 Key 对应的值,并转换为 uint64
func AsUint64(cxt context.Context, key string) uint64 {
	v := cxt.Value(key)
	if i, ok := v.(uint64); ok {
		return i
	}
	return 0
}

//AsUint64Point 从上下文获取 Key 对应的值,并转换为 int64
func AsUint64Point(cxt context.Context, key string) *uint64 {
	v := cxt.Value(key)
	if i, ok := v.(*uint64); ok {
		return i
	}
	return nil
}

//AsInt64Point 从上下文获取 Key 对应的值,并转换为 int64指针
func AsInt64Point(cxt context.Context, key string) *int64 {
	v := cxt.Value(key)
	if i, ok := v.(*int64); ok {
		return i
	}
	return nil
}

//AsFloat32 从上下文获取 Key 对应的值,并转换为 float32
func AsFloat32(cxt context.Context, key string) float32 {
	v := cxt.Value(key)
	if i, ok := v.(float32); ok {
		return i
	}
	return 0
}

//AsFloat64 从上下文获取 Key 对应的值,并转换为 float64
func AsFloat64(cxt context.Context, key string) float64 {
	v := cxt.Value(key)
	if i, ok := v.(float64); ok {
		return i
	}
	return 0
}

//AsTime 从上下文获取 Key 对应的值,并转换为 Time
func AsTime(cxt context.Context, key string) time.Time {
	v := cxt.Value(key)
	if i, ok := v.(time.Time); ok {
		return i
	}
	return time.Unix(0, 0)
}

//AsString 从上下文获取 Key 对应的值,并转换为 string
func AsString(cxt context.Context, key string) string {
	v := cxt.Value(key)
	if i, ok := v.(string); ok {
		return i
	}
	return ""
}
