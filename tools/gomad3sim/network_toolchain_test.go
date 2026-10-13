//go:build gomad3_toolchain

package gomad3sim

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNetworkHeldFIFO(t *testing.T) { runNetworkHeldFIFO(t, BackendInProcess) }
func TestProcessNetworkHeldFIFO(t *testing.T) {
	if !processBackendAvailable() {
		t.Skip("Runner simulation transport is unavailable")
	}
	runNetworkHeldFIFO(t, BackendProcess)
}

func runNetworkHeldFIFO(t *testing.T, backend Backend) {
	t.Helper()
	for _, before := range []bool{true, false} {
		t.Run(fmt.Sprintf("partition-before-write=%t", before), func(t *testing.T) {
			actions := []FaultAction{
				{ID: "delay", Kind: FaultDelay, From: "client", To: "server", DelayNanos: uint64(5 * time.Millisecond)},
				{ID: "hold", Kind: FaultDisconnect, From: "client", To: "server"},
				{ID: "hold-again", Kind: FaultDisconnect, From: "client", To: "server"},
				{ID: "release", Kind: FaultReconnect, From: "client", To: "server"},
				{ID: "release-again", Kind: FaultReconnect, From: "client", To: "server"},
				{ID: "repartition", Kind: FaultDisconnect, From: "client", To: "server"},
				{ID: "release-final", Kind: FaultReconnect, From: "client", To: "server"},
			}
			plan, err := NewFaultPlan(actions)
			require.NoError(t, err)
			result := runNetworkStallPair(t, backend, "held-fifo", func(spec *Spec) {
				spec.Faults = &plan
				spec.Limits.NetworkConnectionBytes = 32
				spec.Limits.NetworkStallNanos = uint64(30 * time.Millisecond)
			}, func(conn net.Conn) error {
				started := time.Now()
				<-time.After(3 * time.Millisecond)
				if err := conn.SetReadDeadline(time.Now().Add(time.Millisecond)); err != nil {
					return err
				}
				if n, err := conn.Read(make([]byte, 1)); n != 0 || !errors.Is(err, os.ErrDeadlineExceeded) {
					return fmt.Errorf("held read=%d,%v", n, err)
				}
				if err := conn.SetReadDeadline(time.Time{}); err != nil {
					return err
				}
				if _, err := conn.Write([]byte{'v'}); err != nil {
					return fmt.Errorf("enabled reverse write: %w", err)
				}
				for _, want := range []string{"one", "two", "fresh"} {
					var header [2]byte
					if _, err := io.ReadFull(conn, header[:]); err != nil {
						return err
					}
					length := binary.BigEndian.Uint16(header[:])
					if length != uint16(len(want)) {
						return fmt.Errorf("frame length=%d, want %d", length, len(want))
					}
					payload := make([]byte, length)
					if _, err := io.ReadFull(conn, payload); err != nil || string(payload) != want {
						return fmt.Errorf("frame=%q,%v, want %q", payload, err, want)
					}
				}
				if elapsed := time.Since(started); elapsed < 15*time.Millisecond {
					return fmt.Errorf("repartitioned FIFO released early at %s", elapsed)
				}
				if err := conn.SetReadDeadline(time.Now().Add(time.Millisecond)); err != nil {
					return err
				}
				if n, err := conn.Read(make([]byte, 1)); n != 0 || !errors.Is(err, os.ErrDeadlineExceeded) {
					return fmt.Errorf("duplicate frame tail=%d,%v", n, err)
				}
				if err := conn.SetReadDeadline(time.Time{}); err != nil {
					return err
				}
				_, err := conn.Write([]byte{'d'})
				return err
			}, func(conn net.Conn) error {
				for _, frame := range []string{"\x00\x03one", "\x00\x03two"} {
					if n, err := conn.Write([]byte(frame)); n != len(frame) || err != nil {
						return fmt.Errorf("held frame write=%d,%v", n, err)
					}
				}
				buffer := make([]byte, 1)
				if _, err := io.ReadFull(conn, buffer); err != nil || buffer[0] != 'v' {
					return fmt.Errorf("enabled reverse read=%q,%v", buffer, err)
				}
				<-time.After(4 * time.Millisecond)
				if n, err := conn.Write([]byte("\x00\x05fresh")); n != 7 || err != nil {
					return fmt.Errorf("fresh frame write=%d,%v", n, err)
				}
				if _, err := io.ReadFull(conn, buffer); err != nil || buffer[0] != 'd' {
					return fmt.Errorf("framed stream acknowledgement=%q,%v", buffer, err)
				}
				return nil
			}, func(ctx context.Context, cluster Cluster, _, _ NodeHandle) error {
				<-time.After(time.Millisecond + time.Nanosecond)
				if _, err := cluster.ApplyFault(ctx, actions[0]); err != nil {
					return err
				}
				if !before {
					<-time.After(2 * time.Millisecond)
				}
				if _, err := cluster.ApplyFault(ctx, actions[1]); err != nil {
					return err
				}
				if _, err := cluster.ApplyFault(ctx, actions[2]); err != nil {
					return err
				}
				wait := 7 * time.Millisecond
				if !before {
					wait = 5 * time.Millisecond
				}
				<-time.After(wait - time.Nanosecond)
				if _, err := cluster.ApplyFault(ctx, actions[3]); err != nil {
					return err
				}
				<-time.After(time.Millisecond)
				if _, err := cluster.ApplyFault(ctx, actions[4]); err != nil {
					return err
				}
				<-time.After(time.Millisecond)
				if _, err := cluster.ApplyFault(ctx, actions[5]); err != nil {
					return err
				}
				<-time.After(time.Millisecond)
				_, err := cluster.ApplyFault(ctx, actions[6])
				return err
			})
			require.Contains(t, networkOutcomes(result.Network.Transitions), NetworkOutcomeHeld)
			require.NotContains(t, networkOutcomes(result.Network.Transitions), NetworkOutcomeTimeout)
		})
	}
}

