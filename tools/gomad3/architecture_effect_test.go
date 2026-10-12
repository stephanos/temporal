package gomad3_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/archcheck"
	"go.temporal.io/server/tools/gomad3/internal/puritycheck"
)

func TestArchitectureEffectFixtures(t *testing.T) {
	data, err := os.ReadFile("testdata/architecture/effects.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []struct {
		Name, Category, Detail string
		Files                  map[string]string
		PureFiles              []string `json:"pure_files"`
		Reviewed               bool
	}
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	if len(fixtures) != 36 {
		t.Fatalf("effect fixtures=%d want 36", len(fixtures))
	}
	for _, fixture := range fixtures {
		t.Run(fixture.Name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.invalid/fitness\n\ngo 1.27.1\n"), 0600); err != nil {
				t.Fatal(err)
			}
			for name, path := range fixture.Files {
				source, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				target := filepath.Join(root, name)
				if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(target, source, 0600); err != nil {
					t.Fatal(err)
				}
			}
			for _, path := range fixture.PureFiles {
				if _, exists := fixture.Files[path]; !exists {
					t.Fatalf("unknown pure fixture file %s", path)
				}
			}
			check := func() []puritycheck.Finding {
				findings, err := puritycheck.Check(root, sourceGoCommand(),
					archcheck.Platform{OS: runtime.GOOS, Arch: runtime.GOARCH},
					func(path string) bool { return slices.Contains(fixture.PureFiles, path) })
				if err != nil {
					t.Fatal(err)
				}
				return findings
			}
			findings := check()
			category := fixture.Category
			if fixture.Reviewed {
				category = "dynamic-call"
			}
			if category != "" && len(findings) == 0 {
				t.Fatalf("missing %s (%s)", category, fixture.Detail)
			}
			for _, finding := range findings {
				if finding.Category != category || !strings.Contains(finding.Detail, fixture.Detail) {
					t.Fatalf("want %s (%s): %+v", category, fixture.Detail, findings)
				}
			}
			if category != "dynamic-call" {
				return
			}
			linesByFile := map[string]map[int]bool{}
			for _, finding := range findings {
				parts := strings.Split(finding.Path, ":")
				if len(parts) != 3 {
					t.Fatalf("invalid finding position %s", finding.Path)
				}
				line, err := strconv.Atoi(parts[1])
				if err != nil {
					t.Fatal(err)
				}
				if linesByFile[parts[0]] == nil {
					linesByFile[parts[0]] = map[int]bool{}
				}
				linesByFile[parts[0]][line-1] = true
			}
			for name, marked := range linesByFile {
				path := filepath.Join(root, name)
				source, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				lines := strings.Split(string(source), "\n")
				for line := range marked {
					lines[line] += " //gomad:dynamic reviewed 2026-10-12 callback boundary is explicitly reviewed by this fixture"
				}
				if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if marked := check(); len(marked) != 0 {
				t.Fatalf("reviewed callback rejected: %+v", marked)
			}
		})
	}
}
