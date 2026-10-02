package deterministicio

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/target"
	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

func TestPinnedModerncMemoryModuleInventory(t *testing.T) {
	moduleRoot := filepath.Join(pinnedModuleCache(t), "modernc.org", "memory@v1.11.0")
	got, err := target.DigestAdapterSourceInventory(moduleRoot)
	if err != nil {
		t.Fatal(err)
	}
	if got != memoryOriginalSourceInventorySHA256 {
		t.Fatalf("modernc memory module inventory = %q, want %q", got, memoryOriginalSourceInventorySHA256)
	}
}

func TestRewriteModerncMemoryModelsOnlyAnonymousAllocatorMappings(t *testing.T) {
	source := readPinnedModerncMemorySource(t)
	rewritten, err := rewriteAdapterSource(memoryModulePath, memoryRewrites[0], source)
	if err != nil {
		t.Fatal(err)
	}
	for _, retained := range []string{"return unix.MunmapPtr", "unix.MmapPtr(-1", "Ask for more so we can align"} {
		if !strings.Contains(string(rewritten), retained) {
			t.Fatalf("rewritten mmap_unix.go omitted %q", retained)
		}
	}
	for _, modeled := range []string{"gomadMemoryEnabled()", "gomadMemoryMap(uintptr(size), pageSize)", "gomadMemoryUnmap(addr, uintptr(size))"} {
		if !strings.Contains(string(rewritten), modeled) {
			t.Fatalf("rewritten mmap_unix.go omitted %q", modeled)
		}
	}
	if got := digestBytes(rewritten); got != memoryMmapReplacementSHA256 {
		t.Fatalf("rewritten mmap_unix.go digest = %q, want %q", got, memoryMmapReplacementSHA256)
	}
}

func TestRewriteModerncMemoryRejectsSourceIdentityDrift(t *testing.T) {
	if _, err := rewriteAdapterSource(memoryModulePath, memoryRewrites[0], append(readPinnedModerncMemorySource(t), '\n')); err == nil {
		t.Fatal("rewriteAdapterSource() accepted changed mmap_unix.go")
	}
}

func TestModerncMemoryRewriteRejectsAnchorDrift(t *testing.T) {
	source := readPinnedModerncMemorySource(t)
	for _, test := range []struct {
		name, want string
		change     func(*sourceRewrite)
	}{
		{name: "missing-anchor", want: "anchor mismatch", change: func(rewrite *sourceRewrite) { rewrite.rewrites[0].anchor = []byte("absent page size anchor") }},
		{name: "ambiguous-anchor", want: "anchor mismatch", change: func(rewrite *sourceRewrite) { rewrite.rewrites[0].anchor = []byte("osPageSize") }},
		{name: "replacement-digest", want: "replacement identity mismatch", change: func(rewrite *sourceRewrite) { rewrite.replacementSHA256 = memoryMmapSourceSHA256 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			rewrite := memoryRewrites[0]
			rewrite.rewrites = append([]anchorRewrite(nil), rewrite.rewrites...)
			test.change(&rewrite)
			if _, err := rewriteAdapterSource(memoryModulePath, rewrite, source); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("modernc memory rewrite: %v, want %s", err, test.want)
			}
		})
	}
}

func TestPrepareModerncMemoryRejectsModuleDrift(t *testing.T) {
	identity := gomadversion.AdapterIdentity{Module: memoryModulePath, Version: memoryVersion, Sum: memorySum}
	pinned := filepath.Join(pinnedModuleCache(t), "modernc.org", "memory@"+memoryVersion)
	for _, test := range []struct {
		name, want, wantRead string
		change               func(t *testing.T, path string, contents []byte)
	}{
		{name: "changed-source", want: "source inventory identity mismatch", change: func(t *testing.T, path string, contents []byte) {
			if err := os.WriteFile(path, append(contents, '\n'), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "non-regular-source", want: "symbolic link", wantRead: "not a regular file", change: func(t *testing.T, path string, contents []byte) {
			outside := filepath.Join(t.TempDir(), memoryMmapPath)
			if err := os.WriteFile(outside, contents, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, path); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			moduleCache := t.TempDir()
			moduleRoot := filepath.Join(moduleCache, "modernc.org", "memory@"+memoryVersion)
			if err := os.MkdirAll(filepath.Dir(moduleRoot), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := copyAdapterModule(pinned, moduleRoot, nil, defaultAdapterCopyLimits); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(moduleRoot, memoryMmapPath)
			contents, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			test.change(t, path, contents)
			if _, err := prepareModerncMemory(moduleCache, t.TempDir(), identity); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("changed modernc memory module: %v, want %s", err, test.want)
			}
			if test.wantRead == "" {
				return
			}
			if _, err := readAdapterSource(memoryModulePath, moduleRoot, memoryMmapPath); err == nil || !strings.Contains(err.Error(), test.wantRead) {
				t.Fatalf("changed modernc memory source: %v, want %s", err, test.wantRead)
			}
		})
	}
}

func TestModerncMemoryRejectsChangedReplacementInventory(t *testing.T) {
	identity := gomadversion.AdapterIdentity{Module: memoryModulePath, Version: memoryVersion, Sum: memorySum}
	_, err := prepareRewrittenModule(pinnedModuleCache(t), t.TempDir(), identity, rewrittenModule{
		module: memoryModulePath, version: memoryVersion, sum: memorySum,
		cacheElements:              []string{"modernc.org", "memory@" + memoryVersion},
		replacementDirectory:       "modernc-memory",
		originalInventorySHA256:    memoryOriginalSourceInventorySHA256,
		replacementInventorySHA256: memoryOriginalSourceInventorySHA256,
		preparedPackage:            memoryModulePath,
		preparedSourceSetSHA256:    memoryPreparedSourceSetSHA256,
		rewrites:                   memoryRewrites,
	})
	if err == nil || !strings.Contains(err.Error(), "replacement inventory identity mismatch") {
		t.Fatalf("changed modernc memory replacement inventory: %v", err)
	}
}

func TestPrepareModerncMemoryRecordsExactPrivateReplacement(t *testing.T) {
	identity := gomadversion.AdapterIdentity{Module: memoryModulePath, Version: memoryVersion, Sum: memorySum}
	prepared, err := prepareModerncMemory(pinnedModuleCache(t), t.TempDir(), identity)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.replacement != prepared.evidence.ReplacementRoot || prepared.evidence.Module != identity.Module || prepared.evidence.Version != identity.Version || prepared.evidence.Sum != identity.Sum {
		t.Fatalf("prepared adapter = %#v", prepared)
	}
	if prepared.evidence.PreparedPackage != memoryModulePath || prepared.evidence.PreparedSourceSetSHA256 != memoryPreparedSourceSetSHA256 {
		t.Fatalf("prepared package evidence = %#v", prepared.evidence)
	}
	if prepared.evidence.SourceSHA256 != memoryMmapSourceSHA256 || prepared.evidence.ReplacementSHA256 != memoryMmapReplacementSHA256 || prepared.evidence.OriginalSourceInventorySHA256 != memoryOriginalSourceInventorySHA256 || prepared.evidence.ReplacementSourceInventorySHA256 != memoryReplacementSourceInventorySHA256 {
		t.Fatalf("prepared identity evidence = %#v", prepared.evidence)
	}
	contents, err := os.ReadFile(prepared.evidence.Replacement)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contents), "gomadMemoryMap") || !strings.Contains(string(contents), "unix.MmapPtr(-1") {
		t.Fatalf("replacement source = %s", contents)
	}
}

