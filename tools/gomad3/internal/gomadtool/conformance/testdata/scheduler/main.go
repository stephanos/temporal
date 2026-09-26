// scheduler interleaves cooperative goroutines and prints the observed order.
package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"

	"gomad3.test/internal/perturb"
)

func main() {
	perturb.Apply(os.Args[1:])
	const workers = 12
	const rounds = 4
	var mutex sync.Mutex
	var order []string
	var group sync.WaitGroup
	for id := range workers {
		group.Go(func() {
			for round := range rounds {
				mutex.Lock()
				order = append(order, fmt.Sprintf("g%dr%d", id, round))
				mutex.Unlock()
				runtime.Gosched()
			}
		})
	}
	group.Wait()
	fmt.Println(strings.Join(order, " "))
	perturb.Marker()
}
