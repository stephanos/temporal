// random prints eight samples from the runtime-seeded generator.
package main

import (
	"fmt"
	"math/rand/v2"
)

func main() {
	for range 8 {
		fmt.Printf("%016x %08x\n", rand.Uint64(), rand.Uint32())
	}
}
