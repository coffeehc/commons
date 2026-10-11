package localqueue

import (
	"errors"
	"testing"
)

func TestOpenReportsOwnershipAndValidationErrorsImmediately(t *testing.T) {
	dir := t.TempDir()
	if queue, err := Open("../escape", dir); err == nil || queue != nil {
		t.Fatalf("unsafe name: queue=%v err=%v", queue, err)
	}
	first, err := Open("safe", dir)
	if err != nil {
		t.Fatal(err)
	}
	if queue, err := Open("safe", dir); !errors.Is(err, errQueueInUse) || queue != nil {
		t.Fatalf("duplicate owner: queue=%v err=%v", queue, err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	next, err := Open("safe", dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := next.Close(); err != nil {
		t.Fatal(err)
	}
}
