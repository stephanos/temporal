// io_ro_mount_failure echoes a mounted file and exits 2 so its failure
// artifact can be replayed after the host source disappears.
package main

import (
	"fmt"
	"os"
)

func main() {
	schema, err := os.ReadFile("/mounted/schema.sql")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if _, err := os.Stdout.Write(schema); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(2)
}
