package sqlxdialect

import (
	"context"
	"errors"
	"testing"
)

type stubTransaction struct {
	commitErr   error
	rollbackErr error
	committed   bool
	rolledBack  bool
}

func (impl *stubTransaction) Commit() error {
	impl.committed = true
	return impl.commitErr
}

func (impl *stubTransaction) Rollback() error {
	impl.rolledBack = true
	return impl.rollbackErr
}

func TestExecuteTransactionReturnsCommitError(t *testing.T) {
	wantErr := errors.New("commit failed")
	transaction := &stubTransaction{commitErr: wantErr}

	err := executeTransaction(context.Background(), transaction, func(context.Context) error { return nil })
	if !errors.Is(err, wantErr) {
		t.Fatalf("executeTransaction() error = %v, want %v", err, wantErr)
	}
	if !transaction.committed || transaction.rolledBack {
		t.Fatalf("transaction state committed=%t rolledBack=%t", transaction.committed, transaction.rolledBack)
	}
}

func TestExecuteTransactionJoinsRollbackError(t *testing.T) {
	handleErr := errors.New("handler failed")
	rollbackErr := errors.New("rollback failed")
	transaction := &stubTransaction{rollbackErr: rollbackErr}

	err := executeTransaction(context.Background(), transaction, func(context.Context) error { return handleErr })
	if !errors.Is(err, handleErr) || !errors.Is(err, rollbackErr) {
		t.Fatalf("executeTransaction() error = %v, want joined handler and rollback errors", err)
	}
	if transaction.committed || !transaction.rolledBack {
		t.Fatalf("transaction state committed=%t rolledBack=%t", transaction.committed, transaction.rolledBack)
	}
}

func TestExecuteTransactionRollsBackOnPanic(t *testing.T) {
	transaction := &stubTransaction{}
	defer func() {
		if recover() == nil {
			t.Fatal("executeTransaction() did not propagate panic")
		}
		if transaction.committed || !transaction.rolledBack {
			t.Fatalf("transaction state committed=%t rolledBack=%t", transaction.committed, transaction.rolledBack)
		}
	}()

	_ = executeTransaction(context.Background(), transaction, func(context.Context) error {
		panic("boom")
	})
}
