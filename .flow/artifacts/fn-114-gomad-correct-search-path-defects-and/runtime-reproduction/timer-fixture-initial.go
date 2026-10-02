package main

import (
 "fmt"
 "runtime"
 "time"
)

func callbackA(done chan string) {
 runtime.Gosched()
 left, right := make(chan struct{}, 1), make(chan struct{}, 1)
 left <- struct{}{}; right <- struct{}{}
 select { case <-left: case <-right: }
 done <- "A"
 runtime.Gosched()
}

func callbackB(done chan string) {
 runtime.Gosched()
 left, right := make(chan struct{}, 1), make(chan struct{}, 1)
 left <- struct{}{}; right <- struct{}{}
 select { case <-left: case <-right: }
 done <- "B"
 runtime.Gosched()
}

func main() {
 armed := make(chan struct{}, 2)
 done := make(chan string, 2)
 go func() { time.AfterFunc(time.Second, func() { callbackA(done) }); armed <- struct{}{} }()
 go func() { time.AfterFunc(time.Second, func() { callbackB(done) }); armed <- struct{}{} }()
 <-armed; <-armed
 // Both timers are armed before the virtual clock advances.
 runtime.Gosched()
 fmt.Println(<-done, <-done)
}
