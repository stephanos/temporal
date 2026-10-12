package gomad3_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func checkerBudget(root string) (production, tests int, err error) {
	for _, budget := range []struct {
		directory string
		limit     int
	}{{"internal/archcheck", 700}, {"internal/puritycheck", 2000}} {
		directory, before := budget.directory, production
		err = filepath.WalkDir(filepath.Join(root, directory), func(path string, entry os.DirEntry, visitErr error) error {
			if visitErr != nil {
				return visitErr
			}
			if entry.IsDir() || filepath.Ext(path) != ".go" {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			lines := bytes.Count(data, []byte{'\n'})
			if len(data) != 0 && data[len(data)-1] != '\n' {
				lines++
			}
			if strings.HasSuffix(path, "_test.go") {
				tests += lines
			} else {
				production += lines
			}
			return nil
		})
		if err != nil {
			return production, tests, err
		}
		if production-before > budget.limit {
			return production, tests, fmt.Errorf("architecture checker budget exceeded: %s has %d production lines (limit %d)", directory, production-before, budget.limit)
		}
	}
	if production > 2500 || tests > 1500 {
		err = fmt.Errorf("architecture checker budget exceeded: %d production (limit 2500), %d tests (limit 1500)", production, tests)
	}
	return production, tests, err
}

func TestArchitectureCheckerBudget(t *testing.T) {
	production, tests, err := checkerBudget(".")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("checker lines: %d production (limit 2500), %d tests (limit 1500)", production, tests)
}

func TestArchitectureCheckerBudgetRejectsOversize(t *testing.T) {
	for _, test := range []struct {
		name, directory, file string
		lines                 int
	}{
		{"production", "internal/puritycheck", "oversized.go", 1999}, {"tests", "internal/puritycheck", "oversized_test.go", 1501},
		{"archcheck", "internal/archcheck", "oversized.go", 701}, {"puritycheck", "internal/puritycheck", "oversized.go", 2001},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			for _, directory := range []string{"internal/archcheck", "internal/puritycheck"} {
				if err := os.MkdirAll(filepath.Join(root, directory), 0700); err != nil {
					t.Fatal(err)
				}
			}
			source := "package puritycheck\n" + strings.Repeat("// oversized\n", test.lines-1)
			if err := os.WriteFile(filepath.Join(root, test.directory, test.file), []byte(source), 0600); err != nil {
				t.Fatal(err)
			}
			if test.name == "production" {
				if err := os.WriteFile(filepath.Join(root, "internal/archcheck", test.file), []byte(strings.Repeat("\n", 600)), 0600); err != nil {
					t.Fatal(err)
				}
			}
			production, tests, err := checkerBudget(root)
			if err == nil || !strings.Contains(err.Error(), "budget exceeded") {
				t.Fatalf("oversized %s accepted: %d production, %d tests", test.name, production, tests)
			}
			if test.name == "production" && production != 2599 || test.name == "tests" && tests != 1501 {
				t.Fatalf("wrong line observation: %d production, %d tests", production, tests)
			}
		})
	}
}
