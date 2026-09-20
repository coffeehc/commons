//go:build bootintegration

package dbsource

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/coffeehc/boot/engine"
)

// TestPostgresShutdownProcess isolates actual SIGTERM handling from the test runner.
func TestPostgresShutdownProcess(t *testing.T) {
	if os.Getenv("DBSOURCE_TEST_POSTGRES") == "" {
		t.Skip("set DBSOURCE_TEST_POSTGRES")
	}
	for _, mode := range []string{"active", "idle"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestPostgresShutdownHelper$", "-test.v")
			command.Env = append(os.Environ(), "DBSOURCE_SHUTDOWN_CHILD="+mode)
			command.Dir = t.TempDir()
			output, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("shutdown child failed: %v\n%s", err, output)
			}
			t.Logf("%s SIGTERM: child exit 0", mode)
		})
	}
}

// TestPostgresShutdownHelper runs boot's signal wait with the same cancel-and-join ownership as a worker Stop.
func TestPostgresShutdownHelper(t *testing.T) {
	mode := os.Getenv("DBSOURCE_SHUTDOWN_CHILD")
	if mode == "" {
		return
	}
	service, admin, _ := postgresFixture(t)
	counts := captureDBLogs(t)
	runctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	if mode == "active" {
		query := "SELECT 1 FROM pg_sleep(30) /*dbsource_shutdown*/"
		go func() { var values []int; done <- service.QueryContext(runctx, &values, query) }()
		waitPostgresQuery(t, admin, query)
	}
	// Wait until os/signal has installed its handler; SIGTERM must never reach the parent process.
	// A short delayed self-signal is bounded by the parent process timeout.
	signalErr := make(chan error, 1)
	timer := time.AfterFunc(100*time.Millisecond, func() { signalErr <- syscall.Kill(os.Getpid(), syscall.SIGTERM) })
	defer timer.Stop()
	engine.WaitServiceStop(t.Context(), func() {
		cancel()
		if mode == "active" {
			select {
			case err := <-done:
				if !errors.Is(err, context.Canceled) {
					t.Errorf("query cancellation: %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Error("query did not stop")
			}
		}
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if err := service.Stop(cleanup); err != nil {
			t.Error(err)
		}
	})
	if err := <-signalErr; err != nil {
		t.Fatal(err)
	}
	if got := counts(); got["warn"]+got["error"]+got["dpanic"]+got["panic"]+got["fatal"] != 0 {
		t.Fatalf("shutdown logs: %v", got)
	}
}
