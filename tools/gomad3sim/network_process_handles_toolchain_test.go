//go:build gomad3_toolchain

package gomad3sim

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestProcessNetworkHandleCapacityPreservesIdentities(t *testing.T) {
	runProcessNetworkHandleCapacity(t, "listeners")
}
func TestProcessNetworkHandleConnectionCapacity(t *testing.T) {
	runProcessNetworkHandleCapacity(t, "connections")
}
func TestProcessNetworkHandleDeliveryCapacity(t *testing.T) {
	runProcessNetworkHandleCapacity(t, "deliveries")
}
func TestProcessNetworkHandleByteCapacity(t *testing.T) { runProcessNetworkHandleCapacity(t, "bytes") }
func runProcessNetworkHandleCapacity(t *testing.T, mode string) {
	t.Helper()
	if !processBackendAvailable() {
		t.Skip("Runner simulation transport is unavailable")
	}

	serverBoot := uniqueBootID("process-handle-capacity-server-" + mode)
	clientBoot := uniqueBootID("process-handle-capacity-client-" + mode)
	require.NoError(t, RegisterBoot(serverBoot, func(ctx context.Context, node NodeContext) error {
		listener, err := net.Listen("tcp4", net.JoinHostPort(node.Address, "7233"))
		if err != nil {
			return err
		}
		defer listener.Close()
		if mode == "listeners" {
			second, err := net.Listen("tcp4", net.JoinHostPort(node.Address, "0"))
			if second != nil {
				second.Close()
				return errors.New("listener capacity admitted second bind")
			}
			if err == nil || !strings.Contains(err.Error(), "network resources exhausted") {
				return fmt.Errorf("listener capacity=%v", err)
			}
		}
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		defer conn.Close()
		if mode == "deliveries" {
			// Keep the first byte queued until the client's failed second write.
			<-time.After(10 * time.Millisecond)
		}
		buffer := make([]byte, 1)
		if n, err := conn.Read(buffer); n != 1 || err != nil || buffer[0] != 'x' {
			return fmt.Errorf("capacity data=%d,%v,%q", n, err, buffer)
		}
		if _, err := conn.Write([]byte{'r'}); err != nil {
			return err
		}
		<-ctx.Done()
		return ctx.Err()
	}))
	require.NoError(t, RegisterBoot(clientBoot, func(ctx context.Context, _ NodeContext) error {
		conn, err := dialNetworkHandlePeer(ctx)
		if err != nil {
			return err
		}
		defer conn.Close()
		if mode == "connections" {
			second, err := (&net.Dialer{}).DialContext(ctx, "tcp4", "10.0.0.1:7233")
			if second != nil {
				second.Close()
				return errors.New("connection capacity admitted second dial")
			}
			if err == nil || !strings.Contains(err.Error(), "network resources exhausted") {
				return fmt.Errorf("connection capacity=%v", err)
			}
		}
		if mode == "bytes" {
			if n, err := conn.Write([]byte("xx")); n != 0 || err == nil || !strings.Contains(err.Error(), "network resources exhausted") {
				return fmt.Errorf("byte capacity=%d,%v", n, err)
			}
		}
		if _, err := conn.Write([]byte{'x'}); err != nil {
			return err
		}
		if mode == "deliveries" {
			if n, err := conn.Write([]byte{'y'}); n != 0 || err == nil || !strings.Contains(err.Error(), "network resources exhausted") {
				return fmt.Errorf("delivery capacity=%d,%v", n, err)
			}
		}
		buffer := make([]byte, 1)
		if _, err := io.ReadFull(conn, buffer); err != nil || buffer[0] != 'r' {
			return fmt.Errorf("capacity reply=%q,%v", buffer, err)
		}
		return nil
	}))
	spec := twoNodeNetworkSpec(serverBoot, clientBoot)
	spec.Backend = BackendProcess
	spec.Fidelity = FidelityHardIsolation
	switch mode {
	case "listeners":
		spec.Limits.NetworkListeners = 1
	case "connections":
		spec.Limits.NetworkConnections = 1
	case "deliveries":
		spec.Limits.NetworkDeliveries = 1
	case "bytes":
		spec.Limits.NetworkBytes = 1
	}
	run := func(spec Spec) Result {
		result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
			server, err := cluster.Start(ctx, "server")
			if err != nil {
				return err
			}
			if err := waitNetworkHandleModelRead(ctx, cluster, server, 2); err != nil {
				return err
			}
			client, err := cluster.Start(ctx, "client")
			if err != nil {
				return err
			}
			if err := waitNetworkHandleExit(ctx, cluster, client); err != nil {
				return err
			}
			return cluster.Stop(ctx, server)
		})
		require.NoError(t, err)
		require.Equal(t, OutcomeCompleted, result.Outcome, result.Reason)
		return result
	}
	first := run(spec)
	require.Equal(t, uint64(1), first.Network.Snapshot.NextConnection)
	require.Equal(t, uint64(2), first.Network.Snapshot.NextDelivery)
	for _, node := range first.Network.Snapshot.Nodes {
		if node.Node == "client" {
			require.Equal(t, uint64(40001), node.NextClientPort)
		}
		if node.Node == "server" {
			require.Equal(t, uint64(20000), node.NextListenerPort)
		}
	}
	plan, err := ReplayPlanFor(first.Record)
	require.NoError(t, err)
	spec.Replay = &plan
	second := run(spec)
	require.Equal(t, first.Record.Identity, second.Record.Identity)

}

