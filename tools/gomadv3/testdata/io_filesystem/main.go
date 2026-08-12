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
	file, err := os.OpenFile("workspace/state", os.O_CREATE|os.O_RDWR|os.O_EXCL, 0o640)
	if err != nil {
		panic(err)
	}
	if written, writeErr := file.Write([]byte("state")); writeErr != nil || written != 5 {
		panic(fmt.Sprintf("write = %d, %v", written, writeErr))
	}
	if _, err = file.Seek(0, 0); err != nil {
		panic(err)
	}
	contents := make([]byte, 5)
	if read, readErr := file.Read(contents); readErr != nil || read != 5 || string(contents) != "state" {
		panic(fmt.Sprintf("read = %d, %q, %v", read, contents, readErr))
	}
	if err = file.Close(); err != nil {
		panic(err)
	}
	if err = os.Rename("workspace/state", "workspace/renamed"); err != nil {
		panic(err)
	}
	contents, err = os.ReadFile("workspace/renamed")
	if err != nil || string(contents) != "state" {
		panic(fmt.Sprintf("ReadFile = %q, %v", contents, err))
	}
	if err = os.Remove("workspace/renamed"); err != nil {
		panic(err)
	}
	if _, err = os.Stat("workspace/renamed"); !errors.Is(err, fs.ErrNotExist) {
		panic(fmt.Sprintf("removed stat error = %v", err))
	}
	fmt.Println("ok")
}
