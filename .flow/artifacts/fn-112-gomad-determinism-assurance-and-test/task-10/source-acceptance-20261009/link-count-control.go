package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	root, err := os.MkdirTemp("", "fn11210-link-control-")
	if err != nil { return err }
	fmt.Println("fixture-root", root)
	shared := filepath.Join(root, "shared")
	if err := os.WriteFile(shared, []byte("fixture"), 0600); err != nil { return err }
	for _, name := range []string{"first", "second"} {
		if err := os.Mkdir(filepath.Join(root, name), 0700); err != nil { return err }
		if err := os.Link(shared, filepath.Join(root, name, "target")); err != nil { return err }
	}
	pinned, err := os.OpenRoot(root)
	if err != nil { return err }
	defer pinned.Close()
	for _, name := range []string{"", "first", "second"} {
		if name != "" {
			if err := pinned.RemoveAll(name); err != nil { return err }
		}
		info, err := os.Lstat(shared)
		if err != nil { return err }
		stat := info.Sys().(*syscall.Stat_t)
		fmt.Printf("removed=%q dev=%d ino=%d nlink=%d\n", name, stat.Dev, stat.Ino, stat.Nlink)
	}
	return nil
}
