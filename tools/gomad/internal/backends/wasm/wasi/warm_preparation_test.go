package wasi

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	hostfs "go.temporal.io/server/tools/gomad/internal/host/fs"
)

const warmPreparationSchema = "gomad3.wasm-warm-preparation/v1"

type warmPreparationRun struct {
	Telemetry                                   Telemetry                  `json:"telemetry"`
	Phases                                      map[string]diagnosticPhase `json:"phases"`
	WallNanos, TerminalToReapNanos              uint64
	EvidenceSHA256                              string
	Exited, Reaped                              bool
	ObservationPath, FinishPath, DiagnosticPath string
	RawObservationPath                          string
}

// The measurement projection is shared with the test-only historical collector;
// production host receipts remain unchanged and are retained separately.
type warmPreparationObservation struct {
	TerminalReceivedAt, SupervisorReturnedAt          time.Time
	TerminalToReapNanos                               uint64
	TerminalFrame                                     []byte
	Result                                            Result
	Error                                             string
	HostExited, GroupGone, Cancelled, WatchdogTimeout bool
	HostExitCode                                      int
}

func warmPreparationObserve(observation terminalExitObservation) warmPreparationObservation {
	return warmPreparationObservation{TerminalReceivedAt: observation.TerminalReceivedAt, SupervisorReturnedAt: observation.SupervisorReturnedAt, TerminalToReapNanos: observation.TerminalToReapNanos, TerminalFrame: observation.TerminalFrame, Result: observation.Result, Error: observation.Error, HostExited: observation.Host.Termination == "exit", GroupGone: observation.Host.GroupGone, Cancelled: observation.Host.Cancelled, WatchdogTimeout: observation.Host.WatchdogTimeout, HostExitCode: observation.Host.ExitCode}
}

func TestWarmPreparationObservationPrivateFinishControls(t *testing.T) {
	code := uint32(0)
	observation := terminalExitObservation{TerminalReceivedAt: time.Unix(1, 0), SupervisorReturnedAt: time.Unix(1, 1), TerminalToReapNanos: 1, Result: Result{Termination: "exit", ExitCode: &code, runtimeTerminal: [16]byte{1}}}
	observation.Host.Termination, observation.Host.GroupGone = "exit", true
	projection := warmPreparationObserve(observation)
	if !projection.HostExited || !projection.GroupGone || projection.TerminalToReapNanos != 1 || projection.SupervisorReturnedAt != observation.SupervisorReturnedAt {
		t.Fatal("actual terminal/host measurements lost")
	}
	data, err := json.Marshal(projection)
	if err != nil {
		t.Fatal(err)
	}
	var imported warmPreparationObservation
	if err := json.Unmarshal(data, &imported); err != nil {
		t.Fatal(err)
	}
	if warmPreparationEvidence(imported.Result) == warmPreparationEvidence(observation.Result) {
		t.Fatal("JSON unexpectedly preserved private finish; missing sidecar must remain visible")
	}
	imported.Result.runtimeTerminal = observation.Result.runtimeTerminal
	if warmPreparationEvidence(imported.Result) != warmPreparationEvidence(observation.Result) {
		t.Fatal("finish sidecar did not restore exact behavior")
	}
}

type warmPreparationSeries struct {
	Schema, Role, SourceCommit, SourceRoot, DriverSHA256     string
	HostBootID, HostCPUInfo, ConfigSHA256                    string
	Engine                                                   EngineIdentity
	ModuleSHA256, HelperSHA256, RuntimeIdentity, ModelSHA256 string
	Seed                                                     uint64
	ArtifactBytes, EnvelopeBytes                             uint64
	Inputs                                                   map[string]string
	HelperReceipt, ModuleReceipt, DriverPath, ArtifactPath   string
	CollectionPath                                           string
	Runs                                                     []warmPreparationRun
	Warmup                                                   *warmPreparationRun
	ComparisonSHA256                                         string `json:"-"`
}

func warmPreparationValidate(series warmPreparationSeries, candidate bool) error {
	role := "baseline"
	if candidate {
		role = "candidate"
	}
	if series.Schema != warmPreparationSchema || series.Role != role || len(series.Runs) < 5 || !warmPreparationHex(series.SourceCommit, 40) || !warmPreparationHex(series.DriverSHA256, 64) || series.SourceRoot == "" || len(series.Inputs) == 0 || !warmPreparationHex(series.ModuleSHA256, 64) || !warmPreparationHex(series.HelperSHA256, 64) || !warmPreparationHex(series.ConfigSHA256, 64) || series.RuntimeIdentity == "" || series.ModelSHA256 == "" || series.ArtifactBytes == 0 || series.ArtifactBytes >= series.EnvelopeBytes {
		return fmt.Errorf("missing source, build, module, artifact or five-run bindings")
	}
	if series.HostBootID == "" || series.Engine.Name != "wasmtime" || series.Engine.Version != "47.0.3" || series.Engine.Configuration == "" || !diagnosticPerformanceHost(series.Engine.HostOS, "arm64", series.HostCPUInfo) || series.Engine.HostArch != "aarch64" {
		return fmt.Errorf("real Linux/arm64 host binding absent")
	}
	var prepare []uint64
	for i, run := range series.Runs {
		if !run.Telemetry.CacheHit || run.Telemetry.CompileNanos != 0 || run.Telemetry.PrepareNanos == 0 || run.WallNanos == 0 || !run.Exited || !run.Reaped || run.TerminalToReapNanos == 0 || candidate && run.TerminalToReapNanos > uint64(time.Second) || !warmPreparationHex(run.EvidenceSHA256, 64) || run.EvidenceSHA256 != series.Runs[0].EvidenceSHA256 {
			return fmt.Errorf("run %d is not a warm successful evidence-identical reaped run", i)
		}
		for _, phase := range strings.Fields("cache_read envelope_verify deserialize cache_identity") {
			if p, ok := run.Phases[phase]; !ok || p.Count == 0 || p.WallNanos > run.Telemetry.PrepareNanos {
				return fmt.Errorf("run %d phase %s absent or inconsistent", i, phase)
			}
		}
		prepare = append(prepare, run.Telemetry.PrepareNanos)
	}
	if candidate && variantTiming(prepare).Median > uint64(time.Second) {
		return fmt.Errorf("warm preparation median exceeds one second")
	}
	return nil
}

func warmPreparationCompare(before, after warmPreparationSeries) error {
	if err := warmPreparationValidate(before, false); err != nil {
		return fmt.Errorf("baseline: %w", err)
	}
	if err := warmPreparationValidate(after, true); err != nil {
		return fmt.Errorf("candidate: %w", err)
	}
	if before.HostBootID != after.HostBootID || before.HostCPUInfo != after.HostCPUInfo || before.ModuleSHA256 != after.ModuleSHA256 || before.Seed != after.Seed || before.ConfigSHA256 != after.ConfigSHA256 || before.RuntimeIdentity != after.RuntimeIdentity {
		return fmt.Errorf("before/after must share the real host, module, seed, modeled configuration and runtime")
	}
	if !warmPreparationHex(before.ComparisonSHA256, 64) || !warmPreparationHex(after.ComparisonSHA256, 64) || before.ComparisonSHA256 != after.ComparisonSHA256 {
		return fmt.Errorf("choice/WASI/network/stdout/stderr/terminal evidence changed")
	}
	if before.Inputs[before.ModuleReceipt] == "" || before.Inputs[before.ModuleReceipt] != after.Inputs[after.ModuleReceipt] {
		return fmt.Errorf("historical origin must bind the authentic candidate-owner-validated current module producer")
	}
	return nil
}

