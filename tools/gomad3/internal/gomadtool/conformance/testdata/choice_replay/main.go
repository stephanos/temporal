// choice_replay makes a short sequence of select decisions whose transcript the
// choice tape must reproduce exactly.
package main

import (
	"fmt"
	"strings"
)

func main() {
	var decisions []string
	for range 6 {
		left := make(chan struct{}, 1)
		right := make(chan struct{}, 1)
		left <- struct{}{}
		right <- struct{}{}
		select {
		case <-left:
			decisions = append(decisions, "L")
		case <-right:
			decisions = append(decisions, "R")
		}
	}
	fmt.Println(strings.Join(decisions, ""))
}