func TestProcessNetworkHandleRevocationDropsDelayedData(t *testing.T) {
	runProcessNetworkHandleRevocation(t, true)
}
func TestProcessNetworkHandleCrashDropsDelayedData(t *testing.T) {
	runProcessNetworkHandleRevocation(t, false)
}
func runProcessNetworkHandleRevocation(t *testing.T, graceful bool) {
	t.Helper()
	if !processBackendAvailable() {
		t.Skip("Runner simulation transport is unavailable")
	}

	serverBoot := uniqueBootID(fmt.Sprintf("process-handle-revoke-server-%t", graceful))
	clientBoot := uniqueBootID(fmt.Sprintf("process-handle-revoke-client-%t", graceful))
	require.NoError(t, RegisterBoot(serverBoot, func(ctx context.Context, node NodeContext) error {
		listener, err := net.Listen("tcp4", net.JoinHostPort(node.Address, "7233"))
		if err != nil {
			return err
		}
		defer listener.Close()
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		defer conn.Close()
		if _, err := conn.Write([]byte{'r'}); err != nil {
			return err
		}
		if graceful {
			<-ctx.Done()
			return ctx.Err()
		}
		if n, err := conn.Read(make([]byte, 1)); n != 0 || !errors.Is(err, net.ErrClosed) {
			return fmt.Errorf("crashed peer read=%d,%v", n, err)
		}
		second, err := listener.Accept()
		if err != nil {
			return err
		}
		defer second.Close()
		buffer := make([]byte, 1)
		if _, err := io.ReadFull(second, buffer); err != nil || buffer[0] != 'z' {
			return fmt.Errorf("restart data=%q,%v", buffer, err)
		}
		if _, err := second.Write(buffer); err != nil {
			return err
		}
		<-ctx.Done()
		return ctx.Err()
	}))
	require.NoError(t, RegisterBoot(clientBoot, func(ctx context.Context, node NodeContext) error {
		conn, err := dialNetworkHandlePeer(ctx)
		if err != nil {
			return err
		}
		defer conn.Close()
		buffer := make([]byte, 1)
		if node.Incarnation == 2 {
			if _, err := conn.Write([]byte{'z'}); err != nil {
				return err
			}
			if _, err := io.ReadFull(conn, buffer); err != nil || buffer[0] != 'z' {
				return fmt.Errorf("restart reply=%q,%v", buffer, err)
			}
			return nil
		}
		if _, err := io.ReadFull(conn, buffer); err != nil {
			return err
		}
		if graceful {
			if n, err := conn.Read(buffer); n != 0 || err != io.EOF {
				return fmt.Errorf("graceful EOF=%d,%v", n, err)
			}
			return nil
		}
		if _, err := conn.Write([]byte{'x'}); err != nil {
			return err
		}
		_, err = conn.Read(buffer)
		return err
	}))
	spec := twoNodeNetworkSpec(serverBoot, clientBoot)
	spec.Backend = BackendProcess
	spec.Fidelity = FidelityHardIsolation
	if !graceful {
		spec.Links[0].DelayNanos = uint64(time.Hour)
	}
	run := func(spec Spec) Result {
		result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
			server, err := cluster.Start(ctx, "server")
			if err != nil {
				return err
			}
			if err := waitNetworkHandleModelRead(ctx, cluster, server, 2); err != nil {
				return err
			}
			client, err := cluster.Start(ctx, "client")
			if err != nil {
				return err
			}
			if graceful {
				if err := waitNetworkHandleModelRead(ctx, cluster, client, 3); err != nil {
					return err
				}
				if err := cluster.Stop(ctx, server); err != nil {
					return err
				}
				return waitNetworkHandleExit(ctx, cluster, client)
			}
			if err := waitNetworkHandleModelRead(ctx, cluster, server, 4); err != nil {
				return err
			}
			if err := waitNetworkHandleModelRead(ctx, cluster, client, 4); err != nil {
				return err
			}
			if err := cluster.Crash(ctx, client); err != nil {
				return err
			}
			if err := cluster.SetDelay(ctx, "client", "server", 0); err != nil {
				return err
			}
			restarted, err := cluster.Restart(ctx, "client")
			if err != nil {
				return err
			}
			if err := waitNetworkHandleExit(ctx, cluster, restarted); err != nil {
				return err
			}
			return cluster.Stop(ctx, server)
		})
		require.NoError(t, err)
		require.Equal(t, OutcomeCompleted, result.Outcome, result.Reason)
		return result
	}
	first := run(spec)
	require.Empty(t, first.Network.Snapshot.Deliveries)
	if !graceful {
		require.Equal(t, uint64(2), first.Network.Snapshot.NextConnection)
		require.True(t, first.Network.Snapshot.Connections[0].Reset)
	}
	plan, err := ReplayPlanFor(first.Record)
	require.NoError(t, err)
	spec.Replay = &plan
	second := run(spec)
	require.Equal(t, first.Record.Identity, second.Record.Identity)

}

