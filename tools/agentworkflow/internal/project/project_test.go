package project

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/agentworkflow/internal/recipe"
)

func TestStarterDetectsGoWithoutExecutingProjectCode(t *testing.T) {
	root := t.TempDir()
	writeProjectFile(t, filepath.Join(root, "go.mod"), "module example.com/test\n")
	writeProjectFile(t, filepath.Join(root, "AGENTS.md"), "instructions")
	profile, err := Starter(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(profile.Checks) != 1 || profile.Checks[0].Command[0] != "go" || profile.Checks[0].Enabled == nil || !*profile.Checks[0].Enabled {
		t.Fatalf("profile = %#v", profile)
	}
	if len(profile.Instructions) != 1 || profile.Instructions[0] != "AGENTS.md" {
		t.Fatalf("instructions = %#v", profile.Instructions)
	}
}

func TestStarterIgnoresManifestAndInstructionSymlinks(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writeProjectFile(t, filepath.Join(outside, "AGENTS.md"), "outside")
	writeProjectFile(t, filepath.Join(outside, "go.mod"), "module outside\n")
	for _, name := range []string{"AGENTS.md", "go.mod"} {
		if err := os.Symlink(filepath.Join(outside, name), filepath.Join(root, name)); err != nil {
			t.Skipf("create symlink: %v", err)
		}
	}
	profile, err := Starter(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(profile.Instructions) != 0 || len(profile.Checks) != 0 {
		t.Fatalf("starter trusted symlinked project metadata: %#v", profile)
	}
}

func TestWriteStarterNeverOverwritesProfile(t *testing.T) {
	root := t.TempDir()
	path, err := WriteStarter(root, "")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, ".spec", "agentworkflow.yaml")
	if path != want {
		t.Fatalf("starter path = %q, want %q", path, want)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "schema: agentworkflow.config/v1") || strings.HasPrefix(strings.TrimSpace(string(data)), "{") {
		t.Fatalf("starter is not YAML:\n%s", data)
	}
	for _, fragment := range []string{"instructions: []", "checks: []", "prompt: |-", "targets: {}"} {
		if !strings.Contains(string(data), fragment) {
			t.Fatalf("starter does not contain %q:\n%s", fragment, data)
		}
	}
	if _, err := Load("", root, ""); err != nil {
		t.Fatalf("load generated starter: %v", err)
	}
	if _, err := WriteStarter(root, ""); err == nil {
		t.Fatal("existing profile was overwritten")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestLoadStrictlyResolvesTargetAndDisabledChecks(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".spec", "custom.yaml")
	writeProjectFile(t, path, validConfigYAML(`
checks:
  - name: disabled
    command: [false]
    required: true
    enabled: false
targets:
  api:
    checks:
      - name: api
        command: [go, test]
        directory: api
        timeout: 1m
        required: true
        enabled: true`))
	resolved, err := Load(path, root, "api")
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved.Checks) != 1 || resolved.Checks[0].Name != "api" || resolved.Checks[0].Timeout.Duration != time.Minute {
		t.Fatalf("resolved = %#v", resolved)
	}
	if _, err := Load(path, root, "missing"); err == nil {
		t.Fatal("missing target was accepted")
	}
}

func TestLoadRejectsUnknownFieldsAndOversizedProfiles(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "profile.yaml")
	writeProjectFile(t, path, validConfigYAML("unknown: true"))
	if _, err := Load(path, root, ""); err == nil {
		t.Fatal("unknown profile field was accepted")
	}
	large := make([]byte, (1<<20)+1)
	if err := os.WriteFile(path, large, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path, root, ""); err == nil {
		t.Fatal("oversized profile was accepted")
	}
}

func TestLoadAcceptsFlowCollectionsAndMultilinePrompts(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".spec", "agentworkflow.yaml")
	configuration := strings.Replace(validConfigYAML("checks: []"), "prompt: discover prompt", `prompt: |-
        Read the manifests first.
        Do not modify files.`, 1)
	writeProjectFile(t, path, configuration)
	resolved, err := Load("", root, "")
	if err != nil {
		t.Fatal(err)
	}
	if prompt := resolved.Workflow.Stage(recipe.Discover).Prompt; prompt != "Read the manifests first.\nDo not modify files." {
		t.Fatalf("discover prompt = %q", prompt)
	}
}

