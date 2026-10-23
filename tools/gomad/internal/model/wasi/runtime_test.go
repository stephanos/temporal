package wasi

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/model/choice"
)

func TestCooperativeFrozenClockPreservesStartupAndStockPolicy(t *testing.T) {
	config := testConfig()
	config.Clock.ReadStepNanos = 0
	if _, err := NewEnvironment(config); err == nil {
		t.Fatal("stock profile admitted frozen clock")
	}
	config.Profile = CooperativeProfile
	e := newTestEnvironment(t, config)
	for range 2 {
		var timestamp timestampOutput
		output(t, call(t, e, "clock_time_get", `{"clock_id":1,"precision":0}`), &timestamp)
		if timestamp.Timestamp != 1 || e.now != 1 || e.config.Clock.ReadStepNanos != 0 {
			t.Fatalf("frozen monotonic clock advanced: %+v now=%d policy=%+v", timestamp, e.now, e.config.Clock)
		}
	}
}

func TestCooperativeEnvironmentSeparatesRuntimeControlFromWASIObservations(t *testing.T) {
	config := testConfig()
	config.Profile = CooperativeProfile
	environment, err := NewEnvironment(config)
	if err != nil {
		t.Fatal(err)
	}
	identity := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256([]byte("guest")), ToolchainBuildKey: strings.Repeat("a", 64), GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte("runtime"))}
	environment.runtime, err = newRuntimeSession(&RuntimeControl{Choice: &choice.Request{Mode: choice.ModeRecord, ExecutionIdentity: identity, Limit: 8192}})
	if err != nil {
		t.Fatal(err)
	}
	input, err := json.Marshal(bytesOutput{Data: make([]byte, 16)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := environment.Handle(Call{Type: "call", ID: 1, Op: "runtime_config", Input: input}); err != nil {
		t.Fatal(err)
	}
	if len(environment.Transcript()) != 0 {
		t.Fatal("control payload leaked into replay observation")
	}
	deadline := make([]byte, 16)
	binary.BigEndian.PutUint32(deadline, 1)
	binary.BigEndian.PutUint64(deadline[8:], 1000000)
	input, err = json.Marshal(bytesOutput{Data: deadline})
	if err != nil {
		t.Fatal(err)
	}
	reply, err := environment.Handle(Call{Type: "call", ID: 2, Op: "runtime_idle", Input: input})
	if err != nil {
		t.Fatal(err)
	}
	var data bytesOutput
	if err := json.Unmarshal(reply.Output, &data); err != nil || binary.BigEndian.Uint64(data.Data[8:]) != 1000000 || environment.now != 1000000 {
		t.Fatalf("idle deadline handshake = %#v, %v", data, err)
	}
	if _, err := environment.Handle(Call{Type: "call", ID: 3, Op: "random_get", Input: json.RawMessage(`{"length":8}`)}); err != nil {
		t.Fatal(err)
	}
	records, err := DecodeTranscript(environment.Transcript())
	if err != nil || len(records) != 1 || records[0].Op != "random_get" || records[0].ID != 3 {
		t.Fatal("runtime and application observation domains mixed")
	}
	before := environment.Transcript()
	if _, err := environment.Handle(Call{Type: "call", ID: 4, Op: "runtime_net", Input: json.RawMessage(`{"data_base64":""}`)}); err == nil {
		t.Fatal("retired network v1 callback admitted")
	}
	if environment.nextCall != 4 || !bytes.Equal(environment.Transcript(), before) {
		t.Fatal("retired network v1 callback mutated environment")
	}
}

func runtimeChoiceInput(kind uint32, site uint64, identities [][32]byte, selected uint32) []byte {
	data := make([]byte, 32+32*len(identities))
	binary.BigEndian.PutUint32(data, 1)
	binary.BigEndian.PutUint32(data[4:], kind)
	binary.BigEndian.PutUint64(data[8:], site)
	binary.BigEndian.PutUint32(data[20:], uint32(len(identities)))
	binary.BigEndian.PutUint32(data[24:], selected)
	for i, id := range identities {
		copy(data[32+32*i:], id[:])
	}
	return data
}

func TestRuntimeTransportForcesLogicalAlternativesAndConsumesWholeTape(t *testing.T) {
	identity := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256([]byte("guest")), ToolchainBuildKey: strings.Repeat("a", 64), GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte("runtime"))}
	a, b := sha256.Sum256([]byte("a")), sha256.Sum256([]byte("b"))
	control := &RuntimeControl{Seed: 7, Choice: &choice.Request{Mode: choice.ModeRecord, ExecutionIdentity: identity, Limit: 8192}}
	observed, err := newRuntimeSession(control)
	if err != nil {
		t.Fatal(err)
	}
	invoke := func(session *runtimeSession, name string, bytes []byte) ([]byte, error) {
		input, err := json.Marshal(bytesOutput{Data: bytes})
		if err != nil {
			t.Fatal(err)
		}
		value, err := session.handle(name, input)
		return value.Data, err
	}
	if _, err := invoke(observed, "runtime_config", make([]byte, 16)); err != nil {
		t.Fatal(err)
	}
	if _, err := invoke(observed, "runtime_decision", runtimeChoiceInput(1, 10, [][32]byte{a, b}, 0)); err != nil {
		t.Fatal(err)
	}
	finish := make([]byte, 16)
	binary.BigEndian.PutUint32(finish, 1)
	if _, err := invoke(observed, "runtime_finish", finish); err != nil {
		t.Fatal(err)
	}
	trace, _, err := observed.collect()
	if err != nil || len(trace.Records) != 1 || trace.Records[0].SelectedIdentity != a {
		t.Fatalf("recorded logical choice: %#v %v", trace, err)
	}
	tape, err := choice.ProjectReplayPlan(trace, identity)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"exact", "missing", "extra", "kind", "high-kind", "site", "enabled", "rank", "readiness", "duplicate", "unconfigured", "unfinished"} {
		t.Run(name, func(t *testing.T) {
			plan := tape
			source := bytes.Clone(trace.Bytes)
			mode := choice.ModeReplay
			if name == "missing" {
				source = nil
				plan, err = choice.ProjectInterruptedReplayPlan(source, 8192, identity)
				if err != nil {
					t.Fatal(err)
				}
			}
			if name == "rank" {
				binary.BigEndian.PutUint32(source[16:20], 99)
			}
			if name == "readiness" {
				binary.BigEndian.PutUint16(source[10:12], uint16(choice.ReadinessKnown))
			}
			request := &choice.Request{Mode: mode, ExecutionIdentity: identity, Limit: 8192}
			setChoiceReplay(t, request, mode, source, plan)
			replay, err := newRuntimeSession(&RuntimeControl{Seed: 100, Choice: request})
			if name == "rank" || name == "readiness" {
				if err == nil {
					_, err = invoke(replay, "runtime_config", make([]byte, 16))
				}
				if err == nil {
					t.Fatal("malformed source rank or readiness accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if name != "unconfigured" {
				if _, err := invoke(replay, "runtime_config", make([]byte, 16)); err != nil {
					t.Fatal(err)
				}
			}
			input := runtimeChoiceInput(1, 10, [][32]byte{b, a}, 0)
			switch name {
			case "kind":
				binary.BigEndian.PutUint32(input[4:], 2)
			case "high-kind":
				binary.BigEndian.PutUint32(input[4:], 257)
			case "site":
				binary.BigEndian.PutUint64(input[8:], 11)
			case "enabled":
				input = runtimeChoiceInput(1, 10, [][32]byte{a, sha256.Sum256([]byte("other"))}, 0)
			case "duplicate":
				input = runtimeChoiceInput(1, 10, [][32]byte{a, a}, 0)
			}
			if name != "extra" {
				output, callErr := invoke(replay, "runtime_decision", input)
				err = callErr
				if name == "exact" || name == "unfinished" {
					if err != nil || binary.BigEndian.Uint32(output[24:]) != 1 {
						t.Fatalf("logical rank failed across physical order: %x %v", output, err)
					}
				} else if err == nil {
					t.Fatal("inapplicable decision accepted")
				}
				if name == "duplicate" || name == "high-kind" {
					var boundary *BoundaryError
					if !errors.As(err, &boundary) || boundary.Kind != "invalid" {
						t.Fatalf("malformed guest decision was not invalid: %v", err)
					}
				}
			}
			if name == "extra" {
				if _, err := invoke(replay, "runtime_finish", finish); err == nil {
					t.Fatal("unconsumed tape accepted")
				}
			}
			if name == "exact" {
				if _, err := invoke(replay, "runtime_finish", finish); err != nil {
					t.Fatal(err)
				}
				forced, _, err := replay.collect()
				if err != nil || forced.SHA256 != trace.SHA256 {
					t.Fatalf("forced complete trace differs: %x %v", forced.SHA256, err)
				}
			}
			if name == "unfinished" {
				if _, _, err := replay.collect(); err == nil {
					t.Fatal("missing runtime finish accepted")
				}
			}
		})
	}
}

// runtimeCreationInput is a header v2 runnable decision: the v1 layout
// followed by one 16-byte creation site (offset, flags, zero padding) per
// alternative, in the guest's physical alternative order.
func runtimeCreationInput(site uint64, siteMissing bool, identities [][32]byte, sites []uint64, selected uint32) []byte {
	data := runtimeChoiceInput(1, site, identities, selected)
	binary.BigEndian.PutUint32(data, 2)
	if siteMissing {
		binary.BigEndian.PutUint32(data[16:], uint32(choice.FlagSiteMissing))
	}
	for _, offset := range sites {
		entry := make([]byte, 16)
		binary.BigEndian.PutUint64(entry, offset)
		if offset == 0 {
			entry[8] = byte(choice.FlagSiteMissing)
		}
		data = append(data, entry...)
	}
	return data
}

// The Go host must record a header v2 runnable decision exactly as the
// helper does: the shared creation-site replay-plan golden is the oracle for
// the trace bytes and the projected tape, and that tape replays exactly.
func TestRuntimeHeaderV2CreationSitesMatchHelperGoldenAndReplayExactly(t *testing.T) {
	raw, err := os.ReadFile("../../../rust/wasmhost/tests/testdata/replay_plan/runnable-creation-sites.json")
	if err != nil {
		t.Fatal(err)
	}
	var golden struct {
		Identity choice.ExecutionIdentity
		Trace    []byte
		Expected struct {
			Bytes  []byte
			SHA256 string
		}
	}
	if err := json.Unmarshal(raw, &golden); err != nil {
		t.Fatal(err)
	}
	identity := golden.Identity
	one, two := [32]byte{1}, [32]byte{2}
	// Physical order differs from identity order; identity two has no site.
	decision := runtimeCreationInput(0, true, [][32]byte{two, one}, []uint64{0, 123}, 0)
	finish := make([]byte, 16)
	binary.BigEndian.PutUint32(finish, 1)
	run := func(t *testing.T, request *choice.Request, diagnostics bool) (choice.Trace, choice.DiagnosticTrace, []byte) {
		t.Helper()
		session, err := newRuntimeSession(&RuntimeControl{Seed: 7, Choice: request, Diagnostics: diagnostics})
		if err != nil {
			t.Fatal(err)
		}
		input := decision
		if diagnostics {
			input = append(slices.Clone(decision), make([]byte, 96)...)
		}
		var selected []byte
		for _, step := range []struct {
			op   string
			data []byte
		}{{"runtime_config", make([]byte, 16)}, {"runtime_decision", input}, {"runtime_finish", finish}} {
			encoded, err := json.Marshal(bytesOutput{Data: step.data})
			if err != nil {
				t.Fatal(err)
			}
			output, err := session.handle(step.op, encoded)
			if err != nil {
				t.Fatalf("%s: %v", step.op, err)
			}
			if step.op == "runtime_decision" {
				selected = output.Data[24:28]
			}
		}
		trace, diagnostic, err := session.collect()
		if err != nil {
			t.Fatal(err)
		}
		return trace, diagnostic, selected
	}
	recorded, _, _ := run(t, &choice.Request{Mode: choice.ModeRecord, ExecutionIdentity: identity, Limit: 8192}, false)
	if !bytes.Equal(recorded.Bytes, golden.Trace) {
		t.Fatalf("Go host trace differs from the helper golden:\n got %x\nwant %x", recorded.Bytes, golden.Trace)
	}
	if got := choice.PredecisionOrdinals(recorded.Records); !slices.Equal(got, []uint64{0, 0, 0}) {
		t.Fatalf("predecision ordinals = %v, want the decision's boundary coordinate 0", got)
	}
	plan, err := choice.ProjectReplayPlan(recorded, identity)
	if err != nil || !bytes.Equal(plan.Bytes, golden.Expected.Bytes) || hex.EncodeToString(plan.SHA256[:]) != golden.Expected.SHA256 {
		t.Fatalf("projected tape differs from the helper golden: %x %v", plan.SHA256, err)
	}
	t.Run("exact-replay", func(t *testing.T) {
		request := &choice.Request{ExecutionIdentity: identity, Limit: 8192}
		setChoiceReplay(t, request, choice.ModeReplay, recorded.Bytes, plan)
		replayed, _, selected := run(t, request, false)
		if replayed.SHA256 != recorded.SHA256 || binary.BigEndian.Uint32(selected) != 0 {
			t.Fatalf("replay trace %x selected %x, want %x selected 0", replayed.SHA256, selected, recorded.SHA256)
		}
	})
	t.Run("diagnostics-follow-trace-ordinals", func(t *testing.T) {
		_, diagnostic, _ := run(t, &choice.Request{Mode: choice.ModeRecord, ExecutionIdentity: identity, Limit: 8192}, true)
		if len(diagnostic.Records) != 3 {
			t.Fatalf("diagnostic records = %d, want one per trace record", len(diagnostic.Records))
		}
		for i, record := range diagnostic.Records {
			if record.Ordinal != uint64(i) {
				t.Fatalf("diagnostic %d has ordinal %d", i, record.Ordinal)
			}
		}
	})
}

// Header v2 widens only runnable decisions; every other layout, and any
// creation site the helper refuses, stays an invalid boundary that records
// nothing.
func TestRuntimeDecisionRefusesMalformedHeaderV2Layouts(t *testing.T) {
	identity := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256([]byte("guest")), ToolchainBuildKey: strings.Repeat("a", 64), GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte("runtime"))}
	a, b := sha256.Sum256([]byte("a")), sha256.Sum256([]byte("b"))
	valid := func() []byte { return runtimeCreationInput(10, false, [][32]byte{a, b}, []uint64{5, 0}, 0) }
	for _, test := range []struct {
		name      string
		input     func() []byte
		dimension string
	}{
		{"header-3", func() []byte { data := valid(); binary.BigEndian.PutUint32(data, 3); return data }, "runtime decision header"},
		{"v2-runnable-without-sites", func() []byte {
			data := runtimeChoiceInput(1, 10, [][32]byte{a, b}, 0)
			binary.BigEndian.PutUint32(data, 2)
			return data
		}, "runtime decision alternatives or layout"},
		{"v1-runnable-with-sites", func() []byte { data := valid(); binary.BigEndian.PutUint32(data, 1); return data }, "runtime decision alternatives or layout"},
		{"v2-select-poll-with-sites", func() []byte { data := valid(); binary.BigEndian.PutUint32(data[4:], 2); return data }, "runtime decision alternatives or layout"},
		{"truncated-site", func() []byte { data := valid(); return data[:len(data)-1] }, "runtime decision alternatives or layout"},
		{"site-flag", func() []byte { data := valid(); data[96+8] = 1; return data }, "runtime creation site"},
		{"site-padding", func() []byte { data := valid(); data[96+15] = 1; return data }, "runtime creation site"},
		{"missing-site-offset", func() []byte { data := valid(); data[96+16+7] = 1; return data }, "runtime creation site"},
	} {
		t.Run(test.name, func(t *testing.T) {
			session, err := newRuntimeSession(&RuntimeControl{Choice: &choice.Request{Mode: choice.ModeRecord, ExecutionIdentity: identity, Limit: 8192}})
			if err != nil {
				t.Fatal(err)
			}
			for _, step := range []struct {
				op   string
				data []byte
			}{{"runtime_config", make([]byte, 16)}, {"runtime_decision", test.input()}} {
				encoded, err := json.Marshal(bytesOutput{Data: step.data})
				if err != nil {
					t.Fatal(err)
				}
				_, err = session.handle(step.op, encoded)
				if step.op == "runtime_config" {
					if err != nil {
						t.Fatal(err)
					}
					continue
				}
				var boundary *BoundaryError
				if !errors.As(err, &boundary) || boundary.Kind != "invalid" || boundary.Dimension != test.dimension {
					t.Fatalf("malformed decision = %v, want invalid %q", err, test.dimension)
				}
			}
			if len(session.records) != 0 {
				t.Fatalf("refused decision recorded %d records", len(session.records))
			}
		})
	}
	t.Run("valid", func(t *testing.T) {
		session, err := newRuntimeSession(&RuntimeControl{Choice: &choice.Request{Mode: choice.ModeRecord, ExecutionIdentity: identity, Limit: 8192}})
		if err != nil {
			t.Fatal(err)
		}
		for _, step := range [][2]any{{"runtime_config", make([]byte, 16)}, {"runtime_decision", valid()}} {
			encoded, _ := json.Marshal(bytesOutput{Data: step[1].([]byte)})
			if _, err := session.handle(step[0].(string), encoded); err != nil {
				t.Fatal(err)
			}
		}
		if len(session.records) != 3 || session.records[2].Ordinal != 2 || session.records[2].Kind != choice.KindRunnable {
			t.Fatalf("valid header v2 decision records = %+v", session.records)
		}
	})
	t.Run("capacity-counts-creation-records", func(t *testing.T) {
		// Room for two records, but the decision needs three.
		session, err := newRuntimeSession(&RuntimeControl{Choice: &choice.Request{Mode: choice.ModeRecord, ExecutionIdentity: identity, Limit: 64 + 2*96}})
		if err != nil {
			t.Fatal(err)
		}
		for _, step := range [][2]any{{"runtime_config", make([]byte, 16)}, {"runtime_decision", valid()}} {
			encoded, _ := json.Marshal(bytesOutput{Data: step[1].([]byte)})
			_, err = session.handle(step[0].(string), encoded)
		}
		var boundary *BoundaryError
		if !errors.As(err, &boundary) || boundary.Kind != "capacity" || len(session.records) != 0 {
			t.Fatalf("over-capacity creation decision = %v records=%d", err, len(session.records))
		}
	})
}

