package localqueue

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestQueuePutReadAndMessageBounds(t *testing.T) {
	queueDir := t.TempDir()
	queue := NewQueue("basic", queueDir)
	t.Cleanup(func() {
		_ = queue.Delete()
	})

	if err := queue.Put([]byte("123")); err == nil {
		t.Fatal("Put() accepted a message smaller than the compatibility minimum")
	}
	if err := queue.Put(make([]byte, defaultMaxMessageSize+1)); err == nil {
		t.Fatal("Put() accepted a message larger than the compatibility maximum")
	}

	want := [][]byte{[]byte("first"), []byte("second")}
	for _, message := range want {
		if err := queue.Put(message); err != nil {
			t.Fatalf("Put(%q) error = %v", message, err)
		}
	}
	if depth := queue.Depth(); depth != int64(len(want)) {
		t.Fatalf("Depth() = %d, want %d", depth, len(want))
	}
	for _, expected := range want {
		if message := receiveMessage(t, queue); string(message) != string(expected) {
			t.Fatalf("ReadChan() = %q, want %q", message, expected)
		}
	}
	waitForDepth(t, queue, 0)
}

func TestQueuePersistsUnreadMessagesAcrossReopen(t *testing.T) {
	queueDir := t.TempDir()
	first := NewQueue("persistent", queueDir)
	for _, message := range [][]byte{[]byte("first"), []byte("second")} {
		if err := first.Put(message); err != nil {
			t.Fatalf("Put(%q) error = %v", message, err)
		}
	}
	if err := first.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	assertReadChannelClosed(t, first)

	second := NewQueue("persistent", queueDir)
	t.Cleanup(func() {
		_ = second.Delete()
	})
	if depth := second.Depth(); depth != 2 {
		t.Fatalf("reopened Depth() = %d, want 2", depth)
	}
	for _, expected := range []string{"first", "second"} {
		if message := receiveMessage(t, second); string(message) != expected {
			t.Fatalf("reopened ReadChan() = %q, want %q", message, expected)
		}
	}
	waitForDepth(t, second, 0)
}

func TestQueueReadsGoDiskQueueFiles(t *testing.T) {
	queueDir := t.TempDir()
	queueName := "legacy"
	segmentPath := filepath.Join(queueDir, queueName+".diskqueue.000000.dat")
	writePosition := writeFrames(t, segmentPath, []byte("first"), []byte("second"))
	writeMetadata(t, queueDir, queueName, 2, 0, 0, 0, writePosition)

	queue := NewQueue(queueName, queueDir)
	t.Cleanup(func() {
		_ = queue.Delete()
	})
	if depth := queue.Depth(); depth != 2 {
		t.Fatalf("Depth() = %d, want 2", depth)
	}
	for _, expected := range []string{"first", "second"} {
		if message := receiveMessage(t, queue); string(message) != expected {
			t.Fatalf("ReadChan() = %q, want %q", message, expected)
		}
	}
}

func TestQueueRecoversFramesAfterStaleMetadata(t *testing.T) {
	queueDir := t.TempDir()
	queueName := "stale"
	segmentPath := filepath.Join(queueDir, queueName+".diskqueue.000000.dat")
	firstPosition := int64(4 + len("first"))
	writeFrames(t, segmentPath, []byte("first"), []byte("second"))
	writeMetadata(t, queueDir, queueName, 1, 0, 0, 0, firstPosition)

	queue := NewQueue(queueName, queueDir)
	t.Cleanup(func() {
		_ = queue.Delete()
	})
	if depth := queue.Depth(); depth != 2 {
		t.Fatalf("recovered Depth() = %d, want 2", depth)
	}
	for _, expected := range []string{"first", "second"} {
		if message := receiveMessage(t, queue); string(message) != expected {
			t.Fatalf("recovered ReadChan() = %q, want %q", message, expected)
		}
	}
}

