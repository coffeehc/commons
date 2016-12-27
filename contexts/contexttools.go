package contexts

import (
	"context"
	"time"
)

func AsInt(cxt context.Context,key string) int{
	v:=cxt.Value(key)
	if i,ok:=v.(int);ok{
		return i
	}
	return 0
}

func AsInt32(cxt context.Context,key string) int32{
	v:=cxt.Value(key)
	if i,ok:=v.(int32);ok{
		return i
	}
	return 0
}

func AsInt64(cxt context.Context,key string) int64{
	v:=cxt.Value(key)
	if i,ok:=v.(int64);ok{
		return i
	}
	return 0
}

func AsFloat32(cxt context.Context,key string) float32{
	v:=cxt.Value(key)
	if i,ok:=v.(float32);ok{
		return i
	}
	return 0
}

func AsFloat64(cxt context.Context,key string) float64{
	v:=cxt.Value(key)
	if i,ok:=v.(float64);ok{
		return i
	}
	return 0
}

func AsTime(cxt context.Context,key string) time.Time{
	v:=cxt.Value(key)
	if i,ok:=v.(time.Time);ok{
		return i
	}
	return time.Unix(0,0)
}

func AsString(cxt context.Context,key string) string{
	v:=cxt.Value(key)
	if i,ok:=v.(string);ok{
		return i
	}
	return ""
}