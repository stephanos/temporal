package target

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/target/internal/compatibility"
)

func TestProjectBuildInfoCanonicalizesModuleAndSettings(t *testing.T) {
	info := &debug.BuildInfo{
		GoVersion: "go1.26.4",
		Path:      "example.com/target",
		Main:      debug.Module{Path: "example.com/main", Version: "v1.2.3"},
		Settings: []debug.BuildSetting{
			{Key: "zeta", Value: "last"},
			{Key: "alpha", Value: "first"},
		},
	}
	want := evidence.BuildInfo{
		GoVersion: "go1.26.4", Path: "example.com/target", MainModule: "example.com/main@v1.2.3",
		Settings: []evidence.BuildSetting{{Key: "alpha", Value: "first"}, {Key: "zeta", Value: "last"}},
	}
	if got := ProjectBuildInfo(info); !reflect.DeepEqual(got, want) {
		t.Fatalf("ProjectBuildInfo() = %#v, want %#v", got, want)
	}
}

func TestPrepareGoRunBuildsOnceWithPinnedToolchain(t *testing.T) {
	module := writeModule(t, map[string]string{
		"go.mod":  "module example.com/target\n\ngo 1.26.4\n",
		"main.go": "package main\nimport (\"fmt\"; \"os\")\nfunc main() { fmt.Println(os.Getenv(\"GOMADSEED\"), os.Args[1]) }\n",
	})
	prepared, err := Prepare(context.Background(), Spec{
		Kind:            KindGoRun,
		Source:          ".",
		Args:            []string{"argument"},
		WorkingDir:      module,
		PreparationRoot: t.TempDir(),
		ToolchainRoot:   toolchainRoot(t),
		BuildTags:       []string{"zeta", "alpha", "zeta"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := prepared.BuildTags, []string{"alpha", "zeta"}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("build tags = %v, want %v", got, want)
	}
	if prepared.Compatibility == nil {
		t.Fatal("compatibility packs are null")
	}
	if prepared.GoVersion != "go1.26.4" || prepared.BuildKey == "" || prepared.TargetGOOS != runtime.GOOS || prepared.TargetGOARCH != runtime.GOARCH {
		t.Fatalf("toolchain identity = %#v", prepared)
	}
	if prepared.Argv[0] != "gomadv3-target" {
		t.Fatalf("argv = %v", prepared.Argv)
	}
	if err := prepared.Verify(); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(module, "main.go"), []byte("invalid source after preparation"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, seed := range []string{"1", "2"} {
		command := exec.Command(prepared.Path, prepared.Argv[1:]...)
		command.Args[0] = prepared.Argv[0]
		command.Env = []string{"GOMADSEED=" + seed, "TZ=UTC"}
		output, runErr := command.CombinedOutput()
		if runErr != nil {
			t.Fatalf("run seed %s: %v: %s", seed, runErr, output)
		}
		if got, want := string(output), seed+" argument\n"; got != want {
			t.Fatalf("seed %s output = %q, want %q", seed, got, want)
		}
	}
}

func TestPrepareGoRunUsesBuildOverlay(t *testing.T) {
	module := writeModule(t, map[string]string{
		"go.mod":  "module example.com/overlay\n\ngo 1.26.4\n",
		"main.go": "package main\nimport \"fmt\"\nfunc main() { fmt.Println(\"original\") }\n",
	})
	replacement := filepath.Join(t.TempDir(), "main.go")
	if err := os.WriteFile(replacement, []byte("package main\nimport \"fmt\"\nfunc main() { fmt.Println(\"overlay\") }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	original, err := filepath.EvalSymlinks(filepath.Join(module, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	replacement, err = filepath.EvalSymlinks(replacement)
	if err != nil {
		t.Fatal(err)
	}
	overlay := filepath.Join(t.TempDir(), "overlay.json")
	encoded, err := json.Marshal(map[string]any{"Replace": map[string]string{original: replacement}})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(overlay, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	prepared, err := Prepare(context.Background(), Spec{
		Kind: KindGoRun, Source: ".", WorkingDir: module, PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot(t), BuildOverlay: overlay,
	})
	if err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(prepared.Path).CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	if string(output) != "overlay\n" {
		t.Fatalf("output = %q", output)
	}
}

func TestPrepareRejectsLinknameInBuildOverlay(t *testing.T) {
	module := writeModule(t, map[string]string{
		"go.mod":  "module example.com/overlayescape\n\ngo 1.26.4\n",
		"main.go": "package main\nfunc main() {}\n",
	})
	replacement := filepath.Join(t.TempDir(), "main.go")
	if err := os.WriteFile(replacement, []byte("package main\nimport _ \"unsafe\"\n//go:linkname escape syscall.Getpid\nfunc escape() int\nfunc main() { _ = escape() }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	original, err := filepath.EvalSymlinks(filepath.Join(module, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	replacement, err = filepath.EvalSymlinks(replacement)
	if err != nil {
		t.Fatal(err)
	}
	overlay := filepath.Join(t.TempDir(), "overlay.json")
	encoded, err := json.Marshal(map[string]any{"Replace": map[string]string{original: replacement}})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(overlay, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = Prepare(context.Background(), Spec{
		Kind: KindGoRun, Source: ".", WorkingDir: module, PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot(t), BuildOverlay: overlay,
	})
	if err == nil || !strings.Contains(err.Error(), "go:linkname") {
		t.Fatalf("Prepare() error = %v", err)
	}
}

func TestReadModuleCacheUsesPinnedToolchain(t *testing.T) {
	moduleCache, err := ReadModuleCache(context.Background(), toolchainRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(moduleCache)
	if err != nil || !info.IsDir() || !filepath.IsAbs(moduleCache) {
		t.Fatalf("module cache = %q, info = %#v, error = %v", moduleCache, info, err)
	}
}

func TestPrepareGoTestWithoutBuildTagsRemainsTagNeutral(t *testing.T) {
	module := writeModule(t, map[string]string{
		"go.mod":         "module example.com/targettest\n\ngo 1.26.4\n",
		"target.go":      "package targettest\n",
		"target_test.go": "package targettest\n\nimport \"testing\"\n\nfunc TestBase(t *testing.T) {}\n",
	})
	prepared, err := Prepare(context.Background(), Spec{
		Kind:            KindGoTest,
		Source:          ".",
		WorkingDir:      module,
		PreparationRoot: t.TempDir(),
		ToolchainRoot:   toolchainRoot(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(prepared.BuildTags) != 0 {
		t.Fatalf("build tags = %v, want none", prepared.BuildTags)
	}
}

func TestPrepareGoTestPreservesExplicitTestDependencyTag(t *testing.T) {
	module := writeModule(t, map[string]string{
		"go.mod": "module example.com/targettest\n\ngo 1.26.4\n",
		"tagged_test.go": `//go:build gomad_fixture

package targettest

import (
	"os"
	"testing"
)

func TestTagged(t *testing.T) {
	if got := os.Getenv("GOMADSEED"); got != "9" {
		t.Fatalf("seed = %q", got)
	}
}
`,
	})
	prepared, err := Prepare(context.Background(), Spec{
		Kind:            KindGoTest,
		Source:          ".",
		Args:            []string{"-test.run=TestTagged", "-test.count=1"},
		BuildTags:       []string{"gomad_fixture"},
		WorkingDir:      module,
		PreparationRoot: t.TempDir(),
		ToolchainRoot:   toolchainRoot(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := prepared.BuildTags, []string{"gomad_fixture"}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("build tags = %v, want %v", got, want)
	}
	command := exec.Command(prepared.Path, prepared.Argv[1:]...)
	command.Args[0] = prepared.Argv[0]
	command.Env = []string{"GOMADSEED=9", "TZ=UTC"}
	if output, runErr := command.CombinedOutput(); runErr != nil {
		t.Fatalf("run generated test: %v: %s", runErr, output)
	}
}

func TestPrepareRejectsEscapeCapabilityClosure(t *testing.T) {
	tests := map[string]struct {
		kind  Kind
		files map[string]string
	}{
		"process": {
			files: map[string]string{"main.go": `package main

import "os/exec"

func main() { _ = exec.Command("true") }
`},
		},
		"signal": {
			files: map[string]string{"main.go": "package main\n\nimport \"os/signal\"\n\nfunc main() { _ = signal.Ignored() }\n"},
		},
		"host user": {
			files: map[string]string{"main.go": "package main\n\nimport \"os/user\"\n\nfunc main() { _, _ = user.Current() }\n"},
		},
		"raw syscall": {
			files: map[string]string{"main.go": "package main\n\nimport \"syscall\"\n\nfunc main() { _ = syscall.Getpid() }\n"},
		},
		"plugin": {
			files: map[string]string{"main.go": "package main\n\nimport \"plugin\"\n\nfunc main() { _, _ = plugin.Open(\"target.so\") }\n"},
		},
		"test process": {
			kind: KindGoTest,
			files: map[string]string{
				"main.go":      "package main\n\nfunc main() {}\n",
				"main_test.go": "package main\n\nimport (\"os/exec\"; \"testing\")\n\nfunc TestEscape(t *testing.T) { _ = exec.Command(\"true\") }\n",
			},
		},
		"assembly": {
			files: map[string]string{
				"main.go":  "package main\n\nfunc escape()\nfunc main() { escape() }\n",
				"escape.s": "TEXT ·escape(SB),$0-0\n\tRET\n",
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			test.files["go.mod"] = "module example.com/escape\n\ngo 1.26.4\n"
			module := writeModule(t, test.files)
			kind := test.kind
			if kind == "" {
				kind = KindGoRun
			}
			_, err := Prepare(context.Background(), Spec{
				Kind: kind, Source: ".", WorkingDir: module,
				PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot(t),
			})
			if err == nil || !strings.Contains(err.Error(), "unsupported target capability") {
				t.Fatalf("Prepare() error = %v", err)
			}
		})
	}
}

func TestPrepareScrubsDeterministicActivationFromBuild(t *testing.T) {
	module := writeModule(t, map[string]string{
		"go.mod":  "module example.com/envtarget\n\ngo 1.26.4\n",
		"main.go": "package main\nfunc main() {}\n",
	})
	t.Setenv("GOMADSEED", "malformed-build-seed")
	t.Setenv("GOMADV3_CHILD_SEED", "malformed-child-seed")
	t.Setenv("GOROOT", filepath.Join(t.TempDir(), "missing-goroot"))
	t.Setenv("GOWORK", filepath.Join(t.TempDir(), "missing.work"))
	_, err := Prepare(context.Background(), Spec{
		Kind:            KindGoRun,
		Source:          ".",
		WorkingDir:      module,
		PreparationRoot: t.TempDir(),
		ToolchainRoot:   toolchainRoot(t),
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPrepareResolvesFilesystemPackageToNestedModule(t *testing.T) {
	parent := t.TempDir()
	module := filepath.Join(parent, "nested")
	if err := os.MkdirAll(filepath.Join(module, "cmd"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(module, "go.mod"), []byte("module example.com/nested\n\ngo 1.26.4\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(module, "cmd", "main.go"), []byte("package main\nfunc main() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	prepared, err := Prepare(context.Background(), Spec{
		Kind: KindGoRun, Source: "./nested/cmd", WorkingDir: parent, PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Source != "./nested/cmd" {
		t.Fatalf("recorded source = %q", prepared.Source)
	}
}

func TestPreparedVerifyRejectsMutation(t *testing.T) {
	module := writeModule(t, map[string]string{
		"go.mod":  "module example.com/mutation\n\ngo 1.26.4\n",
		"main.go": "package main\nfunc main() {}\n",
	})
	prepared, err := Prepare(context.Background(), Spec{
		Kind:            KindGoRun,
		Source:          ".",
		WorkingDir:      module,
		PreparationRoot: t.TempDir(),
		ToolchainRoot:   toolchainRoot(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(prepared.Path, 0o700); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(prepared.Path, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("mutation")); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := prepared.Verify(); err == nil || !strings.Contains(err.Error(), "changed after preparation") {
		t.Fatalf("Verify() error = %v", err)
	}
}

func TestPrepareExecRequiresMatchingProvenance(t *testing.T) {
	module := writeModule(t, map[string]string{
		"go.mod":  "module example.com/exectarget\n\ngo 1.26.4\n",
		"main.go": "package main\nfunc main() {}\n",
	})
	built, err := Prepare(context.Background(), Spec{
		Kind: KindGoRun, Source: ".", WorkingDir: module, PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	binary := built.Path
	contents, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(contents)
	identity, err := ReadToolchainIdentity(toolchainRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	projectedBuild := built.BuildInfo
	closure, err := ReviewCapabilityClosure(context.Background(), Spec{
		Kind: KindGoRun, Source: ".", WorkingDir: module, ToolchainRoot: toolchainRoot(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	provenance := filepath.Join(t.TempDir(), "provenance.json")
	if err := WriteProvenance(provenance, Provenance{
		SchemaVersion:     2,
		GoVersion:         identity.GoVersion,
		BuildKey:          identity.BuildKey,
		TargetGOOS:        identity.TargetGOOS,
		TargetGOARCH:      identity.TargetGOARCH,
		BinarySHA256:      fmt.Sprintf("sha256:%x", hash),
		BinarySize:        uint64(len(contents)),
		BuildInfo:         projectedBuild,
		CapabilityClosure: closure,
	}); err != nil {
		t.Fatal(err)
	}
	prepared, err := Prepare(context.Background(), Spec{
		Kind:            KindExec,
		Source:          binary,
		Provenance:      provenance,
		Args:            []string{"value"},
		PreparationRoot: t.TempDir(),
		ToolchainRoot:   toolchainRoot(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	if prepared.SHA256 != fmt.Sprintf("sha256:%x", hash) || prepared.Size != uint64(len(contents)) || prepared.BuildInfo.Path != projectedBuild.Path {
		t.Fatalf("prepared exec = %#v", prepared)
	}
	preparedProvenance := filepath.Join(filepath.Dir(prepared.Path), "provenance.json")
	preparedProvenanceBytes, err := os.ReadFile(preparedProvenance)
	if err != nil {
		t.Fatal(err)
	}
	originalProvenanceBytes, err := os.ReadFile(provenance)
	if err != nil {
		t.Fatal(err)
	}
	if string(preparedProvenanceBytes) != string(originalProvenanceBytes) {
		t.Fatal("prepared provenance snapshot changed")
	}
	if info, err := os.Stat(preparedProvenance); err != nil {
		t.Fatal(err)
	} else if info.Mode().Perm() != 0o400 {
		t.Fatalf("prepared provenance mode = %#o, want 0400", info.Mode().Perm())
	}

	forgedClosure := closure
	forgedClosure.Packages = append([]CapabilityPackage(nil), closure.Packages...)
	forgedClosure.Packages = append(forgedClosure.Packages, CapabilityPackage{
		ImportPath: "example.com/forged", Name: "forged", Standard: true, Imports: []string{"os/exec"},
		Sources: []CapabilitySource{}, ForeignSources: []CapabilityForeignSource{},
	})
	sort.Slice(forgedClosure.Packages, func(i, j int) bool {
		if forgedClosure.Packages[i].ImportPath != forgedClosure.Packages[j].ImportPath {
			return forgedClosure.Packages[i].ImportPath < forgedClosure.Packages[j].ImportPath
		}
		return forgedClosure.Packages[i].Name < forgedClosure.Packages[j].Name
	})
	forgedProvenance := filepath.Join(t.TempDir(), "forged-provenance.json")
	if err := WriteProvenance(forgedProvenance, Provenance{
		SchemaVersion: 2, GoVersion: identity.GoVersion, BuildKey: identity.BuildKey, TargetGOOS: identity.TargetGOOS, TargetGOARCH: identity.TargetGOARCH,
		BinarySHA256: fmt.Sprintf("sha256:%x", hash), BinarySize: uint64(len(contents)), BuildInfo: projectedBuild, CapabilityClosure: forgedClosure,
	}); err != nil {
		t.Fatal(err)
	}
	_, err = Prepare(context.Background(), Spec{
		Kind: KindExec, Source: binary, Provenance: forgedProvenance, PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot(t),
	})
	if err == nil || !strings.Contains(err.Error(), "standard package") {
		t.Fatalf("forged standard package Prepare() error = %v", err)
	}

	symlinkProvenance := filepath.Join(t.TempDir(), "provenance-link.json")
	if err := os.Symlink(provenance, symlinkProvenance); err != nil {
		t.Fatal(err)
	}
	_, err = Prepare(context.Background(), Spec{
		Kind: KindExec, Source: binary, Provenance: symlinkProvenance, PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot(t),
	})
	if err == nil || !strings.Contains(err.Error(), "regular file") {
		t.Fatalf("symlink provenance Prepare() error = %v", err)
	}

	invalidBinary := filepath.Join(t.TempDir(), "invalid-target")
	invalidContents := []byte("not a Go executable")
	if err := os.WriteFile(invalidBinary, invalidContents, 0o700); err != nil {
		t.Fatal(err)
	}
	invalidHash := sha256.Sum256(invalidContents)
	invalidProvenance := filepath.Join(t.TempDir(), "invalid-provenance.json")
	if err := WriteProvenance(invalidProvenance, Provenance{
		SchemaVersion: 2, GoVersion: identity.GoVersion, BuildKey: identity.BuildKey, TargetGOOS: identity.TargetGOOS, TargetGOARCH: identity.TargetGOARCH,
		BinarySHA256: fmt.Sprintf("sha256:%x", invalidHash), BinarySize: uint64(len(invalidContents)), BuildInfo: projectedBuild, CapabilityClosure: closure,
	}); err != nil {
		t.Fatal(err)
	}
	_, err = Prepare(context.Background(), Spec{
		Kind:            KindExec,
		Source:          invalidBinary,
		Provenance:      invalidProvenance,
		PreparationRoot: t.TempDir(),
		ToolchainRoot:   toolchainRoot(t),
	})
	if err == nil || !strings.Contains(err.Error(), "build info") {
		t.Fatalf("Prepare() error = %v", err)
	}
}

func TestPreparedRecordProjectionDefensivelyCopiesIdentity(t *testing.T) {
	prepared := Prepared{
		Kind: KindGoTest, Source: "./pkg", SHA256: string(evidence.HashBytes([]byte("target"))), Size: 6,
		Argv: []string{"target", "argument"}, BuildTags: []string{"tag"},
		Adapters: []evidence.TargetAdapter{{Module: "adapter"}}, Compatibility: []evidence.CompatibilityPack{{ID: "pack"}},
		BuildInfo: evidence.BuildInfo{GoVersion: "go1.26.4", Path: "example.com/target", Settings: []evidence.BuildSetting{{Key: "setting", Value: "value"}}},
		GoVersion: "go1.26.4", BuildKey: strings.Repeat("a", 64), TargetGOOS: "darwin", TargetGOARCH: "arm64",
	}
	targetRecord := prepared.RecordTarget()
	toolchainRecord := prepared.RecordToolchain()
	prepared.Argv[0] = "changed"
	prepared.BuildTags[0] = "changed"
	prepared.Adapters[0].Module = "changed"
	prepared.Compatibility[0].ID = "changed"
	prepared.BuildInfo.Settings[0].Value = "changed"
	if targetRecord.Kind != "go-test" || targetRecord.Argv[0] != "target" || targetRecord.BuildTags[0] != "tag" || targetRecord.Adapters[0].Module != "adapter" || targetRecord.Compatibility[0].ID != "pack" || targetRecord.BuildInfo.Settings[0].Value != "value" {
		t.Fatalf("RecordTarget() = %#v", targetRecord)
	}
	if toolchainRecord.GoVersion != "go1.26.4" || toolchainRecord.BuildKey != strings.Repeat("a", 64) || toolchainRecord.TargetGOOS != "darwin" || toolchainRecord.TargetGOARCH != "arm64" {
		t.Fatalf("RecordToolchain() = %#v", toolchainRecord)
	}
}

func TestReadProvenanceRoundTripsValidatedDocument(t *testing.T) {
	want := Provenance{
		SchemaVersion: 2, GoVersion: "go1.26.4", BuildKey: strings.Repeat("a", 64),
		TargetGOOS: runtime.GOOS, TargetGOARCH: runtime.GOARCH,
		BinarySHA256: "sha256:" + strings.Repeat("b", 64), BinarySize: 1,
		BuildInfo: evidence.BuildInfo{
			GoVersion: "go1.26.4", Path: "example.com/target",
			Settings: []evidence.BuildSetting{{Key: "-buildmode", Value: "exe"}, {Key: "CGO_ENABLED", Value: "0"}},
		},
		CapabilityClosure: validCapabilityClosure(),
	}
	path := filepath.Join(t.TempDir(), "provenance.json")
	if err := WriteProvenance(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := ReadProvenance(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ReadProvenance() = %#v, want %#v", got, want)
	}
}

func TestValidateProvenanceRejectsUnsupportedBuildModes(t *testing.T) {
	base := provenanceWire{
		Schema: provenanceSchema, SchemaVersion: 2, GoVersion: "go1.26.4", BuildKey: strings.Repeat("a", 64),
		TargetGOOS: runtime.GOOS, TargetGOARCH: runtime.GOARCH, BinarySHA256: "sha256:" + strings.Repeat("b", 64), BinarySize: 1,
		BuildInfo:         evidence.BuildInfo{GoVersion: "go1.26.4", Path: "example.com/target", Settings: []evidence.BuildSetting{{Key: "-buildmode", Value: "exe"}, {Key: "CGO_ENABLED", Value: "0"}}},
		CapabilityClosure: validCapabilityClosure(),
	}
	for name, setting := range map[string]evidence.BuildSetting{
		"cgo":        {Key: "CGO_ENABLED", Value: "1"},
		"race":       {Key: "-race", Value: "true"},
		"plugin":     {Key: "-buildmode", Value: "plugin"},
		"linkshared": {Key: "-linkshared", Value: "true"},
		"external":   {Key: "-ldflags", Value: "-linkmode=external"},
	} {
		t.Run(name, func(t *testing.T) {
			candidate := base
			candidate.BuildInfo.Settings = append([]evidence.BuildSetting(nil), base.BuildInfo.Settings...)
			replaced := false
			for index := range candidate.BuildInfo.Settings {
				if candidate.BuildInfo.Settings[index].Key == setting.Key {
					candidate.BuildInfo.Settings[index] = setting
					replaced = true
				}
			}
			if !replaced {
				candidate.BuildInfo.Settings = append(candidate.BuildInfo.Settings, setting)
			}
			if err := validateProvenance(candidate); err == nil {
				t.Fatal("validateProvenance() succeeded")
			}
		})
	}
}

func TestValidateProvenanceRejectsClosurelessAttestation(t *testing.T) {
	provenance := provenanceWire{
		Schema: "gomadv3.exec-provenance/v1", SchemaVersion: 1, GoVersion: "go1.26.4", BuildKey: strings.Repeat("a", 64),
		TargetGOOS: runtime.GOOS, TargetGOARCH: runtime.GOARCH, BinarySHA256: "sha256:" + strings.Repeat("b", 64), BinarySize: 1,
		BuildInfo: evidence.BuildInfo{GoVersion: "go1.26.4", Path: "example.com/target", Settings: []evidence.BuildSetting{{Key: "-buildmode", Value: "exe"}, {Key: "CGO_ENABLED", Value: "0"}}},
	}
	if err := validateProvenance(provenance); err == nil {
		t.Fatal("validateProvenance() succeeded")
	}
}

func TestValidateExecCapabilityModulesMatchesBuildInfo(t *testing.T) {
	closure := validCapabilityClosure()
	closure.Packages = append([]CapabilityPackage{{
		ImportPath: "example.net/dependency", Name: "dependency", Imports: []string{},
		Module:  &CapabilityModule{Path: "example.net/dependency", Version: "v1.2.3", Sum: "h1:sum"},
		Sources: []CapabilitySource{{Name: "dependency.go", SHA256: "sha256:" + strings.Repeat("d", 64)}}, ForeignSources: []CapabilityForeignSource{},
	}}, closure.Packages...)
	info := &debug.BuildInfo{
		Main: debug.Module{Path: "example.com/target"},
		Deps: []*debug.Module{{Path: "example.net/dependency", Version: "v1.2.3", Sum: "h1:sum"}},
	}
	if err := validateExecCapabilityModules(info, closure); err != nil {
		t.Fatal(err)
	}
	info.Deps[0].Version = "v1.2.4"
	if err := validateExecCapabilityModules(info, closure); err == nil {
		t.Fatal("validateExecCapabilityModules() succeeded with a mismatched module")
	}
}

func TestPrepareRejectsUnsupportedBuildTags(t *testing.T) {
	for _, tags := range [][]string{{""}, {"race"}, {"a,b"}, {"two words"}} {
		t.Run(fmt.Sprint(tags), func(t *testing.T) {
			_, err := Prepare(context.Background(), Spec{
				Kind:            KindGoRun,
				Source:          ".",
				WorkingDir:      t.TempDir(),
				PreparationRoot: t.TempDir(),
				ToolchainRoot:   toolchainRoot(t),
				BuildTags:       tags,
			})
			if err == nil {
				t.Fatal("Prepare() succeeded")
			}
		})
	}
}

func TestPrepareRejectsOptionLikeGoPackage(t *testing.T) {
	_, err := Prepare(context.Background(), Spec{
		Kind: KindGoRun, Source: "-buildmode=plugin", WorkingDir: t.TempDir(), PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot(t),
	})
	if err == nil || !strings.Contains(err.Error(), "package argument") {
		t.Fatalf("Prepare() error = %v", err)
	}
}

func writeModule(t *testing.T, files map[string]string) string {
	t.Helper()
	directory := t.TempDir()
	for name, contents := range files {
		path := filepath.Join(directory, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return directory
}

func toolchainRoot(t *testing.T) string {
	t.Helper()
	directory, err := filepath.Abs(filepath.Join("..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	return directory
}

func validCapabilityClosure() CapabilityClosure {
	return CapabilityClosure{
		Schema:        CapabilityClosureSchema,
		Compatibility: []compatibility.Identity{},
		Packages: []CapabilityPackage{{
			ImportPath: "example.com/target", Name: "main", Imports: []string{},
			Module:  &CapabilityModule{Path: "example.com/target", Main: true},
			Sources: []CapabilitySource{{Name: "main.go", SHA256: "sha256:" + strings.Repeat("c", 64)}}, ForeignSources: []CapabilityForeignSource{},
		}},
	}
}
