package authoring

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/compatibilitypack"
	"go.temporal.io/server/tools/gomad3/target"
)

const refreshDependency = "example.com/dependency"

// refreshFixture is a pack authoring root whose requests live in separate
// target modules. Its reviewer reads the version each module's go.mod
// requires, standing in for the capability review the candidate checkout
// resolves, and records which directory each request was reviewed in.
type refreshFixture struct {
	t        *testing.T
	root     string
	reviewed map[string]string
}

func newRefreshFixture(t *testing.T) *refreshFixture {
	t.Helper()
	return &refreshFixture{t: t, root: t.TempDir(), reviewed: map[string]string{}}
}

func refreshSum(version string) string {
	digest := sha256.Sum256([]byte(version))
	return "h1:" + base64.StdEncoding.EncodeToString(digest[:])
}

// module writes a target module requiring the dependency at version.
func (fixture *refreshFixture) module(directory, path, version string) {
	fixture.t.Helper()
	absolute := filepath.Join(fixture.root, filepath.FromSlash(directory))
	if err := os.MkdirAll(absolute, 0o700); err != nil {
		fixture.t.Fatal(err)
	}
	contents := fmt.Sprintf("module %s\n\ngo 1.27.0\n\nrequire %s %s\n", path, refreshDependency, version)
	if err := os.WriteFile(filepath.Join(absolute, "go.mod"), []byte(contents), 0o600); err != nil {
		fixture.t.Fatal(err)
	}
}

func (fixture *refreshFixture) table(entries ...string) {
	fixture.t.Helper()
	var encoded []string
	for _, entry := range entries {
		request, directory, _ := strings.Cut(entry, "=")
		encoded = append(encoded, fmt.Sprintf(`{"directory":%q,"request":%q}`, directory, request))
	}
	contents := `{"requests":[` + strings.Join(encoded, ",") + `],"schema":"` + WorkingDirectoriesSchema + `"}` + "\n"
	if err := os.WriteFile(filepath.Join(fixture.root, WorkingDirectoriesFile), []byte(contents), 0o600); err != nil {
		fixture.t.Fatal(err)
	}
}

var requiredVersion = regexp.MustCompile(`(?m)^require ` + regexp.QuoteMeta(refreshDependency) + ` (\S+)$`)

func (fixture *refreshFixture) review(_ context.Context, request Request, directory string) (target.CapabilityReview, error) {
	fixture.reviewed[request.ID] = directory
	contents, err := os.ReadFile(filepath.Join(directory, "go.mod"))
	if err != nil {
		return target.CapabilityReview{}, err
	}
	match := requiredVersion.FindSubmatch(contents)
	if match == nil {
		return target.CapabilityReview{}, errors.New("module does not require the dependency")
	}
	version := string(match[1])
	module := &target.CapabilityModule{Path: refreshDependency, Version: version, Sum: refreshSum(version)}
	digest := sha256.Sum256([]byte("runtime.go@" + version))
	return target.CapabilityReview{
		Schema: target.CapabilityReviewSchema, BuildTags: []string{"test_dep"},
		Closure: target.CapabilityClosure{
			Schema: target.CapabilityClosureSchema, Compatibility: []target.CompatibilityIdentity{},
			Packages: []target.CapabilityPackage{{
				ImportPath: refreshDependency + "/internal/runtime", Name: "runtime", Imports: []string{"syscall"}, Module: module,
				Sources: []target.CapabilitySource{{Name: "runtime.go", SHA256: fmt.Sprintf("sha256:%x", digest)}}, ForeignSources: []target.CapabilityForeignSource{},
			}, {
				ImportPath: request.Target.ExpectedModule + "/fixture", Name: "fixture", Root: true,
				Module:  &target.CapabilityModule{Path: request.Target.ExpectedModule, Main: true},
				Sources: []target.CapabilitySource{}, ForeignSources: []target.CapabilityForeignSource{},
			}},
		},
		Packs: []target.CompatibilityPackEvidence{}, Findings: []target.CapabilityFinding{},
	}, nil
}

// approved discovers id in directory and approves it through Generate, as a
// person would after reviewing the report.
func (fixture *refreshFixture) approved(id, expectedModule, directory, platform string) {
	fixture.t.Helper()
	draft := validRequest()
	draft.ID, draft.Target.ExpectedModule, draft.Platforms = id, expectedModule, []string{platform}
	review, err := fixture.review(context.Background(), draft, filepath.Join(fixture.root, filepath.FromSlash(directory)))
	if err != nil {
		fixture.t.Fatal(err)
	}
	discovered, digest, err := Discover(draft, review)
	if err != nil {
		fixture.t.Fatal(err)
	}
	if err := Generate(fixture.root, discovered, digest); err != nil {
		fixture.t.Fatal(err)
	}
}

