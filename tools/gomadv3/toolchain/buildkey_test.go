package toolchain

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeriveHashesBuildSources(t *testing.T) {
	root := t.TempDir()
	patch := filepath.Join(root, "gomad.patch")
	overlay := filepath.Join(root, "overlay")
	writeFile(t, patch, "patch")
	writeFile(t, filepath.Join(overlay, "src", "gomad.go"), "package gomad")
	input := BuildKeySpec{
		GoVersion: "go1.26.4", ArchiveSHA256: "archive", PatchPath: patch, OverlayPath: overlay,
		HostOS: "darwin", HostArch: "arm64", BootstrapVersion: "go version go1.26.4 darwin/arm64",
		RecipeVersion: "canonical-v4", BuildPath: "/usr/bin:/bin", BashPath: "/bin/bash", BashVersion: "5.2",
	}
	baseline, err := DeriveBuildKey(input)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, patch, "changed patch")
	patchChanged, err := DeriveBuildKey(input)
	if err != nil {
		t.Fatal(err)
	}
	if patchChanged == baseline {
		t.Fatal("DeriveBuildKey() did not bind patch contents")
	}
	writeFile(t, patch, "patch")
	writeFile(t, filepath.Join(overlay, "src", "gomad.go"), "package changed")
	changed, err := DeriveBuildKey(input)
	if err != nil {
		t.Fatal(err)
	}
	if changed == baseline {
		t.Fatal("DeriveBuildKey() did not bind overlay contents")
	}
}

func TestComputeChangesForEveryContractedInput(t *testing.T) {
	base := buildKeyIdentity{input: BuildKeySpec{
		GoVersion: "go1.26.4", ArchiveSHA256: "archive",
		HostOS: "darwin", HostArch: "arm64", BootstrapVersion: "go version go1.26.4 darwin/arm64",
		RecipeVersion: "canonical-v4", BuildPath: "/usr/bin:/bin", BashPath: "/bin/bash", BashVersion: "5.2",
	}, patchSHA256: "patch", overlaySHA256: "overlay"}
	baseline, err := computeBuildKeyDigest(base)
	if err != nil {
		t.Fatal(err)
	}
	if repeated, err := computeBuildKeyDigest(base); err != nil || repeated != baseline {
		t.Fatalf("repeated computeBuildKeyDigest() = %q, %v; want %q", repeated, err, baseline)
	}
	mutations := map[string]func(*buildKeyIdentity){
		"Go version":        func(source *buildKeyIdentity) { source.input.GoVersion += ".1" },
		"archive digest":    func(source *buildKeyIdentity) { source.input.ArchiveSHA256 += "1" },
		"patch digest":      func(source *buildKeyIdentity) { source.patchSHA256 += "1" },
		"overlay digest":    func(source *buildKeyIdentity) { source.overlaySHA256 += "1" },
		"host OS":           func(source *buildKeyIdentity) { source.input.HostOS = "linux" },
		"host architecture": func(source *buildKeyIdentity) { source.input.HostArch = "amd64" },
		"bootstrap version": func(source *buildKeyIdentity) { source.input.BootstrapVersion += ".1" },
		"recipe version":    func(source *buildKeyIdentity) { source.input.RecipeVersion += ".1" },
		"build path":        func(source *buildKeyIdentity) { source.input.BuildPath += ":/opt/bin" },
		"bash path":         func(source *buildKeyIdentity) { source.input.BashPath += ".new" },
		"bash version":      func(source *buildKeyIdentity) { source.input.BashVersion += ".1" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			changed := base
			mutate(&changed)
			key, err := computeBuildKeyDigest(changed)
			if err != nil {
				t.Fatal(err)
			}
			if key == baseline {
				t.Fatalf("computeBuildKeyDigest() did not change for %s", name)
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
	firstDigest, err := buildKeyTreeDigest(first)
	if err != nil {
		t.Fatal(err)
	}
	if repeated, err := buildKeyTreeDigest(first); err != nil || repeated != firstDigest {
		t.Fatalf("repeated buildKeyTreeDigest() = %q, %v; want %q", repeated, err, firstDigest)
	}
	for name, root := range map[string]string{"path": second, "contents": third} {
		digest, err := buildKeyTreeDigest(root)
		if err != nil {
			t.Fatal(err)
		}
		if digest == firstDigest {
			t.Fatalf("buildKeyTreeDigest() did not bind %s", name)
		}
	}
}

func TestTreeDigestRejectsNonRegularEntry(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "target"), "target")
	if err := os.Symlink("target", filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	if _, err := buildKeyTreeDigest(root); err == nil {
		t.Fatal("buildKeyTreeDigest() accepted a symlink")
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