func warmPreparationHex(value string, size int) bool {
	_, err := hex.DecodeString(value)
	return len(value) == size && err == nil
}

type warmPreparationCollection struct {
	Schema                                                string `json:"schema"`
	Accepted                                              bool   `json:"accepted"`
	CollectionValid                                       bool   `json:"collection_valid"`
	Failure                                               string `json:"failure"`
	UntimedRunError, BindingSHA256, UntimedEvidenceSHA256 string
	ResultPath, CanonicalPath, ChoicePath, FinishPath     string
	SummaryNanos                                          map[string]variantTimingRange
}

type warmPreparationRawObservation struct {
	TerminalReceivedAt, SupervisorReturnedAt time.Time
	TerminalToReapNanos                      uint64
	TerminalFrame                            []byte
	Result                                   Result
	Error                                    string
	Host                                     struct {
		Termination                           string
		ExitCode                              int
		GroupGone, Cancelled, WatchdogTimeout bool
	}
}

func warmPreparationCollectionBinding(series warmPreparationSeries) string {
	series.Inputs, series.CollectionPath = nil, ""
	data, _ := json.Marshal(series)
	return digest(data)
}

func warmPreparationProducerCheck(series warmPreparationSeries, currentReceipt string, currentInputs map[string]string) error {
	if series.Inputs[series.ModuleReceipt] == "" || series.Inputs[series.ModuleReceipt] != currentInputs[currentReceipt] {
		return fmt.Errorf("baseline module producer is not the authentic current owner-validated receipt")
	}
	for path, sha := range currentInputs {
		if path != currentReceipt && series.Inputs[path] != sha {
			return fmt.Errorf("baseline lacks authentic current producer source closure at %s", path)
		}
	}
	return nil
}

func TestWarmPreparationCurrentProducerClosureControls(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(map[string]string)
		want   bool
	}{
		{"same-owner-validated-producer", func(map[string]string) {}, true},
		{"missing-current-source", func(inputs map[string]string) { delete(inputs, "/current-source.go") }, false},
		{"changed-current-source", func(inputs map[string]string) { inputs["/current-source.go"] = strings.Repeat("f", 64) }, false},
		{"changed-current-producer", func(inputs map[string]string) { inputs["/module-receipt.json"] = strings.Repeat("f", 64) }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			series := warmPreparationFixture()
			series.Inputs["/current-source.go"] = strings.Repeat("b", 64)
			current := maps.Clone(series.Inputs)
			test.mutate(series.Inputs)
			if err := warmPreparationProducerCheck(series, series.ModuleReceipt, current); (err == nil) != test.want {
				t.Fatalf("current producer closure=%v want %v", err, test.want)
			}
		})
	}
}

func warmPreparationCollectionCheck(series warmPreparationSeries, read func(string) ([]byte, error)) error {
	bound := func(path string) ([]byte, error) {
		if !filepath.IsAbs(path) || !warmPreparationHex(series.Inputs[path], 64) {
			return nil, fmt.Errorf("missing bound collection input %s", path)
		}
		data, err := read(path)
		if err != nil {
			return nil, err
		}
		if digest(data) != series.Inputs[path] {
			return nil, fmt.Errorf("changed collection input %s", path)
		}
		return data, nil
	}
	decode := func(path string, value any) error {
		data, err := bound(path)
		if err != nil {
			return err
		}
		return json.Unmarshal(data, value)
	}
	var collection warmPreparationCollection
	if err := decode(series.CollectionPath, &collection); err != nil {
		return err
	}
	if collection.Schema != "gomad3.wasm-warm-preparation-collection/v1" || collection.Accepted || !collection.CollectionValid || collection.Failure != "" || collection.UntimedRunError != "" || collection.BindingSHA256 != warmPreparationCollectionBinding(series) || !reflect.DeepEqual(collection.SummaryNanos, warmPreparationSummary(series)) || len(series.Runs) < 5 {
		return fmt.Errorf("failed or changed source/sample-bound collection")
	}
	var result Result
	if err := decode(collection.ResultPath, &result); err != nil {
		return err
	}
	finish, err := bound(collection.FinishPath)
	if err != nil {
		return err
	}
	if len(finish) != len(result.runtimeTerminal) {
		return fmt.Errorf("incomplete original Run finish")
	}
	copy(result.runtimeTerminal[:], finish)
	canonical, err := bound(collection.CanonicalPath)
	if err != nil {
		return err
	}
	choices, err := bound(collection.ChoicePath)
	if err != nil {
		return err
	}
	if !bytes.Equal(result.Transcript, canonical) || sha256.Sum256(canonical) != result.TranscriptSHA256 || !bytes.Equal(result.ChoiceTrace.Bytes, choices) || !result.Reaped || result.Cancelled || result.WatchdogTimeout || result.Termination != "exit" || result.ExitCode == nil || *result.ExitCode != 0 || result.Started == nil || result.Started.Engine != series.Engine || result.Started.ModuleSHA256 != series.ModuleSHA256 || warmPreparationEvidence(result) != collection.UntimedEvidenceSHA256 || collection.UntimedEvidenceSHA256 != series.Runs[0].EvidenceSHA256 {
		return fmt.Errorf("original Run retained bytes or behavior do not support this collection")
	}
	if series.Warmup == nil {
		return fmt.Errorf("missing actual warm-up observation")
	}
	rawRuns := append([]warmPreparationRun{*series.Warmup}, series.Runs...)
	for _, run := range rawRuns {
		var raw warmPreparationRawObservation
		if err := decode(run.RawObservationPath, &raw); err != nil {
			return err
		}
		var projected warmPreparationObservation
		if err := decode(run.ObservationPath, &projected); err != nil {
			return err
		}
		finish, err := bound(run.FinishPath)
		if err != nil {
			return err
		}
		if len(finish) != len(raw.Result.runtimeTerminal) {
			return fmt.Errorf("incomplete timed finish")
		}
		copy(raw.Result.runtimeTerminal[:], finish)
		copy(projected.Result.runtimeTerminal[:], finish)
		if raw.Error != "" || raw.Error != projected.Error || raw.TerminalReceivedAt != projected.TerminalReceivedAt || raw.SupervisorReturnedAt != projected.SupervisorReturnedAt || raw.TerminalToReapNanos != projected.TerminalToReapNanos || !bytes.Equal(raw.TerminalFrame, projected.TerminalFrame) || raw.Host.Termination != "exit" || !projected.HostExited || raw.Host.ExitCode != 0 || raw.Host.ExitCode != projected.HostExitCode || !raw.Host.GroupGone || raw.Host.GroupGone != projected.GroupGone || raw.Host.Cancelled || raw.Host.Cancelled != projected.Cancelled || raw.Host.WatchdogTimeout || raw.Host.WatchdogTimeout != projected.WatchdogTimeout || !reflect.DeepEqual(raw.Result, projected.Result) || warmPreparationEvidence(raw.Result) != run.EvidenceSHA256 || run.EvidenceSHA256 != collection.UntimedEvidenceSHA256 {
			return fmt.Errorf("original host observation differs from measured projection")
		}
	}
	return nil
}

