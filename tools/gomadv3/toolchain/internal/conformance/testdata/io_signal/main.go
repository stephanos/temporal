package main

import (
	"fmt"
	"os"
	"os/signal"
)

func main() {
	signals := make(chan os.Signal, 1)
	signal.Stop(signals)
	fmt.Println("ok")
}
