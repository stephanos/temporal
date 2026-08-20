//go:build gomadv3_toolchain

package gomadv3sim

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"slices"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var processIsolationGlobal uint64

func TestProcessBackendResetsGlobalsDescriptorsAndGoroutines(t *testing.T) {
	if !processBackendAvailable() {
		t.Skip("Runner simulation transport is unavailable")
	}
	bootID := uniqueBootID("cluster-process-isolation")
	require.NoError(t, RegisterBoot(bootID, func(ctx context.Context, node NodeContext) error {
		processIsolationGlobal++
		if processIsolationGlobal != 1 {
			return fmt.Errorf("process global = %d", processIsolationGlobal)
		}
		if node.Incarnation == 1 {
			go func() { select {} }()
			select {}
		}
		fmt.Fprintf(os.Stdout, "global=%d descriptors=%v\n", processIsolationGlobal, openProcessDescriptors())
		if node.Incarnation == 3 {
			<-ctx.Done()
			return ctx.Err()
		}
		go func() { select {} }()
		return nil
	}))
	spec := Spec{
		Schema: SpecSchema, Backend: BackendProcess, Fidelity: FidelityHardIsolation, Seed: 79, Limits: DefaultLimits(),
		Nodes: []NodeSpec{{ID: "server", Boot: bootID, Address: "10.0.0.1"}},
	}
	run := func(spec Spec) Result {
		result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
			first, err := cluster.Start(ctx, "server")
			if err != nil {
				return err
			}
			if err := cluster.Crash(ctx, first); err != nil {
				return err
			}
			second, err := cluster.Restart(ctx, "server")
			if err != nil {
				return err
			}
			if _, err := cluster.Wait(ctx, second); err != nil {
				return err
			}
			third, err := cluster.Restart(ctx, "server")
			if err != nil {
				return err
			}
			return cluster.Stop(ctx, third)
		})
		require.NoError(t, err)
		return result
	}
	first := run(spec)
	require.Equal(t, OutcomeCompleted, first.Outcome)
	require.Empty(t, first.Leaks)
	require.Len(t, first.Outputs, 2)
	require.Equal(t, first.Outputs[0].Bytes, first.Outputs[1].Bytes)
	require.Contains(t, string(first.Outputs[0].Bytes), "global=1")
	plan, err := ReplayPlanFor(first.Record)
	require.NoError(t, err)
	spec.Replay = &plan
	replayed := run(spec)
	require.Equal(t, first.Record.Identity, replayed.Record.Identity)
}

func TestProcessAndInProcessBackendsHaveEquivalentDetachedModels(t *testing.T) {
	if !processBackendAvailable() {
		t.Skip("Runner simulation transport is unavailable")
	}
	bootID := uniqueBootID("cluster-process-conformance")
	require.NoError(t, RegisterBoot(bootID, func(context.Context, NodeContext) error { return nil }))
	base := Spec{
		Schema: SpecSchema, Backend: BackendProcess, Fidelity: FidelityHardIsolation, Seed: 83, Limits: DefaultLimits(),
		Nodes: []NodeSpec{
			{ID: "first", Boot: bootID, Address: "10.0.0.1"},
			{ID: "second", Boot: bootID, Address: "10.0.0.2"},
		},
		Links: []LinkSpec{{From: "first", To: "second", Enabled: true}, {From: "second", To: "first", Enabled: true}},
	}
	run := func(spec Spec) Result {
		result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
			first, err := cluster.Start(ctx, "first")
			if err != nil {
				return err
			}
			second, err := cluster.Start(ctx, "second")
			if err != nil {
				return err
			}
			if err := cluster.Partition(ctx, "first", "second"); err != nil {
				return err
			}
			if err := cluster.SetDelay(ctx, "first", "second", 17); err != nil {
				return err
			}
			if err := cluster.Heal(ctx, "first", "second"); err != nil {
				return err
			}
			if _, err := cluster.Wait(ctx, first); err != nil {
				return err
			}
			_, err = cluster.Wait(ctx, second)
			return err
		})
		require.NoError(t, err)
		return result
	}
	processResult := run(base)
	base.Backend = BackendInProcess
	base.Fidelity = FidelitySimulationModel
	inProcessResult := run(base)
	require.Equal(t, inProcessResult.Record.Transitions, processResult.Record.Transitions)
	require.Equal(t, inProcessResult.Record.Network, processResult.Record.Network)
	require.Equal(t, inProcessResult.Record.Volumes, processResult.Record.Volumes)
	require.Equal(t, inProcessResult.Nodes, processResult.Nodes)
}

func TestProcessBackendRoutesTCPThroughSharedHostModel(t *testing.T) {
	if !processBackendAvailable() {
		t.Skip("Runner simulation transport is unavailable")
	}
	firstBoot := uniqueBootID("cluster-process-network-first")
	secondBoot := uniqueBootID("cluster-process-network-second")
	require.NoError(t, RegisterBoot(firstBoot, processNetworkPeer("10.0.0.2:7233", 'a')))
	require.NoError(t, RegisterBoot(secondBoot, processNetworkPeer("10.0.0.1:7233", 'b')))
	spec := Spec{
		Schema: SpecSchema, Backend: BackendProcess, Fidelity: FidelityHardIsolation, Seed: 89, Limits: DefaultLimits(),
		Nodes: []NodeSpec{
			{ID: "first", Boot: firstBoot, Address: "10.0.0.1"},
			{ID: "second", Boot: secondBoot, Address: "10.0.0.2"},
		},
		Links: []LinkSpec{{From: "first", To: "second", Enabled: true}, {From: "second", To: "first", Enabled: true}},
	}
	result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
		first, err := cluster.Start(ctx, "first")
		if err != nil {
			return err
		}
		second, err := cluster.Start(ctx, "second")
		if err != nil {
			return err
		}
		if _, err := cluster.Wait(ctx, first); err != nil {
			return err
		}
		_, err = cluster.Wait(ctx, second)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, result.Outcome)
	require.Len(t, result.Nodes, 2)
	require.NotEmpty(t, result.Network.Transitions)
}

