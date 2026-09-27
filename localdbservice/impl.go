package localdbservice

import (
	"context"
	stderrors "errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cockroachdb/pebble/v2"
	baseerrors "github.com/coffeehc/base/errors"
	"github.com/coffeehc/base/log"
	"github.com/coffeehc/commons/coder"
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

// Service owns one Pebble database; write durability is selected when opened.
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
	// Close flushes pending data and releases the Pebble directory lock.
	Close() error
}

// PeriodicSyncService writes through Pebble's WAL without waiting for every
// record to reach disk, then synchronizes the WAL at the configured interval.
// Callers must use CommitBatch instead of committing Pebble batches directly.
type PeriodicSyncService interface {
	Service
	// CommitBatch commits one caller-built batch with the service-owned durability policy.
	CommitBatch(batch *pebble.Batch) error
}

// OpenConfig defines how one independently owned Pebble database is opened.
// PebbleOptions may be nil to use DefaultOpenConfig. SyncInterval equal to zero
// keeps Pebble's synchronous write durability; a positive value batches WAL
// synchronization and bounds the possible unsynchronized window. Callers may
// change top-level scalar fields on the supplied Pebble options after
// OpenWithConfig returns because localdbservice clones them before opening the
// database. Reference-valued resources such as a shared Cache retain Pebble's
// normal caller-owned lifecycle.
type OpenConfig struct {
	// PebbleOptions contains caller-selected Pebble tuning. Nil selects the
	// package default, and unspecified Pebble fields receive Pebble defaults.
	PebbleOptions *pebble.Options
	// SyncInterval controls periodic WAL synchronization. Zero uses synchronous
	// writes; a positive duration permits that much unsynchronized WAL time.
	SyncInterval time.Duration
}

func newService(ctx context.Context) Service {
	if err := ctx.Err(); err != nil {
		log.Panic("打开dataDir前上下文已取消", zap.Error(err))
	}
	viper.SetDefault(configKeyForDataDir, "./datas")
	dataDir := viper.GetString(configKeyForDataDir)
	log.Debug("打开数据文件", zap.String("dataDir", dataDir))
	service, err := Open(dataDir)
	if err != nil {
		log.Panic("打开dataDir文件错误", zap.Error(err))
		return nil
	}
	return service
}

// Open opens one explicitly owned Pebble database at dataDir.
// The caller must close the returned service and must not open the same path twice.
func Open(dataDir string) (Service, error) {
	return OpenWithConfig(dataDir, DefaultOpenConfig())
}

// OpenWithPeriodicSync opens one explicitly owned Pebble database whose writes
// become durable at most one sync interval later. Close performs a final sync.
func OpenWithPeriodicSync(dataDir string, syncInterval time.Duration) (PeriodicSyncService, error) {
	if syncInterval <= 0 {
		return nil, fmt.Errorf("local database sync interval must be positive")
	}
	config := DefaultOpenConfig()
	config.SyncInterval = syncInterval
	return OpenWithConfig(dataDir, config)
}

// DefaultOpenConfig returns a fresh general-purpose configuration based on
// Pebble's maintained defaults. It preserves the whole-key comparer split used
// by existing localdbservice databases without imposing workload-specific SST,
// cache, compaction, compression, or Bloom filter policies on every caller.
func DefaultOpenConfig() OpenConfig {
	comparer := *pebble.DefaultComparer
	comparer.Split = splitWholeKey
	options := &pebble.Options{Comparer: &comparer}
	return OpenConfig{PebbleOptions: options}
}

// OpenWithConfig opens one explicitly owned Pebble database using a cloned,
// validated caller configuration. The returned service owns the database and
// must be closed exactly once when the caller is finished with it.
func OpenWithConfig(dataDir string, config OpenConfig) (PeriodicSyncService, error) {
	if strings.TrimSpace(dataDir) == "" {
		return nil, fmt.Errorf("local database data directory is required")
	}
	if config.SyncInterval < 0 {
		return nil, fmt.Errorf("local database sync interval cannot be negative")
	}
	options := config.PebbleOptions
	if options == nil {
		options = DefaultOpenConfig().PebbleOptions
	} else {
		options = options.Clone()
		if options.Comparer == nil {
			options.Comparer = DefaultOpenConfig().PebbleOptions.Comparer
		} else {
			comparer := *options.Comparer
			options.Comparer = &comparer
		}
	}
	options.EnsureDefaults()
	if err := options.Validate(); err != nil {
		return nil, fmt.Errorf("invalid local database Pebble options: %w", err)
	}
	storage, err := pebble.Open(dataDir, options)
	if err != nil {
		return nil, err
	}
	impl := &serviceImpl{storage: storage, writeOptions: pebble.Sync}
	if config.SyncInterval > 0 {
		impl.writeOptions = pebble.NoSync
		impl.syncInterval = config.SyncInterval
		impl.stopSync = make(chan struct{})
		impl.syncStopped = make(chan struct{})
		go impl.runPeriodicSync()
	}
	return impl, nil
}

