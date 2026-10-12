// Command network_probes drives a TCP echo exchange through the simulated
// network and reports what each round observed at its virtual time.
package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"
	"time"
)

var start = time.Now()

func elapsed() time.Duration { return time.Since(start) }

func serve(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		go func() {
			defer conn.Close()
			reader := bufio.NewReader(conn)
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				if _, err := conn.Write([]byte("echo " + line)); err != nil {
					return
				}
			}
		}()
	}
}

func classify(err error) string {
	switch {
	case errors.Is(err, syscall.ECONNREFUSED) || strings.Contains(err.Error(), "connection refused") || strings.Contains(err.Error(), "Connection refused"):
		return "refused"
	case errors.Is(err, syscall.ECONNRESET):
		return "reset"
	case errors.Is(err, net.ErrClosed):
		return "closed"
	case err.Error() == "EOF":
		return "eof"
	}
	return err.Error()
}

func main() {
	listener, err := net.Listen("tcp", "127.0.0.1:9000")
	if err != nil {
		panic(err)
	}
	go serve(listener)
	echoed, failures := 0, 0
	for round := range 12 {
		conn, err := net.Dial("tcp", "127.0.0.1:9000")
		if err != nil {
			failures++
			fmt.Printf("round %d dial %s at %v\n", round, classify(err), elapsed())
			time.Sleep(time.Second)
			continue
		}
		reader := bufio.NewReader(conn)
		for i := range 4 {
			if _, err := fmt.Fprintf(conn, "r%d-%d\n", round, i); err != nil {
				failures++
				fmt.Printf("round %d write %s at %v\n", round, classify(err), elapsed())
				break
			}
			line, err := reader.ReadString('\n')
			if err != nil {
				failures++
				fmt.Printf("round %d read %s at %v\n", round, classify(err), elapsed())
				break
			}
			echoed++
			_ = line
			time.Sleep(250 * time.Millisecond)
		}
		conn.Close()
		time.Sleep(250 * time.Millisecond)
	}
	fmt.Printf("echoed %d failures %d elapsed %v\n", echoed, failures, elapsed())
	if len(os.Args) > 1 && os.Args[1] == "expect-faults" && failures == 0 {
		fmt.Println("FAIL no fault observed")
		os.Exit(3)
	}
}
