package main

import (
	"context"
	"fmt"
	"os"
	"time"
)

func main() {
	for range 700_000 {
		time.Now()
	}

	client, cancelClient := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelClient()
	firstHop, cancelFirstHop := context.WithTimeout(context.Background(), time.Until(deadline(client)))
	defer cancelFirstHop()
	secondHop, cancelSecondHop := context.WithTimeout(context.Background(), time.Until(deadline(firstHop)))
	defer cancelSecondHop()
	child, cancelChild := context.WithTimeout(context.Background(), time.Until(deadline(secondHop))-time.Second)
	defer cancelChild()

	select {
	case <-child.Done():
		if client.Err() != nil {
			fmt.Fprintln(os.Stderr, "client deadline fired with the child deadline")
			os.Exit(1)
		}
		fmt.Println(time.Until(deadline(client)).Nanoseconds())
	case <-client.Done():
		fmt.Fprintln(os.Stderr, "forward clock fired the client deadline before its buffered child")
		os.Exit(1)
	}
}

func deadline(ctx context.Context) time.Time {
	value, ok := ctx.Deadline()
	if !ok {
		panic("context has no deadline")
	}
	return value
}