func setChoiceReplay(t *testing.T, request *choice.Request, mode choice.Mode, source []byte, plan choice.ReplayPlan) {
	t.Helper()
	request.Mode, request.Trace = mode, source
	request.TraceSHA256, request.TapeSHA256, request.Decisions = sha256.Sum256(source), plan.SHA256, uint64(len(plan.Decisions))
	if mode != choice.ModePrefix {
		return
	}
	validated, err := choice.ValidatePrefixReplayPlan(plan, request.ExecutionIdentity)
	if err != nil {
		t.Fatal(err)
	}
	records, err := choice.TraceRecordCount(source)
	if err != nil {
		t.Fatal(err)
	}
	trace, err := choice.DecodeStoredTrace(choice.Profile, source, choice.TerminalMetadata{State: choice.TerminalComplete, Limit: request.Limit, Records: records, SHA256: request.TraceSHA256})
	if err != nil {
		t.Fatal(err)
	}
	control := &choice.PrefixControl{Cutoff: uint64(len(validated.Decisions))}
	var ordinal uint64
	for _, observed := range trace.Records {
		if observed.Flags&choice.FlagObservation != 0 || observed.Alternatives < 2 {
			continue
		}
		if ordinal+1 == control.Cutoff {
			last := validated.Decisions[ordinal]
			if last.RankOverride {
				control.Override = &choice.RankOverride{Ordinal: ordinal, Rank: last.Selected, Reconstruct: observed.Selected == last.Selected}
			}
		}
		ordinal++
	}
	if control.Cutoff > ordinal {
		t.Fatal("controller prefix exceeds retained source trace")
	}
	request.TapeSHA256, request.Decisions, request.PrefixControl = validated.SHA256, control.Cutoff, control
}