func TestProcessBackendSynchronizesNodeClockWithModelDelay(t *testing.T) {
	if !processBackendAvailable() {
		t.Skip("Runner simulation transport is unavailable")
	}
	serverBoot := uniqueBootID("cluster-process-delay-server")
	clientBoot := uniqueBootID("cluster-process-delay-client")
	require.NoError(t, RegisterBoot(serverBoot, func(_ context.Context, node NodeContext) (resultErr error) {
		listener, err := net.Listen("tcp", net.JoinHostPort(node.Address, "7233"))
		if err != nil {
			return err
		}
		defer func() { resultErr = errors.Join(resultErr, listener.Close()) }()
		connection, err := listener.Accept()
		if err != nil {
			return err
		}
		defer func() { resultErr = errors.Join(resultErr, connection.Close()) }()
		var request [1]byte
		if _, err := io.ReadFull(connection, request[:]); err != nil {
			return err
		}
		_, err = connection.Write(request[:])
		return err
	}))
	require.NoError(t, RegisterBoot(clientBoot, func(ctx context.Context, _ NodeContext) (resultErr error) {
		var connection net.Conn
		var err error
		for attempt := 0; attempt < 4096; attempt++ {
			connection, err = (&net.Dialer{}).DialContext(ctx, "tcp", "10.0.0.1:7233")
			if err == nil {
				break
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			runtime.Gosched()
		}
		if err != nil {
			return err
		}
		defer func() { resultErr = errors.Join(resultErr, connection.Close()) }()
		started := time.Now()
		if _, err := connection.Write([]byte{'x'}); err != nil {
			return err
		}
		var response [1]byte
		if _, err := io.ReadFull(connection, response[:]); err != nil {
			return err
		}
		if response[0] != 'x' {
			return fmt.Errorf("response = %q", response)
		}
		if elapsed := time.Since(started); elapsed < 14*time.Millisecond {
			return fmt.Errorf("round trip elapsed = %s", elapsed)
		}
		return nil
	}))
	spec := Spec{
		Schema: SpecSchema, Backend: BackendProcess, Fidelity: FidelityHardIsolation, Seed: 93, Limits: DefaultLimits(),
		Nodes: []NodeSpec{
			{ID: "client", Boot: clientBoot, Address: "10.0.0.2"},
			{ID: "server", Boot: serverBoot, Address: "10.0.0.1"},
		},
		Links: []LinkSpec{
			{From: "client", To: "server", Enabled: true, DelayNanos: uint64(7 * time.Millisecond)},
			{From: "server", To: "client", Enabled: true, DelayNanos: uint64(7 * time.Millisecond)},
		},
	}
	result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
		server, err := cluster.Start(ctx, "server")
		if err != nil {
			return err
		}
		client, err := cluster.Start(ctx, "client")
		if err != nil {
			return err
		}
		clientTerminal, err := cluster.Wait(ctx, client)
		if err != nil {
			return err
		}
		if clientTerminal.State != NodeStateExited {
			return fmt.Errorf("client state = %s: %s", clientTerminal.State, clientTerminal.Reason)
		}
		serverTerminal, err := cluster.Wait(ctx, server)
		if err != nil {
			return err
		}
		if serverTerminal.State != NodeStateExited {
			return fmt.Errorf("server state = %s: %s", serverTerminal.State, serverTerminal.Reason)
		}
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, result.Outcome, result.Reason)
	require.Contains(t, networkDelays(result.Network.Transitions), uint64(7*time.Millisecond))
}

func TestProcessBackendRoutesListenThroughSharedHostModel(t *testing.T) {
	if !processBackendAvailable() {
		t.Skip("Runner simulation transport is unavailable")
	}
	bootID := uniqueBootID("cluster-process-listen")
	require.NoError(t, RegisterBoot(bootID, func(_ context.Context, node NodeContext) error {
		listener, err := net.Listen("tcp", net.JoinHostPort(node.Address, "7233"))
		if err != nil {
			return err
		}
		return listener.Close()
	}))
	spec := Spec{
		Schema: SpecSchema, Backend: BackendProcess, Fidelity: FidelityHardIsolation, Seed: 97, Limits: DefaultLimits(),
		Nodes: []NodeSpec{{ID: "server", Boot: bootID, Address: "10.0.0.1"}},
	}
	result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
		handle, err := cluster.Start(ctx, "server")
		if err != nil {
			return err
		}
		_, err = cluster.Wait(ctx, handle)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, result.Outcome, result.Reason)
	require.NotEmpty(t, result.Network.Transitions)
}

func TestProcessModelClosureWaitsForAdmittedOperationAndRejectsNewWork(t *testing.T) {
	handle := NodeHandle{Node: "server", Incarnation: 1}
	node := &clusterNode{state: NodeStateRunning, handle: handle, domain: 7}
	cluster := &inProcessCluster{nodes: map[NodeID]*clusterNode{"server": node}, activity: make(chan struct{}, 1)}
	domain, err := cluster.beginProcessModelOperation(handle)
	require.NoError(t, err)
	require.Equal(t, uint64(7), domain)

	cluster.mu.Lock()
	node.modelClosing = true
	cluster.mu.Unlock()
	_, err = cluster.beginProcessModelOperation(handle)
	require.ErrorIs(t, err, ErrInvalidTransition)

	done := make(chan struct{})
	go func() {
		cluster.waitForProcessModelOperations(node)
		close(done)
	}()
	select {
	case <-done:
		t.Fatal("model closure completed before the admitted operation")
	default:
	}
	cluster.finishProcessModelOperation(node)
	<-done
}

func TestProcessBackendCrashDrainsInflightModelOperationDeterministically(t *testing.T) {
	if !processBackendAvailable() {
		t.Skip("Runner simulation transport is unavailable")
	}
	bootID := uniqueBootID("cluster-process-crash-inflight")
	require.NoError(t, RegisterBoot(bootID, func(_ context.Context, node NodeContext) (resultErr error) {
		listener, err := net.Listen("tcp", net.JoinHostPort(node.Address, "7233"))
		if err != nil {
			return err
		}
		defer func() {
			if closeErr := listener.Close(); closeErr != nil && !errors.Is(closeErr, net.ErrClosed) {
				resultErr = errors.Join(resultErr, closeErr)
			}
		}()
		_, err = listener.Accept()
		return err
	}))
	spec := Spec{
		Schema: SpecSchema, Backend: BackendProcess, Fidelity: FidelityHardIsolation, Seed: 101, Limits: DefaultLimits(),
		Nodes: []NodeSpec{{ID: "server", Boot: bootID, Address: "10.0.0.1"}},
	}
	run := func() Result {
		result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
			handle, err := cluster.Start(ctx, "server")
			if err != nil {
				return err
			}
			concrete := cluster.(*inProcessCluster)
			for {
				concrete.mu.Lock()
				node := concrete.nodes[handle.Node]
				acceptActive := node.modelStarted >= 2 && node.modelActive != 0
				concrete.mu.Unlock()
				if acceptActive {
					break
				}
				select {
				case <-concrete.activity:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return cluster.Crash(ctx, handle)
		})
		require.NoError(t, err)
		require.Equal(t, OutcomeCompleted, result.Outcome, result.Reason)
		require.Equal(t, NodeStateCrashed, result.Nodes[0].State)
		require.NotEmpty(t, result.Network.Transitions)
		return result
	}
	first := run()
	second := run()
	require.Equal(t, first.Record.Identity, second.Record.Identity)
	require.Equal(t, first.Record.Transitions, second.Record.Transitions)
	require.Equal(t, first.Network, second.Network)
}

