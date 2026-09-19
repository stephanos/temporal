//go:build gomad3_toolchain

package gomad3sim

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNetworkPartitionTimeoutHealReconnect(t *testing.T) {
	serverBoot := uniqueBootID("network-partition-server")
	clientBoot := uniqueBootID("network-partition-client")
	serverReady := make(chan struct{}, 2)
	clientCommands := make(chan chan error)
	require.NoError(t, RegisterBoot(serverBoot, func(ctx context.Context, node NodeContext) error {
		listener, err := net.Listen("tcp4", net.JoinHostPort(node.Address, "7233"))
		if err != nil {
			return err
		}
		closed := make(chan error, 1)
		go func() {
			<-ctx.Done()
			closed <- listener.Close()
		}()
		serverReady <- struct{}{}
		for {
			connection, err := listener.Accept()
			if err != nil {
				if ctx.Err() != nil {
					return errors.Join(ctx.Err(), <-closed)
				}
				return errors.Join(err, listener.Close())
			}
			request := make([]byte, 1)
			if _, err := io.ReadFull(connection, request); err != nil {
				connection.Close()
				return err
			}
			if _, err := connection.Write(request); err != nil {
				connection.Close()
				return err
			}
			if err := connection.Close(); err != nil {
				return err
			}
		}
	}))
	require.NoError(t, RegisterBoot(clientBoot, func(ctx context.Context, _ NodeContext) error {
		for {
			select {
			case result := <-clientCommands:
				dialCtx, cancel := context.WithTimeout(ctx, time.Millisecond)
				connection, err := (&net.Dialer{}).DialContext(dialCtx, "tcp4", "10.0.0.1:7233")
				cancel()
				if err == nil {
					if _, err = connection.Write([]byte{'x'}); err == nil {
						response := make([]byte, 1)
						_, err = io.ReadFull(connection, response)
						if err == nil && response[0] != 'x' {
							err = fmt.Errorf("response = %q", response)
						}
					}
					err = errors.Join(err, connection.Close())
				}
				result <- err
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}))
	spec := twoNodeNetworkSpec(serverBoot, clientBoot)
	result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
		server, err := cluster.Start(ctx, "server")
		if err != nil {
			return err
		}
		<-serverReady
		client, err := cluster.Start(ctx, "client")
		if err != nil {
			return err
		}
		if err := cluster.Partition(ctx, "client", "server"); err != nil {
			return err
		}
		attempt := make(chan error, 1)
		clientCommands <- attempt
		if err := <-attempt; !errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("partitioned dial error = %v", err)
		}
		if err := cluster.Heal(ctx, "client", "server"); err != nil {
			return err
		}
		clientCommands <- attempt
		if err := <-attempt; err != nil {
			return err
		}
		if err := cluster.Stop(ctx, client); err != nil {
			return err
		}
		return cluster.Stop(ctx, server)
	})
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, result.Outcome)
	require.NotEmpty(t, result.Network.Transitions)
	require.NotEmpty(t, result.Network.Snapshot.TransitionSHA256)
}

func TestNetworkFixedDelayIsRecordedAndExactlyReplayable(t *testing.T) {
	serverBoot := uniqueBootID("network-delay-server")
	clientBoot := uniqueBootID("network-delay-client")
	serverReady := make(chan struct{}, 2)
	require.NoError(t, RegisterBoot(serverBoot, func(ctx context.Context, node NodeContext) error {
		listener, err := net.Listen("tcp4", net.JoinHostPort(node.Address, "7233"))
		if err != nil {
			return err
		}
		defer listener.Close()
		serverReady <- struct{}{}
		connection, err := listener.Accept()
		if err != nil {
			return err
		}
		defer connection.Close()
		_, err = io.Copy(connection, connection)
		return err
	}))
	require.NoError(t, RegisterBoot(clientBoot, func(ctx context.Context, _ NodeContext) error {
		<-serverReady
		connection, err := (&net.Dialer{}).DialContext(ctx, "tcp4", "10.0.0.1:7233")
		if err != nil {
			return err
		}
		defer connection.Close()
		started := time.Now()
		if _, err := connection.Write([]byte("x")); err != nil {
			return err
		}
		response := make([]byte, 1)
		if _, err := io.ReadFull(connection, response); err != nil {
			return err
		}
		if elapsed := time.Since(started); elapsed < 14*time.Millisecond {
			return fmt.Errorf("round trip elapsed = %s", elapsed)
		}
		return nil
	}))
	spec := twoNodeNetworkSpec(serverBoot, clientBoot)
	for index := range spec.Links {
		spec.Links[index].DelayNanos = uint64(7 * time.Millisecond)
	}
	scenario := func(ctx context.Context, cluster Cluster) error {
		server, err := cluster.Start(ctx, "server")
		if err != nil {
			return err
		}
		client, err := cluster.Start(ctx, "client")
		if err != nil {
			return err
		}
		if _, err := cluster.Wait(ctx, client); err != nil {
			return err
		}
		return cluster.Stop(ctx, server)
	}
	var recorded Result
	for iteration := 0; iteration < 5; iteration++ {
		current, err := Run(context.Background(), spec, scenario)
		require.NoError(t, err)
		require.Equal(t, OutcomeCompleted, current.Outcome, current.Reason)
		if iteration != 0 {
			require.Equal(t, recorded.Network.Transitions, current.Network.Transitions)
		}
		recorded = current
	}
	require.Contains(t, networkDelays(recorded.Network.Transitions), uint64(7*time.Millisecond))
	plan, err := ReplayPlanFor(recorded.Record)
	require.NoError(t, err)
	spec.Replay = &plan
	replayed, err := Run(context.Background(), spec, scenario)
	require.NoError(t, err, "%+v", recorded.Network.Transitions)
	require.Equal(t, recorded.Network, replayed.Network)
}

