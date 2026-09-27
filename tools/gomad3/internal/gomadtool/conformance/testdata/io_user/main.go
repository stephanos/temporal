// io_user asks for the current user; the deterministic boundary must refuse
// rather than consult the host account database.
package main

import (
	"fmt"
	"os"
	"os/user"
)

func main() {
	if current, err := user.Current(); err == nil {
		fmt.Fprintln(os.Stderr, "user lookup reached the host:", current.Username)
		os.Exit(1)
	}
	fmt.Println("ok")
}
