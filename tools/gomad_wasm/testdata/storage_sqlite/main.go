package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ncruces/go-sqlite3"
	sqlitedriver "github.com/ncruces/go-sqlite3/driver"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) != 3 {
		return errors.New("expected operation and journal mode")
	}
	operation, journal := os.Args[1], os.Args[2]
	if journal != "WAL" && journal != "DELETE" {
		return errors.New("unsupported journal mode")
	}
	left, right := make(chan struct{}, 1), make(chan struct{}, 1)
	left <- struct{}{}
	right <- struct{}{}
	select {
	case <-left:
	case <-right:
	}
	if operation == "raw-lock" {
		lock, err := os.OpenFile("/volume/db.lock", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
		if err != nil {
			return err
		}
		if err := lock.Close(); err != nil {
			return err
		}
		directory, err := os.Open("/volume")
		if err != nil {
			return err
		}
		if err := errors.Join(directory.Sync(), directory.Close()); err != nil {
			return err
		}
		fmt.Println("RAW LOCK durable")
		return nil
	}
	db, err := sql.Open("sqlite3", "file:/volume/db?_pragma=busy_timeout(1)")
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA journal_mode=" + journal + "; PRAGMA synchronous=FULL; PRAGMA wal_autocheckpoint=0"); err != nil {
		if operation == "probe-lock" {
			var code sqlite3.ExtendedErrorCode
			if errors.As(err, &code) && (code.Code() == sqlite3.BUSY || code.Code() == sqlite3.LOCKED) {
				fmt.Printf("LOCKED code=%d extended=%d\n", code.Code(), code)
				return db.Close()
			}
		}
		return errors.Join(err, db.Close())
	}
	switch operation {
	case "setup", "setup-live":
		if operation == "setup-live" {
			connection, err := db.Conn(context.Background())
			if err != nil {
				return errors.Join(err, db.Close())
			}
			err = connection.Raw(func(driver any) error {
				_, err := driver.(sqlitedriver.Conn).Raw().FileControl("main", sqlite3.FCNTL_PERSIST_WAL, true)
				return err
			})
			if err := errors.Join(err, connection.Close()); err != nil {
				return errors.Join(err, db.Close())
			}
		}
		if _, err := db.Exec("CREATE TABLE history (sequence INTEGER PRIMARY KEY, value TEXT NOT NULL); INSERT INTO history VALUES (1, 'committed')"); err != nil {
			return errors.Join(err, db.Close())
		}
		fmt.Println("ACK 1")
	case "append", "append-live", "append-crash", "unsynced":
		if operation == "unsynced" {
			if _, err := db.Exec("PRAGMA synchronous=OFF"); err != nil {
				return errors.Join(err, db.Close())
			}
		}
		_, err = db.Exec("BEGIN IMMEDIATE; INSERT INTO history VALUES (2, 'committed'); COMMIT")
		if err != nil {
			connection, connectionErr := db.Conn(context.Background())
			if connectionErr != nil {
				return errors.Join(err, connectionErr, db.Close())
			}
			var autocommit bool
			rollbackErr := connection.Raw(func(driver any) error {
				autocommit = driver.(sqlitedriver.Conn).Raw().GetAutocommit()
				return nil
			})
			if rollbackErr == nil && !autocommit {
				_, rollbackErr = connection.ExecContext(context.Background(), "ROLLBACK")
			}
			if rollbackErr := errors.Join(rollbackErr, connection.Close()); rollbackErr != nil {
				return errors.Join(err, rollbackErr, db.Close())
			}
			var sqliteCode sqlite3.ExtendedErrorCode
			if !errors.As(err, &sqliteCode) {
				return errors.Join(errors.New("storage failure did not become a typed SQLite error"), err, db.Close())
			}
			fmt.Printf("REJECTED 2 code=%d extended=%d\n", sqliteCode.Code(), sqliteCode)
		} else {
			fmt.Println("ACK 2")
		}
		if operation == "append-crash" {
			if err := os.WriteFile("/volume/marker", []byte("volatile after acknowledgement"), 0666); err != nil {
				return errors.Join(err, db.Close())
			}
			trigger, err := os.OpenFile("/volume/crash-trigger", os.O_CREATE|os.O_RDWR, 0666)
			if err != nil {
				return errors.Join(err, db.Close())
			}
			return errors.Join(errors.New("crash trigger did not discard the guest"), trigger.Close(), db.Close())
		}
		if operation == "unsynced" || operation == "append-live" {
			// Exiting preserves the acknowledgement but skips checkpoint and close.
			os.Exit(0)
		}
	case "recover":
		var integrity string
		if err := db.QueryRow("PRAGMA integrity_check").Scan(&integrity); err != nil {
			return errors.Join(err, db.Close())
		}
		if integrity != "ok" {
			return errors.Join(fmt.Errorf("integrity: %s", integrity), db.Close())
		}
		rows, err := db.Query("SELECT sequence, value FROM history ORDER BY sequence")
		if err != nil {
			return errors.Join(err, db.Close())
		}
		var history []string
		for rows.Next() {
			var sequence int
			var value string
			if err := rows.Scan(&sequence, &value); err != nil {
				return errors.Join(err, rows.Close(), db.Close())
			}
			history = append(history, fmt.Sprintf("%d:%s", sequence, value))
		}
		if err := errors.Join(rows.Err(), rows.Close()); err != nil {
			return errors.Join(err, db.Close())
		}
		fmt.Println("INTEGRITY ok")
		fmt.Println("HISTORY " + strings.Join(history, ","))
	default:
		return errors.Join(errors.New("unsupported operation"), db.Close())
	}
	if err := db.Close(); err != nil {
		return err
	}
	if _, err := os.Stat("/volume/db.lock"); !errors.Is(err, os.ErrNotExist) {
		return errors.Join(errors.New("SQLite lock remains before namespace sync"), err)
	}
	if operation == "setup-live" {
		if _, err := os.Stat("/volume/db-wal"); err != nil {
			return err
		}
	}
	// The WASI VFS does not sync the directory when closing the database.
	// Sync after close so dot-lock creation never survives without its cleanup.
	directory, err := os.Open("/volume")
	if err != nil {
		return err
	}
	return errors.Join(directory.Sync(), directory.Close())
}
