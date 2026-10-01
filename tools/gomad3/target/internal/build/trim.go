package build

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"go.temporal.io/server/tools/gomad3/internal/hostfs"
)

// MaximumCacheBytes bounds the target build cache. Every go build stores its
// compile and link outputs there, so a qualification set over a large closure
// would otherwise grow it without bound; the go command itself trims only
// entries unused for days.
const MaximumCacheBytes = 4 << 30

// cacheLockName is the cache-root file builds share and trimming takes
// exclusively: the go command looks an archive up before it compiles or links
// against it, and an archive deleted in between fails that build.
const cacheLockName = "gomad-cache.lock"

// UseCache holds the target build cache for one build; TrimCache deletes
// nothing while any build holds it. Release the lock when the build ends.
func UseCache(cache string) (*hostfs.Lock, error) {
	lock, err := hostfs.Shared(filepath.Join(cache, cacheLockName))
	if err != nil {
		return nil, fmt.Errorf("hold target build cache: %w", err)
	}
	return lock, nil
}

// TrimCache deletes the least recently used entries of the target build cache
// until it holds at most maximumBytes. It trims only while no build holds the
// cache and otherwise leaves the bound to the next build's trim; the go
// command rebuilds an entry missing when it looks it up.
func TrimCache(cache string, maximumBytes uint64) (retErr error) {
	lock, err := hostfs.Try(filepath.Join(cache, cacheLockName))
	if errors.Is(err, hostfs.ErrContended) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("lock target build cache: %w", err)
	}
	defer func() { retErr = errors.Join(retErr, lock.Release()) }()
	type entry struct {
		path     string
		size     uint64
		modified int64
	}
	var entries []entry
	var total uint64
	err = filepath.WalkDir(cache, func(path string, item os.DirEntry, visitErr error) error {
		// The go command trims its own long-unused entries too.
		if errors.Is(visitErr, os.ErrNotExist) {
			return nil
		}
		if visitErr != nil {
			return visitErr
		}
		if !item.Type().IsRegular() {
			return nil
		}
		// The cache root's own bookkeeping files stay.
		if filepath.Dir(path) == cache {
			return nil
		}
		info, err := item.Info()
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Size() < 0 {
			return fmt.Errorf("negative cache entry size: %s", path)
		}
		entries = append(entries, entry{path: path, size: uint64(info.Size()), modified: info.ModTime().UnixNano()})
		total += uint64(info.Size())
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk target build cache: %w", err)
	}
	if total <= maximumBytes {
		return nil
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].modified != entries[j].modified {
			return entries[i].modified < entries[j].modified
		}
		return entries[i].path < entries[j].path
	})
	for _, item := range entries {
		if total <= maximumBytes {
			break
		}
		if err := os.Remove(item.path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("trim target build cache: %w", err)
		}
		total -= item.size
	}
	return nil
}
