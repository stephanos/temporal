package exploration

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	"go.temporal.io/server/tools/gomad3/internal/model/choice"
)

func TestRuntimeDecisionsProjectsBranchingChoiceTapeByRank(t *testing.T) {
	first := sha256.Sum256([]byte("first runnable"))
	second := sha256.Sum256([]byte("second runnable"))
	decision, err := choice.CanonicalDecision(0, choice.KindRunnable, 24, false, [][sha256.Size]byte{first, second}, second, 0)
	if err != nil {
		t.Fatal(err)
	}
	trace, err := choice.BuildTrace([]choice.Record{decision.Record()}, choice.TerminalComplete)
	if err != nil {
		t.Fatal(err)
	}
	identity := choice.ExecutionIdentity{
		TargetSHA256: sha256.Sum256([]byte("target")), ToolchainBuildKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		GOOS: "darwin", GOARCH: "arm64", ImplementationSHA256: sha256.Sum256([]byte("implementation")),
	}
	tape, err := choice.ProjectReplayPlan(trace, identity)
	if err != nil {
		t.Fatal(err)
	}

	decisions, err := RuntimeDecisions(tape)
	if err != nil {
		t.Fatal(err)
	}
	if len(decisions) != 1 || decisions[0].Dimension != DimensionRuntime || decisions[0].Ordinal != 0 || len(decisions[0].Alternatives) != 2 || decisions[0].Selected != 1 {
		t.Fatalf("runtime decisions = %#v", decisions)
	}
	canonical, err := CanonicalControlledDecision(
		DimensionRuntime, decisions[0].Ordinal, decisions[0].SiteSHA256, decisions[0].Alternatives, decisions[0].AlternativeControls, decisions[0].Selected,
	)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := json.Marshal(decisions[0])
	if err != nil {
		t.Fatal(err)
	}
	want, err := json.Marshal(canonical)
	if err != nil {
		t.Fatal(err)
	}
	if string(actual) != string(want) {
		t.Fatalf("runtime decision = %s, canonical = %s", actual, want)
	}
}

func TestExecutionForCandidateRestoresRuntimeRankPrefix(t *testing.T) {
	first := sha256.Sum256([]byte("first runnable"))
	second := sha256.Sum256([]byte("second runnable"))
	decision, err := choice.CanonicalDecision(0, choice.KindRunnable, 24, false, [][sha256.Size]byte{first, second}, first, 0)
	if err != nil {
		t.Fatal(err)
	}
	trace, err := choice.BuildTrace([]choice.Record{decision.Record()}, choice.TerminalComplete)
	if err != nil {
		t.Fatal(err)
	}
	identity := choice.ExecutionIdentity{
		TargetSHA256: sha256.Sum256([]byte("target")), ToolchainBuildKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		GOOS: "darwin", GOARCH: "arm64", ImplementationSHA256: sha256.Sum256([]byte("implementation")),
	}
	tape, err := choice.ProjectReplayPlan(trace, identity)
	if err != nil {
		t.Fatal(err)
	}
	runtimeDecisions, err := RuntimeDecisions(tape)
	if err != nil {
		t.Fatal(err)
	}
	config := SimulationConfig{
		ExecutionSHA256: record.HashBytes([]byte("execution")), ControllerSHA256: SimulationImplementationSHA256(), BaseSeed: 17,
		Parallel: 1, MaxExecutions: 2, MaxForcedDecisions: 1, MaxExplorationBytes: 1 << 20, MaxResultBytes: 1 << 20, FailureBudget: 1,
		Limits: DimensionLimits{Runtime: 1, Scenario: 1, Network: 1, Storage: 1, Fault: 1, Crash: 1},
	}
	state, err := NewSimulation(config)
	if err != nil {
		t.Fatal(err)
	}
	root, _ := state.NextRound()
	state, _, err = CommitSimulationRound(state, root, []SimulationResult{{
		CandidateSHA256: root.Candidates[0].SHA256, OutcomeSHA256: record.HashBytes([]byte("root")), Decisions: runtimeDecisions,
	}})
	if err != nil {
		t.Fatal(err)
	}
	children, ok := state.NextRound()
	if !ok || len(children.Candidates) != 1 {
		t.Fatalf("runtime child round = %#v", children)
	}

	execution, err := ExecutionForCandidate(config, children.Candidates[0], identity)
	if err != nil {
		t.Fatal(err)
	}
	if execution.ChoiceMode != choice.ModePrefix || execution.ChoiceReplayPlan == nil || len(execution.ChoiceReplayPlan.Decisions) != 1 || !execution.ChoiceReplayPlan.Decisions[0].RankOverride || execution.ChoiceReplayPlan.Decisions[0].Selected != 1 {
		t.Fatalf("runtime candidate execution = %#v", execution)
	}
}

