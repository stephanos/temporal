package wasi

// Temporary, uncommitted: diagnostic split of the smoke-workflow and
// smoke-activity selections. Delete after profiling.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestScratchSmokeProfile(t *testing.T) {
	root := prototypeRoot(t)
	repo, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	modules := filepath.Join(root, "modules")
	if err := os.MkdirAll(modules, 0700); err != nil {
		t.Fatal(err)
	}
	module := filepath.Join(modules, "tests.wasm")
	if _, err := os.Stat(module); err != nil {
		_, seconds := prototypeBuild(t, repo, "test", "./tests", module, "disable_grpc_modules", "test_dep", "sqlite3_dotlk")
		t.Logf("built tests.wasm in %.1fs", seconds)
	}
	persistence := []string{"-persistenceType=sql", "-persistenceDriver=sqlite"}
	suites := []struct{ name, run string }{
		{"workflow", "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$"},
		{"activity", "^TestActivityTestSuite$"},
	}
	evidence := filepath.Join(root, "evidence")
	if err := os.MkdirAll(evidence, 0700); err != nil {
		t.Fatal(err)
	}
	helpers := strings.Split(os.Getenv("GOMAD_SCRATCH_HELPERS"), ",")
	for _, owner := range helpers {
		name, path, _ := strings.Cut(owner, "=")
		instrumented := strings.HasPrefix(name, "diag")
		t.Setenv("GOMAD_WASM_TEST_HELPER", path)
		warm := prototypeRequest(t, module, prototypeTemporalConfig(t, repo, "/workspace/tests", append([]string{"-test.run=^$"}, persistence...)...), 0, 30*time.Minute)
		if os.Getenv("GOMAD_SCRATCH_SKIP_PREPARE") == "" {
			runDiagnosticProfile(t, evidence, name+"-prepare", warm, instrumented, false)
		}
		for _, suite := range suites {
			if only := os.Getenv("GOMAD_SCRATCH_ONLY"); only != "" && only != suite.name {
				continue
			}
			for _, seed := range []uint64{0, 1000003} {
				config := prototypeTemporalConfig(t, repo, "/workspace/tests", append([]string{"-test.run=" + suite.run}, persistence...)...)
				request := prototypeRequest(t, module, config, seed, 30*time.Minute)
				if os.Getenv("GOMAD_SCRATCH_TRANSCRIPT") != "" {
					request.DigestTranscript = false
				}
				started := time.Now()
				result := runDiagnosticProfile(t, evidence, name+"-"+suite.name+"-seed"+stringSeed(seed), request, instrumented, true)
				if !request.DigestTranscript {
					operations := map[string]int{}
					for _, line := range bytes.Split(result.Transcript, []byte{'\n'}) {
						var frame struct{ Type, Op string }
						if json.Unmarshal(line, &frame) == nil && frame.Type == "call" {
							operations[frame.Op]++
						}
					}
					t.Logf("operations: %v", operations)
				}
				t.Logf("%s %s seed %d: run wall %s, authorized→terminal %s", name, suite.name, seed, time.Since(started), time.Duration(result.AuthorizationToTerminalNanos))
			}
		}
	}
}
