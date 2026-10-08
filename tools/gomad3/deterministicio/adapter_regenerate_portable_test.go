package deterministicio

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/target"
	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
	"golang.org/x/mod/module"
)

func portableAdapterGo(t *testing.T) (string, string) {
	t.Helper()
	command, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("portable adapter check NoError failed: %v", err)
	}
	command, err = filepath.Abs(command)
	if err != nil {
		t.Fatalf("portable adapter check NoError failed: %v", err)
	}
	output, err := exec.CommandContext(t.Context(), command, "env", "GOVERSION", "GOMODCACHE").Output()
	if err != nil {
		t.Fatalf("portable adapter check NoError failed: %v", err)
	}
	settings := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(settings) != 2 {
		t.Fatalf("portable adapter check Len failed: %v", settings)
	}
	if !reflect.DeepEqual(gomadversion.GoVersion, settings[0]) {
		t.Fatalf("portable adapter check Equal failed: %v", gomadversion.GoVersion)
	}
	return command, settings[1]
}

func portableAdapterModule(t *testing.T, cache, name, version string) string {
	t.Helper()
	escaped, err := module.EscapePath(name)
	if err != nil {
		t.Fatalf("portable adapter check NoError failed: %v", err)
	}
	directory := filepath.Join(cache, filepath.FromSlash(escaped+"@"+version))
	if _, err := os.Stat(directory); errors.Is(err, os.ErrNotExist) {
		goCommand, _ := portableAdapterGo(t)
		command := exec.CommandContext(t.Context(), goCommand, "mod", "download", name+"@"+version)
		command.Dir = t.TempDir()
		command.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=")
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("download exact pinned %s@%s: %v\n%s", name, version, err, output)
		}
	}
	info, err := os.Stat(directory)
	if err != nil {
		t.Fatalf("portable adapter check NoError failed: %v", err)
	}
	if !(info.IsDir()) {
		t.Fatalf("portable adapter check True failed: %v", info.IsDir())
	}
	return directory
}

func TestPortableAdapterPreparedSourcePins(t *testing.T) {
	goCommand, cache := portableAdapterGo(t)
	if len(deterministicAdapters.definitions) != len(gomadversion.Adapters) {
		t.Fatalf("portable adapter check Len failed: %v", deterministicAdapters.definitions)
	}
	checked := 0
	for _, definition := range deterministicAdapters.definitions {
		t.Run(definition.identity.Module, func(t *testing.T) {
			portableAdapterModule(t, cache, definition.identity.Module, definition.identity.Version)
			prepared, err := definition.implementation.prepare(cache, t.TempDir(), definition.identity)
			if err != nil {
				t.Fatalf("portable adapter check NoError failed: %v", err)
			}
			if !reflect.DeepEqual(definition.identity.Module, prepared.evidence.Module) {
				t.Fatalf("portable adapter check Equal failed: %v", definition.identity.Module)
			}
			if !reflect.DeepEqual(definition.identity.Version, prepared.evidence.Version) {
				t.Fatalf("portable adapter check Equal failed: %v", definition.identity.Version)
			}
			if !reflect.DeepEqual(definition.identity.Sum, prepared.evidence.Sum) {
				t.Fatalf("portable adapter check Equal failed: %v", definition.identity.Sum)
			}
			if !reflect.DeepEqual(prepared.replacement, prepared.evidence.ReplacementRoot) {
				t.Fatalf("portable adapter check Equal failed: %v", prepared.replacement)
			}
			pins := libcPreparedSourceSetSHA256ByHost
			if spec := definition.implementation.rewritten; spec != nil {
				pins = spec.preparedSourceSetSHA256ByHost
				if !reflect.DeepEqual(spec.originalInventorySHA256, prepared.evidence.OriginalSourceInventorySHA256) {
					t.Fatalf("portable adapter check Equal failed: %v", spec.originalInventorySHA256)
				}
				if !reflect.DeepEqual(spec.replacementInventorySHA256, prepared.evidence.ReplacementSourceInventorySHA256) {
					t.Fatalf("portable adapter check Equal failed: %v", spec.replacementInventorySHA256)
				}
				for _, rewrite := range spec.rewrites {
					contents, err := os.ReadFile(filepath.Join(prepared.replacement, filepath.FromSlash(rewrite.path)))
					if err != nil {
						t.Fatalf("portable adapter check NoError failed: %v", err)
					}
					if !reflect.DeepEqual(rewrite.replacementSHA256, digestBytes(contents)) {
						t.Fatalf("portable adapter check Equal failed: %v", rewrite.replacementSHA256)
					}
				}
			}
			if len(pins) != 2 {
				t.Fatalf("portable adapter check Len failed: %v", pins)
			}
			directory := prepared.replacement
			if prepared.evidence.PreparedPackage != definition.identity.Module {
				directory = filepath.Join(directory, filepath.FromSlash(strings.TrimPrefix(prepared.evidence.PreparedPackage, definition.identity.Module+"/")))
			}
			for _, platform := range []string{"darwin/arm64", "linux/amd64"} {
				t.Run(platform, func(t *testing.T) {
					goos, goarch, _ := strings.Cut(platform, "/")
					got, err := target.AdapterPreparedSourceSetSHA256(t.Context(), goCommand, directory, prepared.evidence.PreparedPackage, goos, goarch)
					if err != nil {
						t.Fatalf("portable adapter check NoError failed: %v", err)
					}
					if !reflect.DeepEqual(pins[platform], got) {
						t.Fatalf("portable adapter check Equal failed: %v", pins[platform])
					}
					checked++
				})
			}
		})
	}
	if !reflect.DeepEqual(2*len(deterministicAdapters.definitions), checked) {
		t.Fatalf("portable adapter check Equal failed: %v", 2*len(deterministicAdapters.definitions))
	}
	t.Logf("reproduced %d exact source-set pins with stock %s", checked, gomadversion.GoVersion)
}

