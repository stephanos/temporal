package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func main() {
	value := make([]byte, 64)
	if _, err := rand.Read(value); err != nil {
		panic(err)
	}
	fmt.Println(hex.EncodeToString(value))
}
