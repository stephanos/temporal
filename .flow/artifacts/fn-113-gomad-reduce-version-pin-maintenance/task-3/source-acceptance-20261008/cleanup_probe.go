package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func main() {
	source := "/Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc/github.com/!masterminds/sprig/v3@v3.2.3"
	parent, err := os.MkdirTemp("", "fn1133-cleanup-probe-")
	if err != nil { panic(err) }
	fmt.Println("probe parent", parent, "source", source, "uid", os.Getuid())
	for _, mode := range []fs.FileMode{0o400, 0o600} {
		for iteration := 0; iteration < 16; iteration++ {
			root := filepath.Join(parent, fmt.Sprintf("mode-%o-%02d", mode, iteration))
			count := 0
			err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
				if walkErr != nil { return walkErr }
				rel, err := filepath.Rel(source, path)
				if err != nil { return err }
				destination := filepath.Join(root, "sprig", rel)
				if entry.IsDir() { return os.MkdirAll(destination, 0o700) }
				bytes, err := os.ReadFile(path)
				if err != nil { return err }
				count++
				return os.WriteFile(destination, bytes, mode)
			})
			if err != nil { panic(err) }
			removeErr := os.RemoveAll(root)
			remaining := []string{}
			walkErr := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
				if err != nil { return err }
				info, err := entry.Info()
				if err != nil { return err }
				remaining = append(remaining, fmt.Sprintf("%s %o %d", path, info.Mode().Perm(), info.Size()))
				return nil
			})
			result := map[string]any{"mode":fmt.Sprintf("%o",mode),"iteration":iteration,"copied_files":count,"remaining":remaining,"remove_error":fmt.Sprint(removeErr),"walk_error":fmt.Sprint(walkErr)}
			encoded, err := json.Marshal(result)
			if err != nil { panic(err) }
			fmt.Println(string(encoded))
		}
	}
}
