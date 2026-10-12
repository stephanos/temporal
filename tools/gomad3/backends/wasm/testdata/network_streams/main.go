package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"syscall"
	"time"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func pair(listener *net.TCPListener) (*net.TCPConn, *net.TCPConn) {
	conn, err := net.DialTCP("tcp", nil, listener.Addr().(*net.TCPAddr))
	check(err)
	peer, err := listener.AcceptTCP()
	check(err)
	for _, addr := range []net.Addr{conn.LocalAddr(), conn.RemoteAddr(), peer.LocalAddr(), peer.RemoteAddr()} {
		tcp := addr.(*net.TCPAddr)
		if tcp.Port == 0 || !tcp.IP.IsLoopback() {
			panic("invalid stream address")
		}
	}
	return conn, peer
}

func closeConn(conn *net.TCPConn) {
	err := conn.Close()
	if err != nil && !errors.Is(err, net.ErrClosed) && !errors.Is(err, syscall.ECONNRESET) {
		panic(err)
	}
}

func readBytes(conn *net.TCPConn, want []byte) {
	got := make([]byte, len(want))
	_, err := io.ReadFull(conn, got)
	check(err)
	if !bytes.Equal(got, want) {
		panic("stream payload mismatch")
	}
}

func at(start time.Time, elapsed time.Duration) {
	if delay := elapsed - time.Since(start); delay > 0 {
		time.Sleep(delay)
	}
}

func main() {
	start := time.Now()
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9000})
	check(err)
	defer listener.Close()
	client, server := pair(listener)
	payload := make([]byte, 3*(64<<10)+137)
	for i := range payload {
		payload[i] = byte(i*31 + i/251)
	}
	n, err := client.Write(payload)
	check(err)
	if n != len(payload) {
		panic("short stream write")
	}
	check(client.CloseWrite())
	readBytes(server, payload)
	if n, err := server.Read(make([]byte, 1)); n != 0 || err != io.EOF {
		panic(fmt.Sprintf("half-close drain: n=%d err=%v", n, err))
	}
	ack := []byte("half-close acknowledgement")
	_, err = server.Write(ack)
	check(err)
	check(server.CloseWrite())
	readBytes(client, ack)
	if _, err := client.Read(make([]byte, 1)); err != io.EOF {
		panic(fmt.Sprintf("reverse half-close: %v", err))
	}
	closeConn(client)
	closeConn(server)
	fmt.Printf("chunks %d sha256 %x half-close drained\n", len(payload), sha256.Sum256(payload))

	client, server = pair(listener)
	check(server.SetReadDeadline(time.Now().Add(3 * time.Millisecond)))
	if _, err := server.Read(make([]byte, 1)); !errors.Is(err, os.ErrDeadlineExceeded) {
		panic(fmt.Sprintf("read deadline: %v", err))
	}
	check(server.SetDeadline(time.Time{}))
	check(server.CloseRead())
	if _, err := client.Write([]byte("closed reader")); !errors.Is(err, syscall.ECONNRESET) {
		panic(fmt.Sprintf("closed read: %v", err))
	}
	closeConn(client)
	closeConn(server)
	check(listener.SetDeadline(time.Now().Add(2 * time.Millisecond)))
	if _, err := listener.AcceptTCP(); !errors.Is(err, os.ErrDeadlineExceeded) {
		panic(fmt.Sprintf("accept deadline: %v", err))
	}
	check(listener.SetDeadline(time.Time{}))
	fmt.Println("read and accept deadlines; close-read rejected write")

	at(start, 50*time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Millisecond)
	_, err = (&net.Dialer{}).DialContext(ctx, "tcp", listener.Addr().String())
	cancel()
	var timeout net.Error
	if !errors.As(err, &timeout) || !timeout.Timeout() {
		panic(fmt.Sprintf("dial deadline: %v", err))
	}
	ctx, cancel = context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		time.Sleep(2 * time.Millisecond)
		cancel()
		close(done)
	}()
	_, err = (&net.Dialer{}).DialContext(ctx, "tcp", listener.Addr().String())
	<-done
	if !errors.Is(err, context.Canceled) {
		panic(fmt.Sprintf("dial cancellation: %v", err))
	}
	fmt.Println("partition dial deadline and cancellation")

	at(start, 90*time.Millisecond)
	client, server = pair(listener)
	_, err = client.Write([]byte("before reset"))
	check(err)
	readBytes(server, []byte("before reset"))
	at(start, 120*time.Millisecond)
	if _, err := client.Write([]byte("reset connection")); !errors.Is(err, syscall.ECONNRESET) {
		panic(fmt.Sprintf("named reset: %v", err))
	}
	closeConn(client)
	closeConn(server)
	client, server = pair(listener)
	_, err = client.Write([]byte("new connection survived reset"))
	check(err)
	readBytes(server, []byte("new connection survived reset"))
	closeConn(client)
	closeConn(server)
	fmt.Println("named reset; new connection survived")
}