func TestProcessBackendModelDigestsIgnoreCompletionOrder(t *testing.T) {
	if !processBackendAvailable() {
		t.Skip("Runner simulation transport is unavailable")
	}
	bootID := uniqueBootID("cluster-process-completion-order")
	require.NoError(t, RegisterBoot(bootID, func(_ context.Context, node NodeContext) error {
		if err := os.MkdirAll("/data", 0o755); err != nil {
			return err
		}
		return os.WriteFile("/data/value", []byte(node.Node), 0o600)
	}))
	spec := Spec{
		Schema: SpecSchema, Backend: BackendProcess, Fidelity: FidelityHardIsolation, Seed: 103, Limits: DefaultLimits(),
		Nodes: []NodeSpec{
			{ID: "first", Boot: bootID, Address: "10.0.0.1", Volumes: []VolumeMount{{Volume: "first-data", Path: "/data"}}},
			{ID: "second", Boot: bootID, Address: "10.0.0.2", Volumes: []VolumeMount{{Volume: "second-data", Path: "/data"}}},
		},
		Volumes: []VolumeSpec{{ID: "first-data", CapacityBytes: 1 << 20}, {ID: "second-data", CapacityBytes: 1 << 20}},
	}
	run := func(reverse bool) Result {
		result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
			first, err := cluster.Start(ctx, "first")
			if err != nil {
				return err
			}
			second, err := cluster.Start(ctx, "second")
			if err != nil {
				return err
			}
			handles := []NodeHandle{first, second}
			if reverse {
				slices.Reverse(handles)
			}
			for _, handle := range handles {
				terminal, err := cluster.Wait(ctx, handle)
				if err != nil {
					return err
				}
				if terminal.State != NodeStateExited {
					return fmt.Errorf("node %q state = %s: %s", handle.Node, terminal.State, terminal.Reason)
				}
			}
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, OutcomeCompleted, result.Outcome, result.Reason)
		return result
	}
	forward := run(false)
	reverse := run(true)
	require.Equal(t, forward.Network, reverse.Network)
	require.Equal(t, forward.Volumes, reverse.Volumes)
}

