// io_entropy prints bytes drawn from crypto/rand. The deterministic I/O
// profile owns that entropy, so the output must not change with the
// schedule seed.
package main

import (
	"crypto/rand"
	"fmt"
	"os"
)

func main() {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("%x\n", buffer)
	fmt.Println(rand.Text())
}
