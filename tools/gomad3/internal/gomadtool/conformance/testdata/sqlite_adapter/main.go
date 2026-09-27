// sqlite_adapter runs SQLite through the modernc libc adapter: the sum proves
// the transaction path and the timestamp proves the virtual clock.
package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "modernc.org/sqlite"
)

func main() {
	database, err := sql.Open("sqlite", "file:gomad?mode=memory&cache=private")
	if err != nil {
		fail(err)
	}
	defer database.Close()
	if _, err = database.Exec("CREATE TABLE values_table (value INTEGER NOT NULL)"); err != nil {
		fail(err)
	}
	transaction, err := database.Begin()
	if err != nil {
		fail(err)
	}
	if _, err = transaction.Exec("INSERT INTO values_table VALUES (40), (2)"); err != nil {
		fail(err)
	}
	if err = transaction.Commit(); err != nil {
		fail(err)
	}
	transaction, err = database.Begin()
	if err != nil {
		fail(err)
	}
	if _, err = transaction.Exec("INSERT INTO values_table VALUES (1000)"); err != nil {
		fail(err)
	}
	if err = transaction.Rollback(); err != nil {
		fail(err)
	}
	var sum int
	var currentTime string
	if err = database.QueryRow("SELECT sum(value), current_timestamp FROM values_table").Scan(&sum, &currentTime); err != nil {
		fail(err)
	}
	fmt.Println(sum, currentTime)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
