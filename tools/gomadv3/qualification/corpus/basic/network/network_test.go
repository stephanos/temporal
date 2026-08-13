package network

import (
	"io"
	"net"
	"testing"
)

func TestLoopbackRoundTripPreservesMessage(t *testing.T) {
	listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := listener.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	serverResult := make(chan error, 1)
	go func() {
		connection, acceptErr := listener.AcceptTCP()
		if acceptErr != nil {
			serverResult <- acceptErr
			return
		}
		defer connection.Close()
		message := make([]byte, 4)
		if _, acceptErr = io.ReadFull(connection, message); acceptErr == nil {
			_, acceptErr = connection.Write(append([]byte("ack:"), message...))
		}
		serverResult <- acceptErr
	}()
	connection, err := net.DialTCP("tcp4", nil, listener.Addr().(*net.TCPAddr))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = connection.Write([]byte("ping")); err != nil {
		t.Fatal(err)
	}
	if err = connection.CloseWrite(); err != nil {
		t.Fatal(err)
	}
	response, err := io.ReadAll(connection)
	if err != nil {
		t.Fatal(err)
	}
	if err = connection.Close(); err != nil {
		t.Fatal(err)
	}
	if err = <-serverResult; err != nil {
		t.Fatal(err)
	}
	if string(response) != "ack:ping" {
		t.Fatalf("response = %q", response)
	}
}
