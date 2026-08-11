package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

func main() {
	listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		panic(err)
	}
	defer listener.Close()
	address := listener.Addr().(*net.TCPAddr)
	if address.Port != 20000 {
		panic(fmt.Sprintf("host-backed TCP listener selected port %d", address.Port))
	}
	if _, err = listener.SyscallConn(); err == nil {
		panic("host-backed TCP listener exposed a raw connection")
	}
	serverError := make(chan error, 1)
	go func() {
		connection, acceptErr := listener.AcceptTCP()
		if acceptErr != nil {
			serverError <- acceptErr
			return
		}
		defer connection.Close()
		request := make([]byte, 4)
		if _, acceptErr = io.ReadFull(connection, request); acceptErr == nil && string(request) != "ping" {
			acceptErr = fmt.Errorf("request = %q", request)
		}
		if acceptErr == nil {
			_, acceptErr = connection.Write([]byte("pong"))
		}
		if acceptErr == nil {
			acceptErr = connection.CloseWrite()
		}
		serverError <- acceptErr
	}()
	connection, err := net.DialTCP("tcp4", nil, address)
	if err != nil {
		panic(err)
	}
	defer connection.Close()
	if _, err = connection.Write([]byte("ping")); err != nil {
		panic(err)
	}
	if err = connection.CloseWrite(); err != nil {
		panic(err)
	}
	response, err := io.ReadAll(connection)
	if err != nil {
		panic(err)
	}
	if string(response) != "pong" {
		panic(fmt.Sprintf("response = %q", response))
	}
	if err = <-serverError; err != nil {
		panic(err)
	}
	genericListener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}
	defer genericListener.Close()
	genericTCPListener := genericListener.(*net.TCPListener)
	genericAddress := genericListener.Addr().(*net.TCPAddr)
	if genericAddress.Port != 20001 {
		panic(fmt.Sprintf("host-backed generic listener selected port %d", genericAddress.Port))
	}
	if _, err = genericTCPListener.SyscallConn(); err == nil {
		panic("host-backed generic listener exposed a raw connection")
	}
	serverError = make(chan error, 1)
	go func() {
		genericConnection, acceptErr := genericListener.Accept()
		if acceptErr == nil {
			defer genericConnection.Close()
			request := make([]byte, 4)
			_, acceptErr = io.ReadFull(genericConnection, request)
			if acceptErr == nil {
				_, acceptErr = genericConnection.Write(request)
			}
		}
		serverError <- acceptErr
	}()
	dialer := net.Dialer{}
	genericConnection, err := dialer.DialContext(context.Background(), "tcp", genericAddress.String())
	if err != nil {
		panic(err)
	}
	if _, err = genericConnection.Write([]byte("echo")); err != nil {
		panic(err)
	}
	response = make([]byte, 4)
	if _, err = io.ReadFull(genericConnection, response); err != nil {
		panic(err)
	}
	if err = genericConnection.Close(); err != nil {
		panic(err)
	}
	if string(response) != "echo" {
		panic(fmt.Sprintf("generic response = %q", response))
	}
	if err = <-serverError; err != nil {
		panic(err)
	}
	deadlineListener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		panic(err)
	}
	defer deadlineListener.Close()
	if deadlineListener.Addr().(*net.TCPAddr).Port != 20002 {
		panic("deadline listener did not use the in-memory port range")
	}
	deadlineClient, err := net.DialTCP("tcp4", nil, deadlineListener.Addr().(*net.TCPAddr))
	if err != nil {
		panic(err)
	}
	defer deadlineClient.Close()
	deadlineServer, err := deadlineListener.AcceptTCP()
	if err != nil {
		panic(err)
	}
	defer deadlineServer.Close()
	if err = deadlineClient.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		panic(err)
	}
	if _, err = deadlineClient.Read(make([]byte, 1)); !errors.Is(err, os.ErrDeadlineExceeded) {
		panic(fmt.Sprintf("read deadline error = %v", err))
	}
	if err = deadlineClient.SetReadDeadline(time.Time{}); err != nil {
		panic(err)
	}
	if _, err = deadlineServer.Write([]byte("x")); err != nil {
		panic(err)
	}
	if _, err = io.ReadFull(deadlineClient, make([]byte, 1)); err != nil {
		panic(err)
	}
	if err = deadlineClient.SetWriteDeadline(time.Now().Add(time.Second)); err != nil {
		panic(err)
	}
	largeWrite := make([]byte, 65*(64<<10))
	written, err := deadlineClient.Write(largeWrite)
	if !errors.Is(err, os.ErrDeadlineExceeded) || written == 0 || written >= len(largeWrite) {
		panic(fmt.Sprintf("write deadline result = %d, %v", written, err))
	}
	acceptListener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		panic(err)
	}
	defer acceptListener.Close()
	if err = acceptListener.SetDeadline(time.Now().Add(time.Second)); err != nil {
		panic(err)
	}
	if _, err = acceptListener.AcceptTCP(); !errors.Is(err, os.ErrDeadlineExceeded) {
		panic(fmt.Sprintf("accept deadline error = %v", err))
	}
	if err = acceptListener.SetDeadline(time.Time{}); err != nil {
		panic(err)
	}
	acceptClient, err := net.DialTCP("tcp4", nil, acceptListener.Addr().(*net.TCPAddr))
	if err != nil {
		panic(err)
	}
	defer acceptClient.Close()
	accepted, err := acceptListener.AcceptTCP()
	if err != nil {
		panic(err)
	}
	if err = accepted.Close(); err != nil {
		panic(err)
	}
	cancelContext, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = (&net.Dialer{}).DialContext(cancelContext, "tcp", acceptListener.Addr().String()); !errors.Is(err, context.Canceled) {
		panic(fmt.Sprintf("canceled dial error = %v", err))
	}
	copyListener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		panic(err)
	}
	defer copyListener.Close()
	copyClient, err := net.DialTCP("tcp4", nil, copyListener.Addr().(*net.TCPAddr))
	if err != nil {
		panic(err)
	}
	defer copyClient.Close()
	copyServer, err := copyListener.AcceptTCP()
	if err != nil {
		panic(err)
	}
	defer copyServer.Close()
	if copied, copyErr := io.Copy(copyClient, struct{ io.Reader }{strings.NewReader("copy")}); copyErr != nil || copied != 4 {
		panic(fmt.Sprintf("ReadFrom result = %d, %v", copied, copyErr))
	}
	if err = copyClient.CloseWrite(); err != nil {
		panic(err)
	}
	var copied bytes.Buffer
	if length, copyErr := io.Copy(&copied, copyServer); copyErr != nil || length != 4 || copied.String() != "copy" {
		panic(fmt.Sprintf("WriteTo result = %d, %v, %q", length, copyErr, copied.String()))
	}
	if err = copyClient.SetLinger(0); err != nil {
		panic(err)
	}
	if err = copyClient.SetKeepAlive(true); err != nil {
		panic(err)
	}
	if err = copyClient.SetKeepAlivePeriod(time.Second); err != nil {
		panic(err)
	}
	if err = copyClient.SetKeepAliveConfig(net.KeepAliveConfig{Enable: true}); err != nil {
		panic(err)
	}
	if err = copyClient.SetNoDelay(true); err != nil {
		panic(err)
	}
	if multipath, multipathErr := copyClient.MultipathTCP(); multipathErr != nil || multipath {
		panic(fmt.Sprintf("MultipathTCP result = %v, %v", multipath, multipathErr))
	}
	if file, fileErr := copyClient.File(); fileErr == nil || file != nil {
		panic("in-memory TCP connection exposed a file")
	}
	fmt.Println("ok")
}