// splitWholeKey keeps the historical localdbservice comparer behavior: keys
// have no MVCC suffix and must participate in compaction as complete values.
func splitWholeKey([]byte) int { return 0 }

type serviceImpl struct {
	storage      *pebble.DB
	writeOptions *pebble.WriteOptions
	syncInterval time.Duration
	stopSync     chan struct{}
	syncStopped  chan struct{}
	syncErrorMu  sync.RWMutex
	syncError    error
	dirty        atomic.Bool
	closeOnce    sync.Once
	closeErr     error
}

// Start verifies that the plugin startup context is still active.
func (impl *serviceImpl) Start(ctx context.Context) error { return ctx.Err() }

// Stop closes the owned Pebble database exactly once.
func (impl *serviceImpl) Stop(context.Context) error { return impl.Close() }

// Close flushes and releases the owned Pebble database exactly once.
func (impl *serviceImpl) Close() error {
	impl.closeOnce.Do(func() {
		if impl.stopSync != nil {
			close(impl.stopSync)
			<-impl.syncStopped
			impl.closeErr = impl.syncWAL()
		}
		impl.closeErr = stderrors.Join(impl.closeErr, impl.storage.Close())
	})
	return impl.closeErr
}

// runPeriodicSync bounds the unsynchronized WAL window without making each
// foreground write wait for filesystem durability.
func (impl *serviceImpl) runPeriodicSync() {
	ticker := time.NewTicker(impl.syncInterval)
	defer ticker.Stop()
	defer close(impl.syncStopped)
	for {
		select {
		case <-ticker.C:
			_ = impl.syncWAL()
		case <-impl.stopSync:
			return
		}
	}
}

func (impl *serviceImpl) syncWAL() error {
	if !impl.dirty.Swap(false) {
		return impl.currentSyncError()
	}
	err := baseerrors.ConverError(impl.storage.LogData(nil, pebble.Sync))
	if err != nil {
		impl.dirty.Store(true)
	}
	impl.syncErrorMu.Lock()
	impl.syncError = err
	impl.syncErrorMu.Unlock()
	return err
}

func (impl *serviceImpl) currentSyncError() error {
	impl.syncErrorMu.RLock()
	defer impl.syncErrorMu.RUnlock()
	return impl.syncError
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
		return baseerrors.MessageError("存储的Key不合法，或者没有添加前缀")
	}
	if err := impl.currentSyncError(); err != nil {
		return err
	}
	err := impl.storage.Set(key, value, impl.writeOptions)
	if err == nil && impl.writeOptions == pebble.NoSync {
		impl.dirty.Store(true)
	}
	return baseerrors.ConverError(err)
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
		return nil, false, baseerrors.ConverError(err)
	}
	result := make([]byte, len(data))
	copy(result, data)
	return result, true, nil
}

func (impl *serviceImpl) Del(key []byte) error {
	if err := impl.currentSyncError(); err != nil {
		return err
	}
	err := impl.storage.Delete(key, impl.writeOptions)
	if err == nil && impl.writeOptions == pebble.NoSync {
		impl.dirty.Store(true)
	}
	return baseerrors.ConverError(err)
}

func (impl *serviceImpl) DelRange(startKey, endKey []byte) error {
	if err := impl.currentSyncError(); err != nil {
		return err
	}
	err := impl.storage.DeleteRange(startKey, endKey, impl.writeOptions)
	if err == nil && impl.writeOptions == pebble.NoSync {
		impl.dirty.Store(true)
	}
	return baseerrors.ConverError(err)
}

// CommitBatch commits one batch using the durability policy selected at open.
func (impl *serviceImpl) CommitBatch(batch *pebble.Batch) error {
	if batch == nil {
		return fmt.Errorf("local database batch is required")
	}
	if err := impl.currentSyncError(); err != nil {
		return err
	}
	err := batch.Commit(impl.writeOptions)
	if err == nil && impl.writeOptions == pebble.NoSync {
		impl.dirty.Store(true)
	}
	return baseerrors.ConverError(err)
}

func (impl *serviceImpl) GetDB() *pebble.DB {
	return impl.storage
}
