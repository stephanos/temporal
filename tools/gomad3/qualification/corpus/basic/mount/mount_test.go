package mount

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"
)

const (
	inputs     = "/inputs"
	readers    = 8
	rounds     = 4
	allocators = 4
)

// Each first lookup of a mount entry is a host syscall that hands the P to
// another M, so the readers park and wake Ms at host-timed moments while the
// allocators keep the collector cycling. The reads are the same on every
// run; the suite exists so the qualification set compares the schedule, and
// the page-heap layout it prints at the end: which M took the P last is
// recorded in the P and scanned by the collector, and a collection that
// greyed that record at a host-timed point once moved a work buffer and with
// it every later page.
func TestMountReadsUnderCollectionAreRepeatable(t *testing.T) {
	entries, err := os.ReadDir(inputs)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	if len(names) == 0 {
		t.Fatal("no mounted inputs")
	}
	digests := make([][]string, readers)
	var group sync.WaitGroup
	for reader := range readers {
		group.Add(1)
		go func() {
			defer group.Done()
			for round := range rounds {
				for _, name := range names {
					// A lookup of a path the mount has not answered before is
					// its own host round trip, so every reader hands the P off
					// on each name of each round, not only on the first read.
					probe := filepath.Join(inputs, fmt.Sprintf("%s.reader-%d.round-%d", name, reader, round))
					if _, err := os.Stat(probe); !os.IsNotExist(err) {
						t.Errorf("stat %s: %v, want not exist", probe, err)
						return
					}
					contents, err := os.ReadFile(filepath.Join(inputs, name))
					if err != nil {
						t.Error(err)
						return
					}
					sum := sha256.Sum256(contents)
					digests[reader] = append(digests[reader], hex.EncodeToString(sum[:]))
				}
			}
		}()
	}
	for range allocators {
		group.Add(1)
		go func() {
			defer group.Done()
			var retained [][]byte
			for i := 0; i < 20000; i++ {
				retained = append(retained, make([]byte, 1024))
				if len(retained) > 256 {
					retained = retained[:0]
				}
			}
		}()
	}
	group.Wait()
	layout := make([]*[64 << 10]byte, 4)
	for index := range layout {
		layout[index] = new([64 << 10]byte)
	}
	fmt.Printf("layout %p %p %p %p\n", layout[0], layout[1], layout[2], layout[3])
	want := digests[0]
	if len(want) != rounds*len(names) {
		t.Fatalf("digests = %d, want %d", len(want), rounds*len(names))
	}
	for reader := 1; reader < readers; reader++ {
		for index := range want {
			if digests[reader][index] != want[index] {
				t.Fatalf("reader %d digest %d = %s, want %s", reader, index, digests[reader][index], want[index])
			}
		}
	}
}
