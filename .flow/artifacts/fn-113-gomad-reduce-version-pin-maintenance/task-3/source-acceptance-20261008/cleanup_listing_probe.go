package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func main() {
	source := "/Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc/github.com/!masterminds/sprig/v3@v3.2.3"
	parent, err := os.MkdirTemp("", "fn1133-cleanup-listing-probe-")
	if err != nil { panic(err) }
	fmt.Println("probe parent", parent, "source", source, "uid", os.Getuid())
	for iteration := 0; iteration < 16; iteration++ {
		root := filepath.Join(parent, fmt.Sprintf("case-%02d", iteration))
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
			return os.WriteFile(destination, bytes, 0o400)
		})
		if err != nil { panic(err) }
		for _, platform := range [][2]string{{"darwin","arm64"},{"linux","amd64"}} {
			command := exec.Command(filepath.Join(runtime.GOROOT(),"bin","go"),"list","-e","-find","-json",".")
			command.Dir = filepath.Join(root,"sprig")
			command.Env = append(os.Environ(),"CGO_ENABLED=0","GO111MODULE=off","GOPATH="+parent,"GOOS="+platform[0],"GOARCH="+platform[1])
			output, err := command.CombinedOutput()
			if err != nil { panic(fmt.Sprintf("listing %v %s",err,output)) }
			fmt.Printf("listing %s/%s exited 0 and waited; %d bytes\n",platform[0],platform[1],len(output))
		}
		removeErr := os.RemoveAll(root)
		remaining := []string{}
		walkErr := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil { return err }
			info, err := entry.Info()
			if err != nil { return err }
			remaining = append(remaining, fmt.Sprintf("%s %o %d", path, info.Mode().Perm(), info.Size()))
			return nil
		})
		encoded, err := json.Marshal(map[string]any{"iteration":iteration,"copied_files":count,"remaining":remaining,"remove_error":fmt.Sprint(removeErr),"walk_error":fmt.Sprint(walkErr)})
		if err != nil { panic(err) }
		fmt.Println(string(encoded))
	}
}
