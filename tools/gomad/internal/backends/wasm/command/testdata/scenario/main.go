package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"time"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	if os.Getenv("GOMAD_REQUIRE_CAPTURED_INPUT") == "1" {
		input, err := os.ReadFile("/inputs/control")
		check(err)
		if string(input) != "retained typed input" {
			panic("captured input changed")
		}
	}
	switch os.Args[1] {
	case "simulation-live":
		file, err := os.Create("/workspace/value")
		check(err)
		_, err = file.WriteString("before!!")
		check(err)
		check(file.Sync())
		dir, err := os.Open("/workspace")
		check(err)
		check(dir.Sync())
		check(dir.Close())
		_, err = file.WriteAt([]byte("survives"), 0)
		check(err)
		check(file.Close())
		ready := make(chan net.Listener, 2)
		for range 2 {
			go func() {
				listener, err := net.Listen("tcp", "127.0.0.1:0")
				check(err)
				ready <- listener
			}()
		}
		deadline := time.Now().Add(time.Second)
		finished := make(chan error, 2)
		for range 2 {
			listener := <-ready
			check(listener.(*net.TCPListener).SetDeadline(deadline))
			go func() {
				defer listener.Close()
				_, err := listener.Accept()
				finished <- err
			}()
		}
		for range 2 {
			err := <-finished
			if timeout, ok := err.(net.Error); !ok || !timeout.Timeout() {
				panic(fmt.Sprintf("simulation accept: %v", err))
			}
		}
		fmt.Println("pending simulation write")
		time.Sleep(time.Hour)
		panic("crash boundary did not revoke guest")
	case "simulation-recover":
		data, err := os.ReadFile("/workspace/value")
		check(err)
		fmt.Printf("observed simulation recovery %q\n", data)
	case "fail-peer":
		fmt.Fprintln(os.Stderr, "failing peer started")
		time.Sleep(time.Second)
		panic("intentional asymmetric peer failure")
	case "wait-peer":
		fmt.Fprintln(os.Stderr, "waiting peer started")
		for {
			time.Sleep(time.Second)
		}
	case "durable":
		file, err := os.Create("/workspace/value")
		check(err)
		_, err = file.WriteString("durable-a")
		check(err)
		check(file.Sync())
		check(file.Close())
		dir, err := os.Open("/workspace")
		check(err)
		check(dir.Sync())
		check(dir.Close())
		fmt.Println("committed durable-a")
		time.Sleep(time.Hour)
		panic("crash boundary did not revoke guest")
	case "recover", "canary", "mixed":
		data, err := os.ReadFile("/workspace/value")
		check(err)
		if string(data) != "durable-a" {
			panic("durable value changed")
		}
		fmt.Println("recovered durable-a")
		if os.Args[1] == "canary" {
			panic("intentional replayable scenario failure")
		}
		if os.Args[1] == "mixed" {
			seed, err := strconv.ParseUint(os.Getenv("GOMADSEED"), 10, 64)
			check(err)
			fmt.Println("campaign seed", seed)
			if seed&1 == 0 {
				panic("intentional replayable scenario failure")
			}
		}
	case "park":
		time.Sleep(time.Second)
		fmt.Println("independent peer")
	case "server", "server-stall":
		listener, err := net.Listen("tcp", "127.0.0.1:7233")
		check(err)
		defer listener.Close()
		if os.Args[1] == "server-stall" {
			check(listener.(*net.TCPListener).SetDeadline(time.Now().Add(15 * time.Second)))
			_, err = listener.Accept()
			if timeout, ok := err.(net.Error); !ok || !timeout.Timeout() {
				panic(fmt.Sprintf("partition accept: %v", err))
			}
			fmt.Println("partition blocked accept")
			return
		}
		connection, err := listener.Accept()
		check(err)
		defer connection.Close()
		var payload [4]byte
		_, err = io.ReadFull(connection, payload[:])
		check(err)
		if string(payload[:]) != "ping" {
			panic("network payload changed")
		}
		_, err = connection.Write(payload[:])
		check(err)
		fmt.Println("healed server ping")
	case "client", "client-stall":
		if os.Args[1] == "client" {
			time.Sleep(25 * time.Second)
		}
		connection, err := net.Dial("tcp", "10.0.0.2:7233")
		if os.Args[1] == "client-stall" {
			if timeout, ok := err.(net.Error); !ok || !timeout.Timeout() {
				panic(fmt.Sprintf("partition dial: %v", err))
			}
			fmt.Println("partition blocked dial")
			return
		}
		check(err)
		defer connection.Close()
		_, err = connection.Write([]byte("ping"))
		check(err)
		var payload [4]byte
		_, err = io.ReadFull(connection, payload[:])
		check(err)
		if string(payload[:]) != "ping" {
			panic("network echo changed")
		}
		fmt.Println("healed client ping")
	case "echo-loop":
		listener, err := net.Listen("tcp", "127.0.0.1:7233")
		check(err)
		defer listener.Close()
		fmt.Fprintln(os.Stderr, "echo server listening")
		connection, err := listener.Accept()
		check(err)
		defer connection.Close()
		fmt.Println("echo server accepted")
		var payload [4]byte
		for {
			_, err = io.ReadFull(connection, payload[:])
			if err == io.EOF {
				fmt.Println("echo server closed")
				return
			}
			check(err)
			_, err = connection.Write(payload[:])
			check(err)
		}
	case "ping-loop":
		fmt.Fprintln(os.Stderr, "ping client dialing")
		time.Sleep(time.Second)
		connection, err := net.Dial("tcp", "10.0.0.2:7233")
		check(err)
		defer connection.Close()
		fmt.Println("ping client connected")
		for round := 1; round <= 4096; round++ {
			_, err = connection.Write([]byte("ping"))
			check(err)
			var payload [4]byte
			_, err = io.ReadFull(connection, payload[:])
			check(err)
			if string(payload[:]) != "ping" {
				panic("network echo changed")
			}
			fmt.Println("ping round", round)
		}
		fmt.Println("ping client finished")
	default:
		panic("unknown scenario mode")
	}
}
