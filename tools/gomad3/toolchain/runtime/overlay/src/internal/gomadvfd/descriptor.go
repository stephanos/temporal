// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadvfd

import (
	"sync"
	_ "unsafe"

	"internal/runtime/atomic"
)

type Status uint8

const (
	OK Status = iota
	WouldBlock
	Closed
	Refused
	AddressInUse
	Capacity
	Unsupported
	NotConnected
	Invalid
	EndOfStream
	BrokenPipe
	Reset
)

const FirstFD = 1 << 20
const MaxDescriptors = 1 << 16

type Address struct {
	IP   [4]byte
	Port int
}

type Notice struct {
	FD         uintptr
	Generation uint64
	Mode       int32
}

type Backend interface {
	Socket() (any, Status)
	Attach(any, uintptr, uint64)
	Bind(any, Address) Status
	Listen(any, int) Status
	Connect(any, Address) Status
	Accept(any) (any, Address, Status)
	Read(any, []byte) (int, Status)
	Write(any, []byte) (int, Status)
	Shutdown(any, int) Status
	Close(any) Status
	Local(any) (Address, Status)
	Remote(any) (Address, Status)
	Refuse(string)
}

type descriptor struct {
	handle          any
	generation      uint64
	references      int
	closing         bool
	flags           int
	descriptorFlags int
}

var boundaryEnabled uint32
var tokens [MaxDescriptors]uint64
var pendingReady [MaxDescriptors]uint32
var allocation sync.Mutex
var table = struct {
	sync.Mutex
	backend Backend
	next    int
	live    int
	entries [MaxDescriptors]*descriptor
	ready   func(uintptr, uint64, int32)
}{}

func RegisterBackend(backend Backend) Status {
	allocation.Lock()
	defer allocation.Unlock()
	table.Lock()
	defer table.Unlock()
	if backend == nil || table.live != 0 {
		return Invalid
	}
	table.backend = backend
	return OK
}

func SetEnabled(enabled bool, reserved []int) Status {
	allocation.Lock()
	defer allocation.Unlock()
	table.Lock()
	defer table.Unlock()
	if !enabled {
		if table.live != 0 {
			return Invalid
		}
		atomic.Store(&boundaryEnabled, 0)
		return OK
	}
	for _, fd := range reserved {
		if InRange(uintptr(fd)) {
			return Invalid
		}
	}
	if table.backend == nil {
		return Unsupported
	}
	atomic.Store(&boundaryEnabled, 1)
	return OK
}

//go:nosplit
//go:norace
func Enabled() bool { return atomic.Load(&boundaryEnabled) != 0 }

//go:nosplit
//go:norace
func InRange(fd uintptr) bool { return fd >= FirstFD && fd < FirstFD+MaxDescriptors }

//go:linkname Token
//go:nosplit
//go:norace
func Token(fd uintptr) uint64 {
	if !Enabled() || !InRange(fd) {
		return 0
	}
	return atomic.Load64(&tokens[fd-FirstFD])
}

//go:nosplit
//go:norace
func IsFD(fd uintptr) bool { return Token(fd) != 0 }

func Owns(fd int) bool { return IsFD(uintptr(fd)) }

//go:linkname RegisterReady
func RegisterReady(ready func(uintptr, uint64, int32)) func(uintptr, uint64, int32) {
	table.Lock()
	previous := table.ready
	table.ready = ready
	table.Unlock()
	return previous
}

//go:linkname TakeReady
//go:nosplit
//go:norace
func TakeReady(fd uintptr, generation uint64) int32 {
	if generation == 0 || Token(fd) != generation {
		return 0
	}
	bits := atomic.Xchg(&pendingReady[fd-FirstFD], 0)
	var mode int32
	if bits&1 != 0 {
		mode += 'r'
	}
	if bits&2 != 0 {
		mode += 'w'
	}
	return mode
}

func Notify(notices ...Notice) {
	for i := 1; i < len(notices); i++ {
		for j := i; j > 0 && (notices[j].FD < notices[j-1].FD || notices[j].FD == notices[j-1].FD && notices[j].Mode < notices[j-1].Mode); j-- {
			notices[j], notices[j-1] = notices[j-1], notices[j]
		}
	}
	table.Lock()
	ready := table.ready
	table.Unlock()
	for i, notice := range notices {
		if notice.Generation == 0 || Token(notice.FD) != notice.Generation || notice.Mode != 'r' && notice.Mode != 'w' {
			continue
		}
		if i != 0 && notice == notices[i-1] {
			continue
		}
		bit := uint32(1)
		if notice.Mode == 'w' {
			bit = 2
		}
		atomic.Or32(&pendingReady[notice.FD-FirstFD], bit)
		if ready != nil {
			ready(notice.FD, notice.Generation, notice.Mode)
		}
	}
}

func acquire(fd int) (*descriptor, Backend, Status) {
	if !Enabled() {
		return nil, nil, Unsupported
	}
	if !InRange(uintptr(fd)) {
		return nil, nil, Closed
	}
	table.Lock()
	defer table.Unlock()
	entry := table.entries[fd-FirstFD]
	if entry == nil || entry.closing {
		return nil, nil, Closed
	}
	entry.references++
	return entry, table.backend, OK
}

