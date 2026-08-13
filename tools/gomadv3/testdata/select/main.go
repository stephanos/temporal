package main

import (
	"fmt"

	"gomadv3.test/internal/layout"
)

func main() {
	padding := layout.New()
	left := make(chan struct{}, 1)
	right := make(chan struct{}, 1)
	left <- struct{}{}
	right <- struct{}{}
	var result [64]byte
	for index := range result {
		select {
		case <-left:
			result[index] = 'L'
			left <- struct{}{}
		case <-right:
			result[index] = 'R'
			right <- struct{}{}
		}
	}
	if !containsBoth(result[:], 'L', 'R') {
		panic("select did not exercise both ready cases")
	}
	mixedReceive := make(chan byte, 1)
	mixedSend := make(chan byte, 1)
	mixedReceive <- 1
	var mixed [64]byte
	for index := range mixed {
		select {
		case value := <-mixedReceive:
			mixed[index] = 'R'
			mixedReceive <- value
		case mixedSend <- 1:
			mixed[index] = 'S'
			<-mixedSend
		}
	}
	if !containsBoth(mixed[:], 'R', 'S') {
		panic("select did not exercise ready send and receive cases")
	}
	leftSend := make(chan struct{}, 1)
	rightSend := make(chan struct{}, 1)
	var sends [64]byte
	for index := range sends {
		select {
		case leftSend <- struct{}{}:
			sends[index] = 'L'
			<-leftSend
		case rightSend <- struct{}{}:
			sends[index] = 'R'
			<-rightSend
		}
	}
	if !containsBoth(sends[:], 'L', 'R') {
		panic("select did not exercise both ready sends")
	}

	closed := make(chan int)
	close(closed)
	value, open := <-closed
	if value != 0 || open {
		panic("closed channel receive returned an invalid result")
	}

	var nilChannel <-chan int
	ready := make(chan int, 1)
	ready <- 7
	select {
	case <-nilChannel:
		panic("select chose a nil channel")
	case value = <-ready:
		if value != 7 {
			panic("select received an invalid ready value")
		}
	default:
		panic("select chose default with a ready case")
	}
	ready <- 8
	select {
	case ready <- 9:
		panic("select sent to a full channel")
	default:
	}

	send := make(chan int, 1)
	select {
	case send <- 11:
	default:
		panic("select defaulted with a ready send")
	}
	if value = <-send; value != 11 {
		panic("select send produced an invalid value")
	}

	fmt.Printf("random:%s\n", string(result[:]))
	fmt.Printf("mixed:%s\n", string(mixed[:]))
	fmt.Printf("sends:%s\n", string(sends[:]))
	fmt.Println("select-oracle:ok")
	padding.Finish()
}

func containsBoth(values []byte, left, right byte) bool {
	var foundLeft, foundRight bool
	for _, value := range values {
		foundLeft = foundLeft || value == left
		foundRight = foundRight || value == right
	}
	return foundLeft && foundRight
}
