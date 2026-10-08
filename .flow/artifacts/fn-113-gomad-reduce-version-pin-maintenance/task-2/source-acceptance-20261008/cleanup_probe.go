package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func run() error {
	for index := range 32 {
		root, err := os.MkdirTemp("", "fn1132-cleanup-probe-")
		if err != nil { return err }
		work := filepath.Join(root, "work")
		denied := filepath.Join(work, "denied")
		prepared := filepath.Join(work, "adapter")
		for _, directory := range []string{work, denied, prepared} {
			if err := os.Mkdir(directory, 0o700); err != nil { return err }
		}
		leaf := filepath.Join(denied, "leaf")
		if err := os.WriteFile(leaf, []byte("scratch"), 0o600); err != nil { return err }
		if err := os.WriteFile(filepath.Join(prepared, "adapter.go"), []byte("package adapter\n"), 0o600); err != nil { return err }
		info, err := os.Stat(leaf)
		if err != nil { return err }
		if err := os.Chmod(denied, 0); err != nil { return err }
		_, denial := os.ReadDir(denied)
		if !errors.Is(denial, fs.ErrPermission) { return fmt.Errorf("expected real ReadDir EACCES, got %v", denial) }
		if err := os.Rename(prepared, filepath.Join(root, "published")); err != nil { return err }
		entries, err := os.ReadDir(work)
		if err != nil { return err }
		var names []string
		for _, entry := range entries { names = append(names, entry.Name()) }
		cleanup := os.RemoveAll(work)
		_, remainder := os.Stat(work)
		result := map[string]any{"iteration":index,"uid":os.Geteuid(),"root":root,"leaf_size_before_denial":info.Size(),"work_entries_before_remove":names,"actual_read_denial":denial.Error(),"cleanup_error":fmt.Sprint(cleanup),"cleanup_is_permission":errors.Is(cleanup,fs.ErrPermission),"work_after_remove":fmt.Sprint(remainder)}
		if err := json.NewEncoder(os.Stdout).Encode(result); err != nil { return err }
		if err := os.Chmod(denied, 0o700); err != nil && !errors.Is(err, fs.ErrNotExist) { return err }
		if err := os.RemoveAll(root); err != nil { return err }
	}
	return nil
}

func main() {
	if err := run(); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
}
