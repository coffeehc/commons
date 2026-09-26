package localdbservice

import (
	"bytes"
	"testing"

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
