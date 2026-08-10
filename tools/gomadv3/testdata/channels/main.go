package main

import "fmt"

func main() {
	const workers = 12
	start := make(chan struct{})
	ready := make(chan struct{}, workers)
	unbuffered := make(chan int)
	for worker := range workers {
		go func() {
			<-start
			ready <- struct{}{}
			unbuffered <- worker
		}()
	}
	close(start)
	for range workers {
		<-ready
	}
	var sendOrder [workers]int
	for index := range sendOrder {
		sendOrder[index] = <-unbuffered
	}

	wake := make(chan struct{})
	wakeReady := make(chan struct{}, workers)
	woken := make(chan int, workers)
	for worker := range workers {
		go func() {
			wakeReady <- struct{}{}
			<-wake
			woken <- worker
		}()
	}
	for range workers {
		<-wakeReady
	}
	close(wake)
	var closeOrder [workers]int
	for index := range closeOrder {
		closeOrder[index] = <-woken
	}

	fmt.Print("send:")
	for _, worker := range sendOrder {
		fmt.Printf("%d,", worker)
	}
	fmt.Println()
	fmt.Print("close:")
	for _, worker := range closeOrder {
		fmt.Printf("%d,", worker)
	}
	fmt.Println()
}