func warmPreparationPublishCollection(t *testing.T, evidence, name string, series *warmPreparationSeries, retained Result, runError, failed string) string {
	t.Helper()
	collection := warmPreparationCollection{Schema: "gomad3.wasm-warm-preparation-collection/v1", CollectionValid: failed == "", Failure: failed, UntimedRunError: runError, BindingSHA256: warmPreparationCollectionBinding(*series), UntimedEvidenceSHA256: warmPreparationEvidence(retained), ResultPath: filepath.Join(evidence, "untimed-evidence-observed-result.json"), CanonicalPath: filepath.Join(evidence, "untimed-evidence-canonical.bin"), ChoicePath: filepath.Join(evidence, "untimed-evidence-choice.bin"), FinishPath: filepath.Join(evidence, "untimed-evidence-finish.bin"), SummaryNanos: warmPreparationSummary(*series)}
	for _, path := range []string{collection.ResultPath, collection.CanonicalPath, collection.ChoicePath, collection.FinishPath} {
		series.Inputs[path] = fileHash(t, path)
	}
	series.CollectionPath = filepath.Join(evidence, name)
	temporalJSON(t, series.CollectionPath, collection)
	series.Inputs[series.CollectionPath] = fileHash(t, series.CollectionPath)
	if failed == "" {
		if err := warmPreparationCollectionCheck(*series, os.ReadFile); err != nil {
			failed = err.Error()
			collection.CollectionValid, collection.Failure = false, failed
			temporalJSON(t, series.CollectionPath, collection)
			series.Inputs[series.CollectionPath] = fileHash(t, series.CollectionPath)
		}
	}
	temporalJSON(t, filepath.Join(evidence, "series.json"), *series)
	return failed
}

func TestWarmPreparationCollectionControls(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*warmPreparationSeries, *warmPreparationCollection, map[string][]byte)
		want   bool
	}{
		{"successful-original-run", func(*warmPreparationSeries, *warmPreparationCollection, map[string][]byte) {}, true},
		{"collection-failed-five-good-samples", func(_ *warmPreparationSeries, c *warmPreparationCollection, _ map[string][]byte) {
			c.CollectionValid = false
		}, false},
		{"reported-failure", func(_ *warmPreparationSeries, c *warmPreparationCollection, _ map[string][]byte) {
			c.Failure = "warmup changed"
		}, false},
		{"original-run-error", func(_ *warmPreparationSeries, c *warmPreparationCollection, _ map[string][]byte) {
			c.UntimedRunError = "original Run failed"
		}, false},
		{"changed-source-binding", func(s *warmPreparationSeries, _ *warmPreparationCollection, _ map[string][]byte) {
			s.SourceCommit = strings.Repeat("b", 40)
		}, false},
		{"changed-sample-binding", func(s *warmPreparationSeries, _ *warmPreparationCollection, _ map[string][]byte) {
			s.Runs[1].WallNanos++
		}, false},
		{"dropped-warmup", func(s *warmPreparationSeries, _ *warmPreparationCollection, _ map[string][]byte) { s.Warmup = nil }, false},
		{"dropped-marker", func(s *warmPreparationSeries, _ *warmPreparationCollection, files map[string][]byte) {
			delete(files, s.CollectionPath)
		}, false},
		{"unbound-marker", func(s *warmPreparationSeries, _ *warmPreparationCollection, _ map[string][]byte) {
			delete(s.Inputs, s.CollectionPath)
		}, false},
		{"dropped-canonical", func(_ *warmPreparationSeries, c *warmPreparationCollection, files map[string][]byte) {
			delete(files, c.CanonicalPath)
		}, false},
		{"tampered-canonical", func(_ *warmPreparationSeries, c *warmPreparationCollection, files map[string][]byte) {
			files[c.CanonicalPath] = []byte("tampered")
		}, false},
		{"rehashed-wrong-canonical", func(s *warmPreparationSeries, c *warmPreparationCollection, files map[string][]byte) {
			files[c.CanonicalPath] = []byte("tampered")
			s.Inputs[c.CanonicalPath] = digest(files[c.CanonicalPath])
		}, false},
		{"rehashed-wrong-finish", func(s *warmPreparationSeries, c *warmPreparationCollection, files map[string][]byte) {
			files[c.FinishPath][0]++
			s.Inputs[c.FinishPath] = digest(files[c.FinishPath])
		}, false},
		{"dropped-original-observation", func(s *warmPreparationSeries, _ *warmPreparationCollection, files map[string][]byte) {
			delete(files, s.Runs[0].RawObservationPath)
		}, false},
		{"tampered-original-observation", func(s *warmPreparationSeries, _ *warmPreparationCollection, files map[string][]byte) {
			files[s.Runs[0].RawObservationPath] = []byte("{}")
		}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			series := warmPreparationFixture()
			series.Role, series.CollectionPath = "baseline", "/collection.json"
			code := uint32(0)
			result := Result{Termination: "exit", ExitCode: &code, Reaped: true, Started: &Started{Engine: series.Engine, ModuleSHA256: series.ModuleSHA256}, Transcript: []byte("canonical"), TranscriptSHA256: sha256.Sum256([]byte("canonical")), runtimeTerminal: [16]byte{1}}
			result.ChoiceTrace.Bytes = []byte("choices")
			files := map[string][]byte{}
			put := func(path string, data []byte) { files[path] = data; series.Inputs[path] = digest(data) }
			encode := func(value any) []byte {
				data, err := json.Marshal(value)
				if err != nil {
					t.Fatal(err)
				}
				return data
			}
			for i := range series.Runs {
				run := &series.Runs[i]
				run.EvidenceSHA256 = warmPreparationEvidence(result)
				run.RawObservationPath, run.ObservationPath = fmt.Sprintf("/run-%d-raw.json", i), fmt.Sprintf("/run-%d-projected.json", i)
				run.FinishPath = fmt.Sprintf("/run-%d-finish", i)
				observation := terminalExitObservation{TerminalReceivedAt: time.Unix(1, 0), SupervisorReturnedAt: time.Unix(2, 0), TerminalToReapNanos: uint64(time.Second), Result: result}
				observation.Host.Termination, observation.Host.GroupGone = "exit", true
				put(run.RawObservationPath, encode(observation))
				put(run.ObservationPath, encode(warmPreparationObserve(observation)))
				put(run.FinishPath, bytes.Clone(result.runtimeTerminal[:]))
			}
			warmup := series.Runs[0]
			series.Warmup = &warmup
			collection := warmPreparationCollection{Schema: "gomad3.wasm-warm-preparation-collection/v1", CollectionValid: true, BindingSHA256: warmPreparationCollectionBinding(series), UntimedEvidenceSHA256: warmPreparationEvidence(result), ResultPath: "/untimed-result.json", CanonicalPath: "/untimed-canonical", ChoicePath: "/untimed-choice", FinishPath: "/untimed-finish", SummaryNanos: warmPreparationSummary(series)}
			put(collection.ResultPath, encode(result))
			put(collection.CanonicalPath, result.Transcript)
			put(collection.ChoicePath, result.ChoiceTrace.Bytes)
			put(collection.FinishPath, bytes.Clone(result.runtimeTerminal[:]))
			test.mutate(&series, &collection, files)
			marker := encode(collection)
			if test.name != "dropped-marker" {
				files[series.CollectionPath] = marker
			}
			if test.name != "unbound-marker" {
				series.Inputs[series.CollectionPath] = digest(marker)
			}
			err := warmPreparationCollectionCheck(series, func(path string) ([]byte, error) {
				data, ok := files[path]
				if !ok {
					return nil, os.ErrNotExist
				}
				return data, nil
			})
			if (err == nil) != test.want {
				t.Fatalf("collection admission=%v want %v", err, test.want)
			}
		})
	}
}

