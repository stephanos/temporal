package adapterregen

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

func TestRealAdapterVersionUsesDefaultPipeline(t *testing.T) {
	goCommand, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	output, err := exec.CommandContext(t.Context(), goCommand, "env", "GOVERSION").Output()
	if err != nil || strings.TrimSpace(string(output)) != gomadversion.GoVersion {
		t.Fatalf("default pipeline requires stock %s: %s (%v)", gomadversion.GoVersion, output, err)
	}
	source, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "root")
	before, err := copyCheckout(source, root)
	if err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(root, ".toolchain", "generator-cache")
	if info, err := os.Lstat(cache); err == nil {
		if info.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("fixture cache is not a symlink: %s", cache)
		}
		if err := os.Remove(cache); err != nil {
			t.Fatal(err)
		}
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
	spec := Spec{Root: root, Module: "github.com/Masterminds/sprig/v3", Version: "v3.2.3", GoCommand: goCommand, Environment: os.Environ()}
	review, err := Run(t.Context(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if review.Applied || review.Regeneration.Previous.Version != "v3.3.0" || review.Regeneration.Proposed.Version != spec.Version || review.Regeneration.ApprovalSHA256 == "" {
		t.Fatalf("real version review = %+v", review)
	}
	for relative, expected := range before {
		contents, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil || digest(contents) != expected {
			t.Fatalf("dry run changed %s: %v", relative, err)
		}
	}
	spec.Approval = review.Regeneration.ApprovalSHA256
	stagedDigests := map[string]string{}
	spec.beforeApplyFile = func(index int) error {
		if index != 0 {
			return nil
		}
		contents, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(stateDirectory), journalDirectory, journalManifest))
		if err != nil {
			return err
		}
		var manifest journal
		if err := json.Unmarshal(contents, &manifest); err != nil {
			return err
		}
		for _, entry := range manifest.Entries {
			stagedDigests[entry.Path] = entry.Next
		}
		return nil
	}
	applied, err := Run(t.Context(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if !applied.Applied || len(applied.Staged) == 0 || len(applied.Warnings) != 0 {
		t.Fatalf("default pipeline publication = %+v", applied)
	}
	changed := map[string]bool{}
	for _, file := range applied.Staged {
		contents, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file.Path)))
		if err != nil || digest(contents) == before[file.Path] || digest(contents) != stagedDigests[file.Path] {
			t.Fatalf("published output %s differs from its staged digest: %v", file.Path, err)
		}
		changed[file.Path] = true
	}
	for _, relative := range []string{"deterministicio/sprig_adapter.go", "toolchain/version/version.json", "toolchain/version/generated.go"} {
		if !changed[relative] {
			t.Fatalf("default pipeline did not publish %s: %v", relative, applied.Staged)
		}
	}
	for relative, expected := range before {
		if changed[relative] {
			continue
		}
		contents, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil || digest(contents) != expected {
			t.Fatalf("publication changed an unlisted file %s: %v", relative, err)
		}
	}
	var rendered bytes.Buffer
	if err := Render(&rendered, applied); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered.String(), "v3.2.3") {
		t.Fatalf("publication output omits the real candidate: %s", &rendered)
	}
	if err := Recover(root); err != nil {
		t.Fatal(err)
	}
	t.Logf("real controlled version %s@v3.3.0 -> %s: %d published files; production pins unchanged", spec.Module, spec.Version, len(applied.Staged))
}
