// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package os

import (
	"internal/gomadtrace"
	"strings"
	"sync"
	"syscall"
	"time"
	_ "unsafe"
)

//go:linkname gomadProfileEnabled runtime.gomadIOProfileEnabled
func gomadProfileEnabled() bool

func gomadIOEnabled() bool {
	return gomadProfileEnabled()
}

var gomadFilesystem = struct {
	sync.RWMutex
	directories map[string]FileMode
}{}

var gomadFilesystemOnce sync.Once

func gomadInitializeFilesystem() {
	gomadFilesystemOnce.Do(func() {
		gomadFilesystem.directories = map[string]FileMode{"/": 0o755}
	})
}

func gomadMkdir(name string, perm FileMode) error {
	gomadInitializeFilesystem()
	path, _, err := gomadNormalizePath(name)
	if err != nil || path == "/" {
		return gomadPathError("os.mkdir", "mkdir", name, syscall.EINVAL)
	}
	parent := path[:strings.LastIndexByte(path, '/')]
	if parent == "" {
		parent = "/"
	}
	gomadFilesystem.Lock()
	defer gomadFilesystem.Unlock()
	if _, found := gomadFilesystem.directories[path]; found {
		return gomadPathError("os.mkdir", "mkdir", path, syscall.EEXIST)
	}
	if _, found := gomadFilesystem.directories[parent]; !found {
		return gomadPathError("os.mkdir", "mkdir", path, syscall.ENOENT)
	}
	gomadFilesystem.directories[path] = perm & ModePerm
	gomadRecordPath("os.mkdir", path, uint64(perm&ModePerm), nil)
	return nil
}

func gomadMkdirAll(name string, perm FileMode) error {
	gomadInitializeFilesystem()
	path, _, err := gomadNormalizePath(name)
	if err != nil {
		return gomadPathError("os.mkdirall", "mkdir", name, syscall.EINVAL)
	}
	gomadFilesystem.Lock()
	defer gomadFilesystem.Unlock()
	current := ""
	for _, component := range strings.Split(strings.TrimPrefix(path, "/"), "/") {
		if component == "" {
			continue
		}
		current += "/" + component
		if _, found := gomadFilesystem.directories[current]; !found {
			gomadFilesystem.directories[current] = perm & ModePerm
		}
	}
	gomadRecordPath("os.mkdirall", path, uint64(perm&ModePerm), nil)
	return nil
}

func gomadStat(name string) (FileInfo, error) {
	gomadInitializeFilesystem()
	path, base, err := gomadNormalizePath(name)
	if err != nil {
		return nil, gomadPathError("os.stat", "stat", name, syscall.EINVAL)
	}
	gomadFilesystem.RLock()
	mode, found := gomadFilesystem.directories[path]
	gomadFilesystem.RUnlock()
	if !found {
		return nil, gomadPathError("os.stat", "stat", path, syscall.ENOENT)
	}
	gomadRecordPath("os.stat", path, uint64(mode|ModeDir), nil)
	return gomadFileInfo{name: base, mode: mode | ModeDir}, nil
}

func gomadHostname() (string, error) {
	const hostname = "gomad-host"
	gomadRecordPath("os.hostname", hostname, uint64(len(hostname)), nil)
	return hostname, nil
}

func gomadPathError(operation, pathOperation, path string, err error) error {
	gomadRecordPath(operation, path, 0, err)
	return &PathError{Op: pathOperation, Path: path, Err: err}
}

func gomadRecordPath(operation, path string, count uint64, err error) {
	gomadtrace.Init()
	result := uint32(0)
	switch err {
	case nil:
	case syscall.EINVAL:
		result = 1
	case syscall.EEXIST:
		result = 2
	case syscall.ENOENT:
		result = 3
	default:
		result = 4
	}
	gomadtrace.Record(operation, []byte(path), nil, count, result, 0, 0)
}

func gomadNormalizePath(name string) (string, string, error) {
	if name == "" || strings.IndexByte(name, 0) >= 0 {
		return "", "", syscall.EINVAL
	}
	components := make([]string, 0, strings.Count(name, "/")+1)
	for _, component := range strings.Split(name, "/") {
		switch component {
		case "", ".":
		case "..":
			if len(components) == 0 {
				return "", "", syscall.EINVAL
			}
			components = components[:len(components)-1]
		default:
			components = append(components, component)
		}
	}
	if len(components) == 0 {
		return "/", "/", nil
	}
	return "/" + strings.Join(components, "/"), components[len(components)-1], nil
}

type gomadFileInfo struct {
	name string
	mode FileMode
}

func (info gomadFileInfo) Name() string       { return info.name }
func (info gomadFileInfo) Size() int64        { return 0 }
func (info gomadFileInfo) Mode() FileMode     { return info.mode }
func (info gomadFileInfo) ModTime() time.Time { return time.Unix(0, 0) }
func (info gomadFileInfo) IsDir() bool        { return true }
func (info gomadFileInfo) Sys() any           { return nil }