func processNetworkPeer(peer string, value byte) BootFunc {
	return func(ctx context.Context, node NodeContext) (resultErr error) {
		listener, err := net.Listen("tcp", net.JoinHostPort(node.Address, "7233"))
		if err != nil {
			return err
		}
		defer func() {
			if closeErr := listener.Close(); closeErr != nil && !errors.Is(closeErr, net.ErrClosed) {
				resultErr = errors.Join(resultErr, closeErr)
			}
		}()
		accepted := make(chan error, 1)
		go func() {
			connection, acceptErr := listener.Accept()
			if acceptErr != nil {
				accepted <- acceptErr
				return
			}
			defer func() { accepted <- errors.Join(acceptErr, connection.Close()) }()
			var request [1]byte
			if _, acceptErr = io.ReadFull(connection, request[:]); acceptErr != nil {
				return
			}
			_, acceptErr = connection.Write([]byte{value})
		}()

		var connection net.Conn
		for attempt := 0; attempt < 4096; attempt++ {
			connection, err = (&net.Dialer{}).DialContext(ctx, "tcp", peer)
			if err == nil {
				break
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			runtime.Gosched()
		}
		if err != nil {
			return err
		}
		if _, err = connection.Write([]byte{value}); err != nil {
			return errors.Join(err, connection.Close())
		}
		var response [1]byte
		if _, err = io.ReadFull(connection, response[:]); err != nil {
			return errors.Join(err, connection.Close())
		}
		if err = connection.Close(); err != nil {
			return err
		}
		return <-accepted
	}
}

func openProcessDescriptors() []int {
	var descriptors []int
	for descriptor := 0; descriptor < 64; descriptor++ {
		var state syscall.Stat_t
		if syscall.Fstat(descriptor, &state) == nil {
			descriptors = append(descriptors, descriptor)
		}
	}
	return descriptors
}

func TestRunExecutesBootFunctionsConcurrently(t *testing.T) {
	firstBoot := uniqueBootID("cluster-concurrent-first")
	secondBoot := uniqueBootID("cluster-concurrent-second")
	entered := make(chan NodeHandle, 2)
	release := make(chan struct{})
	boot := func(ctx context.Context, node NodeContext) error {
		entered <- node.NodeHandle
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	require.NoError(t, RegisterBoot(firstBoot, boot))
	require.NoError(t, RegisterBoot(secondBoot, boot))

	spec := Spec{
		Schema:   SpecSchema,
		Backend:  BackendInProcess,
		Fidelity: FidelitySimulationModel,
		Seed:     23,
		Limits:   DefaultLimits(),
		Nodes: []NodeSpec{
			{ID: "first", Boot: firstBoot, Address: "10.0.0.1"},
			{ID: "second", Boot: secondBoot, Address: "10.0.0.2"},
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := Run(ctx, spec, func(ctx context.Context, cluster Cluster) error {
		first, err := cluster.Start(ctx, "first")
		if err != nil {
			return err
		}
		second, err := cluster.Start(ctx, "second")
		if err != nil {
			return err
		}
		seen := map[NodeHandle]bool{<-entered: true, <-entered: true}
		if !seen[first] || !seen[second] {
			return ErrInvalidTransition
		}
		close(release)
		if _, err := cluster.Wait(ctx, first); err != nil {
			return err
		}
		_, err = cluster.Wait(ctx, second)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, result.Outcome)
	require.Equal(t, []NodeResult{
		{Handle: NodeHandle{Node: "first", Incarnation: 1}, State: NodeStateExited},
		{Handle: NodeHandle{Node: "second", Incarnation: 1}, State: NodeStateExited},
	}, result.Nodes)
	require.Equal(t, []LifecycleTransition{
		{Ordinal: 0, Action: LifecycleStart, Handle: NodeHandle{Node: "first", Incarnation: 1}, From: NodeStateDefined, To: NodeStateRunning},
		{Ordinal: 1, Action: LifecycleStart, Handle: NodeHandle{Node: "second", Incarnation: 1}, From: NodeStateDefined, To: NodeStateRunning},
		{Ordinal: 2, Action: LifecycleWait, Handle: NodeHandle{Node: "first", Incarnation: 1}, From: NodeStateRunning, To: NodeStateExited},
		{Ordinal: 3, Action: LifecycleWait, Handle: NodeHandle{Node: "second", Incarnation: 1}, From: NodeStateRunning, To: NodeStateExited},
	}, result.Record.Transitions)
}

func TestRunRejectsCrashedIncarnationAfterRestart(t *testing.T) {
	bootID := uniqueBootID("cluster-restart")
	entered := make(chan NodeHandle, 2)
	releaseCrashed := make(chan struct{})
	require.NoError(t, RegisterBoot(bootID, func(ctx context.Context, node NodeContext) error {
		entered <- node.NodeHandle
		if node.Incarnation == 1 {
			<-releaseCrashed
			return nil
		}
		<-ctx.Done()
		return ctx.Err()
	}))
	spec := Spec{
		Schema:   SpecSchema,
		Backend:  BackendInProcess,
		Fidelity: FidelitySimulationModel,
		Seed:     29,
		Limits:   DefaultLimits(),
		Nodes:    []NodeSpec{{ID: "server", Boot: bootID, Address: "10.0.0.1"}},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := Run(ctx, spec, func(ctx context.Context, cluster Cluster) error {
		first, err := cluster.Start(ctx, "server")
		if err != nil {
			return err
		}
		require.Equal(t, first, <-entered)
		if err := cluster.Crash(ctx, first); err != nil {
			return err
		}
		second, err := cluster.Restart(ctx, "server")
		if err != nil {
			return err
		}
		require.Equal(t, NodeHandle{Node: "server", Incarnation: 2}, second)
		require.Equal(t, second, <-entered)
		require.ErrorIs(t, cluster.Stop(ctx, first), ErrStaleIncarnation)
		if err := cluster.Stop(ctx, second); err != nil {
			return err
		}
		close(releaseCrashed)
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, result.Outcome)
	require.Equal(t, []NodeResult{{Handle: NodeHandle{Node: "server", Incarnation: 2}, State: NodeStateStopped}}, result.Nodes)
	require.Equal(t, []LifecycleTransition{
		{Ordinal: 0, Action: LifecycleStart, Handle: NodeHandle{Node: "server", Incarnation: 1}, From: NodeStateDefined, To: NodeStateRunning},
		{Ordinal: 1, Action: LifecycleCrash, Handle: NodeHandle{Node: "server", Incarnation: 1}, From: NodeStateRunning, To: NodeStateCrashed},
		{Ordinal: 2, Action: LifecycleRestart, Handle: NodeHandle{Node: "server", Incarnation: 2}, From: NodeStateCrashed, To: NodeStateRunning},
		{Ordinal: 3, Action: LifecycleStop, Handle: NodeHandle{Node: "server", Incarnation: 2}, From: NodeStateRunning, To: NodeStateStopped},
	}, result.Record.Transitions)
}

func TestRunStopCapacityFailsBeforeCancellingBoot(t *testing.T) {
	bootID := uniqueBootID("cluster-stop-capacity")
	entered := make(chan struct{})
	cancelled := make(chan struct{})
	require.NoError(t, RegisterBoot(bootID, func(ctx context.Context, _ NodeContext) error {
		close(entered)
		<-ctx.Done()
		close(cancelled)
		return ctx.Err()
	}))
	limits := DefaultLimits()
	limits.ScenarioActions = 1
	spec := Spec{
		Schema:   SpecSchema,
		Backend:  BackendInProcess,
		Fidelity: FidelitySimulationModel,
		Seed:     31,
		Limits:   limits,
		Nodes:    []NodeSpec{{ID: "server", Boot: bootID, Address: "10.0.0.1"}},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := Run(ctx, spec, func(ctx context.Context, cluster Cluster) error {
		handle, err := cluster.Start(ctx, "server")
		if err != nil {
			return err
		}
		<-entered
		err = cluster.Stop(ctx, handle)
		var capacityErr *CapacityError
		require.ErrorAs(t, err, &capacityErr)
		require.Equal(t, "scenario_actions", capacityErr.Resource)
		select {
		case <-cancelled:
			t.Fatal("capacity rejection cancelled the boot function")
		default:
		}
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, []LifecycleTransition{{
		Ordinal: 0,
		Action:  LifecycleStart,
		Handle:  NodeHandle{Node: "server", Incarnation: 1},
		From:    NodeStateDefined,
		To:      NodeStateRunning,
	}}, result.Record.Transitions)
}

func TestRunRejectsCancelledLifecycleActionBeforeMutation(t *testing.T) {
	bootID := uniqueBootID("cluster-cancelled-action")
	called := make(chan struct{}, 1)
	require.NoError(t, RegisterBoot(bootID, func(context.Context, NodeContext) error {
		called <- struct{}{}
		return nil
	}))
	spec := Spec{
		Schema:   SpecSchema,
		Backend:  BackendInProcess,
		Fidelity: FidelitySimulationModel,
		Seed:     35,
		Limits:   DefaultLimits(),
		Nodes:    []NodeSpec{{ID: "server", Boot: bootID, Address: "10.0.0.1"}},
	}
	result, err := Run(context.Background(), spec, func(_ context.Context, cluster Cluster) error {
		cancelled, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := cluster.Start(cancelled, "server")
		require.ErrorIs(t, err, context.Canceled)
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, []NodeResult{{Handle: NodeHandle{Node: "server"}, State: NodeStateDefined}}, result.Nodes)
	require.Empty(t, result.Record.Transitions)
	select {
	case <-called:
		t.Fatal("cancelled lifecycle action invoked the boot function")
	default:
	}
}

func TestRunReturnsDetachedScenarioAndBootFailureEvidence(t *testing.T) {
	bootID := uniqueBootID("cluster-failure-evidence")
	require.NoError(t, RegisterBoot(bootID, func(context.Context, NodeContext) error {
		return errors.New("boot failed")
	}))
	spec := Spec{
		Schema:   SpecSchema,
		Backend:  BackendInProcess,
		Fidelity: FidelitySimulationModel,
		Seed:     37,
		Limits:   DefaultLimits(),
		Nodes:    []NodeSpec{{ID: "worker", Boot: bootID, Address: "10.0.0.1"}},
	}
	result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
		handle, err := cluster.Start(ctx, "worker")
		if err != nil {
			return err
		}
		node, err := cluster.Wait(ctx, handle)
		if err != nil {
			return err
		}
		require.Equal(t, NodeStateFailed, node.State)
		require.Equal(t, "boot failed", node.Reason)
		return errors.New("oracle failed")
	})
	require.NoError(t, err)
	require.Equal(t, OutcomeScenarioFailed, result.Outcome)
	require.Equal(t, "oracle failed", result.Reason)
	require.Equal(t, []NodeResult{{
		Handle: NodeHandle{Node: "worker", Incarnation: 1},
		State:  NodeStateFailed,
		Reason: "boot failed",
	}}, result.Nodes)
}

func TestRunProducesExactReplayableLifecycleRecord(t *testing.T) {
	bootID := uniqueBootID("cluster-record-replay")
	require.NoError(t, RegisterBoot(bootID, func(context.Context, NodeContext) error { return nil }))
	spec := Spec{
		Schema:   SpecSchema,
		Backend:  BackendInProcess,
		Fidelity: FidelitySimulationModel,
		Seed:     41,
		Limits:   DefaultLimits(),
		Nodes:    []NodeSpec{{ID: "actor", Boot: bootID, Address: "10.0.0.1"}},
	}
	scenario := func(ctx context.Context, cluster Cluster) error {
		handle, err := cluster.Start(ctx, "actor")
		if err != nil {
			return err
		}
		_, err = cluster.Wait(ctx, handle)
		return err
	}
	recorded, err := Run(context.Background(), spec, scenario)
	require.NoError(t, err)
	require.Equal(t, ClusterRecordSchema, recorded.Record.Schema)
	require.Regexp(t, `^sha256:[0-9a-f]{64}$`, recorded.Record.SpecSHA256)
	require.Regexp(t, `^sha256:[0-9a-f]{64}$`, recorded.Record.Identity)

	plan, err := ReplayPlanFor(recorded.Record)
	require.NoError(t, err)
	spec.Replay = &plan
	replayed, err := Run(context.Background(), spec, scenario)
	require.NoError(t, err)
	require.Equal(t, recorded, replayed)
}

func TestRunRejectsReplayDivergenceBeforeLifecycleMutation(t *testing.T) {
	bootID := uniqueBootID("cluster-replay-divergence")
	require.NoError(t, RegisterBoot(bootID, func(context.Context, NodeContext) error { return nil }))
	spec := Spec{
		Schema:   SpecSchema,
		Backend:  BackendInProcess,
		Fidelity: FidelitySimulationModel,
		Seed:     43,
		Limits:   DefaultLimits(),
		Nodes:    []NodeSpec{{ID: "actor", Boot: bootID, Address: "10.0.0.1"}},
	}
	scenario := func(ctx context.Context, cluster Cluster) error {
		handle, err := cluster.Start(ctx, "actor")
		if err != nil {
			return err
		}
		_, err = cluster.Wait(ctx, handle)
		return err
	}
	recorded, err := Run(context.Background(), spec, scenario)
	require.NoError(t, err)
	plan, err := ReplayPlanFor(recorded.Record)
	require.NoError(t, err)
	plan.Transitions[0] = LifecycleTransition{
		Ordinal: 0,
		Action:  LifecycleCrash,
		Handle:  NodeHandle{Node: "actor", Incarnation: 1},
		From:    NodeStateRunning,
		To:      NodeStateCrashed,
	}
	plan.Identity, err = replayPlanIdentity(plan)
	require.NoError(t, err)
	spec.Replay = &plan

	diverged, err := Run(context.Background(), spec, scenario)
	require.NoError(t, err)
	require.Equal(t, OutcomeReplayDiverged, diverged.Outcome)
	require.Equal(t, []NodeResult{{Handle: NodeHandle{Node: "actor"}, State: NodeStateDefined}}, diverged.Nodes)
	require.Empty(t, diverged.Record.Transitions)
	require.NotNil(t, diverged.Divergence)
	require.Equal(t, uint64(0), diverged.Divergence.Ordinal)
	require.Equal(t, LifecycleCrash, diverged.Divergence.Expected.Action)
	require.Equal(t, LifecycleStart, diverged.Divergence.Actual.Action)
}

func TestRunRejectsStopReplayDivergenceBeforeCancellation(t *testing.T) {
	bootID := uniqueBootID("cluster-stop-replay-divergence")
	started := make(chan struct{})
	cancelled := make(chan struct{})
	require.NoError(t, RegisterBoot(bootID, func(ctx context.Context, _ NodeContext) error {
		close(started)
		<-ctx.Done()
		close(cancelled)
		return ctx.Err()
	}))
	spec := Spec{
		Schema:   SpecSchema,
		Backend:  BackendInProcess,
		Fidelity: FidelitySimulationModel,
		Limits:   DefaultLimits(),
		Nodes:    []NodeSpec{{ID: "node", Boot: bootID, Address: "10.0.0.1"}},
	}
	plan := testReplayPlan(t, spec)
	plan.Nodes = []NodeResult{{Handle: NodeHandle{Node: "node", Incarnation: 1}, State: NodeStateCrashed}}
	plan.Transitions = []LifecycleTransition{
		{Ordinal: 0, Action: LifecycleStart, Handle: NodeHandle{Node: "node", Incarnation: 1}, From: NodeStateDefined, To: NodeStateRunning},
		{Ordinal: 1, Action: LifecycleCrash, Handle: NodeHandle{Node: "node", Incarnation: 1}, From: NodeStateRunning, To: NodeStateCrashed},
	}
	plan.Network = testNetworkRecord(t, spec, map[NodeID]uint64{"node": 1})
	var err error
	plan.Identity, err = replayPlanIdentity(plan)
	require.NoError(t, err)
	spec.Replay = &plan
	result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
		handle, err := cluster.Start(ctx, "node")
		if err != nil {
			return err
		}
		<-started
		err = cluster.Stop(ctx, handle)
		require.ErrorIs(t, err, ErrReplayDiverged)
		select {
		case <-cancelled:
			t.Fatal("replay-divergent stop cancelled the boot before returning")
		default:
		}
		return err
	})
	require.NoError(t, err)
	require.Equal(t, OutcomeReplayDiverged, result.Outcome)
	require.Len(t, result.Record.Transitions, 1)
}

func TestRunStopTerminalReplayDivergenceDoesNotCommitTransition(t *testing.T) {
	bootID := uniqueBootID("cluster-stop-terminal-divergence")
	started := make(chan struct{})
	require.NoError(t, RegisterBoot(bootID, func(ctx context.Context, _ NodeContext) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}))
	spec := Spec{
		Schema:   SpecSchema,
		Backend:  BackendInProcess,
		Fidelity: FidelitySimulationModel,
		Limits:   DefaultLimits(),
		Nodes:    []NodeSpec{{ID: "node", Boot: bootID, Address: "10.0.0.1"}},
	}
	plan := testReplayPlan(t, spec)
	plan.Nodes = []NodeResult{{Handle: NodeHandle{Node: "node", Incarnation: 1}, State: NodeStateFailed, Reason: "unexpected"}}
	plan.Transitions = []LifecycleTransition{
		{Ordinal: 0, Action: LifecycleStart, Handle: NodeHandle{Node: "node", Incarnation: 1}, From: NodeStateDefined, To: NodeStateRunning},
		{Ordinal: 1, Action: LifecycleStop, Handle: NodeHandle{Node: "node", Incarnation: 1}, From: NodeStateRunning, To: NodeStateFailed},
	}
	plan.Network = testNetworkRecord(t, spec, map[NodeID]uint64{"node": 1})
	var err error
	plan.Identity, err = replayPlanIdentity(plan)
	require.NoError(t, err)
	spec.Replay = &plan
	result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
		handle, err := cluster.Start(ctx, "node")
		if err != nil {
			return err
		}
		<-started
		err = cluster.Stop(ctx, handle)
		require.ErrorIs(t, err, ErrReplayDiverged)
		_, restartErr := cluster.Restart(ctx, "node")
		require.ErrorIs(t, restartErr, ErrReplayDiverged)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, OutcomeReplayDiverged, result.Outcome)
	require.Len(t, result.Record.Transitions, 1)
	require.Equal(t, NodeStateFailed, result.Divergence.Expected.To)
	require.Equal(t, NodeStateStopped, result.Divergence.Actual.To)
}

func TestRunRuntimeDomainIsInheritedAndRevoked(t *testing.T) {
	bootID := uniqueBootID("cluster-runtime-domain")
	hostnames := make(chan string, 2)
	afterCrash := make(chan struct{})
	stale := make(chan error, 1)
	staleOutput := make(chan error, 1)
	release := make(chan struct{})
	require.NoError(t, RegisterBoot(bootID, func(context.Context, NodeContext) error {
		hostname, err := os.Hostname()
		if err != nil {
			return err
		}
		hostnames <- hostname
		childReady := make(chan struct{})
		go func() {
			hostname, err := os.Hostname()
			if err == nil {
				hostnames <- hostname
			}
			close(childReady)
			<-afterCrash
			_, err = os.Hostname()
			stale <- err
			_, err = os.Stdout.Write([]byte("stale output"))
			staleOutput <- err
		}()
		<-childReady
		<-release
		return nil
	}))
	spec := Spec{
		Schema:   SpecSchema,
		Backend:  BackendInProcess,
		Fidelity: FidelitySimulationModel,
		Seed:     47,
		Limits:   DefaultLimits(),
		Nodes:    []NodeSpec{{ID: "server", Boot: bootID, Address: "10.0.0.1"}},
	}
	result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
		handle, err := cluster.Start(ctx, "server")
		if err != nil {
			return err
		}
		require.Equal(t, "server", <-hostnames)
		require.Equal(t, "server", <-hostnames)
		if err := cluster.Crash(ctx, handle); err != nil {
			return err
		}
		close(afterCrash)
		require.ErrorIs(t, <-stale, syscall.ESTALE)
		require.ErrorIs(t, <-staleOutput, syscall.ESTALE)
		close(release)
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, []NodeResult{{Handle: NodeHandle{Node: "server", Incarnation: 1}, State: NodeStateCrashed}}, result.Nodes)
	require.Equal(t, []LeakDiagnostic{{
		Handle: NodeHandle{Node: "server", Incarnation: 1},
		Kind:   LeakRevokedGoroutineMayRemain,
	}}, result.Record.Leaks)
}

func TestRunRecordsNodeLabelledOutputFromInheritedDomains(t *testing.T) {
	bootID := uniqueBootID("cluster-runtime-output")
	hostnames := make(chan string, 2)
	release := make(chan struct{})
	require.NoError(t, RegisterBoot(bootID, func(_ context.Context, node NodeContext) error {
		childDone := make(chan error, 1)
		go func() {
			hostname, err := os.Hostname()
			if err != nil {
				childDone <- err
				return
			}
			hostnames <- hostname
			_, err = os.Stdout.Write([]byte(fmt.Sprintf("%s-out", node.Node)))
			childDone <- err
		}()
		if _, err := os.Stderr.Write([]byte(fmt.Sprintf("%s-err", node.Node))); err != nil {
			return err
		}
		if err := <-childDone; err != nil {
			return err
		}
		<-release
		return nil
	}))
	spec := Spec{
		Schema:   SpecSchema,
		Backend:  BackendInProcess,
		Fidelity: FidelitySimulationModel,
		Seed:     53,
		Limits:   DefaultLimits(),
		Nodes: []NodeSpec{
			{ID: "first", Boot: bootID, Address: "10.0.0.1"},
			{ID: "second", Boot: bootID, Address: "10.0.0.2"},
		},
	}
	result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
		first, err := cluster.Start(ctx, "first")
		if err != nil {
			return err
		}
		second, err := cluster.Start(ctx, "second")
		if err != nil {
			return err
		}
		require.ElementsMatch(t, []string{"first", "second"}, []string{<-hostnames, <-hostnames})
		close(release)
		if _, err := cluster.Wait(ctx, first); err != nil {
			return err
		}
		_, err = cluster.Wait(ctx, second)
		return err
	})
	require.NoError(t, err)
	expected := []OutputObservation{
		expectedOutput("first", 1, OutputStdout, []byte("first-out")),
		expectedOutput("first", 1, OutputStderr, []byte("first-err")),
		expectedOutput("second", 1, OutputStdout, []byte("second-out")),
		expectedOutput("second", 1, OutputStderr, []byte("second-err")),
	}
	require.Equal(t, expected, result.Outputs)
	require.Equal(t, expected, result.Record.Outputs)
}

func expectedOutput(node NodeID, incarnation uint64, stream OutputStream, data []byte) OutputObservation {
	digest := sha256.Sum256(data)
	return OutputObservation{
		Handle:        NodeHandle{Node: node, Incarnation: incarnation},
		Stream:        stream,
		Bytes:         data,
		FullSHA256:    fmt.Sprintf("sha256:%x", digest),
		TotalBytes:    uint64(len(data)),
		RetainedBytes: uint64(len(data)),
	}
}

func TestRunBoundsRetainedOutputButPreservesFullHashAndCount(t *testing.T) {
	bootID := uniqueBootID("cluster-runtime-output-bound")
	require.NoError(t, RegisterBoot(bootID, func(context.Context, NodeContext) error {
		_, err := os.Stdout.Write([]byte("abcdef"))
		return err
	}))
	limits := DefaultLimits()
	limits.ScenarioActions = 2
	limits.ObservationBytes = 8
	spec := Spec{
		Schema:   SpecSchema,
		Backend:  BackendInProcess,
		Fidelity: FidelitySimulationModel,
		Seed:     73,
		Limits:   limits,
		Nodes:    []NodeSpec{{ID: "node", Boot: bootID, Address: "10.0.0.1"}},
	}
	result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
		handle, err := cluster.Start(ctx, "node")
		if err != nil {
			return err
		}
		_, err = cluster.Wait(ctx, handle)
		return err
	})
	require.NoError(t, err)
	digest := sha256.Sum256([]byte("abcdef"))
	require.Equal(t, []OutputObservation{{
		Handle:         NodeHandle{Node: "node", Incarnation: 1},
		Stream:         OutputStdout,
		Bytes:          []byte("ab"),
		FullSHA256:     fmt.Sprintf("sha256:%x", digest),
		TotalBytes:     6,
		RetainedBytes:  2,
		DiscardedBytes: 4,
		Truncated:      true,
	}}, result.Outputs)
}