func waitNetworkHandleExit(ctx context.Context, cluster Cluster, handle NodeHandle) error {
	terminal, err := cluster.Wait(ctx, handle)
	if err != nil {
		return err
	}
	if terminal.State != NodeStateExited {
		return fmt.Errorf("node %s finished as %s: %s", handle.Node, terminal.State, terminal.Reason)
	}
	return nil
}

func waitNetworkHandleModelRead(ctx context.Context, cluster Cluster, handle NodeHandle, minimum uint64) error {
	concrete := cluster.(*inProcessCluster)
	for {
		concrete.mu.Lock()
		node := concrete.nodes[handle.Node]
		ready := node.modelStarted >= minimum && node.modelActive != 0
		concrete.mu.Unlock()
		if ready {
			return nil
		}
		select {
		case <-concrete.activity:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func TestProcessNetworkHandleReplayRejectsWriteBeforeMutation(t *testing.T) {
	if !processBackendAvailable() {
		t.Skip("Runner simulation transport is unavailable")
	}
	serverBoot := uniqueBootID("process-handle-replay-server")
	clientBoot := uniqueBootID("process-handle-replay-client")
	require.NoError(t, RegisterBoot(serverBoot, func(ctx context.Context, node NodeContext) error {
		listener, err := net.Listen("tcp4", net.JoinHostPort(node.Address, "7233"))
		if err != nil {
			return err
		}
		defer listener.Close()
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		if _, err := conn.Write([]byte{'r'}); err != nil {
			return err
		}
		<-ctx.Done()
		return ctx.Err()
	}))
	require.NoError(t, RegisterBoot(clientBoot, func(ctx context.Context, _ NodeContext) error {
		conn, err := dialNetworkHandlePeer(ctx)
		if err != nil {
			return err
		}
		if _, err := io.ReadFull(conn, make([]byte, 1)); err != nil {
			return err
		}
		if _, err := conn.Write([]byte{'x'}); err != nil {
			return errors.New("network handle replay write rejected")
		}
		return conn.Close()
	}))
	spec := twoNodeNetworkSpec(serverBoot, clientBoot)
	spec.Backend = BackendProcess
	spec.Fidelity = FidelityHardIsolation
	run := func(spec Spec) Result {
		result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
			server, err := cluster.Start(ctx, "server")
			if err != nil {
				return err
			}
			if err := waitNetworkHandleModelRead(ctx, cluster, server, 2); err != nil {
				return err
			}
			client, err := cluster.Start(ctx, "client")
			if err != nil {
				return err
			}
			if err := waitNetworkHandleExit(ctx, cluster, client); err != nil {
				return err
			}
			return cluster.Stop(ctx, server)
		})
		require.NoError(t, err)
		return result
	}
	first := run(spec)
	require.Equal(t, OutcomeCompleted, first.Outcome, first.Reason)
	require.Equal(t, uint64(2), first.Network.Snapshot.NextDelivery)
	plan, err := ReplayPlanFor(first.Record)
	require.NoError(t, err)
	changed := false
	for index := range plan.Network.Transitions {
		transition := &plan.Network.Transitions[index]
		if transition.Kind == NetworkWrite && transition.Source.Node == "client" {
			transition.PayloadSHA256 = fmt.Sprintf("sha256:%x", sha256.Sum256([]byte{'y'}))
			changed = true
			break
		}
	}
	require.True(t, changed)
	// Rejected writes leave no queued bytes for the owner's terminal revoke.
	// Admit that cleanup shape while requiring the original Write divergence.
	for index := range plan.Network.Transitions {
		transition := &plan.Network.Transitions[index]
		if transition.Kind == NetworkStop && transition.Source.Node == "client" {
			transition.Bytes = 0
		}
	}
	encoded, err := encodeRuntimeNetworkTransitions(plan.Network.Transitions)
	require.NoError(t, err)
	plan.Network.Snapshot.TransitionSHA256 = fmt.Sprintf("sha256:%x", sha256.Sum256(append([]byte("gomad3-simulation-network-transitions/v2\x00"), encoded...)))
	plan.Network.Snapshot.Identity, err = networkSnapshotIdentity(plan.Network.Snapshot)
	require.NoError(t, err)
	// The intentionally rejected write fails this node. Admit that terminal
	// transition so lifecycle validation cannot replace the network finding.
	for index := range plan.Transitions {
		transition := &plan.Transitions[index]
		if transition.Action == LifecycleWait && transition.Handle.Node == "client" {
			transition.To = NodeStateFailed
		}
	}
	for index := range plan.Nodes {
		node := &plan.Nodes[index]
		if node.Handle.Node == "client" {
			node.State = NodeStateFailed
			node.Reason = "network handle replay write rejected"
		}
	}
	plan.Identity, err = replayPlanIdentity(plan)
	require.NoError(t, err)
	spec.Replay = &plan
	second := run(spec)
	require.Equal(t, OutcomeReplayDiverged, second.Outcome, second.Reason)
	require.Equal(t, ReplayDimensionNetwork, second.Divergence.Dimension)
	require.Equal(t, NetworkWrite, second.Divergence.ActualNetwork.Kind)
	require.Equal(t, uint64(1), second.Network.Snapshot.NextDelivery)
	require.Empty(t, second.Network.Snapshot.Deliveries)
}
