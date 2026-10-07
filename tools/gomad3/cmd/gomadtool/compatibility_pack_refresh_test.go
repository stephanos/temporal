package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"testing"

	compatibility "go.temporal.io/server/tools/gomad3/internal/compatibilitypack"
	"go.temporal.io/server/tools/gomad3/internal/compatibilitypack/authoring"
	"go.temporal.io/server/tools/gomad3/target"
	"go.temporal.io/server/tools/gomad3/upgrade/pinimpact"
	"golang.org/x/mod/module"
)

const refreshDependency = "example.test/refreshdependency"

func refreshSum(version string) string {
	digest := sha256.Sum256([]byte(version))
	return "h1:" + base64.StdEncoding.EncodeToString(digest[:])
}

var refreshRequirement = regexp.MustCompile(`(?m)^require ` + regexp.QuoteMeta(refreshDependency) + ` (\S+)$`)

// fakeRefreshReviewer stands in for the capability review: the reviewed
// dependency version is the one the working directory's go.mod requires.
func fakeRefreshReviewer(reviewed map[string]string) authoring.Reviewer {
	return func(_ context.Context, request authoring.Request, directory string) (target.CapabilityReview, error) {
		reviewed[request.ID] = directory
		contents, err := os.ReadFile(filepath.Join(directory, "go.mod"))
		if err != nil {
			return target.CapabilityReview{}, err
		}
		match := refreshRequirement.FindSubmatch(contents)
		if match == nil {
			return target.CapabilityReview{}, errors.New("module does not require the dependency")
		}
		version := string(match[1])
		digest := sha256.Sum256([]byte(version))
		return target.CapabilityReview{
			Schema: target.CapabilityReviewSchema, BuildTags: []string{},
			Closure: target.CapabilityClosure{
				Schema: target.CapabilityClosureSchema, Compatibility: []target.CompatibilityIdentity{},
				Packages: []target.CapabilityPackage{{
					ImportPath: refreshDependency + "/runtime", Name: "runtime", Imports: []string{"syscall"},
					Module:  &target.CapabilityModule{Path: refreshDependency, Version: version, Sum: refreshSum(version)},
					Sources: []target.CapabilitySource{{Name: "runtime.go", SHA256: fmt.Sprintf("sha256:%x", digest)}}, ForeignSources: []target.CapabilityForeignSource{},
				}, {
					ImportPath: "example.test/refreshtarget", Name: "main", Root: true,
					Module:  &target.CapabilityModule{Path: "example.test/refreshtarget", Main: true},
					Sources: []target.CapabilitySource{}, ForeignSources: []target.CapabilityForeignSource{},
				}},
			},
			Packs: []target.CompatibilityPackEvidence{}, Findings: []target.CapabilityFinding{},
		}, nil
	}
}

func writeRefreshModule(t *testing.T, directory, version string) {
	t.Helper()
	goMod := fmt.Sprintf("module example.test/refreshtarget\n\ngo 1.27.0\n\nrequire %s %s\n", refreshDependency, version)
	goSum := fmt.Sprintf("%s %s %s\n", refreshDependency, version, refreshSum(version))
	for name, contents := range map[string]string{"go.mod": goMod, "go.sum": goSum} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func runGit(t *testing.T, directory string, arguments ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", directory, "-c", "user.name=test", "-c", "user.email=test@example.test"}, arguments...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", arguments, err, output)
	}
}

// TestRunCompatibilityPackRefreshStopsAtApprovalAndResumes bumps a module the
// pack activates on, refreshes through the real pin impact report against the
// Git baseline, approves the printed digest, and reruns. Refresh judges the
// packs of --compatibility-root whatever GOMAD3_COMPATIBILITY_PACKS names.
func TestRunCompatibilityPackRefreshStopsAtApprovalAndResumes(t *testing.T) {
	for _, environment := range []string{"unset", "root packs", "elsewhere", "saved module report", "saved file report", "current output", "unselected output", "not evaluable output"} {
		t.Run(environment, func(t *testing.T) {
			testRunCompatibilityPackRefresh(t, environment)
		})
	}
}

