package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
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
	"golang.org/x/mod/sumdb/dirhash"
)

type resolvedRefreshFixture struct {
	t           *testing.T
	root        string
	packRoot    string
	goCommand   string
	goRoot      string
	directories map[string]string
	reviewed    map[string]target.CapabilityReview
	approvals   map[string]string
}

func newResolvedRefreshFixture(t *testing.T) *resolvedRefreshFixture {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	goCommand, goRoot := refreshSourceGo(t)
	fixture := &resolvedRefreshFixture{t: t, root: root, packRoot: filepath.Join(t.TempDir(), "packs"), goCommand: goCommand, goRoot: goRoot, directories: map[string]string{}, reviewed: map[string]target.CapabilityReview{}, approvals: map[string]string{}}
	proxy := t.TempDir()
	proxyVersions := filepath.Join(proxy, refreshDependency, "@v")
	if err := os.MkdirAll(proxyVersions, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"v1.0.0", "v1.1.0", "v1.2.0", "v1.3.0"} {
		goMod := "module " + refreshDependency + "\n\ngo 1.21\n"
		source := "package runtime\n\nimport \"syscall\"\n\nvar Err = syscall.EINVAL\nconst Version = " + fmt.Sprintf("%q", version) + "\n"
		if version == "v1.3.0" {
			source = "package runtime\n\nconst Version = \"v1.3.0\"\n"
		}
		var archive bytes.Buffer
		writer := zip.NewWriter(&archive)
		for _, file := range []struct{ name, contents string }{{"go.mod", goMod}, {"runtime/runtime.go", source}} {
			entry, err := writer.Create(refreshDependency + "@" + version + "/" + file.name)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := entry.Write([]byte(file.contents)); err != nil {
				t.Fatal(err)
			}
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		for name, contents := range map[string][]byte{version + ".mod": []byte(goMod), version + ".info": fmt.Appendf(nil, `{"Version":%q,"Time":"2026-01-01T00:00:00Z"}`, version), version + ".zip": archive.Bytes()} {
			if err := os.WriteFile(filepath.Join(proxyVersions, name), contents, 0o600); err != nil {
				t.Fatal(err)
			}
		}
		if sum, err := dirhash.HashZip(filepath.Join(proxyVersions, version+".zip"), dirhash.Hash1); err != nil {
			t.Fatal(err)
		} else {
			t.Logf("proxy %s@%s zip %s", refreshDependency, version, sum)
		}
	}
	t.Setenv("GOPROXY", "file://"+filepath.ToSlash(proxy))
	t.Setenv("GOSUMDB", "off")
	t.Setenv("GOMODCACHE", t.TempDir())
	t.Setenv(compatibility.ExternalPacksEnvironment, "")
	platform := runtime.GOOS + "/" + runtime.GOARCH
	for _, id := range []string{"pack-a", "pack-b"} {
		directory := filepath.Join(filepath.Dir(fixture.packRoot), id)
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		fixture.directories[id] = directory
		fixture.candidate(id, "v1.0.0")
		draft := authoring.Request{
			Schema: authoring.RequestSchema, ID: id,
			Target:     authoring.Target{Kind: target.KindGoRun, Package: ".", ExpectedModule: "example.test/" + id, TestArguments: []string{}, BuildTags: []string{"test_dep"}},
			Activation: []authoring.Activation{{Path: refreshDependency}},
			Packages:   []authoring.Package{{ImportPath: refreshDependency + "/runtime", Facts: []authoring.Fact{{Kind: authoring.FactCapability, Capability: "import:syscall", Disposition: authoring.DispositionAllow, Directives: []string{}}}}},
			Owner:      "runtime-team", ReviewedAt: "2026-10-08T00:00:00Z", Justification: "Resolved refresh source fixture.", Workloads: []string{"refresh-fixture"}, Platforms: []string{platform},
		}
		review, err := fixture.review(t.Context(), draft, directory)
		if err != nil {
			t.Fatal(err)
		}
		request, digest, err := authoring.Discover(draft, review)
		if err != nil {
			t.Fatal(err)
		}
		if err := authoring.Generate(fixture.packRoot, request, digest); err != nil {
			t.Fatal(err)
		}
		fixture.approvals[id] = digest
	}
	fixture.table(`{"directory":"../pack-a","request":"pack-a"},{"directory":"../pack-b","request":"pack-b"}`)
	runGit(t, filepath.Dir(fixture.packRoot), "init", "-q")
	runGit(t, filepath.Dir(fixture.packRoot), "add", ".")
	runGit(t, filepath.Dir(fixture.packRoot), "commit", "-q", "-m", "baseline")
	previous := compatibilityPackReviewer
	compatibilityPackReviewer = func(string) authoring.Reviewer { return fixture.review }
	t.Cleanup(func() { compatibilityPackReviewer = previous })
	fixture.reviewed = map[string]target.CapabilityReview{}
	return fixture
}

func (fixture *resolvedRefreshFixture) candidate(id, version string) {
	fixture.t.Helper()
	directory := fixture.directories[id]
	for name, contents := range map[string]string{
		"go.mod":  fmt.Sprintf("module example.test/%s\n\ngo 1.27.1\n\nrequire %s %s\n", id, refreshDependency, version),
		"main.go": "package main\n\nimport dependency \"" + refreshDependency + "/runtime\"\n\nfunc main() { println(dependency.Version) }\n",
	} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(contents), 0o600); err != nil {
			fixture.t.Fatal(err)
		}
	}
	command := exec.CommandContext(fixture.t.Context(), fixture.goCommand, "mod", "download", "-modcacherw", refreshDependency+"@"+version)
	command.Dir = directory
	if output, err := command.CombinedOutput(); err != nil {
		fixture.t.Fatalf("resolve %s@%s: %v\n%s", id, version, err, output)
	}
}

