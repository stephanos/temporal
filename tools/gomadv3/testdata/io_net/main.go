package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"strings"
	"time"
)

func main() {
	_, err := net.ListenPacket("udp4", "127.0.0.1:0")
	requireUnsupported("ListenPacket", err)
	_, err = net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	requireUnsupported("ListenUDP", err)
	_, err = net.DialUDP("udp4", nil, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 20000})
	requireUnsupported("DialUDP", err)
	_, err = net.DialIP("ip4:icmp", nil, &net.IPAddr{IP: net.IPv4(127, 0, 0, 1)})
	requireUnsupported("DialIP", err)
	_, err = net.ListenIP("ip4:icmp", &net.IPAddr{IP: net.IPv4(127, 0, 0, 1)})
	requireUnsupported("ListenIP", err)
	_, err = net.DialUnix("unix", nil, &net.UnixAddr{Name: "gomad-host.sock", Net: "unix"})
	requireUnsupported("DialUnix", err)
	_, err = net.ListenUnix("unix", &net.UnixAddr{Name: "gomad-host.sock", Net: "unix"})
	requireUnsupported("ListenUnix", err)
	_, err = net.ListenUnixgram("unixgram", &net.UnixAddr{Name: "gomad-host.sock", Net: "unixgram"})
	requireUnsupported("ListenUnixgram", err)
	_, err = net.LookupHost("example.com")
	requireUnsupported("LookupHost", err)
	_, err = net.LookupTXT("example.com")
	requireUnsupported("LookupTXT", err)
	resolver := net.DefaultResolver
	_, err = resolver.LookupHost(context.Background(), "example.com")
	requireUnsupported("Resolver.LookupHost", err)
	_, err = resolver.LookupIP(context.Background(), "ip", "example.com")
	requireUnsupported("Resolver.LookupIP", err)
	_, err = resolver.LookupIPAddr(context.Background(), "example.com")
	requireUnsupported("Resolver.LookupIPAddr", err)
	_, err = resolver.LookupNetIP(context.Background(), "ip", "example.com")
	requireUnsupported("Resolver.LookupNetIP", err)
	_, err = resolver.LookupPort(context.Background(), "tcp", "http")
	requireUnsupported("Resolver.LookupPort", err)
	_, err = resolver.LookupCNAME(context.Background(), "example.com")
	requireUnsupported("Resolver.LookupCNAME", err)
	_, _, err = resolver.LookupSRV(context.Background(), "service", "tcp", "example.com")
	requireUnsupported("Resolver.LookupSRV", err)
	_, err = resolver.LookupMX(context.Background(), "example.com")
	requireUnsupported("Resolver.LookupMX", err)
	_, err = resolver.LookupNS(context.Background(), "example.com")
	requireUnsupported("Resolver.LookupNS", err)
	_, err = resolver.LookupTXT(context.Background(), "example.com")
	requireUnsupported("Resolver.LookupTXT", err)
	_, err = resolver.LookupAddr(context.Background(), "127.0.0.1")
	requireUnsupported("Resolver.LookupAddr", err)
	_, err = net.DialTCP("tcp4", nil, &net.TCPAddr{IP: net.IPv4(8, 8, 8, 8), Port: 53})
	requireUnsupported("non-loopback DialTCP", err)
	dialer := net.Dialer{}
	_, err = dialer.DialIP(context.Background(), "ip4", netip.Addr{}, netip.MustParseAddr("127.0.0.1"))
	requireUnsupported("Dialer.DialIP", err)
	_, err = dialer.DialTCP(context.Background(), "tcp4", netip.AddrPort{}, netip.MustParseAddrPort("127.0.0.1:20000"))
	requireUnsupported("Dialer.DialTCP", err)
	_, err = dialer.DialUDP(context.Background(), "udp4", netip.AddrPort{}, netip.MustParseAddrPort("127.0.0.1:20000"))
	requireUnsupported("Dialer.DialUDP", err)
	_, err = dialer.DialUnix(context.Background(), "unix", nil, &net.UnixAddr{Name: "gomad-host.sock", Net: "unix"})
	requireUnsupported("Dialer.DialUnix", err)
	_, err = net.FileConn(os.Stdout)
	requireUnsupported("FileConn", err)
	_, err = net.FileListener(os.Stdout)
	requireUnsupported("FileListener", err)
	_, err = net.FilePacketConn(os.Stdout)
	requireUnsupported("FilePacketConn", err)
	_, err = net.Interfaces()
	requireUnsupported("Interfaces", err)
	_, err = net.InterfaceAddrs()
	requireUnsupported("InterfaceAddrs", err)
	_, err = net.InterfaceByIndex(1)
	requireUnsupported("InterfaceByIndex", err)
	_, err = net.InterfaceByName("lo0")
	requireUnsupported("InterfaceByName", err)
	_, err = (&net.Interface{}).Addrs()
	requireUnsupported("Interface.Addrs", err)
	_, err = (&net.Interface{}).MulticastAddrs()
	requireUnsupported("Interface.MulticastAddrs", err)
	_, err = net.ListenMulticastUDP("udp4", nil, &net.UDPAddr{IP: net.IPv4(224, 0, 0, 1)})
	requireUnsupported("ListenMulticastUDP", err)
	_, err = net.ResolveIPAddr("ip4", "localhost")
	requireUnsupported("ResolveIPAddr", err)
	_, err = net.ResolveTCPAddr("tcp4", "localhost:80")
	requireUnsupported("ResolveTCPAddr", err)
	_, err = net.ResolveUDPAddr("udp4", "localhost:80")
	requireUnsupported("ResolveUDPAddr", err)
	_, err = net.ResolveUnixAddr("unix", "gomad-host.sock")
	requireUnsupported("ResolveUnixAddr", err)

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
	if connection.LocalAddr() == nil || connection.RemoteAddr() == nil {
		panic("in-memory TCP connection has no addresses")
	}
	if err = connection.SetDeadline(time.Time{}); err != nil {
		panic(err)
	}
	if err = connection.SetReadBuffer(64 << 10); err != nil {
		panic(err)
	}
	if err = connection.SetWriteBuffer(64 << 10); err != nil {
		panic(err)
	}
	if _, err = connection.SyscallConn(); err == nil {
		panic("in-memory TCP connection exposed a raw connection")
	}
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
	if err = copyClient.CloseRead(); err != nil {
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
	if file, fileErr := copyListener.File(); fileErr == nil || file != nil {
		panic("in-memory TCP listener exposed a file")
	}
	fmt.Println("ok")
}

func requireUnsupported(operation string, err error) {
	if err == nil || !strings.Contains(err.Error(), "unsupported Gomad network operation") {
		panic(fmt.Sprintf("%s error = %v", operation, err))
	}
}
