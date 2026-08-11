package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"
)

func main() {
	hostname, err := os.Hostname()
	if err != nil || hostname != "gomad-host" {
		panic(fmt.Sprintf("hostname = %q, %v", hostname, err))
	}
	if _, err = os.Stat("workspace/missing"); !errors.Is(err, fs.ErrNotExist) {
		panic(fmt.Sprintf("missing stat error = %v", err))
	}
	if err = os.Mkdir("workspace", 0o750); err != nil {
		panic(err)
	}
	if err = os.MkdirAll("workspace/.testoutput/nested", 0o755); err != nil {
		panic(err)
	}
	if err = os.MkdirAll("workspace/.testoutput/nested", 0o700); err != nil {
		panic(err)
	}
	info, err := os.Stat("workspace/.testoutput/nested/.")
	if err != nil {
		panic(err)
	}
	if info.Name() != "nested" || !info.IsDir() || info.Mode().Perm() != 0o755 || info.Size() != 0 || !info.ModTime().Equal(time.Unix(0, 0)) || info.Sys() != nil {
		panic(fmt.Sprintf("directory info = %#v", info))
	}
	if err = os.Mkdir("workspace", 0o755); !errors.Is(err, fs.ErrExist) {
		panic(fmt.Sprintf("duplicate mkdir error = %v", err))
	}
	if err = os.MkdirAll("../escape", 0o755); err == nil {
		panic("path traversal was accepted")
	}
	fmt.Println("ok")
}
