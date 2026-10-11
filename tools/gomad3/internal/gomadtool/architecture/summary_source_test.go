package architecture

import (
	"crypto/sha256"
	"fmt"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const summaryFixtureSource = "package errors\nfunc Join(values ...error) error { return nil }\n"

func summarySourceFixture(t *testing.T) (*Program, string) {
	t.Helper()
	const module = "example.invalid/summarysnapshot"
	directory := t.TempDir()
	standard := filepath.Join(directory, "errors")
	record := filepath.Join(directory, "record")
	for _, dir := range []string{standard, record} {
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	filename := filepath.Join(standard, "join.go")
	if err := os.WriteFile(filename, []byte(summaryFixtureSource), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(record, "pure.go"), []byte("package record\nimport \"errors\"\nfunc Check() { _ = errors.Join(nil) }\n"), 0600); err != nil {
		t.Fatal(err)
	}
	previous := callbackSummarySources["errors/join.go"]
	previousStartup := startupSources["errors"]
	previousMemory := memorySummarySources["errors"]
	callbackSummarySources["errors/join.go"] = fmt.Sprintf("%x", sha256.Sum256([]byte(summaryFixtureSource)))
	startupSources["errors"] = fmt.Sprintf("%x", sha256.Sum256(fmt.Appendf(nil, "%x  join.go\n", sha256.Sum256([]byte(summaryFixtureSource)))))
	memorySummarySources["errors"] = startupSources["errors"]
	t.Cleanup(func() {
		callbackSummarySources["errors/join.go"] = previous
		startupSources["errors"] = previousStartup
		memorySummarySources["errors"] = previousMemory
	})
	program := &Program{
		Module: module, Files: token.NewFileSet(),
		Metadata: map[string]Package{
			"errors":           {ImportPath: "errors", Dir: standard, Standard: true, GoFiles: []string{"join.go"}},
			module + "/record": {ImportPath: module + "/record", Dir: record, Imports: []string{"errors"}, GoFiles: []string{"pure.go"}},
		},
		Packages: map[string]*SourcePackage{}, Functions: map[string]*function{},
	}
	return program, filename
}

func importedSummarySource(t *testing.T) (*Program, string) {
	t.Helper()
	program, filename := summarySourceFixture(t)
	if _, err := program.Import("errors"); err != nil {
		t.Fatal(err)
	}
	return program, filename
}

func summarySourceAnalysis(program *Program) *effectAnalysis {
	return &effectAnalysis{program: program, findings: map[string]Finding{}}
}

func requireSummarySourceRejected(t *testing.T, analysis *effectAnalysis) {
	t.Helper()
	if analysis.checkSummarySource("errors/join.go", token.NoPos) {
		t.Fatal("unqualified callback source snapshot accepted")
	}
	if len(analysis.findings) == 0 {
		t.Fatal("source rejection did not report a conservative finding")
	}
}

func TestSummarySourceSnapshotBindings(t *testing.T) {
	for _, name := range []string{"unchanged", "changed-before-import", "changed-after-import", "changed-file-and-pin-after-import", "unbound", "foreign-package", "foreign-source-package", "directory", "import-path", "nonstandard", "source-metadata", "missing-pin", "changed-pin", "read-error", "unparsed-file", "missing-pin-before-import"} {
		t.Run(name, func(t *testing.T) {
			program, filename := summarySourceFixture(t)
			changed := []byte(summaryFixtureSource + "func Changed() {}\n")
			if name == "changed-before-import" {
				if err := os.WriteFile(filename, changed, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if name == "unparsed-file" {
				metadata := program.Metadata["errors"]
				metadata.GoFiles = []string{"other.go"}
				program.Metadata["errors"] = metadata
				if err := os.WriteFile(filepath.Join(metadata.Dir, "other.go"), []byte(summaryFixtureSource), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if name == "missing-pin-before-import" {
				delete(callbackSummarySources, "errors/join.go")
			}
			if _, err := program.Import("errors"); err != nil {
				t.Fatal(err)
			}
			switch name {
			case "changed-after-import", "changed-file-and-pin-after-import":
				if err := os.WriteFile(filename, changed, 0600); err != nil {
					t.Fatal(err)
				}
				if name == "changed-file-and-pin-after-import" {
					callbackSummarySources["errors/join.go"] = fmt.Sprintf("%x", sha256.Sum256(changed))
				}
			case "unbound":
				delete(program.Packages, "errors")
			case "foreign-package":
				program.Packages["errors"].Types = types.NewPackage("errors", "errors")
			case "foreign-source-package":
				copy := *program.Packages["errors"]
				program.Packages["errors"] = &copy
			case "source-metadata":
				program.Packages["errors"].Metadata.Standard = false
			case "directory":
				metadata := program.Metadata["errors"]
				metadata.Dir = t.TempDir()
				if err := os.WriteFile(filepath.Join(metadata.Dir, "join.go"), []byte(summaryFixtureSource), 0600); err != nil {
					t.Fatal(err)
				}
				program.Metadata["errors"] = metadata
			case "import-path", "nonstandard":
				metadata := program.Metadata["errors"]
				if name == "import-path" {
					metadata.ImportPath = "foreign.invalid/errors"
				} else {
					metadata.Standard = false
				}
				program.Metadata["errors"] = metadata
			case "missing-pin":
				delete(callbackSummarySources, "errors/join.go")
			case "changed-pin":
				callbackSummarySources["errors/join.go"] = "different"
			case "read-error":
				if err := os.Remove(filename); err != nil {
					t.Fatal(err)
				}
			}
			analysis := summarySourceAnalysis(program)
			if name == "unchanged" {
				if !analysis.checkSummarySource("errors/join.go", token.NoPos) || len(analysis.findings) != 0 {
					t.Fatalf("bound unchanged snapshot rejected: %v", analysis.findings)
				}
			} else {
				requireSummarySourceRejected(t, analysis)
			}
		})
	}
}

func TestSummarySourceAnalysisSnapshotLifetime(t *testing.T) {
	program, filename := importedSummarySource(t)
	analysis := summarySourceAnalysis(program)
	if !analysis.checkSummarySource("errors/join.go", token.NoPos) {
		t.Fatal("unchanged source not admitted")
	}
	if err := os.WriteFile(filename, []byte(summaryFixtureSource+"func Changed() {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	// The admitted analysis continues to analyze its exact already-parsed bytes.
	if !analysis.checkSummarySource("errors/join.go", token.NoPos) {
		t.Fatal("mid-analysis disk mutation replaced the admitted immutable snapshot")
	}
	// A fresh analysis, even of the same Program, must revalidate current disk.
	requireSummarySourceRejected(t, summarySourceAnalysis(program))
}

func TestSummarySourceSuccessRequiresCurrentBinding(t *testing.T) {
	for _, name := range []string{"metadata", "package", "pin"} {
		t.Run(name, func(t *testing.T) {
			program, _ := importedSummarySource(t)
			analysis := summarySourceAnalysis(program)
			if !analysis.checkSummarySource("errors/join.go", token.NoPos) {
				t.Fatal("initial admission failed")
			}
			switch name {
			case "metadata":
				metadata := program.Metadata["errors"]
				metadata.ImportPath = "foreign.invalid/errors"
				program.Metadata["errors"] = metadata
			case "package":
				program.Packages["errors"].Types = types.NewPackage("errors", "errors")
			case "pin":
				callbackSummarySources["errors/join.go"] = "different"
			}
			requireSummarySourceRejected(t, analysis)
		})
	}
}

func TestSummarySourceFailureNotCachedAndChainsRetained(t *testing.T) {
	program, filename := importedSummarySource(t)
	if err := os.WriteFile(filename, []byte(summaryFixtureSource+"func Changed() {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	analysis := summarySourceAnalysis(program)
	for _, root := range []string{"first", "second"} {
		analysis.root = root
		analysis.chain = []string{root + "-callback"}
		requireSummarySourceRejected(t, analysis)
	}
	if len(analysis.findings) != 2 {
		t.Fatalf("source failure lost chain-specific findings: %v", analysis.findings)
	}
	if err := os.WriteFile(filename, []byte(summaryFixtureSource), 0600); err != nil {
		t.Fatal(err)
	}
	if !analysis.checkSummarySource("errors/join.go", token.NoPos) {
		t.Fatal("failed validation was cached")
	}
}

func TestSummarySourceEffectsRevalidateSameProgram(t *testing.T) {
	program, filename := importedSummarySource(t)
	if _, err := program.Import(program.Module + "/record"); err != nil {
		t.Fatal(err)
	}
	if findings := program.Effects(); len(findings) != 0 {
		t.Fatalf("original snapshot rejected: %v", findings)
	}
	if err := os.WriteFile(filename, []byte(summaryFixtureSource+"func Changed() {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, finding := range program.Effects() {
		if strings.Contains(finding.Detail, "pinned summary source changed") {
			return
		}
	}
	t.Fatal("later Effects invocation accepted stale current source")
}

func TestSummarySourceImportReadFailure(t *testing.T) {
	program, filename := summarySourceFixture(t)
	if err := os.Remove(filename); err != nil {
		t.Fatal(err)
	}
	if _, err := program.Import("errors"); err == nil {
		t.Fatal("missing parsed bytes accepted by importer")
	}
	requireSummarySourceRejected(t, summarySourceAnalysis(program))
	if err := os.WriteFile(filename, []byte(summaryFixtureSource), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := program.Import("errors"); err != nil {
		t.Fatal(err)
	}
	if !summarySourceAnalysis(program).checkSummarySource("errors/join.go", token.NoPos) {
		t.Fatal("failed import contaminated subsequent exact-byte binding")
	}
}

func TestSummarySourceCachedValidationPreservesDirtyCallbacks(t *testing.T) {
	root := t.TempDir()
	for name, source := range map[string]string{
		"go.mod": "module example.invalid/summarycallbacks\n\ngo 1.27.1\n",
		"record/pure.go": `package record
import ("encoding/json"; "time")
type Dirty struct{}
func (Dirty) MarshalJSON() ([]byte,error) { _ = time.Now(); return []byte("null"),nil }
func Check() { _,_ = json.Marshal(1); _,_ = json.Marshal(Dirty{}) }
`,
	} {
		filename := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(filename), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	program, err := Load(root, "go", Platform{"linux", "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range program.Effects() {
		if finding.Category == "host-effect" && strings.Contains(finding.Detail, "record.Check") && strings.Contains(finding.Detail, "MarshalJSON") && strings.Contains(finding.Detail, "time.Now") {
			return
		}
	}
	t.Fatal("successful earlier pin validation hid a later dirty callback")
}
