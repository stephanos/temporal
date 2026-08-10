package main

import "fmt"

func main() {
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
	fmt.Println(string(result[:]))
}
