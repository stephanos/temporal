package main

import (
	"os"
	"time"
)

func main() {
	reader, writer, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	time.AfterFunc(time.Hour, func() {
		os.Exit(99)
	})
	var buffer [1]byte
	if _, err := reader.Read(buffer[:]); err != nil {
		panic(err)
	}
	if err := writer.Close(); err != nil {
		panic(err)
	}
}
