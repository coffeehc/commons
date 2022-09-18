package utils

import (
	"context"
	"time"
)

// AsInt 从上下文获取 Key 对应的值,并转换为 int
func ContextAsInt(cxt context.Context, key interface{}) int {
	v := cxt.Value(key)
	if i, ok := v.(int); ok {
		return i
	}
	return 0
}

// AsUint 从上下文获取 Key 对应的值,并转换为 uint
func ContextAsUint(cxt context.Context, key interface{}) uint {
	v := cxt.Value(key)
	if i, ok := v.(uint); ok {
		return i
	}
	return 0
}

// AsInt32 从上下文获取 Key 对应的值,并转换为 int32
func ContextAsInt32(cxt context.Context, key interface{}) int32 {
	v := cxt.Value(key)
	if i, ok := v.(int32); ok {
		return i
	}
	return 0
}

// AsUint32 从上下文获取 Key 对应的值,并转换为 uint32
func ContextAsUint32(cxt context.Context, key interface{}) uint32 {
	v := cxt.Value(key)
	if i, ok := v.(uint32); ok {
		return i
	}
	return 0
}

// AsUint32Point 从上下文获取 Key 对应的值,并转换为 *uint32
func ContextAsUint32Point(cxt context.Context, key interface{}) *uint32 {
	v := cxt.Value(key)
	if i, ok := v.(*uint32); ok {
		return i
	}
	return nil
}

// AsInt32Point 从上下文获取 Key 对应的值,并转换为 int32指针
func ContextAsInt32Point(cxt context.Context, key interface{}) *int32 {
	v := cxt.Value(key)
	if i, ok := v.(*int32); ok {
		return i
	}
	return nil
}

// AsInt64 从上下文获取 Key 对应的值,并转换为 int64
func ContextAsInt64(cxt context.Context, key interface{}) int64 {
	v := cxt.Value(key)
	if i, ok := v.(int64); ok {
		return i
	}
	return 0
}

// AsUint64 从上下文获取 Key 对应的值,并转换为 uint64
func ContextAsUint64(cxt context.Context, key interface{}) uint64 {
	v := cxt.Value(key)
	if i, ok := v.(uint64); ok {
		return i
	}
	return 0
}

// AsUint64Point 从上下文获取 Key 对应的值,并转换为 int64
func ContextAsUint64Point(cxt context.Context, key interface{}) *uint64 {
	v := cxt.Value(key)
	if i, ok := v.(*uint64); ok {
		return i
	}
	return nil
}

// AsInt64Point 从上下文获取 Key 对应的值,并转换为 int64指针
func ContextAsInt64Point(cxt context.Context, key interface{}) *int64 {
	v := cxt.Value(key)
	if i, ok := v.(*int64); ok {
		return i
	}
	return nil
}

// AsFloat32 从上下文获取 Key 对应的值,并转换为 float32
func ContextAsFloat32(cxt context.Context, key interface{}) float32 {
	v := cxt.Value(key)
	if i, ok := v.(float32); ok {
		return i
	}
	return 0
}

// AsFloat64 从上下文获取 Key 对应的值,并转换为 float64
func ContextAsFloat64(cxt context.Context, key interface{}) float64 {
	v := cxt.Value(key)
	if i, ok := v.(float64); ok {
		return i
	}
	return 0
}

// AsTime 从上下文获取 Key 对应的值,并转换为 Time
func ContextAsTime(cxt context.Context, key interface{}) time.Time {
	v := cxt.Value(key)
	if i, ok := v.(time.Time); ok {
		return i
	}
	return time.Unix(0, 0)
}

// AsString 从上下文获取 Key 对应的值,并转换为 string
func ContextAsString(cxt context.Context, key interface{}) string {
	v := cxt.Value(key)
	if i, ok := v.(string); ok {
		return i
	}
	return ""
}
