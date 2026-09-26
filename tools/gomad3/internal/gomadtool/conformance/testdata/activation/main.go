// activation reports GOMAXPROCS during package initialization and in main so
// the driver can tell whether the seeded runtime activated.
package main

import (
	"fmt"
	"runtime"
)

func init() {
	fmt.Printf("init GOMAXPROCS=%d\n", runtime.GOMAXPROCS(0))
}

func main() {
	fmt.Printf("main GOMAXPROCS=%d\n", runtime.GOMAXPROCS(0))
}