func portableRegenerationFixture(t *testing.T, spec rewrittenModule) regenerationFixture {
	t.Helper()
	goCommand, cache := portableAdapterGo(t)
	pinned := portableAdapterModule(t, cache, spec.module, spec.version)
	fixture := regenerationFixture{spec: spec, previous: filepath.Join(t.TempDir(), "previous"), candidate: filepath.Join(t.TempDir(), "candidate"), goCommand: goCommand}
	copyFixtureTree(t, pinned, fixture.previous)
	copyFixtureTree(t, pinned, fixture.candidate)
	return fixture
}

func TestPortableAdapterRegenerationSourceEdits(t *testing.T) {
	for _, spec := range []rewrittenModule{sentryAdapter, memberlistAdapter, grpcAdapter} {
		t.Run(spec.module, func(t *testing.T) {
			fixture := portableRegenerationFixture(t, spec)
			rewrite := spec.rewrites[0]
			if spec.module == grpcModulePath {
				rewrite = grpcLinuxRewrites[1]
			}
			changed := rewrite.path
			if rewrite.base != "" {
				changed = rewrite.base
			}
			editFixtureFile(t, filepath.Join(fixture.candidate, filepath.FromSlash(changed)), func(source []byte) []byte {
				return append(source, []byte("\n// portable upstream change\n")...)
			})
			if spec.module == memberlistModulePath {
				for _, other := range spec.rewrites[1:] {
					editFixtureFile(t, filepath.Join(fixture.candidate, filepath.FromSlash(other.path)), func(source []byte) []byte {
						return append(source, []byte("\n// portable upstream change\n")...)
					})
				}
			}
			regeneration, err := fixture.regenerate(t)
			if err != nil {
				t.Fatalf("portable adapter check NoError failed: %v", err)
			}
			if !reflect.DeepEqual(spec.version, regeneration.Previous.Version) {
				t.Fatalf("portable adapter check Equal failed: %v", spec.version)
			}
			if !reflect.DeepEqual("v99.0.0-fixture", regeneration.Proposed.Version) {
				t.Fatalf("portable adapter check Equal failed: %v", "v99.0.0-fixture")
			}
			if reflect.DeepEqual(regeneration.Previous.OriginalSourceInventorySHA256, regeneration.Proposed.OriginalSourceInventorySHA256) {
				t.Fatalf("portable adapter check NotEqual failed: %v", regeneration.Previous.OriginalSourceInventorySHA256)
			}
			if reflect.DeepEqual(regeneration.Previous.ReplacementSourceInventorySHA256, regeneration.Proposed.ReplacementSourceInventorySHA256) {
				t.Fatalf("portable adapter check NotEqual failed: %v", regeneration.Previous.ReplacementSourceInventorySHA256)
			}
			again, err := fixture.regenerate(t)
			if err != nil {
				t.Fatalf("portable adapter check NoError failed: %v", err)
			}
			if !reflect.DeepEqual(regeneration.ApprovalSHA256, again.ApprovalSHA256) {
				t.Fatalf("portable adapter check Equal failed: %v", regeneration.ApprovalSHA256)
			}
			for index, next := range regeneration.Proposed.Rewrites {
				previous := regeneration.Previous.Rewrites[index]
				if next.Path == rewrite.path || spec.module == memberlistModulePath {
					if reflect.DeepEqual(previous.ReplacementSHA256, next.ReplacementSHA256) {
						t.Fatalf("portable adapter check NotEqual failed: %v", previous.ReplacementSHA256)
					}
					if rewrite.base != "" {
						if !reflect.DeepEqual(previous.SourceSHA256, next.SourceSHA256) {
							t.Fatalf("portable adapter check Equal failed: %v", previous.SourceSHA256)
						}
						if reflect.DeepEqual(previous.BaseSHA256, next.BaseSHA256) {
							t.Fatalf("portable adapter check NotEqual failed: %v", previous.BaseSHA256)
						}
					} else {
						if reflect.DeepEqual(previous.SourceSHA256, next.SourceSHA256) {
							t.Fatalf("portable adapter check NotEqual failed: %v", previous.SourceSHA256)
						}
					}
				} else {
					if !reflect.DeepEqual(previous, next) {
						t.Fatalf("portable adapter check Equal failed: %v", previous)
					}
				}
			}
			root := t.TempDir()
			copyFixtureTree(t, ".", filepath.Join(root, "deterministicio"))
			beforeOther, err := os.ReadFile(filepath.Join(root, "deterministicio", "hashicorpmetrics_adapter.go"))
			if err != nil {
				t.Fatalf("portable adapter check NoError failed: %v", err)
			}
			if spec.module == memberlistModulePath {
				if err := os.WriteFile(filepath.Join(root, "deterministicio", "portable_reference_test.go"), []byte("package deterministicio\nconst portableReplace = \"replace "+memberlistModulePath+" "+memberlistVersion+" => ./memberlist\\n\"\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			edits, err := regeneration.SourceEdits(root)
			if err != nil {
				t.Fatalf("portable adapter check NoError failed: %v", err)
			}
			var source []byte
			for name, contents := range edits {
				if strings.HasSuffix(name, "_adapter.go") {
					source = append(source, contents...)
				}
			}
			for _, pin := range []string{regeneration.Proposed.Version, regeneration.Proposed.Sum, regeneration.Proposed.OriginalSourceInventorySHA256, regeneration.Proposed.ReplacementSourceInventorySHA256} {
				if !strings.Contains(string(source), pin) {
					t.Fatalf("portable adapter check Contains failed: %v", string(source))
				}
			}
			for _, next := range regeneration.Proposed.Rewrites {
				if !strings.Contains(string(source), next.SourceSHA256) {
					t.Fatalf("portable adapter check Contains failed: %v", string(source))
				}
				if !strings.Contains(string(source), next.ReplacementSHA256) {
					t.Fatalf("portable adapter check Contains failed: %v", string(source))
				}
			}
			for platform, pin := range regeneration.Proposed.PreparedSourceSetSHA256 {
				if !strings.Contains(string(source), pin) {
					t.Fatalf("portable adapter check Contains failed: %v", string(source))
				}
				changedPin := regeneration.Previous.PreparedSourceSetSHA256[platform] != pin
				if changedPin != (spec.module != grpcModulePath) {
					t.Fatalf("%s prepared pin changed=%t for changed source %s", platform, changedPin, changed)
				}
			}
			if spec.module == memberlistModulePath {
				if !strings.Contains(string(edits["deterministicio/portable_reference_test.go"]), memberlistModulePath+" v99.0.0-fixture => ./memberlist") {
					t.Fatalf("portable adapter check Contains failed: %v", string(edits["deterministicio/portable_reference_test.go"]))
				}
				other := edits["deterministicio/hashicorpmetrics_adapter.go"]
				if other != nil && !bytes.Equal(other, beforeOther) {
					t.Fatal("memberlist regeneration changed the metrics adapter")
				}
			}
			if spec.module == sentryModulePath {
				editFixtureFile(t, filepath.Join(root, "deterministicio", "sentry_adapter.go"), func(contents []byte) []byte {
					return bytes.Replace(contents, []byte(`"`+sentryUtilSourceSHA256+`"`), []byte(`"sha256:" + "`+strings.TrimPrefix(sentryUtilSourceSHA256, "sha256:")+`"`), 1)
				})
				_, err := regeneration.SourceEdits(root)
				if err == nil || !strings.Contains(err.Error(), "is not declared as a literal") {
					t.Fatalf("portable adapter check ErrorContains failed: %v", err)
				}
				if err := VerifyRegisteredAdapter(t.Context(), spec.module, fixture.previous, fixture.goCommand); err != nil {
					t.Fatal(err)
				}
				if err := VerifyRegisteredAdapter(t.Context(), spec.module, fixture.candidate, fixture.goCommand); err == nil || !strings.Contains(err.Error(), "source inventory identity mismatch") {
					t.Fatalf("changed adapter verification = %v", err)
				}
			}
		})
	}
}

func TestPortableAdapterRegenerationRefusals(t *testing.T) {
	anchor := sentryRewrites[0].rewrites[1].anchor
	for _, name := range []string{"missing anchor", "ambiguous anchor", "deleted file", "previous inventory"} {
		t.Run(name, func(t *testing.T) {
			fixture := portableRegenerationFixture(t, sentryAdapter)
			path := filepath.Join(fixture.candidate, sentryUtilPath)
			switch name {
			case "missing anchor":
				editFixtureFile(t, path, func(contents []byte) []byte {
					return bytes.Replace(contents, anchor, bytes.Replace(anchor, []byte("exec.LookPath"), []byte("exec.LookPath "), 1), 1)
				})
			case "ambiguous anchor":
				editFixtureFile(t, path, func(contents []byte) []byte {
					return append(contents, append([]byte("\nfunc duplicated() {\n"), append(slices.Clone(anchor), []byte("}\n")...)...)...)
				})
			case "deleted file":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			case "previous inventory":
				editFixtureFile(t, filepath.Join(fixture.previous, sentryUtilPath), func(contents []byte) []byte { return append(contents, '\n') })
			}
			scratch := t.TempDir()
			_, err := regenerateRewrittenModule(t.Context(), fixture.spec, AdapterRegenerationRequest{Module: sentryModulePath, Version: "v99.0.0-fixture", Sum: "h1:fixture", PreviousModule: fixture.previous, CandidateModule: fixture.candidate, GoCommand: fixture.goCommand, Scratch: scratch})
			if err == nil {
				t.Fatal("regeneration accepted source drift")
			}
			if strings.Contains(name, "anchor") {
				var mismatch *AnchorMismatchError
				want := 0
				if name == "ambiguous anchor" {
					want = 2
				}
				if !errors.As(err, &mismatch) || mismatch.Count != want || mismatch.Path != sentryUtilPath || !IsAdapterRegenerationBlocked(err) {
					t.Fatalf("anchor refusal = %v, want count=%d at %s", err, want, sentryUtilPath)
				}
			} else if name == "deleted file" {
				var missing *AdapterSourceMissingError
				if !errors.As(err, &missing) || missing.Path != sentryUtilPath || !IsAdapterRegenerationBlocked(err) {
					t.Fatalf("missing-file refusal = %v", err)
				}
			} else if !strings.Contains(err.Error(), "source inventory identity mismatch") {
				t.Fatalf("previous inventory refusal = %v", err)
			}
			if entries, err := os.ReadDir(scratch); err != nil || len(entries) != 0 {
				t.Fatalf("refused regeneration wrote scratch entries=%v error=%v", entries, err)
			}
		})
	}
}

func TestPortableLibcRegeneration(t *testing.T) {
	goCommand, cache := portableAdapterGo(t)
	pinned := portableAdapterModule(t, cache, libcModulePath, libcPinnedVersion())
	for _, mutation := range []string{"outside anchor", "missing declaration", "duplicate declaration", "deleted source"} {
		t.Run(mutation, func(t *testing.T) {
			candidate := filepath.Join(t.TempDir(), "module")
			copyFixtureTree(t, pinned, candidate)
			file := filepath.Join(candidate, "libc_darwin.go")
			if mutation == "deleted source" {
				if err := os.Remove(file); err != nil {
					t.Fatal(err)
				}
			} else {
				editFixtureFile(t, file, func(contents []byte) []byte {
					switch mutation {
					case "missing declaration":
						return bytes.Replace(contents, []byte("func Xgeteuid(t *TLS) types.Uid_t {"), []byte("func Xmovedgeteuid(t *TLS) types.Uid_t {"), 1)
					case "duplicate declaration":
						return append(contents, []byte("\nfunc Xgeteuid(t *TLS) types.Uid_t {\n\treturn 0\n}\n")...)
					default:
						return append(contents, []byte("\n// portable upstream change\n")...)
					}
				})
			}
			regeneration, err := RegenerateAdapter(t.Context(), AdapterRegenerationRequest{
				Module: libcModulePath, Version: "v99.0.0-fixture", Sum: "h1:fixture", PreviousModule: pinned, CandidateModule: candidate, GoCommand: goCommand, Scratch: t.TempDir(),
			})
			if mutation != "outside anchor" {
				if err == nil {
					t.Fatalf("portable adapter check Error failed: %v", err)
				}
				var blocked *AdapterRegenerationBlockedError
				if !(errors.As(err, &blocked)) {
					t.Fatalf("portable adapter check True failed: %v", errors.As(err, &blocked))
				}
				return
			}
			if err != nil {
				t.Fatalf("portable adapter check NoError failed: %v", err)
			}
			if len(regeneration.Sources) != len(libcRegenerationSources) {
				t.Fatalf("portable adapter check Len failed: %v", regeneration.Sources)
			}
			if len(regeneration.Proposed.PreparedSourceSetSHA256) != 2 {
				t.Fatalf("portable adapter check Len failed: %v", regeneration.Proposed.PreparedSourceSetSHA256)
			}
			if reflect.DeepEqual(regeneration.Previous.OriginalSourceInventorySHA256, regeneration.Proposed.OriginalSourceInventorySHA256) {
				t.Fatalf("portable adapter check NotEqual failed: %v", regeneration.Previous.OriginalSourceInventorySHA256)
			}
			root := t.TempDir()
			copyFixtureTree(t, ".", filepath.Join(root, "deterministicio"))
			edits, err := regeneration.SourceEdits(root)
			if err != nil {
				t.Fatalf("portable adapter check NoError failed: %v", err)
			}
			for _, rewrite := range regeneration.Proposed.Rewrites {
				if !strings.Contains(string(edits["deterministicio/libc_adapter.go"]), rewrite.SourceSHA256) {
					t.Fatalf("portable adapter check Contains failed: %v", string(edits["deterministicio/libc_adapter.go"]))
				}
			}
		})
	}
}

func TestPortableLibcPreparedContract(t *testing.T) {
	_, cache := portableAdapterGo(t)
	portableAdapterModule(t, cache, libcModulePath, libcPinnedVersion())
	var definition adapterDefinition
	for _, candidate := range deterministicAdapters.definitions {
		if candidate.identity.Module == libcModulePath {
			definition = candidate
		}
	}
	prepared, err := definition.implementation.prepare(cache, t.TempDir(), definition.identity)
	if err != nil {
		t.Fatal(err)
	}
	adapter := prepared.evidence
	projected := projectAdapterReplacement(Default().Identity(), adapter)
	if projected.Original.Path != adapter.Module || projected.Adapter.Path != adapter.Module || projected.ReplacementPath != adapter.ReplacementRoot || projected.ReplacementSourceInventorySHA256 != adapter.ReplacementSourceInventorySHA256 || projected.PreparedSourceSetSHA256 != adapter.PreparedSourceSetSHA256 {
		t.Fatalf("adapter projection = %#v, adapter = %#v", projected, adapter)
	}
	if adapter.Module != "modernc.org/libc" || adapter.SourceSHA256 != "sha256:46fc04624c96033980a81d8eeb9b4d73daff0c6cae511931456f2c72a75fcb7e" {
		t.Fatalf("adapter = %#v", adapter)
	}
	replacement, err := os.ReadFile(adapter.Replacement)
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"gomadOpen", "gomadRead", "gomadWrite", "gomad: unsupported modernc libc host capability: Xsocket", "gomad: unsupported modernc libc host capability: Xsystem", "gomad: unsupported modernc libc host capability: Xpause"} {
		if !strings.Contains(string(replacement), text) {
			t.Errorf("replacement omitted %q", text)
		}
	}
	if adapter.ReplacementSHA256 != digestBytes(replacement) {
		t.Fatalf("replacement digest = %q", adapter.ReplacementSHA256)
	}
	for _, name := range []string{"gomad_darwin.go", "gomad_linux.go"} {
		contents, err := os.ReadFile(filepath.Join(adapter.ReplacementRoot, name))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(contents), "internal/gomadio.Enabled") || strings.Contains(string(contents), "Temporal") || strings.Contains(string(contents), "SQLite") {
			t.Fatalf("modernc adapter %s = %s", name, contents)
		}
	}
	trampolines, err := os.ReadFile(filepath.Join(adapter.ReplacementRoot, "syscall_musl.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(trampolines), "gomadSyscall(tls, n,") != 8 {
		t.Fatalf("musl trampolines = %s", trampolines)
	}
	guarded, err := os.ReadFile(filepath.Join(adapter.ReplacementRoot, "libc_musl.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Xsystem", "Xabort", "Xsignal"} {
		if !strings.Contains(string(guarded), "gomad: unsupported modernc libc host capability: "+name) {
			t.Errorf("musl host capability %s is not guarded", name)
		}
	}
}
