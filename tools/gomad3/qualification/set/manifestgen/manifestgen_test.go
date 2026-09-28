package manifestgen

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/qualification/set"
)

const (
	fixtureSpec   = "spec.json"
	fixtureOutput = "tests.json"
)

// fixtureFiles cover the go command's selection rules: an external test
// package, a tag-excluded file, a non-test file, TestMain, a lower-case
// suffix, a method, and a helper.
var fixtureFiles = map[string]string{
	"go.mod": "module example.com/fixture\n\ngo 1.27\n",
	"pkg.go": "package fixture\n\nimport \"testing\"\n\nfunc TestNotInATestFile(t *testing.T) {}\n",
	"a_test.go": `package fixture

import "testing"

type suite struct{}

func (suite) TestMethod(t *testing.T) {}

func TestMain(m *testing.M) { m.Run() }

func TestAlphaSuite(t *testing.T) {}

func Testlowercase(t *testing.T) {}

func helper(t *testing.T) {}

func BenchmarkAlpha(b *testing.B) {}
`,
	"b_test.go": `package fixture_test

import testpkg "testing"

func TestBeta_Parts(t *testpkg.T) {}

func TestNDCGamma(t *testpkg.T) {}
`,
	"excluded_test.go": "//go:build !gomad\n\npackage fixture\n\nimport \"testing\"\n\nfunc TestExcludedByTag(t *testing.T) {}\n",
}

func writeFixture(t *testing.T, spec Spec) string {
	t.Helper()
	root := t.TempDir()
	for name, contents := range fixtureFiles {
		writeFile(t, filepath.Join(root, "pkg", name), contents)
	}
	writeSpec(t, root, spec)
	return root
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeSpec(t *testing.T, root string, spec Spec) {
	t.Helper()
	contents, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, fixtureSpec), string(contents))
}

func baseSpec() Spec {
	return Spec{
		Schema: SpecSchema, Package: "./pkg", BuildTags: []string{"gomad", "test_dep"},
		Platforms: []string{"darwin/arm64", "linux/amd64"}, IDPrefix: "fixture",
		Manifest: ManifestDefaults{
			Name: "fixture-tests", Description: "fixture package tests", Module: "example.com/fixture",
			Seeds: []uint64{11, 17}, Repeat: 2, RunTimeout: "2m", OverallTimeout: "5m", TerminateGrace: "2s",
			OutputBytes: 8 << 20, WorldTransitionBytes: 64 << 20,
		},
		Workload: WorkloadDefaults{
			Tier: 3, CapabilityMode: "closure", Invariant: "passes under virtual time",
			ReadOnlyMounts: []set.Mount{{Source: "./schema", Target: "/example.com/fixture/schema"}},
			ChoiceBytes:    64 << 20, ReplaySuccesses: true, SuccessArtifactLimit: 1, SuccessBytesLimit: 1 << 30,
			OverallTimeout: "20m", Expectation: set.WorkloadExpectation{Classification: "qualified"},
		},
	}
}

func run(root string, check bool) error {
	return Run(Config{Root: root, Spec: fixtureSpec, Output: fixtureOutput, Check: check})
}

func loadGenerated(t *testing.T, root string) set.Manifest {
	t.Helper()
	manifest, err := set.LoadManifest(filepath.Join(root, fixtureOutput))
	if err != nil {
		t.Fatal(err)
	}
	return manifest
}

func workloadFor(manifest set.Manifest, test string) (set.Workload, bool) {
	index := slices.IndexFunc(manifest.Suites, func(workload set.Workload) bool { return workload.Test == test })
	if index < 0 {
		return set.Workload{}, false
	}
	return manifest.Suites[index], true
}

func TestRunGeneratesOneDefaultWorkloadPerListedTest(t *testing.T) {
	root := writeFixture(t, baseSpec())
	if err := run(root, false); err != nil {
		t.Fatal(err)
	}
	manifest := loadGenerated(t, root)
	var tests []string
	for _, workload := range manifest.Suites {
		tests = append(tests, workload.Test)
	}
	if want := []string{"TestAlphaSuite", "TestBeta_Parts", "TestNDCGamma"}; !slices.Equal(tests, want) {
		t.Fatalf("tests = %v, want %v", tests, want)
	}
	alpha, _ := workloadFor(manifest, "TestAlphaSuite")
	want := set.Workload{
		ID: "fixture-alpha-suite", Name: "TestAlphaSuite", Tier: 3, Invariant: "TestAlphaSuite passes under virtual time",
		Package: "./pkg", Test: "TestAlphaSuite", BuildTags: []string{"gomad", "test_dep"}, CapabilityMode: "closure",
		ReadOnlyMounts: []set.Mount{{Source: "./schema", Target: "/example.com/fixture/schema"}},
		ChoiceBytes:    64 << 20, ReplaySuccesses: true, SuccessArtifactLimit: 1, SuccessBytesLimit: 1 << 30,
		OverallTimeout: "20m", Expectation: set.WorkloadExpectation{Classification: "qualified"},
	}
	gotJSON, _ := json.Marshal(alpha)
	wantJSON, _ := json.Marshal(want)
	if string(gotJSON) != string(wantJSON) {
		t.Fatalf("default workload:\n got %s\nwant %s", gotJSON, wantJSON)
	}
	if err := run(root, true); err != nil {
		t.Fatalf("freshly generated manifest is stale: %v", err)
	}
}

