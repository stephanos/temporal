package patchset

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomadv3/internal/sourcearchive"
)

func TestValidateAcceptsCurrentCheckedInputs(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(Config{Root: root}); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsPatchOutsideDescriptorAllowlist(t *testing.T) {
	root := writeFixture(t)
	patch := `diff --git a/src/runtime/chan.go b/src/runtime/chan.go
--- a/src/runtime/chan.go
+++ b/src/runtime/chan.go
@@ -1 +1 @@
-package runtime
+package runtime // changed
`
	if err := os.WriteFile(filepath.Join(root, "gomad.patch"), []byte(patch), 0o600); err != nil {
		t.Fatal(err)
	}
	err := Validate(Config{Root: root})
	if err == nil || !strings.Contains(err.Error(), "prohibited") {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateRejectsMalformedAndNewFilePatches(t *testing.T) {
	for name, patch := range map[string]string{
		"malformed": `diff --git a/src/runtime/proc.go b/src/runtime/proc.go
--- a/src/runtime/proc.go
+++ b/src/runtime/proc.go
@@ -1 +1 @@
invalid context
`,
		"new-file": `diff --git a/src/runtime/proc.go b/src/runtime/proc.go
new file mode 100644
--- /dev/null
+++ b/src/runtime/proc.go
@@ -0,0 +1 @@
+package runtime
`,
	} {
		t.Run(name, func(t *testing.T) {
			root := writeFixture(t)
			if err := os.WriteFile(filepath.Join(root, "gomad.patch"), []byte(patch), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := Validate(Config{Root: root}); err == nil {
				t.Fatal("Validate() accepted invalid patch")
			}
		})
	}
}

func TestValidateRejectsUnlistedAndBinaryOverlayEntries(t *testing.T) {
	for _, test := range []struct {
		name     string
		relative string
		contents []byte
	}{
		{name: "unlisted", relative: "src/runtime/extra.go", contents: []byte("package runtime\n")},
		{name: "binary", relative: "src/runtime/gomad.go", contents: []byte("package runtime\x00")},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := writeFixture(t)
			path := filepath.Join(root, "overlay", filepath.FromSlash(test.relative))
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, test.contents, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := Validate(Config{Root: root}); err == nil {
				t.Fatal("Validate() accepted invalid overlay")
			}
		})
	}
}

func TestValidateClassifiesProhibitedRuntimeAreas(t *testing.T) {
	for _, test := range []struct {
		path string
		want string
	}{
		{path: "src/runtime/chan.go", want: "prohibited runtime area"},
		{path: "src/runtime/netpoll_epoll.go", want: "prohibited runtime area"},
		{path: "src/runtime/malloc.go", want: "prohibited runtime area"},
		{path: "src/runtime/tagptr_64bit.go", want: "prohibited platform file"},
		{path: "src/runtime/nested/gomad.go", want: "prohibited path"},
	} {
		t.Run(strings.ReplaceAll(test.path, "/", "-"), func(t *testing.T) {
			root := writeFixture(t)
			path := filepath.Join(root, "overlay", filepath.FromSlash(test.path))
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("package runtime\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			err := Validate(Config{Root: root})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Validate() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestMaterializeAppliesExactPatch(t *testing.T) {
	root := writeFixture(t)
	source := writeSource(t)
	if err := Materialize(context.Background(), Config{Root: root, SourceRoot: source}); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(filepath.Join(source, "src", "runtime", "proc.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "alpha\nreplacement\nomega\n" {
		t.Fatalf("materialized source = %q", contents)
	}
	if _, err := os.Stat(filepath.Join(source, "src", "runtime", "proc.go.orig")); !os.IsNotExist(err) {
		t.Fatalf("backup file exists: %v", err)
	}
}

func TestMaterializeRejectsNonapplyingPatchWithoutMutation(t *testing.T) {
	root := writeFixture(t)
	patchPath := filepath.Join(root, "gomad.patch")
	contents, err := os.ReadFile(patchPath)
	if err != nil {
		t.Fatal(err)
	}
	contents = []byte(strings.Replace(string(contents), "-target", "-missing", 1))
	if err := os.WriteFile(patchPath, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	source := writeSource(t)
	err = Materialize(context.Background(), Config{Root: root, SourceRoot: source})
	if err == nil || !strings.Contains(err.Error(), "zero fuzz") {
		t.Fatalf("Materialize() error = %v", err)
	}
	after, err := os.ReadFile(filepath.Join(source, "src", "runtime", "proc.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != "alpha\ntarget\nomega\n" {
		t.Fatalf("failed materialization changed source = %q", after)
	}
}

func TestMaterializeRejectsFuzzDependentPatch(t *testing.T) {
	root := writeFixture(t)
	patchPath := filepath.Join(root, "gomad.patch")
	contents, err := os.ReadFile(patchPath)
	if err != nil {
		t.Fatal(err)
	}
	contents = []byte(strings.Replace(string(contents), " alpha", " absent", 1))
	if err := os.WriteFile(patchPath, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	source := writeSource(t)
	err = Materialize(context.Background(), Config{Root: root, SourceRoot: source})
	if err == nil || !strings.Contains(err.Error(), "zero fuzz") {
		t.Fatalf("Materialize() error = %v", err)
	}
	after, err := os.ReadFile(filepath.Join(source, "src", "runtime", "proc.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != "alpha\ntarget\nomega\n" {
		t.Fatalf("fuzz-dependent patch changed source = %q", after)
	}
}

func TestRegeneratePublishesDeterministicExactPatch(t *testing.T) {
	root, archive, candidate := writeRegenerateFixture(t)
	gofmt, err := exec.LookPath("gofmt")
	if err != nil {
		t.Fatal(err)
	}
	candidateFile := filepath.Join(candidate, "src", "runtime", "proc.go")
	if err := os.WriteFile(candidateFile, []byte("package runtime\n\nfunc replacement() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	first := filepath.Join(t.TempDir(), "first.patch")
	second := filepath.Join(t.TempDir(), "second.patch")
	for _, output := range []string{first, second} {
		if err := Regenerate(context.Background(), Config{
			Root: root, CandidateRoot: candidate, Archive: archive, Output: output, Gofmt: gofmt,
		}); err != nil {
			t.Fatal(err)
		}
	}
	firstContents, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	secondContents, err := os.ReadFile(second)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstContents, secondContents) {
		t.Fatal("Regenerate() output is not deterministic")
	}
	if err := Validate(Config{Root: root, Patch: first}); err != nil {
		t.Fatalf("regenerated patch is invalid: %v", err)
	}
	pristine := writeRegenerateSource(t, "go1.26.4\n", "package runtime\n\nfunc target() {}\n")
	if err := Materialize(context.Background(), Config{Root: root, Patch: first, SourceRoot: pristine}); err != nil {
		t.Fatal(err)
	}
	materialized, err := os.ReadFile(filepath.Join(pristine, "src", "runtime", "proc.go"))
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(candidateFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(materialized, want) {
		t.Fatalf("materialized source = %q, want %q", materialized, want)
	}
}

func TestRegenerateRejectsInvalidCandidatesWithoutReplacingOutput(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(t *testing.T, candidate string)
		want   string
	}{
		{
			name: "wrong-version",
			mutate: func(t *testing.T, candidate string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(candidate, "VERSION"), []byte("go1.26.3\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			want: "candidate must be go1.26.4",
		},
		{
			name: "added-file",
			mutate: func(t *testing.T, candidate string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(candidate, "added"), []byte("added"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			want: "adds a source path",
		},
		{
			name: "deleted-file",
			mutate: func(t *testing.T, candidate string) {
				t.Helper()
				if err := os.Remove(filepath.Join(candidate, "README")); err != nil {
					t.Fatal(err)
				}
			},
			want: "deletes a source path",
		},
		{
			name: "unlisted-change",
			mutate: func(t *testing.T, candidate string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(candidate, "README"), []byte("changed\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			want: "prohibited path",
		},
		{
			name: "special-entry",
			mutate: func(t *testing.T, candidate string) {
				t.Helper()
				if err := os.Symlink("README", filepath.Join(candidate, "link")); err != nil {
					t.Fatal(err)
				}
			},
			want: "non-regular entry",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			root, archive, candidate := writeRegenerateFixture(t)
			test.mutate(t, candidate)
			output := filepath.Join(t.TempDir(), "gomad.patch")
			if err := os.WriteFile(output, []byte("previous\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			err := Regenerate(context.Background(), Config{
				Root: root, CandidateRoot: candidate, Archive: archive, Output: output, Gofmt: "gofmt",
			})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Regenerate() error = %v, want %q", err, test.want)
			}
			contents, readErr := os.ReadFile(output)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if string(contents) != "previous\n" {
				t.Fatalf("failed regeneration replaced output with %q", contents)
			}
		})
	}
}

func TestRegenerateRejectsCandidateWithNoChanges(t *testing.T) {
	root, archive, candidate := writeRegenerateFixture(t)
	err := Regenerate(context.Background(), Config{
		Root: root, CandidateRoot: candidate, Archive: archive, Output: filepath.Join(t.TempDir(), "gomad.patch"), Gofmt: "gofmt",
	})
	if err == nil || !strings.Contains(err.Error(), "contains no changes") {
		t.Fatalf("Regenerate() error = %v", err)
	}
}

func TestRegenerateMatchesCheckedPatchForPinnedArchive(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(root, ".toolchain", "downloads", "go1.26.4.src.tar.gz")
	if _, err := os.Stat(archive); os.IsNotExist(err) {
		t.Skip("pinned Go source archive is not cached")
	} else if err != nil {
		t.Fatal(err)
	}
	extracted := filepath.Join(t.TempDir(), "source")
	if err := sourcearchive.Extract(context.Background(), archive, extracted); err != nil {
		t.Fatal(err)
	}
	candidate := filepath.Join(extracted, "go")
	if err := Materialize(context.Background(), Config{Root: root, SourceRoot: candidate}); err != nil {
		t.Fatal(err)
	}
	gofmt, err := exec.LookPath("gofmt")
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "generated.patch")
	if err := Regenerate(context.Background(), Config{
		Root: root, CandidateRoot: candidate, Archive: archive, Output: output, Gofmt: gofmt,
	}); err != nil {
		t.Fatal(err)
	}
	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	checked, err := os.ReadFile(filepath.Join(root, "go1.26.4.patch"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(generated, checked) {
		t.Fatal("regenerated pinned patch differs from checked patch")
	}
}

func writeFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "overlay", "src", "runtime"), 0o700); err != nil {
		t.Fatal(err)
	}
	descriptor := `{
  "schema_version": 1,
  "go_version": "go1.26.4",
  "archive": {"name":"go1.26.4.src.tar.gz","url":"https://go.dev/dl/go1.26.4.src.tar.gz","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
  "supported_platforms": ["darwin/arm64"],
  "boundary_manifest_version": "go1.26.4-darwin-arm64-v1",
  "patch": "gomad.patch",
  "adapters": [{"module":"modernc.org/libc","version":"v1.72.3","sum":"h1:test"}],
  "patch_allowlist": ["src/runtime/proc.go"],
  "overlay_allowlist": ["src/runtime/gomad.go"]
}
`
	patch := `diff --git a/src/runtime/proc.go b/src/runtime/proc.go
--- a/src/runtime/proc.go
+++ b/src/runtime/proc.go
@@ -1,3 +1,3 @@
 alpha
-target
+replacement
 omega
`
	if err := os.WriteFile(filepath.Join(root, "version.json"), []byte(descriptor), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "gomad.patch"), []byte(patch), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "overlay", "src", "runtime", "gomad.go"), []byte("package runtime\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func writeSource(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "src", "runtime", "proc.go")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("alpha\ntarget\nomega\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func writeRegenerateFixture(t *testing.T) (string, string, string) {
	t.Helper()
	archiveContents := regenerateArchive(t)
	digest := sha256.Sum256(archiveContents)
	root := writeFixture(t)
	descriptorPath := filepath.Join(root, "version.json")
	descriptor, err := os.ReadFile(descriptorPath)
	if err != nil {
		t.Fatal(err)
	}
	descriptor = []byte(strings.Replace(string(descriptor), strings.Repeat("a", 64), fmt.Sprintf("%x", digest), 1))
	if err := os.WriteFile(descriptorPath, descriptor, 0o600); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(t.TempDir(), "go1.26.4.src.tar.gz")
	if err := os.WriteFile(archivePath, archiveContents, 0o600); err != nil {
		t.Fatal(err)
	}
	candidate := writeRegenerateSource(t, "go1.26.4\n", "package runtime\n\nfunc target() {}\n")
	return root, archivePath, candidate
}

func writeRegenerateSource(t *testing.T, version, source string) string {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "src", "runtime", "proc.go")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	for name, contents := range map[string]string{"VERSION": version, "README": "fixture\n", "src/runtime/proc.go": source} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func regenerateArchive(t *testing.T) []byte {
	t.Helper()
	var output bytes.Buffer
	zipper := gzip.NewWriter(&output)
	archive := tar.NewWriter(zipper)
	for _, entry := range []struct {
		name     string
		contents string
		mode     int64
		dir      bool
	}{
		{name: "go/", mode: 0o755, dir: true},
		{name: "go/src/", mode: 0o755, dir: true},
		{name: "go/src/runtime/", mode: 0o755, dir: true},
		{name: "go/VERSION", contents: "go1.26.4\n", mode: 0o644},
		{name: "go/README", contents: "fixture\n", mode: 0o644},
		{name: "go/src/runtime/proc.go", contents: "package runtime\n\nfunc target() {}\n", mode: 0o644},
	} {
		typeFlag := byte(tar.TypeReg)
		if entry.dir {
			typeFlag = tar.TypeDir
		}
		header := &tar.Header{Name: entry.name, Mode: entry.mode, Size: int64(len(entry.contents)), Typeflag: typeFlag}
		if entry.dir {
			header.Size = 0
		}
		if err := archive.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if !entry.dir {
			if _, err := archive.Write([]byte(entry.contents)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := zipper.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}
