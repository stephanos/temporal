// clock_tick prints eight successive time.Now readings in Unix nanoseconds, one
// per line, so the conformance tier can check the virtual-clock tick policy.
package main

import (
	"fmt"
	"time"
)

func main() {
	var readings [8]int64
	for index := range readings {
		readings[index] = time.Now().UnixNano()
	}
	for _, reading := range readings {
		fmt.Println(reading)
	}
}