func TestCandidateForArtifactRestoresRuntimeControlAndFaultOverride(t *testing.T) {
	identity := choice.ExecutionIdentity{
		TargetSHA256: sha256.Sum256([]byte("target")), ToolchainBuildKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		GOOS: "darwin", GOARCH: "arm64", ImplementationSHA256: sha256.Sum256([]byte("implementation")),
	}
	unforcedRuntime := testRuntimeDecision(t, 0, 0)
	unforcedTrace, err := choice.BuildTrace([]choice.Record{unforcedRuntime.Record()}, choice.TerminalComplete)
	if err != nil {
		t.Fatal(err)
	}
	unforcedTape, err := choice.ProjectReplayPlan(unforcedTrace, identity)
	if err != nil {
		t.Fatal(err)
	}
	runtimeDecisions, err := RuntimeDecisions(unforcedTape)
	if err != nil {
		t.Fatal(err)
	}
	runtimeOverride, err := ForceDecision(runtimeDecisions[0], 1)
	if err != nil {
		t.Fatal(err)
	}
	faultDecision, err := CanonicalDecision(
		DimensionFault, 0, record.HashBytes([]byte("fault site")),
		[]record.SHA256{record.HashBytes([]byte("default")), record.HashBytes([]byte("drop"))}, 0,
	)
	if err != nil {
		t.Fatal(err)
	}
	faultOverride, err := ForceDecision(faultDecision, 1)
	if err != nil {
		t.Fatal(err)
	}
	config := testCombinedConfig()
	candidate, err := CanonicalCandidate(config, []ForcedDecision{runtimeOverride, faultOverride}, "")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanForCandidate(config, candidate)
	if err != nil {
		t.Fatal(err)
	}
	observedRuntime := testRuntimeDecision(t, 0, 1)
	observedTrace, err := choice.BuildTrace([]choice.Record{observedRuntime.Record()}, choice.TerminalComplete)
	if err != nil {
		t.Fatal(err)
	}
	observedTape, err := choice.ProjectReplayPlan(observedTrace, identity)
	if err != nil {
		t.Fatal(err)
	}
	profile := record.SimulationProfile{
		ControllerSHA256: config.ControllerSHA256, ExecutionSHA256: config.ExecutionSHA256, CandidateSHA256: candidate.SHA256,
	}

	gotConfig, gotCandidate, err := CandidateForArtifact(profile, plan, &observedTape)
	if err != nil {
		t.Fatal(err)
	}
	if gotConfig.ExecutionSHA256 != config.ExecutionSHA256 || gotConfig.BaseSeed != config.BaseSeed || gotCandidate.SHA256 != candidate.SHA256 || len(gotCandidate.Overrides) != 2 {
		t.Fatalf("reconstructed candidate = %#v with config %#v", gotCandidate, gotConfig)
	}
	if string(gotCandidate.Overrides[0].Control) != string(runtimeOverride.Control) || gotCandidate.Overrides[1].Dimension != DimensionFault {
		t.Fatalf("reconstructed overrides = %#v", gotCandidate.Overrides)
	}
}

