// io_net_races drives the in-memory TCP loopback through the close, deadline,
// and backlog races named by os.Args[1] and prints "ok" when the modeled
// outcome holds.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

// backlog mirrors the model's pending-connection limit per listener.
const backlog = 64

// pendingChunks mirrors the model's per-connection queue of unread writes.
const pendingChunks = 64

var cases = map[string]func() error{
	"close-after-write": closeAfterWrite,
	"accept-close":      acceptClose,
	"write-close":       writeClose,
	"read-close":        readClose,
	"read-close-read":   readCloseRead,
	"accept-deadline":   acceptDeadline,
	"write-deadline":    writeDeadline,
	"dial-cancel":       dialCancel,
	"dial-close":        dialClose,
	"port-exhaustion":   portExhaustion,
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: io_net_races <case>")
		os.Exit(2)
	}
	run, ok := cases[os.Args[1]]
	if !ok {
		fmt.Fprintln(os.Stderr, "unknown case:", os.Args[1])
		os.Exit(2)
	}
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("ok")
}

func listen() (*net.TCPListener, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	return listener.(*net.TCPListener), nil
}

func dial(listener *net.TCPListener) (*net.TCPConn, error) {
	connection, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		return nil, err
	}
	return connection.(*net.TCPConn), nil
}

func pair() (*net.TCPListener, *net.TCPConn, *net.TCPConn, error) {
	listener, err := listen()
	if err != nil {
		return nil, nil, nil, err
	}
	client, err := dial(listener)
	if err != nil {
		return nil, nil, nil, err
	}
	server, err := listener.AcceptTCP()
	if err != nil {
		return nil, nil, nil, err
	}
	return listener, client, server, nil
}

func expectClosed(operation string, err error) error {
	if err == nil || errors.Is(err, io.EOF) || errors.Is(err, os.ErrDeadlineExceeded) {
		return fmt.Errorf("%s = %v, want a closed-connection error", operation, err)
	}
	return nil
}

func expectDeadline(operation string, err error) error {
	if !errors.Is(err, os.ErrDeadlineExceeded) {
		return fmt.Errorf("%s = %v, want a deadline error", operation, err)
	}
	return nil
}

// settle lets every runnable goroutine block before the caller races it; the
// virtual clock only advances once nothing else can run.
func settle() {
	time.Sleep(time.Millisecond)
}

func closeAfterWrite() error {
	_, client, server, err := pair()
	if err != nil {
		return err
	}
	if _, err := client.Write([]byte("final")); err != nil {
		return err
	}
	if err := client.Close(); err != nil {
		return err
	}
	received, err := io.ReadAll(server)
	if err != nil {
		return err
	}
	if string(received) != "final" {
		return fmt.Errorf("received %q after close", received)
	}
	return server.Close()
}

func acceptClose() error {
	listener, err := listen()
	if err != nil {
		return err
	}
	client, err := dial(listener)
	if err != nil {
		return err
	}
	if err := listener.Close(); err != nil {
		return err
	}
	server, err := listener.AcceptTCP()
	if err != nil {
		return fmt.Errorf("accept of a queued connection after close: %w", err)
	}
	if server.RemoteAddr().String() != client.LocalAddr().String() {
		return fmt.Errorf("accepted %v, want %v", server.RemoteAddr(), client.LocalAddr())
	}
	if _, err := listener.AcceptTCP(); err == nil {
		return errors.New("accept on a drained closed listener succeeded")
	}
	return nil
}

func writeClose() error {
	_, client, server, err := pair()
	if err != nil {
		return err
	}
	if err := client.CloseWrite(); err != nil {
		return err
	}
	written, err := client.Write([]byte("late"))
	if written != 0 {
		return fmt.Errorf("wrote %d bytes after CloseWrite", written)
	}
	if err := expectClosed("write after CloseWrite", err); err != nil {
		return err
	}
	if _, err := server.Read(make([]byte, 1)); !errors.Is(err, io.EOF) {
		return fmt.Errorf("peer read = %v, want EOF", err)
	}
	return nil
}

func readClose() error {
	_, client, _, err := pair()
	if err != nil {
		return err
	}
	return interruptRead(client, client.Close)
}

func readCloseRead() error {
	_, client, _, err := pair()
	if err != nil {
		return err
	}
	return interruptRead(client, client.CloseRead)
}

