// choice_exploration has exactly two outcomes decided by one select over two
// ready channels, which pins the choice-frontier benchmark.
package main

import "fmt"

func main() {
	first := make(chan struct{}, 1)
	second := make(chan struct{}, 1)
	first <- struct{}{}
	second <- struct{}{}
	select {
	case <-first:
		fmt.Println("outcome first")
	case <-second:
		fmt.Println("outcome second")
	}
}