func TestNetworkReplayRejectsReorderedTopologyBeforeMutation(t *testing.T) {
	serverBoot := uniqueBootID("network-replay-order-server")
	clientBoot := uniqueBootID("network-replay-order-client")
	require.NoError(t, RegisterBoot(serverBoot, func(context.Context, NodeContext) error { return nil }))
	require.NoError(t, RegisterBoot(clientBoot, func(context.Context, NodeContext) error { return nil }))
	spec := twoNodeNetworkSpec(serverBoot, clientBoot)
	recorded, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
		if err := cluster.Partition(ctx, "client", "server"); err != nil {
			return err
		}
		return cluster.Heal(ctx, "client", "server")
	})
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, recorded.Outcome, recorded.Reason)
	plan, err := ReplayPlanFor(recorded.Record)
	require.NoError(t, err)
	spec.Replay = &plan
	var actionErr error
	replayed, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
		actionErr = cluster.Heal(ctx, "client", "server")
		if actionErr == nil {
			return errors.New("reordered topology operation was admitted")
		}
		return actionErr
	})
	require.NoError(t, err)
	require.ErrorIs(t, actionErr, ErrReplayDiverged)
	require.Equal(t, OutcomeReplayDiverged, replayed.Outcome)
	require.Equal(t, ReplayDimensionNetwork, replayed.Divergence.Dimension)
	require.Equal(t, NetworkPartition, replayed.Divergence.ExpectedNetwork.Kind)
	require.Equal(t, NetworkHeal, replayed.Divergence.ActualNetwork.Kind)
}

func TestNetworkCrashDropsDelayedDataBeforeRestart(t *testing.T) {
	serverBoot := uniqueBootID("network-stale-server")
	clientBoot := uniqueBootID("network-stale-client")
	serverReady := make(chan NodeHandle, 2)
	firstAccepted := make(chan struct{})
	secondPayload := make(chan string, 1)
	clientCommands := make(chan string)
	clientResults := make(chan error)
	require.NoError(t, RegisterBoot(serverBoot, func(ctx context.Context, node NodeContext) error {
		listener, err := net.Listen("tcp4", net.JoinHostPort(node.Address, "7233"))
		if err != nil {
			return err
		}
		defer listener.Close()
		serverReady <- node.NodeHandle
		connection, err := listener.Accept()
		if err != nil {
			return err
		}
		defer connection.Close()
		if node.Incarnation == 1 {
			close(firstAccepted)
			select {}
		}
		payload := make([]byte, 3)
		if _, err := io.ReadFull(connection, payload); err != nil {
			return err
		}
		secondPayload <- string(payload)
		return nil
	}))
	require.NoError(t, RegisterBoot(clientBoot, func(ctx context.Context, _ NodeContext) error {
		var first net.Conn
		for {
			select {
			case command := <-clientCommands:
				var err error
				switch command {
				case "first":
					first, err = (&net.Dialer{}).DialContext(ctx, "tcp4", "10.0.0.1:7233")
					if err == nil {
						_, err = first.Write([]byte("old"))
					}
				case "stale":
					_, err = first.Write([]byte("bad"))
				case "second":
					var connection net.Conn
					connection, err = (&net.Dialer{}).DialContext(ctx, "tcp4", "10.0.0.1:7233")
					if err == nil {
						_, err = connection.Write([]byte("new"))
						err = errors.Join(err, connection.Close())
					}
				}
				clientResults <- err
			case <-ctx.Done():
				if first != nil {
					return errors.Join(ctx.Err(), first.Close())
				}
				return ctx.Err()
			}
		}
	}))
	spec := twoNodeNetworkSpec(serverBoot, clientBoot)
	for index := range spec.Links {
		spec.Links[index].DelayNanos = uint64(20 * time.Millisecond)
	}
	result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
		server, err := cluster.Start(ctx, "server")
		if err != nil {
			return err
		}
		<-serverReady
		client, err := cluster.Start(ctx, "client")
		if err != nil {
			return err
		}
		clientCommands <- "first"
		if err := <-clientResults; err != nil {
			return err
		}
		<-firstAccepted
		if err := cluster.Crash(ctx, server); err != nil {
			return err
		}
		restarted, err := cluster.Restart(ctx, "server")
		if err != nil {
			return err
		}
		<-serverReady
		clientCommands <- "stale"
		if err := <-clientResults; err == nil {
			return errors.New("stale connection remained writable after crash")
		}
		clientCommands <- "second"
		if err := <-clientResults; err != nil {
			return err
		}
		if payload := <-secondPayload; payload != "new" {
			return fmt.Errorf("restarted server payload = %q", payload)
		}
		if _, err := cluster.Wait(ctx, restarted); err != nil {
			return err
		}
		return cluster.Stop(ctx, client)
	})
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, result.Outcome, result.Reason)
	require.Contains(t, networkOutcomes(result.Network.Transitions), NetworkOutcomeReset)
}