func TestRunExactlyReplaysOutputAndClassifiesTerminalDivergence(t *testing.T) {
	bootID := uniqueBootID("cluster-runtime-output-replay")
	require.NoError(t, RegisterBoot(bootID, func(context.Context, NodeContext) error {
		_, err := os.Stdout.Write([]byte("same"))
		return err
	}))
	spec := Spec{
		Schema:   SpecSchema,
		Backend:  BackendInProcess,
		Fidelity: FidelitySimulationModel,
		Seed:     89,
		Limits:   DefaultLimits(),
		Nodes:    []NodeSpec{{ID: "node", Boot: bootID, Address: "10.0.0.1"}},
	}
	scenario := func(ctx context.Context, cluster Cluster) error {
		handle, err := cluster.Start(ctx, "node")
		if err != nil {
			return err
		}
		_, err = cluster.Wait(ctx, handle)
		return err
	}
	recorded, err := Run(context.Background(), spec, scenario)
	require.NoError(t, err)
	plan, err := ReplayPlanFor(recorded.Record)
	require.NoError(t, err)
	spec.Replay = &plan
	replayed, err := Run(context.Background(), spec, scenario)
	require.NoError(t, err)
	require.Equal(t, recorded, replayed)

	plan.Outputs[0] = expectedOutput("node", 1, OutputStdout, []byte("fame"))
	plan.Identity, err = replayPlanIdentity(plan)
	require.NoError(t, err)
	spec.Replay = &plan
	diverged, err := Run(context.Background(), spec, scenario)
	require.NoError(t, err)
	require.Equal(t, OutcomeReplayDiverged, diverged.Outcome)
	require.NotNil(t, diverged.Divergence)
	require.Equal(t, ReplayDimensionTerminal, diverged.Divergence.Dimension)
	require.NotEqual(t, diverged.Divergence.ExpectedSHA256, diverged.Divergence.ActualSHA256)
}

