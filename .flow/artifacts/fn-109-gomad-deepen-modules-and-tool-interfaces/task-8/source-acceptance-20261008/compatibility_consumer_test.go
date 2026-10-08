package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/internal/compatibilitypack/authoring"
	capabilityanalysis "go.temporal.io/server/tools/gomad3/qualification/analysis"
	"go.temporal.io/server/tools/gomad3/target"
)

type sourceDiagnosticWriter struct {
	file      io.Writer
	attempted bytes.Buffer
	calls     int
	err       error
}

func (w *sourceDiagnosticWriter) Write(p []byte) (int, error) {
	w.calls++
	if _, err := w.attempted.Write(p); err != nil {
		return 0, err
	}
	n, err := w.file.Write(p)
	w.err = err
	return n, err
}

func sourceDiagnostics(t *testing.T, output io.Writer, closed bool) *sourceDiagnosticWriter {
	t.Helper()
	if closed {
		file, err := os.CreateTemp(t.TempDir(), "closed-output")
		if err != nil {
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		output = file
	}
	return &sourceDiagnosticWriter{file: output}
}

func TestCompatibilityPackSourceUsageCharacterization(t *testing.T) {
	for _, arguments := range [][]string{nil, {"SOURCE-unknown-command"}} {
		for _, failed := range []bool{false, true} {
			var stdout, stderr bytes.Buffer
			var status, calls int
			var attempted string
			if failed {
				file, err := os.CreateTemp(t.TempDir(), "closed-output")
				if err != nil {
					t.Fatal(err)
				}
				if err := file.Close(); err != nil {
					t.Fatal(err)
				}
				writer := sourceDiagnosticWriter{file: file}
				status = runCompatibilityPack(arguments, &stdout, &writer)
				calls = writer.calls
				attempted = writer.attempted.String()
				if _, err := file.Write([]byte("probe")); !errors.Is(err, os.ErrClosed) {
					t.Fatalf("closed writer was not closed: %v", err)
				}
				if calls != 1 {
					t.Fatalf("diagnostic attempts=%d", calls)
				}
			} else {
				writer := sourceDiagnosticWriter{file: &stderr}
				status = runCompatibilityPack(arguments, &stdout, &writer)
				attempted = writer.attempted.String()
				calls = writer.calls
			}
			encoded, err := json.Marshal(map[string]any{"arguments": arguments, "failed_writer": failed, "status": status, "stdout": stdout.String(), "stderr": stderr.String(), "attempted": attempted, "write_calls": calls})
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("SOURCE_OBSERVATION %s", encoded)
		}
	}
}

func TestCompatibilityPackSourceActualOwnerConsumers(t *testing.T) {
	for _, closedDiagnostic := range []bool{false, true} {
		for _, invalidReview := range []bool{false, true} {
			for _, failedRemoval := range []bool{false, true} {
				t.Run(map[bool]string{false: "removed", true: "failed-removal"}[failedRemoval]+map[bool]string{false: "/ordinary", true: "/closed"}[closedDiagnostic]+map[bool]string{false: "/valid-review", true: "/invalid-review"}[invalidReview], func(t *testing.T) {
					root, working := t.TempDir(), t.TempDir()
					toolchain, err := filepath.Abs(filepath.Join("..", "..", ".toolchain"))
					if err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink(toolchain, filepath.Join(root, ".toolchain")); err != nil {
						t.Fatal(err)
					}
					packRoot := filepath.Join(root, "internal", "compatibilitypack", "requests")
					if err := os.MkdirAll(packRoot, 0o700); err != nil {
						t.Fatal(err)
					}
					for file, body := range map[string]string{
						"go.mod":  "module example.com/sourceconsumer\n\ngo 1.27.1\n\nrequire github.com/google/uuid v1.6.0\n",
						"go.sum":  "github.com/google/uuid v1.6.0 h1:NIvaJDMOsjHA8n1jAhLSgzrAzy1Hgr+hNrb57e+94F0=\ngithub.com/google/uuid v1.6.0/go.mod h1:TIyPZe4MgqvfeYDBFedMoGGpEw/LqOeaOT+nhxU+yHo=\n",
						"main.go": "package main\nimport \"github.com/google/uuid\"\nfunc main() { _ = uuid.Nil; panic(\"SOURCE target launched\") }\n",
					} {
						if err := os.WriteFile(filepath.Join(working, file), []byte(body), 0o600); err != nil {
							t.Fatal(err)
						}
					}
					if invalidReview {
						if err := os.WriteFile(filepath.Join(working, "main.go"), []byte("package main\nimport _ \"example.com/SOURCE-unavailable\"\nfunc main() {}\n"), 0o600); err != nil {
							t.Fatal(err)
						}
					}
					draft := authoring.Request{Schema: authoring.RequestSchema, ID: "source-pack", Target: authoring.Target{Kind: target.KindGoRun, Package: ".", TestArguments: []string{}, BuildTags: []string{}, ExpectedModule: "example.com/sourceconsumer"}, Activation: []authoring.Activation{{Path: "github.com/google/uuid"}}, Packages: []authoring.Package{{ImportPath: "github.com/google/uuid", Facts: []authoring.Fact{{Kind: authoring.FactCapability, Capability: "import:bytes", Directives: []string{}, Disposition: authoring.DispositionAllow}}}}, Owner: "source-fixture", ReviewedAt: "2026-10-08T00:00:00Z", Justification: "Fixed source-computation control.", Workloads: []string{"source-fixture"}, Platforms: []string{"linux/amd64"}}
					initial, err := canonicaljson.CanonicalJSON(draft)
					if err != nil {
						t.Fatal(err)
					}
					request := filepath.Join(packRoot, "source-pack.json")
					if err := os.WriteFile(request, initial, 0o600); err != nil {
						t.Fatal(err)
					}
					calls := 0
					remove := func(path string) error {
						calls++
						if failedRemoval {
							t.Cleanup(func() {
								if err := os.RemoveAll(path); err != nil {
									t.Error(err)
								}
							})
							return errors.New("SOURCE-remove-failed")
						}
						return os.RemoveAll(path)
					}
					prepare := func(ctx context.Context, spec target.Spec) (capabilityanalysis.PreparedCapabilityReview, error) {
						return capabilityanalysis.PrepareCapabilityReviewSourceControl(ctx, spec, remove)
					}
					var stdout, stderr bytes.Buffer
					diagnostics := sourceDiagnostics(t, &stderr, closedDiagnostic)
					status := runCompatibilityPackDiscoverWith([]string{"--root=" + root, "--request=" + request, "--working-dir=" + working}, &stdout, diagnostics, prepare)
					published, err := os.ReadFile(request)
					if err != nil {
						t.Fatal(err)
					}
					if calls != 1 {
						t.Fatalf("discover owner removal calls=%d stderr=%s", calls, &stderr)
					}
					if failedRemoval || invalidReview {
						if status != 1 || !bytes.Equal(initial, published) || stdout.Len() != 0 || diagnostics.calls != 1 {
							t.Fatalf("failed discover status=%d output=%s diagnostics=%s published=%s", status, &stdout, &stderr, published)
						}
						if failedRemoval && !strings.Contains(diagnostics.attempted.String(), "SOURCE-remove-failed") {
							t.Fatal("owner removal diagnostic missing")
						}
						if closedDiagnostic && !errors.Is(diagnostics.err, os.ErrClosed) {
							t.Fatalf("diagnostic failure not observed: %v", diagnostics.err)
						}
					} else {
						if status != 0 || stdout.Len() == 0 || stderr.Len() != 0 || bytes.Equal(initial, published) {
							t.Fatalf("discover status=%d output=%s diagnostics=%s", status, &stdout, &stderr)
						}
						decoded, err := authoring.DecodeRequest(published)
						if err != nil {
							t.Fatal(err)
						}
						if decoded.Activation[0].Evidence.Sum == "" || len(decoded.Packages[0].Evidence.GoSources) == 0 {
							t.Fatalf("incomplete published identity: %#v", decoded)
						}
					}
					encoded, err := json.Marshal(map[string]any{"operation": "discover", "failed_removal": failedRemoval, "invalid_review": invalidReview, "failed_writer": closedDiagnostic, "status": status, "stdout": stdout.String(), "stderr": stderr.String(), "attempted": diagnostics.attempted.String(), "write_calls": diagnostics.calls, "request": string(published), "remove_calls": calls})
					if err != nil {
						t.Fatal(err)
					}
					t.Logf("SOURCE_OBSERVATION %s", encoded)
					if failedRemoval || invalidReview {
						return
					}
					for _, qualifyInvalid := range []bool{false, true} {
						for _, cleanupFails := range []bool{false, true} {
							stdout.Reset()
							stderr.Reset()
							calls = 0
							body := "package main\nimport \"github.com/google/uuid\"\nfunc main() { _ = uuid.Nil; panic(\"SOURCE target launched\") }\n"
							if qualifyInvalid {
								body = "package main\nimport _ \"example.com/SOURCE-unavailable\"\nfunc main() {}\n"
							}
							if err := os.WriteFile(filepath.Join(working, "main.go"), []byte(body), 0o600); err != nil {
								t.Fatal(err)
							}
							diagnostics = sourceDiagnostics(t, &stderr, closedDiagnostic)
							qualifyRemove := func(path string) error {
								calls++
								if cleanupFails {
									t.Cleanup(func() {
										if err := os.RemoveAll(path); err != nil {
											t.Error(err)
										}
									})
									return errors.New("SOURCE-remove-failed")
								}
								return os.RemoveAll(path)
							}
							qualifyPrepare := func(ctx context.Context, spec target.Spec) (capabilityanalysis.PreparedCapabilityReview, error) {
								return capabilityanalysis.PrepareCapabilityReviewSourceControl(ctx, spec, qualifyRemove)
							}
							status = sourceQualify(root, request, working, &stdout, diagnostics, qualifyPrepare)
							want := 0
							if cleanupFails || qualifyInvalid {
								want = 1
							}
							if status != want || calls != 1 {
								t.Fatalf("qualify status=%d want=%d owner calls=%d diagnostics=%s", status, want, calls, &stderr)
							}
							if cleanupFails || qualifyInvalid {
								if stdout.Len() != 0 || diagnostics.calls != 1 {
									t.Fatalf("qualify failure output=%s diagnostic=%s", &stdout, &stderr)
								}
								if cleanupFails && !strings.Contains(diagnostics.attempted.String(), "SOURCE-remove-failed") {
									t.Fatal("qualify cleanup diagnostic missing")
								}
								if closedDiagnostic && !errors.Is(diagnostics.err, os.ErrClosed) {
									t.Fatalf("qualify diagnostic failure not observed: %v", diagnostics.err)
								}
							} else if stdout.String() != "qualified compatibility-pack request source-pack\n" || stderr.Len() != 0 {
								t.Fatalf("qualify output=%s diagnostic=%s", &stdout, &stderr)
							}
							after, err := os.ReadFile(request)
							if err != nil {
								t.Fatal(err)
							}
							if !bytes.Equal(published, after) {
								t.Fatal("qualify mutated request")
							}
							encoded, err := json.Marshal(map[string]any{"operation": "qualify", "failed_removal": cleanupFails, "invalid_review": qualifyInvalid, "failed_writer": closedDiagnostic, "status": status, "stdout": stdout.String(), "stderr": stderr.String(), "attempted": diagnostics.attempted.String(), "write_calls": diagnostics.calls, "request": string(after), "remove_calls": calls})
							if err != nil {
								t.Fatal(err)
							}
							t.Logf("SOURCE_OBSERVATION %s", encoded)
						}
					}
				})
			}
		}
	}
}
