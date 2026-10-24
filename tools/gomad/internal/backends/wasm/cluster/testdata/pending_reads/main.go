package main

import (
	"fmt"
	"io"
	"net"
	"time"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	const count = 32
	listener, err := net.Listen("tcp", "127.0.0.1:9200")
	check(err)
	accepted := make(chan net.Conn)
	go func() {
		for range count {
			connection, err := listener.Accept()
			check(err)
			accepted <- connection
		}
	}()
	clients := make([]net.Conn, count)
	servers := make([]net.Conn, count)
	results := make(chan int)
	started := make(chan struct{})
	for i := range count {
		clients[i], err = net.Dial("tcp", "127.0.0.1:9200")
		check(err)
		servers[i] = <-accepted
		go func() {
			started <- struct{}{}
			var value [1]byte
			_, err := io.ReadFull(clients[i], value[:])
			check(err)
			if value[0] != byte(i) {
				panic("wrong completion bytes")
			}
			results <- i
		}()
		<-started
	}
	start := time.Now()
	for range 48 {
		time.Sleep(time.Millisecond)
		select {
		case <-results:
			panic("read completed before write")
		default:
		}
	}
	if elapsed := time.Since(start); elapsed != 48*time.Millisecond {
		panic(fmt.Sprintf("wrong virtual elapsed time %s", elapsed))
	}
	fmt.Println("pending reads survived 48 scans at 48ms")
	for i := range count {
		n, err := servers[i].Write([]byte{byte(i)})
		check(err)
		if n != 1 || <-results != i {
			panic("wrong completion order")
		}
		check(clients[i].Close())
		check(servers[i].Close())
	}
	check(listener.Close())
	fmt.Println("completed 32 reads in order")
}
