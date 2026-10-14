// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadfs

import (
	"io"
	"sort"
	"syscall"
)

type localHandle struct {
	fs              *FS
	node            *node
	name            string
	offset          int64
	directoryOffset int
	readable        bool
	writable        bool
	append          bool
	closed          bool
	revoked         bool
	generation      uint64
}

type localMapping struct {
	fs         *FS
	node       *node
	offset     uint64
	data       []byte
	writable   bool
	owner      bool
	closed     bool
	revoked    bool
	generation uint64
}

func (handle *localHandle) Read(destination []byte) (int, error) {
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if err := handle.errorLocked(); err != nil {
		return 0, err
	}
	if !handle.readable {
		return 0, syscall.EBADF
	}
	if handle.node.kind != KindFile {
		return 0, syscall.EISDIR
	}
	handle.fs.flushMappingsLocked(handle.node)
	if handle.offset >= int64(len(handle.node.data)) {
		return 0, io.EOF
	}
	n := copy(destination, handle.node.data[handle.offset:])
	handle.offset += int64(n)
	return n, nil
}

func (handle *localHandle) ReadAt(destination []byte, offset int64) (int, error) {
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if err := handle.errorLocked(); err != nil {
		return 0, err
	}
	if !handle.readable {
		return 0, syscall.EBADF
	}
	if offset < 0 {
		return 0, syscall.EINVAL
	}
	if handle.node.kind != KindFile {
		return 0, syscall.EISDIR
	}
	handle.fs.flushMappingsLocked(handle.node)
	if offset >= int64(len(handle.node.data)) {
		return 0, io.EOF
	}
	n := copy(destination, handle.node.data[offset:])
	if n != len(destination) {
		return n, io.EOF
	}
	return n, nil
}

func (handle *localHandle) Write(source []byte) (int, error) {
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if err := handle.errorLocked(); err != nil {
		return 0, err
	}
	if !handle.writable {
		return 0, syscall.EBADF
	}
	if handle.node.readonly {
		return 0, syscall.EROFS
	}
	if handle.append {
		handle.offset = int64(len(handle.node.data))
	}
	end := handle.offset + int64(len(source))
	if end < handle.offset || end > MaximumFileBytes {
		return 0, syscall.EFBIG
	}
	operationCount := uint64(1)
	if end > int64(len(handle.node.data)) {
		operationCount++
	}
	if err := handle.fs.preflightVolumeOperationsLocked(handle.node.volume, operationCount); err != nil {
		return 0, err
	}
	if err := handle.fs.preflightVolumeGrowthLocked(handle.node, end); err != nil {
		return 0, err
	}
	start := handle.offset
	modTime := handle.fs.nowLocked()
	if err := handle.fs.recordWriteLocked(handle.node, uint64(start), source, uint64(end), modTime); err != nil {
		return 0, err
	}
	if end > int64(len(handle.node.data)) {
		growth := uint64(end - int64(len(handle.node.data)))
		if growth > maximumTotalBytes-handle.fs.usedBytes {
			return 0, syscall.ENOSPC
		}
		handle.node.data = append(handle.node.data, make([]byte, int(end)-len(handle.node.data))...)
		handle.fs.usedBytes += growth
	}
	copy(handle.node.data[start:end], source)
	handle.fs.updateMappingsLocked(handle.node, uint64(start), source)
	handle.offset = end
	handle.node.modTime = modTime
	return len(source), nil
}