func TestResultForRecordProjectsValidatedDecisionsAndSemanticOutcome(t *testing.T) {
	config := SimulationConfig{
		ExecutionSHA256: record.HashBytes([]byte("execution")), ControllerSHA256: SimulationImplementationSHA256(), BaseSeed: 17,
		Parallel: 1, MaxExecutions: 2, MaxForcedDecisions: 1, MaxExplorationBytes: 1 << 20, MaxResultBytes: 1 << 20, FailureBudget: 1,
		Limits: DimensionLimits{Runtime: 1, Scenario: 1, Network: 1, Storage: 1, Fault: 1, Crash: 1},
	}
	state, err := NewSimulation(config)
	if err != nil {
		t.Fatal(err)
	}
	round, ok := state.NextRound()
	if !ok {
		t.Fatal("simulation exploration root round is unavailable")
	}
	candidate := round.Candidates[0]
	plan, err := PlanForCandidate(config, candidate)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := CanonicalDecision(
		DimensionScenario, 0, record.HashBytes([]byte("route")),
		[]record.SHA256{record.HashBytes([]byte("alpha")), record.HashBytes([]byte("beta"))}, 1,
	)
	if err != nil {
		t.Fatal(err)
	}
	recordValue := struct {
		Schema               string          `json:"schema"`
		Seed                 uint64          `json:"seed"`
		SpecSHA256           record.SHA256   `json:"spec_sha256"`
		Outcome              string          `json:"outcome"`
		ExplorationPlan      json.RawMessage `json:"exploration_plan"`
		ExplorationDecisions []Decision      `json:"exploration_decisions"`
		ScenarioTape         []string        `json:"scenario_tape"`
		Identity             record.SHA256   `json:"identity"`
	}{
		Schema: "gomad3.cluster-record/v8", Seed: 17, SpecSHA256: record.HashBytes([]byte("candidate-bound-spec")), Outcome: "completed",
		ExplorationPlan: plan, ExplorationDecisions: []Decision{decision}, ScenarioTape: []string{"beta"}, Identity: record.HashBytes([]byte("record")),
	}
	encoded, err := json.Marshal(recordValue)
	if err != nil {
		t.Fatal(err)
	}

	result, err := ResultForRecord(config, candidate, encoded, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.CandidateSHA256 != candidate.SHA256 || result.Failed || result.Diverged || len(result.Decisions) != 1 || result.Decisions[0].Identity != decision.Identity {
		t.Fatalf("result = %#v", result)
	}
	if _, err := result.OutcomeSHA256.Bytes(); err != nil {
		t.Fatalf("outcome identity = %q: %v", result.OutcomeSHA256, err)
	}

	recordValue.ExplorationDecisions[0].Identity = record.HashBytes([]byte("changed"))
	changed, err := json.Marshal(recordValue)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ResultForRecord(config, candidate, changed, nil); err == nil {
		t.Fatal("ResultForRecord() accepted a changed decision identity")
	}
}

func TestResultForRecordRejectsUnknownOwnedDecisionFields(t *testing.T) {
	config := testCombinedConfig()
	state, err := NewSimulation(config)
	if err != nil {
		t.Fatal(err)
	}
	round, ok := state.NextRound()
	if !ok {
		t.Fatal("simulation exploration root round is unavailable")
	}
	candidate := round.Candidates[0]
	plan, err := PlanForCandidate(config, candidate)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := CanonicalDecision(
		DimensionScenario, 0, record.HashBytes([]byte("route")),
		[]record.SHA256{record.HashBytes([]byte("alpha")), record.HashBytes([]byte("beta"))}, 1,
	)
	if err != nil {
		t.Fatal(err)
	}
	encodedDecision, err := json.Marshal(decision)
	if err != nil {
		t.Fatal(err)
	}
	encodedDecision = append(encodedDecision[:len(encodedDecision)-1], []byte(`,"producer_owned":true}`)...)
	recordBytes := []byte(fmt.Sprintf(
		`{"schema":"%s","seed":%d,"spec_sha256":%q,"outcome":"completed","exploration_plan":%s,"exploration_decisions":[%s],"identity":%q}`,
		recordSchema, config.BaseSeed, record.HashBytes([]byte("spec")), plan, encodedDecision, record.HashBytes([]byte("record")),
	))

	if _, err := ResultForRecord(config, candidate, recordBytes, nil); err == nil {
		t.Fatal("ResultForRecord() accepted an unknown owned decision field")
	}
}

func TestResultForRecordRejectsUnpairedSurrogateInProjectedProducerField(t *testing.T) {
	config := testCombinedConfig()
	state, err := NewSimulation(config)
	if err != nil {
		t.Fatal(err)
	}
	round, ok := state.NextRound()
	if !ok {
		t.Fatal("simulation exploration root round is unavailable")
	}
	candidate := round.Candidates[0]
	plan, err := PlanForCandidate(config, candidate)
	if err != nil {
		t.Fatal(err)
	}
	recordBytes := []byte(fmt.Sprintf(
		`{"schema":"%s","seed":%d,"spec_sha256":%q,"outcome":"completed","exploration_plan":%s,"producer_owned":"\ud800","identity":%q}`,
		recordSchema, config.BaseSeed, record.HashBytes([]byte("spec")), plan, record.HashBytes([]byte("record")),
	))

	_, err = ResultForRecord(config, candidate, recordBytes, nil)
	if err == nil || !strings.Contains(err.Error(), "surrogate") {
		t.Fatalf("ResultForRecord() unpaired-surrogate error = %v", err)
	}
}

func TestProjectRetainedRecordRejectsMalformedOwnedFieldsIndependently(t *testing.T) {
	expectedPlan := []byte(`{"plan":"exact"}`)
	valid := fmt.Sprintf(
		`{"schema":%q,"seed":17,"spec_sha256":%q,"outcome":"completed","exploration_plan":%s,"producer_owned":{"value":1},"identity":%q}`,
		recordSchema, record.HashBytes([]byte("spec")), expectedPlan, record.HashBytes([]byte("record")),
	)
	tests := []struct {
		name  string
		value []byte
	}{
		{name: "malformed", value: []byte(`{"schema":`)},
		{name: "duplicate", value: []byte(strings.Replace(valid, `"seed":17`, `"seed":17,"seed":17`, 1))},
		{name: "trailing", value: append([]byte(valid), []byte(` {}`)...)},
		{name: "invalid numeric", value: []byte(strings.Replace(valid, `"seed":17`, `"seed":1.5`, 1))},
		{name: "invalid string", value: []byte(strings.Replace(valid, fmt.Sprintf(`"schema":%q`, recordSchema), `"schema":1`, 1))},
		{name: "changed raw plan", value: []byte(strings.Replace(valid, string(expectedPlan), `{"plan": "exact"}`, 1))},
		{name: "malformed decision", value: []byte(strings.Replace(valid, `"outcome":"completed",`, `"outcome":"completed","exploration_decisions":[{"dimension":1}],`, 1))},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := projectRetainedRecord(test.value, expectedPlan); err == nil {
				t.Fatal("projectRetainedRecord() accepted malicious owned input")
			}
		})
	}
}

