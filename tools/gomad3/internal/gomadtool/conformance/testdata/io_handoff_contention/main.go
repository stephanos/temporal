// io_handoff_contention contends the scheduler lock at host-timed moments and
// then prints where the seeded runtime stream stands. Every lookup of a path
// the read-only mount has not answered is a runner syscall that hands the P to
// another M, and the wake-ups after it take the scheduler lock on the M that
// holds the P while the Ms returning from earlier lookups queue and park under
// the same lock. A type-assertion site then misses its cache on every call:
// each miss draws from the seeded stream to decide whether the cache grows, so
// the calls on which it allocates place the stream exactly.
package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
)

const (
	readers    = 6
	lookups    = 3000
	wakes      = 4
	assertions = 1 << 13
)

type (
	s0  int
	s1  int
	s2  int
	s3  int
	s4  int
	s5  int
	s6  int
	s7  int
	s8  int
	s9  int
	s10 int
	s11 int
	s12 int
	s13 int
	s14 int
	s15 int
)

func (s0) String() string  { return "" }
func (s1) String() string  { return "" }
func (s2) String() string  { return "" }
func (s3) String() string  { return "" }
func (s4) String() string  { return "" }
func (s5) String() string  { return "" }
func (s6) String() string  { return "" }
func (s7) String() string  { return "" }
func (s8) String() string  { return "" }
func (s9) String() string  { return "" }
func (s10) String() string { return "" }
func (s11) String() string { return "" }
func (s12) String() string { return "" }
func (s13) String() string { return "" }
func (s14) String() string { return "" }
func (s15) String() string { return "" }

var values = [...]any{s0(0), s1(0), s2(0), s3(0), s4(0), s5(0), s6(0), s7(0), s8(0), s9(0), s10(0), s11(0), s12(0), s13(0), s14(0), s15(0)}

//go:noinline
func assert(value any) fmt.Stringer {
	return value.(fmt.Stringer)
}

func main() {
	if err := contend(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var grown []string
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	previous := stats.Mallocs
	for index := range assertions {
		assert(values[index%len(values)])
		runtime.ReadMemStats(&stats)
		if stats.Mallocs != previous {
			grown = append(grown, fmt.Sprint(index))
			runtime.ReadMemStats(&stats)
			previous = stats.Mallocs
		}
	}
	fmt.Printf("type-assertion cache grew at %s\n", strings.Join(grown, " "))
}

func contend() error {
	errs := make([]error, readers)
	var group sync.WaitGroup
	for reader := range readers {
		group.Add(1)
		go func() {
			defer group.Done()
			ping, pong := make(chan struct{}), make(chan struct{})
			go func() {
				for range ping {
					pong <- struct{}{}
				}
			}()
			defer close(ping)
			for lookup := range lookups {
				probe := fmt.Sprintf("/mounted/missing.reader-%d.lookup-%d", reader, lookup)
				if _, err := os.Stat(probe); !os.IsNotExist(err) {
					errs[reader] = fmt.Errorf("stat %s: %v, want not exist", probe, err)
					return
				}
				for range wakes {
					ping <- struct{}{}
					<-pong
				}
			}
		}()
	}
	group.Wait()
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