func (fixture *refreshFixture) approve(id, digest string) {
	fixture.t.Helper()
	request := fixture.request(id)
	if err := Generate(fixture.root, request, digest); err != nil {
		fixture.t.Fatal(err)
	}
}

func (fixture *refreshFixture) request(id string) Request {
	fixture.t.Helper()
	contents, err := os.ReadFile(filepath.Join(fixture.root, "requests", id+".json"))
	if err != nil {
		fixture.t.Fatal(err)
	}
	request, err := DecodeRequest(contents)
	if err != nil {
		fixture.t.Fatal(err)
	}
	return request
}

func (fixture *refreshFixture) files() map[string]string {
	fixture.t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(fixture.root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		contents, err := os.ReadFile(path)
		relative, _ := filepath.Rel(fixture.root, path)
		files[filepath.ToSlash(relative)] = string(contents)
		return err
	})
	if err != nil {
		fixture.t.Fatal(err)
	}
	return files
}

func (fixture *refreshFixture) refresh(platform string, invalidated ...string) []RefreshResult {
	fixture.t.Helper()
	selected := map[string]string{}
	for _, id := range invalidated {
		selected[id] = "pin impact"
	}
	results, err := Refresh(context.Background(), RefreshSpec{Root: fixture.root, Platform: platform, Invalidated: selected, Review: fixture.review})
	if err != nil {
		fixture.t.Fatal(err)
	}
	return results
}

func resultFor(t *testing.T, results []RefreshResult, id string) RefreshResult {
	t.Helper()
	for _, result := range results {
		if result.ID == id {
			return result
		}
	}
	t.Fatalf("results %+v lack %s", results, id)
	return RefreshResult{}
}

// twoModuleFixture holds two approved requests discovered at v1.0.0 in two
// target modules, then bumps each module to a different candidate version.
func twoModuleFixture(t *testing.T) *refreshFixture {
	fixture := newRefreshFixture(t)
	fixture.module("modules/a", "example.com/a", "v1.0.0")
	fixture.module("modules/b", "example.com/b", "v1.0.0")
	fixture.approved("pack-a", "example.com/a", "modules/a", "darwin/arm64")
	fixture.approved("pack-b", "example.com/b", "modules/b", "darwin/arm64")
	fixture.table("pack-a=modules/a", "pack-b=modules/b")
	if err := Check(fixture.root); err != nil {
		t.Fatal(err)
	}
	fixture.module("modules/a", "example.com/a", "v1.1.0")
	fixture.module("modules/b", "example.com/b", "v1.2.0")
	return fixture
}

func TestRefreshDiscoversEachRequestInItsModuleAndStopsAtApproval(t *testing.T) {
	fixture := twoModuleFixture(t)
	results := fixture.refresh("darwin/arm64", "pack-a", "pack-b")
	for id, want := range map[string]struct{ directory, version string }{
		"pack-a": {"modules/a", "v1.1.0"},
		"pack-b": {"modules/b", "v1.2.0"},
	} {
		result := resultFor(t, results, id)
		if result.Status != RefreshAwaitingApproval || !result.Rewritten || !strings.HasPrefix(result.ReviewSHA256, "sha256:") {
			t.Fatalf("%s result = %+v", id, result)
		}
		if fixture.reviewed[id] != filepath.Join(fixture.root, want.directory) {
			t.Fatalf("%s was reviewed in %s, want %s", id, fixture.reviewed[id], want.directory)
		}
		request := fixture.request(id)
		if request.ApprovalSHA256 != "" || request.Activation[0].Evidence.Version != want.version || request.Packages[0].Evidence.Module.Version != want.version {
			t.Fatalf("%s refreshed request = %+v", id, request)
		}
		if digest, err := ApprovalSHA256(request); err != nil || digest != result.ReviewSHA256 {
			t.Fatalf("%s review digest = %s, %v; refresh printed %s", id, digest, err, result.ReviewSHA256)
		}
		report, err := os.ReadFile(filepath.Join(fixture.root, "reports", id+".md"))
		if err != nil || !strings.Contains(string(report), result.ReviewSHA256) || !strings.Contains(string(report), refreshDependency+"@"+want.version) {
			t.Fatalf("%s review report does not show the fresh evidence: %v\n%s", id, err, report)
		}
		if _, err := os.Stat(filepath.Join(fixture.root, "packs", id+".json")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("%s pack of the older evidence remains: %v", id, err)
		}
	}
	if err := Check(fixture.root); err != nil {
		t.Fatalf("refreshed root is not current: %v", err)
	}
}

