package backends

import (
	"bytes"
	"crypto/sha256"
	"reflect"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/model/choice"
)

func requestTrace(t *testing.T) (choice.Trace, choice.ReplayPlan) {
	t.Helper()
	identities := [][32]byte{sha256.Sum256([]byte("first")), sha256.Sum256([]byte("second"))}
	records := make([]choice.Record, 2)
	for index := range records {
		decision, err := choice.CanonicalDecision(uint64(index), choice.KindRunnable, uint64(index+1), false, identities, identities[index], 0)
		if err != nil {
			t.Fatal(err)
		}
		records[index] = decision.Record()
	}
	trace, err := choice.BuildTrace(records, choice.TerminalComplete)
	if err != nil {
		t.Fatal(err)
	}
	identity := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256([]byte("target")), ToolchainBuildKey: strings.Repeat("a", 64), GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte("implementation"))}
	plan, err := choice.ProjectReplayPlan(trace, identity)
	if err != nil {
		t.Fatal(err)
	}
	return trace, plan
}

func TestChoiceRequestRetainsControllerSourceAndPrefixSemantics(t *testing.T) {
	t.Parallel()
	trace, source := requestTrace(t)
	for _, test := range []struct {
		name        string
		cutoff      uint64
		override    bool
		reconstruct bool
	}{
		{name: "zero"}, {name: "short", cutoff: 1}, {name: "full", cutoff: 2},
		{name: "new rank", cutoff: 2, override: true}, {name: "reconstruct", cutoff: 2, override: true, reconstruct: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			plan, err := source.Prefix(test.cutoff)
			control := &ChoicePrefixControl{Cutoff: test.cutoff}
			if test.override {
				rank := source.Decisions[test.cutoff-1].Selected
				if test.reconstruct {
					plan, err = choice.BuildForcedRankPrefix(source, test.cutoff-1, rank)
				} else {
					rank = (rank + 1) % source.Decisions[test.cutoff-1].Alternatives
					plan, err = choice.BuildRankPrefix(source, test.cutoff-1, rank)
				}
				control.Override = &ChoiceRankOverride{Ordinal: test.cutoff - 1, Rank: rank, Reconstruct: test.reconstruct}
			}
			if err != nil {
				t.Fatal(err)
			}
			request, err := ChoiceRequestForCapability(&ChoiceCapability{Mode: choice.ModePrefix, ExecutionIdentity: source.Identity, Limit: 4096, ReplayPlan: &plan, Trace: trace.Bytes, TraceSHA256: trace.SHA256})
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(request.Trace, trace.Bytes) || request.TraceSHA256 != trace.SHA256 || request.TapeSHA256 != plan.SHA256 || request.Decisions != test.cutoff || !reflect.DeepEqual(request.PrefixControl, control) {
				t.Fatalf("controller request differs: %+v", request)
			}
		})
	}
}

func TestChoiceRequestRejectsControllerSourceMismatch(t *testing.T) {
	t.Parallel()
	trace, plan := requestTrace(t)
	for _, test := range []struct {
		name   string
		change func(*ChoiceCapability)
	}{
		{"missing source", func(c *ChoiceCapability) { c.Trace = nil }},
		{"source digest", func(c *ChoiceCapability) { c.TraceSHA256 = [32]byte{} }},
		{"source bytes", func(c *ChoiceCapability) { c.Trace = []byte("invalid") }},
		{"identity", func(c *ChoiceCapability) { c.ExecutionIdentity.TargetSHA256 = [32]byte{} }},
		{"explicit controls", func(c *ChoiceCapability) { c.PrefixControl = &ChoicePrefixControl{} }},
		{"exact controller", func(c *ChoiceCapability) { c.Mode = choice.ModeReplay }},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			capability := &ChoiceCapability{Mode: choice.ModePrefix, ExecutionIdentity: plan.Identity, Limit: 4096, ReplayPlan: &plan, Trace: trace.Bytes, TraceSHA256: trace.SHA256}
			test.change(capability)
			if _, err := ChoiceRequestForCapability(capability); err == nil {
				t.Fatal("accepted inconsistent controller source")
			}
		})
	}
}

func TestObservedReplayPreservesHelperAttestationWithoutDecoding(t *testing.T) {
	t.Parallel()
	trace, plan := requestTrace(t)
	trace.Records = nil
	base := Result{Termination: Exit, Reaped: true, Evidence: []byte("evidence"), ChoiceTrace: trace, ImplementationSHA256: plan.Identity.ImplementationSHA256, InputTapeSHA256: plan.SHA256, InputDecisions: uint64(len(plan.Decisions))}
	request := Request{OutputBytes: 1024, Choice: &ChoiceRequest{Mode: choice.ModeReplay, ExecutionIdentity: plan.Identity, Limit: 4096, Trace: trace.Bytes, TraceSHA256: trace.SHA256, TapeSHA256: plan.SHA256, Decisions: uint64(len(plan.Decisions))}}
	result, err := ProjectObservedResult(request, base, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result.ChoiceTrace.Trace, trace) {
		t.Fatal("replay decoded or changed helper-attested trace")
	}
	for _, test := range []struct {
		name   string
		change func(*Result)
	}{
		{"implementation", func(r *Result) { r.ImplementationSHA256 = [32]byte{} }},
		{"admitted digest", func(r *Result) { r.InputTapeSHA256 = [32]byte{} }},
		{"admitted count", func(r *Result) { r.InputDecisions++ }},
		{"incomplete", func(r *Result) { r.ChoiceTrace.Summary.Terminal = choice.TerminalOverflow }},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			observed := base
			test.change(&observed)
			if _, err := ProjectObservedResult(request, observed, nil, nil); err == nil {
				t.Fatal("accepted inconsistent replay attestation")
			}
		})
	}
}