func TestQueueTruncatesIncompleteSegmentTail(t *testing.T) {
	queueDir := t.TempDir()
	queueName := "partial"
	segmentPath := filepath.Join(queueDir, queueName+".diskqueue.000000.dat")
	validSize := writeFrames(t, segmentPath, []byte("valid"))
	file, err := os.OpenFile(segmentPath, os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	partial := make([]byte, 5)
	binary.BigEndian.PutUint32(partial[:4], 8)
	partial[4] = 'x'
	if _, err = file.Write(partial); err != nil {
		_ = file.Close()
		t.Fatalf("Write() error = %v", err)
	}
	if err = file.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	queue := NewQueue(queueName, queueDir)
	t.Cleanup(func() {
		_ = queue.Delete()
	})
	if depth := queue.Depth(); depth != 1 {
		t.Fatalf("recovered Depth() = %d, want 1", depth)
	}
	info, err := os.Stat(segmentPath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Size() != validSize {
		t.Fatalf("repaired segment size = %d, want %d", info.Size(), validSize)
	}
	if message := receiveMessage(t, queue); string(message) != "valid" {
		t.Fatalf("ReadChan() = %q, want valid", message)
	}
}

func TestQueueQuarantinesRuntimeCorruptionAndContinues(t *testing.T) {
	queueDir := t.TempDir()
	queue := openTestQueue(t, "corrupt", queueDir, testQueueConfig())
	t.Cleanup(func() {
		_ = queue.Delete()
	})
	if err := queue.Put([]byte("first")); err != nil {
		t.Fatalf("Put(first) error = %v", err)
	}
	if err := queue.Put([]byte("second")); err != nil {
		t.Fatalf("Put(second) error = %v", err)
	}

	file, err := os.OpenFile(queue.segmentPath(0), os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	invalidHeader := make([]byte, 4)
	if _, err = file.WriteAt(invalidHeader, int64(4+len("first"))); err != nil {
		_ = file.Close()
		t.Fatalf("WriteAt() error = %v", err)
	}
	if err = file.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if message := receiveMessage(t, queue); string(message) != "first" {
		t.Fatalf("ReadChan() = %q, want first", message)
	}
	waitForDepth(t, queue, 0)
	waitForFile(t, queue.segmentPath(0)+".bad")

	if err = queue.Put([]byte("third")); err != nil {
		t.Fatalf("Put(third) after corruption error = %v", err)
	}
	if message := receiveMessage(t, queue); string(message) != "third" {
		t.Fatalf("ReadChan() after corruption = %q, want third", message)
	}
}

func TestQueueRollsSegments(t *testing.T) {
	queueDir := t.TempDir()
	config := testQueueConfig()
	config.maxBytesPerFile = 12
	queue := openTestQueue(t, "rolling", queueDir, config)
	t.Cleanup(func() {
		_ = queue.Delete()
	})

	for _, message := range []string{"first", "other", "third"} {
		if err := queue.Put([]byte(message)); err != nil {
			t.Fatalf("Put(%q) error = %v", message, err)
		}
	}
	segments, err := queue.segmentNumbers()
	if err != nil {
		t.Fatalf("segmentNumbers() error = %v", err)
	}
	if len(segments) != 3 {
		t.Fatalf("segment count = %d, want 3: %v", len(segments), segments)
	}
	for _, expected := range []string{"first", "other", "third"} {
		if message := receiveMessage(t, queue); string(message) != expected {
			t.Fatalf("ReadChan() = %q, want %q", message, expected)
		}
	}
}

func TestQueueEmptyKeepsQueueUsable(t *testing.T) {
	queueDir := t.TempDir()
	first := NewQueue("empty", queueDir)
	t.Cleanup(func() {
		_ = first.Delete()
	})
	for _, message := range []string{"first", "second"} {
		if err := first.Put([]byte(message)); err != nil {
			t.Fatalf("Put(%q) error = %v", message, err)
		}
	}
	if err := first.Empty(); err != nil {
		t.Fatalf("Empty() error = %v", err)
	}
	if depth := first.Depth(); depth != 0 {
		t.Fatalf("Depth() after Empty() = %d, want 0", depth)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("Close() after Empty() error = %v", err)
	}

	second := NewQueue("empty", queueDir)
	t.Cleanup(func() {
		_ = second.Delete()
	})
	if depth := second.Depth(); depth != 0 {
		t.Fatalf("reopened Depth() after Empty() = %d, want 0", depth)
	}
	if err := second.Put([]byte("after")); err != nil {
		t.Fatalf("Put() after Empty() error = %v", err)
	}
	if message := receiveMessage(t, second); string(message) != "after" {
		t.Fatalf("ReadChan() = %q, want after", message)
	}
}

func TestQueueCloseAndDeleteAreIdempotent(t *testing.T) {
	queueDir := t.TempDir()
	queue := NewQueue("lifecycle", queueDir)
	if err := queue.Put([]byte("value")); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	if err := queue.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := queue.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
	if err := queue.Put([]byte("after")); !errors.Is(err, errQueueClosed) {
		t.Fatalf("Put() after Close() error = %v, want %v", err, errQueueClosed)
	}
	if err := queue.Empty(); !errors.Is(err, errQueueClosed) {
		t.Fatalf("Empty() after Close() error = %v, want %v", err, errQueueClosed)
	}
	assertReadChannelClosed(t, queue)

	if err := queue.Delete(); err != nil {
		t.Fatalf("Delete() after Close() error = %v", err)
	}
	if err := queue.Delete(); err != nil {
		t.Fatalf("second Delete() error = %v", err)
	}
	entries, err := os.ReadDir(queueDir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".dat" {
			t.Fatalf("Delete() left queue file %q", entry.Name())
		}
	}
}

func TestQueueDeleteDoesNotRemovePrefixRelatedQueue(t *testing.T) {
	queueDir := t.TempDir()
	primary := NewQueue("orders", queueDir)
	secondary := NewQueue("orders.diskqueue.archive", queueDir)
	t.Cleanup(func() {
		_ = primary.Delete()
		_ = secondary.Delete()
	})
	if err := primary.Put([]byte("primary")); err != nil {
		t.Fatalf("primary.Put() error = %v", err)
	}
	if err := secondary.Put([]byte("secondary")); err != nil {
		t.Fatalf("secondary.Put() error = %v", err)
	}
	if err := primary.Delete(); err != nil {
		t.Fatalf("primary.Delete() error = %v", err)
	}
	if message := receiveMessage(t, secondary); string(message) != "secondary" {
		t.Fatalf("secondary.ReadChan() = %q, want secondary", message)
	}
	if err := secondary.Put([]byte("after")); err != nil {
		t.Fatalf("secondary.Put() after primary deletion error = %v", err)
	}
	if message := receiveMessage(t, secondary); string(message) != "after" {
		t.Fatalf("secondary.ReadChan() = %q, want after", message)
	}
}

func TestQueueRejectsDuplicateOwnerAndProtectsReopenedQueue(t *testing.T) {
	queueDir := t.TempDir()
	first := NewQueue("owned", queueDir)
	t.Cleanup(func() {
		_ = first.Delete()
	})
	duplicate := NewQueue("owned", queueDir)
	if err := duplicate.Put([]byte("duplicate")); !errors.Is(err, errQueueInUse) {
		t.Fatalf("duplicate.Put() error = %v, want %v", err, errQueueInUse)
	}
	assertReadChannelClosed(t, duplicate)

	if err := first.Close(); err != nil {
		t.Fatalf("first.Close() error = %v", err)
	}
	reopened := NewQueue("owned", queueDir)
	t.Cleanup(func() {
		_ = reopened.Delete()
	})
	if err := reopened.Put([]byte("reopened")); err != nil {
		t.Fatalf("reopened.Put() error = %v", err)
	}
	if err := first.Delete(); !errors.Is(err, errQueueInUse) {
		t.Fatalf("stale first.Delete() error = %v, want %v", err, errQueueInUse)
	}
	if message := receiveMessage(t, reopened); string(message) != "reopened" {
		t.Fatalf("reopened.ReadChan() = %q, want reopened", message)
	}
	if err := reopened.Close(); err != nil {
		t.Fatalf("reopened.Close() error = %v", err)
	}
	if err := first.Delete(); err != nil {
		t.Fatalf("first.Delete() after reopened Close() error = %v", err)
	}
}

func TestQueueReleasesOwnershipWhenCloseFails(t *testing.T) {
	queueDir := t.TempDir()
	first := NewQueue("close-error", queueDir)
	if err := first.Put([]byte("first")); err != nil {
		t.Fatalf("first.Put() error = %v", err)
	}
	metadataPath := filepath.Join(queueDir, "close-error"+metadataFileSuffix)
	if err := os.Mkdir(metadataPath, 0o700); err != nil {
		t.Fatalf("Mkdir(metadata path) error = %v", err)
	}
	if err := first.Close(); err == nil {
		t.Fatal("first.Close() error = nil, want metadata persistence error")
	}
	if err := os.Remove(metadataPath); err != nil {
		t.Fatalf("Remove(metadata path) error = %v", err)
	}

	reopened := NewQueue("close-error", queueDir)
	if err := reopened.Delete(); err != nil {
		t.Fatalf("reopened.Delete() error = %v", err)
	}
}

func TestQueueOwnershipAcrossProcesses(t *testing.T) {
	const helperEnvironment = "LOCALQUEUE_OWNERSHIP_HELPER"
	if os.Getenv(helperEnvironment) == "1" {
		runQueueOwnershipHelper(t)
		return
	}

	queueDir := t.TempDir()
	readyPath := filepath.Join(queueDir, "helper.ready")
	releasePath := filepath.Join(queueDir, "helper.release")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestQueueOwnershipAcrossProcesses$")
	command.Env = append(
		os.Environ(),
		helperEnvironment+"=1",
		"LOCALQUEUE_OWNERSHIP_DIR="+queueDir,
		"LOCALQUEUE_OWNERSHIP_READY="+readyPath,
		"LOCALQUEUE_OWNERSHIP_RELEASE="+releasePath,
	)
	var childOutput bytes.Buffer
	command.Stdout = &childOutput
	command.Stderr = &childOutput
	if err := command.Start(); err != nil {
		t.Fatalf("helper Start() error = %v", err)
	}
	childRunning := true
	defer func() {
		if !childRunning {
			return
		}
		_ = os.WriteFile(releasePath, []byte("release"), 0o600)
		_ = command.Process.Kill()
		_ = command.Wait()
	}()
	waitForFile(t, readyPath)

	blocked := NewQueue("cross-process", queueDir)
	if err := blocked.Put([]byte("blocked")); !errors.Is(err, errQueueInUse) {
		t.Fatalf("cross-process Put() error = %v, want %v", err, errQueueInUse)
	}
	if err := os.WriteFile(releasePath, []byte("release"), 0o600); err != nil {
		t.Fatalf("WriteFile(release) error = %v", err)
	}
	if err := command.Wait(); err != nil {
		t.Fatalf("helper Wait() error = %v, output = %s", err, childOutput.String())
	}
	childRunning = false

	reopened := NewQueue("cross-process", queueDir)
	if err := reopened.Delete(); err != nil {
		t.Fatalf("Delete() after helper exit error = %v", err)
	}
}

func runQueueOwnershipHelper(t *testing.T) {
	t.Helper()
	queueDir := os.Getenv("LOCALQUEUE_OWNERSHIP_DIR")
	readyPath := os.Getenv("LOCALQUEUE_OWNERSHIP_READY")
	releasePath := os.Getenv("LOCALQUEUE_OWNERSHIP_RELEASE")
	queue := NewQueue("cross-process", queueDir)
	if err := queue.Put([]byte("held")); err != nil {
		t.Fatalf("helper Put() error = %v", err)
	}
	if err := os.WriteFile(readyPath, []byte("ready"), 0o600); err != nil {
		t.Fatalf("helper WriteFile(ready) error = %v", err)
	}
	waitForFile(t, releasePath)
	if err := queue.Close(); err != nil {
		t.Fatalf("helper Close() error = %v", err)
	}
}

func TestQueueSupportsConcurrentProducersAndConsumers(t *testing.T) {
	queueDir := t.TempDir()
	queue := openTestQueue(t, "concurrent", queueDir, testQueueConfig())
	t.Cleanup(func() {
		_ = queue.Delete()
	})

	const producerCount = 8
	const messagesPerProducer = 100
	const totalMessages = producerCount * messagesPerProducer
	received := make(chan string, totalMessages)
	go func() {
		for range totalMessages {
			received <- string(<-queue.ReadChan())
		}
	}()

	var producers sync.WaitGroup
	errorsChannel := make(chan error, producerCount)
	for producer := range producerCount {
		producers.Add(1)
		go func() {
			defer producers.Done()
			for sequence := range messagesPerProducer {
				message := fmt.Sprintf("p%02d-%03d", producer, sequence)
				if err := queue.Put([]byte(message)); err != nil {
					errorsChannel <- fmt.Errorf("Put(%q): %w", message, err)
					return
				}
			}
		}()
	}
	producers.Wait()
	close(errorsChannel)
	for err := range errorsChannel {
		t.Fatal(err)
	}

	seen := make(map[string]struct{}, totalMessages)
	deadline := time.After(10 * time.Second)
	for range totalMessages {
		select {
		case message := <-received:
			if _, exists := seen[message]; exists {
				t.Fatalf("received duplicate message %q", message)
			}
			seen[message] = struct{}{}
		case <-deadline:
			t.Fatalf("received %d/%d messages before timeout", len(seen), totalMessages)
		}
	}
	waitForDepth(t, queue, 0)
}

func TestQueueRejectsUnsafeName(t *testing.T) {
	queueDir := t.TempDir()
	queue := NewQueue("../escape", queueDir)
	if err := queue.Put([]byte("value")); err == nil {
		t.Fatal("Put() error = nil for unsafe queue name")
	}
	if err := queue.Close(); err == nil {
		t.Fatal("Close() error = nil for unsafe queue name")
	}
	assertReadChannelClosed(t, queue)
}

func testQueueConfig() queueConfig {
	return queueConfig{
		maxBytesPerFile: 1 << 20,
		minMessageSize:  1,
		maxMessageSize:  1024,
		syncEvery:       10_000,
		syncTimeout:     time.Hour,
	}
}

func openTestQueue(t *testing.T, name, queueDir string, config queueConfig) *diskQueue {
	t.Helper()
	queue, ok := newDiskQueue(name, queueDir, config).(*diskQueue)
	if !ok {
		t.Fatal("newDiskQueue() did not return *diskQueue")
	}
	if queue.startErr != nil {
		t.Fatalf("newDiskQueue() error = %v", queue.startErr)
	}
	return queue
}

func receiveMessage(t *testing.T, queue Interface) []byte {
	t.Helper()
	select {
	case message, ok := <-queue.ReadChan():
		if !ok {
			t.Fatal("ReadChan() closed before delivering a message")
		}
		return message
	case <-time.After(5 * time.Second):
		t.Fatal("ReadChan() did not deliver a message before timeout")
		return nil
	}
}

func waitForDepth(t *testing.T, queue Interface, expected int64) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if queue.Depth() == expected {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("Depth() = %d, want %d", queue.Depth(), expected)
}

func waitForFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("file %q did not appear before timeout", path)
}

func assertReadChannelClosed(t *testing.T, queue Interface) {
	t.Helper()
	select {
	case _, ok := <-queue.ReadChan():
		if ok {
			t.Fatal("ReadChan() remained open")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("ReadChan() did not close before timeout")
	}
}

func writeFrames(t *testing.T, path string, messages ...[]byte) int64 {
	t.Helper()
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	position := int64(0)
	for _, message := range messages {
		header := make([]byte, 4)
		binary.BigEndian.PutUint32(header, uint32(len(message)))
		if _, err = file.Write(header); err == nil {
			_, err = file.Write(message)
		}
		if err != nil {
			_ = file.Close()
			t.Fatalf("Write() error = %v", err)
		}
		position += int64(len(header) + len(message))
	}
	if err = file.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	return position
}

func writeMetadata(
	t *testing.T,
	queueDir string,
	queueName string,
	depth int64,
	readFileNum int64,
	readPosition int64,
	writeFileNum int64,
	writePosition int64,
) {
	t.Helper()
	metadata := fmt.Sprintf(
		"%d\n%d,%d\n%d,%d\n",
		depth,
		readFileNum,
		readPosition,
		writeFileNum,
		writePosition,
	)
	path := filepath.Join(queueDir, queueName+metadataFileSuffix)
	if err := os.WriteFile(path, []byte(metadata), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}
