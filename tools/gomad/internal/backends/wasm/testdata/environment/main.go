package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

func main() {
	values := map[string]int{"a": 1, "b": 2, "c": 3, "d": 4, "e": 5, "f": 6, "g": 7, "h": 8, "i": 9}
	for key, value := range values {
		fmt.Printf("map %s %d\n", key, value)
	}
	first, second := make(chan int, 1), make(chan int, 1)
	for i := 0; i < 16; i++ {
		first <- 1
		second <- 2
		select {
		case value := <-first:
			fmt.Println("select", value)
			<-second
		case value := <-second:
			fmt.Println("select", value)
			<-first
		}
	}
	entropy := make([]byte, 48)
	if _, err := io.ReadFull(rand.Reader, entropy); err != nil {
		panic(err)
	}
	fmt.Printf("random %x\n", entropy)
	begin := time.Now()
	time.Sleep(3 * time.Millisecond)
	fmt.Println("timer", time.Since(begin).Nanoseconds())
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	done := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			done <- err
			return
		}
		_, err = io.CopyN(conn, conn, 4)
		if closeErr := conn.Close(); err == nil {
			err = closeErr
		}
		done <- err
	}()
	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		panic(err)
	}
	if _, err := conn.Write([]byte("echo")); err != nil {
		panic(err)
	}
	data := make([]byte, 4)
	if _, err := io.ReadFull(conn, data); err != nil {
		panic(err)
	}
	fmt.Println("fakeTCP", string(data))
	if err := conn.Close(); err != nil {
		panic(err)
	}
	if err := <-done; err != nil {
		panic(err)
	}
	if err := listener.Close(); err != nil {
		panic(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	fmt.Println("cwd", cwd, "env", os.Getenv("ONLY"), "arg", os.Args[1])
	if err := os.WriteFile("relative", []byte("scratch"), 0600); err != nil {
		panic(err)
	}
	data, err = os.ReadFile("relative")
	if err != nil {
		panic(err)
	}
	fmt.Println("file", string(data))
	if err := os.Remove("relative"); err != nil {
		panic(err)
	}
}