func TestRefreshKeepsAnApprovedRequestAndReportsOnlyTheOther(t *testing.T) {
	fixture := twoModuleFixture(t)
	first := fixture.refresh("darwin/arm64", "pack-a", "pack-b")
	fixture.approve("pack-a", resultFor(t, first, "pack-a").ReviewSHA256)
	approved := fixture.files()

	// The candidate is now approved for pack-a, so the pin impact report no
	// longer names it; pack-b still lacks an approval.
	second := fixture.refresh("darwin/arm64")
	if len(second) != 1 || second[0].ID != "pack-b" || second[0].Status != RefreshAwaitingApproval || second[0].Rewritten ||
		second[0].ReviewSHA256 != resultFor(t, first, "pack-b").ReviewSHA256 {
		t.Fatalf("rerun results = %+v", second)
	}
	if !maps.Equal(approved, fixture.files()) {
		t.Fatal("a rerun with unchanged evidence rewrote the pack root")
	}
	if fixture.request("pack-a").ApprovalSHA256 != resultFor(t, first, "pack-a").ReviewSHA256 {
		t.Fatal("the approved request lost its approval")
	}

	// Naming the approved request again finds it current and leaves it.
	third := fixture.refresh("darwin/arm64", "pack-a")
	if result := resultFor(t, third, "pack-a"); result.Status != RefreshCurrent || result.Rewritten {
		t.Fatalf("approved request result = %+v", result)
	}
	if !maps.Equal(approved, fixture.files()) {
		t.Fatal("a current request was rewritten")
	}
}

func TestRefreshNeverTreatsAnApprovalOfOlderEvidenceAsCurrent(t *testing.T) {
	fixture := twoModuleFixture(t)
	older := fixture.request("pack-a").ApprovalSHA256

	// The stored approval names the v1.0.0 evidence; the module now
	// resolves v1.1.0.
	result := resultFor(t, fixture.refresh("darwin/arm64", "pack-a"), "pack-a")
	if result.Status != RefreshAwaitingApproval || result.ReviewSHA256 == older || fixture.request("pack-a").ApprovalSHA256 != "" {
		t.Fatalf("result = %+v", result)
	}

	// A request file that carries the fresh evidence with the older
	// approval is not current either: the approval is cleared.
	fresh := fixture.request("pack-a")
	fresh.ApprovalSHA256 = older
	if err := PublishRequest(filepath.Join(fixture.root, "requests", "pack-a.json"), fresh); err != nil {
		t.Fatal(err)
	}
	result = resultFor(t, fixture.refresh("darwin/arm64", "pack-a"), "pack-a")
	if result.Status != RefreshAwaitingApproval || !result.Rewritten || fixture.request("pack-a").ApprovalSHA256 != "" {
		t.Fatalf("stale approval result = %+v", result)
	}
	// The older approval cannot generate the fresh request.
	if err := Generate(fixture.root, fixture.request("pack-a"), older); err == nil {
		t.Fatal("an approval of older evidence generated the refreshed pack")
	}
}

func TestRefreshReportsOtherPlatformRequestsAndLeavesThemUntouched(t *testing.T) {
	fixture := newRefreshFixture(t)
	fixture.module("modules/a", "example.com/a", "v1.0.0")
	fixture.module("modules/linux", "example.com/linux", "v1.0.0")
	fixture.approved("pack-a", "example.com/a", "modules/a", "darwin/arm64")
	fixture.approved("pack-linux", "example.com/linux", "modules/linux", "linux/amd64")
	fixture.table("pack-a=modules/a", "pack-linux=modules/linux")
	fixture.module("modules/a", "example.com/a", "v1.1.0")
	fixture.module("modules/linux", "example.com/linux", "v1.1.0")
	before := fixture.files()
	fixture.reviewed = map[string]string{}

	results := fixture.refresh("darwin/arm64", "pack-a", "pack-linux")
	if result := resultFor(t, results, "pack-linux"); result.Status != RefreshNotEvaluable || !strings.Contains(result.Reason, "linux/amd64") {
		t.Fatalf("other-platform result = %+v", result)
	}
	if _, reviewed := fixture.reviewed["pack-linux"]; reviewed {
		t.Fatal("the other-platform request was reviewed on this host")
	}
	after := fixture.files()
	for _, path := range []string{"requests/pack-linux.json", "reports/pack-linux.md", "packs/pack-linux.json"} {
		if before[path] == "" || after[path] != before[path] {
			t.Fatalf("%s changed or is missing", path)
		}
	}
	if resultFor(t, results, "pack-a").Status != RefreshAwaitingApproval || after["requests/pack-a.json"] == before["requests/pack-a.json"] {
		t.Fatal("the host-platform request was not refreshed")
	}
}

