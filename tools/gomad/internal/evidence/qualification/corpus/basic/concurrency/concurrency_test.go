package concurrency

import (
	"sync"
	"testing"
)

func TestConcurrentTransfersPreserveBalance(t *testing.T) {
	const (
		accounts = 8
		workers  = 16
		moves    = 64
	)
	balances := make([]int, accounts)
	for index := range balances {
		balances[index] = 100
	}
	var lock sync.Mutex
	start := make(chan struct{})
	var group sync.WaitGroup
	for worker := range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			for move := range moves {
				from := (worker + move) % accounts
				to := (from + 1 + worker%3) % accounts
				lock.Lock()
				balances[from]--
				balances[to]++
				lock.Unlock()
			}
		}()
	}
	close(start)
	group.Wait()
	total := 0
	for _, balance := range balances {
		if balance < 0 {
			t.Fatalf("negative account balance: %v", balances)
		}
		total += balance
	}
	if total != accounts*100 {
		t.Fatalf("total balance = %d, want %d: %v", total, accounts*100, balances)
	}
}
