package gomadio

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"runtime"
	"syscall"
	"testing"
	"time"

	"internal/gomadsim"
)

func TestNetworkHandleLocalOperations(t *testing.T) {
	listener, err := ListenTCP("tcp4", "127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	client, err := DialTCP(context.Background(), "tcp4", "127.0.0.1", listener.Address().Port)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	server, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	if client.LocalAddress() != server.RemoteAddress() || client.RemoteAddress() != server.LocalAddress() {
		t.Fatal("paired addresses differ")
	}
	payload := bytes.Repeat([]byte{'a'}, maximumChunkBytes+7)
	if n, err := client.Write(payload); n != len(payload) || err != nil {
		t.Fatalf("write=%d,%v", n, err)
	}
	if err := client.CloseWrite(); err != nil {
		t.Fatal(err)
	}
	if err := server.SetReadDeadline(time.Now().Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	var got []byte
	for {
		buffer := make([]byte, 997)
		n, err := server.Read(buffer)
		got = append(got, buffer[:n]...)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("final data=%d", len(got))
	}
	if n, err := client.Read(nil); n != 0 || err != nil {
		t.Fatalf("empty read=%d,%v", n, err)
	}
}

func networkHandleModel(t *testing.T) (*simulationNetwork, gomadsim.NetworkDomain, gomadsim.NetworkDomain) {
	t.Helper()
	run := gomadsim.Begin(1<<20, 8)
	left := gomadsim.Register(run, "a", "10.0.0.1", 1)
	right := gomadsim.Register(run, "b", "10.0.0.2", 1)
	model, err := newSimulationNetwork(run, simulationNetworkConfig{
		Limits: simulationLimits{Listeners: 8, Connections: 8, Deliveries: 8, Bytes: 1 << 20, Transitions: 128},
		Nodes:  []simulationNodeConfig{{Node: "a", Address: "10.0.0.1"}, {Node: "b", Address: "10.0.0.2"}},
		Links:  []simulationLink{{From: "a", To: "b", Enabled: true}, {From: "b", To: "a", Enabled: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	a, _ := gomadsim.DescribeNetworkDomain(left)
	b, _ := gomadsim.DescribeNetworkDomain(right)
	t.Cleanup(func() { gomadsim.Leave(0) })
	return model, a, b
}

func TestNetworkHandleSimulationValidationBeforeMutation(t *testing.T) {
	model, a, b := networkHandleModel(t)
	gomadsim.Enter(b.Token)
	listener, err := model.listen("tcp4", b.Address, 7233, b)
	if err != nil {
		t.Fatal(err)
	}
	gomadsim.Enter(a.Token)
	client, err := model.dial(context.Background(), "tcp4", b.Address, 7233, a)
	if err != nil {
		t.Fatal(err)
	}
	gomadsim.Enter(b.Token)
	server, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	// Wrong incarnation must not change stream state, deadlines or the tape.
	wrong := gomadsim.Register(model.run, "a", a.Address, 2)
	gomadsim.Enter(wrong)
	before := len(model.transitions)
	for _, token := range []uint64{wrong, b.Token} {
		gomadsim.Enter(token)
		for _, operation := range []func() error{
			func() error { _, err := client.Read(make([]byte, 1)); return err },
			func() error { _, err := client.Write([]byte{'x'}); return err },
			client.Close, client.CloseRead, client.CloseWrite,
			func() error { return client.SetDeadline(time.Now()) },
			func() error { return client.SetReadDeadline(time.Now()) },
			func() error { return client.SetWriteDeadline(time.Now()) },
		} {
			if err := operation(); !errors.Is(err, syscall.ESTALE) {
				t.Fatalf("stale conn operation=%v", err)
			}
		}
	}
	gomadsim.Enter(a.Token)
	for _, operation := range []func() error{
		func() error { _, err := listener.Accept(); return err }, listener.Close,
		func() error { return listener.SetDeadline(time.Now()) },
	} {
		if err := operation(); !errors.Is(err, syscall.ESTALE) {
			t.Fatalf("wrong listener owner=%v", err)
		}
	}
	gomadsim.Enter(wrong)
	if len(model.transitions) != before || model.pendingBytes != 0 {
		t.Fatal("stale operation mutated model")
	}
	if n, err := client.Read(nil); n != 0 || err != nil {
		t.Fatalf("empty stale read=%d,%v", n, err)
	}
	if n, err := client.Write(nil); n != 0 || !errors.Is(err, syscall.ESTALE) {
		t.Fatalf("empty stale write=%d,%v", n, err)
	}
	gomadsim.Enter(a.Token)
	model.limits.Bytes = 1
	if _, err := client.Write([]byte("xx")); !errors.Is(err, ErrResourceExhausted) {
		t.Fatalf("capacity=%v", err)
	}
	if model.nextDelivery != 0 || model.pendingBytes != 0 {
		t.Fatal("capacity consumed delivery identity")
	}
	model.limits.Bytes = 1 << 20
	model.links[simulationLinkKey("a", "b")] = simulationLink{From: "a", To: "b", Enabled: true, DelayNanos: uint64(time.Hour)}
	if _, err := client.Write([]byte{'x'}); err != nil {
		t.Fatal(err)
	}
	if err := model.revoke(a, false); err != nil {
		t.Fatal(err)
	}
	if len(model.deliveries) != 0 || model.pendingBytes != 0 {
		t.Fatal("crash left delayed delivery")
	}
	gomadsim.Enter(b.Token)
	if n, err := server.Read(make([]byte, 1)); n != 0 || !errors.Is(err, ErrClosed) {
		t.Fatalf("crash read=%d,%v", n, err)
	}
}

func TestNetworkHandleHostRejectsWrongResourceEveryOperation(t *testing.T) {
	for _, op := range []processNetworkOperation{processNetworkAcceptOp, processNetworkListenerCloseOp, processNetworkListenerSetDeadlineOp, processNetworkConnReadOp, processNetworkConnWriteOp, processNetworkConnCloseOp, processNetworkConnCloseReadOp, processNetworkConnCloseWriteOp, processNetworkConnSetDeadlineOp, processNetworkConnSetReadDeadlineOp, processNetworkConnSetWriteDeadlineOp} {
		for _, resource := range []processNetworkResource{{domain: 1, listener: &Listener{}}, {domain: 1, conn: &Conn{}}} {
			handle, err := registerProcessNetworkResource(resource)
			if err != nil {
				t.Fatal(err)
			}
			result := applyProcessNetworkOperation(gomadsim.NetworkDomain{Token: 2}, processNetworkCommand{Operation: op, Handle: handle})
			if !errors.Is(result.Err, syscall.ESTALE) {
				t.Fatalf("op=%d wrong domain=%v", op, result.Err)
			}
			listenerOp := op == processNetworkAcceptOp || op == processNetworkListenerCloseOp || op == processNetworkListenerSetDeadlineOp
			if listenerOp != (resource.listener != nil) {
				result = applyProcessNetworkOperation(gomadsim.NetworkDomain{Token: 1}, processNetworkCommand{Operation: op, Handle: handle})
				if !errors.Is(result.Err, syscall.ESTALE) {
					t.Fatalf("op=%d wrong kind=%v", op, result.Err)
				}
			}
			removeProcessNetworkResource(handle)
			result = applyProcessNetworkOperation(gomadsim.NetworkDomain{Token: 1}, processNetworkCommand{Operation: op, Handle: handle})
			if !errors.Is(result.Err, syscall.ESTALE) {
				t.Fatalf("op=%d removed=%v", op, result.Err)
			}
		}
	}
}

func TestNetworkHandleDeadlineChangeWakesBlockedRead(t *testing.T) {
	listener, err := ListenTCP("tcp4", "127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	client, err := DialTCP(context.Background(), "tcp4", "127.0.0.1", listener.Address().Port)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	server, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	result := make(chan error, 1)
	go func() { _, err := server.Read(make([]byte, 1)); result <- err }()
	// Acquiring the direction lock covers entry into the entire real Read.
	for server.readMu.TryLock() {
		server.readMu.Unlock()
		runtime.Gosched()
	}
	if err := server.SetReadDeadline(time.Now().Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := <-result; !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatalf("blocked read=%v", err)
	}
	if err := server.SetReadDeadline(time.Time{}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Write([]byte{'z'}); err != nil {
		t.Fatal(err)
	}
	buffer := make([]byte, 1)
	if n, err := server.Read(buffer); n != 1 || err != nil || buffer[0] != 'z' {
		t.Fatalf("cleared read=%d,%v,%q", n, err, buffer)
	}
}

func TestNetworkHandleReplayRejectsWriteAndDeliveryBeforeMutation(t *testing.T) {
	for _, reject := range []string{"write", "deliver"} {
		t.Run(reject, func(t *testing.T) {
			model, a, b := networkHandleModel(t)
			gomadsim.Enter(b.Token)
			listener, err := model.listen("tcp4", b.Address, 7233, b)
			if err != nil {
				t.Fatal(err)
			}
			gomadsim.Enter(a.Token)
			client, err := model.dial(context.Background(), "tcp4", b.Address, 7233, a)
			if err != nil {
				t.Fatal(err)
			}
			gomadsim.Enter(b.Token)
			server, err := listener.Accept()
			if err != nil {
				t.Fatal(err)
			}
			gomadsim.Enter(a.Token)
			if reject == "deliver" {
				if _, err := client.Write([]byte{'x'}); err != nil {
					t.Fatal(err)
				}
			}
			beforeDelivery, beforeBytes := model.nextDelivery, model.pendingBytes
			beforeTransitions := len(model.transitions)
			model.replay = &simulationRecord{}
			model.replayLanes = make(map[string][]simulationTransition)
			model.replayNext = make(map[string]int)
			if reject == "write" {
				_, err = client.Write([]byte{'x'})
			} else {
				gomadsim.Enter(b.Token)
				_, err = server.Read(make([]byte, 1))
			}
			if _, ok := err.(*simulationBridgeError); !ok {
				t.Fatalf("replay error=%T,%v", err, err)
			}
			if model.nextDelivery != beforeDelivery || model.pendingBytes != beforeBytes || len(model.transitions) != beforeTransitions {
				t.Fatal("replay rejection mutated delivery or tape")
			}
			if reject == "deliver" && len(model.deliveries) != 1 {
				t.Fatal("replay rejection consumed incoming delivery")
			}
		})
	}
}

func TestNetworkHandleHostCloseRemovesResources(t *testing.T) {
	listener, err := ListenTCP("tcp4", "127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	client, err := DialTCP(context.Background(), "tcp4", "127.0.0.1", listener.Address().Port)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	server, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		resource processNetworkResource
		op       processNetworkOperation
	}{
		{processNetworkResource{domain: 1, listener: listener}, processNetworkListenerCloseOp},
		{processNetworkResource{domain: 1, conn: server}, processNetworkConnCloseOp},
	} {
		handle, err := registerProcessNetworkResource(test.resource)
		if err != nil {
			t.Fatal(err)
		}
		command := processNetworkCommand{Operation: test.op, Handle: handle}
		if result := applyProcessNetworkOperation(gomadsim.NetworkDomain{Token: 1}, command); result.Err != nil {
			t.Fatal(result.Err)
		}
		if result := applyProcessNetworkOperation(gomadsim.NetworkDomain{Token: 1}, command); !errors.Is(result.Err, syscall.ESTALE) {
			t.Fatalf("repeated host close=%v", result.Err)
		}
	}
}
