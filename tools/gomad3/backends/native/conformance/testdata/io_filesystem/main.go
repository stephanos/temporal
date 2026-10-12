// io_filesystem exercises every modeled os filesystem and process-identity
// operation against the in-memory filesystem and prints "ok". Given a host
// path argument it instead reports whether that file is reachable: the
// deterministic filesystem starts empty, so the answer must be "isolated".
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"
)

func main() {
	if len(os.Args) > 1 {
		if _, err := os.ReadFile(os.Args[1]); err == nil {
			fmt.Fprintln(os.Stderr, "host file is visible:", os.Args[1])
			os.Exit(1)
		}
		fmt.Println("isolated")
		return
	}
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("ok")
}

func run() error {
	if err := exerciseIdentity(); err != nil {
		return err
	}
	if err := expectWorkingDirectory("/"); err != nil {
		return err
	}
	if err := os.Mkdir("workspace", 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll("workspace/nested/deep", 0o755); err != nil {
		return err
	}
	if err := os.Chdir("workspace"); err != nil {
		return err
	}
	if err := expectWorkingDirectory("/workspace"); err != nil {
		return err
	}
	if err := exerciseFile(); err != nil {
		return err
	}
	if err := exercisePaths(); err != nil {
		return err
	}
	if err := exerciseDirectories(); err != nil {
		return err
	}
	if err := exerciseDenied(); err != nil {
		return err
	}
	if err := os.Chdir("/"); err != nil {
		return err
	}
	if err := os.Remove("workspace/renamed"); err != nil {
		return err
	}
	if err := os.RemoveAll("workspace"); err != nil {
		return err
	}
	if _, err := os.Stat("workspace"); !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("workspace survived RemoveAll: %v", err)
	}
	return nil
}

func exerciseIdentity() error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	if hostname != "gomad-host" {
		return fmt.Errorf("hostname = %q", hostname)
	}
	if file := os.NewFile(7, "seven"); file != nil {
		return errors.New("NewFile exposed an undeclared descriptor")
	}
	if pid := os.Getpid(); pid != 1 {
		return fmt.Errorf("pid = %d", pid)
	}
	if ppid := os.Getppid(); ppid != 0 {
		return fmt.Errorf("ppid = %d", ppid)
	}
	if ids := []int{os.Getuid(), os.Geteuid(), os.Getgid(), os.Getegid()}; !slices.Equal(ids, []int{0, 0, 0, 0}) {
		return fmt.Errorf("identity = %v", ids)
	}
	groups, err := os.Getgroups()
	if err != nil {
		return err
	}
	if !slices.Equal(groups, []int{0}) {
		return fmt.Errorf("groups = %v", groups)
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	if executable != "/gomad3-target" {
		return fmt.Errorf("executable = %q", executable)
	}
	return nil
}

func expectWorkingDirectory(want string) error {
	directory, err := os.Getwd()
	if err != nil {
		return err
	}
	if directory != want {
		return fmt.Errorf("working directory = %q, want %q", directory, want)
	}
	return nil
}

