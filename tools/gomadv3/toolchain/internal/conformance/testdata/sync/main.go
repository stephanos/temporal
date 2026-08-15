package main

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"

	"gomadv3.test/internal/layout"
)

func requirePermutation(values []int, size int) {
	seen := make([]bool, size)
	for _, value := range values {
		if value < 0 || value >= size || seen[value] {
			panic("synchronization lost or duplicated a worker")
		}
		seen[value] = true
	}
}

func mutexContention(mutex *sync.Mutex, workers int) []int {
	var waitGroup sync.WaitGroup
	ready := make(chan struct{})
	order := make(chan int, workers)
	mutex.Lock()
	for worker := range workers {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			ready <- struct{}{}
			mutex.Lock()
			order <- worker
			mutex.Unlock()
		}()
		<-ready
		for range 4 {
			runtime.Gosched()
		}
	}
	mutex.Unlock()
	waitGroup.Wait()
	close(order)
	values := make([]int, 0, workers)
	for worker := range order {
		values = append(values, worker)
	}
	requirePermutation(values, workers)
	return values
}

func main() {
	padding := layout.New()
	const workers = 10
	var mutex sync.Mutex
	mutexOrders := make([][]int, 4)
	for wave := range mutexOrders {
		mutexOrders[wave] = mutexContention(&mutex, workers)
	}

	conditionMutex := sync.Mutex{}
	condition := sync.NewCond(&conditionMutex)
	conditionReady := make(chan struct{}, workers)
	conditionOrder := make(chan int, workers)
	for worker := range workers {
		go func() {
			condition.L.Lock()
			conditionReady <- struct{}{}
			condition.Wait()
			conditionOrder <- worker
			condition.L.Unlock()
		}()
	}
	for range workers {
		<-conditionReady
	}
	condition.L.Lock()
	condition.Broadcast()
	condition.L.Unlock()
	conditionValues := make([]int, workers)
	for index := range conditionValues {
		conditionValues[index] = <-conditionOrder
	}
	requirePermutation(conditionValues, workers)

	var once sync.Once
	var onceCount atomic.Int32
	var atomicCount atomic.Int32
	var waitGroup sync.WaitGroup
	for range workers {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			once.Do(func() { onceCount.Add(1) })
			atomicCount.Add(1)
		}()
	}
	waitGroup.Wait()
	if onceCount.Load() != 1 || atomicCount.Load() != workers {
		panic("Once or atomic operation violated its contract")
	}

	var readWrite sync.RWMutex
	readWrite.RLock()
	writerStarted := make(chan struct{})
	writerAcquired := make(chan struct{})
	go func() {
		close(writerStarted)
		readWrite.Lock()
		close(writerAcquired)
		readWrite.Unlock()
	}()
	<-writerStarted
	for range 16 {
		runtime.Gosched()
	}
	select {
	case <-writerAcquired:
		panic("RWMutex writer acquired while a reader held the lock")
	default:
	}
	readWrite.RUnlock()
	<-writerAcquired

	for wave, order := range mutexOrders {
		fmt.Printf("mutex-%d:", wave)
		for _, worker := range order {
			fmt.Print(worker)
		}
		fmt.Println()
	}
	fmt.Print("condition:")
	for _, worker := range conditionValues {
		fmt.Print(worker)
	}
	fmt.Println()
	fmt.Println("sync-oracle:ok")
	padding.Finish()
}
