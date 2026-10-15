package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

var incarnationHeap int

func main() {
	incarnationHeap++
	fmt.Println("fresh heap", incarnationHeap)
	if os.Args[1] == "boundary-stderr" {
		fmt.Fprint(os.Stderr, "output boundary")
		fmt.Println("stderr marker did not stop stdout")
		return
	}
	if os.Args[1] == "boundary-split" {
		fmt.Print("output ")
		fmt.Print("boundary")
		panic("split output boundary continuation executed")
	}
	if os.Args[1] == "observation" {
		history := []byte("committed history")
		if len(os.Args) > 2 && os.Args[2] == "canary" {
			history = []byte("different history")
		}
		value := struct {
			ClusterID string   `json:"cluster_id"`
			History   []byte   `json:"history"`
			EventIDs  []int64  `json:"event_ids"`
			Types     []string `json:"types"`
		}{"independent-oracle-reference", history, []int64{1, 2}, []string{"started", "completed"}}
		data, err := json.Marshal(value)
		check(err)
		fmt.Printf("GOMAD_HISTORY_OBSERVATION %s\n", data)
		return
	}
	if os.Args[1] == "durable" {
		if data, err := os.ReadFile("/workspace/value"); err == nil {
			fmt.Println("recovered", string(data))
			return
		}
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
		panic("crash marker continuation executed")
	}
	if os.Args[1] == "recover" {
		data, err := os.ReadFile("/workspace/value")
		check(err)
		fmt.Println("recovered", string(data))
		return
	}
	if os.Args[1] == "park" {
		time.Sleep(time.Second)
		fmt.Println("independent peer")
		return
	}
	if os.Args[1] == "server" || os.Args[1] == "server-stall" || os.Args[1] == "server-open" || os.Args[1] == "server-storage-crash" {
		listener, err := net.Listen("tcp", "127.0.0.1:7233")
		check(err)
		if os.Args[1] == "server-stall" {
			check(listener.(*net.TCPListener).SetDeadline(time.Now().Add(15 * time.Second)))
			_, err := listener.Accept()
			if timeout, ok := err.(net.Error); !ok || !timeout.Timeout() {
				panic(fmt.Sprintf("accept stall: %v", err))
			}
			check(listener.Close())
			fmt.Println("stall expired")
			return
		}
		connection, err := listener.Accept()
		check(err)
		defer connection.Close()
		payload := make([]byte, 12)
		_, err = io.ReadFull(connection, payload)
		check(err)
		if !bytes.Equal(payload, []byte("ordered-byte")) {
			panic("stream changed bytes")
		}
		_, err = connection.Write(payload)
		check(err)
		fmt.Println("server ordered bytes", string(payload))
		if os.Args[1] == "server-open" || os.Args[1] == "server-storage-crash" {
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
			if os.Args[1] == "server-storage-crash" {
				trigger, err := os.Create("/workspace/crash-trigger")
				check(err)
				check(trigger.Close())
				panic("storage crash continuation executed")
			}
			fmt.Println("open stream ready")
			time.Sleep(time.Hour)
			panic("open connection crash continuation executed")
		}
		time.Sleep(2 * time.Second)
		check(listener.Close())
	} else {
		connection, err := net.Dial("tcp", os.Args[2]+":7233")
		if os.Args[1] == "client-stall" {
			if timeout, ok := err.(net.Error); !ok || !timeout.Timeout() {
				panic(fmt.Sprintf("dial stall: %v", err))
			}
			fmt.Println("stall expired")
			return
		}
		check(err)
		defer connection.Close()
		for _, payload := range []string{"ordered", "-byte"} {
			_, err = connection.Write([]byte(payload))
			check(err)
		}
		payload := make([]byte, 12)
		_, err = io.ReadFull(connection, payload)
		check(err)
		fmt.Println("client ordered bytes", string(payload))
		if os.Args[1] == "client-open" {
			fmt.Println("open stream ready")
			time.Sleep(time.Hour)
			panic("open connection crash continuation executed")
		}
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