func TestNetworkGracefulStopReturnsEOFAndCrashResetsConnection(t *testing.T) {
	serverBoot := uniqueBootID("network-lifecycle-server")
	clientBoot := uniqueBootID("network-lifecycle-client")
	serverReady := make(chan struct{}, 2)
	serverAccepted := make(chan struct{}, 2)
	clientCommands := make(chan string)
	clientResults := make(chan error)
	require.NoError(t, RegisterBoot(serverBoot, func(ctx context.Context, node NodeContext) error {
		listener, err := net.Listen("tcp4", net.JoinHostPort(node.Address, "7233"))
		if err != nil {
			return err
		}
		serverReady <- struct{}{}
		connection, err := listener.Accept()
		if err != nil {
			return errors.Join(err, listener.Close())
		}
		serverAccepted <- struct{}{}
		<-ctx.Done()
		return errors.Join(ctx.Err(), connection.Close(), listener.Close())
	}))
	require.NoError(t, RegisterBoot(clientBoot, func(ctx context.Context, _ NodeContext) error {
		var connection net.Conn
		for {
			select {
			case command := <-clientCommands:
				var err error
				switch command {
				case "dial":
					connection, err = (&net.Dialer{}).DialContext(ctx, "tcp4", "10.0.0.1:7233")
				case "read":
					_, err = connection.Read(make([]byte, 1))
					err = errors.Join(err, connection.Close())
					connection = nil
				default:
					err = fmt.Errorf("unknown client command %q", command)
				}
				clientResults <- err
			case <-ctx.Done():
				if connection != nil {
					return errors.Join(ctx.Err(), connection.Close())
				}
				return ctx.Err()
			}
		}
	}))
	spec := twoNodeNetworkSpec(serverBoot, clientBoot)
	result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
		server, err := cluster.Start(ctx, "server")
		if err != nil {
			return err
		}
		<-serverReady
		client, err := cluster.Start(ctx, "client")
		if err != nil {
			return err
		}
		clientCommands <- "dial"
		if err := <-clientResults; err != nil {
			return err
		}
		<-serverAccepted
		clientCommands <- "read"
		if err := cluster.Stop(ctx, server); err != nil {
			return err
		}
		if err := <-clientResults; !errors.Is(err, io.EOF) {
			return fmt.Errorf("read after graceful stop = %v", err)
		}
		restarted, err := cluster.Restart(ctx, "server")
		if err != nil {
			return err
		}
		<-serverReady
		clientCommands <- "dial"
		if err := <-clientResults; err != nil {
			return err
		}
		<-serverAccepted
		clientCommands <- "read"
		if err := cluster.Crash(ctx, restarted); err != nil {
			return err
		}
		if err := <-clientResults; err == nil || errors.Is(err, io.EOF) {
			return fmt.Errorf("read after crash = %v", err)
		}
		return cluster.Stop(ctx, client)
	})
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, result.Outcome, result.Reason)
	require.Contains(t, networkOutcomes(result.Network.Transitions), NetworkOutcomeReset)
}

