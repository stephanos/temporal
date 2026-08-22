package main

import (
	"fmt"
	"os"

	"gomadv3.test/internal/layout"
)

func requirePermutation(values []int, size int) {
	seen := make([]bool, size)
	for _, value := range values {
		if value < 0 || value >= size || seen[value] {
			panic("channel contention lost or duplicated a worker")
		}
		seen[value] = true
	}
}

func main() {
	padding := layout.New(os.Args[1:])
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
	requirePermutation(sendOrder[:], workers)
	requirePermutation(closeOrder[:], workers)

	buffered := make(chan int, workers)
	for value := range workers {
		buffered <- value
	}
	close(buffered)
	for value := range workers {
		observed, open := <-buffered
		if !open || observed != value {
			panic("buffered channel violated FIFO order")
		}
	}
	if value, open := <-buffered; value != 0 || open {
		panic("drained closed channel returned an invalid result")
	}

	receiverChannel := make(chan int)
	receiverReady := make(chan struct{}, workers)
	receiverResults := make(chan [2]int, workers)
	for worker := range workers {
		go func() {
			receiverReady <- struct{}{}
			receiverResults <- [2]int{worker, <-receiverChannel}
		}()
	}
	for range workers {
		<-receiverReady
	}
	for value := range workers {
		receiverChannel <- value
	}
	receiverOrder := make([]int, workers)
	receivedValues := make([]int, workers)
	for index := range workers {
		result := <-receiverResults
		receiverOrder[index] = result[0]
		receivedValues[index] = result[1]
	}
	requirePermutation(receiverOrder, workers)
	requirePermutation(receivedValues, workers)

	full := make(chan int, 1)
	full <- -1
	senderReady := make(chan struct{}, workers)
	senderDone := make(chan int, workers)
	for worker := range workers {
		go func() {
			senderReady <- struct{}{}
			full <- worker
			senderDone <- worker
		}()
	}
	for range workers {
		<-senderReady
	}
	if value := <-full; value != -1 {
		panic("full buffered channel lost its initial value")
	}
	bufferedSendOrder := make([]int, workers)
	for index := range bufferedSendOrder {
		bufferedSendOrder[index] = <-full
	}
	bufferedSendDone := make([]int, workers)
	for index := range bufferedSendDone {
		bufferedSendDone[index] = <-senderDone
	}
	requirePermutation(bufferedSendOrder, workers)
	requirePermutation(bufferedSendDone, workers)

	empty := make(chan int, 1)
	emptyReady := make(chan struct{}, workers)
	emptyResults := make(chan [2]int, workers)
	for worker := range workers {
		go func() {
			emptyReady <- struct{}{}
			emptyResults <- [2]int{worker, <-empty}
		}()
	}
	for range workers {
		<-emptyReady
	}
	for value := range workers {
		empty <- value
	}
	bufferedReceiveOrder := make([]int, workers)
	bufferedReceiveValues := make([]int, workers)
	for index := range workers {
		result := <-emptyResults
		bufferedReceiveOrder[index] = result[0]
		bufferedReceiveValues[index] = result[1]
	}
	requirePermutation(bufferedReceiveOrder, workers)
	requirePermutation(bufferedReceiveValues, workers)

	closing := make(chan int, 4)
	for value := range 4 {
		closing <- value
	}
	type closeResult struct {
		value int
		open  bool
	}
	closeReady := make(chan struct{}, 8)
	closeResults := make(chan closeResult, 8)
	for range 8 {
		go func() {
			closeReady <- struct{}{}
			value, open := <-closing
			closeResults <- closeResult{value: value, open: open}
		}()
	}
	for range 8 {
		<-closeReady
	}
	close(closing)
	bufferedValues := make([]int, 0, 4)
	closedReceivers := 0
	for range 8 {
		result := <-closeResults
		if result.open {
			bufferedValues = append(bufferedValues, result.value)
		} else if result.value == 0 {
			closedReceivers++
		} else {
			panic("closed channel returned a nonzero value")
		}
	}
	if len(bufferedValues) != 4 || closedReceivers != 4 {
		panic("close did not preserve buffered values and wake receivers")
	}
	requirePermutation(bufferedValues, 4)

	waveChannel := make(chan int)
	waveDigest := uint64(14695981039346656037)
	for range 4 {
		waveReady := make(chan struct{}, workers)
		for worker := range workers {
			go func() {
				waveReady <- struct{}{}
				waveChannel <- worker
			}()
		}
		for range workers {
			<-waveReady
		}
		waveValues := make([]int, workers)
		for index := range waveValues {
			waveValues[index] = <-waveChannel
			waveDigest ^= uint64(waveValues[index])
			waveDigest *= 1099511628211
		}
		requirePermutation(waveValues, workers)
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
	fmt.Print("receive:")
	for _, worker := range receiverOrder {
		fmt.Printf("%d,", worker)
	}
	fmt.Println()
	fmt.Print("buffered-send:")
	for _, worker := range bufferedSendOrder {
		fmt.Printf("%d,", worker)
	}
	fmt.Println()
	fmt.Print("buffered-receive:")
	for _, worker := range bufferedReceiveOrder {
		fmt.Printf("%d,", worker)
	}
	fmt.Println()
	fmt.Printf("waves:%016x\n", waveDigest)
	fmt.Println("channels-oracle:ok")
	padding.Finish()
}
