package localdbservice

import (
	"bytes"
	"testing"
	"time"

	"github.com/cockroachdb/pebble/v2"
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

func TestNewPebbleOptionsUsesV2LevelConfiguration(t *testing.T) {
	options := newPebbleOptions()
	defer options.Cache.Unref()
	if options.Comparer == pebble.DefaultComparer {
		t.Fatal("local comparer must not mutate Pebble's global default comparer")
	}
	options.EnsureDefaults()
	if err := options.Validate(); err != nil {
		t.Fatalf("Pebble options should be valid: %v", err)
	}
	wantTargetFileSizes := []int64{4 << 30, 8 << 30, 16 << 30, 32 << 30}
	for level, want := range wantTargetFileSizes {
		if got := options.TargetFileSizes[level]; got != want {
			t.Fatalf("unexpected target file size for L%d: got=%d want=%d", level, got, want)
		}
	}
	if options.Levels[0].Compression() != sstable.NoCompression {
		t.Fatal("L0 should keep no-compression policy")
	}
	if options.Levels[1].Compression() != sstable.NoCompression {
		t.Fatal("L1 should keep no-compression policy")
	}
	if options.Levels[2].Compression() != sstable.SnappyCompression {
		t.Fatal("L2 should keep Snappy compression policy")
	}
	if options.Levels[3].Compression() != sstable.SnappyCompression {
		t.Fatal("later levels should inherit the L2 compression policy")
	}
}