func (handle *localHandle) WriteAt(source []byte, offset int64) (int, error) {
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if err := handle.errorLocked(); err != nil {
		return 0, err
	}
	if !handle.writable {
		return 0, syscall.EBADF
	}
	if handle.append || offset < 0 {
		return 0, syscall.EINVAL
	}
	if handle.node.readonly {
		return 0, syscall.EROFS
	}
	end := offset + int64(len(source))
	if end < offset || end > MaximumFileBytes {
		return 0, syscall.EFBIG
	}
	operationCount := uint64(1)
	if end > int64(len(handle.node.data)) {
		operationCount++
	}
	if err := handle.fs.preflightVolumeOperationsLocked(handle.node.volume, operationCount); err != nil {
		return 0, err
	}
	if err := handle.fs.preflightVolumeGrowthLocked(handle.node, end); err != nil {
		return 0, err
	}
	modTime := handle.fs.nowLocked()
	if err := handle.fs.recordWriteLocked(handle.node, uint64(offset), source, uint64(end), modTime); err != nil {
		return 0, err
	}
	if end > int64(len(handle.node.data)) {
		growth := uint64(end - int64(len(handle.node.data)))
		if growth > maximumTotalBytes-handle.fs.usedBytes {
			return 0, syscall.ENOSPC
		}
		handle.node.data = append(handle.node.data, make([]byte, int(end)-len(handle.node.data))...)
		handle.fs.usedBytes += growth
	}
	copy(handle.node.data[offset:end], source)
	handle.fs.updateMappingsLocked(handle.node, uint64(offset), source)
	handle.node.modTime = modTime
	return len(source), nil
}

func (handle *localHandle) Truncate(size int64) error {
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if err := handle.errorLocked(); err != nil {
		return err
	}
	if !handle.writable {
		return syscall.EBADF
	}
	if handle.node.readonly {
		return syscall.EROFS
	}
	if size < 0 {
		return syscall.EINVAL
	}
	if size > MaximumFileBytes {
		return syscall.EFBIG
	}
	if err := handle.fs.preflightVolumeOperationsLocked(handle.node.volume, 1); err != nil {
		return err
	}
	if err := handle.fs.preflightVolumeGrowthLocked(handle.node, size); err != nil {
		return err
	}
	modTime := handle.fs.nowLocked()
	if err := handle.fs.recordResizeLocked(handle.node, uint64(size), modTime); err != nil {
		return err
	}
	if size <= int64(len(handle.node.data)) {
		handle.fs.truncateMappingsLocked(handle.node, uint64(size), uint64(len(handle.node.data)))
		handle.fs.usedBytes -= uint64(int64(len(handle.node.data)) - size)
		handle.node.data = handle.node.data[:size]
	} else {
		growth := uint64(size - int64(len(handle.node.data)))
		if growth > maximumTotalBytes-handle.fs.usedBytes {
			return syscall.ENOSPC
		}
		handle.node.data = append(handle.node.data, make([]byte, int(size)-len(handle.node.data))...)
		handle.fs.usedBytes += growth
	}
	handle.node.modTime = modTime
	return nil
}

func (handle *localHandle) Chmod(mode uint32) error {
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if err := handle.errorLocked(); err != nil {
		return err
	}
	if handle.node.readonly {
		return syscall.EROFS
	}
	if err := handle.fs.preflightVolumeOperationsLocked(handle.node.volume, 1); err != nil {
		return err
	}
	nextMode := mode & 0o777
	nextModTime := handle.fs.nowLocked()
	if err := handle.fs.recordMetadataLocked(handle.node, nextMode, nextModTime); err != nil {
		return err
	}
	handle.node.mode = nextMode
	handle.node.modTime = nextModTime
	return nil
}

func (handle *localHandle) Chtimes(modTime int64) error {
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if err := handle.errorLocked(); err != nil {
		return err
	}
	if handle.node.readonly {
		return syscall.EROFS
	}
	if err := handle.fs.preflightVolumeOperationsLocked(handle.node.volume, 1); err != nil {
		return err
	}
	if err := handle.fs.recordMetadataLocked(handle.node, handle.node.mode, modTime); err != nil {
		return err
	}
	handle.node.modTime = modTime
	return nil
}

func (handle *localHandle) Chdir() error {
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if err := handle.errorLocked(); err != nil {
		return err
	}
	if handle.node.kind != KindDirectory {
		return syscall.ENOTDIR
	}
	if !handle.node.linked {
		return syscall.ENOENT
	}
	for name, candidate := range handle.fs.nodes {
		if candidate == handle.node {
			handle.fs.cwd = name
			return nil
		}
	}
	return syscall.ENOENT
}