func (fixture *resolvedRefreshFixture) review(ctx context.Context, request authoring.Request, directory string) (target.CapabilityReview, error) {
	spec := request.ReviewSpec(directory, fixture.goRoot)
	review, err := target.ReviewCapabilities(ctx, spec)
	if err == nil {
		fixture.reviewed[request.ID] = review
	}
	return review, err
}

func (fixture *resolvedRefreshFixture) table(entries string) {
	fixture.t.Helper()
	contents := `{"requests":[` + entries + `],"schema":"` + authoring.WorkingDirectoriesSchema + `"}` + "\n"
	if err := os.WriteFile(filepath.Join(fixture.packRoot, authoring.WorkingDirectoriesFile), []byte(contents), 0o600); err != nil {
		fixture.t.Fatal(err)
	}
}

func (fixture *resolvedRefreshFixture) request(id string) authoring.Request {
	fixture.t.Helper()
	contents, err := os.ReadFile(filepath.Join(fixture.packRoot, "requests", id+".json"))
	if err != nil {
		fixture.t.Fatal(err)
	}
	request, err := authoring.DecodeRequest(contents)
	if err != nil {
		fixture.t.Fatal(err)
	}
	return request
}

func (fixture *resolvedRefreshFixture) files() map[string]string {
	fixture.t.Helper()
	files := map[string]string{}
	if err := filepath.WalkDir(fixture.packRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		contents, err := os.ReadFile(path)
		files[path] = string(contents)
		return err
	}); err != nil {
		fixture.t.Fatal(err)
	}
	return files
}

func (fixture *resolvedRefreshFixture) refresh(extra ...string) string {
	fixture.t.Helper()
	arguments := append([]string{"compatibility-pack", "refresh", "--root=" + fixture.root, "--compatibility-root=" + fixture.packRoot, "--go=" + fixture.goCommand}, extra...)
	var stdout, stderr bytes.Buffer
	if status := run(arguments, &stdout, &stderr); status != 1 || stderr.Len() != 0 {
		fixture.t.Fatalf("refresh status = %d, stdout = %s, stderr = %s", status, &stdout, &stderr)
	}
	return stdout.String()
}