func TestNetworkStallExpiry(t *testing.T) { runNetworkStallExpiry(t, BackendInProcess) }
func TestProcessNetworkStallExpiry(t *testing.T) {
	if !processBackendAvailable() {
		t.Skip("Runner simulation transport is unavailable")
	}
	runNetworkStallExpiry(t, BackendProcess)
}

func runNetworkStallExpiry(t *testing.T, backend Backend) {
	t.Helper()
	t.Run("returned-writer-and-both-endpoints", func(t *testing.T) {
		result := runNetworkStallPair(t, backend, "autonomous-expiry", func(spec *Spec) {
			spec.Limits.NetworkStallNanos = uint64(5 * time.Millisecond)
		}, func(conn net.Conn) error {
			<-time.After(12 * time.Millisecond)
			return checkNetworkPersistentStallTimeout(conn)
		}, func(conn net.Conn) error {
			if n, err := conn.Write([]byte("\x00\x03one")); n != 5 || err != nil {
				return fmt.Errorf("returned held writer=%d,%v", n, err)
			}
			<-time.After(10 * time.Millisecond)
			return checkNetworkPersistentStallTimeout(conn)
		}, func(ctx context.Context, cluster Cluster, _, _ NodeHandle) error {
			<-time.After(time.Millisecond + time.Nanosecond)
			if err := cluster.Partition(ctx, "client", "server"); err != nil {
				return err
			}
			<-time.After(3 * time.Millisecond)
			if err := cluster.Partition(ctx, "client", "server"); err != nil {
				return err
			}
			<-time.After(4 * time.Millisecond)
			return cluster.Heal(ctx, "client", "server")
		})
		resets := 0
		for _, transition := range result.Network.Transitions {
			if transition.Kind == NetworkReset && transition.Outcome == NetworkOutcomeTimeout {
				resets++
			}
		}
		require.Equal(t, 1, resets)
	})
	t.Run("idle-survives-and-old-epoch-cannot-reset", func(t *testing.T) {
		result := runNetworkStallPair(t, backend, "idle-survival", func(spec *Spec) {
			spec.Limits.NetworkStallNanos = uint64(3 * time.Millisecond)
		}, func(conn net.Conn) error {
			buffer := make([]byte, 1)
			if _, err := io.ReadFull(conn, buffer); err != nil || buffer[0] != 'x' {
				return fmt.Errorf("healed byte=%q,%v", buffer, err)
			}
			<-time.After(12 * time.Millisecond)
			if _, err := conn.Write([]byte{'r'}); err != nil {
				return fmt.Errorf("idle server survived: %w", err)
			}
			return nil
		}, func(conn net.Conn) error {
			if _, err := conn.Write([]byte{'x'}); err != nil {
				return err
			}
			<-time.After(12 * time.Millisecond)
			buffer := make([]byte, 1)
			if _, err := io.ReadFull(conn, buffer); err != nil || buffer[0] != 'r' {
				return fmt.Errorf("idle client survived=%q,%v", buffer, err)
			}
			return nil
		}, func(ctx context.Context, cluster Cluster, _, _ NodeHandle) error {
			<-time.After(time.Millisecond + time.Nanosecond)
			if err := cluster.Partition(ctx, "client", "server"); err != nil {
				return err
			}
			<-time.After(2 * time.Millisecond)
			if err := cluster.Heal(ctx, "client", "server"); err != nil {
				return err
			}
			<-time.After(time.Millisecond)
			if err := cluster.Partition(ctx, "client", "server"); err != nil {
				return err
			}
			<-time.After(8 * time.Millisecond)
			return cluster.Heal(ctx, "client", "server")
		})
		require.NotContains(t, networkOutcomes(result.Network.Transitions), NetworkOutcomeTimeout)
	})
}

