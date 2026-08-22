package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"strings"
	"time"
)

const (
	attempts      = 16
	closeAttempts = 4096
)

func main() {
	defer func() {
		if failure := recover(); failure != nil {
			fmt.Printf("FAIL: %v\n", failure)
		}
	}()
	if len(os.Args) != 2 {
		panic(fmt.Sprintf("expected one test case argument, got %d", len(os.Args)-1))
	}
	switch testCase := os.Args[1]; testCase {
	case "close-after-write":
		testCloseAfterWrite()
	case "accept-close":
		testAcceptClose()
	case "write-close":
		testWriteClose()
	case "read-close":
		testReadClose()
	case "read-close-read":
		testReadCloseRead()
	case "accept-deadline":
		testAcceptDeadline()
	case "write-deadline":
		testWriteDeadline()
	case "dial-cancel":
		testDialCancel()
	case "dial-close":
		testDialClose()
	case "port-exhaustion":
		testPortExhaustion()
	default:
		panic(fmt.Sprintf("unknown test case %q", testCase))
	}
	fmt.Println("ok")
}

func testCloseAfterWrite() {
	for attempt := 0; attempt < closeAttempts; attempt++ {
		listener, client, server := connectedPair()
		result := make(chan struct {
			content []byte
			err     error
		}, 1)
		start := make(chan struct{})
		go func() {
			<-start
			content, err := io.ReadAll(server)
			result <- struct {
				content []byte
				err     error
			}{content: content, err: err}
		}()
		writeResult := make(chan error, 1)
		go func() {
			<-start
			if _, err := client.Write([]byte("final")); err != nil {
				writeResult <- err
				return
			}
			writeResult <- client.CloseWrite()
		}()
		close(start)
		if err := <-writeResult; err != nil {
			panic(err)
		}
		read := <-result
		if read.err != nil || string(read.content) != "final" {
			panic(fmt.Sprintf("attempt %d: ReadAll() = %q, %v", attempt, read.content, read.err))
		}
		closePair(listener, client, server)
	}
}

func testAcceptClose() {
	for attempt := 0; attempt < attempts; attempt++ {
		listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
		if err != nil {
			panic(err)
		}
		client, err := net.DialTCP("tcp4", nil, listener.Addr().(*net.TCPAddr))
		if err != nil {
			panic(err)
		}
		if err := listener.Close(); err != nil {
			panic(err)
		}
		server, err := listener.AcceptTCP()
		if err != nil {
			panic(fmt.Sprintf("attempt %d: AcceptTCP() error = %v", attempt, err))
		}
		closeConnections(client, server)
	}
}

func testWriteClose() {
	for attempt := 0; attempt < attempts; attempt++ {
		listener, client, server := connectedPair()
		if err := client.CloseWrite(); err != nil {
			panic(err)
		}
		if written, err := client.Write([]byte("x")); written != 0 || err == nil {
			panic(fmt.Sprintf("attempt %d: Write() = %d, %v after CloseWrite", attempt, written, err))
		}
		closePair(listener, client, server)

		listener, client, server = connectedPair()
		if err := server.CloseRead(); err != nil {
			panic(err)
		}
		if written, err := client.Write([]byte("x")); written != 0 || err == nil {
			panic(fmt.Sprintf("attempt %d: Write() = %d, %v after peer CloseRead", attempt, written, err))
		}
		closePair(listener, client, server)
	}
}

func testReadClose() {
	listener, client, server := connectedPair()
	if _, err := client.Write([]byte("queued")); err != nil {
		panic(err)
	}
	if err := server.Close(); err != nil {
		panic(err)
	}
	buffer := make([]byte, len("queued"))
	if read, err := server.Read(buffer); read != 0 || err == nil {
		panic(fmt.Sprintf("Read() = %d, %v after Close", read, err))
	}
	closeListenerAndConnections(listener, []*net.TCPConn{client})
}

func testReadCloseRead() {
	listener, client, server := connectedPair()
	if _, err := client.Write([]byte("queued")); err != nil {
		panic(err)
	}
	if err := server.CloseRead(); err != nil {
		panic(err)
	}
	buffer := make([]byte, len("queued"))
	if read, err := server.Read(buffer); read != 0 || err == nil {
		panic(fmt.Sprintf("Read() = %d, %v after CloseRead", read, err))
	}
	closePair(listener, client, server)
}

func testAcceptDeadline() {
	for attempt := 0; attempt < attempts; attempt++ {
		listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
		if err != nil {
			panic(err)
		}
		client, err := net.DialTCP("tcp4", nil, listener.Addr().(*net.TCPAddr))
		if err != nil {
			panic(err)
		}
		if err := listener.SetDeadline(time.Now().Add(-time.Second)); err != nil {
			panic(err)
		}
		server, err := listener.AcceptTCP()
		if err != nil {
			panic(fmt.Sprintf("attempt %d: AcceptTCP() error = %v", attempt, err))
		}
		closePair(listener, client, server)
	}
}