func interruptRead(connection *net.TCPConn, interrupt func() error) error {
	results := make(chan error, 1)
	go func() {
		read, err := connection.Read(make([]byte, 1))
		if read != 0 {
			results <- fmt.Errorf("interrupted read returned %d bytes", read)
			return
		}
		results <- expectClosed("interrupted read", err)
	}()
	settle()
	if err := interrupt(); err != nil {
		return err
	}
	return <-results
}

func acceptDeadline() error {
	listener, err := listen()
	if err != nil {
		return err
	}
	if _, err := dial(listener); err != nil {
		return err
	}
	if err := listener.SetDeadline(time.Now().Add(-time.Second)); err != nil {
		return err
	}
	if _, err := listener.AcceptTCP(); err != nil {
		return fmt.Errorf("accept of a queued connection past the deadline: %w", err)
	}
	_, err = listener.AcceptTCP()
	return expectDeadline("accept on an empty backlog past the deadline", err)
}

func writeDeadline() error {
	_, client, server, err := pair()
	if err != nil {
		return err
	}
	if err := client.SetWriteDeadline(time.Now().Add(-time.Second)); err != nil {
		return err
	}
	for index := range pendingChunks {
		if _, err := client.Write([]byte{byte(index)}); err != nil {
			return fmt.Errorf("write %d into free buffer past the deadline: %w", index, err)
		}
	}
	written, err := client.Write([]byte{0xff})
	if written != 0 {
		return fmt.Errorf("wrote %d bytes into a full buffer", written)
	}
	if err := expectDeadline("write into a full buffer past the deadline", err); err != nil {
		return err
	}
	received := make([]byte, pendingChunks)
	if _, err := io.ReadFull(server, received); err != nil {
		return err
	}
	for index, value := range received {
		if int(value) != index {
			return fmt.Errorf("received %v", received)
		}
	}
	return nil
}

func fillBacklog(listener *net.TCPListener) error {
	for range backlog {
		if _, err := dial(listener); err != nil {
			return err
		}
	}
	return nil
}

func dialCancel() error {
	listener, err := listen()
	if err != nil {
		return err
	}
	if err := fillBacklog(listener); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	results := make(chan error, 1)
	go func() {
		var dialer net.Dialer
		connection, err := dialer.DialContext(ctx, "tcp", listener.Addr().String())
		if err == nil {
			connection.Close()
			results <- errors.New("canceled dial connected")
			return
		}
		if !errors.Is(err, context.Canceled) {
			results <- fmt.Errorf("canceled dial = %v", err)
			return
		}
		results <- nil
	}()
	settle()
	cancel()
	if err := <-results; err != nil {
		return err
	}
	return drainBacklog(listener)
}

func drainBacklog(listener *net.TCPListener) error {
	for range backlog {
		if _, err := listener.AcceptTCP(); err != nil {
			return err
		}
	}
	if err := listener.SetDeadline(time.Now().Add(-time.Second)); err != nil {
		return err
	}
	_, err := listener.AcceptTCP()
	return expectDeadline("accept after draining the backlog", err)
}

func dialClose() error {
	listener, err := listen()
	if err != nil {
		return err
	}
	if err := fillBacklog(listener); err != nil {
		return err
	}
	results := make(chan error, 1)
	go func() {
		connection, err := dial(listener)
		if err == nil {
			connection.Close()
			results <- errors.New("dial completed against a closed listener")
			return
		}
		results <- nil
	}()
	settle()
	if err := listener.Close(); err != nil {
		return err
	}
	return <-results
}

func portExhaustion() error {
	listener, err := listen()
	if err != nil {
		return err
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		return err
	}
	const clientPorts = 65535 - 40000 + 1
	for attempt := 0; ; attempt++ {
		_, err := net.Dial("tcp", address)
		if err == nil {
			return errors.New("dial to a closed port connected")
		}
		if strings.Contains(err.Error(), "network resources exhausted") {
			if attempt != clientPorts {
				return fmt.Errorf("client ports exhausted after %d dials, want %d", attempt, clientPorts)
			}
			return nil
		}
		if attempt > clientPorts {
			return fmt.Errorf("client ports never exhausted: %v", err)
		}
	}
}
