package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
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
	if _, err = os.ReadFile("/dev/null"); !errors.Is(err, fs.ErrNotExist) {
		panic(fmt.Sprintf("device read error = %v", err))
	}
	if os.TempDir() != "/tmp" {
		panic(fmt.Sprintf("TempDir = %q", os.TempDir()))
	}
	if home, homeErr := os.UserHomeDir(); homeErr == nil || home != "" {
		panic(fmt.Sprintf("UserHomeDir = %q, %v", home, homeErr))
	}
	if cache, cacheErr := os.UserCacheDir(); cacheErr == nil || cache != "" {
		panic(fmt.Sprintf("UserCacheDir = %q, %v", cache, cacheErr))
	}
	if config, configErr := os.UserConfigDir(); configErr == nil || config != "" {
		panic(fmt.Sprintf("UserConfigDir = %q, %v", config, configErr))
	}
	if err = os.MkdirAll(os.TempDir(), 0o700); err != nil {
		panic(err)
	}
	temporary, err := os.MkdirTemp("", "gomad-")
	if err != nil || !strings.HasPrefix(temporary, "/tmp/gomad-") {
		panic(fmt.Sprintf("MkdirTemp = %q, %v", temporary, err))
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
	directory, err := os.Open("workspace")
	if err != nil {
		panic(err)
	}
	if err = directory.Chdir(); err != nil {
		panic(err)
	}
	if err = os.Chdir("/"); err != nil {
		panic(err)
	}
	if err = directory.Close(); err != nil {
		panic(err)
	}
	if entries, readErr := os.ReadDir("workspace"); readErr != nil || len(entries) == 0 {
		panic(fmt.Sprintf("ReadDir = %v, %v", entries, readErr))
	}
	directory, err = os.Open("workspace")
	if err != nil {
		panic(err)
	}
	if entries, readErr := directory.ReadDir(-1); readErr != nil || len(entries) == 0 {
		panic(fmt.Sprintf("File.ReadDir = %v, %v", entries, readErr))
	}
	if err = directory.Close(); err != nil {
		panic(err)
	}
	directory, err = os.Open("workspace")
	if err != nil {
		panic(err)
	}
	if entries, readErr := directory.Readdir(-1); readErr != nil || len(entries) == 0 {
		panic(fmt.Sprintf("File.Readdir = %v, %v", entries, readErr))
	}
	if err = directory.Close(); err != nil {
		panic(err)
	}
	directory, err = os.Open("workspace")
	if err != nil {
		panic(err)
	}
	if names, readErr := directory.Readdirnames(-1); readErr != nil || len(names) == 0 {
		panic(fmt.Sprintf("File.Readdirnames = %v, %v", names, readErr))
	}
	if err = directory.Close(); err != nil {
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
	if written, writeErr := file.WriteAt([]byte("s"), 0); writeErr != nil || written != 1 {
		panic(fmt.Sprintf("writeat = %d, %v", written, writeErr))
	}
	if read, readErr := file.ReadAt(make([]byte, 1), 0); readErr != nil || read != 1 {
		panic(fmt.Sprintf("readat = %d, %v", read, readErr))
	}
	if err = file.Sync(); err != nil {
		panic(err)
	}
	if err = file.Chmod(0o640); err != nil {
		panic(err)
	}
	if err = file.Chown(0, 0); !isUnsupported(err) {
		panic(fmt.Sprintf("File.Chown = %v", err))
	}
	if descriptor := file.Fd(); descriptor != ^uintptr(0) {
		panic(fmt.Sprintf("Fd = %d", descriptor))
	}
	if raw, rawErr := file.SyscallConn(); raw != nil || !isUnsupported(rawErr) {
		panic(fmt.Sprintf("SyscallConn = %v, %v", raw, rawErr))
	}
	if _, err = file.Seek(0, 0); err != nil {
		panic(err)
	}
	if copied, copyErr := file.ReadFrom(strings.NewReader("more")); copied != 0 || !isUnsupported(copyErr) {
		panic(fmt.Sprintf("ReadFrom = %d, %v", copied, copyErr))
	}
	if copied, copyErr := file.WriteTo(io.Discard); copied != 0 || !isUnsupported(copyErr) {
		panic(fmt.Sprintf("WriteTo = %d, %v", copied, copyErr))
	}
	if err = file.SetDeadline(time.Time{}); !isUnsupported(err) {
		panic(fmt.Sprintf("SetDeadline = %v", err))
	}
	if err = file.SetReadDeadline(time.Time{}); !isUnsupported(err) {
		panic(fmt.Sprintf("SetReadDeadline = %v", err))
	}
	if err = file.SetWriteDeadline(time.Time{}); !isUnsupported(err) {
		panic(fmt.Sprintf("SetWriteDeadline = %v", err))
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
	if err = os.Symlink("target", "workspace/link"); !isUnsupported(err) {
		panic(fmt.Sprintf("Symlink error = %v", err))
	}
	if err = os.Link("workspace/state", "workspace/hard-link"); !isUnsupported(err) {
		panic(fmt.Sprintf("Link error = %v", err))
	}
	if err = os.Chown("workspace/state", 0, 0); !isUnsupported(err) {
		panic(fmt.Sprintf("Chown error = %v", err))
	}
	if err = os.Lchown("workspace/state", 0, 0); !isUnsupported(err) {
		panic(fmt.Sprintf("Lchown error = %v", err))
	}
	if _, err = os.Readlink("workspace/link"); !isUnsupported(err) {
		panic(fmt.Sprintf("Readlink error = %v", err))
	}
	if root, rootErr := os.OpenRoot("workspace"); root != nil || !isUnsupported(rootErr) {
		panic(fmt.Sprintf("OpenRoot = %v, %v", root, rootErr))
	}
	if rooted, rootErr := os.OpenInRoot("workspace", "file"); rooted != nil || !isUnsupported(rootErr) {
		panic(fmt.Sprintf("OpenInRoot = %v, %v", rooted, rootErr))
	}
	if os.Getpid() != 1 || os.Getppid() != 0 || os.Getuid() != 0 || os.Geteuid() != 0 || os.Getgid() != 0 || os.Getegid() != 0 {
		panic("process identity reached the host")
	}
	if groups, groupsErr := os.Getgroups(); groupsErr != nil || len(groups) != 1 || groups[0] != 0 {
		panic(fmt.Sprintf("Getgroups = %v, %v", groups, groupsErr))
	}
	if executable, executableErr := os.Executable(); executableErr != nil || executable != "/gomadv3-target" {
		panic(fmt.Sprintf("Executable = %q, %v", executable, executableErr))
	}
	if process, processErr := os.StartProcess("/gomadv3-denied-process", []string{"denied"}, nil); process != nil || !isUnsupported(processErr) {
		panic(fmt.Sprintf("StartProcess = %v, %v", process, processErr))
	}
	if process, processErr := os.FindProcess(2_147_483_647); process != nil || !isUnsupported(processErr) {
		panic(fmt.Sprintf("FindProcess = %v, %v", process, processErr))
	}
	process := &os.Process{Pid: 2_147_483_647}
	if err = process.Kill(); !isUnsupported(err) {
		panic(fmt.Sprintf("Process.Kill = %v", err))
	}
	if state, waitErr := process.Wait(); state != nil || !isUnsupported(waitErr) {
		panic(fmt.Sprintf("Process.Wait = %v, %v", state, waitErr))
	}
	if err = process.Signal(os.Interrupt); !isUnsupported(err) {
		panic(fmt.Sprintf("Process.Signal = %v", err))
	}
	if err = process.WithHandle(func(uintptr) {}); !isUnsupported(err) {
		panic(fmt.Sprintf("Process.WithHandle = %v", err))
	}
	if wrapped := os.NewFile(42, "host"); wrapped != nil {
		panic("NewFile exposed a host descriptor")
	}
	if raw, rawErr := os.Stdout.SyscallConn(); raw != nil || !isUnsupported(rawErr) {
		panic(fmt.Sprintf("stdout SyscallConn = %v, %v", raw, rawErr))
	}
	if read, write, pipeErr := os.Pipe(); !isUnsupported(pipeErr) || read != nil || write != nil {
		panic(fmt.Sprintf("Pipe = %v, %v, %v", read, write, pipeErr))
	}
	fmt.Println("ok")
}

func isUnsupported(err error) bool {
	return err != nil && strings.Contains(err.Error(), "operation not supported")
}