func release(fd int, entry *descriptor) {
	table.Lock()
	entry.references--
	if entry.closing && entry.references == 0 {
		table.entries[fd-FirstFD] = nil
		table.live--
	}
	table.Unlock()
}

func allocate(create func(Backend) (any, Address, Status)) (int, Address, Status) {
	allocation.Lock()
	defer allocation.Unlock()
	table.Lock()
	if !Enabled() || table.backend == nil {
		table.Unlock()
		return -1, Address{}, Unsupported
	}
	if table.next == MaxDescriptors {
		table.Unlock()
		return -1, Address{}, refuseCapacity()
	}
	backend := table.backend
	index := table.next
	table.Unlock()
	handle, address, status := create(backend)
	if status != OK {
		return -1, address, status
	}
	generation := uint64(index) + 1
	fd := FirstFD + index
	table.Lock()
	table.next++
	table.live++
	table.entries[index] = &descriptor{handle: handle, generation: generation}
	atomic.Store64(&tokens[index], generation)
	table.Unlock()
	backend.Attach(handle, uintptr(fd), generation)
	return fd, address, OK
}

func refuseCapacity() Status {
	table.Lock()
	backend := table.backend
	table.Unlock()
	if backend != nil {
		backend.Refuse("descriptor.capacity")
	}
	return Capacity
}

func Socket() (int, Status) {
	fd, _, status := allocate(func(backend Backend) (any, Address, Status) {
		handle, status := backend.Socket()
		return handle, Address{}, status
	})
	return fd, status
}

func Bind(fd int, address Address) Status {
	entry, backend, status := acquire(fd)
	if status != OK {
		return status
	}
	defer release(fd, entry)
	return backend.Bind(entry.handle, address)
}
func Listen(fd, backlog int) Status {
	entry, backend, status := acquire(fd)
	if status != OK {
		return status
	}
	defer release(fd, entry)
	return backend.Listen(entry.handle, backlog)
}
func Connect(fd int, address Address) Status {
	entry, backend, status := acquire(fd)
	if status != OK {
		return status
	}
	defer release(fd, entry)
	return backend.Connect(entry.handle, address)
}
func Accept(fd int) (int, Address, Status) {
	entry, backend, status := acquire(fd)
	if status != OK {
		return -1, Address{}, status
	}
	defer release(fd, entry)
	return allocate(func(Backend) (any, Address, Status) { return backend.Accept(entry.handle) })
}
func Read(fd int, buffer []byte) (int, Status) {
	entry, backend, status := acquire(fd)
	if status != OK {
		return 0, status
	}
	defer release(fd, entry)
	return backend.Read(entry.handle, buffer)
}
func Write(fd int, buffer []byte) (int, Status) {
	entry, backend, status := acquire(fd)
	if status != OK {
		return 0, status
	}
	defer release(fd, entry)
	return backend.Write(entry.handle, buffer)
}
func Shutdown(fd, how int) Status {
	entry, backend, status := acquire(fd)
	if status != OK {
		return status
	}
	defer release(fd, entry)
	return backend.Shutdown(entry.handle, how)
}
func Local(fd int) (Address, Status) {
	entry, backend, status := acquire(fd)
	if status != OK {
		return Address{}, status
	}
	defer release(fd, entry)
	return backend.Local(entry.handle)
}
func Remote(fd int) (Address, Status) {
	entry, backend, status := acquire(fd)
	if status != OK {
		return Address{}, status
	}
	defer release(fd, entry)
	return backend.Remote(entry.handle)
}

func Close(fd int) Status {
	if !Enabled() {
		return Unsupported
	}
	if !InRange(uintptr(fd)) {
		return Closed
	}
	table.Lock()
	entry := table.entries[fd-FirstFD]
	if entry == nil || entry.closing {
		table.Unlock()
		return Closed
	}
	entry.closing = true
	entry.references++
	backend := table.backend
	table.Unlock()
	status := backend.Close(entry.handle)
	atomic.Store64(&tokens[fd-FirstFD], 0)
	release(fd, entry)
	return status
}

func flags(fd int, value int, set, descriptorFlags bool) (int, Status) {
	entry, _, status := acquire(fd)
	if status != OK {
		return 0, status
	}
	defer release(fd, entry)
	table.Lock()
	defer table.Unlock()
	field := &entry.flags
	if descriptorFlags {
		field = &entry.descriptorFlags
	}
	if set {
		*field = value
	}
	return *field, OK
}
func Flags(fd int) (int, Status)           { return flags(fd, 0, false, false) }
func SetFlags(fd, value int) Status        { _, status := flags(fd, value, true, false); return status }
func DescriptorFlags(fd int) (int, Status) { return flags(fd, 0, false, true) }
func SetDescriptorFlags(fd, value int) Status {
	_, status := flags(fd, value, true, true)
	return status
}
func Refuse(operation string) Status {
	table.Lock()
	backend := table.backend
	table.Unlock()
	if backend != nil {
		backend.Refuse(operation)
	}
	return Unsupported
}