func exerciseFile() error {
	file, err := os.OpenFile("data.txt", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write([]byte("hello world")); err != nil {
		return err
	}
	if _, err := file.WriteAt([]byte("HELLO"), 0); err != nil {
		return err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	buffer := make([]byte, 16)
	read, err := file.Read(buffer)
	if err != nil {
		return err
	}
	if string(buffer[:read]) != "HELLO world" {
		return fmt.Errorf("read %q", buffer[:read])
	}
	read, err = file.ReadAt(buffer[:5], 6)
	if err != nil {
		return err
	}
	if string(buffer[:read]) != "world" {
		return fmt.Errorf("read at offset %q", buffer[:read])
	}
	if err := file.Truncate(5); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Chmod(0o600); err != nil {
		return err
	}
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.Size() != 5 || info.Mode().Perm() != 0o600 || info.Name() != "data.txt" {
		return fmt.Errorf("file stat = %d %v %q", info.Size(), info.Mode(), info.Name())
	}
	return file.Close()
}

func exercisePaths() error {
	info, err := os.Stat("data.txt")
	if err != nil {
		return err
	}
	if info.Size() != 5 || info.Mode().Perm() != 0o600 {
		return fmt.Errorf("stat = %d %v", info.Size(), info.Mode())
	}
	if err := os.Chmod("data.txt", 0o644); err != nil {
		return err
	}
	if err := os.Truncate("data.txt", 2); err != nil {
		return err
	}
	modified := time.Unix(1_000_000, 0)
	if err := os.Chtimes("data.txt", modified, modified); err != nil {
		return err
	}
	info, err = os.Lstat("data.txt")
	if err != nil {
		return err
	}
	if info.Size() != 2 || info.Mode().Perm() != 0o644 || !info.ModTime().Equal(modified) {
		return fmt.Errorf("lstat = %d %v %v", info.Size(), info.Mode(), info.ModTime())
	}
	if err := os.Rename("data.txt", "renamed"); err != nil {
		return err
	}
	if _, err := os.Stat("data.txt"); !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("renamed source survived: %v", err)
	}
	contents, err := os.ReadFile("renamed")
	if err != nil {
		return err
	}
	if string(contents) != "HE" {
		return fmt.Errorf("renamed contents = %q", contents)
	}
	return nil
}

func exerciseDirectories() error {
	want := []string{"nested", "renamed"}
	entries, err := os.ReadDir(".")
	if err != nil {
		return err
	}
	if names := entryNames(entries); !slices.Equal(names, want) {
		return fmt.Errorf("ReadDir = %v", names)
	}
	names, err := listDirectory(func(directory *os.File) ([]string, error) { return directory.Readdirnames(-1) })
	if err != nil {
		return err
	}
	if !slices.Equal(names, want) {
		return fmt.Errorf("Readdirnames = %v", names)
	}
	names, err = listDirectory(func(directory *os.File) ([]string, error) {
		infos, err := directory.Readdir(-1)
		if err != nil {
			return nil, err
		}
		if len(infos) != 2 || !infos[0].IsDir() || infos[1].IsDir() {
			return nil, fmt.Errorf("Readdir modes = %v", infos)
		}
		names := make([]string, 0, len(infos))
		for _, info := range infos {
			names = append(names, info.Name())
		}
		return names, nil
	})
	if err != nil {
		return err
	}
	if !slices.Equal(names, want) {
		return fmt.Errorf("Readdir = %v", names)
	}
	names, err = listDirectory(func(directory *os.File) ([]string, error) {
		entries, err := directory.ReadDir(-1)
		if err != nil {
			return nil, err
		}
		return entryNames(entries), nil
	})
	if err != nil {
		return err
	}
	if !slices.Equal(names, want) {
		return fmt.Errorf("File.ReadDir = %v", names)
	}
	nested, err := os.Open("nested")
	if err != nil {
		return err
	}
	defer nested.Close()
	if err := nested.Chdir(); err != nil {
		return err
	}
	return expectWorkingDirectory("/workspace/nested")
}

func listDirectory(list func(*os.File) ([]string, error)) ([]string, error) {
	directory, err := os.Open(".")
	if err != nil {
		return nil, err
	}
	defer directory.Close()
	return list(directory)
}

func entryNames(entries []os.DirEntry) []string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

// exerciseDenied calls every os operation the boundary refuses; each one must
// fail without touching the in-memory or host filesystem.
func exerciseDenied() error {
	file, err := os.Open("/workspace/renamed")
	if err != nil {
		return err
	}
	defer file.Close()
	if descriptor := file.Fd(); descriptor != ^uintptr(0) {
		return fmt.Errorf("Fd exposed descriptor %d", descriptor)
	}
	var process *os.Process
	denied := []struct {
		name string
		call func() error
	}{
		{"Chown", func() error { return os.Chown("/workspace/renamed", 0, 0) }},
		{"Lchown", func() error { return os.Lchown("/workspace/renamed", 0, 0) }},
		{"File.Chown", func() error { return file.Chown(0, 0) }},
		{"Link", func() error { return os.Link("/workspace/renamed", "/workspace/linked") }},
		{"Symlink", func() error { return os.Symlink("/workspace/renamed", "/workspace/symlinked") }},
		{"Readlink", func() error { _, err := os.Readlink("/workspace/renamed"); return err }},
		{"File.ReadFrom", func() error { _, err := file.ReadFrom(strings.NewReader("more")); return err }},
		{"File.WriteTo", func() error { _, err := file.WriteTo(io.Discard); return err }},
		{"File.SetDeadline", func() error { return file.SetDeadline(time.Now()) }},
		{"File.SetReadDeadline", func() error { return file.SetReadDeadline(time.Now()) }},
		{"File.SetWriteDeadline", func() error { return file.SetWriteDeadline(time.Now()) }},
		{"File.SyscallConn", func() error { _, err := file.SyscallConn(); return err }},
		{"OpenRoot", func() error { _, err := os.OpenRoot("/workspace"); return err }},
		{"OpenInRoot", func() error { _, err := os.OpenInRoot("/workspace", "/workspace/renamed"); return err }},
		{"Pipe", func() error { _, _, err := os.Pipe(); return err }},
		{"StartProcess", func() error {
			_, err := os.StartProcess("/gomad3-target", []string{"target"}, &os.ProcAttr{})
			return err
		}},
		{"FindProcess", func() error { _, err := os.FindProcess(1); return err }},
		{"Process.Kill", func() error { return process.Kill() }},
		{"Process.Signal", func() error { return process.Signal(os.Interrupt) }},
		{"Process.Wait", func() error { _, err := process.Wait(); return err }},
		{"Process.WithHandle", func() error { return process.WithHandle(func(uintptr) {}) }},
	}
	for _, operation := range denied {
		if err := operation.call(); err == nil {
			return fmt.Errorf("%s succeeded", operation.name)
		}
	}
	for _, path := range []string{"/workspace/linked", "/workspace/symlinked"} {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("denied operation created %s: %v", path, err)
		}
	}
	return nil
}