func TestPrepareModerncMemoryRejectsChangedIdentity(t *testing.T) {
	for _, identity := range []gomadversion.AdapterIdentity{
		{Module: memoryModulePath, Version: "v1.11.1", Sum: "h1:changed"},
		{Module: memoryModulePath, Version: "v1.11.1", Sum: memorySum},
		{Module: memoryModulePath, Version: memoryVersion, Sum: "h1:changed"},
	} {
		if _, err := prepareModerncMemory(pinnedModuleCache(t), t.TempDir(), identity); err == nil {
			t.Fatalf("prepareModerncMemory() accepted a changed identity %#v", identity)
		}
	}
}

func TestModerncMemoryPreparedPackageSourceSetIdentity(t *testing.T) {
	workingDirectory := t.TempDir()
	moduleFile := "module modernc.org/memory/adaptertest\n\ngo 1.26.4\n\nrequire (\n\tgolang.org/x/sys v0.47.0 // indirect\n\tmodernc.org/memory v1.11.0\n)\n"
	sumFile := "github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec h1:W09IVJc94icq4NjY3clb7Lk8O1qJ8BdBEF8z0ibU0rE=\ngithub.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec/go.mod h1:qqbHyh8v60DhA7CoWK5oRCqLrMHRGoxYCSS9EjAz6Eo=\ngolang.org/x/sys v0.47.0 h1:o7XGOvZQCADBQQ4Y7VNq2dRWQR7JmOUW8Kxx4ZsNgWs=\ngolang.org/x/sys v0.47.0/go.mod h1:4GL1E5IUh+htKOUEOaiffhrAeqysfVGipDYzABqnCmw=\nmodernc.org/mathutil v1.7.1 h1:GCZVGXdaN8gTqB1Mf/usp1Y/hSqgI2vAGGP4jZMCxOU=\nmodernc.org/mathutil v1.7.1/go.mod h1:4p5IwJITfppl0G4sUEDtCr4DthTaT47/N3aT6MhfgJg=\nmodernc.org/memory v1.11.0 h1:o4QC8aMQzmcwCK3t3Ux/ZHmwFPzE6hf2Y5LbkRs+hbI=\nmodernc.org/memory v1.11.0/go.mod h1:/JP4VbVC+K5sU2wZi9bHoq2MAkCnrt2r98UGeSK7Mjw=\n"
	for name, contents := range map[string]string{
		"go.mod":  moduleFile,
		"go.sum":  sumFile,
		"main.go": "package main\n\nimport _ \"modernc.org/memory\"\n\nfunc main() {}\n",
	} {
		if err := os.WriteFile(filepath.Join(workingDirectory, name), []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	toolchainRoot, err := filepath.Abs(filepath.Join("..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	spec, adapters, err := Default().PrepareBuildAdapters(target.Spec{
		Kind: target.KindGoRun, Source: ".", WorkingDir: workingDirectory,
		PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot,
	}, pinnedModuleCache(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(adapters) != 1 || adapters[0].Module != memoryModulePath {
		t.Fatalf("selected adapters = %#v", adapters)
	}
	if _, err := target.ReviewCapabilities(context.Background(), spec); err != nil {
		t.Fatal(err)
	}
}

func readPinnedModerncMemorySource(t *testing.T) []byte {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(pinnedModuleCache(t), "modernc.org", "memory@v1.11.0", memoryMmapPath))
	if err != nil {
		t.Fatal(err)
	}
	return contents
}
