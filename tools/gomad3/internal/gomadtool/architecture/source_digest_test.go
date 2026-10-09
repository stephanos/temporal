package architecture

import (
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

func TestSourceDigestFraming(t *testing.T) {
	const path = "example.invalid/source-digest"
	const want = "67ad170a9788e8b8d82ab27c22a54c6f98af43713537fa5775a11f1424d4bcd5"
	for _, test := range []struct {
		name string
		pins map[string]string
	}{
		{"startup", startupSources},
		{"memory", memorySummarySources},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			for _, file := range []struct{ name, contents string }{
				{"z.go", "package z\n"},
				{"ignored.txt", "ignored\n"},
				{"a.go", "package a\n"},
			} {
				if err := os.WriteFile(filepath.Join(directory, file.name), []byte(file.contents), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Mkdir(filepath.Join(directory, "directory.go"), 0700); err != nil {
				t.Fatal(err)
			}
			test.pins[path] = want
			t.Cleanup(func() { delete(test.pins, path) })
			metadata := Package{ImportPath: path, Dir: directory, Standard: true}
			analysis := &effectAnalysis{
				program:  &Program{Files: token.NewFileSet(), Metadata: map[string]Package{path: metadata}},
				findings: map[string]Finding{},
			}
			if test.name == "startup" {
				analysis.checkStartupSource(metadata, token.NoPos)
			} else if !analysis.checkMemorySource(path, token.NoPos) {
				t.Fatalf("literal source digest rejected: %v", analysis.findings)
			}
			if len(analysis.findings) != 0 {
				t.Fatalf("literal source digest rejected: %v", analysis.findings)
			}
			if err := os.WriteFile(filepath.Join(directory, "a.go"), []byte("package changed\n"), 0600); err != nil {
				t.Fatal(err)
			}
			analysis.checkedMemory = nil
			if test.name == "startup" {
				analysis.checkStartupSource(metadata, token.NoPos)
			} else if analysis.checkMemorySource(path, token.NoPos) {
				t.Fatal("changed source digest was accepted")
			}
			if len(analysis.findings) != 1 {
				t.Fatalf("changed source digest findings = %v", analysis.findings)
			}
		})
	}
}
