package gomadio

import (
	"bytes"
	"context"
	"internal/gomadvfd"
	"testing"
)

func descriptorPair(t *testing.T) (listener, client, server int) {
	t.Helper()
	if status := gomadvfd.SetEnabled(true, []int{0, 1, 2, 4}); status != gomadvfd.OK {
		t.Fatalf("enable=%v", status)
	}
	t.Cleanup(func() {
		if status := gomadvfd.SetEnabled(false, nil); status != gomadvfd.OK {
			t.Errorf("disable=%v", status)
		}
	})
	listener, status := gomadvfd.Socket()
	if status != gomadvfd.OK {
		t.Fatal(status)
	}
	t.Cleanup(func() {
		if gomadvfd.Owns(listener) {
			if status := gomadvfd.Close(listener); status != gomadvfd.OK {
				t.Errorf("listener close=%v", status)
			}
		}
	})
	if status = gomadvfd.Bind(listener, gomadvfd.Address{IP: [4]byte{127, 0, 0, 1}}); status != gomadvfd.OK {
		t.Fatal(status)
	}
	if status = gomadvfd.Listen(listener, 64); status != gomadvfd.OK {
		t.Fatal(status)
	}
	address, status := gomadvfd.Local(listener)
	if status != gomadvfd.OK || address.Port == 0 {
		t.Fatalf("address=%v,%v", address, status)
	}
	if _, _, status = gomadvfd.Accept(listener); status != gomadvfd.WouldBlock {
		t.Fatalf("empty accept=%v", status)
	}
	client, status = gomadvfd.Socket()
	if status != gomadvfd.OK {
		t.Fatal(status)
	}
	t.Cleanup(func() {
		if status := gomadvfd.Close(client); status != gomadvfd.OK {
			t.Errorf("client close=%v", status)
		}
	})
	if status = gomadvfd.Connect(client, address); status != gomadvfd.OK {
		t.Fatal(status)
	}
	server, _, status = gomadvfd.Accept(listener)
	if status != gomadvfd.OK {
		t.Fatal(status)
	}
	t.Cleanup(func() {
		if status := gomadvfd.Close(server); status != gomadvfd.OK {
			t.Errorf("server close=%v", status)
		}
	})
	return listener, client, server
}

func TestDescriptorBackendImplicitClientPortSkipsReservations(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(map[bool]string{false: "descriptor", true: "legacy"}[legacy], func(t *testing.T) {
			listener, _, _ := descriptorPair(t)
			address, status := gomadvfd.Local(listener)
			if status != gomadvfd.OK {
				t.Fatal(status)
			}
			networkState.Lock()
			first := networkState.nextClientPort
			networkState.Unlock()
			reservedListener, err := ListenTCP("tcp4", "127.0.0.1", first)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := reservedListener.Close(); err != nil {
					t.Error(err)
				}
			})
			reserved, status := gomadvfd.Socket()
			if status != gomadvfd.OK {
				t.Fatal(status)
			}
			t.Cleanup(func() {
				if status := gomadvfd.Close(reserved); status != gomadvfd.OK {
					t.Error(status)
				}
			})
			if status := gomadvfd.Bind(reserved, gomadvfd.Address{IP: [4]byte{127, 0, 0, 1}, Port: first + 1}); status != gomadvfd.OK {
				t.Fatal(status)
			}
			var localPort int
			if legacy {
				client, err := DialTCP(context.Background(), "tcp4", "127.0.0.1", address.Port)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					if err := client.Close(); err != nil {
						t.Error(err)
					}
				})
				localPort = client.LocalAddress().Port
			} else {
				client, status := gomadvfd.Socket()
				if status != gomadvfd.OK {
					t.Fatal(status)
				}
				t.Cleanup(func() {
					if status := gomadvfd.Close(client); status != gomadvfd.OK {
						t.Error(status)
					}
				})
				if status := gomadvfd.Connect(client, address); status != gomadvfd.OK {
					t.Fatal(status)
				}
				local, status := gomadvfd.Local(client)
				if status != gomadvfd.OK {
					t.Fatal(status)
				}
				localPort = local.Port
			}
			if localPort != first+2 {
				t.Fatalf("implicit client port=%d, want unreserved=%d", localPort, first+2)
			}
		})
	}
}

func TestDescriptorBackendCopiesPartialAndEOF(t *testing.T) {
	_, client, server := descriptorPair(t)
	if n, s := gomadvfd.Read(server, make([]byte, 1)); n != 0 || s != gomadvfd.WouldBlock {
		t.Fatalf("empty read=%d,%v", n, s)
	}
	source := bytes.Repeat([]byte{'a'}, maximumChunkBytes+7)
	if n, s := gomadvfd.Write(client, source); n != maximumChunkBytes || s != gomadvfd.OK {
		t.Fatalf("partial write=%d,%v", n, s)
	}
	source[0] = 'b'
	if s := gomadvfd.Shutdown(client, 1); s != gomadvfd.OK {
		t.Fatal(s)
	}
	buffer := make([]byte, maximumChunkBytes)
	if n, s := gomadvfd.Read(server, buffer); n != len(buffer) || s != gomadvfd.OK || buffer[0] != 'a' {
		t.Fatalf("read=%d,%v first=%c", n, s, buffer[0])
	}
	if n, s := gomadvfd.Read(server, buffer); n != 0 || s != gomadvfd.EndOfStream {
		t.Fatalf("EOF=%d,%v", n, s)
	}
	if n, s := gomadvfd.Write(client, nil); n != 0 || s != gomadvfd.OK {
		t.Fatalf("empty write=%d,%v", n, s)
	}
}