func TestNetworkStallOrdering(t *testing.T) { runNetworkStallOrdering(t, BackendInProcess) }
func TestProcessNetworkStallOrdering(t *testing.T) {
	if !processBackendAvailable() {
		t.Skip("Runner simulation transport is unavailable")
	}
	runNetworkStallOrdering(t, BackendProcess)
}

func runNetworkStallOrdering(t *testing.T, backend Backend) {
	t.Helper()
	for _, test := range []struct {
		name  string
		heal  time.Duration
		reset bool
		tie   bool
	}{
		{"limit-minus-one-nanosecond", 5*time.Millisecond - time.Nanosecond, false, false},
		{"exact-limit", 5 * time.Millisecond, false, false},
		{"first-due-instant", 5*time.Millisecond + time.Nanosecond, false, true},
		{"expiry-before-heal", 5*time.Millisecond + 2*time.Nanosecond, true, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := runNetworkStallPair(t, backend, "stall-order-"+test.name, func(spec *Spec) {
				spec.Limits.NetworkStallNanos = uint64(3 * time.Millisecond)
			}, func(conn net.Conn) error {
				<-time.After(8 * time.Millisecond)
				if test.reset {
					return checkNetworkPersistentStallTimeout(conn)
				}
				buffer := make([]byte, 1)
				_, err := io.ReadFull(conn, buffer)
				if test.tie && errors.Is(err, syscall.ETIMEDOUT) {
					return checkNetworkPersistentStallTimeout(conn)
				}
				if err != nil || buffer[0] != 'x' {
					return fmt.Errorf("boundary heal data=%q,%v", buffer, err)
				}
				_, err = conn.Write([]byte{'r'})
				return err
			}, func(conn net.Conn) error {
				if _, err := conn.Write([]byte{'x'}); err != nil {
					return err
				}
				<-time.After(6 * time.Millisecond)
				if test.reset {
					return checkNetworkPersistentStallTimeout(conn)
				}
				buffer := make([]byte, 1)
				_, err := io.ReadFull(conn, buffer)
				if test.tie && errors.Is(err, syscall.ETIMEDOUT) {
					return checkNetworkPersistentStallTimeout(conn)
				}
				if err != nil || buffer[0] != 'r' {
					return fmt.Errorf("boundary heal reply=%q,%v", buffer, err)
				}
				return nil
			}, func(ctx context.Context, cluster Cluster, _, _ NodeHandle) error {
				<-time.After(time.Millisecond + time.Nanosecond)
				if err := cluster.Partition(ctx, "client", "server"); err != nil {
					return err
				}
				<-time.After(test.heal - time.Millisecond - time.Nanosecond)
				return cluster.Heal(ctx, "client", "server")
			})
			if !test.tie {
				require.Equal(t, test.reset, result.Network.Snapshot.Connections[0].Reset)
			}
		})
	}
}

func checkNetworkPersistentStallTimeout(conn net.Conn) error {
	for range 2 {
		if err := conn.SetDeadline(time.Time{}); err != nil {
			return err
		}
		for _, operation := range []func([]byte) (int, error){conn.Read, conn.Write} {
			n, err := operation([]byte{'x'})
			var timeout net.Error
			if n != 0 || !errors.Is(err, syscall.ETIMEDOUT) || !errors.As(err, &timeout) || !timeout.Timeout() || errors.Is(err, os.ErrDeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
				return fmt.Errorf("persistent stall timeout=%d,%v", n, err)
			}
		}
	}
	return nil
}

func TestNetworkPartitionedDialStall(t *testing.T) {
	runNetworkPartitionedDialStall(t, BackendInProcess)
}
func TestProcessNetworkPartitionedDialStall(t *testing.T) {
	if !processBackendAvailable() {
		t.Skip("Runner simulation transport is unavailable")
	}
	runNetworkPartitionedDialStall(t, BackendProcess)
}

