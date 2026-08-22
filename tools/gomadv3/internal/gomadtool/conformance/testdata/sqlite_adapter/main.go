package main

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

func main() {
	database, err := sql.Open("sqlite", "file:gomad?mode=memory&cache=private")
	if err != nil {
		panic(err)
	}
	defer database.Close()
	if _, err = database.Exec("CREATE TABLE values_table (value INTEGER NOT NULL)"); err != nil {
		panic(err)
	}
	if _, err = database.Exec("INSERT INTO values_table VALUES (42)"); err != nil {
		panic(err)
	}
	var value int
	var currentTime string
	if err = database.QueryRow("SELECT value, current_timestamp FROM values_table").Scan(&value, &currentTime); err != nil {
		panic(err)
	}
	fmt.Printf("%d %s\n", value, currentTime)
}
