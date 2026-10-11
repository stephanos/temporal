package volume

// Metadata returns detached file attributes and size without copying file data.
func (handle *Handle) Metadata() (Entry, uint64, error) {
	if handle == nil {
		return Entry{}, 0, ErrClosed
	}
	implementation, ok := handle.implementation.(*localHandle)
	if !ok || implementation == nil {
		return Entry{}, 0, ErrClosed
	}
	implementation.fs.mu.Lock()
	defer implementation.fs.mu.Unlock()
	if err := implementation.errorLocked(); err != nil {
		return Entry{}, 0, err
	}
	_, base, _ := Normalize(implementation.name)
	n := implementation.node
	return Entry{Name: base, Mode: n.mode, Kind: n.kind, ModTime: n.modTime}, uint64(len(n.data)), nil
}
