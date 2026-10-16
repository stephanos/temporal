package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	input, err := os.ReadFile("/inputs/control")
	check(err)
	if string(input) != "retained typed input" {
		panic("captured input changed")
	}
	switch os.Args[1] {
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
			var value [1]byte
			_, err := rand.Read(value[:])
			check(err)
			fmt.Println("modeled entropy", value[0])
			if value[0]&1 == 0 {
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
	default:
		panic("unknown scenario mode")
	}
}