func TestRunWaitRetainsBootCompletionAfterCapacityRejection(t *testing.T) {
	bootID := uniqueBootID("cluster-wait-capacity-retry")
	released := make(chan struct{})
	require.NoError(t, RegisterBoot(bootID, func(context.Context, NodeContext) error {
		close(released)
		return nil
	}))
	limits := DefaultLimits()
	limits.ScenarioActions = 1
	spec := Spec{
		Schema:   SpecSchema,
		Backend:  BackendInProcess,
		Fidelity: FidelitySimulationModel,
		Limits:   limits,
		Nodes:    []NodeSpec{{ID: "node", Boot: bootID, Address: "10.0.0.1"}},
	}
	result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
		handle, err := cluster.Start(ctx, "node")
		if err != nil {
			return err
		}
		<-released
		_, err = cluster.Wait(ctx, handle)
		require.ErrorIs(t, err, ErrCapacity)
		retryCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_, err = cluster.Wait(retryCtx, handle)
		require.ErrorIs(t, err, ErrCapacity)
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, result.Outcome)
}

func TestRunWaitingForOneNodeDoesNotBlockStoppingAnother(t *testing.T) {
	firstBoot := uniqueBootID("cluster-independent-wait")
	secondBoot := uniqueBootID("cluster-independent-stop")
	firstRelease := make(chan struct{})
	secondStarted := make(chan struct{})
	require.NoError(t, RegisterBoot(firstBoot, func(context.Context, NodeContext) error {
		<-firstRelease
		return nil
	}))
	require.NoError(t, RegisterBoot(secondBoot, func(ctx context.Context, _ NodeContext) error {
		close(secondStarted)
		<-ctx.Done()
		return ctx.Err()
	}))
	spec := Spec{
		Schema:   SpecSchema,
		Backend:  BackendInProcess,
		Fidelity: FidelitySimulationModel,
		Limits:   DefaultLimits(),
		Nodes: []NodeSpec{
			{ID: "first", Boot: firstBoot, Address: "10.0.0.1"},
			{ID: "second", Boot: secondBoot, Address: "10.0.0.2"},
		},
	}
	result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
		first, err := cluster.Start(ctx, "first")
		if err != nil {
			return err
		}
		second, err := cluster.Start(ctx, "second")
		if err != nil {
			return err
		}
		<-secondStarted
		waitResult := make(chan error, 1)
		go func() {
			_, waitErr := cluster.Wait(ctx, first)
			waitResult <- waitErr
		}()
		stopCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		require.NoError(t, cluster.Stop(stopCtx, second))
		close(firstRelease)
		return <-waitResult
	})
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, result.Outcome)
}