func TestDescriptorBackendBackpressureWakeAndHalfClose(t *testing.T) {
	_, client, server := descriptorPair(t)
	for i := 0; i < maximumPendingChunks; i++ {
		if n, s := gomadvfd.Write(client, []byte{byte(i)}); n != 1 || s != gomadvfd.OK {
			t.Fatalf("fill=%d,%v", n, s)
		}
	}
	if n, s := gomadvfd.Write(client, []byte{99}); n != 0 || s != gomadvfd.WouldBlock {
		t.Fatalf("full=%d,%v", n, s)
	}
	var got []gomadvfd.Notice
	previous := gomadvfd.RegisterReady(func(fd uintptr, g uint64, mode int32) {
		got = append(got, gomadvfd.Notice{FD: fd, Generation: g, Mode: mode})
	})
	defer gomadvfd.RegisterReady(previous)
	buffer := make([]byte, 1)
	if n, s := gomadvfd.Read(server, buffer); n != 1 || s != gomadvfd.OK || buffer[0] != 0 {
		t.Fatalf("FIFO=%d,%v,%v", n, s, buffer)
	}
	if len(got) != 1 || got[0].FD != uintptr(client) || got[0].Mode != 'w' {
		t.Fatalf("capacity notification=%v", got)
	}
	if n, s := gomadvfd.Write(client, []byte{99}); n != 1 || s != gomadvfd.OK {
		t.Fatalf("retry=%d,%v", n, s)
	}
	if s := gomadvfd.Shutdown(server, 0); s != gomadvfd.OK {
		t.Fatal(s)
	}
	if n, s := gomadvfd.Write(client, []byte{1}); n != 0 || s != gomadvfd.BrokenPipe {
		t.Fatalf("peer read closed=%d,%v", n, s)
	}
}

func TestDescriptorBackendBacklogCompletionAndRefusal(t *testing.T) {
	listener, _, _ := descriptorPair(t)
	address, status := gomadvfd.Local(listener)
	if status != gomadvfd.OK {
		t.Fatal(status)
	}
	var clients []int
	t.Cleanup(func() {
		for _, fd := range clients {
			if status := gomadvfd.Close(fd); status != gomadvfd.OK {
				t.Errorf("pending client close=%v", status)
			}
		}
	})
	for i := 0; i < maximumPendingConns+1; i++ {
		fd, status := gomadvfd.Socket()
		if status != gomadvfd.OK {
			t.Fatal(status)
		}
		clients = append(clients, fd)
		status = gomadvfd.Connect(fd, address)
		want := gomadvfd.OK
		if i == maximumPendingConns {
			want = gomadvfd.WouldBlock
		}
		if status != want {
			t.Fatalf("connect %d=%v", i, status)
		}
	}
	waiting := clients[len(clients)-1]
	if mode := gomadvfd.TakeReady(uintptr(waiting), gomadvfd.Token(uintptr(waiting))); mode != 0 {
		t.Fatalf("pending readiness=%c", mode)
	}
	if _, status := gomadvfd.Remote(waiting); status != gomadvfd.WouldBlock {
		t.Fatalf("pending remote=%v", status)
	}
	var ready bool
	previous := gomadvfd.RegisterReady(func(fd uintptr, _ uint64, mode int32) {
		if fd == uintptr(waiting) && mode == 'w' {
			ready = true
		}
	})
	defer gomadvfd.RegisterReady(previous)
	accepted, _, status := gomadvfd.Accept(listener)
	if status != gomadvfd.OK {
		t.Fatal(status)
	}
	if status := gomadvfd.Close(accepted); status != gomadvfd.OK {
		t.Fatal(status)
	}
	if _, status := gomadvfd.Remote(waiting); status != gomadvfd.OK || !ready {
		t.Fatalf("completion=%v ready=%v", status, ready)
	}
	if mode := gomadvfd.TakeReady(uintptr(waiting), gomadvfd.Token(uintptr(waiting))); mode != 'w' {
		t.Fatalf("completion before registration readiness=%d", mode)
	}
	refused, status := gomadvfd.Socket()
	if status != gomadvfd.OK {
		t.Fatal(status)
	}
	clients = append(clients, refused)
	if status := gomadvfd.Connect(refused, gomadvfd.Address{IP: [4]byte{127, 0, 0, 1}, Port: 1}); status != gomadvfd.Refused {
		t.Fatalf("refusal=%v", status)
	}
	if status := gomadvfd.Bind(refused, gomadvfd.Address{IP: [4]byte{10, 0, 0, 1}, Port: 123}); status != gomadvfd.Unsupported {
		t.Fatalf("address refusal=%v", status)
	}
	pending, status := gomadvfd.Socket()
	if status != gomadvfd.OK {
		t.Fatal(status)
	}
	clients = append(clients, pending)
	if status := gomadvfd.Connect(pending, address); status != gomadvfd.WouldBlock {
		t.Fatalf("pending refusal setup=%v", status)
	}
	if status := gomadvfd.Close(listener); status != gomadvfd.OK {
		t.Fatal(status)
	}
	if _, status := gomadvfd.Remote(pending); status != gomadvfd.Refused {
		t.Fatalf("pending refusal=%v", status)
	}
	if mode := gomadvfd.TakeReady(uintptr(pending), gomadvfd.Token(uintptr(pending))); mode != 'r'+'w' {
		t.Fatalf("refusal before registration readiness=%d", mode)
	}
}

