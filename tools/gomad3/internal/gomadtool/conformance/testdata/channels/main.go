// channels runs producers over shared channels and prints the delivery order.
package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"

	"gomad3.test/internal/perturb"
)

const producers = 4
const messagesPerProducer = 6

func main() {
	perturb.Apply(os.Args[1:])
	unbuffered := make(chan int)
	buffered := make(chan int, 3)
	var group sync.WaitGroup
	for id := range producers {
		group.Go(func() {
			for message := range messagesPerProducer {
				value := id*messagesPerProducer + message
				if message%2 == 0 {
					unbuffered <- value
				} else {
					buffered <- value
				}
				runtime.Gosched()
			}
		})
	}
	go func() {
		group.Wait()
		close(unbuffered)
		close(buffered)
	}()
	var order []string
	sum := 0
	for unbuffered != nil || buffered != nil {
		select {
		case value, ok := <-unbuffered:
			if !ok {
				unbuffered = nil
				continue
			}
			order = append(order, fmt.Sprintf("u%d", value))
			sum += value
		case value, ok := <-buffered:
			if !ok {
				buffered = nil
				continue
			}
			order = append(order, fmt.Sprintf("b%d", value))
			sum += value
		}
	}
	fmt.Println(strings.Join(order, " "))
	total := producers * messagesPerProducer
	if sum == total*(total-1)/2 && len(order) == total {
		fmt.Println("channels-oracle:ok")
	} else {
		fmt.Printf("channels-oracle:received %d messages summing to %d\n", len(order), sum)
	}
	perturb.Marker()
}
