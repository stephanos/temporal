package gomadvfd

import (
	"sync/atomic"
	"testing"
)

type testBackend struct{}

func (testBackend) Socket() (any, Status)               { return new(int), OK }
func (testBackend) Attach(any, uintptr, uint64)         {}
func (testBackend) Bind(any, Address) Status            { return OK }
func (testBackend) Listen(any, int) Status              { return OK }
func (testBackend) Connect(any, Address) Status         { return OK }
func (testBackend) Accept(any) (any, Address, Status)   { return new(int), Address{}, OK }
func (testBackend) Read(any, []byte) (int, Status)      { return 0, WouldBlock }
func (testBackend) Write(_ any, b []byte) (int, Status) { return len(b), OK }
func (testBackend) Shutdown(any, int) Status            { return OK }
func (testBackend) Close(any) Status                    { return OK }
func (testBackend) Local(any) (Address, Status)         { return Address{}, OK }
func (testBackend) Remote(any) (Address, Status)        { return Address{}, OK }
func (testBackend) Refuse(string)                       {}

func resetDescriptors(t *testing.T) {
	t.Helper()
	atomic.StoreUint32(&boundaryEnabled, 0)
	table.backend = nil
	table.next = 0
	table.live = 0
	table.ready = nil
	for index := range table.entries {
		table.entries[index] = nil
		atomic.StoreUint64(&tokens[index], 0)
		atomic.StoreUint32(&pendingReady[index], 0)
	}
	t.Cleanup(func() { atomic.StoreUint32(&boundaryEnabled, 0) })
}

func TestDescriptorActivationOwnershipAndBound(t *testing.T) {
	resetDescriptors(t)
	if Enabled() || Token(FirstFD) != 0 {
		t.Fatal("boundary enabled by default")
	}
	if status := SetEnabled(true, nil); status != Unsupported {
		t.Fatalf("missing backend=%v", status)
	}
	if status := RegisterBackend(testBackend{}); status != OK {
		t.Fatalf("register=%v", status)
	}
	for _, reserved := range []int{FirstFD, FirstFD + MaxDescriptors - 1} {
		if status := SetEnabled(true, []int{0, 1, 2, 4, reserved}); status != Invalid {
			t.Fatalf("collision %d=%v", reserved, status)
		}
	}
	if status := SetEnabled(true, []int{0, 1, 2, 4, FirstFD - 1, FirstFD + MaxDescriptors}); status != OK {
		t.Fatalf("enable=%v", status)
	}
	for index := 0; index < MaxDescriptors; index++ {
		fd, status := Socket()
		if status != OK || fd != FirstFD+index || Token(uintptr(fd)) == 0 {
			t.Fatalf("allocate %d=%d,%v", index, fd, status)
		}
		if index == 0 && SetEnabled(false, nil) != Invalid {
			t.Fatal("disabled live ownership")
		}
		if Close(fd) != OK || Token(uintptr(fd)) != 0 || Close(fd) != Closed {
			t.Fatalf("stale ownership %d", fd)
		}
	}
	if _, status := Socket(); status != Capacity {
		t.Fatalf("exhaustion=%v", status)
	}
	if SetEnabled(false, nil) != OK {
		t.Fatal("disable")
	}
}

