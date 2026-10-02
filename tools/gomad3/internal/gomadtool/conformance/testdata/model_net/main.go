package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"strconv"
	"time"
)

type operation struct {
	name string
	call func() (string, error)
}
type result struct {
	Operation string `json:"operation"`
	Result    string `json:"result"`
	Error     string `json:"error"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) != 3 {
		return fmt.Errorf("usage: model_net seed prefix-length")
	}
	seed, err := strconv.ParseInt(os.Args[1], 10, 64)
	if err != nil {
		return err
	}
	length, err := strconv.Atoi(os.Args[2])
	if err != nil {
		return err
	}
	if length < 1 || length > 64 {
		return fmt.Errorf("prefix length must be 1..64")
	}
	rng := rand.New(rand.NewSource(seed))
	listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		return err
	}
	if err := listener.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		return err
	}
	dialer := net.Dialer{Timeout: 2 * time.Second}
	dialed, err := dialer.Dial("tcp4", listener.Addr().String())
	if err != nil {
		return err
	}
	client := dialed.(*net.TCPConn)
	server, err := listener.AcceptTCP()
	if err != nil {
		return err
	}
	deadline := time.Now().Add(2 * time.Second)
	if err := client.SetDeadline(deadline); err != nil {
		return err
	}
	if err := server.SetDeadline(deadline); err != nil {
		return err
	}
	var clientClosed, serverClosed, listenerClosed bool
	steps := []operation{{"addresses", func() (string, error) {
		return fmt.Sprintf("%t:%t:%s:%s", client.LocalAddr().String() == server.RemoteAddr().String(), client.RemoteAddr().String() == server.LocalAddr().String(), client.LocalAddr().Network(), client.RemoteAddr().Network()), nil
	}}}
	for len(steps) < 52 {
		side := rng.Intn(2)
		from, to := client, server
		name := "client"
		if side == 1 {
			from, to = server, client
			name = "server"
		}
		var generated []operation
		switch rng.Intn(4) {
		case 0, 1:
			data := make([]byte, 1+rng.Intn(32))
			for index := range data {
				data[index] = byte(rng.Intn(256))
			}
			generated = append(generated, operation{fmt.Sprintf("write %s %x", name, data), func() (string, error) { n, err := from.Write(data); return strconv.Itoa(n), err }}, operation{fmt.Sprintf("read-full peer-of-%s %d", name, len(data)), func() (string, error) {
				buffer := make([]byte, len(data))
				n, err := io.ReadFull(to, buffer)
				return fmt.Sprintf("%d:%x", n, buffer[:n]), err
			}})
		case 2:
			generated = append(generated, operation{"read-deadline-expired " + name, func() (string, error) { return "", from.SetReadDeadline(time.Unix(1, 0)) }}, operation{"read-timeout " + name, func() (string, error) {
				buffer := make([]byte, 1)
				n, err := from.Read(buffer)
				return fmt.Sprintf("%d:%x", n, buffer[:n]), err
			}}, operation{"read-deadline-reset " + name, func() (string, error) { return "", from.SetReadDeadline(time.Now().Add(2 * time.Second)) }})
		case 3:
			enabled := rng.Intn(2) == 1
			generated = append(generated, operation{fmt.Sprintf("no-delay %s %t", name, enabled), func() (string, error) { return "", from.SetNoDelay(enabled) }})
		}
		if len(steps)+len(generated) > 52 {
			steps = append(steps, operation{"no-delay client true", func() (string, error) { return "", client.SetNoDelay(true) }})
		} else {
			steps = append(steps, generated...)
		}

	}
	// Keep the terminal sequence at a fixed position, independent of prefix length.
	// A generated write must retain its matching read before half-closing.
	steps = append(steps,
		operation{"deadline-reset client", func() (string, error) { return "", client.SetDeadline(time.Now().Add(2 * time.Second)) }},
		operation{"deadline-reset server", func() (string, error) { return "", server.SetDeadline(time.Now().Add(2 * time.Second)) }},
		operation{"close-write client", func() (string, error) { return "", client.CloseWrite() }},
		operation{"read-eof server", func() (string, error) {
			buffer := make([]byte, 1)
			n, err := server.Read(buffer)
			return fmt.Sprintf("%d:%x", n, buffer[:n]), err
		}},
		operation{"write server 66696e", func() (string, error) { n, err := server.Write([]byte("fin")); return strconv.Itoa(n), err }},
		operation{"read-full client 3", func() (string, error) {
			buffer := make([]byte, 3)
			n, err := io.ReadFull(client, buffer)
			return fmt.Sprintf("%d:%x", n, buffer[:n]), err
		}},
		operation{"close client", func() (string, error) { clientClosed = true; return "", client.Close() }},
		operation{"read closed client", func() (string, error) {
			buffer := make([]byte, 1)
			n, err := client.Read(buffer)
			return fmt.Sprintf("%d:%x", n, buffer[:n]), err
		}},
		operation{"close server", func() (string, error) { serverClosed = true; return "", server.Close() }},
		operation{"close listener", func() (string, error) { listenerClosed = true; return "", listener.Close() }},
		operation{"accept closed listener", func() (string, error) { _, err := listener.AcceptTCP(); return "", err }},
		operation{"close listener again", func() (string, error) { return "", listener.Close() }},
	)
	encoder := json.NewEncoder(os.Stdout)
	for _, step := range steps[:length] {
		value, err := step.call()
		if err := encoder.Encode(result{step.name, value, errorClass(err)}); err != nil {
			return err
		}
	}
	if !clientClosed {
		if err := client.Close(); err != nil {
			return err
		}
	}
	if !serverClosed {
		if err := server.Close(); err != nil {
			return err
		}
	}
	if !listenerClosed {
		if err := listener.Close(); err != nil {
			return err
		}
	}
	return nil
}

func errorClass(err error) string {
	if err == nil {
		return "ok"
	}
	for _, entry := range []struct {
		err   error
		class string
	}{{io.EOF, "eof"}, {io.ErrUnexpectedEOF, "unexpected-eof"}, {net.ErrClosed, "closed"}, {os.ErrDeadlineExceeded, "timeout"}} {
		if errors.Is(err, entry.err) {
			return entry.class
		}
	}
	return fmt.Sprintf("unclassified:%T:%v", err, err)
}
