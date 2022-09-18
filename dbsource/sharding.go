package dbsource

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type ShardService interface {
	GetTableName(v interface{}) string
	ShardingTableName(sql string, v interface{}) string
}
type shardServiceImpl struct {
	tableName     string
	suffixBuilder SuffixBuilder
}

func (impl *shardServiceImpl) ShardingTableName(sql string, v interface{}) string {
	return strings.ReplaceAll(sql, impl.tableName, impl.GetTableName(v))
}

func (impl *shardServiceImpl) GetTableName(v interface{}) string {
	if impl.suffixBuilder == nil {
		return impl.tableName
	}
	return fmt.Sprintf(" %s%s ", strings.TrimSpace(impl.tableName), impl.suffixBuilder(v))
}

func NewShardService(tableName string, suffixBuilder SuffixBuilder) ShardService {
	impl := &shardServiceImpl{
		tableName:     fmt.Sprintf(" %s ", tableName),
		suffixBuilder: suffixBuilder,
	}
	return impl
}

type SuffixBuilder = func(value interface{}) string

func NoSuffixBuilder(value interface{}) string {
	return ""
}

func ModSuffixBuilder(mod int64) SuffixBuilder {
	return func(value interface{}) string {
		i := value.(int64)
		return fmt.Sprintf("_%s", strconv.FormatInt(i%mod, 10))
	}
}

func RangSuffixBuilder(rang int64) SuffixBuilder {
	return func(value interface{}) string {
		i := value.(int64)
		return fmt.Sprintf("_%s", strconv.FormatInt(int64(math.Ceil(float64(i/rang))), 10))
	}
}