// TestListTestsMatchesGoTestList pins the parser to the go command it stands
// in for.
func TestListTestsMatchesGoTestList(t *testing.T) {
	root := writeFixture(t, baseSpec())
	command := exec.Command("go", "test", "-vet=off", "-tags", "gomad,test_dep", "-list", ".*", ".")
	command.Dir = filepath.Join(root, "pkg")
	command.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod", "GOTOOLCHAIN=local", "CGO_ENABLED=0")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go test -list: %v\n%s", err, output)
	}
	var listed []string
	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "Test") {
			listed = append(listed, line)
		}
	}
	slices.Sort(listed)
	parsed, err := ListTests(filepath.Join(root, "pkg"), []string{"gomad", "test_dep"}, []string{"darwin/arm64", "linux/amd64"})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(parsed, listed) {
		t.Fatalf("parsed %v, go test -list %v", parsed, listed)
	}
}

func TestCheckFailsUntilAddedOrRemovedTestsAreRegenerated(t *testing.T) {
	for name, change := range map[string]struct {
		mutate  func(root string)
		stale   string
		present string
		absent  string
	}{
		"added test": {
			mutate: func(root string) {
				writeFile(t, filepath.Join(root, "pkg", "new_test.go"), "package fixture\n\nimport \"testing\"\n\nfunc TestDelta(t *testing.T) {}\n")
			},
			stale:   "missing tests TestDelta",
			present: "TestDelta",
		},
		"removed test": {
			mutate: func(root string) {
				if err := os.Remove(filepath.Join(root, "pkg", "b_test.go")); err != nil {
					t.Fatal(err)
				}
			},
			stale:  "removed tests TestBeta_Parts, TestNDCGamma",
			absent: "TestNDCGamma",
		},
	} {
		t.Run(name, func(t *testing.T) {
			root := writeFixture(t, baseSpec())
			if err := run(root, false); err != nil {
				t.Fatal(err)
			}
			change.mutate(root)
			if err := run(root, true); err == nil || !strings.Contains(err.Error(), "is stale") || !strings.Contains(err.Error(), change.stale) {
				t.Fatalf("check error = %v, want stale with %q", err, change.stale)
			}
			if err := run(root, false); err != nil {
				t.Fatal(err)
			}
			manifest := loadGenerated(t, root)
			if change.present != "" {
				workload, found := workloadFor(manifest, change.present)
				if !found || workload.Expectation != (set.WorkloadExpectation{Classification: "qualified"}) || workload.PlatformExpectations != nil {
					t.Fatalf("new test %s = %+v (found %v), want default qualified", change.present, workload, found)
				}
			}
			if _, found := workloadFor(manifest, change.absent); change.absent != "" && found {
				t.Fatalf("removed test %s is still generated", change.absent)
			}
		})
	}
}

func TestSpecOverridesAndExclusionsApplyByName(t *testing.T) {
	spec := baseSpec()
	root := writeFixture(t, spec)
	if err := run(root, false); err != nil {
		t.Fatal(err)
	}
	intermittent := set.WorkloadExpectation{Classification: "intermittent", Finding: "GOMAD_MILESTONES.md#f7"}
	spec.Tests = map[string]TestOverride{"TestAlphaSuite": {
		RequiredProbes:       []string{"stdlib.os.openfile"},
		Expectation:          &intermittent,
		PlatformExpectations: map[string]set.WorkloadExpectation{"darwin/arm64": {Classification: "qualified"}},
	}}
	spec.Exclusions = map[string]Exclusion{"TestBeta_Parts": {Owner: "stephanos", Date: "2026-09-28", Reason: "needs a real network"}}
	writeSpec(t, root, spec)
	if err := run(root, true); err == nil || !strings.Contains(err.Error(), "is stale") {
		t.Fatalf("check after a spec change = %v, want stale", err)
	}
	if err := run(root, false); err != nil {
		t.Fatal(err)
	}
	manifest := loadGenerated(t, root)
	alpha, _ := workloadFor(manifest, "TestAlphaSuite")
	if !slices.Equal(alpha.RequiredProbes, []string{"stdlib.os.openfile"}) || alpha.Expectation != intermittent || alpha.PlatformExpectations["darwin/arm64"].Classification != "qualified" {
		t.Fatalf("overridden workload = %+v", alpha)
	}
	if _, found := workloadFor(manifest, "TestBeta_Parts"); found {
		t.Fatal("excluded test is generated")
	}
	if gamma, _ := workloadFor(manifest, "TestNDCGamma"); gamma.ID != "fixture-ndc-gamma" || gamma.Expectation.Classification != "qualified" {
		t.Fatalf("untouched workload = %+v", gamma)
	}
}

