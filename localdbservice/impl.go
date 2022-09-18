package localdbservice

import (
	"bytes"
	"context"
	"github.com/cockroachdb/pebble"
	"github.com/coffeehc/base/errors"
	"github.com/coffeehc/base/log"
	"github.com/coffeehc/commons/sequences"
	"github.com/spf13/viper"
	"go.uber.org/zap"
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
}

func newService(ctx context.Context) Service {
	viper.SetDefault(configKeyForDataDir, "./datas")
	dataDir := viper.GetString(configKeyForDataDir)
	comparer := pebble.DefaultComparer
	comparer.Split = func(a []byte) int {
		return bytes.LastIndex(a, Separator)
	}
	log.Debug("打开数据文件", zap.String("dataDir", dataDir))
	options := &pebble.Options{
		Comparer: comparer,
	}
	options.Experimental.MinDeletionRate = 20
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

type serviceImpl struct {
	storage         *pebble.DB
	sequenceService sequences.SequenceService
}

func (impl *serviceImpl) Range(startKey, endKey []byte, reverse bool, maxCount int, handler RangeHandler) error {
	iter := impl.storage.NewIter(nil)
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
