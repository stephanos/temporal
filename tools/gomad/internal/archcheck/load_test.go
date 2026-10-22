package archcheck

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestStdlibAllowlistVersion(t *testing.T) {
	if err := checkVersion(runtime.Version()); err != nil {
		t.Fatal(err)
	}
	root := purityFixture(t, map[string]string{"internal/evidence/record/pure.go": "package record"})
	fake := filepath.Join(root, "fake-go")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\n[ \"$1\" = version ] || exit 13\nprintf 'go version go1.99.2 linux/arm64\\n'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	_, err := CheckPurity(root, fake, host, allPure)
	if err == nil || !strings.Contains(err.Error(), "go1.27") || !strings.Contains(err.Error(), "go1.99") {
		t.Fatalf("version mismatch must name both minors: %v", err)
	}
}

func TestStdlibSymbols(t *testing.T) {
	for _, tc := range []struct {
		pkg, symbol string
		want        bool
	}{
		{"time", "Now", false}, {"time", "Date", true}, {"time", "UTC", true}, {"time", "Missing", false}, {"bytes", "NewReader", true}, {"bytes", "Buffer.Write", true}, {"os", "Getenv", false}, {"syscall", "EINVAL", true}, {"syscall", "EpollCreate1", false}, {"syscall", "Open", false}, {"fmt", "Println", false},
	} {
		if got := allowed(tc.pkg, tc.symbol); got != tc.want {
			t.Errorf("%s.%s=%t want %t", tc.pkg, tc.symbol, got, tc.want)
		}
	}
	original := pureSymbols["bytes"]
	t.Cleanup(func() { pureSymbols["bytes"] = original })
	for _, patterns := range [][]string{{"*", "!NewReader"}, {"!NewReader", "*"}} {
		pureSymbols["bytes"] = patterns
		if allowed("bytes", "NewReader") || !allowed("bytes", "Buffer.Write") {
			t.Fatalf("exclusion must win independent of order: %v", patterns)
		}
	}
}

func TestLoaderUsesExportData(t *testing.T) {
	root := purityFixture(t, map[string]string{
		"internal/evidence/record/pure.go":             "package record;import(\"bytes\";\"example.invalid/fitness/host\");func F(){var b bytes.Buffer;_,_ = b.Write(nil);host.F()}",
		"host/host.go":                                 "package host;func F(){}",
		"internal/evidence/record/excluded_windows.go": "package record;import \"os\";func Windows(){_,_ = os.ReadFile(\"x\")}",
	})
	p, err := load(root, goCommand(), host, func(path string) bool { return path == "internal/evidence/record/pure.go" })
	if err != nil {
		t.Fatal(err)
	}
	if len(p.packages) != 1 || p.packages["example.invalid/fitness/internal/evidence/record"] == nil {
		t.Fatalf("loaded non-pure source packages: %v", p.packages)
	}
	if p.metadata["bytes"].Export == "" || p.metadata["example.invalid/fitness/host"].Export == "" {
		t.Fatal("dependency export data missing")
	}
	for _, pkg := range p.packages {
		for _, file := range pkg.files {
			filename := p.fset.Position(file.Pos()).Filename
			if !strings.HasPrefix(filename, p.root+string(filepath.Separator)) {
				t.Fatalf("parsed dependency source: %s", filename)
			}
			if strings.Contains(filename, "excluded_windows.go") && runtime.GOOS != "windows" {
				t.Fatalf("parsed platform-excluded file: %s", filename)
			}
		}
	}
}

func TestPackageErrorsNotHiddenByAssetNames(t *testing.T) {
	findings, err := CheckPurity(purityFixture(t, map[string]string{
		"internal/evidence/record/pure.go": "package record;func F(){}",
		"other/runtime/overlay/bad.go":     "package bad;import _ \"example.invalid/missing\"",
	}), goCommand(), host, allPure)
	if err == nil {
		t.Fatal("arbitrary runtime/overlay package error was hidden")
	}
	requireCategories(t, findings, "package-error")
}

func TestReviewMarkLog(t *testing.T) {
	p, err := load(purityFixture(t, map[string]string{"internal/evidence/record/pure.go": "package record;func F(f func()){\n//gomad:dynamic reviewed 2020-01-01 supplied by caller\nf()\n}"}), goCommand(), host, allPure)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	findings := analyze(p, time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC), log.New(&output, "", 0))
	requireCategories(t, findings)
	for _, text := range []string{"example.invalid/fitness/internal/evidence/record: 1 dynamic review marks", "warning: aged dynamic review", "supplied by caller"} {
		if !strings.Contains(output.String(), text) {
			t.Fatalf("missing %q: %s", text, &output)
		}
	}
}

func TestLoadPureSet(t *testing.T) {
	root, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	pure := func(path string) bool {
		prefixes := []string{"internal/model/world/"}
		for _, prefix := range prefixes {
			if strings.HasPrefix(path, prefix) && !strings.HasPrefix(path, "internal/model/world/process/") {
				return true
			}
		}
		return path == "internal/model/clock/simulation_progress.go"
	}
	started := time.Now()
	p, err := load(root, goCommand(), host, pure)
	if err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(started)
	if elapsed >= 30*time.Second {
		t.Fatalf("pure set loading took %s, limit 30s", elapsed)
	}
	for _, required := range []string{"internal/model/world", "internal/model/world/mailbox", "internal/model/clock"} {
		if p.packages["go.temporal.io/server/tools/gomad/"+required] == nil {
			t.Errorf("retained pure package %s was omitted", required)
		}
	}
	t.Logf("loaded %d pure-source packages using export data in %s", len(p.packages), elapsed)
}
