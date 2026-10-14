package wasi

// Temporary, uncommitted: diagnostic split of the smoke-workflow and
// smoke-activity selections. Delete after profiling.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestScratchSmokeProfile(t *testing.T) {
	root := prototypeRoot(t)
	repo, err := filepath.Abs("../../../../../..")
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
					operations, err := scratchOperationCounts(result.Transcript)
					if err != nil {
						t.Fatal(err)
					}
					t.Logf("operations: %v", operations)
				}
				t.Logf("%s %s seed %d: run wall %s, authorized→terminal %s", name, suite.name, seed, time.Since(started), time.Duration(result.AuthorizationToTerminalNanos))
			}
		}
	}
}

func scratchOperationCounts(data []byte) (map[string]uint64, error) {
	records, err := DecodeTranscript(data)
	if err != nil {
		return nil, err
	}
	operations := map[string]uint64{}
	for _, record := range records {
		operations[record.Op] += record.Count
	}
	return operations, nil
}

func TestScratchOperationCountsPreservesBinaryCountsAndRefusesMalformed(t *testing.T) {
	// Frozen clock run followed by yield; the run represents three clock reads.
	clock := "\x00\x00\x00\x00" + strings.Repeat("\x00", 8) + "\x01" + strings.Repeat("\x00", 7) + "\x03" + strings.Repeat("\x00", 7)
	data := []byte("GMWT\x01\x00\x00\x00" + operationCountRecord(1, 33, clock) + operationCountRecord(4, 2, ""))
	counts, err := scratchOperationCounts(data)
	if err != nil || len(counts) != 2 || counts["clock_time_get"] != 3 || counts["sched_yield"] != 1 {
		t.Fatalf("binary operation counts = %v, %v", counts, err)
	}
	for _, malformed := range [][]byte{data[:len(data)-1], []byte(`{"type":"call","op":"sched_yield"}` + "\n")} {
		if _, err := scratchOperationCounts(malformed); err == nil {
			t.Fatal("malformed or retired transcript accepted")
		}
	}
}
