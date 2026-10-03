package target

import (
	"context"
	"debug/buildinfo"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/toolchain/installation"
)

func TestPrepareRestoresCachedTargetForIdenticalInputs(t *testing.T) {
	cacheRoot := isolatePreparedTargetCache(t)
	module := writeEmbedModule(t, "one")
	first := prepareEmbedModule(t, module, nil)
	entries := preparedTargetEntries(t, cacheRoot)
	if len(entries) != 1 {
		t.Fatalf("prepared target entries = %v, want one", entries)
	}
	recordPath := filepath.Join(cacheRoot, entries[0], "record.json")
	past := time.Unix(1_000_000_000, 0)
	if err := os.Chtimes(recordPath, past, past); err != nil {
		t.Fatal(err)
	}
	second := prepareEmbedModule(t, module, nil)
	if second.SHA256 != first.SHA256 || second.Size != first.Size || second.Path == first.Path {
		t.Fatalf("restored target = %#v, want the identity of %#v", second, first)
	}
	if entries := preparedTargetEntries(t, cacheRoot); len(entries) != 1 {
		t.Fatalf("prepared target entries after restore = %v, want one", entries)
	}
	info, err := os.Stat(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().After(past) {
		t.Fatal("restore did not mark the entry as recently used")
	}
	if got := runEmbedTarget(t, second); got != "one\n" {
		t.Fatalf("restored target output = %q", got)
	}
}

func TestPrepareRebuildsWhenAnyBuildInputChanges(t *testing.T) {
	cacheRoot := isolatePreparedTargetCache(t)
	module := writeEmbedModule(t, "one")
	base := prepareEmbedModule(t, module, nil)
	for _, test := range []struct {
		name       string
		change     func(t *testing.T)
		tags       []string
		wantOutput string
	}{
		{name: "embedded file", change: func(t *testing.T) { writeFile(t, filepath.Join(module, "data.txt"), "two") }, wantOutput: "two\n"},
		{name: "source file", change: func(t *testing.T) {
			writeFile(t, filepath.Join(module, "main.go"), embedMain("3"))
		}, wantOutput: "3two\n"},
		{name: "build tags", tags: []string{"other"}, wantOutput: "3two\n"},
		{name: "module file", change: func(t *testing.T) {
			writeFile(t, filepath.Join(module, "go.mod"), "module example.com/target\n\ngo 1.26.4\n\n// changed\n")
		}, tags: []string{"other"}, wantOutput: "3two\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			before := len(preparedTargetEntries(t, cacheRoot))
			if test.change != nil {
				test.change(t)
			}
			prepared := prepareEmbedModule(t, module, test.tags)
			if got := len(preparedTargetEntries(t, cacheRoot)); got != before+1 {
				t.Fatalf("prepared target entries = %d, want %d", got, before+1)
			}
			if got := runEmbedTarget(t, prepared); got != test.wantOutput {
				t.Fatalf("rebuilt target output = %q, want %q", got, test.wantOutput)
			}
		})
	}
	if again := prepareEmbedModule(t, module, nil); again.SHA256 == base.SHA256 {
		t.Fatal("changed inputs reproduced the original binary")
	}
}

func TestPrepareRebuildsWhenADependencyLanguageVersionChanges(t *testing.T) {
	isolatePreparedTargetCache(t)
	module := writeModule(t, map[string]string{
		"go.mod":     "module example.com/target\n\ngo 1.26.4\n\nrequire example.com/dep v0.0.0\n\nreplace example.com/dep => ./dep\n",
		"main.go":    "package main\n\nimport (\n\t\"fmt\"\n\n\t\"example.com/dep\"\n)\n\nfunc main() { fmt.Println(dep.Values()) }\n",
		"dep/go.mod": "module example.com/dep\n\ngo 1.21\n",
		// Before Go 1.22 the closures share one loop variable.
		"dep/dep.go": "package dep\n\nimport \"fmt\"\n\nfunc Values() string {\n\tvar values []func() int\n\tfor i := 0; i < 3; i++ {\n\t\tvalues = append(values, func() int { return i })\n\t}\n\treturn fmt.Sprint(values[0](), values[1](), values[2]())\n}\n",
	})
	if got := runEmbedTarget(t, prepareEmbedModule(t, module, nil)); got != "3 3 3\n" {
		t.Fatalf("go 1.21 dependency output = %q", got)
	}
	writeFile(t, filepath.Join(module, "dep", "go.mod"), "module example.com/dep\n\ngo 1.22\n")
	if got := runEmbedTarget(t, prepareEmbedModule(t, module, nil)); got != "0 1 2\n" {
		t.Fatalf("go 1.22 dependency output = %q, want the rebuilt per-iteration loop variable", got)
	}
}