func TestLoadRejectsUnsafeOrAmbiguousYAML(t *testing.T) {
	base := validConfigYAML("checks: []")
	cases := map[string]string{
		"JSON syntax":        `{"schema":"agentworkflow.config/v1"}`,
		"unknown field":      base + "unknown: true\n",
		"duplicate key":      base + "schema: agentworkflow.config/v1\n",
		"anchor":             strings.Replace(base, "prompt: discover prompt", "prompt: &shared discover prompt", 1),
		"alias":              strings.Replace(base, "prompt: discover prompt", "prompt: &shared discover prompt", 1) + "alias: *shared\n",
		"merge key":          base + "merged: &base {value: one}\ncopy:\n  <<: *base\n",
		"explicit tag":       strings.Replace(base, "schema: agentworkflow.config/v1", "schema: !contract agentworkflow.config/v1", 1),
		"non-string key":     base + "1: value\n",
		"null":               strings.Replace(base, "schema: agentworkflow.config/v1", "schema: null", 1),
		"multiple documents": base + "---\nschema: second\n",
	}
	for name, configuration := range cases {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "config.yaml")
			writeProjectFile(t, path, configuration)
			if _, err := Load(path, root, ""); err == nil {
				t.Fatalf("unsafe configuration was accepted:\n%s", configuration)
			}
		})
	}
}

func TestLoadRequiresCanonicalExplicitWorkflow(t *testing.T) {
	base := validConfigYAML("checks: []")
	cases := map[string]string{
		"missing workflow": base[:strings.Index(base, "workflow:\n")] + "checks: []\n",
		"missing stage": strings.Replace(base, `    - kind: check
      enabled: true
`, "", 1),
		"reordered stage": strings.Replace(base, `    - kind: check
      enabled: true
    - kind: review
      enabled: true
      prompt: review prompt
`, `    - kind: review
      enabled: true
      prompt: review prompt
    - kind: check
      enabled: true
`, 1),
		"missing enabled": strings.Replace(base, `    - kind: discover
      enabled: true
`, `    - kind: discover
`, 1),
		"missing prompt": strings.Replace(base, "      prompt: implement prompt\n", "", 1),
		"check prompt": strings.Replace(base, `    - kind: check
      enabled: true
`, `    - kind: check
      enabled: true
      prompt: bypass checks
`, 1),
		"implicit apply": strings.Replace(base, "      mode: explicit\n", "      mode: automatic\n", 1),
	}
	for name, configuration := range cases {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "config.yaml")
			writeProjectFile(t, path, configuration)
			if _, err := Load(path, root, ""); err == nil {
				t.Fatalf("invalid workflow was accepted:\n%s", configuration)
			}
		})
	}
}

func TestLoadProtectsHumanInputsAndRejectsExcludedSpec(t *testing.T) {
	root := t.TempDir()
	writeProjectFile(t, filepath.Join(root, ".spec", "instructions", "architecture.md"), "architecture")
	path := filepath.Join(root, ".spec", "custom.yml")
	configuration := strings.Replace(
		validConfigYAML("checks: []"),
		"instructions: []",
		"instructions: [.spec/instructions/architecture.md]",
		1,
	)
	writeProjectFile(t, path, configuration)
	resolved, err := Load(path, root, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, protected := range []string{".spec", ".spec/custom.yml", ".spec/instructions/architecture.md"} {
		if !slicesContains(resolved.ForbiddenPaths, protected) {
			t.Fatalf("forbidden paths %v do not protect %q", resolved.ForbiddenPaths, protected)
		}
	}

	excluded := strings.Replace(configuration, "source:\n  mode: directory-copy", "source:\n  mode: directory-copy\n  exclude: [.spec]", 1)
	writeProjectFile(t, path, excluded)
	if _, err := Load(path, root, ""); err == nil {
		t.Fatal("excluding .spec was accepted")
	}

	escaped := strings.Replace(configuration, "instructions: [.spec/instructions/architecture.md]", "instructions: [../outside.md]", 1)
	writeProjectFile(t, path, escaped)
	if _, err := Load(path, root, ""); err == nil {
		t.Fatal("escaping instruction path was accepted")
	}
}

func TestLoadRequiresYAMLInsideProjectAndReportsLegacyConfiguration(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "agentworkflow.yaml")
	writeProjectFile(t, outside, validConfigYAML("checks: []"))
	if _, err := Load(outside, root, ""); err == nil {
		t.Fatal("configuration outside project was accepted")
	}
	jsonPath := filepath.Join(root, "agentworkflow.json")
	writeProjectFile(t, jsonPath, `{}`)
	if _, err := Load(jsonPath, root, ""); err == nil {
		t.Fatal("JSON extension was accepted")
	}
	legacy := filepath.Join(root, ".agentworkflow", "project.json")
	writeProjectFile(t, legacy, `{}`)
	if _, err := Load("", root, ""); err == nil || !strings.Contains(err.Error(), "legacy JSON configuration") || !strings.Contains(err.Error(), ".spec") {
		t.Fatalf("legacy error = %v", err)
	}
}