func testRunCompatibilityPackRefresh(t *testing.T, environment string) {
	goCommand, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go command is unavailable")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is unavailable")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	proxy := t.TempDir()
	escaped, err := module.EscapePath(refreshDependency)
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"v1.0.0", "v1.1.0"} {
		directory := filepath.Join(proxy, filepath.FromSlash(escaped), "@v")
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		for name, contents := range map[string]string{
			version + ".mod":  "module " + refreshDependency + "\n\ngo 1.21\n",
			version + ".info": fmt.Sprintf(`{"Version":%q,"Time":"2026-01-01T00:00:00Z"}`, version),
		} {
			if err := os.WriteFile(filepath.Join(directory, name), []byte(contents), 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Setenv("GOPROXY", "file://"+filepath.ToSlash(proxy))
	t.Setenv("GOSUMDB", "off")

	repository := t.TempDir()
	moduleDirectory := filepath.Join(repository, "target")
	packRoot := filepath.Join(repository, "packs-root")
	for _, directory := range []string{moduleDirectory, packRoot} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	writeRefreshModule(t, moduleDirectory, "v1.0.0")
	platform := runtime.GOOS + "/" + runtime.GOARCH
	if environment == "not evaluable output" {
		platform = "darwin/arm64"
		if platform == runtime.GOOS+"/"+runtime.GOARCH {
			platform = "linux/amd64"
		}
	}
	reviewed := map[string]string{}
	reviewer := fakeRefreshReviewer(reviewed)
	draft := authoring.Request{
		Schema: authoring.RequestSchema, ID: "refresh-pack",
		Target:     authoring.Target{Kind: target.KindGoTest, Package: ".", TestArguments: []string{}, BuildTags: []string{}, ExpectedModule: "example.test/refreshtarget"},
		Activation: []authoring.Activation{{Path: refreshDependency}},
		Packages: []authoring.Package{{
			ImportPath: refreshDependency + "/runtime",
			Facts:      []authoring.Fact{{Kind: authoring.FactCapability, Capability: "import:syscall", Directives: []string{}, Disposition: authoring.DispositionAllow}},
		}},
		Owner: "runtime-team", ReviewedAt: "2026-10-03T00:00:00Z", Justification: "Refresh fixture.",
		Workloads: []string{"refresh-fixture"}, Platforms: []string{platform},
	}
	review, err := reviewer(context.Background(), draft, moduleDirectory)
	if err != nil {
		t.Fatal(err)
	}
	discovered, digest, err := authoring.Discover(draft, review)
	if err != nil {
		t.Fatal(err)
	}
	if err := authoring.Generate(packRoot, discovered, digest); err != nil {
		t.Fatal(err)
	}
	table := `{"requests":[{"directory":"../target","request":"refresh-pack"}],"schema":"` + authoring.WorkingDirectoriesSchema + `"}` + "\n"
	if err := os.WriteFile(filepath.Join(packRoot, authoring.WorkingDirectoriesFile), []byte(table), 0o600); err != nil {
		t.Fatal(err)
	}
	switch environment {
	case "unset":
		t.Setenv(compatibility.ExternalPacksEnvironment, "")
	case "root packs":
		t.Setenv(compatibility.ExternalPacksEnvironment, filepath.Join(packRoot, "packs"))
	default:
		t.Setenv(compatibility.ExternalPacksEnvironment, t.TempDir())
	}
	runGit(t, repository, "init", "-q")
	runGit(t, repository, "add", ".")
	runGit(t, repository, "commit", "-q", "-m", "baseline")
	baselineMod, err := os.ReadFile(filepath.Join(moduleDirectory, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	baselineSum, err := os.ReadFile(filepath.Join(moduleDirectory, "go.sum"))
	if err != nil {
		t.Fatal(err)
	}
	writeRefreshModule(t, moduleDirectory, "v1.1.0")
	switch environment {
	case "current output":
		writeRefreshModule(t, moduleDirectory, "v1.0.0")
		contents := fmt.Sprintf("%s v1.0.0 %s\n", refreshDependency, refreshSum("changed sum"))
		if err := os.WriteFile(filepath.Join(moduleDirectory, "go.sum"), []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	case "unselected output":
		if err := os.WriteFile(filepath.Join(moduleDirectory, "go.mod"), []byte("module example.test/refreshtarget\n\ngo 1.27.0\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(moduleDirectory, "go.sum"), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	previous := compatibilityPackReviewer
	compatibilityPackReviewer = func(string) authoring.Reviewer { return reviewer }
	t.Cleanup(func() { compatibilityPackReviewer = previous })
	refresh := []string{"compatibility-pack", "refresh", "--root=" + root, "--compatibility-root=" + packRoot, "--go=" + goCommand}
	if strings.HasSuffix(environment, " output") {
		wantStatus, prefix := 1, "unselected refresh-pack: "
		switch environment {
		case "current output":
			wantStatus, prefix = 0, "current refresh-pack\n"
		case "not evaluable output":
			prefix = "not-evaluable refresh-pack: "
		}
		before := map[string][]byte{}
		for _, relative := range []string{"requests/refresh-pack.json", "reports/refresh-pack.md", "packs/refresh-pack.json", "generation.json"} {
			contents, err := os.ReadFile(filepath.Join(packRoot, relative))
			if err != nil {
				t.Fatal(err)
			}
			before[relative] = contents
		}
		var stdout, stderr bytes.Buffer
		if status := run(refresh, &stdout, &stderr); status != wantStatus || stderr.Len() != 0 || !strings.Contains(stdout.String(), "\n"+prefix) {
			t.Fatalf("refresh status = %d, stdout = %q, stderr = %q", status, &stdout, &stderr)
		}
		output := newRefreshOutput(t, 1)
		status := run(refresh, output, &stderr)
		if !errors.Is(output.err, syscall.EBADF) {
			t.Fatalf("stdout write error = %v, want EBADF", output.err)
		}
		t.Logf("stdout write observed EBADF after 1 successful write")
		if status != 3 || stderr.Len() != 0 || output.String() != strings.SplitAfter(stdout.String(), "\n")[0] {
			t.Fatalf("stdout failure status = %d, stdout = %q, stderr = %q", status, output.String(), &stderr)
		}
		for relative, want := range before {
			contents, err := os.ReadFile(filepath.Join(packRoot, relative))
			if err != nil || !bytes.Equal(contents, want) {
				t.Fatalf("refresh changed %s: %v", relative, err)
			}
		}
		return
	}
	if strings.HasPrefix(environment, "saved ") {
		candidateMod, err := os.ReadFile(filepath.Join(moduleDirectory, "go.mod"))
		if err != nil {
			t.Fatal(err)
		}
		candidateSum, err := os.ReadFile(filepath.Join(moduleDirectory, "go.sum"))
		if err != nil {
			t.Fatal(err)
		}
		digest := func(contents []byte) string { return fmt.Sprintf("sha256:%x", sha256.Sum256(contents)) }
		combined := func(goMod, goSum []byte) string {
			joined := append(append(append([]byte{}, goMod...), 0), goSum...)
			return digest(joined)
		}
		var report string
		if environment == "saved module report" {
			report = fmt.Sprintf(`{"schema":"gomad3.pin-impact/v1","candidate":{"go_mod_sha256":%q,"go_sum_sha256":%q},"baseline":{"go_mod_sha256":%q,"go_sum_sha256":%q},"pins":[]}`,
				digest(candidateMod), digest(candidateSum), digest(baselineMod), digest(baselineSum))
		} else {
			report = fmt.Sprintf(`{"schema":"gomad3.pin-impact/v1","candidate_sha256":%q,"baseline_sha256":%q,"pins":[]}`,
				combined(candidateMod, candidateSum), combined(baselineMod, baselineSum))
		}
		reportPath := filepath.Join(repository, "impact.json")
		if err := os.WriteFile(reportPath, []byte(report), 0o600); err != nil {
			t.Fatal(err)
		}
		var staleStdout, staleStderr bytes.Buffer
		if err := os.WriteFile(reportPath, []byte(strings.Replace(report, digest(candidateMod), "sha256:"+strings.Repeat("0", 64), 1)), 0o600); err != nil {
			t.Fatal(err)
		}
		if environment == "saved file report" {
			stale := strings.Replace(report, combined(candidateMod, candidateSum), "sha256:"+strings.Repeat("0", 64), 1)
			if err := os.WriteFile(reportPath, []byte(stale), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		if status := run(append(refresh, "--impact-report="+reportPath), &staleStdout, &staleStderr); status != 2 {
			t.Fatalf("stale saved report status = %d, want 2; stdout: %s stderr: %s", status, &staleStdout, &staleStderr)
		}
		if err := os.WriteFile(reportPath, []byte(report), 0o600); err != nil {
			t.Fatal(err)
		}
		refresh = append(refresh, "--impact-report="+reportPath)
	}

	var stdout, stderr bytes.Buffer
	if status := run(refresh, &stdout, &stderr); status != 1 {
		t.Fatalf("refresh status = %d\nstdout:\n%s\nstderr:\n%s", status, stdout.String(), stderr.String())
	}
	match := regexp.MustCompile(`(?m)^awaiting-approval refresh-pack (sha256:[0-9a-f]{64}) \(invalidated .*v1\.1\.0.*\)$`).FindStringSubmatch(stdout.String())
	if match == nil || !strings.Contains(stdout.String(), "--approve-review="+match[1]) {
		t.Fatalf("refresh output:\n%s", stdout.String())
	}
	if reviewed["refresh-pack"] != moduleDirectory {
		t.Fatalf("reviewed in %q, want %q", reviewed["refresh-pack"], moduleDirectory)
	}
	contents, err := os.ReadFile(filepath.Join(packRoot, "requests", "refresh-pack.json"))
	if err != nil {
		t.Fatal(err)
	}
	refreshed, err := authoring.DecodeRequest(contents)
	if err != nil || refreshed.ApprovalSHA256 != "" || refreshed.Activation[0].Evidence.Version != "v1.1.0" {
		t.Fatalf("refreshed request = %+v, %v", refreshed, err)
	}
	if environment == "unset" {
		published := map[string][]byte{"requests/refresh-pack.json": contents}
		for _, relative := range []string{"reports/refresh-pack.md", "generation.json", "packs_generated_test.go"} {
			contents, err := os.ReadFile(filepath.Join(packRoot, relative))
			if err != nil {
				t.Fatal(err)
			}
			published[relative] = contents
		}
		for successfulWrites := 0; successfulWrites < 3; successfulWrites++ {
			t.Run(fmt.Sprintf("output after %d writes", successfulWrites), func(t *testing.T) {
				if err := authoring.Generate(packRoot, discovered, digest); err != nil {
					t.Fatal(err)
				}
				output := newRefreshOutput(t, successfulWrites)
				var stderr bytes.Buffer
				status := run(refresh, output, &stderr)
				if !errors.Is(output.err, syscall.EBADF) {
					t.Fatalf("stdout write error = %v, want EBADF", output.err)
				}
				t.Logf("stdout write observed EBADF after %d successful writes", successfulWrites)
				for relative, want := range published {
					contents, err := os.ReadFile(filepath.Join(packRoot, relative))
					if err != nil || !bytes.Equal(contents, want) {
						t.Fatalf("published %s differs after stdout failure: %v", relative, err)
					}
				}
				if _, err := os.Stat(filepath.Join(packRoot, "packs", "refresh-pack.json")); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("unapproved pack remains after stdout failure: %v", err)
				}
				wantPrefix := strings.Join(strings.SplitAfter(stdout.String(), "\n")[:successfulWrites], "")
				if status != 3 || stderr.Len() != 0 || output.String() != wantPrefix {
					t.Fatalf("stdout failure status = %d, stdout = %q, stderr = %q", status, output.String(), &stderr)
				}
			})
		}
	}

	stdout.Reset()
	stderr.Reset()
	if status := run([]string{"compatibility-pack", "generate", "--root=" + root, "--compatibility-root=" + packRoot, "--request=requests/refresh-pack.json", "--approve-review=" + match[1]}, &stdout, &stderr); status != 0 {
		t.Fatalf("approve status = %d: %s", status, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if status := run(refresh, &stdout, &stderr); status != 0 || !strings.Contains(stdout.String(), ": 0 requests selected") {
		t.Fatalf("rerun status = %d\nstdout:\n%s\nstderr:\n%s", status, stdout.String(), stderr.String())
	}

	// A request without a working directory is invalid input.
	if err := os.WriteFile(filepath.Join(packRoot, authoring.WorkingDirectoriesFile), []byte(`{"requests":[],"schema":"`+authoring.WorkingDirectoriesSchema+`"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	stderr.Reset()
	if status := run(refresh, &stdout, &stderr); status != 2 || !strings.Contains(stderr.String(), "refresh-pack") {
		t.Fatalf("unmapped status = %d: %s", status, stderr.String())
	}
}

// TestRunCompatibilityPackCheckRequiresTheRepositoryWorkingDirectoryTable
// deletes the table of the default root, which validate checks, and expects
// check to fail; a downstream --compatibility-root may omit it.
func TestRunCompatibilityPackCheckRequiresTheRepositoryWorkingDirectoryTable(t *testing.T) {
	root := t.TempDir()
	packRoot := filepath.Join(root, "internal", "compatibilitypack")
	moduleDirectory := filepath.Join(root, "target")
	for _, directory := range []string{packRoot, moduleDirectory} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	writeRefreshModule(t, moduleDirectory, "v1.0.0")
	draft := authoring.Request{
		Schema: authoring.RequestSchema, ID: "check-pack",
		Target:     authoring.Target{Kind: target.KindGoTest, Package: ".", TestArguments: []string{}, BuildTags: []string{}, ExpectedModule: "example.test/refreshtarget"},
		Activation: []authoring.Activation{{Path: refreshDependency}},
		Packages: []authoring.Package{{
			ImportPath: refreshDependency + "/runtime",
			Facts:      []authoring.Fact{{Kind: authoring.FactCapability, Capability: "import:syscall", Directives: []string{}, Disposition: authoring.DispositionAllow}},
		}},
		Owner: "runtime-team", ReviewedAt: "2026-10-03T00:00:00Z", Justification: "Check fixture.",
		Workloads: []string{"check-fixture"}, Platforms: []string{runtime.GOOS + "/" + runtime.GOARCH},
	}
	review, err := fakeRefreshReviewer(map[string]string{})(context.Background(), draft, moduleDirectory)
	if err != nil {
		t.Fatal(err)
	}
	discovered, digest, err := authoring.Discover(draft, review)
	if err != nil {
		t.Fatal(err)
	}
	if err := authoring.Generate(packRoot, discovered, digest); err != nil {
		t.Fatal(err)
	}
	check := func(arguments ...string) (int, string) {
		var stdout, stderr bytes.Buffer
		status := run(append([]string{"compatibility-pack", "check", "--root=" + root}, arguments...), &stdout, &stderr)
		return status, stderr.String()
	}
	if status, stderr := check(); status != 1 || !strings.Contains(stderr, authoring.WorkingDirectoriesFile) {
		t.Fatalf("check without the table = %d: %s", status, stderr)
	}
	if status, stderr := check("--compatibility-root=" + packRoot); status != 0 {
		t.Fatalf("check of a downstream root without the table = %d: %s", status, stderr)
	}
	table := `{"requests":[{"directory":"../../target","request":"check-pack"}],"schema":"` + authoring.WorkingDirectoriesSchema + `"}` + "\n"
	if err := os.WriteFile(filepath.Join(packRoot, authoring.WorkingDirectoriesFile), []byte(table), 0o600); err != nil {
		t.Fatal(err)
	}
	if status, stderr := check(); status != 0 {
		t.Fatalf("check with the table = %d: %s", status, stderr)
	}
}

func TestCompatibilityPackRefreshAcceptsSavedPinImpactSchemas(t *testing.T) {
	for _, test := range []struct {
		name, class, pack, id string
	}{
		{name: "module report", class: "pack-rule", pack: "reviewed-pack", id: "reviewed-pack:example.com/pkg"},
		{name: "file report", class: "pack_rule", id: "reviewed-pack:example.com/pkg"},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "impact.json")
			identity := `,"candidate":{"go_mod_sha256":"sha256:mod-candidate","go_sum_sha256":"sha256:sum-candidate"},"baseline":{"go_mod_sha256":"sha256:mod-baseline","go_sum_sha256":"sha256:sum-baseline"}`
			if test.name == "file report" {
				identity = `,"candidate_sha256":"sha256:candidate","baseline_sha256":"sha256:baseline"`
			}
			contents := `{"schema":"gomad3.pin-impact/v1"` + identity + `,"pins":[{"class":"` + test.class + `","pack":"` + test.pack + `","id":"` + test.id + `","status":"invalidated","reason":"module changed"}]}`
			if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
				t.Fatal(err)
			}
			impact, err := readPackImpactReport(path, map[string]string{"reviewed-pack": t.TempDir()})
			if err != nil || impact.invalidated["reviewed-pack"] == "" {
				t.Fatalf("impact = %#v, error = %v", impact, err)
			}
		})
	}
}

func TestCompatibilityPackRefreshInfrastructureErrorsUseStatusThree(t *testing.T) {
	if status := compatibilityPackRefreshStatus(errors.New("working-directory read failed")); status != 3 {
		t.Fatalf("infrastructure status = %d, want 3", status)
	}
}

func TestSavedReportCannotSuppressAnotherMappedModule(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	const firstCandidate = "sha256:first-candidate"
	const firstBaseline = "sha256:first-baseline"
	live := packImpact{
		invalidated: map[string]string{"second-pack": "invalidated by live module evaluation"},
		identities: map[string]packModuleIdentity{
			first:  {candidateCombined: firstCandidate, baselineCombined: firstBaseline},
			second: {candidateCombined: "sha256:second-candidate", baselineCombined: "sha256:second-baseline"},
		},
	}
	saved := packImpact{
		invalidated:   map[string]string{"first-pack": "invalidated by saved report"},
		savedIdentity: packModuleIdentity{candidateCombined: firstCandidate, baselineCombined: firstBaseline},
		reportedIDs:   map[string]bool{"first-pack": true},
	}
	directories := map[string]string{"first-pack": first, "second-pack": second}
	merged, err := mergeSavedPackImpact(live, saved, directories)
	if err != nil || merged.invalidated["second-pack"] != "invalidated by live module evaluation" || merged.invalidated["first-pack"] != "invalidated by saved report" {
		t.Fatalf("mapped module impacts were not merged: impact = %#v, error = %v", merged, err)
	}
	saved.reportedIDs["second-pack"] = true
	if _, err := mergeSavedPackImpact(live, saved, directories); !pinimpact.IsInputError(err) {
		t.Fatalf("report for another mapped module error = %v, want invalid input", err)
	}
	delete(saved.reportedIDs, "second-pack")
	saved.savedIdentity.candidateCombined = "sha256:stale-candidate"
	if _, err := mergeSavedPackImpact(live, saved, directories); !pinimpact.IsInputError(err) {
		t.Fatalf("stale report error = %v, want invalid input", err)
	}
}