func TestRunAdmittedStopCommitsAfterCallerDeadline(t *testing.T) {
	bootID := uniqueBootID("cluster-stop-timeout")
	started := make(chan struct{})
	cancelled := make(chan struct{})
	release := make(chan struct{})
	require.NoError(t, RegisterBoot(bootID, func(ctx context.Context, _ NodeContext) error {
		close(started)
		<-ctx.Done()
		close(cancelled)
		<-release
		return ctx.Err()
	}))
	spec := Spec{
		Schema:   SpecSchema,
		Backend:  BackendInProcess,
		Fidelity: FidelitySimulationModel,
		Limits:   DefaultLimits(),
		Nodes:    []NodeSpec{{ID: "node", Boot: bootID, Address: "10.0.0.1"}},
	}
	result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
		handle, err := cluster.Start(ctx, "node")
		if err != nil {
			return err
		}
		<-started
		stopCtx, cancel := context.WithTimeout(ctx, time.Millisecond)
		defer cancel()
		stopResult := make(chan error, 1)
		go func() { stopResult <- cluster.Stop(stopCtx, handle) }()
		<-cancelled
		<-stopCtx.Done()
		select {
		case err := <-stopResult:
			t.Fatalf("admitted Stop returned before its terminal commit: %v", err)
		default:
		}
		close(release)
		return <-stopResult
	})
	require.NoError(t, err)
	require.Equal(t, []NodeResult{{Handle: NodeHandle{Node: "node", Incarnation: 1}, State: NodeStateStopped}}, result.Nodes)
}