func testWriteDeadline() {
	for attempt := 0; attempt < attempts; attempt++ {
		listener, client, server := connectedPair()
		if err := client.SetWriteDeadline(time.Now().Add(-time.Second)); err != nil {
			panic(err)
		}
		if written, err := client.Write([]byte("x")); written != 1 || err != nil {
			panic(fmt.Sprintf("attempt %d: Write() = %d, %v with buffer space", attempt, written, err))
		}
		buffer := make([]byte, 1)
		if _, err := io.ReadFull(server, buffer); err != nil || string(buffer) != "x" {
			panic(fmt.Sprintf("attempt %d: ReadFull() = %q, %v", attempt, buffer, err))
		}
		closePair(listener, client, server)
	}
}

func testDialCancel() {
	for attempt := 0; attempt < 1; attempt++ {
		listener, clients := fullListener()
		ctx, cancel := context.WithCancel(context.Background())
		result := make(chan struct {
			connection net.Conn
			err        error
		}, 1)
		started := make(chan struct{})
		go func() {
			close(started)
			connection, err := (&net.Dialer{}).DialContext(ctx, "tcp4", listener.Addr().String())
			if err == nil {
				_ = connection.Close()
			}
			result <- struct {
				connection net.Conn
				err        error
			}{connection: connection, err: err}
		}()
		<-started
		runtime.Gosched()
		cancel()
		server, err := listener.AcceptTCP()
		if err != nil {
			panic(err)
		}
		dial := <-result
		if !errors.Is(dial.err, context.Canceled) || dial.connection != nil {
			panic(fmt.Sprintf("attempt %d: DialContext() = %v, %v", attempt, dial.connection, dial.err))
		}
		closeConnections(server)
		closeListenerAndConnections(listener, clients)
	}
}

func testDialClose() {
	for attempt := 0; attempt < attempts; attempt++ {
		listener, clients := fullListener()
		result := make(chan error, 1)
		started := make(chan struct{})
		go func() {
			close(started)
			connection, err := net.DialTCP("tcp4", nil, listener.Addr().(*net.TCPAddr))
			if connection != nil {
				_ = connection.Close()
			}
			result <- err
		}()
		<-started
		runtime.Gosched()
		if err := listener.Close(); err != nil {
			panic(err)
		}
		server, err := listener.AcceptTCP()
		if err != nil {
			panic(fmt.Sprintf("attempt %d: AcceptTCP() after Close() error = %v", attempt, err))
		}
		if err := <-result; err == nil || !strings.Contains(err.Error(), "connection refused") {
			panic(fmt.Sprintf("attempt %d: DialTCP() error = %v", attempt, err))
		}
		closeConnections(server)
		closeConnections(clients...)
	}
}

func testPortExhaustion() {
	for port := 20000; port <= 65535; port++ {
		listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
		if err != nil {
			panic(fmt.Sprintf("listener port %d: %v", port, err))
		}
		if got := listener.Addr().(*net.TCPAddr).Port; got != port {
			panic(fmt.Sprintf("listener port = %d, want %d", got, port))
		}
		if err := listener.Close(); err != nil {
			panic(err)
		}
	}
	if listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)}); listener != nil || err == nil || !strings.Contains(err.Error(), "network resources exhausted") {
		panic(fmt.Sprintf("ListenTCP() after exhaustion = %v, %v", listener, err))
	}
	for port := 40000; port <= 65535; port++ {
		connection, err := net.DialTCP("tcp4", nil, &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1})
		if connection != nil || err == nil || !strings.Contains(err.Error(), "connection refused") {
			panic(fmt.Sprintf("client port %d: DialTCP() = %v, %v", port, connection, err))
		}
	}
	if connection, err := net.DialTCP("tcp4", nil, &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1}); connection != nil || err == nil || !strings.Contains(err.Error(), "network resources exhausted") {
		panic(fmt.Sprintf("DialTCP() after exhaustion = %v, %v", connection, err))
	}
}

func connectedPair() (*net.TCPListener, *net.TCPConn, *net.TCPConn) {
	listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		panic(err)
	}
	client, err := net.DialTCP("tcp4", nil, listener.Addr().(*net.TCPAddr))
	if err != nil {
		panic(err)
	}
	server, err := listener.AcceptTCP()
	if err != nil {
		panic(err)
	}
	return listener, client, server
}

func fullListener() (*net.TCPListener, []*net.TCPConn) {
	listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		panic(err)
	}
	clients := make([]*net.TCPConn, 64)
	for index := range clients {
		clients[index], err = net.DialTCP("tcp4", nil, listener.Addr().(*net.TCPAddr))
		if err != nil {
			panic(err)
		}
	}
	return listener, clients
}

func closePair(listener *net.TCPListener, client, server *net.TCPConn) {
	closeListenerAndConnections(listener, []*net.TCPConn{client, server})
}

func closeListenerAndConnections(listener *net.TCPListener, connections []*net.TCPConn) {
	closeConnections(connections...)
	if err := listener.Close(); err != nil {
		panic(err)
	}
}

func closeConnections(connections ...*net.TCPConn) {
	for _, connection := range connections {
		if err := connection.Close(); err != nil {
			panic(err)
		}
	}
}