func runNetworkPartitionedDialStall(t *testing.T, backend Backend) {
	t.Helper()
	for _, mode := range []string{"heal", "heal-at-exact-limit", "caller-deadline", "stall", "completed-context"} {
		t.Run(mode, func(t *testing.T) {
			serverBoot := uniqueBootID("dial-stall-server-" + string(backend) + "-" + mode)
			clientBoot := uniqueBootID("dial-stall-client-" + string(backend) + "-" + mode)
			listening := make(chan struct{}, 1)
			dialing := make(chan struct{}, 1)
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
				if backend == BackendInProcess {
					listening <- struct{}{}
				}
				conn, err := listener.Accept()
				if err != nil {
					if ctx.Err() != nil {
						return errors.Join(ctx.Err(), <-closed)
					}
					return err
				}
				defer conn.Close()
				buffer := make([]byte, 1)
				if _, err := io.ReadFull(conn, buffer); err != nil || buffer[0] != 'x' {
					return fmt.Errorf("healed dial request=%q,%v", buffer, err)
				}
				if _, err := conn.Write(buffer); err != nil {
					return err
				}
				<-ctx.Done()
				return errors.Join(ctx.Err(), <-closed)
			}))
			require.NoError(t, RegisterBoot(clientBoot, func(ctx context.Context, _ NodeContext) error {
				dialCtx := ctx
				if mode == "caller-deadline" {
					var cancel context.CancelFunc
					dialCtx, cancel = context.WithTimeout(ctx, 2*time.Millisecond)
					defer cancel()
				}
				if mode == "completed-context" {
					var cancel context.CancelFunc
					dialCtx, cancel = context.WithCancel(ctx)
					cancel()
				}
				if backend == BackendInProcess && mode != "completed-context" {
					dialing <- struct{}{}
				}
				started := time.Now()
				conn, err := (&net.Dialer{}).DialContext(dialCtx, "tcp4", "10.0.0.1:7233")
				if conn != nil {
					defer conn.Close()
				}
				switch mode {
				case "caller-deadline":
					if conn != nil || !errors.Is(err, context.DeadlineExceeded) || errors.Is(err, syscall.ETIMEDOUT) {
						return fmt.Errorf("caller dial deadline=%v", err)
					}
				case "completed-context":
					if conn != nil || !errors.Is(err, context.Canceled) {
						return fmt.Errorf("completed dial context=%v", err)
					}
				case "stall":
					var timeout net.Error
					if conn != nil || !errors.Is(err, syscall.ETIMEDOUT) || !errors.As(err, &timeout) || !timeout.Timeout() || errors.Is(err, context.DeadlineExceeded) {
						return fmt.Errorf("independent dial stall=%v", err)
					}
					if elapsed := time.Since(started); elapsed != 5*time.Millisecond+time.Nanosecond {
						return fmt.Errorf("redundant partition restarted dial stall: elapsed=%s", elapsed)
					}
				case "heal", "heal-at-exact-limit":
					if err != nil {
						return err
					}
					wantElapsed := 2 * time.Millisecond
					if mode == "heal-at-exact-limit" {
						wantElapsed = 5 * time.Millisecond
					}
					if elapsed := time.Since(started); elapsed != wantElapsed {
						return fmt.Errorf("healed dial elapsed=%s", elapsed)
					}
					if _, err := conn.Write([]byte{'x'}); err != nil {
						return err
					}
					buffer := make([]byte, 1)
					if _, err := io.ReadFull(conn, buffer); err != nil || buffer[0] != 'x' {
						return fmt.Errorf("healed dial response=%q,%v", buffer, err)
					}
				}
				return nil
			}))
			spec := twoNodeNetworkSpec(serverBoot, clientBoot)
			spec.Backend = backend
			if backend == BackendProcess {
				spec.Fidelity = FidelityHardIsolation
			}
			spec.Limits.NetworkStallNanos = uint64(5 * time.Millisecond)
			scenario := func(ctx context.Context, cluster Cluster) error {
				server, err := cluster.Start(ctx, "server")
				if err != nil {
					return err
				}
				if backend == BackendProcess {
					if err := waitNetworkHandleModelRead(ctx, cluster, server, 2); err != nil {
						return err
					}
				} else {
					<-listening
				}
				if err := cluster.Partition(ctx, "client", "server"); err != nil {
					return err
				}
				client, err := cluster.Start(ctx, "client")
				if err != nil {
					return err
				}
				if mode != "completed-context" {
					if backend == BackendProcess {
						if err := waitNetworkHandleModelRead(ctx, cluster, client, 1); err != nil {
							return err
						}
					} else {
						<-dialing
					}
				}
				if mode == "heal" || mode == "heal-at-exact-limit" || mode == "stall" {
					wait := 2 * time.Millisecond
					if mode == "heal-at-exact-limit" {
						wait = 5 * time.Millisecond
					}
					<-time.After(wait)
					if mode != "stall" {
						if err := cluster.Heal(ctx, "client", "server"); err != nil {
							return err
						}
					} else if err := cluster.Partition(ctx, "client", "server"); err != nil {
						return err
					}
				}
				if err := waitNetworkHandleExit(ctx, cluster, client); err != nil {
					return err
				}
				return cluster.Stop(ctx, server)
			}
			recorded, err := Run(context.Background(), spec, scenario)
			require.NoError(t, err)
			require.Equal(t, OutcomeCompleted, recorded.Outcome, recorded.Reason)
			if mode != "heal" && mode != "heal-at-exact-limit" {
				require.Zero(t, recorded.Network.Snapshot.NextConnection)
				require.Empty(t, recorded.Network.Snapshot.Connections)
			}
			plan, err := ReplayPlanFor(recorded.Record)
			require.NoError(t, err)
			spec.Replay = &plan
			replayed, err := Run(context.Background(), spec, scenario)
			require.NoError(t, err)
			require.Equal(t, OutcomeCompleted, replayed.Outcome, replayed.Reason)
			require.Equal(t, recorded.Record.Identity, replayed.Record.Identity)
		})
	}
}