func TestRunCleanupWaitsForBootAndPublishesTerminalIncarnation(t *testing.T) {
	bootID := uniqueBootID("cluster-cleanup-wait")
	started := make(chan struct{})
	finished := make(chan struct{})
	require.NoError(t, RegisterBoot(bootID, func(ctx context.Context, _ NodeContext) error {
		close(started)
		<-ctx.Done()
		close(finished)
		return ctx.Err()
	}))
	spec := Spec{
		Schema:   SpecSchema,
		Backend:  BackendInProcess,
		Fidelity: FidelitySimulationModel,
		Limits:   DefaultLimits(),
		Nodes:    []NodeSpec{{ID: "node", Boot: bootID, Address: "10.0.0.1"}},
	}
	result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
		_, err := cluster.Start(ctx, "node")
		if err != nil {
			return err
		}
		<-started
		return nil
	})
	require.NoError(t, err)
	select {
	case <-finished:
	default:
		t.Fatal("Run returned before boot cleanup completed")
	}
	require.Equal(t, []NodeResult{{Handle: NodeHandle{Node: "node", Incarnation: 1}, State: NodeStateStopped}}, result.Nodes)
	require.Equal(t, result.Nodes, result.Record.Nodes)
}

func TestRunRecordsEveryTerminalIncarnation(t *testing.T) {
	bootID := uniqueBootID("cluster-incarnation-history")
	firstRelease := make(chan struct{})
	require.NoError(t, RegisterBoot(bootID, func(ctx context.Context, node NodeContext) error {
		if node.Incarnation == 1 {
			<-firstRelease
			return nil
		}
		<-ctx.Done()
		return ctx.Err()
	}))
	spec := Spec{
		Schema:   SpecSchema,
		Backend:  BackendInProcess,
		Fidelity: FidelitySimulationModel,
		Limits:   DefaultLimits(),
		Nodes:    []NodeSpec{{ID: "node", Boot: bootID, Address: "10.0.0.1"}},
	}
	result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
		first, err := cluster.Start(ctx, "node")
		if err != nil {
			return err
		}
		if err := cluster.Crash(ctx, first); err != nil {
			return err
		}
		second, err := cluster.Restart(ctx, "node")
		if err != nil {
			return err
		}
		if err := cluster.Stop(ctx, second); err != nil {
			return err
		}
		close(firstRelease)
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, []NodeResult{
		{Handle: NodeHandle{Node: "node", Incarnation: 1}, State: NodeStateCrashed},
		{Handle: NodeHandle{Node: "node", Incarnation: 2}, State: NodeStateStopped},
	}, result.Record.Nodes)
}

func TestRunBoundsScenarioAndBootFailureReasons(t *testing.T) {
	bootID := uniqueBootID("cluster-bounded-reasons")
	require.NoError(t, RegisterBoot(bootID, func(context.Context, NodeContext) error {
		return errors.New(string(make([]byte, MaximumTerminalReasonBytes+1)))
	}))
	spec := Spec{
		Schema:   SpecSchema,
		Backend:  BackendInProcess,
		Fidelity: FidelitySimulationModel,
		Limits:   DefaultLimits(),
		Nodes:    []NodeSpec{{ID: "node", Boot: bootID, Address: "10.0.0.1"}},
	}
	result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
		handle, err := cluster.Start(ctx, "node")
		if err != nil {
			return err
		}
		if _, err := cluster.Wait(ctx, handle); err != nil {
			return err
		}
		return errors.New(string(make([]byte, MaximumTerminalReasonBytes+1)))
	})
	require.NoError(t, err)
	require.LessOrEqual(t, len(result.Reason), MaximumTerminalReasonBytes)
	require.LessOrEqual(t, len(result.Record.Nodes[0].Reason), MaximumTerminalReasonBytes)
	_, err = EncodeClusterRecord(result.Record)
	require.NoError(t, err)
}