func TestDescriptorNotificationsValidateOwnershipAndOrder(t *testing.T) {
	resetDescriptors(t)
	if status := RegisterBackend(testBackend{}); status != OK {
		t.Fatalf("register=%v", status)
	}
	if status := SetEnabled(true, nil); status != OK {
		t.Fatalf("enable=%v", status)
	}
	left, s := Socket()
	if s != OK {
		t.Fatal(s)
	}
	defer func() {
		if Close(left) != OK {
			t.Error("left close")
		}
	}()
	right, s := Socket()
	if s != OK {
		t.Fatal(s)
	}
	defer func() {
		if Close(right) != OK {
			t.Error("right close")
		}
	}()
	var got []Notice
	previous := RegisterReady(func(fd uintptr, g uint64, mode int32) {
		if Token(fd) != g {
			t.Fatal("unowned callback")
		}
		got = append(got, Notice{fd, g, mode})
	})
	defer RegisterReady(previous)
	l, r := Token(uintptr(left)), Token(uintptr(right))
	Notify(Notice{uintptr(right), r, 'w'}, Notice{uintptr(left), l, 'w'}, Notice{uintptr(left), l, 'r'}, Notice{uintptr(left), l + 1, 'r'}, Notice{4, 1, 'r'})
	want := []Notice{{uintptr(left), l, 'r'}, {uintptr(left), l, 'w'}, {uintptr(right), r, 'w'}}
	if len(got) != len(want) {
		t.Fatalf("notices=%v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order=%v", got)
		}
	}
}

func TestDescriptorRetainsReadinessBeforeRegistration(t *testing.T) {
	resetDescriptors(t)
	if RegisterBackend(testBackend{}) != OK || SetEnabled(true, nil) != OK {
		t.Fatal("setup")
	}
	fd, status := Socket()
	if status != OK {
		t.Fatal(status)
	}
	defer func() {
		if status := Close(fd); status != OK {
			t.Error(status)
		}
	}()
	previous := RegisterReady(nil)
	defer RegisterReady(previous)
	token := Token(uintptr(fd))
	Notify(Notice{uintptr(fd), token, 'r'}, Notice{uintptr(fd), token, 'w'})
	if mode := TakeReady(uintptr(fd), token+1); mode != 0 {
		t.Fatalf("stale token readiness=%d", mode)
	}
	if mode := TakeReady(uintptr(fd), token); mode != 'r'+'w' {
		t.Fatalf("early readiness=%d, want=%d", mode, 'r'+'w')
	}
	if mode := TakeReady(uintptr(fd), token); mode != 0 {
		t.Fatalf("readiness not consumed=%d", mode)
	}
	Notify(Notice{uintptr(fd), token + 1, 'r'}, Notice{4, 1, 'w'})
	if mode := TakeReady(uintptr(fd), token); mode != 0 {
		t.Fatalf("stale notice stored=%d", mode)
	}
	if mode := TakeReady(4, 1); mode != 0 {
		t.Fatalf("unowned readiness=%d", mode)
	}
}

type blockingBackend struct {
	testBackend
	entered chan struct{}
	release chan struct{}
}

func (backend blockingBackend) Read(any, []byte) (int, Status) {
	close(backend.entered)
	<-backend.release
	return 0, OK
}

func TestDescriptorCloseRetainsInFlightReference(t *testing.T) {
	resetDescriptors(t)
	backend := blockingBackend{entered: make(chan struct{}), release: make(chan struct{})}
	if RegisterBackend(backend) != OK || SetEnabled(true, nil) != OK {
		t.Fatal("setup")
	}
	fd, status := Socket()
	if status != OK {
		t.Fatal(status)
	}
	if SetFlags(fd, 7) != OK || SetDescriptorFlags(fd, 1) != OK {
		t.Fatal("set flags")
	}
	if flags, s := Flags(fd); flags != 7 || s != OK {
		t.Fatalf("flags=%d,%v", flags, s)
	}
	if flags, s := DescriptorFlags(fd); flags != 1 || s != OK {
		t.Fatalf("descriptor flags=%d,%v", flags, s)
	}
	result := make(chan Status, 1)
	go func() { _, status := Read(fd, nil); result <- status }()
	<-backend.entered
	if Close(fd) != OK || Token(uintptr(fd)) != 0 {
		t.Fatal("close ownership")
	}
	if _, status := Write(fd, nil); status != Closed {
		t.Fatalf("closed write=%v", status)
	}
	if SetEnabled(false, nil) != Invalid {
		t.Fatal("released in-flight reference")
	}
	close(backend.release)
	if status := <-result; status != OK {
		t.Fatal(status)
	}
	if SetEnabled(false, nil) != OK {
		t.Fatal("reference leaked")
	}
}