func warmPreparationFixture() warmPreparationSeries {
	series := warmPreparationSeries{Schema: warmPreparationSchema, Role: "candidate", SourceCommit: strings.Repeat("a", 40), SourceRoot: "/source", DriverSHA256: strings.Repeat("d", 64), HostBootID: "same-boot", HostCPUInfo: "Features: fp asimd\nCPU architecture: 8\n", ConfigSHA256: strings.Repeat("c", 64), Engine: PinnedEngine(), ModuleSHA256: strings.Repeat("a", 64), HelperSHA256: strings.Repeat("b", 64), RuntimeIdentity: "runtime", ModelSHA256: "model", ArtifactBytes: 100, EnvelopeBytes: 200, Inputs: map[string]string{"/source/driver.go": strings.Repeat("d", 64)}}
	series.ModuleReceipt = "/module-receipt.json"
	series.ComparisonSHA256 = strings.Repeat("e", 64)
	series.Inputs[series.ModuleReceipt] = strings.Repeat("a", 64)
	for range 5 {
		series.Runs = append(series.Runs, warmPreparationRun{Telemetry: Telemetry{CacheHit: true, PrepareNanos: uint64(time.Second)}, Phases: map[string]diagnosticPhase{"cache_read": {Count: 1, WallNanos: 2}, "envelope_verify": {Count: 1, WallNanos: 3}, "deserialize": {Count: 1, WallNanos: 4}, "cache_identity": {Count: 1, WallNanos: 1}}, WallNanos: 10, TerminalToReapNanos: uint64(time.Second), EvidenceSHA256: strings.Repeat("e", 64), Exited: true, Reaped: true})
	}
	return series
}

func TestWarmPreparationAdmissionControls(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*warmPreparationSeries)
		want   bool
	}{
		{"five-warm-at-bound", func(*warmPreparationSeries) {}, true},
		{"four-runs", func(s *warmPreparationSeries) { s.Runs = s.Runs[:4] }, false},
		{"cold-run", func(s *warmPreparationSeries) { s.Runs[2].Telemetry.CacheHit = false }, false},
		{"compiled-run", func(s *warmPreparationSeries) { s.Runs[2].Telemetry.CompileNanos = 1 }, false},
		{"prepare-over-bound", func(s *warmPreparationSeries) {
			for i := range s.Runs {
				s.Runs[i].Telemetry.PrepareNanos++
			}
		}, false},
		{"one-slow-prepare-median-still-bound", func(s *warmPreparationSeries) { s.Runs[0].Telemetry.PrepareNanos++ }, true},
		{"slow-terminal", func(s *warmPreparationSeries) { s.Runs[2].TerminalToReapNanos++ }, false},
		{"missing-terminal", func(s *warmPreparationSeries) { s.Runs[2].TerminalToReapNanos = 0 }, false},
		{"unreaped", func(s *warmPreparationSeries) { s.Runs[2].Reaped = false }, false},
		{"failed-exit", func(s *warmPreparationSeries) { s.Runs[2].Exited = false }, false},
		{"missing-phase", func(s *warmPreparationSeries) { delete(s.Runs[2].Phases, "deserialize") }, false},
		{"missing-phase-sample", func(s *warmPreparationSeries) { s.Runs[2].Phases["deserialize"] = diagnosticPhase{} }, false},
		{"unstable-evidence", func(s *warmPreparationSeries) { s.Runs[2].EvidenceSHA256 = strings.Repeat("f", 64) }, false},
		{"emulated-host", func(s *warmPreparationSeries) { s.HostCPUInfo = "vendor_id: GenuineIntel\n" }, false},
		{"missing-host-binding", func(s *warmPreparationSeries) { s.HostBootID = "" }, false},
		{"missing-engine-binding", func(s *warmPreparationSeries) { s.Engine.Configuration = "" }, false},
		{"missing-source-binding", func(s *warmPreparationSeries) { s.SourceCommit = "" }, false},
		{"missing-receipts", func(s *warmPreparationSeries) { s.Inputs = nil }, false},
		{"wrong-artifact-size", func(s *warmPreparationSeries) { s.ArtifactBytes = s.EnvelopeBytes }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			series := warmPreparationFixture()
			test.mutate(&series)
			if err := warmPreparationValidate(series, true); (err == nil) != test.want {
				t.Fatalf("admission=%v, want %v", err, test.want)
			}
		})
	}
}

func TestWarmPreparationComparisonControls(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*warmPreparationSeries)
		want   bool
	}{
		{"separate-engine-helper-model-bindings", func(s *warmPreparationSeries) {
			s.Engine.Configuration += "-historical"
			s.HelperSHA256 = strings.Repeat("f", 64)
			s.ModelSHA256 = "historical-model"
			s.DriverSHA256 = strings.Repeat("f", 64)
			s.SourceCommit = strings.Repeat("b", 40)
		}, true},
		{"different-host-boot", func(s *warmPreparationSeries) { s.HostBootID = "other-boot" }, false},
		{"different-module", func(s *warmPreparationSeries) { s.ModuleSHA256 = strings.Repeat("f", 64) }, false},
		{"different-seed", func(s *warmPreparationSeries) { s.Seed++ }, false},
		{"different-config", func(s *warmPreparationSeries) { s.ConfigSHA256 = strings.Repeat("f", 64) }, false},
		{"different-runtime", func(s *warmPreparationSeries) { s.RuntimeIdentity = "other-runtime" }, false},
		{"different-current-producer", func(s *warmPreparationSeries) { s.Inputs[s.ModuleReceipt] = strings.Repeat("f", 64) }, false},
		{"different-evidence", func(s *warmPreparationSeries) {
			s.ComparisonSHA256 = strings.Repeat("f", 64)
			for i := range s.Runs {
				s.Runs[i].EvidenceSHA256 = strings.Repeat("f", 64)
			}
		}, false},
		{"different-raw-encoding-same-validated-behavior", func(s *warmPreparationSeries) {
			for i := range s.Runs {
				s.Runs[i].EvidenceSHA256 = strings.Repeat("f", 64)
			}
		}, true},
		{"missing-validated-comparison", func(s *warmPreparationSeries) { s.ComparisonSHA256 = "" }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			before, after := warmPreparationFixture(), warmPreparationFixture()
			before.Role = "baseline"
			test.mutate(&before)
			if err := warmPreparationCompare(before, after); (err == nil) != test.want {
				t.Fatalf("comparison=%v, want %v", err, test.want)
			}
		})
	}
}

func warmPreparationEvidence(result Result) string {
	data, _ := json.Marshal(struct {
		Stdout, Stderr, Choices, Network []byte
		WASI                             [32]byte
		Terminal                         [16]byte
		Termination                      string
		ExitCode                         *uint32
	}{result.Stdout, result.Stderr, result.ChoiceTrace.Bytes, result.NetworkTranscript, result.TranscriptSHA256, result.runtimeTerminal, result.Termination, result.ExitCode})
	return digest(data)
}

