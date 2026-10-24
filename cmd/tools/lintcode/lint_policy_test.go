package main

import (
	"crypto/sha256"
	"encoding/json"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type lintPolicyRule struct {
	Path       string
	PathExcept string `yaml:"path-except"`
	Source     string
	Text       string
	Linters    []string
}

type lintPolicyRuleID struct {
	text, linter, path string
}

func readLintPolicy(t *testing.T) ([]lintPolicyRule, []string) {
	t.Helper()
	data, err := os.ReadFile("../../../.github/.golangci.yml")
	require.NoError(t, err)
	var policy struct {
		Linters struct {
			Exclusions struct {
				Paths []string
				Rules []lintPolicyRule
			}
		}
	}
	require.NoError(t, yaml.Unmarshal(data, &policy))
	require.Len(t, policy.Linters.Exclusions.Rules, 19)
	return policy.Linters.Exclusions.Rules, policy.Linters.Exclusions.Paths
}

func lintPolicyRuleByIdentity(t *testing.T, rules []lintPolicyRule, id lintPolicyRuleID, except bool) lintPolicyRule {
	t.Helper()
	var matches []lintPolicyRule
	for _, rule := range rules {
		if rule.Text != id.text || len(rule.Linters) != 1 || rule.Linters[0] != id.linter || (rule.PathExcept != "") != except {
			continue
		}
		path := rule.Path
		if except {
			path = rule.PathExcept
		}
		if id.path == "" || path == id.path {
			matches = append(matches, rule)
		}
	}
	require.Len(t, matches, 1, "unique exclusion for %+v, path-except=%t", id, except)
	return matches[0]
}

func TestLintPolicyPathExpressions(t *testing.T) {
	t.Parallel()
	rules, paths := readLintPolicy(t)
	for _, tc := range []struct {
		name    string
		rules   []lintPolicyRuleID
		except  bool
		matches []string
		misses  []string
	}{
		{"sleep tests", []lintPolicyRuleID{{text: "time.Sleep", linter: "forbidigo"}}, true, []string{"ordinary/value_test.go", "tests/testcore/value.go", "tools/gomad/value_test.go"}, []string{"ordinary/value.go", "ordinary/value_testXgo", "tests/value.txt"}},
		{"chasm clock", []lintPolicyRuleID{{text: "time.Now", linter: "forbidigo"}}, true, []string{"chasm/lib/value.go", "chasm/lib/model/value.go"}, []string{"chasm/library/value.go", "chasm/lib/valueXgo", "chasm/lib/value.go.txt"}},
		{"chasm clock tests", []lintPolicyRuleID{{text: "time.Now", linter: "forbidigo"}}, false, []string{"chasm/lib/value_test.go", "chasm/lib/model/value_test.go"}, []string{"chasm/lib/value.go", "chasm/lib/value_testXgo", "chasm/lib/value_test.go.txt"}},
		{"cassandra timestamps", []lintPolicyRuleID{{text: "Unix|UnixMilli|UnixNano", linter: "forbidigo"}}, true, []string{"common/persistence/cassandra/value.go", "common/persistence/cassandra/model/value.go"}, []string{"common/persistence/cassandra_extra/value.go", "common/persistence/cassandra/valueXgo", "common/persistence/cassandra/value.go.txt"}},
		{"timestamp tests", []lintPolicyRuleID{{text: "Unix|UnixMilli|UnixNano", linter: "forbidigo"}}, false, []string{"ordinary/value_test.go", "tools/gomad/value_test.go"}, []string{"ordinary/value.go", "ordinary/value_testXgo", "ordinary/value_test.go.txt"}},
		{"test and testing helpers", []lintPolicyRuleID{{text: "panic", linter: "forbidigo", path: "_test\\.go|tests/.+\\.go|common/testing/"}, {text: "(cyclomatic|cognitive)", linter: "revive"}, {text: "(dot-imports|unchecked-type-assertion)", linter: "revive"}}, false, []string{"ordinary/value_test.go", "tests/testcore/value.go", "common/testing/helper.go", "tools/gomad/value_test.go"}, []string{"ordinary/value.go", "ordinary/value_testXgo", "tests/value.txt", "common/testing_extra/value.go"}},
		{"test names", []lintPolicyRuleID{{text: "ST1003: should not use underscores in Go names; (func|method) ", linter: "staticcheck"}}, false, []string{"ordinary/value_test.go", "tests/testcore/value.go", "common/testing/helper.go", "tools/gomad/value_test.go"}, []string{"ordinary/value.go", "ordinary/value_testXgo", "tests/value.txt", "common/testing_extra/value.go"}},
		{"activity model", []lintPolicyRuleID{{text: "panic", linter: "forbidigo", path: "chasm/lib/activity/model/.*\\.go$"}}, false, []string{"chasm/lib/activity/model/value.go", "chasm/lib/activity/model/nested/value.go"}, []string{"chasm/lib/activity/model_extra/value.go", "chasm/lib/activity/model/valueXgo", "chasm/lib/activity/model/value.go.txt"}},
		{"testcore through broad panic rule", []lintPolicyRuleID{{text: "panic", linter: "forbidigo", path: "_test\\.go|tests/.+\\.go|common/testing/"}}, false, []string{"tests/testcore/value.go", "tests/testcore/nested/value.go", "tests/testcore_extra/value.go", "tests/testcore/value.go.txt"}, []string{"tests/testcore/valueXgo"}},
		{"namespace definitions", []lintPolicyRuleID{{text: "ActiveInCluster", linter: "forbidigo"}}, false, []string{"common/namespace/namespace.go", "common/namespace/namespace_test.go", "common/namespace/replication_resolver.go", "common/namespace/replication_resolver_test.go"}, []string{"common/namespace/namespace_extra.go", "common/namespace/namespaceXgo", "common/namespace/replication_resolver.go.txt", "common/namespace_extra/namespace.go"}},
		{"functional test context", []lintPolicyRuleID{{text: "context.Background", linter: "forbidigo"}}, true, []string{"tests/value_test.go", "tests/nested/value_test.go"}, []string{"tests/value.go", "ordinary/value_test.go", "tests/value_testXgo", "tests/value.txt"}},
		{"legacy eventually", []lintPolicyRuleID{{text: "Eventually", linter: "forbidigo"}}, false, []string{"tests/nexus_standalone_test.go", "tests/nexus_workflow_test.go"}, []string{"tests/schedule_test.go", "tests/schedule_migration_test.go", "tests/nexus_standalone_testXgo", "tests/schedule_test.go.txt", "tests/schedule_extra_test.go", "tests/nested/schedule_test.go"}},
		{"legacy collect", []lintPolicyRuleID{{text: "assert\\.CollectT", linter: "forbidigo"}}, false, []string{"tests/nexus_standalone_test.go", "tests/nexus_workflow_test.go"}, []string{"tests/schedule_test.go", "tests/nexus_workflow_testXgo", "tests/nexus_workflow_test.go.txt", "tests/nested/nexus_workflow_test.go"}},
		{"tools revive", []lintPolicyRuleID{{text: "", linter: "revive"}}, false, []string{"tools/gomad/policy.go", "tools/gomad/internal/model/cluster/policy_test.go", "tools/helper.go"}, []string{"tools_extra/helper.go", "ordinary/tools/helper.go", "../tools/gomad/policy.go", "tools/helperXgo", "tools/helper.txt"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, id := range tc.rules {
				rule := lintPolicyRuleByIdentity(t, rules, id, tc.except)
				pattern := rule.Path
				if tc.except {
					pattern = rule.PathExcept
				}
				require.NotEmpty(t, pattern)
				t.Logf("rule=%+v path-except=%t parsed-expression=%q", id, tc.except, pattern)
				compiled, err := regexp.Compile(pattern)
				require.NoError(t, err)
				for _, path := range tc.matches {
					require.True(t, compiled.MatchString(path), "rule %+v: %q must match %q", id, pattern, path)
				}
				for _, path := range tc.misses {
					require.False(t, compiled.MatchString(path), "rule %+v: %q must not match %q", id, pattern, path)
				}
			}
		})
	}
	for _, tc := range []struct{ pattern, match, miss string }{{"^api", "api/policy.go", "service/policy.go"}, {"^proto", "proto/policy.go", "service/policy.go"}, {"^.git", ".github/actions/policy.go", "service/policy.go"}} {
		require.Contains(t, paths, tc.pattern)
		compiled, err := regexp.Compile(tc.pattern)
		require.NoError(t, err)
		require.True(t, compiled.MatchString(tc.match))
		require.False(t, compiled.MatchString(tc.miss))
	}
}

func TestLintPolicyRealGolangci(t *testing.T) {
	binary := os.Getenv("LINT_POLICY_GOLANGCI")
	if binary == "" {
		t.Skip("set LINT_POLICY_GOLANGCI to the pinned golangci-lint v2.13.0 binary to run actual policy fixtures")
	}
	data, err := os.ReadFile(binary)
	require.NoError(t, err)
	t.Logf("actual golangci binary sha256=%x", sha256.Sum256(data))
	versionOutput, err := exec.CommandContext(t.Context(), binary, "version", "--json").Output()
	require.NoError(t, err)
	var version struct{ Version string }
	require.NoError(t, json.Unmarshal(versionOutput, &version))
	require.Equal(t, "2.13.0", version.Version)
	config, err := os.ReadFile("../../../.github/.golangci.yml")
	require.NoError(t, err)
	repo := &lintRepo{t: t, root: t.TempDir()}
	repo.write(".github/.golangci.yml", string(config))
	repo.write("go.mod", "module example.invalid/policy\n\ngo 1.27.1\n")
	writeSource := func(path, source string) {
		t.Helper()
		formatted, err := format.Source([]byte(source))
		require.NoError(t, err)
		repo.write(path, string(formatted))
	}
	for _, path := range []string{"ordinary/policy.go", "api/policy.go", "proto/policy.go", ".github/actions/policy/policy.go"} {
		writeSource(path, "package "+filepath.Base(filepath.Dir(path))+"\n\nfunc Fail() { panic(\"ordinary finding\") }\n")
	}
	writeSource("tools/policy/policy.go", "package policy\n\nfunc Choose(value bool) int { if value { return 1 } else { return 2 } }\n")
	for _, dir := range []string{"tools/policy", "tools/gomad", "tests/mixedbrain"} {
		writeSource(dir+"/policy_test.go", "package "+filepath.Base(dir)+"\n\nimport \"testing\"\n\nfunc TestAllowed(t *testing.T) { panic(\"allowed test finding\") }\n")
	}
	for _, dir := range []string{"tools/gomad", "tests/mixedbrain"} {
		repo.write(dir+"/go.mod", "module example.invalid/nested\n\ngo 1.27.1\n")
		writeSource(dir+"/policy.go", "package "+filepath.Base(dir)+"\n\nfunc Fail() { panic(\"ordinary finding\") }\n")
	}
	writeSource("tools/gomad/choose.go", "package gomad\n\nfunc Choose(value bool) int { if value { return 1 } else { return 2 } }\n")
	repo.git("init", "-q")
	for _, tc := range []struct {
		name, dir string
		packages  []string
		findings  map[string][]string
	}{
		{"root excluded", ".", []string{"./tools/policy", "./api", "./proto", "./.github/actions/policy"}, map[string][]string{}},
		{"root ordinary", ".", []string{"./ordinary"}, map[string][]string{"forbidigo": {"ordinary/policy.go"}}},
		{"nested tools", "tools/gomad", []string{"./..."}, map[string][]string{"forbidigo": {"tools/gomad/policy.go"}}},
		{"nested tests", "tests/mixedbrain", []string{"./..."}, map[string][]string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			outputPath := filepath.Join(t.TempDir(), "issues.json")
			args := append([]string{"run", "--config=" + filepath.Join(repo.root, ".github/.golangci.yml"), "--fix=false", "--build-tags=test_dep", "--output.json.path=" + outputPath}, tc.packages...)
			command := exec.CommandContext(t.Context(), binary, args...)
			command.Dir = filepath.Join(repo.root, tc.dir)
			output, err := command.CombinedOutput()
			if len(tc.findings) == 0 {
				require.NoError(t, err, string(output))
			} else {
				var exit *exec.ExitError
				require.ErrorAs(t, err, &exit, string(output))
				require.Equal(t, 1, exit.ExitCode(), string(output))
			}
			data, err := os.ReadFile(outputPath)
			require.NoError(t, err, string(output))
			var report struct {
				Issues []struct {
					FromLinter string
					Pos        struct{ Filename string }
				}
			}
			require.NoError(t, json.Unmarshal(data, &report))
			got := map[string][]string{}
			for _, issue := range report.Issues {
				got[issue.FromLinter] = append(got[issue.FromLinter], filepath.ToSlash(issue.Pos.Filename))
			}
			require.Equal(t, tc.findings, got, "actual command: %q; output: %s", args, output)
			t.Logf("cwd=%s argv=%q findings=%v", tc.dir, args, got)
		})
	}
}
