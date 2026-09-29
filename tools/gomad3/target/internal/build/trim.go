package build

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// MaximumCacheBytes bounds the target build cache. Every go build stores its
// compile and link outputs there, so a qualification set over a large closure
// would otherwise grow it without bound; the go command itself trims only
// entries unused for days.
const MaximumCacheBytes = 4 << 30

// TrimCache deletes the least recently used entries of the target build cache
// until it holds at most maximumBytes. The go command touches an entry when
// it reuses it and tolerates a missing one by rebuilding, so trimming never
// changes a build's result, only its cost.
func TrimCache(cache string, maximumBytes uint64) error {
	type entry struct {
		path     string
		size     uint64
		modified int64
	}
	var entries []entry
	var total uint64
	err := filepath.WalkDir(cache, func(path string, item os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !item.Type().IsRegular() {
			return nil
		}
		// The cache root's own bookkeeping files stay.
		if filepath.Dir(path) == cache {
			return nil
		}
		info, err := item.Info()
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
