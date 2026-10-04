// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadfs

type handleImplementation interface {
	Read([]byte) (int, error)
	ReadAt([]byte, int64) (int, error)
	Write([]byte) (int, error)
	WriteAt([]byte, int64) (int, error)
	Truncate(int64) error
	Chmod(uint32) error
	Chtimes(int64) error
	Chdir() error
	Seek(int64, int) (int64, error)
	Stat() (Entry, error)
	Path() string
	ReadDir(int) ([]Entry, error)
	Close() error
	Sync() error
	Map(int64, uint64, bool) (*Mapping, error)
}
type mappingImplementation interface {
	Bytes() ([]byte, error)
	Close() error
}
type Handle struct{ implementation handleImplementation }

// Mapping is a memory view of one file region. Mappings of the same region
// share one buffer, so a store through one is visible through every other and,
// once flushed, through the file itself; that is the shared-memory contract
// SQLite's WAL index relies on. A writable mapping is available only for a
// volatile file, because stores through memory bypass the volume journal.
type Mapping struct{ implementation mappingImplementation }

func (handle *Handle) Read(destination []byte) (int, error) {
	return handle.implementation.Read(destination)
}
func (handle *Handle) ReadAt(destination []byte, offset int64) (int, error) {
	return handle.implementation.ReadAt(destination, offset)
}
func (handle *Handle) Write(source []byte) (int, error) { return handle.implementation.Write(source) }
func (handle *Handle) WriteAt(source []byte, offset int64) (int, error) {
	return handle.implementation.WriteAt(source, offset)
}
func (handle *Handle) Truncate(size int64) error   { return handle.implementation.Truncate(size) }
func (handle *Handle) Chmod(mode uint32) error     { return handle.implementation.Chmod(mode) }
func (handle *Handle) Chtimes(modTime int64) error { return handle.implementation.Chtimes(modTime) }
func (handle *Handle) Chdir() error                { return handle.implementation.Chdir() }
func (handle *Handle) Seek(offset int64, whence int) (int64, error) {
	return handle.implementation.Seek(offset, whence)
}
func (handle *Handle) Stat() (Entry, error) { return handle.implementation.Stat() }
func (handle *Handle) Path() string         { return handle.implementation.Path() }
func (handle *Handle) ReadDir(count int) ([]Entry, error) {
	return handle.implementation.ReadDir(count)
}
func (handle *Handle) Close() error { return handle.implementation.Close() }
func (handle *Handle) Sync() error  { return handle.implementation.Sync() }
func (handle *Handle) Map(offset int64, length uint64, writable bool) (*Mapping, error) {
	return handle.implementation.Map(offset, length, writable)
}
func (mapping *Mapping) Bytes() ([]byte, error) { return mapping.implementation.Bytes() }
func (mapping *Mapping) Close() error           { return mapping.implementation.Close() }
