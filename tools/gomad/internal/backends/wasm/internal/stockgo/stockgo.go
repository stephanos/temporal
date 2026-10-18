// Package stockgo shares one private copy of a stock Go distribution between
// test packages and test processes.
//
// Cooperative guest builds pass -overlay to the stock compiler, and overlays
// cannot replace files beneath GOMODCACHE, where GOTOOLCHAIN downloads live.
// Tests therefore need the distribution copied somewhere writable. That copy
// is about 250 MiB, so this package makes it once per distribution, under a
// cross-process lock, below the module's git-ignored .toolchain directory, and
// reuses it on later runs. The shared copy is read-only by convention: a test
// that renames or deletes its distribution must take Clone instead.
package stockgo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Environment names an alternative cache directory. The default is the
// .toolchain directory beside the module's go.mod.
const Environment = "GOMAD_WASM_TOOLCHAIN_CACHE"

const module = "go.temporal.io/server/tools/gomad"

// Command returns the go command tests use to locate a stock distribution:
// GOMAD_STOCK_GO when set, otherwise "go" from PATH.
func Command() string {
	if command := os.Getenv("GOMAD_STOCK_GO"); command != "" {
		return command
	}
	return "go"
}

// TB is the part of testing.TB these helpers use, so measurement producers
// running outside `go test` (cmd/wasmmeasure) can call them too.
type TB interface {
	Helper()
	Fatal(args ...any)
}

// Root returns the shared read-only copy of Command's GOROOT, creating it
// once when missing.
func Root(t TB) string {
	t.Helper()
	root, err := Shared(Command())
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// Compiler returns the go executable inside Root.
func Compiler(t TB) string {
	t.Helper()
	return filepath.Join(Root(t), "bin", "go")
}

// Clone copies the shared root into destination, which must not exist, for a
// test that mutates, renames or removes its distribution. It returns destination.
func Clone(t TB, destination string) string {
	t.Helper()
	if err := os.CopyFS(destination, os.DirFS(Root(t))); err != nil {
		t.Fatal(err)
	}
	return destination
}

type stamp struct {
	Source  string    `json:"source"`
	Version string    `json:"version"`
	Created time.Time `json:"created"`
}

var shared sync.Map // command -> *sharedResult

type sharedResult struct {
	once sync.Once
	root string
	err  error
}

// Shared returns the shared copy of command's GOROOT. Within a process each
// command resolves once; across processes a lock file serializes creation and
// the copy is published by rename, so a reader never observes a partial tree.
func Shared(command string) (string, error) {
	value, _ := shared.LoadOrStore(command, &sharedResult{})
	result := value.(*sharedResult)
	result.once.Do(func() { result.root, result.err = create(command) })
	return result.root, result.err
}

// SharedWithCache resolves an explicit cache without changing process environment.
func SharedWithCache(command, cache string) (string, error) {
	if cache == "" {
		return Shared(command)
	}
	if !filepath.IsAbs(cache) {
		return "", fmt.Errorf("stock distribution cache must be absolute: %q", cache)
	}
	key := command + "\x00" + cache
	value, _ := shared.LoadOrStore(key, &sharedResult{})
	result := value.(*sharedResult)
	result.once.Do(func() { result.root, result.err = createInDirectory(command, cache) })
	return result.root, result.err
}

func create(command string) (string, error) {
	cache, err := cacheDirectory()
	if err != nil {
		return "", err
	}
	return createInDirectory(command, cache)
}

func createInDirectory(command, cache string) (string, error) {
	source, version, err := describe(command)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256([]byte(source + "\n" + version))
	entry := filepath.Join(cache, "stock", hex.EncodeToString(digest[:8]))
	root := filepath.Join(entry, "go")
	if published(entry, root) {
		return root, nil
	}
	if err := os.MkdirAll(filepath.Dir(entry), 0o700); err != nil {
		return "", err
	}
	unlock, err := lock(entry + ".lock")
	if err != nil {
		return "", fmt.Errorf("lock stock distribution copy: %w", err)
	}
	defer unlock()
	if published(entry, root) {
		return root, nil
	}
	// A crashed earlier run may have left staging behind; nothing else can hold the lock now.
	stale, _ := filepath.Glob(entry + ".staging-*")
	for _, path := range stale {
		if err := os.RemoveAll(path); err != nil {
			return "", err
		}
	}
	staging := fmt.Sprintf("%s.staging-%d", entry, os.Getpid())
	if err := os.CopyFS(staging, os.DirFS(source)); err != nil {
		return "", errors.Join(fmt.Errorf("copy stock distribution %s: %w", source, err), os.RemoveAll(staging))
	}
	if err := os.RemoveAll(entry); err != nil {
		return "", err
	}
	if err := os.MkdirAll(entry, 0o700); err != nil {
		return "", err
	}
	if err := os.Rename(staging, root); err != nil {
		return "", errors.Join(err, os.RemoveAll(staging))
	}
	data, err := json.Marshal(stamp{Source: source, Version: version, Created: time.Now().UTC()})
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(entry, "stamp.json"), append(data, '\n'), 0o600); err != nil {
		return "", err
	}
	return root, nil
}

// published reports a complete copy: the stamp is written last, after the rename.
func published(entry, root string) bool {
	if _, err := os.Stat(filepath.Join(entry, "stamp.json")); err != nil {
		return false
	}
	_, err := os.Stat(filepath.Join(root, "bin", "go"))
	return err == nil
}

// describe resolves the distribution root and version of command without
// letting the ambient GOROOT or a toolchain switch redirect it.
func describe(command string) (source, version string, err error) {
	environment := append(os.Environ(), "GOROOT=", "GOTOOLCHAIN=local", "GOENV=off", "GOFLAGS=", "GOWORK=off")
	run := func(arguments ...string) (string, error) {
		cmd := exec.Command(command, arguments...)
		cmd.Env = environment
		output, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("%s %s: %w", command, strings.Join(arguments, " "), err)
		}
		return strings.TrimSpace(string(output)), nil
	}
	if source, err = run("env", "GOROOT"); err != nil {
		return "", "", err
	}
	if source, err = filepath.Abs(source); err != nil {
		return "", "", err
	}
	if version, err = run("version"); err != nil {
		return "", "", err
	}
	return source, version, nil
}

func cacheDirectory() (string, error) {
	if directory := os.Getenv(Environment); directory != "" {
		if !filepath.IsAbs(directory) {
			return "", fmt.Errorf("%s must be absolute: %q", Environment, directory)
		}
		return directory, nil
	}
	directory, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		data, err := os.ReadFile(filepath.Join(directory, "go.mod"))
		if err == nil && strings.HasPrefix(string(data), "module "+module+"\n") {
			return filepath.Join(directory, ".toolchain"), nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", fmt.Errorf("module root of %s not found above the working directory; set %s", module, Environment)
		}
		directory = parent
	}
}
