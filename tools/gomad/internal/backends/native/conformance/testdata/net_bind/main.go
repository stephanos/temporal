// net_bind checks the in-memory network's bind contract that services rely on:
// an unspecified host binds the loopback listener, listeners and their
// addresses have the concrete net types, and a closed port can be bound again.
// It prints "net-bind ok" when every check held.
package main

import (
	"fmt"
	"io"
	"net"
	"os"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "net-bind:", err)
		os.Exit(1)
	}
	fmt.Println("net-bind ok")
}

func run() error {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return fmt.Errorf("listen on an unspecified host: %w", err)
	}
	tcpListener, ok := listener.(*net.TCPListener)
	if !ok {
		return fmt.Errorf("listener is %T, want *net.TCPListener", listener)
	}
	address, ok := tcpListener.Addr().(*net.TCPAddr)
	if !ok {
		return fmt.Errorf("listener address is %T, want *net.TCPAddr", tcpListener.Addr())
	}
	if !address.IP.Equal(net.IPv4(127, 0, 0, 1)) || address.Port == 0 {
		return fmt.Errorf("unspecified bind resolved to %v, want loopback with a port", address)
	}

	accepted := make(chan error, 1)
	go func() {
		connection, err := tcpListener.AcceptTCP()
		if err == nil {
			_, err = io.WriteString(connection, "ok")
			if closeErr := connection.Close(); err == nil {
				err = closeErr
			}
		}
		accepted <- err
	}()
	connection, err := net.Dial("tcp", address.String())
	if err != nil {
		return fmt.Errorf("dial the loopback listener: %w", err)
	}
	reply, err := io.ReadAll(connection)
	if err != nil || string(reply) != "ok" {
		return fmt.Errorf("read reply %q: %v", reply, err)
	}
	if err := connection.Close(); err != nil {
		return err
	}
	if err := <-accepted; err != nil {
		return fmt.Errorf("accept: %w", err)
	}

	port := address.Port
	if err := tcpListener.Close(); err != nil {
		return err
	}
	rebound, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		return fmt.Errorf("bind the closed port %d again: %w", port, err)
	}
	if _, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port)); err == nil {
		return fmt.Errorf("port %d was bound twice", port)
	}
	return rebound.Close()
}
