package persistence

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestSQLiteCommitAndRollbackPreserveState(t *testing.T) {
	database, err := sql.Open("sqlite", "file:gomad?mode=memory&cache=private")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := database.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	if _, err = database.Exec("CREATE TABLE values_table (value INTEGER NOT NULL)"); err != nil {
		t.Fatal(err)
	}
	transaction, err := database.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = transaction.Exec("INSERT INTO values_table VALUES (40), (2)"); err != nil {
		t.Fatal(err)
	}
	if err = transaction.Commit(); err != nil {
		t.Fatal(err)
	}
	transaction, err = database.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = transaction.Exec("INSERT INTO values_table VALUES (1000)"); err != nil {
		t.Fatal(err)
	}
	if err = transaction.Rollback(); err != nil {
		t.Fatal(err)
	}
	var count, sum int
	var currentTime string
	if err = database.QueryRow("SELECT count(*), sum(value), current_timestamp FROM values_table").Scan(&count, &sum, &currentTime); err != nil {
		t.Fatal(err)
	}
	if count != 2 || sum != 42 || currentTime != "2000-01-01 00:00:00" {
		t.Fatalf("count=%d sum=%d current_time=%q", count, sum, currentTime)
	}
}

func TestSQLiteWriteAheadLogCommitIsReadable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	database, err := sql.Open("sqlite", "file:"+path+"?_pragma=journal_mode(WAL)")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := database.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	var journalMode string
	if err = database.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatal(err)
	}
	if journalMode != "wal" {
		t.Fatalf("journal_mode=%q", journalMode)
	}
	if _, err = database.Exec("CREATE TABLE values_table (value INTEGER NOT NULL)"); err != nil {
		t.Fatal(err)
	}
	if _, err = database.Exec("INSERT INTO values_table VALUES (40), (2)"); err != nil {
		t.Fatal(err)
	}
	var count, sum int
	if err = database.QueryRow("SELECT count(*), sum(value) FROM values_table").Scan(&count, &sum); err != nil {
		t.Fatal(err)
	}
	if count != 2 || sum != 42 {
		t.Fatalf("count=%d sum=%d", count, sum)
	}
}
