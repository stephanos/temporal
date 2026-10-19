package archcheck

import (
	"slices"
	"testing"
)

func TestHostEffectSitesReportsClockGoroutineAndMapRange(t *testing.T) {
	root := purityFixture(t, map[string]string{
		"host/host.go": `package host
import ("context";"maps";"math/rand/v2";"slices";"sort";"time")
type Session struct{ seen map[string]int }
func start() {}
func (s *Session) Run() time.Duration {
	begin := time.Now()
	go start()
	go func() {}()
	for key := range s.seen { _ = key }
	for range []int{1} {}
	clock := time.Now
	_ = clock
	timer := time.NewTimer(time.Second)
	timer.Reset(time.Second)
	_, cancel := context.WithTimeout(context.Background(), time.Second)
	cancel()
	_ = rand.N(3)
	for key := range maps.Keys(s.seen) { _ = key }
	return time.Since(begin)
}
func Pure(values map[int]bool) (n int) {
	for range values { n++ }
	_ = time.Unix(0, 0)
	keys := []int{2, 1}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	sort.SliceStable(keys, func(i, j int) bool { return keys[i] < keys[j] })
	slices.SortFunc(keys, func(a, b int) int { return a - b })
	slices.Sort(keys)
	return n
}
`,
		"other/other.go": "package other\nimport \"time\"\nvar Now = time.Now\n",
	})
	sites, err := HostEffectSites(root, goCommand(), host, func(path string) bool { return path == "host/host.go" })
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for _, site := range sites {
		// time.Now is called once and taken once as a function value.
		if want := map[bool]int{true: 2, false: 1}[site.Symbol == "time.Now"]; site.Count != want {
			t.Fatalf("count %+v, want %d", site, want)
		}
		keys = append(keys, site.Key())
	}
	want := []string{
		"host/host.go|Pure|map-range|values",
		"host/host.go|Pure|unstable-sort|slices.SortFunc keys",
		"host/host.go|Pure|unstable-sort|sort.Slice keys",
		"host/host.go|Session.Run|clock|context.WithTimeout",
		"host/host.go|Session.Run|clock|time.NewTimer",
		"host/host.go|Session.Run|clock|time.Now",
		"host/host.go|Session.Run|clock|time.Since",
		"host/host.go|Session.Run|clock|time.Timer.Reset",
		"host/host.go|Session.Run|entropy|math/rand/v2.N",
		"host/host.go|Session.Run|goroutine|func literal",
		"host/host.go|Session.Run|goroutine|start",
		"host/host.go|Session.Run|map-range|maps.Keys s.seen",
		"host/host.go|Session.Run|map-range|s.seen",
	}
	if !slices.Equal(keys, want) {
		t.Fatalf("sites = %q\nwant %q", keys, want)
	}
}
