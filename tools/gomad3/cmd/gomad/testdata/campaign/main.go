package main

import (
	"crypto/sha256"
	"fmt"
	"math/rand/v2"
	"os"
	"strconv"
)

func main() {
	seed := strconv.FormatUint(rand.Uint64(), 10)
	payload := []byte("seed=" + seed)
	if err := os.WriteFile("/campaign-fixture", payload, 0o600); err != nil {
		panic(err)
	}
	observed, err := os.ReadFile("/campaign-fixture")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", observed)
	if len(os.Args) > 1 && os.Args[1] == "work" {
		digest := sha256.Sum256(payload)
		for range 15_000_000 {
			digest = sha256.Sum256(digest[:])
		}
		fmt.Printf("digest=%x\n", digest)
	}
	if seed == "16720882452029570533" {
		fmt.Fprintln(os.Stderr, "known failing seed")
		os.Exit(1)
	}
}