func (handle *localHandle) Seek(offset int64, whence int) (int64, error) {
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if err := handle.errorLocked(); err != nil {
		return 0, err
	}
	var next int64
	switch whence {
	case io.SeekStart:
		next = offset
	case io.SeekCurrent:
		next = handle.offset + offset
	case io.SeekEnd:
		next = int64(len(handle.node.data)) + offset
	default:
		return 0, syscall.EINVAL
	}
	if next < 0 {
		return 0, syscall.EINVAL
	}
	handle.offset = next
	return next, nil
}

func (handle *localHandle) Stat() (Entry, error) {
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if err := handle.errorLocked(); err != nil {
		return Entry{}, err
	}
	_, base, _ := Normalize(handle.name)
	return entryForNode(base, handle.node), nil
}

func (handle *localHandle) Path() string {
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	return handle.name
}

func (handle *localHandle) ReadDir(count int) ([]Entry, error) {
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if err := handle.errorLocked(); err != nil {
		return nil, err
	}
	if handle.node.kind != KindDirectory {
		return nil, syscall.ENOTDIR
	}
	entries := make(map[string]Entry)
	for _, child := range handle.node.children {
		entries[child.Name] = Entry{Name: child.Name, Mode: child.Mode, Kind: child.Kind}
	}
	for name, childNode := range handle.fs.nodes {
		if parentPath(name) == handle.name && name != handle.name {
			_, base, _ := Normalize(name)
			entries[base] = entryForNode(base, childNode)
		}
	}
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)
	if handle.directoryOffset == len(names) {
		if count > 0 {
			return nil, io.EOF
		}
		return []Entry{}, nil
	}
	take := len(names) - handle.directoryOffset
	if count > 0 && count < take {
		take = count
	}
	result := make([]Entry, 0, take)
	for _, name := range names[handle.directoryOffset : handle.directoryOffset+take] {
		result = append(result, entries[name])
	}
	handle.directoryOffset += take
	return result, nil
}

func (handle *localHandle) Close() error {
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if err := handle.errorLocked(); err != nil {
		return err
	}
	handle.closed = true
	delete(handle.fs.handles, handle)
	handle.fs.openHandles--
	handle.node.handles--
	handle.fs.releaseNodeLocked(handle.node)
	return nil
}

func (handle *localHandle) Sync() error {
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if err := handle.errorLocked(); err != nil {
		return err
	}
	handle.fs.flushMappingsLocked(handle.node)
	return handle.fs.syncNodeLocked(handle.node)
}

func (handle *localHandle) Map(offset int64, length uint64, writable bool) (*Mapping, error) {
	handle.fs.mu.Lock()
	defer handle.fs.mu.Unlock()
	if err := handle.errorLocked(); err != nil {
		return nil, err
	}
	if !handle.readable || writable && !handle.writable {
		return nil, syscall.EBADF
	}
	if handle.node.kind != KindFile {
		return nil, syscall.ENODEV
	}
	if offset < 0 || length == 0 || length > MaximumFileBytes || uint64(offset) > MaximumFileBytes-length {
		return nil, syscall.EINVAL
	}
	if writable && (handle.node.readonly || handle.node.volume != "") {
		return nil, syscall.ENOTSUP
	}
	mapping := &localMapping{fs: handle.fs, node: handle.node, offset: uint64(offset), writable: writable, generation: handle.fs.generation}
	for existing := range handle.fs.mappings {
		if existing.node != handle.node {
			continue
		}
		if existing.offset == mapping.offset && uint64(len(existing.data)) == length {
			mapping.data = existing.data
			continue
		}
		if (existing.writable || writable) && existing.offset < mapping.offset+length && mapping.offset < existing.offset+uint64(len(existing.data)) {
			return nil, syscall.EINVAL
		}
	}
	if mapping.data == nil {
		if length > maximumMappedBytes-handle.fs.mappedBytes {
			return nil, syscall.ENOMEM
		}
		mapping.data = make([]byte, length)
		mapping.owner = true
		handle.fs.mappedBytes += length
		if mapping.offset < uint64(len(handle.node.data)) {
			copy(mapping.data, handle.node.data[mapping.offset:])
		}
	}
	handle.fs.mappings[mapping] = struct{}{}
	return &Mapping{implementation: mapping}, nil
}

