// io_net drives every modeled net operation over the in-memory TCP loopback
// and prints "ok".
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("ok")
}

func run() error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	tcpListener, ok := listener.(*net.TCPListener)
	if !ok {
		return fmt.Errorf("listener type %T", listener)
	}
	address, ok := tcpListener.Addr().(*net.TCPAddr)
	if !ok || address.Port == 0 {
		return fmt.Errorf("listener address %v", tcpListener.Addr())
	}
	dialed, err := net.Dial("tcp", address.String())
	if err != nil {
		return err
	}
	client, ok := dialed.(*net.TCPConn)
	if !ok {
		return fmt.Errorf("connection type %T", dialed)
	}
	accepted, err := tcpListener.Accept()
	if err != nil {
		return err
	}
	server, ok := accepted.(*net.TCPConn)
	if !ok {
		return fmt.Errorf("accepted connection type %T", accepted)
	}
	if err := configure(client); err != nil {
		return err
	}
	if client.LocalAddr().String() != server.RemoteAddr().String() || client.RemoteAddr().String() != server.LocalAddr().String() {
		return fmt.Errorf("addresses %v/%v vs %v/%v", client.LocalAddr(), client.RemoteAddr(), server.LocalAddr(), server.RemoteAddr())
	}
	if err := exerciseDenied(tcpListener, client); err != nil {
		return err
	}
	if err := exchange(client, server, "ping"); err != nil {
		return err
	}
	if err := exchange(server, client, "pong"); err != nil {
		return err
	}
	if err := client.CloseWrite(); err != nil {
		return err
	}
	if _, err := server.Read(make([]byte, 1)); !errors.Is(err, io.EOF) {
		return fmt.Errorf("read after peer CloseWrite = %v, want EOF", err)
	}
	if err := server.CloseRead(); err != nil {
		return err
	}
	if err := client.Close(); err != nil {
		return err
	}
	if err := server.Close(); err != nil {
		return err
	}
	if err := tcpListener.SetDeadline(time.Now()); err != nil {
		return err
	}
	if _, err := tcpListener.AcceptTCP(); !errors.Is(err, os.ErrDeadlineExceeded) {
		return fmt.Errorf("accept past deadline = %v", err)
	}
	if err := tcpListener.Close(); err != nil {
		return err
	}
	if _, err := net.Dial("tcp", address.String()); err == nil {
		return errors.New("dial to a closed listener succeeded")
	}
	if _, err := tcpListener.AcceptTCP(); err == nil {
		return errors.New("accept on a closed listener succeeded")
	}
	addresses, err := net.DefaultResolver.LookupIPAddr(context.Background(), "localhost")
	if err != nil {
		return err
	}
	if len(addresses) != 1 || !addresses[0].IP.Equal(net.IPv4(127, 0, 0, 1)) {
		return fmt.Errorf("localhost = %v", addresses)
	}
	if interfaces, err := net.Interfaces(); err == nil {
		return fmt.Errorf("host interfaces are visible: %v", interfaces)
	}
	return nil
}

func configure(connection *net.TCPConn) error {
	future := time.Now().Add(time.Hour)
	steps := []struct {
		name string
		call func() error
	}{
		{"SetDeadline", func() error { return connection.SetDeadline(future) }},
		{"SetReadDeadline", func() error { return connection.SetReadDeadline(future) }},
		{"SetWriteDeadline", func() error { return connection.SetWriteDeadline(future) }},
		{"SetReadBuffer", func() error { return connection.SetReadBuffer(1 << 16) }},
		{"SetWriteBuffer", func() error { return connection.SetWriteBuffer(1 << 16) }},
		{"SetLinger", func() error { return connection.SetLinger(0) }},
		{"SetKeepAlive", func() error { return connection.SetKeepAlive(true) }},
		{"SetKeepAlivePeriod", func() error { return connection.SetKeepAlivePeriod(time.Minute) }},
		{"SetKeepAliveConfig", func() error { return connection.SetKeepAliveConfig(net.KeepAliveConfig{Enable: true}) }},
		{"SetNoDelay", func() error { return connection.SetNoDelay(true) }},
	}
	for _, step := range steps {
		if err := step.call(); err != nil {
			return fmt.Errorf("%s: %w", step.name, err)
		}
	}
	multipath, err := connection.MultipathTCP()
	if err != nil {
		return err
	}
	if multipath {
		return errors.New("loopback reported multipath TCP")
	}
	return nil
}

func exchange(from, to *net.TCPConn, message string) error {
	if _, err := from.Write([]byte(message)); err != nil {
		return err
	}
	buffer := make([]byte, len(message))
	if _, err := io.ReadFull(to, buffer); err != nil {
		return err
	}
	if string(buffer) != message {
		return fmt.Errorf("received %q, want %q", buffer, message)
	}
	return nil
}