func TestInvalidSpecsAreRefused(t *testing.T) {
	for name, testCase := range map[string]struct {
		mutate func(*Spec)
		want   string
	}{
		"exclusion without owner": {
			mutate: func(spec *Spec) {
				spec.Exclusions = map[string]Exclusion{"TestAlphaSuite": {Date: "2026-09-28", Reason: "flaky"}}
			},
			want: "requires an owner",
		},
		"exclusion without date": {
			mutate: func(spec *Spec) {
				spec.Exclusions = map[string]Exclusion{"TestAlphaSuite": {Owner: "stephanos", Reason: "flaky"}}
			},
			want: "requires a YYYY-MM-DD date",
		},
		"exclusion without reason": {
			mutate: func(spec *Spec) {
				spec.Exclusions = map[string]Exclusion{"TestAlphaSuite": {Owner: "stephanos", Date: "2026-09-28"}}
			},
			want: "requires a reason",
		},
		"excluded and overridden": {
			mutate: func(spec *Spec) {
				spec.Exclusions = map[string]Exclusion{"TestAlphaSuite": {Owner: "stephanos", Date: "2026-09-28", Reason: "flaky"}}
				spec.Tests = map[string]TestOverride{"TestAlphaSuite": {RequiredProbes: []string{"stdlib.os.openfile"}}}
			},
			want: "both excluded and overridden",
		},
		"override of an undeclared test": {
			mutate: func(spec *Spec) {
				spec.Tests = map[string]TestOverride{"TestGone": {RequiredProbes: []string{"stdlib.os.openfile"}}}
			},
			want: "override names TestGone",
		},
		"exclusion of an undeclared test": {
			mutate: func(spec *Spec) {
				spec.Exclusions = map[string]Exclusion{"TestGone": {Owner: "stephanos", Date: "2026-09-28", Reason: "flaky"}}
			},
			want: "exclusion names TestGone",
		},
		"override expectation without finding": {
			mutate: func(spec *Spec) {
				spec.Tests = map[string]TestOverride{"TestAlphaSuite": {Expectation: &set.WorkloadExpectation{Classification: "intermittent"}}}
			},
			want: "requires a finding",
		},
	} {
		t.Run(name, func(t *testing.T) {
			spec := baseSpec()
			testCase.mutate(&spec)
			root := writeFixture(t, spec)
			if err := run(root, false); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error = %v, want %q", err, testCase.want)
			}
			if _, err := os.Stat(filepath.Join(root, fixtureOutput)); !os.IsNotExist(err) {
				t.Fatalf("refused spec wrote a manifest: %v", err)
			}
		})
	}
}

func TestPackagesThatCannotYieldStableWorkloadsAreRefused(t *testing.T) {
	for name, testCase := range map[string]struct {
		file, contents, want string
	}{
		"platform-specific test": {
			file: "only_linux_test.go", contents: "package fixture\n\nimport \"testing\"\n\nfunc TestLinuxOnly(t *testing.T) {}\n",
			want: "lists different tests on linux/amd64",
		},
		"colliding identities": {
			file: "collide_test.go", contents: "package fixture\n\nimport \"testing\"\n\nfunc TestAlpha_Suite(t *testing.T) {}\n",
			want: "share workload identity fixture-alpha-suite",
		},
		"wrong test signature": {
			file: "signature_test.go", contents: "package fixture\n\nimport \"testing\"\n\nfunc TestTwo(t *testing.T, n int) {}\n",
			want: "wrong signature",
		},
	} {
		t.Run(name, func(t *testing.T) {
			root := writeFixture(t, baseSpec())
			writeFile(t, filepath.Join(root, "pkg", testCase.file), testCase.contents)
			if err := run(root, false); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error = %v, want %q", err, testCase.want)
			}
		})
	}
}

func TestWorkloadID(t *testing.T) {
	for test, want := range map[string]string{
		"TestActivityTestSuite":            "tests-activity-test-suite",
		"TestNDCFuncTestSuite":             "tests-ndc-func-test-suite",
		"TestSignalWorkflowTestSuiteChasm": "tests-signal-workflow-test-suite-chasm",
		"TestVersioning3FunctionalSuite":   "tests-versioning3-functional-suite",
		"TestPartitionScalingUpFromDC":     "tests-partition-scaling-up-from-dc",
		"TestSchedule_V1":                  "tests-schedule-v1",
		"TestHTTPAPITestSuite":             "tests-httpapi-test-suite",
	} {
		if got := WorkloadID("tests", test); got != want {
			t.Errorf("WorkloadID(%s) = %s, want %s", test, got, want)
		}
	}
}

// TestCheckedInTestsManifestIsCurrent fails when ./tests or its generator
// spec changed without regenerating the manifest.
func TestCheckedInTestsManifestIsCurrent(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	err = Run(Config{
		Root: root, Spec: "tools/gomad3integration/qualification/tests.generator.json",
		Output: "tools/gomad3integration/qualification/tests.json", Check: true,
	})
	if err != nil {
		t.Fatal(err)
	}
}
