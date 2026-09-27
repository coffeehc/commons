package localdbservice

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/bloom"
	"github.com/cockroachdb/pebble/v2/sstable"
)

func TestOpenOwnsLifecycleAndPersistsData(t *testing.T) {
	directory := t.TempDir()
	service, err := Open(directory)
	if err != nil {
		t.Fatalf("open local database: %v", err)
	}
	key := []byte("account/polymarket/example/core")
	value := []byte("persisted")
	if err := service.Set(key, value); err != nil {
		t.Fatalf("write local database: %v", err)
	}
	if err := service.Close(); err != nil {
		t.Fatalf("close local database: %v", err)
	}
	if err := service.Close(); err != nil {
		t.Fatalf("close local database twice: %v", err)
	}

	reopened, err := Open(directory)
	if err != nil {
		t.Fatalf("reopen local database: %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	stored, found, err := reopened.Get(key)
	if err != nil {
		t.Fatalf("read reopened local database: %v", err)
	}
	if !found || !bytes.Equal(stored, value) {
		t.Fatalf("unexpected reopened value: found=%v value=%q", found, stored)
	}
}

func TestOpenReadsDatabaseCreatedWithLegacyDefaults(t *testing.T) {
	directory := t.TempDir()
	comparer := *pebble.DefaultComparer
	comparer.Split = splitWholeKey
	legacyOptions := &pebble.Options{
		Comparer:              &comparer,
		BytesPerSync:          32 << 20,
		MaxOpenFiles:          500,
		LBaseMaxBytes:         64 << 20,
		L0CompactionThreshold: 50,
		L0StopWritesThreshold: 200,
	}
	legacyOptions.TargetFileSizes[0] = 4 << 30
	legacyOptions.TargetFileSizes[1] = 8 << 30
	legacyOptions.TargetFileSizes[2] = 16 << 30
	legacyOptions.Levels[0] = pebble.LevelOptions{Compression: noCompression, FilterPolicy: bloom.FilterPolicy(10)}
	legacyOptions.Levels[1] = pebble.LevelOptions{Compression: noCompression, FilterType: pebble.TableFilter, FilterPolicy: bloom.FilterPolicy(5)}
	legacyOptions.Levels[2] = pebble.LevelOptions{Compression: snappyCompression, FilterType: pebble.TableFilter, FilterPolicy: bloom.FilterPolicy(1)}
	legacyStore, err := pebble.Open(directory, legacyOptions)
	if err != nil {
		t.Fatalf("open database with legacy defaults: %v", err)
	}
	key := []byte("account/polymarket/legacy/core")
	value := []byte("legacy")
	if err := legacyStore.Set(key, value, pebble.Sync); err != nil {
		t.Fatalf("write database with legacy defaults: %v", err)
	}
	if err := legacyStore.Flush(); err != nil {
		t.Fatalf("flush database with legacy defaults: %v", err)
	}
	if err := legacyStore.Close(); err != nil {
		t.Fatalf("close database with legacy defaults: %v", err)
	}
	store, err := Open(directory)
	if err != nil {
		t.Fatalf("reopen legacy database with current defaults: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	stored, found, err := store.Get(key)
	if err != nil || !found || !bytes.Equal(stored, value) {
		t.Fatalf("unexpected legacy value: found=%v value=%q err=%v", found, stored, err)
	}
}

func noCompression() *sstable.CompressionProfile { return sstable.NoCompression }

func snappyCompression() *sstable.CompressionProfile { return sstable.SnappyCompression }

func TestOpenWithPeriodicSyncUsesOneDurabilityPolicyForWritesAndBatches(t *testing.T) {
	directory := t.TempDir()
	service, err := OpenWithPeriodicSync(directory, time.Second)
	if err != nil {
		t.Fatalf("open periodic-sync local database: %v", err)
	}
	impl := service.(*serviceImpl)
	if impl.writeOptions != pebble.NoSync || impl.syncInterval != time.Second {
		t.Fatal("periodic-sync database did not select buffered WAL writes")
	}
	if err := service.Set([]byte("account/1/profile"), []byte("profile")); err != nil {
		t.Fatalf("write periodic-sync value: %v", err)
	}
	batch := service.GetDB().NewBatch()
	if err := batch.Set([]byte("account/1/economics"), []byte("economics"), nil); err != nil {
		t.Fatalf("prepare periodic-sync batch: %v", err)
	}
	if err := service.CommitBatch(batch); err != nil {
		t.Fatalf("commit periodic-sync batch: %v", err)
	}
	if err := batch.Close(); err != nil {
		t.Fatalf("close periodic-sync batch: %v", err)
	}
	if err := service.Close(); err != nil {
		t.Fatalf("close periodic-sync local database: %v", err)
	}

	reopened, err := Open(directory)
	if err != nil {
		t.Fatalf("reopen periodic-sync local database: %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	for key, want := range map[string]string{"account/1/profile": "profile", "account/1/economics": "economics"} {
		stored, found, err := reopened.Get([]byte(key))
		if err != nil || !found || string(stored) != want {
			t.Fatalf("unexpected reopened value for %s: found=%v value=%q err=%v", key, found, stored, err)
		}
	}
}

func TestOpenWithPeriodicSyncRejectsInvalidInput(t *testing.T) {
	if _, err := OpenWithPeriodicSync(t.TempDir(), 0); err == nil {
		t.Fatal("zero periodic sync interval was accepted")
	}
	service, err := OpenWithPeriodicSync(t.TempDir(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = service.Close() })
	if err := service.CommitBatch(nil); err == nil {
		t.Fatal("nil periodic-sync batch was accepted")
	}
}

func TestDefaultOpenConfigUsesPebbleDefaults(t *testing.T) {
	config := DefaultOpenConfig()
	options := config.PebbleOptions.Clone()
	if options.Comparer == pebble.DefaultComparer {
		t.Fatal("local comparer must not mutate Pebble's global default comparer")
	}
	options.EnsureDefaults()
	if err := options.Validate(); err != nil {
		t.Fatalf("Pebble options should be valid: %v", err)
	}
	wantTargetFileSizes := []int64{2 << 20, 4 << 20, 8 << 20, 16 << 20}
	for level, want := range wantTargetFileSizes {
		if got := options.TargetFileSizes[level]; got != want {
			t.Fatalf("unexpected target file size for L%d: got=%d want=%d", level, got, want)
		}
	}
	if options.L0CompactionThreshold != 4 {
		t.Fatalf("unexpected L0 compaction threshold: %d", options.L0CompactionThreshold)
	}
	if options.L0StopWritesThreshold != 12 {
		t.Fatalf("unexpected L0 stop-writes threshold: %d", options.L0StopWritesThreshold)
	}
	if options.LBaseMaxBytes != 64<<20 {
		t.Fatalf("unexpected base-level size: %d", options.LBaseMaxBytes)
	}
}

func TestOpenWithConfigUsesCallerOptions(t *testing.T) {
	directory := t.TempDir()
	config := DefaultOpenConfig()
	config.SyncInterval = 25 * time.Millisecond
	config.PebbleOptions.MemTableSize = 8 << 20
	config.PebbleOptions.L0CompactionThreshold = 6
	config.PebbleOptions.L0StopWritesThreshold = 18
	config.PebbleOptions.TargetFileSizes[0] = 64 << 20
	service, err := OpenWithConfig(directory, config)
	if err != nil {
		t.Fatalf("open configured local database: %v", err)
	}
	t.Cleanup(func() { _ = service.Close() })
	impl := service.(*serviceImpl)
	if impl.writeOptions != pebble.NoSync || impl.syncInterval != config.SyncInterval {
		t.Fatal("configured local database did not apply periodic durability")
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("read configured local database directory: %v", err)
	}
	var optionsText string
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "OPTIONS-") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		if err != nil {
			t.Fatalf("read persisted Pebble options: %v", err)
		}
		optionsText = string(data)
		break
	}
	for _, expected := range []string{
		"mem_table_size=8388608",
		"l0_compaction_threshold=6",
		"l0_stop_writes_threshold=18",
		"target_file_size=67108864",
	} {
		if !strings.Contains(optionsText, expected) {
			t.Fatalf("persisted Pebble options do not contain %q", expected)
		}
	}
}

func TestOpenWithConfigRejectsInvalidConfiguration(t *testing.T) {
	config := DefaultOpenConfig()
	config.SyncInterval = -time.Second
	if _, err := OpenWithConfig(t.TempDir(), config); err == nil {
		t.Fatal("negative sync interval was accepted")
	}
	config = DefaultOpenConfig()
	config.PebbleOptions.L0CompactionThreshold = 20
	config.PebbleOptions.L0StopWritesThreshold = 10
	if _, err := OpenWithConfig(t.TempDir(), config); err == nil {
		t.Fatal("invalid Pebble options were accepted")
	}
}
