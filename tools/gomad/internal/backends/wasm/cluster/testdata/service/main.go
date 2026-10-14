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
	if os.Args[1] == "server" {
		listener, err := net.Listen("tcp", "127.0.0.1:7233")
		check(err)
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
		time.Sleep(2 * time.Second)
		check(listener.Close())
	} else {
		connection, err := net.Dial("tcp", os.Args[2]+":7233")
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
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
