//go:build gomad3_toolchain

package gomad3sim

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type networkHandleCase struct {
	name   string
	server func(*net.TCPListener, *net.TCPConn, bool) error
	client func(*net.TCPConn) error
}

func TestNetworkHandleOperationParity(t *testing.T) {
	for index, test := range networkHandleCases() {
		t.Run("standalone/"+test.name, func(t *testing.T) {
			listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.ParseIP("127.0.0.1")})
			require.NoError(t, err)
			defer listener.Close()
			result := make(chan error, 1)
			go func() {
				conn, err := listener.AcceptTCP()
				if err == nil {
					defer conn.Close()
					control, controlErr := listener.AcceptTCP()
					if controlErr != nil {
						result <- controlErr
						return
					}
					defer control.Close()
					err = test.server(listener, conn, false)
					if err == nil {
						_, err = control.Write([]byte{'d'})
					}
				}
				result <- err
			}()
			conn, err := net.DialTCP("tcp4", nil, listener.Addr().(*net.TCPAddr))
			require.NoError(t, err)
			control, err := net.Dial("tcp4", conn.RemoteAddr().String())
			require.NoError(t, err)
			defer control.Close()
			require.NoError(t, test.client(conn))
			_, err = io.ReadFull(control, make([]byte, 1))
			require.NoError(t, err)
			conn.Close()
			require.NoError(t, <-result)
		})
		t.Run("in-process/"+test.name, func(t *testing.T) { runNetworkHandleCase(t, BackendInProcess, index) })
	}
}
func TestProcessNetworkHandleOperationParity(t *testing.T)        { runProcessNetworkHandleCase(t, 1) }
func TestProcessNetworkHandleQueuedAccept(t *testing.T)           { runProcessNetworkHandleCase(t, 0) }
func TestProcessNetworkHandleShortReadsAndHalfClose(t *testing.T) { runProcessNetworkHandleCase(t, 2) }
func TestProcessNetworkHandleEmptyIO(t *testing.T)                { runProcessNetworkHandleCase(t, 3) }
func TestProcessNetworkHandleDeadlines(t *testing.T)              { runProcessNetworkHandleCase(t, 4) }
func TestProcessNetworkHandlePartialWrite(t *testing.T)           { runProcessNetworkHandleCase(t, 5) }
func runProcessNetworkHandleCase(t *testing.T, index int) {
	t.Helper()
	if !processBackendAvailable() {
		t.Skip("Runner simulation transport is unavailable")
	}
	runNetworkHandleCase(t, BackendProcess, index)
}

