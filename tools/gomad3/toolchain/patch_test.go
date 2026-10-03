package toolchain

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

	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

func TestValidateAcceptsCurrentCheckedInputs(t *testing.T) {
	root, err := filepath.Abs(filepath.Join(".."))
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidatePatch(PatchSpec{Root: root}); err != nil {
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
	if err := os.WriteFile(filepath.Join(root, "toolchain", "runtime", "gomad.patch"), []byte(patch), 0o600); err != nil {
		t.Fatal(err)
	}
	err := ValidatePatch(PatchSpec{Root: root})
	if err == nil || !strings.Contains(err.Error(), "prohibited") {
		t.Fatalf("ValidatePatch() error = %v", err)
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
			if err := os.WriteFile(filepath.Join(root, "toolchain", "runtime", "gomad.patch"), []byte(patch), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := ValidatePatch(PatchSpec{Root: root}); err == nil {
				t.Fatal("ValidatePatch() accepted invalid patch")
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
			path := filepath.Join(root, "toolchain", "runtime", "overlay", filepath.FromSlash(test.relative))
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, test.contents, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := ValidatePatch(PatchSpec{Root: root}); err == nil {
				t.Fatal("ValidatePatch() accepted invalid overlay")
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
			path := filepath.Join(root, "toolchain", "runtime", "overlay", filepath.FromSlash(test.path))
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("package runtime\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			err := ValidatePatch(PatchSpec{Root: root})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ValidatePatch() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestMaterializeAppliesExactPatch(t *testing.T) {
	root := writeFixture(t)
	source := writeSource(t)
	if err := MaterializePatch(context.Background(), PatchSpec{Root: root, SourceRoot: source}); err != nil {
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
	patchPath := filepath.Join(root, "toolchain", "runtime", "gomad.patch")
	contents, err := os.ReadFile(patchPath)
	if err != nil {
		t.Fatal(err)
	}
	contents = []byte(strings.Replace(string(contents), "-target", "-missing", 1))
	if err := os.WriteFile(patchPath, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	source := writeSource(t)
	err = MaterializePatch(context.Background(), PatchSpec{Root: root, SourceRoot: source})
	if err == nil || !strings.Contains(err.Error(), "zero fuzz") {
		t.Fatalf("MaterializePatch() error = %v", err)
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
	patchPath := filepath.Join(root, "toolchain", "runtime", "gomad.patch")
	contents, err := os.ReadFile(patchPath)
	if err != nil {
		t.Fatal(err)
	}
	contents = []byte(strings.Replace(string(contents), " alpha", " absent", 1))
	if err := os.WriteFile(patchPath, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	source := writeSource(t)
	err = MaterializePatch(context.Background(), PatchSpec{Root: root, SourceRoot: source})
	if err == nil || !strings.Contains(err.Error(), "zero fuzz") {
		t.Fatalf("MaterializePatch() error = %v", err)
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
		if err := RegeneratePatch(context.Background(), PatchSpec{
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
		t.Fatal("RegeneratePatch() output is not deterministic")
	}
	if err := ValidatePatch(PatchSpec{Root: root, Patch: first}); err != nil {
		t.Fatalf("regenerated patch is invalid: %v", err)
	}
	pristine := writeRegenerateSource(t, "go1.26.4\n", "package runtime\n\nfunc target() {}\n")
	if err := MaterializePatch(context.Background(), PatchSpec{Root: root, Patch: first, SourceRoot: pristine}); err != nil {
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
			err := RegeneratePatch(context.Background(), PatchSpec{
				Root: root, CandidateRoot: candidate, Archive: archive, Output: output, Gofmt: "gofmt",
			})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("RegeneratePatch() error = %v, want %q", err, test.want)
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
	err := RegeneratePatch(context.Background(), PatchSpec{
		Root: root, CandidateRoot: candidate, Archive: archive, Output: filepath.Join(t.TempDir(), "gomad.patch"), Gofmt: "gofmt",
	})
	if err == nil || !strings.Contains(err.Error(), "contains no changes") {
		t.Fatalf("RegeneratePatch() error = %v", err)
	}
}

func TestRegenerateEmitsOneContextLine(t *testing.T) {
	const source = "package runtime\n\nfunc a() {}\n\nfunc b() {}\n\nfunc target() {}\n\nfunc c() {}\n\nfunc d() {}\n"
	root, archive, candidate := writeRegenerateFixtureWithSource(t, source)
	gofmt, err := exec.LookPath("gofmt")
	if err != nil {
		t.Fatal(err)
	}
	candidateFile := filepath.Join(candidate, "src", "runtime", "proc.go")
	if err := os.WriteFile(candidateFile, []byte(strings.Replace(source, "func target() {}", "func replacement() {}", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name         string
		contextLines int
		hunk         string
	}{
		{name: "canonical", contextLines: canonicalPatchContext, hunk: "@@ -6,3 +6,3 @@ func b() {}\n \n-func target() {}\n+func replacement() {}\n \n"},
		{name: "three", contextLines: 3, hunk: "@@ -4,7 +4,7 @@ func a() {}\n \n func b() {}\n \n-func target() {}\n+func replacement() {}\n \n func c() {}\n \n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			output := filepath.Join(t.TempDir(), "gomad.patch")
			if err := regeneratePatch(context.Background(), PatchSpec{
				Root: root, CandidateRoot: candidate, Archive: archive, Output: output, Gofmt: gofmt,
			}, test.contextLines); err != nil {
				t.Fatal(err)
			}
			patch, err := os.ReadFile(output)
			if err != nil {
				t.Fatal(err)
			}
			if _, hunk, found := strings.Cut(string(patch), "@@ "); !found || "@@ "+hunk != test.hunk {
				t.Fatalf("regenerated hunk = %q, want %q", "@@ "+hunk, test.hunk)
			}
			pristine := writeRegenerateSource(t, "go1.26.4\n", source)
			if err := MaterializePatch(context.Background(), PatchSpec{Root: root, Patch: output, SourceRoot: pristine}); err != nil {
				t.Fatal(err)
			}
			requireSameFile(t, filepath.Join(pristine, "src", "runtime", "proc.go"), candidateFile)
		})
	}
}

func TestRegenerateMatchesCheckedPatchForPinnedArchive(t *testing.T) {
	root, archive, descriptor := pinnedArchive(t)
	candidate := materializePinnedSource(t, root, archive, "")
	first := regeneratePinnedPatch(t, root, archive, candidate, canonicalPatchContext)
	second := regeneratePinnedPatch(t, root, archive, candidate, canonicalPatchContext)
	generated, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := os.ReadFile(second)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(generated, repeated) {
		t.Fatal("repeated pinned regeneration is not byte-identical")
	}
	checked, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(descriptor.Patch)))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(generated, checked) {
		t.Fatal("regenerated pinned patch differs from checked patch")
	}
}

func TestPinnedArchiveFollowsDescriptorAndRejectsChecksumMismatch(t *testing.T) {
	root := writeFixture(t)
	if _, descriptor, cached, err := cachedPinnedArchive(root); err != nil || cached || descriptor.Archive.Name != "go1.26.4.src.tar.gz" {
		t.Fatalf("uncached archive = %q, %t, %v", descriptor.Archive.Name, cached, err)
	}
	downloads := filepath.Join(root, ".toolchain", "downloads")
	if err := os.MkdirAll(downloads, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(downloads, "go1.26.4.src.tar.gz"), []byte("not the pinned archive"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, cached, err := cachedPinnedArchive(root); err == nil || cached || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("mismatched archive cached = %t, error = %v", cached, err)
	}
}

func TestPinnedContextRepresentationsMaterializeIdenticalSource(t *testing.T) {
	root, archive, descriptor := pinnedArchive(t)
	candidate := materializePinnedSource(t, root, archive, "")
	three := regeneratePinnedPatch(t, root, archive, candidate, 3)
	one := regeneratePinnedPatch(t, root, archive, candidate, canonicalPatchContext)
	threeContents, err := os.ReadFile(three)
	if err != nil {
		t.Fatal(err)
	}
	oneContents, err := os.ReadFile(one)
	if err != nil {
		t.Fatal(err)
	}
	if len(oneContents) >= len(threeContents) {
		t.Fatalf("one-context patch is %d bytes, three-context patch is %d bytes", len(oneContents), len(threeContents))
	}
	requireHunkContext(t, oneContents, canonicalPatchContext)
	fromThree := materializePinnedSource(t, root, archive, three)
	fromOne := materializePinnedSource(t, root, archive, one)
	for _, path := range descriptor.PatchAllowlist {
		requireSameFile(t, filepath.Join(fromThree, filepath.FromSlash(path)), filepath.Join(candidate, filepath.FromSlash(path)))
		requireSameFile(t, filepath.Join(fromOne, filepath.FromSlash(path)), filepath.Join(candidate, filepath.FromSlash(path)))
	}
}

// pinnedArchive returns the module root, the cached source archive the
// descriptor pins, and the descriptor. It skips only when that archive is not
// cached; a cached archive with another checksum fails.
func pinnedArchive(t *testing.T) (string, string, gomadversion.Descriptor) {
	t.Helper()
	root, err := filepath.Abs(filepath.Join(".."))
	if err != nil {
		t.Fatal(err)
	}
	archive, descriptor, cached, err := cachedPinnedArchive(root)
	if err != nil {
		t.Fatal(err)
	}
	if !cached {
		t.Skipf("pinned Go source archive %s is not cached", descriptor.Archive.Name)
	}
	return root, archive, descriptor
}

func cachedPinnedArchive(root string) (string, gomadversion.Descriptor, bool, error) {
	descriptor, err := gomadversion.Load(root)
	if err != nil {
		return "", gomadversion.Descriptor{}, false, err
	}
	archive := filepath.Join(root, ".toolchain", "downloads", descriptor.Archive.Name)
	if _, err := os.Stat(archive); os.IsNotExist(err) {
		return archive, descriptor, false, nil
	} else if err != nil {
		return "", descriptor, false, err
	}
	digest, err := FileSHA256(archive)
	if err != nil {
		return "", descriptor, false, err
	}
	if digest != descriptor.Archive.SHA256 {
		return "", descriptor, false, fmt.Errorf("cached %s checksum = %s, want %s", descriptor.Archive.Name, digest, descriptor.Archive.SHA256)
	}
	return archive, descriptor, true, nil
}

// materializePinnedSource extracts the pinned archive and applies patch, or the
// checked patch when patch is empty.
func materializePinnedSource(t *testing.T, root, archive, patch string) string {
	t.Helper()
	extracted := filepath.Join(t.TempDir(), "source")
	if err := ExtractSource(context.Background(), archive, extracted); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(extracted, "go")
	if err := MaterializePatch(context.Background(), PatchSpec{Root: root, Patch: patch, SourceRoot: source}); err != nil {
		t.Fatal(err)
	}
	return source
}

func regeneratePinnedPatch(t *testing.T, root, archive, candidate string, contextLines int) string {
	t.Helper()
	gofmt, err := exec.LookPath("gofmt")
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "generated.patch")
	if err := regeneratePatch(context.Background(), PatchSpec{
		Root: root, CandidateRoot: candidate, Archive: archive, Output: output, Gofmt: gofmt,
	}, contextLines); err != nil {
		t.Fatal(err)
	}
	return output
}

// requireHunkContext checks that no hunk carries more than contextLines
// unchanged lines before its first change or after its last change.
func requireHunkContext(t *testing.T, patch []byte, contextLines int) {
	t.Helper()
	var hunk []string
	check := func() {
		first, last := -1, -1
		for index, line := range hunk {
			if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") {
				if first < 0 {
					first = index
				}
				last = index
			}
		}
		if first < 0 || first > contextLines || len(hunk)-1-last > contextLines {
			t.Fatalf("hunk has more than %d context lines: %q", contextLines, hunk)
		}
	}
	inHunk := false
	for _, line := range strings.Split(strings.TrimSuffix(string(patch), "\n"), "\n") {
		switch {
		case strings.HasPrefix(line, "@@ "):
			if inHunk {
				check()
			}
			hunk, inHunk = nil, true
		case strings.HasPrefix(line, "diff --git "):
			if inHunk {
				check()
			}
			inHunk = false
		case inHunk && !strings.HasPrefix(line, `\`):
			hunk = append(hunk, line)
		}
	}
	if inHunk {
		check()
	}
}

func requireSameFile(t *testing.T, got, want string) {
	t.Helper()
	gotContents, err := os.ReadFile(got)
	if err != nil {
		t.Fatal(err)
	}
	wantContents, err := os.ReadFile(want)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotContents, wantContents) {
		t.Fatalf("%s differs from %s", got, want)
	}
}

func writeFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, directory := range []string{"toolchain/runtime/overlay/src/runtime", "toolchain/version"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(directory)), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	descriptor := `{
  "schema_version": 1,
  "go_version": "go1.26.4",
  "archive": {"name":"go1.26.4.src.tar.gz","url":"https://go.dev/dl/go1.26.4.src.tar.gz","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
  "supported_platforms": ["darwin/arm64"],
  "boundary_manifest_version": "go1.26.4-darwin-arm64-v1",
  "patch": "toolchain/runtime/gomad.patch",
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
	if err := os.WriteFile(filepath.Join(root, "toolchain", "version", "version.json"), []byte(descriptor), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "toolchain", "runtime", "gomad.patch"), []byte(patch), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "toolchain", "runtime", "overlay", "src", "runtime", "gomad.go"), []byte("package runtime\n"), 0o600); err != nil {
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
	return writeRegenerateFixtureWithSource(t, "package runtime\n\nfunc target() {}\n")
}

func writeRegenerateFixtureWithSource(t *testing.T, source string) (string, string, string) {
	t.Helper()
	archiveContents := regenerateArchive(t, source)
	digest := sha256.Sum256(archiveContents)
	root := writeFixture(t)
	descriptorPath := filepath.Join(root, "toolchain", "version", "version.json")
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
	candidate := writeRegenerateSource(t, "go1.26.4\n", source)
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

func regenerateArchive(t *testing.T, source string) []byte {
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
		{name: "go/src/runtime/proc.go", contents: source, mode: 0o644},
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
