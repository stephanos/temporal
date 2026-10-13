package gomadsim_test

import (
	"errors"
	"syscall"
	"testing"

	"internal/gomadsim"
)

func TestDomainInheritanceRevocationAndOutput(t *testing.T) {
	run := gomadsim.Begin(64, 2)
	if run == 0 {
		t.Fatal("Begin returned token zero")
	}
	domain := gomadsim.Register(run, "node", "10.0.0.1", 1)
	if domain == 0 {
		t.Fatal("Register returned token zero")
	}
	previous, ok := gomadsim.Enter(domain)
	if !ok {
		t.Fatal("Enter rejected an active domain")
	}
	child := make(chan string, 1)
	go func() {
		hostname, err, handled := gomadsim.Hostname()
		if err != nil || !handled {
			child <- ""
			return
		}
		if err, handled := gomadsim.ObserveOutput(gomadsim.OutputStdout, []byte("output")); err != nil || !handled {
			child <- ""
			return
		}
		child <- hostname
	}()
	if hostname := <-child; hostname != "node" {
		t.Fatalf("inherited hostname = %q, want node", hostname)
	}
	gomadsim.Leave(previous)
	if !gomadsim.Revoke(domain) {
		t.Fatal("Revoke rejected an active domain")
	}
	if _, ok := gomadsim.Enter(domain); ok {
		t.Fatal("Enter accepted a revoked domain")
	}
	encoded, ok := gomadsim.Finish(run)
	if !ok || len(encoded) == 0 {
		t.Fatal("Finish did not return output evidence")
	}
}

func TestDomainRegistrationIsBoundedPerRun(t *testing.T) {
	run := gomadsim.Begin(8, 1)
	if run == 0 {
		t.Fatal("Begin returned token zero")
	}
	if token := gomadsim.Register(run, "first", "10.0.0.1", 1); token == 0 {
		t.Fatal("first registration failed")
	}
	if token := gomadsim.Register(run, "second", "10.0.0.2", 1); token != 0 {
		t.Fatalf("registration beyond capacity returned token %d", token)
	}
	if _, ok := gomadsim.Finish(run); !ok {
		t.Fatal("Finish rejected an active run")
	}
}

func TestOutputWriteIsAtomicWithRevocation(t *testing.T) {
	run := gomadsim.Begin(64, 1)
	if run == 0 {
		t.Fatal("Begin returned token zero")
	}
	domain := gomadsim.Register(run, "node", "10.0.0.1", 1)
	if domain == 0 {
		t.Fatal("Register returned token zero")
	}
	previous, ok := gomadsim.Enter(domain)
	if !ok {
		t.Fatal("Enter rejected an active domain")
	}
	defer gomadsim.Leave(previous)

	admitted := make(chan struct{})
	release := make(chan struct{})
	writeDone := make(chan error, 1)
	go func() {
		_, err, handled := gomadsim.ObserveWrite(gomadsim.OutputStdout, []byte("written"), func(source []byte) (int, error) {
			close(admitted)
			<-release
			return len(source), nil
		})
		if !handled {
			err = syscall.EINVAL
		}
		writeDone <- err
	}()
	<-admitted
	revokeDone := make(chan bool, 1)
	go func() { revokeDone <- gomadsim.Revoke(domain) }()
	select {
	case <-revokeDone:
		t.Fatal("Revoke completed during an admitted output write")
	default:
	}
	close(release)
	if err := <-writeDone; err != nil {
		t.Fatalf("ObserveWrite returned %v", err)
	}
	if ok := <-revokeDone; !ok {
		t.Fatal("Revoke rejected the domain after the output write")
	}
	called := false
	if _, err, handled := gomadsim.ObserveWrite(gomadsim.OutputStdout, []byte("stale"), func(source []byte) (int, error) {
		called = true
		return len(source), nil
	}); !handled || !errors.Is(err, syscall.ESTALE) {
		t.Fatalf("stale ObserveWrite = (%v, %t), want ESTALE and handled", err, handled)
	}
	if called {
		t.Fatal("stale ObserveWrite invoked the host write")
	}
	if _, ok := gomadsim.Finish(run); !ok {
		t.Fatal("Finish rejected an active run")
	}
}
