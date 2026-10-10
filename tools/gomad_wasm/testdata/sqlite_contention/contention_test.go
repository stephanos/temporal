package contention

import (
	"database/sql"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/vfs/memdb"
)

func TestBusyTimeoutAllowsQueuedLockHolder(t *testing.T) {
	db, err := sql.Open("sqlite3", "file:/contention-diagnostic?vfs=memdb")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	}()
	db.SetMaxOpenConns(2)
	if _, err := db.Exec("CREATE TABLE data (value INTEGER)"); err != nil {
		t.Fatal(err)
	}
	tx, err := db.BeginTx(t.Context(), &sql.TxOptions{Isolation: sql.LevelLinearizable})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec("INSERT INTO data VALUES (1)"); err != nil {
		t.Fatal(err)
	}
	released := make(chan error, 1)
	go func() { released <- tx.Commit() }()
	_, err = db.Exec("INSERT INTO data VALUES (2)")
	releaseErr := <-released
	if releaseErr != nil {
		t.Fatal(releaseErr)
	}
	if err != nil {
		t.Fatalf("contender failed before queued lock holder could release: %v", err)
	}
}