func (mapping *localMapping) Bytes() ([]byte, error) {
	mapping.fs.mu.Lock()
	defer mapping.fs.mu.Unlock()
	if err := mapping.errorLocked(); err != nil {
		return nil, err
	}
	return mapping.data, nil
}

func (mapping *localMapping) Close() error {
	mapping.fs.mu.Lock()
	defer mapping.fs.mu.Unlock()
	if err := mapping.errorLocked(); err != nil {
		return err
	}
	if mapping.writable {
		mapping.fs.flushMappingLocked(mapping)
	}
	mapping.closed = true
	delete(mapping.fs.mappings, mapping)
	if mapping.owner {
		// The buffer stays alive while an alias still maps it; the accounting
		// moves to the first surviving alias.
		mapping.owner = false
		for alias := range mapping.fs.mappings {
			if alias.node == mapping.node && alias.offset == mapping.offset && len(alias.data) == len(mapping.data) {
				alias.owner = true
				break
			}
		}
		if !mapping.ownerTransferredLocked() {
			mapping.fs.mappedBytes -= uint64(len(mapping.data))
		}
	}
	mapping.data = nil
	return nil
}

func (mapping *localMapping) ownerTransferredLocked() bool {
	for alias := range mapping.fs.mappings {
		if alias.node == mapping.node && alias.offset == mapping.offset && len(alias.data) == len(mapping.data) && alias.owner {
			return true
		}
	}
	return false
}

func (mapping *localMapping) errorLocked() error {
	if mapping.fs.unavailable != nil {
		return mapping.fs.unavailable
	}
	if mapping.revoked || mapping.generation != mapping.fs.generation {
		return syscall.ESTALE
	}
	if mapping.closed {
		return syscall.EINVAL
	}
	return nil
}

// flushMappingsLocked makes stores through writable mappings of n visible to
// file reads. Bytes beyond the file's size stay in the mapping only, as they
// would beyond a real file's end.
func (fs *FS) flushMappingsLocked(n *node) {
	for mapping := range fs.mappings {
		if mapping.node == n && mapping.writable {
			fs.flushMappingLocked(mapping)
		}
	}
}

func (fs *FS) flushMappingLocked(mapping *localMapping) {
	if mapping.offset >= uint64(len(mapping.node.data)) {
		return
	}
	copy(mapping.node.data[mapping.offset:], mapping.data)
}

func (fs *FS) truncateMappingsLocked(n *node, size, previous uint64) {
	for mapping := range fs.mappings {
		if mapping.node != n || size >= mapping.offset+uint64(len(mapping.data)) {
			continue
		}
		start := uint64(0)
		if size > mapping.offset {
			start = size - mapping.offset
		}
		end := uint64(len(mapping.data))
		if previous < mapping.offset+end {
			if previous <= mapping.offset {
				continue
			}
			end = previous - mapping.offset
		}
		clear(mapping.data[start:end])
	}
}

func (fs *FS) updateMappingsLocked(n *node, offset uint64, source []byte) {
	end := offset + uint64(len(source))
	for mapping := range fs.mappings {
		mappingEnd := mapping.offset + uint64(len(mapping.data))
		if mapping.node != n || offset >= mappingEnd || end <= mapping.offset {
			continue
		}
		from := max(offset, mapping.offset)
		to := min(end, mappingEnd)
		copy(mapping.data[from-mapping.offset:to-mapping.offset], source[from-offset:to-offset])
	}
}

func (handle *localHandle) errorLocked() error {
	if handle.fs.unavailable != nil {
		return handle.fs.unavailable
	}
	if handle.revoked || handle.generation != handle.fs.generation {
		return syscall.ESTALE
	}
	if handle.closed {
		return ErrClosed
	}
	return nil
}
