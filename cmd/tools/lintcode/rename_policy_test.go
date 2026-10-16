package main

import (
	"encoding/json"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLintPolicyRenamedModule(t *testing.T) {
	binary := os.Getenv("LINT_POLICY_GOLANGCI")
	if binary == "" {
		t.Skip("set LINT_POLICY_GOLANGCI to the pinned golangci-lint binary")
	}
	config, err := os.ReadFile("../../../.github/.golangci.yml")
	require.NoError(t, err)
	for _, tc := range []struct {
		name, path, source                               string
		tracked, committed, finding, historicalTypeError bool
		linter                                           string
	}{
		{name: "historical finding after rename"},
		{name: "edited renamed file keeps historical line", path: "internal/runner/source.go", source: "\nfunc Added() int { return 42 }\n"},
		{name: "new unstaged finding in renamed file", path: "internal/runner/source.go", source: "\nfunc Added() { panic(\"new finding\") }\n", finding: true},
		{name: "new staged finding", path: "internal/runner/new.go", source: "package sample\nfunc Added() { panic(\"new finding\") }\n", tracked: true, finding: true},
		{name: "new committed finding", path: "internal/runner/new.go", source: "package sample\nfunc Added() { panic(\"new finding\") }\n", tracked: true, committed: true, finding: true},
		{name: "new untracked finding", path: "internal/runner/new.go", source: "package sample\nfunc Added() { panic(\"new finding\") }\n", finding: true},
		{name: "new untracked spaced path", path: "internal/runner/new file.go", source: "package sample\nfunc Added() { panic(\"new finding\") }\n", finding: true},
		{name: "new tracked spaced path", path: "internal/runner/new file.go", source: "package sample\nfunc Added() { panic(\"new finding\") }\n", tracked: true, finding: true},
		{name: "new tracked quoted path", path: "internal/runner/new\"file.go", source: "package sample\nfunc Added() { panic(\"new finding\") }\n", tracked: true, finding: true},
		{name: "header shaped added source retains finding", path: "internal/runner/new.go", source: "package sample\nconst Marker = `\n++ b/tools/gomad/other.go\n`\nfunc Added() { panic(\"new finding\") }\n", tracked: true, finding: true},
		{name: "historical typecheck failure survives diff filter", historicalTypeError: true, finding: true, linter: "typecheck"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := newLintRepo(t)
			repo.write(".github/.golangci.yml", string(config))
			historical := "package sample\n\nfunc Historical() { panic(\"historical finding\") }\n"
			if tc.historicalTypeError {
				historical = "package sample\n\nfunc Historical() int { return \"invalid return type\" }\n"
			}
			for i := range 20 {
				historical += "\nfunc Keep" + string(rune('A'+i)) + "() int { return 42 }\n"
			}
			formatted, err := format.Source([]byte(historical))
			require.NoError(t, err)
			historical = string(formatted)
			repo.write("tools/gomad/internal/runner/source.go", historical)
			repo.git("mv", "tools/gomad", "tools/gomad-legacy")
			repo.git("add", ".")
			repo.git("commit", "-qm", "historical module")
			repo.base = strings.TrimSpace(repo.git("rev-parse", "HEAD"))
			repo.git("mv", "tools/gomad-legacy", "tools/gomad")
			repo.git("commit", "-qm", "rename module")
			if tc.path != "" {
				source := tc.source
				if tc.path == "internal/runner/source.go" {
					source = historical + source
				}
				formatted, err := format.Source([]byte(source))
				require.NoError(t, err)
				repo.write("tools/gomad/"+tc.path, string(formatted))
				if tc.tracked {
					repo.git("add", "tools/gomad/"+tc.path)
				}
				if tc.committed {
					repo.git("commit", "-qm", "new finding")
				}
			}
			reportPath := filepath.Join(repo.root, "issues.json")
			repo.write("bin/golangci", "#!/bin/sh\nexec '"+binary+"' \"$@\" --output.json.path='"+reportPath+"'\n")
			output, err := repo.runMake("lint-code-gomad")
			if tc.finding {
				require.Error(t, err, output)
			} else {
				require.NoError(t, err, output)
			}
			require.NotContains(t, output, "historical finding", output)
			data, err := os.ReadFile(reportPath)
			require.NoError(t, err, output)
			var report struct {
				Issues []struct {
					FromLinter string
					Pos        struct{ Filename string }
				}
			}
			require.NoError(t, json.Unmarshal(data, &report))
			if tc.finding {
				require.Len(t, report.Issues, 1, output)
				linter := tc.linter
				if linter == "" {
					linter = "forbidigo"
				}
				require.Equal(t, linter, report.Issues[0].FromLinter)
				path := tc.path
				if tc.historicalTypeError {
					path = "internal/runner/source.go"
				}
				require.Equal(t, "tools/gomad/"+path, filepath.ToSlash(report.Issues[0].Pos.Filename))
			} else {
				require.Empty(t, report.Issues, output)
			}
			t.Log(output)
		})
	}
}

func TestLintPolicyOrdinaryScopes(t *testing.T) {
	binary := os.Getenv("LINT_POLICY_GOLANGCI")
	if binary == "" {
		t.Skip("set LINT_POLICY_GOLANGCI to the pinned golangci-lint binary")
	}
	config, err := os.ReadFile("../../../.github/.golangci.yml")
	require.NoError(t, err)
	for _, tc := range []struct {
		name, path string
		finding    bool
	}{
		{"root new finding", "ordinary/source.go", true},
		{"root historical finding", "ordinary/source.go", false},
		{"nested new finding", "tools/gomad/internal/runner/source.go", true},
		{"nested historical finding", "tools/gomad/internal/runner/source.go", false},
		{"mixedbrain exclusion", "tests/mixedbrain/source.go", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := newLintRepo(t)
			repo.write(".github/.golangci.yml", string(config))
			historical := "package sample\n\nfunc Historical() { panic(\"historical finding\") }\n"
			repo.write(tc.path, historical)
			repo.git("add", ".")
			repo.git("commit", "-qm", "historical ordinary source")
			repo.base = strings.TrimSpace(repo.git("rev-parse", "HEAD"))
			addition := "\nfunc Added() int { return 42 }\n"
			if tc.finding {
				addition = "\nfunc Added() { panic(\"new finding\") }\n"
			}
			formatted, err := format.Source([]byte(historical + addition))
			require.NoError(t, err)
			repo.write(tc.path, string(formatted))
			reportPath := filepath.Join(repo.root, "issues.json")
			repo.write("bin/golangci", "#!/bin/sh\nexec '"+binary+"' \"$@\" --output.json.path='"+reportPath+"'\n")
			output, err := repo.runMake("lint-code-fast")
			finding := tc.finding && !strings.HasPrefix(tc.path, "tests/")
			if finding {
				require.Error(t, err, output)
			} else {
				require.NoError(t, err, output)
			}
			data, err := os.ReadFile(reportPath)
			require.NoError(t, err, output)
			var report struct {
				Issues []struct {
					FromLinter string
					Pos        struct{ Filename string }
				}
			}
			require.NoError(t, json.Unmarshal(data, &report))
			if finding {
				require.Len(t, report.Issues, 1, output)
				require.Equal(t, "forbidigo", report.Issues[0].FromLinter)
				require.Equal(t, tc.path, filepath.ToSlash(report.Issues[0].Pos.Filename))
			} else {
				require.Empty(t, report.Issues, output)
			}
		})
	}
}
