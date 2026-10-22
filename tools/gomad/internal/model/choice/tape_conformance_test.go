package choice

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type replayPlanCorpus struct {
	Schema        string            `json:"schema"`
	Name          string            `json:"name"`
	Source        string            `json:"source"`
	Identity      ExecutionIdentity `json:"identity"`
	Trace         []byte            `json:"trace"`
	Interrupted   bool              `json:"interrupted"`
	PrefixControl *replayPlanPrefix `json:"prefix_control,omitempty"`
	Expected      replayPlanResult  `json:"expected"`
}

type replayPlanPrefix struct {
	Cutoff   uint64              `json:"cutoff"`
	Override *replayPlanOverride `json:"override,omitempty"`
}

type replayPlanOverride struct {
	Ordinal     uint64 `json:"ordinal"`
	Rank        uint32 `json:"rank"`
	Reconstruct bool   `json:"reconstruct"`
}

type replayPlanResult struct {
	Error     string     `json:"error"`
	Bytes     []byte     `json:"bytes"`
	SHA256    string     `json:"sha256"`
	Decisions []Decision `json:"decisions"`
}

func replayPlanCorpusDirectory() string {
	return filepath.FromSlash("../../../rust/wasmhost/tests/testdata/replay_plan")
}

func projectReplayPlanCorpus(corpus replayPlanCorpus) replayPlanResult {
	var plan ReplayPlan
	var err error
	if corpus.Interrupted {
		plan, err = ProjectInterruptedReplayPlan(corpus.Trace, 8192, corpus.Identity)
	} else {
		var trace Trace
		trace, err = DecodeStoredTrace(Profile, corpus.Trace, TerminalMetadata{
			State: TerminalComplete, Limit: 8192, Records: uint64(len(corpus.Trace) / traceRecordBytes), SHA256: sha256.Sum256(corpus.Trace),
		})
		if err == nil {
			plan, err = ProjectReplayPlan(trace, corpus.Identity)
		}
	}
	if err == nil && corpus.PrefixControl != nil {
		control := corpus.PrefixControl
		if override := control.Override; override != nil {
			if control.Cutoff == 0 || override.Ordinal != control.Cutoff-1 {
				err = ErrInvalidReplayPlan
			} else if override.Reconstruct {
				plan, err = BuildForcedRankPrefix(plan, override.Ordinal, override.Rank)
			} else {
				plan, err = BuildRankPrefix(plan, override.Ordinal, override.Rank)
			}
		} else {
			plan, err = plan.Prefix(control.Cutoff)
		}
	}
	if err != nil {
		classification := "malformed_trace"
		if errors.Is(err, ErrInvalidReplayPlan) {
			classification = "invalid_replay_plan"
		}
		return replayPlanResult{Error: classification}
	}
	return replayPlanResult{Bytes: plan.Bytes, SHA256: fmt.Sprintf("%x", plan.SHA256), Decisions: plan.Decisions}
}

