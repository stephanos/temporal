package backend

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"go.temporal.io/server/tools/gomad3/target"
)

func TestAssemblyHeaderChangesInvalidateCompilationCache(t *testing.T) {
	for _, test := range []struct {
		name, directive string
		nested, shadow  bool
	}{
		{"selected-header", "#include \"value.h\"", false, false},
		{"nested-include", "#include \"value.h\"", true, false},
		{"adjacent-include", "#include\"value.h\"", true, false},
		{"comment-include", "#/**/include \"value.h\"", true, false},
		{"local-generated-header-shadow", "#include \"go_asm.h\"", true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			provider := integrationProvider(t)
			source := t.TempDir()
			files := map[string]string{
				"go.mod":  "module cache.header.fixture\n\ngo 1.27.1\n",
				"main.go": "package main\nimport \"fmt\"\nvar Value uint64\nfunc main(){fmt.Println(Value)}\n",
				"value.s": "#include \"textflag.h\"\n" + test.directive + "\nDATA ·Value+0(SB)/8, $FIXTURE_VALUE\nGLOBL ·Value(SB), NOPTR, $8\n",
				"value.h": "#define FIXTURE_VALUE 1\n",
			}
			header := "value.h"
			if test.nested {
				files["value.h"] = "#include \"nested/value.h\"\n"
				files["nested/value.h"] = "#define FIXTURE_VALUE 1\n"
				header = "nested/value.h"
			}
			if test.shadow {
				files["go_asm.h"] = files["value.h"]
				delete(files, "value.h")
			}
			for path, data := range files {
				path = filepath.Join(source, path)
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(data), 0600); err != nil {
					t.Fatal(err)
				}
			}
			spec := target.Spec{Backend: Name, Kind: target.KindGoRun, Source: ".", WorkingDir: source, PreparationRoot: filepath.Join(t.TempDir(), "first"), BuildTags: []string{"test_dep"}}
			first, err := provider.Prepare(t.Context(), spec)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(source, header), []byte("#define FIXTURE_VALUE 2\n"), 0600); err != nil {
				t.Fatal(err)
			}
			spec.PreparationRoot = filepath.Join(t.TempDir(), "second")
			second, err := provider.Prepare(t.Context(), spec)
			if err != nil {
				t.Fatal(err)
			}
			if first.BuildKey == second.BuildKey || first.SHA256 == second.SHA256 {
				t.Fatal("changed assembly header reused stale compilation cache")
			}
		})
	}
}

func TestStockPreparationDisablesAmbientPGO(t *testing.T) {
	provider := integrationProvider(t)
	source := t.TempDir()
	for path, data := range map[string]string{"go.mod": "module cache.pgo.fixture\n\ngo 1.27.1\n", "main.go": "package main\nfunc main(){}\n"} {
		if err := os.WriteFile(filepath.Join(source, path), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	spec := target.Spec{Backend: Name, Kind: target.KindGoRun, Source: ".", WorkingDir: source, PreparationRoot: filepath.Join(t.TempDir(), "first"), BuildTags: []string{"test_dep"}}
	first, err := provider.Prepare(t.Context(), spec)
	if err != nil {
		t.Fatal(err)
	}
	profile := filepath.Join(source, "default.pgo")
	for _, data := range []string{"malformed ambient profile", "changed malformed ambient profile", ""} {
		if data == "" {
			if err := os.Remove(profile); err != nil {
				t.Fatal(err)
			}
		} else if err := os.WriteFile(profile, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		spec.PreparationRoot = filepath.Join(t.TempDir(), "next")
		options := provider.options
		options.CacheRoot = filepath.Join(t.TempDir(), "uncached")
		uncached, err := New(options)
		if err != nil {
			t.Fatal(err)
		}
		prepared, err := uncached.Prepare(t.Context(), spec)
		if err != nil {
			t.Fatalf("disabled ambient PGO affected preparation: %v", err)
		}
		if prepared.BuildKey != first.BuildKey || prepared.SHA256 != first.SHA256 {
			t.Fatal("disabled ambient PGO changed build identity")
		}
		pr, err := decodeProvenance(prepared.BackendPayloads[0].Data)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Contains(pr.BuildFlags, "-pgo=off") {
			t.Fatal("disabled PGO not bound by provenance")
		}
	}
}
