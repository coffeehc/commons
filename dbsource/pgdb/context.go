package pgdb

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const contextCurPoolKey = "__db_pool"
const contextCurTxKey = "__db_tx"

func setPoolToContext(ctx context.Context, pool *pgxpool.Pool) context.Context {
	return context.WithValue(ctx, contextCurPoolKey, pool)
}

func GetPoolFormContext(ctx context.Context) *pgxpool.Pool {
	value := ctx.Value(contextCurPoolKey)
	if value == nil {
		return nil
	}
	return value.(*pgxpool.Pool)
}

func setTxToContext(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, contextCurTxKey, tx)
}

func GetTxFormContext(ctx context.Context) pgx.Tx {
	value := ctx.Value(contextCurTxKey)
	if value == nil {
		return nil
	}
	return value.(pgx.Tx)
}