//parallelize:ignore
func TestReplayPlanConformance(t *testing.T) {
	directory := replayPlanCorpusDirectory()
	if override := os.Getenv("GOMAD_REPLAY_PLAN_CORPUS_DIR"); override != "" {
		directory = override
	}
	regenerate := os.Getenv("GOMAD_REPLAY_PLAN_CORPUS_REGENERATE") == "1"
	fixtures := replayPlanFixtures(t)
	if regenerate {
		if err := os.MkdirAll(directory, 0755); err != nil {
			t.Fatal(err)
		}
	} else {
		files, err := filepath.Glob(filepath.Join(directory, "*.json"))
		if err != nil || len(files) != len(fixtures) {
			t.Fatalf("replay plan corpus: %d files, want %d: %v", len(files), len(fixtures), err)
		}
	}
	for _, fixture := range fixtures {
		t.Run(fixture.Name, func(t *testing.T) {
			got := projectReplayPlanCorpus(fixture)
			if got.Error != fixture.Expected.Error {
				t.Fatalf("projection classification = %q, want %q", got.Error, fixture.Expected.Error)
			}
			fixture.Expected = got
			encoded, err := json.MarshalIndent(fixture, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			encoded = append(encoded, '\n')
			path := filepath.Join(directory, fixture.Name+".json")
			if regenerate {
				if err := os.WriteFile(path, encoded, 0644); err != nil {
					t.Fatal(err)
				}
				return
			}
			retained, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var corpus replayPlanCorpus
			if err := json.Unmarshal(retained, &corpus); err != nil {
				t.Fatal(err)
			}
			actual, err := json.Marshal(projectReplayPlanCorpus(corpus))
			if err != nil {
				t.Fatal(err)
			}
			want, err := json.Marshal(corpus.Expected)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(actual, want) || !bytes.Equal(retained, encoded) {
				t.Fatal("replay plan corpus differs from Go projection; review then regenerate with GOMAD_REPLAY_PLAN_CORPUS_REGENERATE=1")
			}
		})
	}
}

func replayPlanFixtures(t *testing.T) []replayPlanCorpus {
	t.Helper()
	var fixtures []replayPlanCorpus
	add := func(name, source string, records []Record, classification string) {
		t.Helper()
		trace, err := BuildTrace(records, TerminalComplete)
		if err != nil {
			t.Fatal(err)
		}
		fixtures = append(fixtures, replayPlanCorpus{Schema: "gomad.replay-plan-conformance/v1", Name: name, Source: source,
			Identity: readinessTestIdentity(), Trace: append([]byte{}, trace.Bytes...), Expected: replayPlanResult{Error: classification}})
	}
	ready := SelectReadiness{Known: true, Ready: 2}
	poll := func(ordinal uint64, site uint64, alternatives int) Record {
		return selectPollDecision(t, ordinal, site, alternatives).Record()
	}
	result := func(ordinal, origin, site uint64, cases, polled uint32, readiness SelectReadiness) Record {
		return selectResult(t, ordinal, origin, site, cases, polled, readiness)
	}
	add("empty", "trace_test.go", nil, "")
	add("observation-only", "readiness_test.go", []Record{result(0, 0, 40, 1, 1, SelectReadiness{Known: true, Ready: 1})}, "")
	single := runnableDecision(t, 0).Record()
	single.Alternatives, single.Selected = 1, 0
	add("single-alternative", "tape_test.go", []Record{single}, "")
	add("runnable", "readiness_test.go", []Record{runnableDecision(t, 0).Record()}, "")
	add("select-readiness-origin", "readiness_test.go", []Record{
		runnableDecision(t, 0).Record(), poll(1, 40, 2),
		result(2, 1, 40, 3, 2, SelectReadiness{Known: true, Ready: 1, ClosedChannel: true, NilChannel: true}),
		poll(3, 50, 2), poll(4, 50, 3), poll(5, 50, 4), runnableDecision(t, 6).Record(), poll(7, 60, 2),
		result(8, 3, 50, 4, 4, SelectReadiness{Known: true, Default: true, TimerChannel: true}),
	}, "")
	add("single-case-unmatched", "readiness_test.go", []Record{poll(0, 40, 2), result(1, 0, 50, 1, 1, SelectReadiness{Known: true, Ready: 1})}, "")
	add("zero-polled-default", "readiness_test.go", []Record{result(0, 0, 40, 1, 0, SelectReadiness{Known: true, Default: true, NilChannel: true})}, "")
	add("unfinished-select-prefix", "readiness_test.go", []Record{poll(0, 40, 2), poll(1, 40, 3)}, "")
	add("repeated-channel", "readiness_test.go", []Record{poll(0, 40, 2), result(1, 0, 40, 2, 2, SelectReadiness{Known: true, Ready: 2, RepeatedChannel: true})}, "")
	missingPoll := poll(0, 0, 2)
	missingPoll.Flags |= FlagSiteMissing
	missingResult := result(1, 0, 0, 2, 2, ready)
	missingResult.Flags |= FlagSiteMissing
	add("missing-site", "readiness_test.go", []Record{missingPoll, missingResult}, "")
	for _, test := range []struct {
		name    string
		records []Record
	}{
		{"origin-runnable", []Record{runnableDecision(t, 0).Record(), poll(1, 40, 2), result(2, 0, 40, 2, 2, ready)}},
		{"origin-site", []Record{poll(0, 41, 2), result(1, 0, 40, 2, 2, ready)}},
		{"poll-order", []Record{poll(0, 40, 3), poll(1, 40, 2), result(2, 0, 40, 3, 3, ready)}},
		{"before-origin", []Record{poll(0, 40, 2), result(1, 0, 40, 3, 3, ready)}},
		{"double-named", []Record{poll(0, 40, 2), result(1, 0, 40, 2, 2, ready), result(2, 0, 40, 2, 2, ready)}},
		{"missing-site-mismatch", []Record{poll(0, 0, 2), missingResult}},
	} {
		add(test.name, "readiness_test.go", test.records, "invalid_replay_plan")
	}
	storagePath := filepath.Join(replayPlanCorpusDirectory(), "..", "helper_transcript", "typed_interrupt_storage_crash_cooperative.json")
	storageBytes, err := os.ReadFile(storagePath)
	if err != nil {
		t.Fatal(err)
	}
	var storage struct {
		Expected struct {
			Termination string `json:"termination"`
			Frames      []struct {
				Stream string `json:"stream"`
				Data   []byte `json:"data_base64"`
			} `json:"frames"`
		} `json:"expected"`
	}
	if err := json.Unmarshal(storageBytes, &storage); err != nil {
		t.Fatal(err)
	}
	var interrupted []byte
	for _, frame := range storage.Expected.Frames {
		if frame.Stream == "choice" {
			interrupted = append(interrupted, frame.Data...)
		}
	}
	if storage.Expected.Termination != "storage_crash" || len(interrupted) == 0 {
		t.Fatal("storage crash fixture has no interrupted choice evidence")
	}
	fixtures = append(fixtures, replayPlanCorpus{Schema: "gomad.replay-plan-conformance/v1", Name: "storage-crash-interrupted", Source: filepath.Base(storagePath),
		Identity: readinessTestIdentity(), Trace: interrupted, Interrupted: true})
	for _, mutation := range []struct {
		name   string
		mutate func([]byte) []byte
	}{
		{"truncated", func(b []byte) []byte { return b[:len(b)-1] }},
		{"ordinal", func(b []byte) []byte { binary.BigEndian.PutUint64(b, 1); return b }},
		{"kind", func(b []byte) []byte { b[8] = 4; return b }},
		{"flags", func(b []byte) []byte { b[9] = 3; return b }},
		{"unknown-flags", func(b []byte) []byte { b[9] |= 16; return b }},
		{"selection", func(b []byte) []byte { binary.BigEndian.PutUint32(b[16:], 2); return b }},
		{"no-alternatives", func(b []byte) []byte { binary.BigEndian.PutUint32(b[12:], 0); return b }},
		{"missing-site-offset", func(b []byte) []byte { binary.BigEndian.PutUint64(b[24:], 1); return b }},
		{"selected-identity", func(b []byte) []byte { clear(b[32:64]); return b }},
		{"alternative-set", func(b []byte) []byte { clear(b[64:96]); return b }},
		{"trace-readiness", func(b []byte) []byte { b[11] = 1; return b }},
		{"rank-override", func(b []byte) []byte { b[9] |= byte(FlagRankOverride); clear(b[32:64]); return b }},
	} {
		fixtures = append(fixtures, replayPlanCorpus{Schema: "gomad.replay-plan-conformance/v1", Name: "malformed-" + mutation.name, Source: "trace_test.go",
			Identity: readinessTestIdentity(), Trace: mutation.mutate(slices.Clone(fixtures[3].Trace)), Expected: replayPlanResult{Error: "malformed_trace"}})
	}
	for _, mutation := range []struct {
		name   string
		mutate func([]byte)
	}{
		{"result-origin", func(b []byte) { binary.BigEndian.PutUint64(b[32:], 1) }},
		{"result-reserved", func(b []byte) { b[40] = 1 }},
		{"result-identities", func(b []byte) { b[64] = 1 }},
		{"result-unknown-readiness", func(b []byte) { binary.BigEndian.PutUint16(b[10:], 0) }},
		{"result-unflagged-readiness", func(b []byte) { binary.BigEndian.PutUint16(b[10:], 64) }},
		{"result-ready-count", func(b []byte) { binary.BigEndian.PutUint16(b[10:], 129) }},
		{"result-kind-flags", func(b []byte) { b[9] = byte(FlagDecision) }},
	} {
		payload := slices.Clone(fixtures[1].Trace)
		mutation.mutate(payload)
		fixtures = append(fixtures, replayPlanCorpus{Schema: "gomad.replay-plan-conformance/v1", Name: "malformed-" + mutation.name, Source: "trace_test.go",
			Identity: readinessTestIdentity(), Trace: payload, Expected: replayPlanResult{Error: "malformed_trace"}})
	}
	for _, mutation := range []struct {
		name   string
		mutate func(*ExecutionIdentity)
	}{
		{"target", func(identity *ExecutionIdentity) { identity.TargetSHA256 = [32]byte{} }},
		{"implementation", func(identity *ExecutionIdentity) { identity.ImplementationSHA256 = [32]byte{} }},
		{"goos", func(identity *ExecutionIdentity) { identity.GOOS = "" }},
		{"goarch", func(identity *ExecutionIdentity) { identity.GOARCH = "" }},
		{"toolchain-short", func(identity *ExecutionIdentity) { identity.ToolchainBuildKey = "aa" }},
		{"toolchain-uppercase", func(identity *ExecutionIdentity) { identity.ToolchainBuildKey = strings.Repeat("A", 64) }},
		{"toolchain-nonhex", func(identity *ExecutionIdentity) { identity.ToolchainBuildKey = strings.Repeat("g", 64) }},
		{"toolchain-zero", func(identity *ExecutionIdentity) { identity.ToolchainBuildKey = strings.Repeat("0", 64) }},
	} {
		identity := readinessTestIdentity()
		mutation.mutate(&identity)
		fixtures = append(fixtures, replayPlanCorpus{Schema: "gomad.replay-plan-conformance/v1", Name: "identity-" + mutation.name, Source: "tape_test.go",
			Identity: identity, Trace: slices.Clone(fixtures[3].Trace), Expected: replayPlanResult{Error: "invalid_replay_plan"}})
	}
	base := fixtures[4]
	plan, err := ProjectReplayPlan(mustReplayPlanTrace(t, base.Trace), base.Identity)
	if err != nil {
		t.Fatal(err)
	}
	observed := plan.Decisions[3].Selected
	for _, test := range []struct {
		name           string
		control        replayPlanPrefix
		classification string
	}{
		{"zero", replayPlanPrefix{Cutoff: 0}, ""},
		{"short", replayPlanPrefix{Cutoff: 2}, ""},
		{"full", replayPlanPrefix{Cutoff: uint64(len(plan.Decisions))}, ""},
		{"too-long", replayPlanPrefix{Cutoff: uint64(len(plan.Decisions)) + 1}, "invalid_replay_plan"},
		{"rank", replayPlanPrefix{Cutoff: 4, Override: &replayPlanOverride{Ordinal: 3, Rank: (observed + 1) % 3}}, ""},
		{"forced", replayPlanPrefix{Cutoff: 4, Override: &replayPlanOverride{Ordinal: 3, Rank: observed, Reconstruct: true}}, ""},
		{"rank-observed", replayPlanPrefix{Cutoff: 4, Override: &replayPlanOverride{Ordinal: 3, Rank: observed}}, "invalid_replay_plan"},
		{"forced-unobserved", replayPlanPrefix{Cutoff: 4, Override: &replayPlanOverride{Ordinal: 3, Rank: (observed + 1) % 3, Reconstruct: true}}, "invalid_replay_plan"},
		{"rank-outside", replayPlanPrefix{Cutoff: 4, Override: &replayPlanOverride{Ordinal: 3, Rank: 3}}, "invalid_replay_plan"},
		{"rank-not-final", replayPlanPrefix{Cutoff: 4, Override: &replayPlanOverride{Ordinal: 2, Rank: 0}}, "invalid_replay_plan"},
		{"rank-empty", replayPlanPrefix{Cutoff: 0, Override: &replayPlanOverride{}}, "invalid_replay_plan"},
		{"rank-beyond", replayPlanPrefix{Cutoff: 100, Override: &replayPlanOverride{Ordinal: 99}}, "invalid_replay_plan"},
	} {
		fixture := base
		fixture.Name, fixture.Source = "prefix-"+test.name, "readiness_test.go/tape_test.go"
		fixture.PrefixControl, fixture.Expected = &test.control, replayPlanResult{Error: test.classification}
		fixtures = append(fixtures, fixture)
	}
	for _, source := range []struct{ name, fixture string }{
		{"helper-runtime", "model_runtime_record"},
	} {
		payload := retainedReplayFixture(t, source.fixture)
		fixtures = append(fixtures, replayPlanCorpus{Schema: "gomad.replay-plan-conformance/v1", Name: source.name, Source: source.fixture + ".json", Identity: readinessTestIdentity(), Trace: payload})
		if source.name == "helper-runtime" {
			fixtures = append(fixtures, replayPlanCorpus{Schema: "gomad.replay-plan-conformance/v1", Name: "helper-runtime-rank", Source: source.fixture + ".json", Identity: readinessTestIdentity(), Trace: payload, PrefixControl: &replayPlanPrefix{Cutoff: 1, Override: &replayPlanOverride{Ordinal: 0, Rank: 1}}})
		}
	}
	var first, second [32]byte
	for index := range first {
		first[index], second[index] = 1, 2
	}
	kindDecision, err := CanonicalDecision(0, KindSelectPoll, 42, false, [][32]byte{first, second}, first, 0)
	if err != nil {
		t.Fatal(err)
	}
	add("helper-runtime-kind", "model_runtime_kind_divergence.json", []Record{kindDecision.Record()}, "")
	fixtures[len(fixtures)-1].PrefixControl = &replayPlanPrefix{Cutoff: 1, Override: &replayPlanOverride{Ordinal: 0, Rank: 1}}
	creationIDs := [][32]byte{{1}, {2}}
	creationDecision, err := CanonicalDecision(2, KindRunnable, 0, true, creationIDs, creationIDs[1], 0)
	if err != nil {
		t.Fatal(err)
	}
	add("runnable-creation-sites", "creation_test.go", []Record{
		{Ordinal: 0, Kind: KindGoroutineCreation, Alternatives: 2, SiteOffset: 123, SelectedIdentity: creationIDs[0]},
		{Ordinal: 1, Kind: KindGoroutineCreation, Flags: FlagSiteMissing, Alternatives: 2, Selected: 1, SelectedIdentity: creationIDs[1]},
		creationDecision.Record(),
	}, "")
	creationTrace := slices.Clone(fixtures[len(fixtures)-1].Trace)
	for _, mutation := range []struct {
		name   string
		offset int
		value  uint32
	}{
		{"count", 96 + 12, 3},
		{"rank", 96 + 16, 0},
	} {
		payload := slices.Clone(creationTrace)
		binary.BigEndian.PutUint32(payload[mutation.offset:mutation.offset+4], mutation.value)
		fixtures = append(fixtures, replayPlanCorpus{Schema: "gomad.replay-plan-conformance/v1", Name: "malformed-creation-" + mutation.name, Source: "creation_test.go", Identity: readinessTestIdentity(), Trace: payload, Expected: replayPlanResult{Error: "malformed_trace"}})
	}
	return fixtures
}

func retainedReplayFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(replayPlanCorpusDirectory(), "..", "helper_transcript", name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Expected struct {
			Frames []struct {
				Stream string `json:"stream"`
				Data   []byte `json:"data_base64"`
			} `json:"frames"`
		} `json:"expected"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	var result []byte
	for _, frame := range fixture.Expected.Frames {
		if frame.Stream == "choice" {
			result = append(result, frame.Data...)
		}
	}
	if len(result) == 0 {
		t.Fatalf("%s has no retained choice evidence", name)
	}
	return result
}

func mustReplayPlanTrace(t *testing.T, payload []byte) Trace {
	t.Helper()
	trace, err := DecodeStoredTrace(Profile, payload, TerminalMetadata{State: TerminalComplete, Limit: 8192, Records: uint64(len(payload) / traceRecordBytes), SHA256: sha256.Sum256(payload)})
	if err != nil {
		t.Fatal(err)
	}
	return trace
}