// exerciseDenied calls every net operation the boundary refuses; each one must
// fail without reaching the host network stack.
func exerciseDenied(listener *net.TCPListener, connection *net.TCPConn) error {
	ctx := context.Background()
	var dialer net.Dialer
	var config net.ListenConfig
	resolver := net.DefaultResolver
	loopback := netip.MustParseAddr("127.0.0.1")
	loopbackPort := netip.AddrPortFrom(loopback, 1)
	ipAddress := &net.IPAddr{IP: net.IPv4(127, 0, 0, 1)}
	udpAddress := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1}
	unixAddress := &net.UnixAddr{Name: "/socket", Net: "unix"}
	denied := []struct {
		name string
		call func() error
	}{
		{"TCPConn.File", func() error { _, err := connection.File(); return err }},
		{"TCPConn.SyscallConn", func() error { _, err := connection.SyscallConn(); return err }},
		{"TCPListener.File", func() error { _, err := listener.File(); return err }},
		{"TCPListener.SyscallConn", func() error { _, err := listener.SyscallConn(); return err }},
		{"Dialer.DialIP", func() error { _, err := dialer.DialIP(ctx, "ip4:icmp", loopback, loopback); return err }},
		{"Dialer.DialTCP", func() error { _, err := dialer.DialTCP(ctx, "tcp", netip.AddrPort{}, loopbackPort); return err }},
		{"Dialer.DialUDP", func() error { _, err := dialer.DialUDP(ctx, "udp", netip.AddrPort{}, loopbackPort); return err }},
		{"Dialer.DialUnix", func() error { _, err := dialer.DialUnix(ctx, "unix", nil, unixAddress); return err }},
		{"DialIP", func() error { _, err := net.DialIP("ip4:icmp", nil, ipAddress); return err }},
		{"DialUDP", func() error { _, err := net.DialUDP("udp", nil, udpAddress); return err }},
		{"DialUnix", func() error { _, err := net.DialUnix("unix", nil, unixAddress); return err }},
		{"FileConn", func() error { _, err := net.FileConn(os.Stdin); return err }},
		{"FileListener", func() error { _, err := net.FileListener(os.Stdin); return err }},
		{"FilePacketConn", func() error { _, err := net.FilePacketConn(os.Stdin); return err }},
		{"Interface.Addrs", func() error { _, err := (&net.Interface{Index: 1}).Addrs(); return err }},
		{"Interface.MulticastAddrs", func() error { _, err := (&net.Interface{Index: 1}).MulticastAddrs(); return err }},
		{"InterfaceAddrs", func() error { _, err := net.InterfaceAddrs(); return err }},
		{"InterfaceByIndex", func() error { _, err := net.InterfaceByIndex(1); return err }},
		{"InterfaceByName", func() error { _, err := net.InterfaceByName("lo0"); return err }},
		{"ListenConfig.ListenPacket", func() error { _, err := config.ListenPacket(ctx, "udp", "127.0.0.1:0"); return err }},
		{"ListenIP", func() error { _, err := net.ListenIP("ip4:icmp", ipAddress); return err }},
		{"ListenMulticastUDP", func() error { _, err := net.ListenMulticastUDP("udp", nil, udpAddress); return err }},
		{"ListenUDP", func() error { _, err := net.ListenUDP("udp", udpAddress); return err }},
		{"ListenUnix", func() error { _, err := net.ListenUnix("unix", unixAddress); return err }},
		{"ListenUnixgram", func() error { _, err := net.ListenUnixgram("unixgram", unixAddress); return err }},
		{"LookupTXT", func() error { _, err := net.LookupTXT("localhost"); return err }},
		{"ResolveIPAddr", func() error { _, err := net.ResolveIPAddr("ip", "localhost"); return err }},
		{"ResolveTCPAddr", func() error { _, err := net.ResolveTCPAddr("tcp", "localhost:1"); return err }},
		{"ResolveUDPAddr", func() error { _, err := net.ResolveUDPAddr("udp", "localhost:1"); return err }},
		{"ResolveUnixAddr", func() error { _, err := net.ResolveUnixAddr("unix", "/socket"); return err }},
		{"Resolver.LookupAddr", func() error { _, err := resolver.LookupAddr(ctx, "127.0.0.1"); return err }},
		{"Resolver.LookupCNAME", func() error { _, err := resolver.LookupCNAME(ctx, "localhost"); return err }},
		{"Resolver.LookupHost", func() error { _, err := resolver.LookupHost(ctx, "localhost"); return err }},
		{"Resolver.LookupIP", func() error { _, err := resolver.LookupIP(ctx, "ip", "localhost"); return err }},
		{"Resolver.LookupMX", func() error { _, err := resolver.LookupMX(ctx, "localhost"); return err }},
		{"Resolver.LookupNetIP", func() error { _, err := resolver.LookupNetIP(ctx, "ip", "localhost"); return err }},
		{"Resolver.LookupNS", func() error { _, err := resolver.LookupNS(ctx, "localhost"); return err }},
		{"Resolver.LookupPort", func() error { _, err := resolver.LookupPort(ctx, "tcp", "http"); return err }},
		{"Resolver.LookupSRV", func() error { _, _, err := resolver.LookupSRV(ctx, "http", "tcp", "localhost"); return err }},
		{"Resolver.LookupTXT", func() error { _, err := resolver.LookupTXT(ctx, "localhost"); return err }},
	}
	for _, operation := range denied {
		if err := operation.call(); err == nil {
			return fmt.Errorf("%s succeeded", operation.name)
		}
	}
	return nil
}