func TestConfigurationAndInstructionSymlinksCannotEscapeProject(t *testing.T) {
	t.Run("load configuration", func(t *testing.T) {
		root := t.TempDir()
		outside := t.TempDir()
		writeProjectFile(t, filepath.Join(outside, "agentworkflow.yaml"), validConfigYAML("checks: []"))
		if err := os.Symlink(outside, filepath.Join(root, ".spec")); err != nil {
			t.Skipf("create symlink: %v", err)
		}
		if _, err := Load("", root, ""); err == nil {
			t.Fatal("configuration symlink escaping the project was accepted")
		}
	})

	t.Run("write starter", func(t *testing.T) {
		root := t.TempDir()
		outside := t.TempDir()
		if err := os.Symlink(outside, filepath.Join(root, ".spec")); err != nil {
			t.Skipf("create symlink: %v", err)
		}
		if _, err := WriteStarter(root, ""); err == nil {
			t.Fatal("starter followed a configuration directory symlink outside the project")
		}
		if _, err := os.Stat(filepath.Join(outside, "agentworkflow.yaml")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("starter wrote outside the project: %v", err)
		}
	})

	t.Run("instruction", func(t *testing.T) {
		root := t.TempDir()
		outside := filepath.Join(t.TempDir(), "outside.md")
		writeProjectFile(t, outside, "outside")
		if err := os.Symlink(outside, filepath.Join(root, "GUIDE.md")); err != nil {
			t.Skipf("create symlink: %v", err)
		}
		path := filepath.Join(root, "config.yaml")
		configuration := strings.Replace(validConfigYAML("checks: []"), "instructions: []", "instructions: [GUIDE.md]", 1)
		writeProjectFile(t, path, configuration)
		if _, err := Load(path, root, ""); err == nil {
			t.Fatal("instruction symlink escaping the project was accepted")
		}
	})
}

func TestLoadValidatesTheCompleteConfiguration(t *testing.T) {
	base := validConfigYAML("checks: []")
	cases := map[string]string{
		"source mode":           strings.Replace(base, "mode: directory-copy", "mode: git", 1),
		"nested spec exclusion": strings.Replace(base, "mode: directory-copy", "mode: directory-copy\n  exclude: [.spec/tasks]", 1),
		"missing instruction":   strings.Replace(base, "instructions: []", "instructions: [.spec/missing.md]", 1),
		"invalid environment":   strings.Replace(base, "allow: [PATH]", "allow: [PATH, BAD=VALUE]", 1),
		"invalid assurance":     strings.Replace(base, "assurance: standard", "assurance: heroic", 1),
		"invalid severity":      strings.Replace(base, "blocking_severity: medium", "blocking_severity: urgent", 1),
		"invalid reviewer":      strings.Replace(base, "blocking_severity: medium", "blocking_severity: medium\n  reviewers: [bad/reviewer]", 1),
		"duplicate checks": strings.Replace(base, "checks: []", `checks:
  - name: test
    command: [go, test]
    required: true
    enabled: true
  - name: test
    command: [go, vet]
    required: true
    enabled: true`, 1),
		"invalid target name":  strings.Replace(base, "checks: []", "checks: []\ntargets:\n  bad/name: {}", 1),
		"invalid target check": strings.Replace(base, "checks: []", "checks: []\ntargets:\n  api:\n    checks:\n      - name: test\n        command: []\n        required: true\n        enabled: true", 1),
	}
	for name, configuration := range cases {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "config.yaml")
			writeProjectFile(t, path, configuration)
			if _, err := Load(path, root, ""); err == nil {
				t.Fatalf("invalid configuration was accepted:\n%s", configuration)
			}
		})
	}

	root := t.TempDir()
	if _, err := Load("", root, ""); err == nil || !strings.Contains(err.Error(), "agentworkflow init") {
		t.Fatalf("missing configuration error = %v", err)
	}
}