func TestDescriptorBackendLegacyPeerPublishesReadiness(t *testing.T) {
	if status := gomadvfd.SetEnabled(true, nil); status != gomadvfd.OK {
		t.Fatal(status)
	}
	t.Cleanup(func() {
		if status := gomadvfd.SetEnabled(false, nil); status != gomadvfd.OK {
			t.Errorf("disable=%v", status)
		}
	})
	listener, err := ListenTCP("tcp4", "127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := listener.Close(); err != nil {
			t.Error(err)
		}
	})
	client, status := gomadvfd.Socket()
	if status != gomadvfd.OK {
		t.Fatal(status)
	}
	t.Cleanup(func() {
		if status := gomadvfd.Close(client); status != gomadvfd.OK {
			t.Errorf("close=%v", status)
		}
	})
	if status := gomadvfd.Connect(client, leafAddress(listener.Address())); status != gomadvfd.OK {
		t.Fatal(status)
	}
	server, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := server.Close(); err != nil {
			t.Error(err)
		}
	})
	var notices []gomadvfd.Notice
	previous := gomadvfd.RegisterReady(func(fd uintptr, g uint64, mode int32) {
		notices = append(notices, gomadvfd.Notice{FD: fd, Generation: g, Mode: mode})
	})
	defer gomadvfd.RegisterReady(previous)
	if n, err := server.Write([]byte("a")); n != 1 || err != nil {
		t.Fatalf("legacy write=%d,%v", n, err)
	}
	if len(notices) != 1 || notices[0].FD != uintptr(client) || notices[0].Mode != 'r' {
		t.Fatalf("legacy write readiness=%v", notices)
	}
	if n, status := gomadvfd.Write(client, []byte("b")); n != 1 || status != gomadvfd.OK {
		t.Fatalf("write=%d,%v", n, status)
	}
	notices = nil
	var buffer [1]byte
	if n, err := server.Read(buffer[:]); n != 1 || err != nil || buffer[0] != 'b' {
		t.Fatalf("legacy read=%d,%v,%v", n, err, buffer)
	}
	if len(notices) != 1 || notices[0].FD != uintptr(client) || notices[0].Mode != 'w' {
		t.Fatalf("legacy read readiness=%v", notices)
	}
	if err := server.CloseWrite(); err != nil {
		t.Fatal(err)
	}
	if n, status := gomadvfd.Read(client, buffer[:]); n != 1 || status != gomadvfd.OK || buffer[0] != 'a' {
		t.Fatalf("read=%d,%v,%v", n, status, buffer)
	}
	if n, status := gomadvfd.Read(client, buffer[:]); n != 0 || status != gomadvfd.EndOfStream {
		t.Fatalf("EOF=%d,%v", n, status)
	}
}

func TestDescriptorBackendBindReservesLegacyListenerPort(t *testing.T) {
	if status := gomadvfd.SetEnabled(true, nil); status != gomadvfd.OK {
		t.Fatal(status)
	}
	t.Cleanup(func() {
		if status := gomadvfd.SetEnabled(false, nil); status != gomadvfd.OK {
			t.Errorf("disable=%v", status)
		}
	})
	fd, status := gomadvfd.Socket()
	if status != gomadvfd.OK {
		t.Fatal(status)
	}
	t.Cleanup(func() {
		if status := gomadvfd.Close(fd); status != gomadvfd.OK {
			t.Errorf("close=%v", status)
		}
	})
	if status := gomadvfd.Bind(fd, gomadvfd.Address{IP: [4]byte{127, 0, 0, 1}}); status != gomadvfd.OK {
		t.Fatal(status)
	}
	address, status := gomadvfd.Local(fd)
	if status != gomadvfd.OK {
		t.Fatal(status)
	}
	listener, err := ListenTCP("tcp4", "127.0.0.1", address.Port)
	if listener != nil {
		t.Cleanup(func() {
			if err := listener.Close(); err != nil {
				t.Error(err)
			}
		})
	}
	if err != ErrAddressInUse {
		t.Fatalf("legacy listener claimed reserved port: %v", err)
	}
	if status := gomadvfd.Listen(fd, 64); status != gomadvfd.OK {
		t.Fatalf("reserved listener=%v", status)
	}
}
