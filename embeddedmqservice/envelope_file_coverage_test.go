package embeddedmqservice

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestQueueEnvelopeAtomicWriteReadAndRename(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "nested", "message.json")
	want := queueEnvelope{MessageID: "message-1", Payload: []byte(`{"ok":true}`), CreatedAt: 1, UpdatedAt: 2, AvailableAt: 3}
	if err := writeQueueEnvelope(path, want); err != nil {
		t.Fatalf("writeQueueEnvelope() error = %v", err)
	}
	got, err := readQueueEnvelope(path)
	if err != nil || got.MessageID != want.MessageID || !bytes.Equal(got.Payload, want.Payload) {
		t.Fatalf("readQueueEnvelope() = %+v, %v", got, err)
	}
	payload, err := os.ReadFile(path)
	if err != nil || len(payload) == 0 || payload[len(payload)-1] != '\n' {
		t.Fatalf("queue envelope bytes = %q, %v", payload, err)
	}
	otherDir := filepath.Join(directory, "other")
	source := filepath.Join(directory, "source.tmp")
	if err = os.WriteFile(source, []byte("move"), 0o600); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(otherDir, "target.json")
	if err = renameQueueFile(source, target); err != nil {
		t.Fatalf("renameQueueFile() error = %v", err)
	}
	if _, err = os.Stat(target); err != nil {
		t.Fatalf("renamed target missing: %v", err)
	}
	blocked := filepath.Join(directory, "blocked")
	if err = os.WriteFile(blocked, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err = writeQueueEnvelope(filepath.Join(blocked, "message.json"), queueEnvelope{}); err == nil {
		t.Fatal("non-directory parent should fail")
	}
}
