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
	"testing"

	compatibility "go.temporal.io/server/tools/gomad3/internal/compatibilitypack"
	"go.temporal.io/server/tools/gomad3/internal/compatibilitypack/authoring"
	"go.temporal.io/server/tools/gomad3/target"
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
// Git baseline, approves the printed digest, and reruns.
func TestRunCompatibilityPackRefreshStopsAtApprovalAndResumes(t *testing.T) {
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
	t.Setenv(compatibility.ExternalPacksEnvironment, filepath.Join(packRoot, "packs"))
	runGit(t, repository, "init", "-q")
	runGit(t, repository, "add", ".")
	runGit(t, repository, "commit", "-q", "-m", "baseline")
	writeRefreshModule(t, moduleDirectory, "v1.1.0")

	previous := compatibilityPackReviewer
	compatibilityPackReviewer = func(string) authoring.Reviewer { return reviewer }
	t.Cleanup(func() { compatibilityPackReviewer = previous })
	refresh := []string{"compatibility-pack", "refresh", "--root=" + root, "--compatibility-root=" + packRoot, "--go=" + goCommand}

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
