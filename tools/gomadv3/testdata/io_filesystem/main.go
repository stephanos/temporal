package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"syscall"
	"time"
)

func main() {
	hostEscape := os.Getenv("GOMADV3_HOST_ESCAPE")
	if hostEscape != "" {
		if _, err := os.ReadFile(hostEscape); !errors.Is(err, fs.ErrNotExist) {
			panic(fmt.Sprintf("host escape read error = %v", err))
		}
		fmt.Println("isolated")
		return
	}
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
	if err = os.Chdir("workspace"); err != nil {
		panic(err)
	}
	if workingDirectory, getwdErr := os.Getwd(); getwdErr != nil || workingDirectory != "/workspace" {
		panic(fmt.Sprintf("Getwd = %q, %v", workingDirectory, getwdErr))
	}
	if err = os.Chdir("/"); err != nil {
		panic(err)
	}
	info, err := os.Stat("workspace/.testoutput/nested/.")
	if err != nil {
		panic(err)
	}
	if info.Name() != "nested" || !info.IsDir() || info.Mode().Perm() != 0o755 || info.Size() != 0 || !info.ModTime().Equal(time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)) || info.Sys() != nil {
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
	if err = os.Truncate("workspace/state", 3); err != nil {
		panic(err)
	}
	if err = os.Chmod("workspace/state", 0o600); err != nil {
		panic(err)
	}
	if err = os.Chtimes("workspace/state", time.Unix(1, 2), time.Unix(3, 4)); err != nil {
		panic(err)
	}
	if info, statErr := os.Lstat("workspace/state"); statErr != nil || info.Size() != 3 || info.Mode().Perm() != 0o600 || !info.ModTime().Equal(time.Unix(3, 4)) {
		panic(fmt.Sprintf("Lstat = %#v, %v", info, statErr))
	}
	if err = os.Rename("workspace/state", "workspace/renamed"); err != nil {
		panic(err)
	}
	contents, err = os.ReadFile("workspace/renamed")
	if err != nil || string(contents) != "sta" {
		panic(fmt.Sprintf("ReadFile = %q, %v", contents, err))
	}
	if err = os.Remove("workspace/renamed"); err != nil {
		panic(err)
	}
	if _, err = os.Stat("workspace/renamed"); !errors.Is(err, fs.ErrNotExist) {
		panic(fmt.Sprintf("removed stat error = %v", err))
	}
	if err = os.MkdirAll("workspace/remove/nested", 0o700); err != nil {
		panic(err)
	}
	if err = os.WriteFile("workspace/remove/nested/file", []byte("data"), 0o600); err != nil {
		panic(err)
	}
	if err = os.RemoveAll("workspace/remove"); err != nil {
		panic(err)
	}
	if _, err = os.Stat("workspace/remove"); !errors.Is(err, fs.ErrNotExist) {
		panic(fmt.Sprintf("RemoveAll stat error = %v", err))
	}
	if err = os.Symlink("target", "workspace/link"); !errors.Is(err, syscall.ENOTSUP) {
		panic(fmt.Sprintf("Symlink error = %v", err))
	}
	if _, err = os.Readlink("workspace/link"); !errors.Is(err, syscall.ENOTSUP) {
		panic(fmt.Sprintf("Readlink error = %v", err))
	}
	fmt.Println("ok")
}
