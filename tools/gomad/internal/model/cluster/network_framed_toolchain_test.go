//go:build gomad_toolchain

package cluster

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNetworkFramedPartition(t *testing.T) { runNetworkFramedPartition(t, BackendInProcess) }

func TestProcessNetworkFramedPartition(t *testing.T) {
	if !processBackendAvailable() {
		t.Skip("Runner simulation transport is unavailable")
	}
	runNetworkFramedPartition(t, BackendProcess)
}

func runNetworkFramedPartition(t *testing.T, backend Backend) {
	t.Helper()
	for _, mode := range []string{"short", "long", "repeated"} {
		t.Run(mode, func(t *testing.T) {
			frames := []string{"first request", "second request with a longer body", "last request"}
			result := runNetworkStallPair(t, backend, "framed-"+mode, func(spec *Spec) {
				spec.Limits.NetworkConnectionBytes = 256
				spec.Limits.NetworkStallNanos = uint64(5 * time.Millisecond)
			}, func(conn net.Conn) error {
				if mode == "short" {
					if err := conn.SetReadDeadline(time.Now().Add(2 * time.Millisecond)); err != nil {
						return err
					}
					var header [2]byte
					if n, err := io.ReadFull(conn, header[:]); n != 0 || !errors.Is(err, os.ErrDeadlineExceeded) {
						return fmt.Errorf("partitioned frame caller deadline=%d,%v", n, err)
					}
					if err := conn.SetReadDeadline(time.Time{}); err != nil {
						return err
					}
				}
				if mode == "long" {
					var header [2]byte
					if n, err := io.ReadFull(conn, header[:]); n != 0 || !isNetworkTimeout(err) {
						return fmt.Errorf("partitioned frame stall timeout=%d,%v", n, err)
					}
					return checkNetworkPersistentStallTimeout(conn)
				}
				for _, want := range frames {
					var header [2]byte
					if _, err := io.ReadFull(conn, header[:]); err != nil {
						return fmt.Errorf("read frame header: %w", err)
					}
					length := binary.BigEndian.Uint16(header[:])
					if length != uint16(len(want)) {
						return fmt.Errorf("frame length=%d, want %d", length, len(want))
					}
					payload := make([]byte, length)
					if _, err := io.ReadFull(conn, payload); err != nil || string(payload) != want {
						return fmt.Errorf("frame payload=%q,%v, want %q", payload, err, want)
					}
				}
				if err := conn.SetReadDeadline(time.Now().Add(time.Millisecond)); err != nil {
					return err
				}
				if n, err := conn.Read(make([]byte, 1)); n != 0 || !errors.Is(err, os.ErrDeadlineExceeded) {
					return fmt.Errorf("extra frame bytes=%d,%v", n, err)
				}
				_, err := conn.Write([]byte{'d'})
				return err
			}, func(conn net.Conn) error {
				for _, payload := range frames {
					var header [2]byte
					binary.BigEndian.PutUint16(header[:], uint16(len(payload)))
					for _, fragment := range [][]byte{header[:1], header[1:], []byte(payload[:3]), []byte(payload[3:])} {
						if n, err := conn.Write(fragment); n != len(fragment) || err != nil {
							return fmt.Errorf("frame fragment write=%d,%v, want %d", n, err, len(fragment))
						}
					}
				}
				marker := make([]byte, 1)
				if mode == "long" {
					if n, err := conn.Read(marker); n != 0 || !isNetworkTimeout(err) {
						return fmt.Errorf("frame writer stall timeout=%d,%v", n, err)
					}
					return checkNetworkPersistentStallTimeout(conn)
				}
				if _, err := io.ReadFull(conn, marker); err != nil || marker[0] != 'd' {
					return fmt.Errorf("complete frames acknowledgement=%q,%v", marker, err)
				}
				return nil
			}, func(ctx context.Context, cluster Cluster, _, _ NodeHandle) error {
				<-time.After(time.Millisecond + time.Nanosecond)
				if err := cluster.SetDelay(ctx, "client", "server", uint64(2*time.Millisecond)); err != nil {
					return err
				}
				if err := cluster.Partition(ctx, "client", "server"); err != nil {
					return err
				}
				if mode == "long" {
					<-time.After(9 * time.Millisecond)
					return cluster.Heal(ctx, "client", "server")
				}
				<-time.After(3 * time.Millisecond)
				if err := cluster.Heal(ctx, "client", "server"); err != nil {
					return err
				}
				if mode == "repeated" {
					<-time.After(time.Millisecond)
					if err := cluster.Partition(ctx, "client", "server"); err != nil {
						return err
					}
					<-time.After(3 * time.Millisecond)
					return cluster.Heal(ctx, "client", "server")
				}
				return nil
			})
			require.Contains(t, networkOutcomes(result.Network.Transitions), NetworkOutcomeHeld)
			require.Equal(t, mode == "long", result.Network.Snapshot.Connections[0].Reset)
		})
	}
}

func isNetworkTimeout(err error) bool {
	var timeout net.Error
	return errors.As(err, &timeout) && timeout.Timeout() && !errors.Is(err, os.ErrDeadlineExceeded)
}
