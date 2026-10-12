package contention

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/ncruces/go-sqlite3"
)

func lockedContender(t *testing.T, pragma string) *sql.Conn {
	t.Helper()
	db, err := sql.Open("sqlite3", "file:/"+t.Name()+"?vfs=memdb"+pragma)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})
	db.SetMaxOpenConns(2)
	holder, err := db.Conn(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := holder.Close(); err != nil {
			t.Error(err)
		}
	})
	contender, err := db.Conn(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := contender.Close(); err != nil {
			t.Error(err)
		}
	})
	if _, err := holder.ExecContext(t.Context(), "CREATE TABLE data (value INTEGER)"); err != nil {
		t.Fatal(err)
	}
	tx, err := holder.BeginTx(t.Context(), &sql.TxOptions{Isolation: sql.LevelLinearizable})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := tx.Rollback(); err != nil {
			t.Error(err)
		}
	})
	if _, err := tx.Exec("INSERT INTO data VALUES (1)"); err != nil {
		t.Fatal(err)
	}
	return contender
}

func TestUnreleasedLockPreservesDefaultTimeout(t *testing.T) {
	conn := lockedContender(t, "")
	start := time.Now()
	_, err := conn.ExecContext(t.Context(), "INSERT INTO data VALUES (2)")
	elapsed := time.Since(start)
	if !errors.Is(err, sqlite3.BUSY) || elapsed < time.Minute || elapsed > time.Minute+time.Second {
		t.Fatalf("default busy timeout changed: elapsed=%s error=%v", elapsed, err)
	}
	t.Logf("unreleased lock retained default timeout: %s", elapsed)
}

func TestQueuedCancellationInterruptsBusyRetry(t *testing.T) {
	conn := lockedContender(t, "")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go cancel()
	start := time.Now()
	_, err := conn.ExecContext(ctx, "INSERT INTO data VALUES (2)")
	elapsed := time.Since(start)
	if ctx.Err() != context.Canceled || err == nil || elapsed > 500*time.Millisecond {
		t.Fatalf("queued cancellation did not stop retry before default timeout: elapsed=%s context=%v error=%v", elapsed, ctx.Err(), err)
	}
	t.Logf("queued cancellation retained: %s, %v", elapsed, err)
}

func TestZeroBusyTimeoutPreserved(t *testing.T) {
	conn := lockedContender(t, "&_pragma=busy_timeout(0)")
	start := time.Now()
	_, err := conn.ExecContext(t.Context(), "INSERT INTO data VALUES (2)")
	elapsed := time.Since(start)
	if !errors.Is(err, sqlite3.BUSY) || elapsed > 20*time.Millisecond {
		t.Fatalf("zero busy timeout changed: elapsed=%s error=%v", elapsed, err)
	}
	t.Logf("zero timeout retained: %s", elapsed)
}

func TestCustomBusyTimeoutPreserved(t *testing.T) {
	conn := lockedContender(t, "&_pragma=busy_timeout(20)")
	start := time.Now()
	_, err := conn.ExecContext(t.Context(), "INSERT INTO data VALUES (2)")
	elapsed := time.Since(start)
	if !errors.Is(err, sqlite3.BUSY) || elapsed < 20*time.Millisecond || elapsed > 50*time.Millisecond {
		t.Fatalf("custom busy timeout changed: elapsed=%s error=%v", elapsed, err)
	}
	t.Logf("custom timeout retained: %s", elapsed)
}
