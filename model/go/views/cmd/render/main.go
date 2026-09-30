// Command render writes every generated view into a directory.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"go.temporal.io/server/model/go/views"
)

func main() {
	out := flag.String("out", "", "directory to write the views into")
	flag.Parse()
	if err := run(*out); err != nil {
		fmt.Fprintln(os.Stderr, "render:", err)
		os.Exit(1)
	}
}

func run(out string) error {
	if out == "" {
		return errors.New("-out is required")
	}
	rendered, err := views.All()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	for name, content := range rendered {
		if err := os.WriteFile(filepath.Join(out, name), []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}