func runNetworkStallPair(t *testing.T, backend Backend, name string, configure func(*Spec), serverAction, clientAction func(net.Conn) error, action func(context.Context, Cluster, NodeHandle, NodeHandle) error) Result {
	t.Helper()
	serverBoot := uniqueBootID(name + "-server-" + string(backend))
	clientBoot := uniqueBootID(name + "-client-" + string(backend))
	listening := make(chan struct{}, 1)
	accepted := make(chan struct{}, 1)
	require.NoError(t, RegisterBoot(serverBoot, func(ctx context.Context, node NodeContext) error {
		listener, err := net.Listen("tcp4", net.JoinHostPort(node.Address, "7233"))
		if err != nil {
			return err
		}
		defer listener.Close()
		if backend == BackendInProcess {
			listening <- struct{}{}
		}
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		defer conn.Close()
		if _, err := conn.Write([]byte{'r'}); err != nil {
			return err
		}
		if backend == BackendInProcess {
			accepted <- struct{}{}
		}
		marker := make([]byte, 1)
		if _, err := io.ReadFull(conn, marker); err != nil || marker[0] != 's' {
			return fmt.Errorf("setup marker=%q,%v", marker, err)
		}
		return serverAction(conn)
	}))
	require.NoError(t, RegisterBoot(clientBoot, func(ctx context.Context, _ NodeContext) error {
		conn, err := dialNetworkHandlePeer(ctx)
		if err != nil {
			return err
		}
		defer conn.Close()
		greeting := make([]byte, 1)
		if _, err := io.ReadFull(conn, greeting); err != nil || greeting[0] != 'r' {
			return fmt.Errorf("setup greeting=%q,%v", greeting, err)
		}
		<-time.After(time.Millisecond)
		if _, err := conn.Write([]byte{'s'}); err != nil {
			return err
		}
		<-time.After(time.Millisecond)
		return clientAction(conn)
	}))
	spec := twoNodeNetworkSpec(serverBoot, clientBoot)
	spec.Backend = backend
	if backend == BackendProcess {
		spec.Fidelity = FidelityHardIsolation
	}
	configure(&spec)
	scenario := func(ctx context.Context, cluster Cluster) error {
		server, err := cluster.Start(ctx, "server")
		if err != nil {
			return err
		}
		if backend == BackendProcess {
			if err := waitNetworkHandleModelRead(ctx, cluster, server, 2); err != nil {
				return err
			}
		} else {
			<-listening
		}
		client, err := cluster.Start(ctx, "client")
		if err != nil {
			return err
		}
		if backend == BackendProcess {
			if err := waitNetworkHandleModelRead(ctx, cluster, server, 4); err != nil {
				return err
			}
		} else {
			<-accepted
		}
		if err := action(ctx, cluster, server, client); err != nil {
			return err
		}
		if err := waitNetworkHandleExit(ctx, cluster, client); err != nil {
			return err
		}
		return waitNetworkHandleExit(ctx, cluster, server)
	}
	recorded, err := Run(context.Background(), spec, scenario)
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, recorded.Outcome, recorded.Reason)
	plan, err := ReplayPlanFor(recorded.Record)
	require.NoError(t, err)
	spec.Replay = &plan
	replayed, err := Run(context.Background(), spec, scenario)
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, replayed.Outcome, replayed.Reason)
	require.Equal(t, recorded.Record.Identity, replayed.Record.Identity)
	return recorded
}

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

