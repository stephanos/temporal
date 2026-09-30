// Package main reaches the address library the way a membership layer does, so
// the adapter's prepared source set is reviewed through a real module graph.
package main

import (
	"fmt"

	sockaddr "github.com/hashicorp/go-sockaddr"
)

func main() {
	address, err := sockaddr.NewIPv4Addr("127.0.0.1")
	if err != nil {
		panic(err)
	}
	fmt.Println(address.String())
}