func TestProjectRetainedRecordKeepsProducerFieldsInOutcomeIdentity(t *testing.T) {
	expectedPlan := []byte(`{"plan":"exact"}`)
	recordForValue := func(value string) []byte {
		return []byte(fmt.Sprintf(
			`{"schema":%q,"seed":17,"spec_sha256":%q,"outcome":"completed","exploration_plan":%s,"producer_owned":%q,"identity":%q}`,
			recordSchema, record.HashBytes([]byte("spec")), expectedPlan, value, record.HashBytes([]byte("record")),
		))
	}
	first, err := projectRetainedRecord(recordForValue("first"), expectedPlan)
	if err != nil {
		t.Fatal(err)
	}
	second, err := projectRetainedRecord(recordForValue("second"), expectedPlan)
	if err != nil {
		t.Fatal(err)
	}
	if first.outcomeSHA256 == second.outcomeSHA256 {
		t.Fatal("producer-owned extra field did not contribute to the outcome identity")
	}
}

func TestResultForRecordRejectsEachChangedDecisionIdentityInput(t *testing.T) {
	config := testCombinedConfig()
	state, err := NewSimulation(config)
	if err != nil {
		t.Fatal(err)
	}
	round, ok := state.NextRound()
	if !ok {
		t.Fatal("simulation exploration root round is unavailable")
	}
	candidate := round.Candidates[0]
	plan, err := PlanForCandidate(config, candidate)
	if err != nil {
		t.Fatal(err)
	}
	decision := simulationTestDecision(t, DimensionScenario, 2, 3, 1)
	tests := map[string]func(*Decision){
		"dimension":                func(value *Decision) { value.Dimension = DimensionFault },
		"ordinal":                  func(value *Decision) { value.Ordinal++ },
		"site identity":            func(value *Decision) { value.SiteSHA256 = record.HashBytes([]byte("changed site")) },
		"alternative":              func(value *Decision) { value.Alternatives[0] = record.HashBytes([]byte("changed alternative")) },
		"alternative set identity": func(value *Decision) { value.AlternativeSetSHA256 = record.HashBytes([]byte("changed set")) },
		"selected rank":            func(value *Decision) { value.Selected = 0 },
		"decision identity":        func(value *Decision) { value.Identity = record.HashBytes([]byte("changed decision")) },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			changed := decision
			changed.Alternatives = append([]record.SHA256(nil), decision.Alternatives...)
			mutate(&changed)
			recordBytes, err := json.Marshal(struct {
				Schema               string          `json:"schema"`
				Seed                 uint64          `json:"seed"`
				SpecSHA256           record.SHA256   `json:"spec_sha256"`
				Outcome              string          `json:"outcome"`
				ExplorationPlan      json.RawMessage `json:"exploration_plan"`
				ExplorationDecisions []Decision      `json:"exploration_decisions"`
				ProducerOwned        string          `json:"producer_owned"`
				Identity             record.SHA256   `json:"identity"`
			}{
				Schema: recordSchema, Seed: config.BaseSeed, SpecSHA256: record.HashBytes([]byte("spec")), Outcome: "completed",
				ExplorationPlan: plan, ExplorationDecisions: []Decision{changed}, ProducerOwned: "retained", Identity: record.HashBytes([]byte("record")),
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ResultForRecord(config, candidate, recordBytes, nil); err == nil {
				t.Fatal("ResultForRecord() accepted a changed decision identity input")
			}
		})
	}

	runtimeDecision := simulationTestDecision(t, DimensionRuntime, 0, 2, 0)
	runtimeDecision.AlternativeControls[1][0] ^= 0xff
	if err := validateRecordDecision(runtimeDecision); err == nil {
		t.Fatal("validateRecordDecision() accepted changed runtime control bytes")
	}
}

