// io_ro_mount reads the files a read-only mount declared and proves the host
// directory the program runs in stays invisible.
package main

import (
	"errors"
	"fmt"
	"os"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("ok")
}

func run() error {
	schema, err := os.ReadFile("/mounted/schema.sql")
	if err != nil {
		return err
	}
	if string(schema) != "select 1;\n" {
		return fmt.Errorf("schema.sql = %q", schema)
	}
	info, err := os.Stat("/mounted/empty")
	if err != nil {
		return err
	}
	if info.Size() != 0 || info.IsDir() {
		return fmt.Errorf("empty = %d bytes, dir %v", info.Size(), info.IsDir())
	}
	empty, err := os.ReadFile("/mounted/empty")
	if err != nil {
		return err
	}
	if len(empty) != 0 {
		return fmt.Errorf("empty contents = %q", empty)
	}
	for _, path := range []string{"undeclared", "/mounted/undeclared"} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s is visible: %v", path, err)
		}
	}
	return nil
}