func TestPrepareStampsNoRepositoryState(t *testing.T) {
	isolatePreparedTargetCache(t)
	module := writeEmbedModule(t, "one")
	for _, arguments := range [][]string{{"init", "-q"}, {"add", "."}, {"-c", "user.name=gomad", "-c", "user.email=gomad@example.com", "commit", "-q", "-m", "fixture"}} {
		command := exec.Command("git", arguments...)
		command.Dir = module
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", arguments, err, output)
		}
	}
	info, err := buildinfo.ReadFile(prepareEmbedModule(t, module, nil).Path)
	if err != nil {
		t.Fatal(err)
	}
	for _, setting := range info.Settings {
		if strings.HasPrefix(setting.Key, "vcs") {
			t.Fatalf("prepared target records repository state %s=%s", setting.Key, setting.Value)
		}
	}
}

func TestPrepareDiscardsCachedTargetThatDoesNotHashToItsRecord(t *testing.T) {
	cacheRoot := isolatePreparedTargetCache(t)
	module := writeEmbedModule(t, "one")
	first := prepareEmbedModule(t, module, nil)
	entry := filepath.Join(cacheRoot, preparedTargetEntries(t, cacheRoot)[0])
	if err := os.Chmod(filepath.Join(entry, "target"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(entry, "target"), []byte("#!/bin/sh\necho poisoned\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	second := prepareEmbedModule(t, module, nil)
	if second.SHA256 != first.SHA256 {
		t.Fatalf("rebuilt target = %s, want %s", second.SHA256, first.SHA256)
	}
	if got := runEmbedTarget(t, second); got != "one\n" {
		t.Fatalf("rebuilt target output = %q", got)
	}
	hash, _, err := hashRegularFile(filepath.Join(entry, "target"))
	if err != nil || hash != first.SHA256 {
		t.Fatalf("republished entry hash = %s, %v; want %s", hash, err, first.SHA256)
	}
}

func TestPreparedTargetCacheEvictsLeastRecentlyUsedEntriesPastTheByteBound(t *testing.T) {
	cacheRoot := isolatePreparedTargetCache(t)
	module := writeEmbedModule(t, "one")
	first := prepareEmbedModule(t, module, nil)
	previous := maximumPreparedTargetBytes
	// Two binaries fit; a third pushes the least recently used one out.
	maximumPreparedTargetBytes = 2*first.Size + first.Size/2
	t.Cleanup(func() { maximumPreparedTargetBytes = previous })
	for index, value := range []string{"one", "two", "three"} {
		writeFile(t, filepath.Join(module, "data.txt"), value)
		prepareEmbedModule(t, module, nil)
		entries := preparedTargetEntries(t, cacheRoot)
		if want := min(index+1, 2); len(entries) != want {
			t.Fatalf("after %s: prepared target entries = %v, want %d", value, entries, want)
		}
		for _, entry := range entries {
			past := time.Unix(1_000_000_000+int64(index), 0)
			if err := os.Chtimes(filepath.Join(cacheRoot, entry, "record.json"), past, past); err != nil {
				t.Fatal(err)
			}
		}
	}
	if got := runEmbedTarget(t, prepareEmbedModule(t, module, nil)); got != "three\n" {
		t.Fatalf("newest target output = %q", got)
	}
}

func isolatePreparedTargetCache(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	previous := preparedTargetCacheRoot
	preparedTargetCacheRoot = func(installation.Description) string { return root }
	t.Cleanup(func() { preparedTargetCacheRoot = previous })
	return root
}

func preparedTargetEntries(t *testing.T, root string) []string {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() && entry.Name()[0] != '.' {
			names = append(names, entry.Name())
		}
	}
	return names
}

func embedMain(prefix string) string {
	return "package main\n\nimport (\n\t_ \"embed\"\n\t\"fmt\"\n)\n\n//go:embed data.txt\nvar data string\n\nfunc main() { fmt.Println(\"" + prefix + "\" + data) }\n"
}

func writeEmbedModule(t *testing.T, data string) string {
	t.Helper()
	return writeModule(t, map[string]string{
		"go.mod":   "module example.com/target\n\ngo 1.26.4\n",
		"main.go":  embedMain(""),
		"data.txt": data,
	})
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func prepareEmbedModule(t *testing.T, module string, tags []string) Prepared {
	t.Helper()
	prepared, err := Prepare(context.Background(), Spec{
		Kind: KindGoRun, Source: ".", WorkingDir: module, PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot(t), BuildTags: tags,
	})
	if err != nil {
		t.Fatal(err)
	}
	return prepared
}

func runEmbedTarget(t *testing.T, prepared Prepared) string {
	t.Helper()
	command := exec.Command(prepared.Path)
	command.Args[0] = prepared.Argv[0]
	command.Env = []string{"GOMADSEED=1", "TZ=UTC"}
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("run prepared target: %v: %s", err, output)
	}
	return string(output)
}

// TestMain keeps every preparation the package's tests make out of the
// developer's prepared-target cache, whose eviction bound they would
// otherwise churn.
func TestMain(m *testing.M) {
	root, err := os.MkdirTemp("", "gomad3-prepared-targets-")
	if err != nil {
		panic(err)
	}
	preparedTargetCacheRoot = func(installation.Description) string { return root }
	status := m.Run()
	if err := os.RemoveAll(root); err != nil {
		panic(err)
	}
	os.Exit(status)
}
