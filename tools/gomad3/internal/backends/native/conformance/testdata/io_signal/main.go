// io_signal stops signal delivery to a channel that never registered, the
// one os/signal operation the deterministic boundary models.
package main

import (
	"fmt"
	"os"
	"os/signal"
)

func main() {
	signals := make(chan os.Signal, 1)
	signal.Stop(signals)
	fmt.Println("ok")
}
