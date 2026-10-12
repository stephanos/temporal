package puritycheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRetainedHostFixtures(t *testing.T) {
	data, err := os.ReadFile("../../testdata/architecture/effects.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []struct {
		Name  string
		Files map[string]string
	}
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	names := map[string][]string{"indirect-clock": {"host-effect"}, "environment": {"host-effect"}, "filesystem": {"host-effect"}, "process": {"host-effect", "host-effect"}, "network": {"host-effect"}, "timer": {"host-effect"}, "entropy": {"host-effect"}, "goroutine": {"host-effect"}}
	count := 0
	for _, tc := range fixtures {
		if names[tc.Name] == nil {
			continue
		}
		count++
		t.Run(tc.Name, func(t *testing.T) {
			files := map[string]string{}
			for name, path := range tc.Files {
				data, err := os.ReadFile(filepath.Join("../..", path))
				if err != nil {
					t.Fatal(err)
				}
				files[name] = string(data)
			}
			requireCategories(t, checkFixture(t, files, allPure), names[tc.Name]...)
		})
	}
	if count != 8 {
		t.Fatalf("retained host fixtures=%d want 8", count)
	}
}

func TestRetainedCallbackFixtures(t *testing.T) {
	for _, name := range []string{"pure-callback", "pure-recursive-cycle"} {
		t.Run(name, func(t *testing.T) {
			files := map[string]string{}
			for _, part := range []string{"helper", "record"} {
				data, err := os.ReadFile(filepath.Join("../../testdata/architecture/effects", name, part+".go.txt"))
				if err != nil {
					t.Fatal(err)
				}
				files[part+"/pure.go"] = string(data)
			}
			requireCategories(t, checkFixture(t, files, allPure), "dynamic-call")
			lines := strings.Split(files["helper/pure.go"], "\n")
			for i, line := range lines {
				if strings.Contains(line, "f()") {
					lines[i] += " //gomad:dynamic reviewed 2026-10-12 fixture supplies only a pure callback"
				}
			}
			files["helper/pure.go"] = strings.Join(lines, "\n")
			requireCategories(t, checkFixture(t, files, allPure))
		})
	}
}
