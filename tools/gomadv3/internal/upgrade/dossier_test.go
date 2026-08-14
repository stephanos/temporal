package upgrade

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomadv3/internal/capabilityanalysis"
	"go.temporal.io/server/tools/gomadv3/internal/compatibility"
	"go.temporal.io/server/tools/gomadv3/internal/ioprofile"
	"go.temporal.io/server/tools/gomadv3/internal/qualificationset"
	"go.temporal.io/server/tools/gomadv3/internal/qualify"
	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/target"
)

func TestRunPublishesCheckedUpgradeEvidence(t *testing.T) {
	root := writeUpgradeFixture(t, false)
	output := filepath.Join(root, ".toolchain", "upgrade-dossier.json")
	baseline := []byte(`{"manifest_version":"go1.26.3-darwin-arm64-v1","intercepts":[{"package":"os","symbol":"Open","signature":"func(name string) (*File, error)","hook":"oldHook"}]}`)
	approvedDiff := boundaryApprovalFor(t, root, baseline)
	err := Run(context.Background(), Options{
		Root: root, Output: output, BaselineManifest: baseline, CorpusReport: writeQualifiedCorpus(t, root, "gomadv3-core"),
		ApprovedBoundaryDiffSHA256: approvedDiff,
		Gates:                      []Gate{{Name: "unit", Command: []string{"/usr/bin/printf", "gate passed\n"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var dossier Dossier
	if err := json.Unmarshal(contents, &dossier); err != nil {
		t.Fatal(err)
	}
	if dossier.Schema != "gomadv3.upgrade-dossier/v2" || !dossier.Qualified || !dossier.BoundaryApproved || dossier.Version.GoVersion != "go1.26.4" {
		t.Fatalf("dossier identity = %#v", dossier)
	}
	if !strings.Contains(dossier.UpstreamPatch.Diff, "src/runtime/proc.go") || dossier.UpstreamPatch.SHA256 == "" {
		t.Fatalf("upstream patch evidence = %#v", dossier.UpstreamPatch)
	}
	if len(dossier.BoundaryDiff.Added) != 1 || dossier.BoundaryDiff.Added[0] != "os.OpenFile" || len(dossier.BoundaryDiff.Removed) != 1 || dossier.BoundaryDiff.Removed[0] != "os.Open" {
		t.Fatalf("boundary diff = %#v", dossier.BoundaryDiff)
	}
	if dossier.BoundaryDiff.SHA256 != approvedDiff || dossier.BoundaryApprovalSHA256 != approvedDiff {
		t.Fatalf("boundary approval = %q for %#v", dossier.BoundaryApprovalSHA256, dossier.BoundaryDiff)
	}
	if !dossier.OverlayCollision.Checked || len(dossier.OverlayCollision.Collisions) != 0 {
		t.Fatalf("overlay collision evidence = %#v", dossier.OverlayCollision)
	}
	if len(dossier.Gates) != 1 || dossier.Gates[0].Status != "passed" || dossier.Gates[0].Output != "gate passed\n" {
		t.Fatalf("gate evidence = %#v", dossier.Gates)
	}
	if dossier.RetainedCorpus.Status != "checked" || dossier.RetainedCorpus.SHA256 == "" {
		t.Fatalf("retained corpus evidence = %#v", dossier.RetainedCorpus)
	}
}

func TestRunDoesNotQualifyUnapprovedBoundaryChanges(t *testing.T) {
	root := writeUpgradeFixture(t, false)
	output := filepath.Join(root, ".toolchain", "upgrade-dossier.json")
	baseline := []byte(`{"manifest_version":"go1.26.3-darwin-arm64-v1","intercepts":[{"package":"os","symbol":"Open","signature":"func(name string) (*File, error)","hook":"oldHook"}]}`)
	err := Run(context.Background(), Options{
		Root: root, Output: output, BaselineManifest: baseline, CorpusReport: writeQualifiedCorpus(t, root, "gomadv3-core"),
		Gates: []Gate{{Name: "unit", Command: []string{"/usr/bin/printf", "gate passed\n"}}},
	})
	if err == nil || !strings.Contains(err.Error(), "boundary changes require explicit approval") {
		t.Fatalf("Run() error = %v", err)
	}
	contents, readErr := os.ReadFile(output)
	if readErr != nil {
		t.Fatal(readErr)
	}
	var public map[string]any
	if unmarshalErr := json.Unmarshal(contents, &public); unmarshalErr != nil {
		t.Fatal(unmarshalErr)
	}
	if public["qualified"] != false || public["boundary_changes_approved"] != false {
		t.Fatalf("dossier approval evidence = %#v", public)
	}
}

func TestRunDoesNotQualifyBoundaryApprovalForAnotherDiff(t *testing.T) {
	root := writeUpgradeFixture(t, false)
	output := filepath.Join(root, ".toolchain", "upgrade-dossier.json")
	baseline := []byte(`{"manifest_version":"go1.26.3-darwin-arm64-v1","intercepts":[{"package":"os","symbol":"Open","signature":"func(name string) (*File, error)","hook":"oldHook"}]}`)
	err := Run(context.Background(), Options{
		Root: root, Output: output, BaselineManifest: baseline, CorpusReport: writeQualifiedCorpus(t, root, "gomadv3-core"),
		ApprovedBoundaryDiffSHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Gates:                      []Gate{{Name: "unit", Command: []string{"/usr/bin/printf", "gate passed\n"}}},
	})
	if err == nil || !strings.Contains(err.Error(), "boundary changes require explicit approval for sha256:") {
		t.Fatalf("Run() error = %v", err)
	}
	contents, readErr := os.ReadFile(output)
	if readErr != nil {
		t.Fatal(readErr)
	}
	var dossier Dossier
	if unmarshalErr := json.Unmarshal(contents, &dossier); unmarshalErr != nil {
		t.Fatal(unmarshalErr)
	}
	if dossier.Qualified || dossier.BoundaryApproved || dossier.BoundaryApprovalSHA256 != "" || dossier.BoundaryDiff.SHA256 == "" {
		t.Fatalf("dossier approval evidence = %#v", dossier)
	}
}

func TestRunPublishesUnqualifiedEvidenceWhenCorpusIsMissing(t *testing.T) {
	root := writeUpgradeFixture(t, false)
	output := filepath.Join(root, ".toolchain", "upgrade-dossier.json")
	err := Run(context.Background(), Options{
		Root: root, Output: output,
		Gates: []Gate{{Name: "unit", Command: []string{"/usr/bin/printf", "gate passed\n"}}},
	})
	if err == nil || !strings.Contains(err.Error(), "corpus") {
		t.Fatalf("Run() error = %v", err)
	}
	contents, readErr := os.ReadFile(output)
	if readErr != nil {
		t.Fatal(readErr)
	}
	var dossier Dossier
	if unmarshalErr := json.Unmarshal(contents, &dossier); unmarshalErr != nil {
		t.Fatal(unmarshalErr)
	}
	if dossier.Qualified || dossier.RetainedCorpus.Status != "not-configured" {
		t.Fatalf("dossier = %#v", dossier)
	}
}

func TestRunPublishesFailedGateEvidence(t *testing.T) {
	root := writeUpgradeFixture(t, false)
	output := filepath.Join(root, ".toolchain", "upgrade-dossier.json")
	err := Run(context.Background(), Options{
		Root: root, Output: output,
		Gates: []Gate{{Name: "failure", Command: []string{"/bin/sh", "-c", "printf failed-output; exit 23"}}},
	})
	if err == nil || !strings.Contains(err.Error(), "qualification gate failure") {
		t.Fatalf("Run() error = %v", err)
	}
	contents, readErr := os.ReadFile(output)
	if readErr != nil {
		t.Fatal(readErr)
	}
	var dossier Dossier
	if unmarshalErr := json.Unmarshal(contents, &dossier); unmarshalErr != nil {
		t.Fatal(unmarshalErr)
	}
	if dossier.Qualified || len(dossier.Gates) != 1 || dossier.Gates[0].Status != "failed" || dossier.Gates[0].ExitCode != 23 || dossier.Gates[0].Output != "failed-output" {
		t.Fatalf("failed dossier = %#v", dossier)
	}
}

func TestRunRejectsOverlayCollisionBeforeGates(t *testing.T) {
	root := writeUpgradeFixture(t, true)
	err := Run(context.Background(), Options{Root: root, Output: filepath.Join(root, "dossier.json")})
	if err == nil || !strings.Contains(err.Error(), "overlay collides with upstream source") {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestCompareBoundariesRetainsNewManifestAndEntryFields(t *testing.T) {
	baseline := []byte(`{
		"manifest_version":"v1",
		"future_contract":"old",
		"hook_policies":[{"id":"deny/v1","enabled":"old"}],
		"intercepts":[{"package":"os","symbol":"OpenFile","future_field":"old"}]
	}`)
	current := []byte(`{
		"manifest_version":"v1",
		"future_contract":"new",
		"hook_policies":[{"id":"deny/v1","enabled":"new"}],
		"intercepts":[{"package":"os","symbol":"OpenFile","future_field":"new"}]
	}`)
	difference, err := compareBoundaries(baseline, current)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"hook-policy:deny/v1", "manifest", "os.OpenFile"}
	if fmt.Sprint(difference.Changed) != fmt.Sprint(want) {
		t.Fatalf("changed boundaries = %v, want %v", difference.Changed, want)
	}
	otherCurrent := []byte(`{
		"manifest_version":"v1",
		"future_contract":"new",
		"hook_policies":[{"id":"deny/v1","enabled":"new"}],
		"intercepts":[{"package":"os","symbol":"OpenFile","future_field":"another-new-value"}]
	}`)
	otherDifference, err := compareBoundaries(baseline, otherCurrent)
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(otherDifference.Changed) != fmt.Sprint(want) || otherDifference.SHA256 == difference.SHA256 {
		t.Fatalf("boundary diff identities = %v/%v, SHA-256 = %q/%q", difference.Changed, otherDifference.Changed, difference.SHA256, otherDifference.SHA256)
	}
}

func TestLoadCorpusRejectsUncheckedOrUnqualifiedJSON(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "corpus.json")
	if err := os.WriteFile(path, []byte("{\"qualified\":false,\"schema\":\"gomadv3.qualification-set-report/v1\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadCorpus(root, path); err == nil || !strings.Contains(err.Error(), "qualification set") {
		t.Fatalf("loadCorpus() error = %v", err)
	}
}

func TestLoadCorpusRejectsQualifiedDownstreamSet(t *testing.T) {
	root := t.TempDir()
	path := writeQualifiedCorpus(t, root, "temporal-representative")
	if _, err := loadCorpus(root, path); err == nil || !strings.Contains(err.Error(), "gomadv3-core") {
		t.Fatalf("loadCorpus() error = %v", err)
	}
}

func writeQualifiedCorpus(t *testing.T, root, name string) string {
	t.Helper()
	manifest := map[string]any{
		"schema": qualificationset.LegacyManifestSchema, "name": name, "seed": 7, "repeat": 2,
		"run_timeout": "30s", "overall_timeout": "2m", "terminate_grace": "2s",
		"output_bytes": 1024, "world_transition_bytes": 2048,
		"suites": []any{map[string]any{
			"name": "boundary", "package": "./pkg", "test": "TestBoundary",
			"expectation": map[string]any{
				"classification": "unsupported_target", "import_path": "example.com/escape", "capability": "imports os/exec",
			},
		}},
	}
	contents, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, "corpus-manifest.json")
	if err = os.WriteFile(manifestPath, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/corpus\n\ngo 1.26.4\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(root, ".toolchain", "corpus.json")
	_, err = qualificationset.Run(context.Background(), qualificationset.Config{
		ManifestPath: manifestPath, GomadPath: filepath.Join(root, "gomad"), WorkingDir: root,
		ArtifactRoot: filepath.Join(root, ".toolchain", "corpus-artifacts"), OutputPath: output,
		Execute: func(_ context.Context, command qualificationset.Command) qualificationset.CommandResult {
			if command.Args[0] == "analyze" {
				boundaryVersion, boundaryDigest := ioprofile.BoundaryManifestIdentity()
				analysis := capabilityanalysis.Report{
					Schema: capabilityanalysis.Schema, Classification: capabilityanalysis.ClassificationUnsupported,
					Target:    capabilityanalysis.Target{Kind: target.KindGoTest, Source: "pkg", Arguments: []string{"-test.run=^TestBoundary$"}, BuildTags: []string{}},
					Toolchain: capabilityanalysis.Toolchain{GoVersion: "go1.26.4", BuildKey: strings.Repeat("a", 64), TargetGOOS: "darwin", TargetGOARCH: "arm64", BoundaryManifestVersion: boundaryVersion, BoundaryManifestSHA256: boundaryDigest},
					Closure:   capabilityanalysis.Closure{SHA256: record.HashBytes([]byte("closure")), PackageCount: 1, Roots: []target.CapabilityPackageReference{{ImportPath: "example.com/corpus/pkg", Name: "pkg"}}},
					IOProfile: ioprofile.Default().Identity(), Packs: []compatibility.PackEvidence{}, Requirements: []ioprofile.Requirement{},
					Blockers: []capabilityanalysis.Blocker{{
						CapabilityFinding: target.CapabilityFinding{Kind: target.FindingForbiddenImport, Package: target.CapabilityPackageReference{ImportPath: "example.com/escape", Name: "escape"}, Capability: "imports os/exec", Directives: []string{}, PolicyDisposition: compatibility.DispositionDenied, Remediation: compatibility.RemediationRemainUnsupported},
						DependencyPath:    []target.CapabilityPackageReference{{ImportPath: "example.com/corpus/pkg", Name: "pkg"}, {ImportPath: "example.com/escape", Name: "escape"}},
					}},
				}
				encoded, encodeErr := record.CanonicalJSON(analysis)
				if encodeErr != nil {
					t.Fatal(encodeErr)
				}
				return qualificationset.CommandResult{ExitCode: 1, Stdout: append(encoded, '\n')}
			}
			logicalCommand := append([]string{"gomad"}, command.Args...)
			report, buildErr := qualify.BuildFailure(logicalCommand, 7, 2, nil, qualify.Failure{
				Classification: "unsupported_target", Iteration: 1, Message: "unsupported boundary",
				ImportPath: "example.com/escape", Capability: "imports os/exec",
			})
			if buildErr != nil {
				t.Fatal(buildErr)
			}
			path, writeErr := qualify.Write(command.ArtifactRoot, report)
			if writeErr != nil {
				t.Fatal(writeErr)
			}
			event, marshalErr := json.Marshal(map[string]any{
				"schema": "gomadv3.qualify-event/v1", "type": "result", "classification": "unsupported_target", "report_path": path, "report": report,
			})
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			return qualificationset.CommandResult{ExitCode: 2, Stdout: append(event, '\n')}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return output
}

func boundaryApprovalFor(t *testing.T, root string, baseline []byte) string {
	t.Helper()
	current, err := os.ReadFile(filepath.Join(root, "boundary", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	difference, err := compareBoundaries(baseline, current)
	if err != nil {
		t.Fatal(err)
	}
	return difference.SHA256
}

func writeUpgradeFixture(t *testing.T, collide bool) string {
	t.Helper()
	root := t.TempDir()
	for _, directory := range []string{"boundary", "overlay/src/os", ".toolchain/downloads"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(directory)), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	patch := "diff --git a/src/runtime/proc.go b/src/runtime/proc.go\n--- a/src/runtime/proc.go\n+++ b/src/runtime/proc.go\n"
	if err := os.WriteFile(filepath.Join(root, "go1.26.4.patch"), []byte(patch), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "overlay", "src", "os", "gomad.go"), []byte("package os\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := `{"schema_version":1,"manifest_version":"go1.26.4-darwin-arm64-v1","go_version":"go1.26.4","platforms":["darwin/arm64"],"intercepts":[{"package":"os","symbol":"OpenFile","signature":"func(name string, flag int, perm FileMode) (*File, error)","hook":"gomadInterceptOpenFile"}]}`
	if err := os.WriteFile(filepath.Join(root, "boundary", "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "expected-intercepts-go1.26.4.txt"), []byte("os.OpenFile -> os.gomadInterceptOpenFile\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(root, ".toolchain", "downloads", "go1.26.4.src.tar.gz")
	archive := createArchive(t, archivePath, collide)
	digest := sha256.Sum256(archive)
	descriptor := fmt.Sprintf(`{
  "schema_version": 1,
  "go_version": "go1.26.4",
  "archive": {"name":"go1.26.4.src.tar.gz","url":"https://go.dev/dl/go1.26.4.src.tar.gz","sha256":"%x"},
  "supported_platforms": ["darwin/arm64"],
  "boundary_manifest_version": "go1.26.4-darwin-arm64-v1",
  "patch": "go1.26.4.patch",
  "adapters": [{"module":"modernc.org/libc","version":"v1.72.3","sum":"h1:test"}],
  "patch_allowlist": ["src/runtime/proc.go"],
  "overlay_allowlist": ["src/os/gomad.go"]
}
`, digest)
	if err := os.WriteFile(filepath.Join(root, "version.json"), []byte(descriptor), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func createArchive(t *testing.T, path string, collide bool) []byte {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zipper := gzip.NewWriter(file)
	archive := tar.NewWriter(zipper)
	paths := []string{"go/src/runtime/proc.go"}
	if collide {
		paths = append(paths, "go/src/os/gomad.go")
	}
	for _, name := range paths {
		contents := []byte("package fixture\n")
		if err := archive.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(contents)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := archive.Write(contents); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := zipper.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return contents
}
