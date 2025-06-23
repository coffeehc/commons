package localdbservice

import (
	"bytes"
	"context"
	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/bloom"
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
	comparer := pebble.DefaultComparer
	comparer.Split = func(a []byte) int {
		index := bytes.LastIndex(a, Separator)
		if index < 0 {
			index = 0
		}
		return 0
	}
	log.Debug("打开数据文件", zap.String("dataDir", dataDir))
	options := &pebble.Options{
		Cache:                 pebble.NewCache(1024 * 1024 * 32),
		BytesPerSync:          32 << 20, //128MB = 128 << 20, // 512 KB = 512 << 10
		Comparer:              comparer,
		MaxOpenFiles:          500,
		LBaseMaxBytes:         64 << 20, //64 MB
		L0CompactionThreshold: 50,
		L0StopWritesThreshold: 200,
		Levels: []pebble.LevelOptions{
			{
				TargetFileSize: 4 << 30, //TargetFileSize：每个层级的目标文件大小。 1G
				Compression: func() pebble.Compression {
					return pebble.NoCompression
				},
				FilterPolicy: bloom.FilterPolicy(10),
				//BlockSize: 每个表块的目标未压缩大小，默认值为4096
				//BlockSizeThreshold：当块大小超过目标块大小的指定百分比，并且添加下一个条目将导致块超过目标块大小时，结束块，默认值为90。
				//FilterPolicy：减少Get操作的磁盘读取的过滤算法，默认值为nil，表示不使用过滤器。
				//IndexBlockSize：每个索引块的目标未压缩大小，默认值为BlockSize的值。
			},
			{
				TargetFileSize: 8 << 30,
				Compression: func() pebble.Compression {
					return pebble.NoCompression
				},
				FilterType:   pebble.TableFilter,
				FilterPolicy: bloom.FilterPolicy(5),
			},
			{
				TargetFileSize: 16 << 30,
				Compression: func() pebble.Compression {
					return pebble.SnappyCompression
				},
				//FilterType:     pebble.TableFilter,
				FilterType:   pebble.TableFilter,
				FilterPolicy: bloom.FilterPolicy(1),
			},
		},
	}
	//options.MaxConcurrentCompactions =
	// options.Experimental  这个是试验性功能
	options.Experimental.L0CompactionConcurrency = 15
	options.Experimental.CompactionDebtConcurrency = 10
	options.Experimental.MaxWriterConcurrency = 10
	//options.MemTableSize
	//options.Experimental.LevelMultiplier
	storage, err := pebble.Open(dataDir, options)
	if err != nil {
		log.Panic("打开dataDir文件错误", zap.Error(err))
		return nil
	}
	//err = storage.RatchetFormatMajorVersion(pebble.FormatVirtualSSTables)
	//if err != nil {
	//	log.Panic("升级dataDir文件错误", zap.Error(err))
	//	return nil
	//}
	sequences.EnablePlugin(ctx)
	impl := &serviceImpl{
		storage:         storage,
		sequenceService: sequences.GetService(),
	}
	return impl
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

func (impl *serviceImpl) Range(startKey, endKey []byte, reverse bool, maxCount int, handler RangeHandler) error {
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
	for ok := first(); ok && count < maxCount; ok = next() {
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