func TestNetworkConnectionCapacityDoesNotConsumePort(t *testing.T) {
	serverBoot := uniqueBootID("network-capacity-server")
	clientBoot := uniqueBootID("network-capacity-client")
	serverReady := make(chan struct{}, 1)
	clientResult := make(chan error, 1)
	require.NoError(t, RegisterBoot(serverBoot, func(ctx context.Context, node NodeContext) error {
		listener, err := net.Listen("tcp4", net.JoinHostPort(node.Address, "7233"))
		if err != nil {
			return err
		}
		closed := make(chan error, 1)
		go func() {
			<-ctx.Done()
			closed <- listener.Close()
		}()
		serverReady <- struct{}{}
		connection, err := listener.Accept()
		if err != nil {
			return errors.Join(err, <-closed)
		}
		defer connection.Close()
		<-ctx.Done()
		return errors.Join(ctx.Err(), <-closed)
	}))
	require.NoError(t, RegisterBoot(clientBoot, func(ctx context.Context, _ NodeContext) error {
		<-serverReady
		first, err := (&net.Dialer{}).DialContext(ctx, "tcp4", "10.0.0.1:7233")
		if err != nil {
			return err
		}
		defer first.Close()
		second, err := (&net.Dialer{}).DialContext(ctx, "tcp4", "10.0.0.1:7233")
		if second != nil {
			err = errors.Join(err, second.Close())
		}
		clientResult <- err
		<-ctx.Done()
		return ctx.Err()
	}))
	spec := twoNodeNetworkSpec(serverBoot, clientBoot)
	spec.Limits.NetworkConnections = 1
	result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
		server, err := cluster.Start(ctx, "server")
		if err != nil {
			return err
		}
		client, err := cluster.Start(ctx, "client")
		if err != nil {
			return err
		}
		if err := <-clientResult; err == nil || !strings.Contains(err.Error(), "network resources exhausted") {
			return fmt.Errorf("second dial error = %v", err)
		}
		if err := cluster.Stop(ctx, client); err != nil {
			return err
		}
		return cluster.Stop(ctx, server)
	})
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, result.Outcome, result.Reason)
	require.Equal(t, uint64(1), result.Network.Snapshot.NextConnection)
	for _, node := range result.Network.Snapshot.Nodes {
		if node.Node == "client" {
			require.Equal(t, uint64(40001), node.NextClientPort)
		}
	}
	require.Contains(t, networkOutcomes(result.Network.Transitions), NetworkOutcomeCapacity)
}

func TestNetworkListenerCapacityDoesNotConsumePort(t *testing.T) {
	bootID := uniqueBootID("network-listener-capacity")
	type observation struct {
		firstPort int
		err       error
	}
	observed := make(chan observation, 1)
	require.NoError(t, RegisterBoot(bootID, func(context.Context, NodeContext) error {
		first, err := net.Listen("tcp4", "10.0.0.1:0")
		if err != nil {
			return err
		}
		second, secondErr := net.Listen("tcp4", "10.0.0.1:0")
		if second != nil {
			secondErr = errors.Join(secondErr, second.Close())
		}
		observed <- observation{firstPort: first.Addr().(*net.TCPAddr).Port, err: secondErr}
		return first.Close()
	}))
	spec := Spec{
		Schema: SpecSchema, Backend: BackendInProcess, Fidelity: FidelitySimulationModel, Seed: 23, Limits: DefaultLimits(),
		Nodes: []NodeSpec{{ID: "server", Boot: bootID, Address: "10.0.0.1"}},
	}
	spec.Limits.NetworkListeners = 1
	result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
		handle, err := cluster.Start(ctx, "server")
		if err != nil {
			return err
		}
		_, err = cluster.Wait(ctx, handle)
		return err
	})
	require.NoError(t, err)
	value := <-observed
	require.Equal(t, 20000, value.firstPort)
	require.ErrorContains(t, value.err, "network resources exhausted")
	require.Equal(t, uint64(20001), result.Network.Snapshot.Nodes[0].NextListenerPort)
	require.Len(t, result.Network.Snapshot.Listeners, 1)
	require.Contains(t, networkOutcomes(result.Network.Transitions), NetworkOutcomeCapacity)
}