func TestRefreshRejectsAnUnmappedRequestAsInvalidInput(t *testing.T) {
	fixture := twoModuleFixture(t)
	fixture.table("pack-a=modules/a")
	before := fixture.files()
	_, err := Refresh(context.Background(), RefreshSpec{Root: fixture.root, Platform: "darwin/arm64", Invalidated: map[string]string{"pack-a": "x"}, Review: fixture.review})
	if !IsInputError(err) || !strings.Contains(err.Error(), "pack-b") {
		t.Fatalf("refresh with an unmapped request = %v", err)
	}
	if !maps.Equal(before, fixture.files()) || len(fixture.reviewed) != 2 {
		t.Fatal("refresh of invalid input reviewed or wrote something")
	}
	if err := Check(fixture.root); !IsInputError(err) {
		t.Fatalf("check with an unmapped request = %v", err)
	}
	for name, table := range map[string][]string{
		"unknown request": {"pack-a=modules/a", "pack-b=modules/b", "pack-c=modules/c"},
		"absolute":        {"pack-a=/modules/a", "pack-b=modules/b"},
		"unclean":         {"pack-a=modules/../a", "pack-b=modules/b"},
		"unsorted":        {"pack-b=modules/b", "pack-a=modules/a"},
	} {
		fixture.table(table...)
		if _, err := LoadWorkingDirectories(fixture.root); !IsInputError(err) {
			t.Fatalf("%s table = %v", name, err)
		}
	}
	fixture.table("pack-a=modules/a", "pack-b=modules/b")
	if _, err := Refresh(context.Background(), RefreshSpec{Root: fixture.root, Platform: "darwin/arm64", Invalidated: map[string]string{"pack-z": "x"}, Review: fixture.review}); !IsInputError(err) {
		t.Fatalf("refresh of a pack without a request = %v", err)
	}
}

func TestRefreshReportsAReviewFailureAndContinues(t *testing.T) {
	fixture := twoModuleFixture(t)
	// pack-a's module no longer requires the dependency at all.
	if err := os.WriteFile(filepath.Join(fixture.root, "modules", "a", "go.mod"), []byte("module example.com/a\n\ngo 1.27.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	results := fixture.refresh("darwin/arm64", "pack-a", "pack-b")
	if result := resultFor(t, results, "pack-a"); result.Status != RefreshFailed || result.Rewritten {
		t.Fatalf("failed result = %+v", result)
	}
	if fixture.request("pack-a").ApprovalSHA256 == "" {
		t.Fatal("a request that failed review was rewritten")
	}
	if !slices.ContainsFunc(results, func(result RefreshResult) bool {
		return result.ID == "pack-b" && result.Status == RefreshAwaitingApproval
	}) {
		t.Fatalf("results = %+v", results)
	}
}

func TestCheckedInWorkingDirectoriesMapEveryRequest(t *testing.T) {
	directories, err := LoadWorkingDirectories("..")
	if err != nil {
		t.Fatal(err)
	}
	for id, directory := range directories {
		if info, err := os.Stat(filepath.Join(directory, "go.mod")); err != nil || !info.Mode().IsRegular() {
			t.Errorf("%s working directory %s has no go.mod: %v", id, directory, err)
		}
	}
}

func TestRefreshSelectsAHostRequestBoundToAnotherProfile(t *testing.T) {
	fixture := newRefreshFixture(t)
	fixture.module("modules/a", "example.com/a", "v1.0.0")
	fixture.approved("pack-a", "example.com/a", "modules/a", "darwin/arm64")
	fixture.table("pack-a=modules/a")
	request := fixture.request("pack-a")
	request.Activation[0].Evidence.Replacement = compatibility.PackReplacement{Kind: compatibility.ReplacementAdapter, Adapter: &compatibility.PackAdapter{ProfileName: "profile/v1", ProfileImplementationSHA256: "sha256:old"}}
	current := &ProfileIdentity{Name: "profile/v1", ImplementationSHA256: "sha256:current"}
	if binding := staleProfileBinding(request, "darwin/arm64", current); !strings.Contains(binding, "sha256:old") {
		t.Fatalf("stale binding = %q", binding)
	}
	if binding := staleProfileBinding(request, "linux/amd64", current); binding != "" {
		t.Fatalf("other-platform binding = %q", binding)
	}
	request.Activation[0].Evidence.Replacement.Adapter.ProfileImplementationSHA256 = "sha256:current"
	if binding := staleProfileBinding(request, "darwin/arm64", current); binding != "" {
		t.Fatalf("current binding = %q", binding)
	}
	// Without bindings the approved, unchanged request is not selected.
	results, err := Refresh(context.Background(), RefreshSpec{Root: fixture.root, Platform: "darwin/arm64", Profile: current, Review: fixture.review})
	if err != nil || len(results) != 0 {
		t.Fatalf("results = %+v, %v", results, err)
	}
}