func TestProjectArtifactBindsExactReplayAndNormalizedFailure(t *testing.T) {
	config := SimulationConfig{
		ExecutionSHA256: record.HashBytes([]byte("execution")), ControllerSHA256: SimulationImplementationSHA256(), BaseSeed: 17,
		Parallel: 1, MaxExecutions: 2, MaxForcedDecisions: 1, MaxExplorationBytes: 1 << 20, MaxResultBytes: 1 << 20, FailureBudget: 1,
		Limits: DimensionLimits{Runtime: 1, Scenario: 1, Network: 1, Storage: 1, Fault: 1, Crash: 1},
	}
	state, err := NewSimulation(config)
	if err != nil {
		t.Fatal(err)
	}
	round, ok := state.NextRound()
	if !ok {
		t.Fatal("simulation exploration root round is unavailable")
	}
	candidate := round.Candidates[0]
	plan, err := PlanForCandidate(config, candidate)
	if err != nil {
		t.Fatal(err)
	}
	failure := record.HashBytes([]byte("normalized oracle failure"))
	recordBytes, err := json.Marshal(struct {
		Schema          string          `json:"schema"`
		Seed            uint64          `json:"seed"`
		SpecSHA256      record.SHA256   `json:"spec_sha256"`
		Outcome         string          `json:"outcome"`
		FailureIdentity record.SHA256   `json:"failure_identity"`
		ExplorationPlan json.RawMessage `json:"exploration_plan"`
		Identity        record.SHA256   `json:"identity"`
	}{
		Schema: "gomad3.cluster-record/v8", Seed: config.BaseSeed, SpecSHA256: record.HashBytes([]byte("spec")),
		Outcome: "oracle_failed", FailureIdentity: failure, ExplorationPlan: plan, Identity: record.HashBytes([]byte("record")),
	})
	if err != nil {
		t.Fatal(err)
	}

	profile, err := ProjectArtifact(config, candidate, plan, recordBytes, nil, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if profile.CandidateSHA256 != candidate.SHA256 || profile.ControllerSHA256 != config.ControllerSHA256 || profile.ExecutionSHA256 != config.ExecutionSHA256 || profile.FailureSHA256 != failure || profile.Plan.SHA256 != record.HashBytes(plan) || profile.Record.SHA256 != record.HashBytes(recordBytes) {
		t.Fatalf("simulation profile = %#v", profile)
	}
	if err := ValidateArtifact(profile, plan, recordBytes); err != nil {
		t.Fatal(err)
	}

	changedConfig := config
	changedConfig.BaseSeed = 18
	changedState, err := NewSimulation(changedConfig)
	if err != nil {
		t.Fatal(err)
	}
	changedRound, ok := changedState.NextRound()
	if !ok {
		t.Fatal("changed simulation exploration root round is unavailable")
	}
	changedPlan, err := PlanForCandidate(changedConfig, changedRound.Candidates[0])
	if err != nil {
		t.Fatal(err)
	}
	profile.CandidateSHA256 = changedRound.Candidates[0].SHA256
	profile.Plan.SHA256 = record.HashBytes(changedPlan)
	profile.Plan.Bytes = record.Uint64String(len(changedPlan))
	if err := ValidateArtifact(profile, changedPlan, recordBytes); err == nil {
		t.Fatal("ValidateArtifact() accepted a record bound to a different exploration plan")
	}
}

func TestProjectArtifactRequiresEveryForcedDecisionRecord(t *testing.T) {
	config := testCombinedConfig()
	decision, err := CanonicalDecision(
		DimensionFault, 0, record.HashBytes([]byte("fault site")),
		[]record.SHA256{record.HashBytes([]byte("none")), record.HashBytes([]byte("drop"))}, 0,
	)
	if err != nil {
		t.Fatal(err)
	}
	forced, err := ForceDecision(decision, 1)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := CanonicalCandidate(config, []ForcedDecision{forced}, "")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanForCandidate(config, candidate)
	if err != nil {
		t.Fatal(err)
	}
	record, err := json.Marshal(struct {
		Schema          string          `json:"schema"`
		Seed            uint64          `json:"seed"`
		SpecSHA256      record.SHA256   `json:"spec_sha256"`
		Outcome         string          `json:"outcome"`
		FailureIdentity record.SHA256   `json:"failure_identity"`
		ExplorationPlan json.RawMessage `json:"exploration_plan"`
		Identity        record.SHA256   `json:"identity"`
	}{
		Schema: "gomad3.cluster-record/v8", Seed: config.BaseSeed, SpecSHA256: record.HashBytes([]byte("spec")),
		Outcome: "oracle_failed", FailureIdentity: record.HashBytes([]byte("failure")), ExplorationPlan: plan,
		Identity: record.HashBytes([]byte("record")),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ProjectArtifact(config, candidate, plan, record, nil, 1<<20); err == nil {
		t.Fatal("ProjectArtifact() accepted a record that omitted its forced fault decision")
	}
}

func testCombinedConfig() SimulationConfig {
	return SimulationConfig{
		ExecutionSHA256: record.HashBytes([]byte("execution")), ControllerSHA256: SimulationImplementationSHA256(), BaseSeed: 17,
		Parallel: 1, MaxExecutions: 8, MaxForcedDecisions: 8, MaxExplorationBytes: 1 << 20, MaxResultBytes: 1 << 20, FailureBudget: 1,
		Limits: DimensionLimits{Runtime: 8, Scenario: 8, Network: 8, Storage: 8, Fault: 8, Crash: 8},
	}
}

func testRuntimeDecision(t *testing.T, ordinal uint64, selected uint32) choice.Decision {
	t.Helper()
	first := sha256.Sum256([]byte("first runnable"))
	second := sha256.Sum256([]byte("second runnable"))
	decision, err := choice.CanonicalDecision(ordinal, choice.KindRunnable, 24, false, [][sha256.Size]byte{first, second}, [][sha256.Size]byte{first, second}[selected], 0)
	if err != nil {
		t.Fatal(err)
	}
	return decision
}