func TestNetworkByteCapacityPreservesConnectionIdentity(t *testing.T) {
	tests := map[string]func(*Limits){
		"bytes": func(limits *Limits) {
			limits.NetworkBytes = 1
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
			require.Equal(t, uint64(1), result.Network.Snapshot.NextConnection)
			require.Contains(t, networkOutcomes(result.Network.Transitions), NetworkOutcomeCapacity)
		})
	}
}

func TestNetworkTinyWritesUseByteCapacity(t *testing.T) {
	runNetworkStallPair(t, BackendInProcess, "tiny-writes-byte-capacity", func(spec *Spec) {
		spec.Limits.NetworkBytes = 65
		spec.Limits.NetworkConnectionBytes = 66
	}, func(conn net.Conn) error {
		<-time.After(10 * time.Millisecond)
		payload := make([]byte, 65)
		if _, err := io.ReadFull(conn, payload); err != nil || string(payload) != strings.Repeat("x", 65) {
			return fmt.Errorf("tiny-write stream=%q,%v", payload, err)
		}
		_, err := conn.Write([]byte{'r'})
		return err
	}, func(conn net.Conn) error {
		for range 65 {
			if n, err := conn.Write([]byte{'x'}); n != 1 || err != nil {
				return fmt.Errorf("tiny write within byte capacity=%d,%v", n, err)
			}
		}
		if n, err := conn.Write([]byte{'y'}); n != 0 || err == nil || !strings.Contains(err.Error(), "network resources exhausted") {
			return fmt.Errorf("global byte capacity after 65 writes=%d,%v", n, err)
		}
		payload := make([]byte, 1)
		if _, err := io.ReadFull(conn, payload); err != nil || payload[0] != 'r' {
			return fmt.Errorf("tiny-write acknowledgement=%q,%v", payload, err)
		}
		return nil
	}, func(context.Context, Cluster, NodeHandle, NodeHandle) error { return nil })
}

func TestNetworkReplayPermitsSameLengthPayloadChange(t *testing.T) {
	serverBoot := uniqueBootID("network-replay-payload-waiver-server")
	clientBoot := uniqueBootID("network-replay-payload-waiver-client")
	listening := make(chan struct{}, 1)
	received := make(chan byte, 1)
	payload := byte('x')
	require.NoError(t, RegisterBoot(serverBoot, func(ctx context.Context, node NodeContext) error {
		listener, err := net.Listen("tcp4", net.JoinHostPort(node.Address, "7233"))
		if err != nil {
			return err
		}
		defer listener.Close()
		listening <- struct{}{}
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		defer conn.Close()
		buffer := make([]byte, 1)
		if _, err := io.ReadFull(conn, buffer); err != nil {
			return err
		}
		received <- buffer[0]
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
		if _, err := conn.Write([]byte{payload}); err != nil {
			return err
		}
		buffer := make([]byte, 1)
		_, err = io.ReadFull(conn, buffer)
		return err
	}))
	spec := twoNodeNetworkSpec(serverBoot, clientBoot)
	scenario := func(ctx context.Context, cluster Cluster) error {
		server, err := cluster.Start(ctx, "server")
		if err != nil {
			return err
		}
		<-listening
		client, err := cluster.Start(ctx, "client")
		if err != nil {
			return err
		}
		if err := waitNetworkHandleExit(ctx, cluster, client); err != nil {
			return err
		}
		return cluster.Stop(ctx, server)
	}
	recorded, err := Run(context.Background(), spec, scenario)
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, recorded.Outcome, recorded.Reason)
	require.Equal(t, byte('x'), <-received)
	plan, err := ReplayPlanFor(recorded.Record)
	require.NoError(t, err)
	spec.Replay = &plan
	payload = 'y'
	replayed, err := Run(context.Background(), spec, scenario)
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, replayed.Outcome, replayed.Reason)
	require.Equal(t, byte('y'), <-received)
	require.Equal(t, recorded.Network, replayed.Network)
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
