// select drains several ready channels through one select statement and prints
// the branch sequence the runtime chose.
package main

import (
	"fmt"
	"os"
	"strings"

	"gomad3.test/internal/perturb"
)

const channels = 6
const tokensPerChannel = 8

func main() {
	perturb.Apply(os.Args[1:])
	sources := make([]chan int, channels)
	for index := range sources {
		sources[index] = make(chan int, tokensPerChannel)
		for range tokensPerChannel {
			sources[index] <- index
		}
	}
	counts := make([]int, channels)
	var sequence strings.Builder
	for range channels * tokensPerChannel {
		var chosen int
		select {
		case chosen = <-sources[0]:
		case chosen = <-sources[1]:
		case chosen = <-sources[2]:
		case chosen = <-sources[3]:
		case chosen = <-sources[4]:
		case chosen = <-sources[5]:
		}
		counts[chosen]++
		sequence.WriteByte(byte('a' + chosen))
	}
	fmt.Println(sequence.String())
	oracle := "select-oracle:ok"
	for index, count := range counts {
		if count != tokensPerChannel {
			oracle = fmt.Sprintf("select-oracle:channel %d delivered %d tokens", index, count)
		}
	}
	fmt.Println(oracle)
	perturb.Marker()
}