func TestWarmPreparationEvidenceControls(t *testing.T) {
	code := uint32(0)
	baseline := Result{Stdout: []byte("stdout"), Stderr: []byte("stderr"), NetworkTranscript: []byte("network"), TranscriptSHA256: [32]byte{1}, runtimeTerminal: [16]byte{1}, Termination: "exit", ExitCode: &code}
	baseline.ChoiceTrace.Bytes = []byte("choices")
	for _, test := range []struct {
		field  string
		mutate func(*Result)
		same   bool
	}{
		{"stdout", func(r *Result) { r.Stdout = []byte("other") }, false},
		{"stderr", func(r *Result) { r.Stderr = []byte("other") }, false},
		{"network", func(r *Result) { r.NetworkTranscript = []byte("other") }, false},
		{"wasi", func(r *Result) { r.TranscriptSHA256[0]++ }, false},
		{"terminal", func(r *Result) { r.runtimeTerminal[0]++ }, false},
		{"choices", func(r *Result) { r.ChoiceTrace.Bytes = []byte("other") }, false},
		{"termination", func(r *Result) { r.Termination = "other" }, false},
		{"exit-code", func(r *Result) { other := uint32(1); r.ExitCode = &other }, false},
		{"environment-identity", func(r *Result) { r.EnvironmentIdentity = "other" }, true},
		{"started-identity", func(r *Result) { r.Started = &Started{} }, true},
	} {
		t.Run(test.field, func(t *testing.T) {
			candidate := baseline
			test.mutate(&candidate)
			if (warmPreparationEvidence(candidate) == warmPreparationEvidence(baseline)) != test.same {
				t.Fatal("behavior/identity separation lost", test.field)
			}
		})
	}
}

func TestWarmPreparationDiagnosticCargoFeatureControls(t *testing.T) {
	data := []byte(`{"reason":"compiler-artifact","target":{"name":"gomad3-wasmhost"},"profile":{"opt_level":"3","debug_assertions":false},"executable":"/release/helper","features":["diagnostic-profile"]} {"reason":"build-finished","success":true}`)
	if _, err := terminalExitCargoHelperFeatures(data, []string{"diagnostic-profile"}); err != nil {
		t.Fatal(err)
	}
	if _, err := terminalExitCargoHelper(data); err == nil {
		t.Fatal("release-only gate weakened")
	}
	for _, features := range []string{`[]`, `["guest-profile"]`, `["diagnostic-profile","fuel-off"]`} {
		if _, err := terminalExitCargoHelperFeatures(bytes.ReplaceAll(data, []byte(`["diagnostic-profile"]`), []byte(features)), []string{"diagnostic-profile"}); err == nil {
			t.Fatal("wrong feature admitted", features)
		}
	}
}

func TestBuildWarmPreparationDiagnosticHelper(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_WARM_HELPER_BUILD_ROOT")
	if root == "" {
		t.Skip("set GOMAD_WASM_WARM_HELPER_BUILD_ROOT to build a retained diagnostic-profile helper")
	}
	terminalExitBuildHelper(t, root, []string{"diagnostic-profile"})
}

func TestWarmPreparationPrimeCache(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_WARM_PRIME_ROOT")
	if root == "" {
		t.Skip("set GOMAD_WASM_WARM_PRIME_ROOT for explicit serial cache priming, never an acceptance sample")
	}
	for _, path := range []string{root, os.Getenv("GOMAD_WASM_TEST_HELPER"), os.Getenv("GOMAD_WASM_WARM_MODULE"), os.Getenv("GOMAD_WASM_WARM_CACHE"), os.Getenv("GOMAD_WASM_WARM_HELPER_RECEIPT"), os.Getenv("GOMAD_WASM_WARM_MODULE_RECEIPT")} {
		if !filepath.IsAbs(path) {
			t.Fatal("cache priming requires the same explicit bound inputs as measurement")
		}
	}
	if _, host := diagnosticPerformanceHostEvidence("/proc/cpuinfo"); !host {
		t.Fatal("real Linux/arm64 host required")
	}
	evidence, err := diagnosticRunRoot(root, "prime-only-")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOMAD_WASM_PROTOTYPE_ROOT", evidence)
	repo, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	config := prototypeTemporalConfig(t, repo, "/workspace/tests", "-test.run=^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$", "-persistenceType=sql", "-persistenceDriver=sqlite")
	request := prototypeRequest(t, os.Getenv("GOMAD_WASM_WARM_MODULE"), config, 0, 30*time.Minute)
	request.CompileCache = os.Getenv("GOMAD_WASM_WARM_CACHE")
	crate, err := filepath.Abs("../wasmhost")
	if err != nil {
		t.Fatal(err)
	}
	inputs := terminalExitHelperInputsFeatures(t, os.Getenv("GOMAD_WASM_WARM_HELPER_RECEIPT"), request, crate, []string{"diagnostic-profile"})
	maps.Copy(inputs, terminalExitModuleInputs(t, os.Getenv("GOMAD_WASM_WARM_MODULE_RECEIPT"), request))
	result, runErr := matchedPrepareAdmitted("linux", "/proc/meminfo", "1", func() (Result, error) { return Run(t.Context(), request) })
	persistDiagnosticRawResult(t, evidence, "prime-only", result)
	temporalWrite(t, filepath.Join(evidence, "prime-only-helper-stderr.txt"), result.helperStderr)
	temporalJSON(t, filepath.Join(evidence, "prime-only.json"), map[string]any{"accepted": false, "measurement_sample": false, "purpose": "explicit compiled-cache priming outside all warm measurement windows", "inputs": inputs, "telemetry": result.Telemetry})
	if runErr != nil {
		t.Fatal(runErr)
	}
	requireExit(t, "prime-only", result, 0)
	if _, err := decodeDiagnosticReport(result.helperStderr); err != nil {
		t.Fatal(err)
	}
	temporalVerifyInputs(t, inputs)
	terminalExitHelperInputsFeatures(t, os.Getenv("GOMAD_WASM_WARM_HELPER_RECEIPT"), request, crate, []string{"diagnostic-profile"})
	terminalExitModuleInputs(t, os.Getenv("GOMAD_WASM_WARM_MODULE_RECEIPT"), request)
}

func warmPreparationSummary(series warmPreparationSeries) map[string]variantTimingRange {
	values := map[string][]uint64{}
	for _, run := range series.Runs {
		values["prepare"] = append(values["prepare"], run.Telemetry.PrepareNanos)
		values["whole_run"] = append(values["whole_run"], run.WallNanos)
		values["terminal_to_reap"] = append(values["terminal_to_reap"], run.TerminalToReapNanos)
		for _, phase := range strings.Fields("cache_read envelope_verify deserialize cache_identity") {
			values[phase] = append(values[phase], run.Phases[phase].WallNanos)
		}
		values["cache_read_plus_deserialize"] = append(values["cache_read_plus_deserialize"], run.Phases["cache_read"].WallNanos+run.Phases["deserialize"].WallNanos)
	}
	summary := map[string]variantTimingRange{}
	for name, samples := range values {
		summary[name] = variantTiming(samples)
	}
	return summary
}