func TestRunCompatibilityPackRefreshResolvesTwoModulesAndKeepsPartialApproval(t *testing.T) {
	fixture := newResolvedRefreshFixture(t)
	fixture.candidate("pack-a", "v1.1.0")
	fixture.candidate("pack-b", "v1.2.0")
	for name, entries := range map[string]string{
		"unmapped":       `{"directory":"../pack-a","request":"pack-a"}`,
		"absolute":       `{"directory":"/pack-a","request":"pack-a"},{"directory":"../pack-b","request":"pack-b"}`,
		"missing module": `{"directory":"../missing","request":"pack-a"},{"directory":"../pack-b","request":"pack-b"}`,
	} {
		t.Run(name, func(t *testing.T) {
			fixture.table(entries)
			before := fixture.files()
			var stdout, stderr bytes.Buffer
			status := run([]string{"compatibility-pack", "refresh", "--root=" + fixture.root, "--compatibility-root=" + fixture.packRoot, "--go=" + fixture.goCommand}, &stdout, &stderr)
			if status != 2 || stdout.Len() != 0 || stderr.Len() == 0 || !maps.Equal(before, fixture.files()) || len(fixture.reviewed) != 0 {
				t.Fatalf("invalid mapping status = %d, stdout = %s, stderr = %s, reviewed = %v", status, &stdout, &stderr, fixture.reviewed)
			}
		})
	}
	fixture.table(`{"directory":"../pack-a","request":"pack-a"},{"directory":"../pack-b","request":"pack-b"}`)
	output := fixture.refresh()
	digests := map[string]string{}
	for id, version := range map[string]string{"pack-a": "v1.1.0", "pack-b": "v1.2.0"} {
		matches := regexp.MustCompile(`(?m)^awaiting-approval ` + id + ` (sha256:[0-9a-f]{64}) \(invalidated .*` + regexp.QuoteMeta(version) + `.*\)$`).FindStringSubmatch(output)
		if len(matches) != 2 {
			t.Fatalf("missing resolved %s candidate %s in %s", id, version, output)
		}
		digests[id] = matches[1]
		request := fixture.request(id)
		if request.ApprovalSHA256 != "" || request.Activation[0].Evidence.Version != version || request.Packages[0].Evidence.Module.Version != version {
			t.Fatalf("candidate %s request = %+v", id, request)
		}
		var resolved bool
		for _, pkg := range fixture.reviewed[id].Closure.Packages {
			if pkg.ImportPath == refreshDependency+"/runtime" && pkg.Module.Version == version && pkg.Module.Sum == request.Activation[0].Evidence.Sum && len(pkg.Sources) == 1 && pkg.Sources[0].SHA256 == request.Packages[0].Evidence.GoSources[0].SHA256 {
				resolved = true
			}
		}
		if !resolved {
			t.Fatalf("stock closure did not resolve %s in %s", version, fixture.directories[id])
		}
		t.Logf("stock Go %s command %s root %s reviewed %s in %s; actual candidate %s@%s sum %s source %s fresh approval %s", runtime.Version(), fixture.goCommand, fixture.goRoot, id, fixture.directories[id], refreshDependency, version, request.Activation[0].Evidence.Sum, request.Packages[0].Evidence.GoSources[0].SHA256, digests[id])
	}
	before := fixture.files()
	var stdout, stderr bytes.Buffer
	generate := []string{"compatibility-pack", "generate", "--root=" + fixture.root, "--compatibility-root=" + fixture.packRoot, "--request=requests/pack-a.json"}
	if status := run(append(generate, "--approve-review="+fixture.approvals["pack-a"]), &stdout, &stderr); status != 1 || !maps.Equal(before, fixture.files()) {
		t.Fatalf("older approval status = %d, stdout = %s, stderr = %s", status, &stdout, &stderr)
	}
	stdout.Reset()
	stderr.Reset()
	if status := run(append(generate, "--approve-review="+digests["pack-a"]), &stdout, &stderr); status != 0 {
		t.Fatalf("approve fresh request status = %d, stderr = %s", status, &stderr)
	}
	approved := fixture.files()
	fixture.reviewed = map[string]target.CapabilityReview{}
	output = fixture.refresh()
	if !strings.Contains(output, ": 1 requests selected\n") || !strings.Contains(output, "awaiting-approval pack-b "+digests["pack-b"]) || len(fixture.reviewed) != 1 || fixture.reviewed["pack-b"].Schema != target.CapabilityReviewSchema || fixture.request("pack-a").ApprovalSHA256 != digests["pack-a"] || !maps.Equal(approved, fixture.files()) {
		t.Fatalf("partial-approval rerun = %s", output)
	}
	if err := authoring.Check(fixture.packRoot); err != nil {
		t.Fatal(err)
	}
}

func refreshSourceGo(t *testing.T) (string, string) {
	t.Helper()
	command, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	output, err := exec.CommandContext(t.Context(), command, "env", "GOROOT").CombinedOutput()
	if err != nil {
		t.Fatalf("resolve stock Go root: %v\n%s", err, output)
	}
	root := strings.TrimSpace(string(output))
	if !filepath.IsAbs(root) {
		t.Fatalf("stock Go root is not absolute: %q", root)
	}
	return command, root
}

func TestRunCompatibilityPackRefreshContinuesAfterActualDiscoveryFailure(t *testing.T) {
	fixture := newResolvedRefreshFixture(t)
	fixture.candidate("pack-a", "v1.3.0")
	fixture.candidate("pack-b", "v1.2.0")
	before := fixture.request("pack-a")
	artifacts := fixture.files()
	output := fixture.refresh()
	if !strings.Contains(output, "failed pack-a: discover compatibility-pack facts for "+refreshDependency+"/runtime: requested capability import:syscall is absent") || !strings.Contains(output, "awaiting-approval pack-b") || fixture.request("pack-a").ApprovalSHA256 != before.ApprovalSHA256 || fixture.request("pack-b").ApprovalSHA256 != "" {
		t.Fatalf("discovery failure/continuation = %s", output)
	}
	encodedBefore, err := json.Marshal(before)
	if err != nil {
		t.Fatal(err)
	}
	encodedAfter, err := json.Marshal(fixture.request("pack-a"))
	if err != nil || !bytes.Equal(encodedBefore, encodedAfter) {
		t.Fatalf("failed request changed: %v", err)
	}
	after := fixture.files()
	for _, relative := range []string{"requests/pack-a.json", "reports/pack-a.md", "packs/pack-a.json"} {
		path := filepath.Join(fixture.packRoot, relative)
		if artifacts[path] == "" || artifacts[path] != after[path] {
			t.Fatalf("failed request artifact changed: %s", relative)
		}
	}
	if len(fixture.reviewed) != 2 {
		t.Fatalf("both actual stock closures must execute, got %v", fixture.reviewed)
	}
	t.Logf("actual %s discovery lacked syscall and failed; %s continued to fresh approval", fixture.request("pack-a").ID, fixture.request("pack-b").ID)
}
