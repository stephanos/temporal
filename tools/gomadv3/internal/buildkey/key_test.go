package buildkey

import (
	"os"
	"path/filepath"
	"testing"
)

func TestComputeChangesForEveryContractedInput(t *testing.T) {
	base := Input{
		GoVersion: "go1.26.4", ArchiveSHA256: "archive", PatchSHA256: "patch", OverlaySHA256: "overlay",
		HostOS: "darwin", HostArch: "arm64", BootstrapVersion: "go version go1.26.4 darwin/arm64",
		RecipeVersion: "canonical-v4", BuildPath: "/usr/bin:/bin", BashPath: "/bin/bash", BashVersion: "5.2",
	}
	baseline, err := Compute(base)
	if err != nil {
		t.Fatal(err)
	}
	if repeated, err := Compute(base); err != nil || repeated != baseline {
		t.Fatalf("repeated Compute() = %q, %v; want %q", repeated, err, baseline)
	}
	mutations := map[string]func(*Input){
		"Go version":        func(input *Input) { input.GoVersion += ".1" },
		"archive digest":    func(input *Input) { input.ArchiveSHA256 += "1" },
		"patch digest":      func(input *Input) { input.PatchSHA256 += "1" },
		"overlay digest":    func(input *Input) { input.OverlaySHA256 += "1" },
		"host OS":           func(input *Input) { input.HostOS = "linux" },
		"host architecture": func(input *Input) { input.HostArch = "amd64" },
		"bootstrap version": func(input *Input) { input.BootstrapVersion += ".1" },
		"recipe version":    func(input *Input) { input.RecipeVersion += ".1" },
		"build path":        func(input *Input) { input.BuildPath += ":/opt/bin" },
		"bash path":         func(input *Input) { input.BashPath += ".new" },
		"bash version":      func(input *Input) { input.BashVersion += ".1" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			changed := base
			mutate(&changed)
			key, err := Compute(changed)
			if err != nil {
				t.Fatal(err)
			}
			if key == baseline {
				t.Fatalf("Compute() did not change for %s", name)
			}
		})
	}
}

func TestTreeDigestBindsPathsAndContents(t *testing.T) {
	first := t.TempDir()
	second := t.TempDir()
	third := t.TempDir()
	writeFile(t, filepath.Join(first, "src", "a.go"), "same")
	writeFile(t, filepath.Join(second, "src", "b.go"), "same")
	writeFile(t, filepath.Join(third, "src", "a.go"), "different")
	firstDigest, err := TreeDigest(first)
	if err != nil {
		t.Fatal(err)
	}
	if repeated, err := TreeDigest(first); err != nil || repeated != firstDigest {
		t.Fatalf("repeated TreeDigest() = %q, %v; want %q", repeated, err, firstDigest)
	}
	for name, root := range map[string]string{"path": second, "contents": third} {
		digest, err := TreeDigest(root)
		if err != nil {
			t.Fatal(err)
		}
		if digest == firstDigest {
			t.Fatalf("TreeDigest() did not bind %s", name)
		}
	}
}

func TestTreeDigestRejectsNonRegularEntry(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "target"), "target")
	if err := os.Symlink("target", filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	if _, err := TreeDigest(root); err == nil {
		t.Fatal("TreeDigest() accepted a symlink")
	}
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