func TestWarmPreparationSummaryControls(t *testing.T) {
	series := warmPreparationFixture()
	for i, value := range []uint64{7, 2, 5, 9, 4} {
		series.Runs[i].Telemetry.PrepareNanos = value
	}
	if got := warmPreparationSummary(series)["prepare"]; got != (variantTimingRange{Median: 5, Min: 2, Max: 9}) {
		t.Fatal("median/range lost", got)
	}
	baseline := warmPreparationFixture()
	baseline.Role = "baseline"
	for i := range baseline.Runs {
		baseline.Runs[i].Telemetry.PrepareNanos++
		baseline.Runs[i].TerminalToReapNanos++
	}
	if err := warmPreparationValidate(baseline, false); err != nil {
		t.Fatal("historical timings incorrectly subject to candidate bounds", err)
	}
}

func warmPreparationRead(t *testing.T, path, evidence, currentProducer string, currentRequest Request) warmPreparationSeries {
	t.Helper()
	var series warmPreparationSeries
	decoder := json.NewDecoder(bytes.NewReader(mustTerminalExitRead(t, path)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&series); err != nil {
		t.Fatal(err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		t.Fatal("trailing warm-preparation receipt", err)
	}
	temporalVerifyInputs(t, series.Inputs)
	if err := warmPreparationCollectionCheck(series, os.ReadFile); err != nil {
		t.Fatal("historical collection did not close original Run equivalence", err)
	}
	currentInputs := terminalExitModuleInputs(t, currentProducer, currentRequest)
	if err := warmPreparationProducerCheck(series, currentProducer, currentInputs); err != nil {
		t.Fatal(err)
	}
	commit := strings.TrimSpace(string(temporalCommand(t, evidence, "baseline-commit-check", series.SourceRoot, nil, []string{"git", "rev-parse", "HEAD"}).Stdout.RawBytes))
	if commit != series.SourceCommit {
		t.Fatal("historical source commit changed")
	}
	temporalCommand(t, evidence, "baseline-ancestor-check", series.SourceRoot, nil, []string{"git", "merge-base", "--is-ancestor", "15119dbbeb03f5a57fa79d57be0ab2d9ba4038c3", series.SourceCommit})
	changes := temporalCommand(t, evidence, "baseline-production-check", series.SourceRoot, nil, []string{"git", "diff", "--name-only", "15119dbbeb03f5a57fa79d57be0ab2d9ba4038c3", series.SourceCommit, "--", "."})
	for _, path := range strings.Fields(string(changes.Stdout.RawBytes)) {
		if path != "tools/gomad3/backends/wasm/wasi/warm_preparation_test.go" && path != "tools/gomad3/backends/wasm/wasi/warm_preparation_inputs_test.go" && path != "tools/gomad3/backends/wasm/wasi/warm_preparation_terminal_test.go" {
			t.Fatal("historical production or original driver changed", path)
		}
	}
	if series.Inputs[series.DriverPath] != series.DriverSHA256 || !filepath.IsAbs(series.SourceRoot) || !strings.HasPrefix(series.DriverPath, series.SourceRoot+string(filepath.Separator)) || series.Inputs[series.HelperReceipt] == "" || series.Inputs[series.ModuleReceipt] == "" || series.Inputs[series.ArtifactPath] == "" {
		t.Fatal("historical receipt lacks actual driver/build/module/cache bindings")
	}
	var admission terminalExitHelperAdmission
	if err := json.Unmarshal(mustTerminalExitRead(t, series.HelperReceipt), &admission); err != nil {
		t.Fatal(err)
	}
	var build terminalExitHelperBuild
	if err := json.Unmarshal(mustTerminalExitRead(t, admission.BuildReceipt), &build); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(build.Crate, series.SourceRoot+string(filepath.Separator)) || admission.HelperSHA256 != series.HelperSHA256 {
		t.Fatal("historical helper does not belong to its retained source/build")
	}
	terminalExitHelperInputsFeatures(t, series.HelperReceipt, Request{HelperPath: admission.Helper, HelperSHA256: admission.HelperSHA256}, build.Crate, []string{"diagnostic-profile"})
	for _, run := range series.Runs {
		for _, rawPath := range []string{run.ObservationPath, run.FinishPath, run.DiagnosticPath} {
			if series.Inputs[rawPath] == "" {
				t.Fatal("historical run lacks bound raw observation/finish/diagnostic evidence")
			}
		}
		var observation warmPreparationObservation
		if err := json.Unmarshal(mustTerminalExitRead(t, run.ObservationPath), &observation); err != nil {
			t.Fatal(err)
		}
		finish := mustTerminalExitRead(t, run.FinishPath)
		if len(finish) != len(observation.Result.runtimeTerminal) {
			t.Fatal("missing historical private terminal sidecar")
		}
		copy(observation.Result.runtimeTerminal[:], finish)
		profile, err := decodeDiagnosticReport(mustTerminalExitRead(t, run.DiagnosticPath))
		if err != nil || !reflect.DeepEqual(profile.Phases, run.Phases) || observation.Result.Telemetry == nil || *observation.Result.Telemetry != run.Telemetry || warmPreparationEvidence(observation.Result) != run.EvidenceSHA256 || observation.TerminalToReapNanos != run.TerminalToReapNanos || observation.Error != "" || !observation.HostExited || observation.HostExitCode != 0 || !observation.GroupGone || observation.Cancelled || observation.WatchdogTimeout || observation.TerminalReceivedAt.IsZero() || !observation.SupervisorReturnedAt.After(observation.TerminalReceivedAt) || uint64(observation.SupervisorReturnedAt.Sub(observation.TerminalReceivedAt).Nanoseconds()) != run.TerminalToReapNanos || !observation.Result.Reaped || observation.Result.Termination != "exit" || observation.Result.ExitCode == nil || *observation.Result.ExitCode != 0 || observation.Result.Started == nil || observation.Result.Started.Engine != series.Engine || observation.Result.Started.ModuleSHA256 != series.ModuleSHA256 {
			t.Fatal("historical raw observation does not support the declared run", err)
		}
	}
	configBytes, err := json.Marshal(currentRequest.Config)
	if err != nil || series.ConfigSHA256 != digest(configBytes) {
		t.Fatal("historical comparison requires the bound modeled configuration", err)
	}
	var collection warmPreparationCollection
	if err := json.Unmarshal(mustTerminalExitRead(t, series.CollectionPath), &collection); err != nil {
		t.Fatal(err)
	}
	var retained Result
	if err := json.Unmarshal(mustTerminalExitRead(t, collection.ResultPath), &retained); err != nil {
		t.Fatal(err)
	}
	copy(retained.runtimeTerminal[:], mustTerminalExitRead(t, collection.FinishPath))
	series.ComparisonSHA256, err = warmPreparationBehavior(retained, currentRequest.Config, true)
	if err != nil {
		t.Fatal("source-bound historical transcript comparison", err)
	}
	return series
}

func warmPreparationArtifact(t *testing.T, request Request) (string, uint64, uint64) {
	t.Helper()
	entries, err := filepath.Glob(filepath.Join(request.CompileCache, "*.gomad-cache-v3"))
	if err != nil {
		t.Fatal(err)
	}
	selected := ""
	var artifactBytes, envelopeBytes uint64
	for _, entry := range entries {
		file, info, err := hostfs.OpenPath(entry)
		if err != nil {
			t.Fatal(err)
		}
		var header [24]byte
		_, err = io.ReadFull(file, header[:])
		manifestBytes := binary.LittleEndian.Uint32(header[8:12])
		artifact := binary.LittleEndian.Uint64(header[12:20])
		if err != nil || string(header[:8]) != "GOMADC03" || manifestBytes > 16<<10 || artifact == 0 || artifact > 1<<30 || uint64(info.Size()) != 24+uint64(manifestBytes)+artifact {
			_ = file.Close()
			t.Fatal("invalid current cache envelope", entry, err)
		}
		manifest := make([]byte, manifestBytes)
		_, err = io.ReadFull(file, manifest)
		closeErr := file.Close()
		if err != nil || closeErr != nil {
			t.Fatal("read cache manifest", err, closeErr)
		}
		var identity struct {
			Version       uint64 `json:"version"`
			ModuleSHA256  string `json:"module_sha256"`
			HelperSHA256  string `json:"helper_sha256"`
			Configuration string `json:"configuration"`
			HostOS        string `json:"host_os"`
			HostArch      string `json:"host_arch"`
		}
		if err := json.Unmarshal(manifest, &identity); err != nil {
			t.Fatal(err)
		}
		if identity.ModuleSHA256 != request.ModuleSHA256 || identity.HelperSHA256 != request.HelperSHA256 {
			continue
		}
		if selected != "" || identity.Version != 3 || identity.Configuration != request.Engine.Configuration || identity.HostOS != request.Engine.HostOS || identity.HostArch != request.Engine.HostArch || filepath.Base(entry) != digest(manifest)+".gomad-cache-v3" {
			t.Fatal("ambiguous or incompatible measured cache identity")
		}
		selected, artifactBytes, envelopeBytes = entry, artifact, uint64(info.Size())
	}
	if selected == "" {
		t.Fatal("prewarmed matching cache artifact required; this driver never cold-compiles intentionally")
	}
	return selected, artifactBytes, envelopeBytes
}

func TestWarmPreparationArtifactSizeControls(t *testing.T) {
	request := Request{CompileCache: t.TempDir(), ModuleSHA256: strings.Repeat("a", 64), HelperSHA256: strings.Repeat("b", 64), Engine: PinnedEngine()}
	var expected string
	var expectedEnvelope uint64
	for _, module := range []string{request.ModuleSHA256, strings.Repeat("c", 64)} {
		manifest, err := json.Marshal(map[string]any{"version": 3, "module_sha256": module, "helper_sha256": request.HelperSHA256, "configuration": request.Engine.Configuration, "host_os": request.Engine.HostOS, "host_arch": request.Engine.HostArch})
		if err != nil {
			t.Fatal(err)
		}
		header := make([]byte, 24)
		copy(header, "GOMADC03")
		binary.LittleEndian.PutUint32(header[8:12], uint32(len(manifest)))
		binary.LittleEndian.PutUint64(header[12:20], 100)
		entry := filepath.Join(request.CompileCache, digest(manifest)+".gomad-cache-v3")
		data := append(append(header, manifest...), make([]byte, 100)...)
		if err := os.WriteFile(entry, data, 0600); err != nil {
			t.Fatal(err)
		}
		if module == request.ModuleSHA256 {
			expected, expectedEnvelope = entry, uint64(len(data))
		}
	}
	entry, artifact, envelope := warmPreparationArtifact(t, request)
	if entry != expected || artifact != 100 || envelope != expectedEnvelope {
		t.Fatal("artifact/envelope bytes or matching module identity lost", entry, artifact, envelope)
	}
}

func TestTemporalWarmPreparationAcceptance(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_WARM_PREPARATION_ROOT")
	if root == "" {
		t.Skip("set GOMAD_WASM_WARM_PREPARATION_ROOT for serial real-host five-run acceptance collection")
	}
	paths := map[string]string{"root": root, "helper": os.Getenv("GOMAD_WASM_TEST_HELPER"), "module": os.Getenv("GOMAD_WASM_WARM_MODULE"), "cache": os.Getenv("GOMAD_WASM_WARM_CACHE"), "helper_receipt": os.Getenv("GOMAD_WASM_WARM_HELPER_RECEIPT"), "module_receipt": os.Getenv("GOMAD_WASM_WARM_MODULE_RECEIPT")}
	for name, path := range paths {
		if !filepath.IsAbs(path) {
			t.Fatal("explicit absolute warm-preparation input required", name)
		}
	}
	cpuinfo, realHost := diagnosticPerformanceHostEvidence("/proc/cpuinfo")
	if !realHost {
		t.Fatal("real Linux/arm64 performance host required")
	}
	boot, err := hostfs.ReadBounded("/proc/sys/kernel/random/boot_id", 128)
	if err != nil || strings.TrimSpace(string(boot)) == "" {
		t.Fatal("host boot identity required", err)
	}
	evidence, err := diagnosticRunRoot(root, "warm-preparation-")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOMAD_WASM_PROTOTYPE_ROOT", evidence)
	repo, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	config := prototypeTemporalConfig(t, repo, "/workspace/tests", "-test.run=^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$", "-persistenceType=sql", "-persistenceDriver=sqlite")
	request := prototypeRequest(t, paths["module"], config, 0, 30*time.Minute)
	request.CompileCache = paths["cache"]
	// Digest-only matches the unchanged timing boundary; raw evidence is retained
	// once in an untimed final run and checked against every measured run.
	request.DigestTranscript = true
	crate, err := filepath.Abs("../wasmhost")
	if err != nil {
		t.Fatal(err)
	}
	inputs := terminalExitHelperInputsFeatures(t, paths["helper_receipt"], request, crate, []string{"diagnostic-profile"})
	maps.Copy(inputs, terminalExitModuleInputs(t, paths["module_receipt"], request))
	commit := strings.TrimSpace(string(temporalCommand(t, evidence, "source-commit", repo, nil, []string{"git", "rev-parse", "HEAD"}).Stdout.RawBytes))
	status := temporalCommand(t, evidence, "source-status", repo, nil, []string{"git", "status", "--porcelain", "--", "tools/gomad"})
	if len(status.Stdout.RawBytes) != 0 {
		t.Fatal("freeze and commit the measured tool source first")
	}
	tracked := temporalCommand(t, evidence, "source-inventory", repo, nil, []string{"git", "ls-files", "-z", "tools/gomad"})
	for _, name := range bytes.Split(tracked.Stdout.RawBytes, []byte{0}) {
		if len(name) != 0 {
			path := filepath.Join(repo, string(name))
			inputs[path] = fileHash(t, path)
		}
	}
	driver := filepath.Join(repo, "tools/gomad/internal/backends/wasm/wasi/warm_preparation_test.go")
	artifact, artifactBytes, envelopeBytes := warmPreparationArtifact(t, request)
	inputs[artifact] = fileHash(t, artifact) // outside all timing windows; warms its pages deliberately.
	configBytes, err := json.Marshal(request.Config)
	if err != nil {
		t.Fatal(err)
	}
	series := warmPreparationSeries{Schema: warmPreparationSchema, Role: "candidate", SourceCommit: commit, SourceRoot: repo, DriverSHA256: fileHash(t, driver), HostBootID: strings.TrimSpace(string(boot)), HostCPUInfo: cpuinfo, ConfigSHA256: digest(configBytes), Engine: request.Engine, ModuleSHA256: request.ModuleSHA256, HelperSHA256: request.HelperSHA256, RuntimeIdentity: request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey, ModelSHA256: ImplementationSHA256(), Seed: request.Runtime.Seed, ArtifactBytes: artifactBytes, EnvelopeBytes: envelopeBytes, Inputs: inputs, HelperReceipt: paths["helper_receipt"], ModuleReceipt: paths["module_receipt"], DriverPath: driver, ArtifactPath: artifact}
	failed := ""
	var reference Result
	for i := range 6 {
		var observation terminalExitObservation
		started := time.Now()
		_, admissionErr := matchedPrepareAdmitted("linux", "/proc/meminfo", "1", func() (Result, error) {
			observation = measureTerminalExitContext(t, t.Context(), request, false)
			return observation.Result, nil
		})
		wall := uint64(time.Since(started).Nanoseconds())
		if admissionErr != nil {
			t.Fatal(admissionErr)
		}
		name := fmt.Sprintf("run-%d", i)
		temporalJSON(t, filepath.Join(evidence, name+"-observation.json"), observation)
		temporalWrite(t, filepath.Join(evidence, name+"-helper-stderr.txt"), observation.Host.Stderr.RawBytes)
		if observation.Error != "" || observation.Result.Telemetry == nil {
			t.Fatal("helper observation/telemetry failed", observation.Error)
		}
		if !observation.Result.Telemetry.CacheHit || observation.Result.Telemetry.CompileNanos != 0 {
			t.Fatal("every run, including the unmeasured warm-up, must hit the existing cache without compilation")
		}
		profile, err := decodeDiagnosticReport(observation.Host.Stderr.RawBytes)
		if err != nil || profile.HostOS != "linux" || profile.HostArch != "aarch64" {
			t.Fatal("diagnostic phase/host evidence required", err)
		}
		if i == 0 {
			reference = observation.Result
		} else if difference := sameEvidence(reference, observation.Result); difference != "" {
			failed = "within-series evidence changed: " + difference
		}
		{
			finishPath := filepath.Join(evidence, name+"-finish.bin")
			temporalWrite(t, finishPath, observation.Result.runtimeTerminal[:])
			observationPath, diagnosticPath := filepath.Join(evidence, name+"-warm-observation.json"), filepath.Join(evidence, name+"-helper-stderr.txt")
			temporalJSON(t, observationPath, warmPreparationObserve(observation))
			rawObservationPath := filepath.Join(evidence, name+"-observation.json")
			for _, path := range []string{observationPath, rawObservationPath, diagnosticPath, finishPath} {
				inputs[path] = fileHash(t, path)
			}
			run := warmPreparationRun{Telemetry: *observation.Result.Telemetry, Phases: profile.Phases, WallNanos: wall, TerminalToReapNanos: observation.TerminalToReapNanos, EvidenceSHA256: warmPreparationEvidence(observation.Result), Exited: observation.within(time.Second) && observation.Result.Termination == "exit" && observation.Result.ExitCode != nil && *observation.Result.ExitCode == 0, Reaped: observation.Result.Reaped, ObservationPath: observationPath, RawObservationPath: rawObservationPath, FinishPath: finishPath, DiagnosticPath: diagnosticPath}
			if i == 0 {
				series.Warmup = &run
			} else {
				series.Runs = append(series.Runs, run)
			}
		}
		if !strings.Contains(string(observation.Result.Stdout), "--- PASS: TestWorkflowTypeEncodingSuite") || !strings.Contains(string(observation.Result.Stdout), "TestPlainASCII/Succeeds") {
			failed = "unchanged selected workflow did not pass"
		}
	}
	request.DigestTranscript = false
	var retained terminalExitObservation
	_, retainedAdmissionErr := matchedPrepareAdmitted("linux", "/proc/meminfo", "1", func() (Result, error) {
		retained = measureTerminalExitContext(t, t.Context(), request, false)
		return retained.Result, nil
	})
	if retainedAdmissionErr != nil {
		t.Fatal("untimed retention admission refused before execution", retainedAdmissionErr)
	}
	persistDiagnosticRawResult(t, evidence, "untimed-evidence", retained.Result)
	temporalJSON(t, filepath.Join(evidence, "untimed-evidence-observation.json"), retained)
	if retained.Error != "" || sameEvidence(reference, retained.Result) != "" {
		failed = "untimed canonical retention changed behavior"
	}
	temporalVerifyInputs(t, inputs)
	terminalExitHelperInputsFeatures(t, paths["helper_receipt"], request, crate, []string{"diagnostic-profile"})
	terminalExitModuleInputs(t, paths["module_receipt"], request)
	afterCommit := strings.TrimSpace(string(temporalCommand(t, evidence, "source-commit-after", repo, nil, []string{"git", "rev-parse", "HEAD"}).Stdout.RawBytes))
	if afterCommit != commit {
		failed = "source commit changed during collection"
	}
	afterBoot, err := hostfs.ReadBounded("/proc/sys/kernel/random/boot_id", 128)
	afterCPU, afterHost := diagnosticPerformanceHostEvidence("/proc/cpuinfo")
	if err != nil || !bytes.Equal(boot, afterBoot) || !afterHost || cpuinfo != afterCPU {
		failed = "host bindings changed during collection"
	}
	if err := warmPreparationValidate(series, true); err != nil {
		failed = err.Error()
	}
	series.ComparisonSHA256, err = warmPreparationBehavior(retained.Result, request.Config, false)
	if err != nil {
		failed = "candidate transcript comparison: " + err.Error()
	}
	failed = warmPreparationPublishCollection(t, evidence, "candidate-collection.json", &series, retained.Result, retained.Error, failed)
	baselinePath := os.Getenv("GOMAD_WASM_WARM_BASELINE_RECEIPT")
	var baseline *warmPreparationSeries
	var baselineSummary map[string]variantTimingRange
	baselineComparison := ""
	if baselinePath == "" {
		failed = "baseline-material-gap: collect the historical source with its original matching engine/driver; this candidate never overrides engine identity"
	} else {
		if !filepath.IsAbs(baselinePath) {
			t.Fatal("baseline receipt must be absolute")
		}
		before := warmPreparationRead(t, baselinePath, evidence, paths["module_receipt"], request)
		baseline = &before
		baselineSummary = warmPreparationSummary(before)
		baselineComparison = before.ComparisonSHA256
		if err := warmPreparationCompare(before, series); err != nil {
			failed = err.Error()
		}
		inputs[baselinePath] = fileHash(t, baselinePath)
	}
	summary := warmPreparationSummary(series)
	temporalJSON(t, filepath.Join(evidence, "acceptance.json"), map[string]any{"schema": "gomad3.wasm-warm-preparation-acceptance/v1", "accepted": failed == "", "failure": failed, "candidate": series, "baseline": baseline, "candidate_summary_nanos": summary, "baseline_summary_nanos": baselineSummary, "candidate_comparison_sha256": series.ComparisonSHA256, "baseline_comparison_sha256": baselineComparison, "comparison_policy": "source-bound historical paired JSONL validated through the typed GMWT owner; original raw digests remain separate; no format fallback or modeled-input masking", "artifact_bytes": artifactBytes, "envelope_bytes": envelopeBytes, "parked_mapping_owner_decision_required": summary["cache_read_plus_deserialize"].Median > uint64(time.Second), "boundary": "one unmeasured warm-up, five fresh helpers with same workflow and seed on prewarmed page cache; no artifact hashing in timed windows; full source/build/module/cache bindings verified before and after; terminal receipt through supervisor wait/capture/group reaping", "baseline_policy": "historical matching driver and PinnedEngine only; helper/engine/model/source identities separate, modeled inputs and exact behavior must remain equal"})
	if failed != "" {
		t.Fatalf("warm-preparation acceptance not met: %s; retained %s", failed, evidence)
	}
}
