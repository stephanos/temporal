package volume

import (
	"errors"
	"strings"
	"syscall"
)

type CapturedEntry struct {
	Path  string
	Entry LoadEntry
}

type CapturedSnapshotLimits struct {
	Entries uint64
	Bytes   uint64
	Lookups uint64
}

// CapturedSnapshot owns immutable input bytes; each filesystem receives its own
// bounded lookup accounting when the snapshot is installed or cloned.
type CapturedSnapshot struct {
	mounts         []string
	entries        map[string]LoadEntry
	maximumLookups uint64
	lookups        uint64
}

func NewCapturedSnapshot(mounts []string, entries []CapturedEntry, limits CapturedSnapshotLimits) (*CapturedSnapshot, error) {
	if limits.Entries == 0 || limits.Entries > maximumNodes || limits.Bytes == 0 || limits.Bytes > maximumTotalBytes || limits.Lookups == 0 {
		return nil, errors.New("captured snapshot limits are invalid")
	}
	if len(mounts) == 0 || uint64(len(mounts)) > limits.Entries || uint64(len(entries)) > limits.Entries {
		return nil, syscall.ENOSPC
	}
	for index, mount := range mounts {
		path, _, err := Normalize(mount)
		if err != nil || path != mount || index > 0 && mount <= mounts[index-1] {
			return nil, syscall.EINVAL
		}
		for _, prior := range mounts[:index] {
			if prior == "/" || strings.HasPrefix(mount+"/", prior+"/") {
				return nil, syscall.EINVAL
			}
		}
	}
	snapshot := &CapturedSnapshot{mounts: append([]string(nil), mounts...), entries: make(map[string]LoadEntry, len(entries)), maximumLookups: limits.Lookups}
	var usedBytes uint64
	for _, mount := range mounts {
		if uint64(len(mount)) > limits.Bytes-usedBytes {
			return nil, syscall.ENOSPC
		}
		usedBytes += uint64(len(mount))
	}
	usedEntries := uint64(len(entries))
	for _, entry := range entries {
		path, _, err := Normalize(entry.Path)
		if err != nil || path != entry.Path || !snapshot.mounted(path) {
			return nil, syscall.EINVAL
		}
		if _, exists := snapshot.entries[path]; exists {
			return nil, syscall.EEXIST
		}
		if entry.Entry.Kind != KindFile && entry.Entry.Kind != KindDirectory {
			return nil, syscall.EINVAL
		}
		if uint64(len(entry.Entry.Children)) > limits.Entries-usedEntries || uint64(len(entry.Entry.Data)) > MaximumFileBytes || uint64(len(entry.Entry.Data)) > limits.Bytes-usedBytes {
			return nil, syscall.ENOSPC
		}
		usedEntries += uint64(len(entry.Entry.Children))
		if uint64(len(entry.Path)) > limits.Bytes-usedBytes {
			return nil, syscall.ENOSPC
		}
		usedBytes += uint64(len(entry.Path))
		if entry.Entry.Kind == KindFile && len(entry.Entry.Children) != 0 || entry.Entry.Kind == KindDirectory && len(entry.Entry.Data) != 0 {
			return nil, syscall.EINVAL
		}
		for _, child := range entry.Entry.Children {
			if child.Name == "" || child.Name == "." || child.Name == ".." || len(child.Name) > MaximumPathBytes || strings.ContainsAny(child.Name, "/\x00") || child.Kind != KindFile && child.Kind != KindDirectory {
				return nil, syscall.EINVAL
			}
			if uint64(len(child.Name)) > limits.Bytes-usedBytes {
				return nil, syscall.ENOSPC
			}
			usedBytes += uint64(len(child.Name))
		}
		if uint64(len(entry.Entry.Data)) > limits.Bytes-usedBytes {
			return nil, syscall.ENOSPC
		}
		usedBytes += uint64(len(entry.Entry.Data))
		snapshot.entries[path] = cloneLoadEntry(entry.Entry)
	}
	return snapshot, nil
}

func (snapshot *CapturedSnapshot) Load(path string) (LoadEntry, MountStatus, error) {
	if snapshot == nil {
		return LoadEntry{}, MountUnmounted, nil
	}
	if snapshot.lookups == snapshot.maximumLookups {
		return LoadEntry{}, MountUnmounted, &VolumeCapacityError{Resource: "captured_lookups", Required: snapshot.lookups + 1, Maximum: snapshot.maximumLookups}
	}
	snapshot.lookups++
	if entry, exists := snapshot.entries[path]; exists {
		return cloneLoadEntry(entry), MountOK, nil
	}
	if snapshot.mounted(path) {
		return LoadEntry{}, MountNotExist, nil
	}
	return LoadEntry{}, MountUnmounted, nil
}

func (snapshot *CapturedSnapshot) mounted(path string) bool {
	for _, mount := range snapshot.mounts {
		if mount == "/" || path == mount || strings.HasPrefix(path, mount+"/") {
			return true
		}
	}
	return false
}

func (snapshot *CapturedSnapshot) clone() *CapturedSnapshot {
	if snapshot == nil {
		return nil
	}
	return &CapturedSnapshot{mounts: snapshot.mounts, entries: snapshot.entries, maximumLookups: snapshot.maximumLookups}
}

func cloneLoadEntry(entry LoadEntry) LoadEntry {
	entry.Data = append([]byte(nil), entry.Data...)
	entry.Children = append([]Child(nil), entry.Children...)
	return entry
}
