package main

import (
	"context"
	"fmt"
	"os"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Microsecond)
	defer cancel()
	deadline, ok := ctx.Deadline()
	if !ok {
		panic("context has no deadline")
	}
	for time.Until(deadline) > 0 {
		time.Now()
	}
	before := time.Now()
	<-ctx.Done()
	if ctx.Err() != context.DeadlineExceeded {
		fmt.Fprintln(os.Stderr, "due context did not expire at the next timer check")
		os.Exit(1)
	}
	fmt.Println(time.Since(before).Nanoseconds())
}
