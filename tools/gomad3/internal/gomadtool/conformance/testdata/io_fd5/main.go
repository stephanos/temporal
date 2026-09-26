// io_fd5 writes to inherited descriptor 5, which the bootstrap reserves for
// its own transport only when a deterministic I/O profile is active. Without a
// profile the descriptor must reach the program untouched.
package main

import (
	"fmt"
	"os"
)

func main() {
	file := os.NewFile(5, "fd5")
	if file == nil {
		fmt.Fprintln(os.Stderr, "descriptor 5 is not open")
		os.Exit(1)
	}
	if _, err := file.WriteString("preserved"); err != nil {
		fmt.Fprintln(os.Stderr, "write descriptor 5:", err)
		os.Exit(1)
	}
	if err := file.Close(); err != nil {
		fmt.Fprintln(os.Stderr, "close descriptor 5:", err)
		os.Exit(1)
	}
}