func runNetworkHandleCase(t *testing.T, backend Backend, index int) {
	t.Helper()
	test := networkHandleCases()[index]
	serverBoot := uniqueBootID("network-handles-server-" + string(backend) + "-" + test.name)
	clientBoot := uniqueBootID("network-handles-client-" + string(backend) + "-" + test.name)
	require.NoError(t, RegisterBoot(serverBoot, func(ctx context.Context, node NodeContext) error {
		listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.ParseIP(node.Address), Port: 7233})
		if err != nil {
			return err
		}
		defer listener.Close()
		conn, err := listener.AcceptTCP()
		if err != nil {
			return err
		}
		defer conn.Close()
		control, err := listener.AcceptTCP()
		if err != nil {
			return err
		}
		defer control.Close()
		if err := test.server(listener, conn, backend == BackendProcess); err != nil {
			return err
		}
		if _, err := control.Write([]byte{'d'}); err != nil {
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
		control, err := net.Dial("tcp4", conn.RemoteAddr().String())
		if err != nil {
			return err
		}
		defer control.Close()
		if err := test.client(conn); err != nil {
			return err
		}
		_, err = io.ReadFull(control, make([]byte, 1))
		return err
	}))
	spec := twoNodeNetworkSpec(serverBoot, clientBoot)
	spec.Backend = backend
	if backend == BackendProcess {
		spec.Fidelity = FidelityHardIsolation
	}
	run := func(spec Spec) Result {
		result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
			server, err := cluster.Start(ctx, "server")
			if err != nil {
				return err
			}
			if backend == BackendProcess {
				if err := waitNetworkHandleModelRead(ctx, cluster, server, 2); err != nil {
					return err
				}
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
	plan, err := ReplayPlanFor(first.Record)
	require.NoError(t, err)
	spec.Replay = &plan
	second := run(spec)
	require.Equal(t, first.Record.Identity, second.Record.Identity)
}
func dialNetworkHandlePeer(ctx context.Context) (*net.TCPConn, error) {
	var err error
	for range 4096 {
		var conn net.Conn
		conn, err = (&net.Dialer{}).DialContext(ctx, "tcp4", "10.0.0.1:7233")
		if err == nil {
			return conn.(*net.TCPConn), nil
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		runtime.Gosched()
	}
	return nil, err
}
func networkHandleCases() []networkHandleCase {
	return []networkHandleCase{
		{"queued-accept-close-precedence", func(listener *net.TCPListener, conn *net.TCPConn, process bool) error {
			buffer := make([]byte, 1)
			if _, err := io.ReadFull(conn, buffer); err != nil {
				return err
			}
			if err := listener.SetDeadline(time.Now().Add(-time.Second)); err != nil {
				return err
			}
			if err := listener.Close(); err != nil {
				return err
			}
			pending, err := listener.AcceptTCP()
			if process {
				if pending != nil || !errors.Is(err, syscall.ESTALE) {
					return fmt.Errorf("closed process accept=%v,%v", pending, err)
				}
			} else {
				if err != nil {
					return err
				}
				pending.Close()
				if _, err := listener.AcceptTCP(); !errors.Is(err, net.ErrClosed) {
					return fmt.Errorf("drained accept=%v", err)
				}
			}
			_, err = conn.Write([]byte{'r'})
			return err
		}, func(conn *net.TCPConn) error {
			pending, err := net.Dial("tcp4", conn.RemoteAddr().String())
			if err != nil {
				return err
			}
			defer pending.Close()
			if _, err := conn.Write([]byte{'q'}); err != nil {
				return err
			}
			_, err = io.ReadFull(conn, make([]byte, 1))
			return err
		}},
		{"addresses-bind-rebind", func(listener *net.TCPListener, conn *net.TCPConn, process bool) error {
			if conn.LocalAddr().String() != listener.Addr().String() || conn.RemoteAddr().(*net.TCPAddr).Port < 40000 {
				return fmt.Errorf("addresses=%v,%v", conn.LocalAddr(), conn.RemoteAddr())
			}
			duplicate, err := net.Listen("tcp4", listener.Addr().String())
			if duplicate != nil {
				duplicate.Close()
				return fmt.Errorf("duplicate bind succeeded")
			}
			if err == nil {
				return fmt.Errorf("duplicate bind has no error")
			}
			address := listener.Addr().String()
			if err := listener.Close(); err != nil {
				return err
			}
			err = listener.Close()
			if process && !errors.Is(err, syscall.ESTALE) || !process && !errors.Is(err, net.ErrClosed) {
				return fmt.Errorf("repeat close=%v", err)
			}
			rebound, err := net.Listen("tcp4", address)
			if err != nil {
				return err
			}
			return rebound.Close()
		}, func(conn *net.TCPConn) error {
			if conn.RemoteAddr().(*net.TCPAddr).Port != 7233 && conn.RemoteAddr().(*net.TCPAddr).IP.String() != "127.0.0.1" {
				return fmt.Errorf("remote address=%v", conn.RemoteAddr())
			}
			return nil
		}},
		{"short-read-chunk-half-close-final-bytes", func(_ *net.TCPListener, conn *net.TCPConn, _ bool) error {
			var got []byte
			buffer := make([]byte, 997)
			for {
				n, err := conn.Read(buffer)
				got = append(got, buffer[:n]...)
				if err == io.EOF {
					break
				}
				if err != nil {
					return err
				}
			}
			if !bytes.Equal(got, bytes.Repeat([]byte{'a'}, (64<<10)+7)) {
				return fmt.Errorf("final data=%d", len(got))
			}
			_, err := conn.Write([]byte{'r'})
			return err
		}, func(conn *net.TCPConn) error {
			payload := bytes.Repeat([]byte{'a'}, (64<<10)+7)
			if n, err := conn.Write(payload); n != len(payload) || err != nil {
				return fmt.Errorf("write=%d,%v", n, err)
			}
			if err := conn.CloseWrite(); err != nil {
				return err
			}
			if n, err := conn.Write([]byte{'x'}); n != 0 || !errors.Is(err, net.ErrClosed) {
				return fmt.Errorf("half-close write=%d,%v", n, err)
			}
			buffer := make([]byte, 1)
			if _, err := io.ReadFull(conn, buffer); err != nil || buffer[0] != 'r' {
				return fmt.Errorf("reply=%q,%v", buffer, err)
			}
			return nil
		}},
		{"empty-io-after-close", func(_ *net.TCPListener, conn *net.TCPConn, _ bool) error { return networkHandleEmptyIO(conn) }, networkHandleEmptyIO},
		{"deadline-change-clear-read", func(listener *net.TCPListener, conn *net.TCPConn, _ bool) error {
			buffer := make([]byte, 1)
			if err := conn.SetDeadline(time.Now().Add(-time.Second)); err != nil {
				return err
			}
			if n, err := conn.Read(buffer); n != 0 || !errors.Is(err, os.ErrDeadlineExceeded) {
				return fmt.Errorf("expired read=%d,%v", n, err)
			}
			if err := conn.SetReadDeadline(time.Time{}); err != nil {
				return err
			}
			result := make(chan error, 1)
			go func() {
				n, err := conn.Read(buffer)
				if n != 0 || !errors.Is(err, os.ErrDeadlineExceeded) {
					result <- fmt.Errorf("changed read=%d,%v", n, err)
				} else {
					result <- nil
				}
			}()
			control, err := listener.AcceptTCP()
			if err != nil {
				return err
			}
			defer control.Close()
			if _, err := io.ReadFull(control, make([]byte, 1)); err != nil {
				return err
			}
			if err := conn.SetReadDeadline(time.Now().Add(-time.Second)); err != nil {
				return err
			}
			if err := <-result; err != nil {
				return err
			}
			if err := conn.SetDeadline(time.Time{}); err != nil {
				return err
			}
			if _, err := conn.Write([]byte{'r'}); err != nil {
				return err
			}
			if n, err := conn.Read(buffer); n != 1 || err != nil || buffer[0] != 'z' {
				return fmt.Errorf("cleared read=%d,%v,%q", n, err, buffer)
			}
			return nil
		}, func(conn *net.TCPConn) error {
			control, err := net.Dial("tcp4", conn.RemoteAddr().String())
			if err != nil {
				return err
			}
			defer control.Close()
			// This node has no outstanding model request. Its timer can advance
			// after the peer's data Read blocks, then TCP wakes the peer owner.
			<-time.After(time.Millisecond)
			if _, err := control.Write([]byte{'d'}); err != nil {
				return err
			}
			buffer := make([]byte, 1)
			if _, err := io.ReadFull(conn, buffer); err != nil {
				return err
			}
			_, err = conn.Write([]byte{'z'})
			return err
		}},
		{"partial-write-deadline-closure", func(_ *net.TCPListener, conn *net.TCPConn, _ bool) error {
			if _, err := conn.Write([]byte{'r'}); err != nil {
				return err
			}
			<-time.After(10 * time.Millisecond)
			return conn.CloseRead()
		}, func(conn *net.TCPConn) error {
			buffer := make([]byte, 1)
			if _, err := io.ReadFull(conn, buffer); err != nil {
				return err
			}
			if err := conn.SetWriteDeadline(time.Now().Add(time.Millisecond)); err != nil {
				return err
			}
			n, err := conn.Write(make([]byte, 65*(64<<10)))
			if n != 64*(64<<10) || !errors.Is(err, os.ErrDeadlineExceeded) {
				return fmt.Errorf("partial write=%d,%v", n, err)
			}
			if err := conn.SetWriteDeadline(time.Time{}); err != nil {
				return err
			}
			if n, err := conn.Write([]byte{'x'}); n != 0 || !errors.Is(err, net.ErrClosed) {
				return fmt.Errorf("closed write=%d,%v", n, err)
			}
			return nil
		}},
	}
}
func networkHandleEmptyIO(conn *net.TCPConn) error {
	if err := conn.Close(); err != nil {
		return err
	}
	if n, err := conn.Read(nil); n != 0 || err != nil {
		return fmt.Errorf("empty read=%d,%v", n, err)
	}
	if n, err := conn.Write(nil); n != 0 || err != nil {
		return fmt.Errorf("empty write=%d,%v", n, err)
	}
	return nil
}