func TestNetworkDeliveryAndByteCapacityDoNotConsumeDeliveryIdentity(t *testing.T) {
	tests := map[string]func(*Limits){
		"bytes": func(limits *Limits) {
			limits.NetworkBytes = 1
		},
		"deliveries": func(limits *Limits) {
			limits.NetworkDeliveries = 1
		},
	}
	for name, setLimit := range tests {
		t.Run(name, func(t *testing.T) {
			serverBoot := uniqueBootID("network-pending-capacity-server-" + name)
			clientBoot := uniqueBootID("network-pending-capacity-client-" + name)
			serverReady := make(chan struct{}, 1)
			clientResult := make(chan error, 1)
			require.NoError(t, RegisterBoot(serverBoot, func(ctx context.Context, node NodeContext) error {
				listener, err := net.Listen("tcp4", net.JoinHostPort(node.Address, "7233"))
				if err != nil {
					return err
				}
				serverReady <- struct{}{}
				connection, err := listener.Accept()
				if err != nil {
					return errors.Join(err, listener.Close())
				}
				<-ctx.Done()
				return errors.Join(ctx.Err(), connection.Close(), listener.Close())
			}))
			require.NoError(t, RegisterBoot(clientBoot, func(ctx context.Context, _ NodeContext) error {
				<-serverReady
				connection, err := (&net.Dialer{}).DialContext(ctx, "tcp4", "10.0.0.1:7233")
				if err != nil {
					return err
				}
				if _, err := connection.Write([]byte{'a'}); err != nil {
					return errors.Join(err, connection.Close())
				}
				_, secondErr := connection.Write([]byte{'b'})
				clientResult <- secondErr
				<-ctx.Done()
				return errors.Join(ctx.Err(), connection.Close())
			}))
			spec := twoNodeNetworkSpec(serverBoot, clientBoot)
			setLimit(&spec.Limits)
			result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
				server, err := cluster.Start(ctx, "server")
				if err != nil {
					return err
				}
				client, err := cluster.Start(ctx, "client")
				if err != nil {
					return err
				}
				if err := <-clientResult; err == nil || !strings.Contains(err.Error(), "network resources exhausted") {
					return fmt.Errorf("second write error = %v", err)
				}
				if err := cluster.Stop(ctx, client); err != nil {
					return err
				}
				return cluster.Stop(ctx, server)
			})
			require.NoError(t, err)
			require.Equal(t, uint64(1), result.Network.Snapshot.NextDelivery)
			require.Contains(t, networkOutcomes(result.Network.Transitions), NetworkOutcomeCapacity)
		})
	}
}

func TestNetworkTransitionCapacityDoesNotHealPartition(t *testing.T) {
	serverBoot := uniqueBootID("network-transition-capacity-server")
	clientBoot := uniqueBootID("network-transition-capacity-client")
	require.NoError(t, RegisterBoot(serverBoot, func(context.Context, NodeContext) error { return nil }))
	require.NoError(t, RegisterBoot(clientBoot, func(context.Context, NodeContext) error { return nil }))
	spec := twoNodeNetworkSpec(serverBoot, clientBoot)
	spec.Limits.NetworkTransitions = 1
	var healErr error
	result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
		if err := cluster.Partition(ctx, "client", "server"); err != nil {
			return err
		}
		healErr = cluster.Heal(ctx, "client", "server")
		return nil
	})
	require.NoError(t, err)
	require.ErrorContains(t, healErr, "network resources exhausted")
	require.Len(t, result.Network.Transitions, 1)
	require.Equal(t, NetworkPartition, result.Network.Transitions[0].Kind)
	for _, link := range result.Network.Snapshot.Links {
		require.False(t, link.Enabled)
	}
}

func twoNodeNetworkSpec(serverBoot, clientBoot BootID) Spec {
	return Spec{
		Schema: SpecSchema, Backend: BackendInProcess, Fidelity: FidelitySimulationModel, Seed: 23, Limits: DefaultLimits(),
		Nodes: []NodeSpec{
			{ID: "client", Boot: clientBoot, Address: "10.0.0.2"},
			{ID: "server", Boot: serverBoot, Address: "10.0.0.1"},
		},
		Links: []LinkSpec{
			{From: "client", To: "server", Enabled: true},
			{From: "server", To: "client", Enabled: true},
		},
	}
}

func networkDelays(transitions []NetworkTransition) []uint64 {
	result := make([]uint64, 0, len(transitions))
	for _, transition := range transitions {
		if transition.DelayNanos != 0 {
			result = append(result, transition.DelayNanos)
		}
	}
	return result
}

func networkOutcomes(transitions []NetworkTransition) []NetworkOutcome {
	result := make([]NetworkOutcome, 0, len(transitions))
	for _, transition := range transitions {
		result = append(result, transition.Outcome)
	}
	return result
}
