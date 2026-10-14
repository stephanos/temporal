// io_entropy prints bytes drawn from crypto/rand and a key generated from the
// FIPS DRBG. The deterministic I/O profile owns that entropy, so the output
// must not change with the schedule seed.
package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
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
	// Key generation ignores its reader argument and reads the DRBG.
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("%x\n", key.D.Bytes())
}
