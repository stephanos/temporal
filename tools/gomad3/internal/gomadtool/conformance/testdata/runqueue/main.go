// runqueue spawns a goroutine tree and prints the order in which nodes start.
package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"

	"gomad3.test/internal/perturb"
)

const depth = 3
const fanout = 3

func main() {
	perturb.Apply(os.Args[1:])
	var mutex sync.Mutex
	var started []string
	var group sync.WaitGroup
	var spawn func(name string, level int)
	spawn = func(name string, level int) {
		mutex.Lock()
		started = append(started, name)
		mutex.Unlock()
		if level == depth {
			return
		}
		for child := range fanout {
			childName := fmt.Sprintf("%s.%d", name, child)
			group.Go(func() {
				runtime.Gosched()
				spawn(childName, level+1)
			})
		}
	}
	spawn("root", 0)
	group.Wait()
	fmt.Println(strings.Join(started, " "))
	perturb.Marker()
}
