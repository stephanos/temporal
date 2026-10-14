package wasi

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestHostCallActivityProfile records the diagnostic helper's nested encoding
// and transcript-hash time on the Temporal activity selection (fn-174 R5). It is
// opt-in and diagnostic-only. The same file runs unchanged in the pre-change
// baseline tree, so both sides use one collector. GOMAD_WASM_TEST_HELPER must be
// a diagnostic-profile helper built from the tree under test.
func TestHostCallActivityProfile(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_HOST_CALL_PROFILE_ROOT")
	if root == "" {
		t.Skip("set GOMAD_WASM_HOST_CALL_PROFILE_ROOT and a diagnostic GOMAD_WASM_TEST_HELPER")
	}
	if !filepath.IsAbs(root) || !filepath.IsAbs(os.Getenv("GOMAD_WASM_TEST_HELPER")) {
		t.Fatal("profile root and GOMAD_WASM_TEST_HELPER must be absolute")
	}
	repetitions := 3
	if value := os.Getenv("GOMAD_WASM_HOST_CALL_PROFILE_REPETITIONS"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 20 {
			t.Fatal("repetitions must be 1..20")
		}
		repetitions = parsed
	}
	repo, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOMAD_WASM_PROTOTYPE_ROOT", root)
	evidence := filepath.Join(root, "evidence")
	if err := os.MkdirAll(evidence, 0700); err != nil {
		t.Fatal(err)
	}
	module := os.Getenv("GOMAD_WASM_HOST_CALL_PROFILE_MODULE")
	if module == "" {
		module = filepath.Join(root, "tests.wasm")
		_, seconds := prototypeBuild(t, repo, "test", "./tests", module, "disable_grpc_modules", "test_dep", "sqlite3_dotlk")
		t.Logf("built Temporal guest in %.1fs", seconds)
	} else if !filepath.IsAbs(module) {
		t.Fatal("GOMAD_WASM_HOST_CALL_PROFILE_MODULE must be absolute")
	}
	persistence := []string{"-persistenceType=sql", "-persistenceDriver=sqlite"}
	warm := prototypeRequest(t, module, prototypeTemporalConfig(t, repo, "/workspace/tests", append([]string{"-test.run=^$"}, persistence...)...), 0, 30*time.Minute)
	runDiagnosticProfile(t, evidence, "prepare", warm, true, false)
	commit, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	var rows []map[string]any
	for _, seed := range []uint64{0, 1000003, 2000006} {
		var first Result
		for repetition := range repetitions {
			config := prototypeTemporalConfig(t, repo, "/workspace/tests", append([]string{"-test.run=^TestActivityTestSuite$"}, persistence...)...)
			request := prototypeRequest(t, module, config, seed, 30*time.Minute)
			name := "activity-seed" + stringSeed(seed) + "-rep" + strconv.Itoa(repetition)
			result := runDiagnosticProfile(t, evidence, name, request, true, true)
			report, err := decodeDiagnosticReport(result.helperStderr)
			if err != nil {
				t.Fatal(err)
			}
			if repetition == 0 {
				first = result
			} else {
				requireDiagnosticEvidence(t, first, result)
			}
			encoding, hash := report.Nested["encoding"], report.Nested["transcript_hash"]
			rows = append(rows, map[string]any{
				"seed": seed, "repetition": repetition, "calls": result.Calls,
				"encoding_count": encoding.Count, "encoding_wall_nanos": encoding.WallNanos,
				"transcript_hash_count": hash.Count, "transcript_hash_wall_nanos": hash.WallNanos,
				"encoding_plus_hash_wall_nanos":   encoding.WallNanos + hash.WallNanos,
				"authorization_to_terminal_nanos": result.AuthorizationToTerminalNanos,
				"transcript_sha256":               fmt.Sprintf("%x", result.TranscriptSHA256),
			})
			t.Logf("%s: encoding=%d/%dns transcript_hash=%d/%dns calls=%d", name, encoding.Count, encoding.WallNanos, hash.Count, hash.WallNanos, result.Calls)
		}
	}
	cpuinfo, requiredHost := diagnosticPerformanceHostEvidence("/proc/cpuinfo")
	temporalJSON(t, filepath.Join(root, "host-call-profile.json"), map[string]any{
		"diagnostic_only": true, "selection": "^TestActivityTestSuite$", "source_commit": strings.TrimSpace(string(commit)),
		"helper_sha256": warm.HelperSHA256, "module_sha256": warm.ModuleSHA256, "model_sha256": ImplementationSHA256(),
		"host_goos": runtime.GOOS, "host_goarch": runtime.GOARCH, "required_host_measured": requiredHost, "host_cpuinfo": cpuinfo,
		"repetitions": repetitions, "rows": rows,
	})
}
