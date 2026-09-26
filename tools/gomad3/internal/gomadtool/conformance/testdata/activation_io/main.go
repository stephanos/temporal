// activation_io prints the hostname: the deterministic boundary answers with
// its modeled name, the host boundary with the real one.
package main

import (
	"fmt"
	"os"
)

func main() {
	name, err := os.Hostname()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(name)
}