func TestExplainEmitsResolvedYAMLWorkflow(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".spec", "agentworkflow.yaml")
	writeProjectFile(t, path, validConfigYAML("checks: []"))
	resolved, err := Load("", root, "")
	if err != nil {
		t.Fatal(err)
	}
	data, err := Explain(resolved)
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(strings.TrimSpace(string(data)), "{") || !strings.Contains(string(data), "schema: agentworkflow.resolved-config/v1") || !strings.Contains(string(data), "kind: discover") {
		t.Fatalf("explanation is not resolved YAML:\n%s", data)
	}
}

func TestDiscoverRetainsOnlyDeclaredInstructionContents(t *testing.T) {
	root := t.TempDir()
	writeProjectFile(t, filepath.Join(root, "AGENTS.md"), "trusted instructions")
	writeProjectFile(t, filepath.Join(root, "go.mod"), "module example.com/test")
	writeProjectFile(t, filepath.Join(root, "secret.txt"), "not prompt context")
	inventory, err := Discover(context.Background(), root, []string{"AGENTS.md"}, 100, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Instructions) != 1 || inventory.Instructions[0].Content != "trusted instructions" {
		t.Fatalf("inventory = %#v", inventory)
	}
	for _, instruction := range inventory.Instructions {
		if instruction.Path == "secret.txt" {
			t.Fatal("undeclared file content entered prompt inventory")
		}
	}
	if _, err := Discover(context.Background(), root, []string{"missing"}, 100, 1<<20); err == nil {
		t.Fatal("missing instruction was accepted")
	}
}

func TestDurationStrictlyParsesString(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "profile.yaml")
	writeProjectFile(t, path, validConfigYAML(`
checks:
  - name: test
    command: [go, test]
    timeout: 2m
    required: true
    enabled: true`))
	resolved, err := Load(path, root, "")
	if err != nil || len(resolved.Checks) != 1 || resolved.Checks[0].Timeout.Duration != 2*time.Minute {
		t.Fatalf("duration = %#v, %v", resolved.Checks, err)
	}
	writeProjectFile(t, path, validConfigYAML(`
checks:
  - name: test
    command: [go, test]
    timeout: 120
    required: true
    enabled: true`))
	if _, err := Load(path, root, ""); err == nil {
		t.Fatal("numeric duration was accepted")
	}
	writeProjectFile(t, path, validConfigYAML(`
checks:
  - name: test
    command: [go, test]
    timeout: ""
    required: true
    enabled: true`))
	if _, err := Load(path, root, ""); err == nil {
		t.Fatal("empty duration was accepted")
	}
}

func TestLoadPreservesExplicitEmptyEnvironmentAndZeroRepairBudget(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".spec", "agentworkflow.yaml")
	configuration := strings.Replace(validConfigYAML("checks: []"), "allow: [PATH]", "allow: []", 1)
	configuration = strings.Replace(configuration, "max_repairs: 1", "max_repairs: 0", 1)
	writeProjectFile(t, path, configuration)
	resolved, err := Load(path, root, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved.Environment.Allow) != 0 || resolved.Policy.MaxRepairs != 0 {
		t.Fatalf("resolved environment=%v repairs=%d", resolved.Environment.Allow, resolved.Policy.MaxRepairs)
	}
}

func validConfigYAML(extra string) string {
	return `schema: agentworkflow.config/v1
source:
  mode: directory-copy
instructions: []
environment:
  allow: [PATH]
forbidden_paths: [.env, .git]
policy:
  assurance: standard
  max_repairs: 1
  blocking_severity: medium
workflow:
  stages:
    - kind: discover
      enabled: true
      prompt: discover prompt
    - kind: plan
      enabled: true
      prompt: plan prompt
      review_prompt: review plan prompt
      revision_prompt: revise plan prompt
    - kind: implement
      enabled: true
      prompt: implement prompt
    - kind: check
      enabled: true
    - kind: review
      enabled: true
      prompt: review prompt
    - kind: repair
      enabled: true
      prompt: repair prompt
    - kind: apply
      enabled: true
      mode: explicit
` + extra + "\n"
}

func writeProjectFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil && !errors.Is(err, os.ErrExist) {
		t.Fatal(err)
	}
}

func slicesContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
