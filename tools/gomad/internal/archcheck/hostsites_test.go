package archcheck

import (
	"slices"
	"testing"
)

func TestHostEffectSitesReportsClockGoroutineAndMapRange(t *testing.T) {
	root := purityFixture(t, map[string]string{
		"host/host.go": `package host
import ("slices";"sort";"time")
type Session struct{ seen map[string]int }
func start() {}
func (s *Session) Run() time.Duration {
	begin := time.Now()
	go start()
	go func() {}()
	for key := range s.seen { _ = key }
	for range []int{1} {}
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
		if site.Count != 1 {
			t.Fatalf("count %+v", site)
		}
		keys = append(keys, site.Key())
	}
	want := []string{
		"host/host.go|Pure|map-range|values",
		"host/host.go|Pure|unstable-sort|slices.SortFunc keys",
		"host/host.go|Pure|unstable-sort|sort.Slice keys",
		"host/host.go|Session.Run|clock|time.Now",
		"host/host.go|Session.Run|clock|time.Since",
		"host/host.go|Session.Run|goroutine|func literal",
		"host/host.go|Session.Run|goroutine|start",
		"host/host.go|Session.Run|map-range|s.seen",
	}
	if !slices.Equal(keys, want) {
		t.Fatalf("sites = %q\nwant %q", keys, want)
	}
}
