package localdbservice

import (
	"context"

	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/bloom"
	"github.com/cockroachdb/pebble/v2/sstable"
	"github.com/coffeehc/base/errors"
	"github.com/coffeehc/base/log"
	"github.com/coffeehc/commons/coder"
	"github.com/coffeehc/commons/sequences"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

const configKeyForDataDir = "localstorage.datadir"

func SetDataDir(dataDir string) {
	viper.Set(configKeyForDataDir, dataDir)
}

var Separator = []byte("\t\t\t")

type RangeHandler func(key []byte, value []byte) (bool, error)

type Service interface {
	Range(startKey, endKey []byte, reverse bool, maxCount int, handler RangeHandler) error
	RangeWithContext(ctx context.Context, startKey, endKey []byte, reverse bool, maxCount int, handler RangeHandler) error

	Set(key []byte, value []byte) error
	Get(key []byte) ([]byte, bool, error)
	Del(key []byte) error
	DelRange(start, end []byte) error
	GetDB() *pebble.DB

	SetPB(key []byte, body proto.Message) error
	GetPB(key []byte, body proto.Message) (bool, error)
	SetWithCoder(key []byte, body interface{}, coder2 coder.Coder) error
	GetWithCoder(key []byte, body interface{}, coder2 coder.Coder) (bool, error)
}

func newService(ctx context.Context) Service {
	viper.SetDefault(configKeyForDataDir, "./datas")
	dataDir := viper.GetString(configKeyForDataDir)
	log.Debug("打开数据文件", zap.String("dataDir", dataDir))
	options := newPebbleOptions()
	defer options.Cache.Unref()
	storage, err := pebble.Open(dataDir, options)
	if err != nil {
		log.Panic("打开dataDir文件错误", zap.Error(err))
		return nil
	}
	sequences.EnablePlugin(ctx)
	impl := &serviceImpl{
		storage:         storage,
		sequenceService: sequences.GetService(),
	}
	return impl
}

// newPebbleOptions 按 Pebble v2 的配置模型构造本地存储参数。
// TargetFileSizes 从 L0 起按倍数增长，未显式配置的层级由 Pebble 继承前一层策略。
func newPebbleOptions() *pebble.Options {
	comparer := *pebble.DefaultComparer
	// Pebble v2 迁移只调整 Options 结构，保留旧数据库的 comparer split 行为。
	comparer.Split = func([]byte) int {
		return 0
	}
	options := &pebble.Options{
		Cache:                 pebble.NewCache(1024 * 1024 * 32),
		BytesPerSync:          32 << 20, // 32 MiB
		Comparer:              &comparer,
		MaxOpenFiles:          500,
		LBaseMaxBytes:         64 << 20, // 64 MB
		L0CompactionThreshold: 50,
		L0StopWritesThreshold: 200,
	}
	options.TargetFileSizes[0] = 4 << 30
	options.TargetFileSizes[1] = 8 << 30
	options.TargetFileSizes[2] = 16 << 30
	options.Levels[0] = pebble.LevelOptions{
		Compression: func() *sstable.CompressionProfile {
			return sstable.NoCompression
		},
		FilterPolicy: bloom.FilterPolicy(10),
	}
	options.Levels[1] = pebble.LevelOptions{
		Compression: func() *sstable.CompressionProfile {
			return sstable.NoCompression
		},
		FilterType:   pebble.TableFilter,
		FilterPolicy: bloom.FilterPolicy(5),
	}
	options.Levels[2] = pebble.LevelOptions{
		Compression: func() *sstable.CompressionProfile {
			return sstable.SnappyCompression
		},
		FilterType:   pebble.TableFilter,
		FilterPolicy: bloom.FilterPolicy(1),
	}
	options.Experimental.L0CompactionConcurrency = 15
	options.Experimental.CompactionDebtConcurrency = 10
	return options
}

type serviceImpl struct {
	storage         *pebble.DB
	sequenceService sequences.SequenceService
}

func (impl *serviceImpl) SetPB(key []byte, body proto.Message) error {
	return impl.SetWithCoder(key, body, coder.PBCoder)
}

func (impl *serviceImpl) GetPB(key []byte, body proto.Message) (bool, error) {
	return impl.GetWithCoder(key, body, coder.PBCoder)
}

func (impl *serviceImpl) SetWithCoder(key []byte, body interface{}, coder2 coder.Coder) error {
	data, err := coder2.Marshal(body)
	if err != nil {
		return err
	}
	return impl.Set(key, data)
}

func (impl *serviceImpl) GetWithCoder(key []byte, body interface{}, coder2 coder.Coder) (bool, error) {
	data, ok, err := impl.Get(key)
	if err != nil || !ok {
		return ok, err
	}
	err = coder2.Unmarshal(data, body)
	if err != nil {
		return false, err
	}
	return ok, nil
}

func (impl *serviceImpl) RangeWithContext(ctx context.Context, startKey, endKey []byte, reverse bool, maxCount int, handler RangeHandler) error {
	iter, err := impl.storage.NewIter(nil)
	if err != nil {
		return err
	}
	defer iter.Close()
	iter.SetBounds(startKey, endKey)
	next := iter.Next
	first := iter.First
	if reverse {
		next = iter.Prev
		first = iter.Last
	}
	count := 0
	for ok := first(); ok && count < maxCount && ctx.Err() == nil; ok = next() {
		ok, err := handler(iter.Key(), iter.Value())
		if err != nil {
			return err
		}
		if !ok {
			break
		}
		count++
	}
	return nil
}

func (impl *serviceImpl) Range(startKey, endKey []byte, reverse bool, maxCount int, handler RangeHandler) error {
	return impl.RangeWithContext(context.TODO(), startKey, endKey, reverse, maxCount, handler)
}

func (impl *serviceImpl) Set(key []byte, value []byte) error {
	if len(key) == 0 {
		return errors.MessageError("存储的Key不合法，或者没有添加前缀")
	}
	err := impl.storage.Set(key, value, pebble.Sync)
	return errors.ConverError(err)
}

func (impl *serviceImpl) Get(key []byte) ([]byte, bool, error) {
	data, closer, err := impl.storage.Get(key)
	defer func() {
		if closer != nil {
			closer.Close()
		}
	}()
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, false, nil
		}
		return nil, false, errors.ConverError(err)
	}
	result := make([]byte, len(data))
	copy(result, data)
	return result, true, nil
}

func (impl *serviceImpl) Del(key []byte) error {
	err := impl.storage.Delete(key, pebble.Sync)
	return errors.ConverError(err)
}

func (impl *serviceImpl) DelRange(startKey, endKey []byte) error {
	err := impl.storage.DeleteRange(startKey, endKey, pebble.Sync)
	return errors.ConverError(err)
}

func (impl *serviceImpl) GetDB() *pebble.DB {
	return impl.storage
}
