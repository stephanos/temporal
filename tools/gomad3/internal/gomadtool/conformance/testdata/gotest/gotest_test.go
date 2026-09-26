package gotest

import (
	"fmt"
	"os"
	"runtime"
	"slices"
	"sync"
	"testing"
)

// TestSeedReachesTestBinary proves the exec wrapper delivered the seed and
// prints a schedule the seed decides.
func TestSeedReachesTestBinary(t *testing.T) {
	seed, ok := os.LookupEnv("GOMADSEED")
	if !ok || seed == "" {
		t.Fatal("GOMADSEED did not reach the test binary")
	}
	var mutex sync.Mutex
	var order []int
	var group sync.WaitGroup
	for id := range 6 {
		group.Go(func() {
			runtime.Gosched()
			mutex.Lock()
			order = append(order, id)
			mutex.Unlock()
		})
	}
	group.Wait()
	fmt.Printf("seed=%s order=%v\n", seed, order)
}

// TestDisabledCompatibility prints one line that must be identical under the
// patched and the stock toolchain when no seed is set.
func TestDisabledCompatibility(t *testing.T) {
	if _, set := os.LookupEnv("GOMADSEED"); set {
		t.Fatal("GOMADSEED must be unset for the compatibility comparison")
	}
	keys := make([]string, 0, 8)
	for key := range map[string]int{"delta": 4, "alpha": 1, "charlie": 3, "bravo": 2} {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	fmt.Printf("GOMAD3_COMPAT gomaxprocs=%d keys=%v\n", runtime.GOMAXPROCS(0), keys)
}
