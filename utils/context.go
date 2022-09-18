package utils

import (
	"context"
	"time"
)

func ContextTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.TODO()
	}
	return context.WithTimeout(ctx, timeout)
}

func ContextTimeoutNoCancel(ctx context.Context, timeout time.Duration) context.Context {
	ctx, _ = ContextTimeout(ctx, timeout)
	return ctx
}
