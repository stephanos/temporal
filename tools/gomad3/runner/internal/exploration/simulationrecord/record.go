package simulationrecord

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/record"
	simulationengine "go.temporal.io/server/tools/gomad3/runner/internal/exploration/simulation"
)

const recordSchema = "gomad3.cluster-record/v7"
const maximumRecordBytes = 128 << 20

type retainedRecord struct {
	seed          uint64
	failed        bool
	diverged      bool
	outcomeSHA256 record.SHA256
	failureSHA256 record.SHA256
	decisions     []simulationengine.Decision
}

func RuntimeDecisions(tape choice.ReplayPlan) ([]simulationengine.Decision, error) {
	validated, err := choice.ValidateReplayPlan(tape, tape.Identity)
	if err != nil {
		return nil, fmt.Errorf("validate runtime choice tape: %w", err)
	}
	decisions := make([]simulationengine.Decision, 0, len(validated.Decisions))
	for _, runtimeDecision := range validated.Decisions {
		if runtimeDecision.Alternatives < 2 {
			continue
		}
		site, alternatives, err := runtimeDecisionIdentities(runtimeDecision)
		if err != nil {
			return nil, err
		}
		controls := make([][]byte, runtimeDecision.Alternatives)
		for rank := range controls {
			if uint32(rank) == runtimeDecision.Selected {
				continue
			}
			prefix, err := choice.BuildRankPrefix(validated, runtimeDecision.Ordinal, uint32(rank))
			if err != nil {
				return nil, fmt.Errorf("build runtime choice control %d rank %d: %w", runtimeDecision.Ordinal, rank, err)
			}
			controls[rank] = append([]byte(nil), prefix.Bytes...)
		}
		decision, err := simulationengine.CanonicalControlledDecision(
			simulationengine.DimensionRuntime, runtimeDecision.Ordinal, site, alternatives, controls, runtimeDecision.Selected,
		)
		if err != nil {
			return nil, fmt.Errorf("project runtime choice decision %d: %w", runtimeDecision.Ordinal, err)
		}
		decisions = append(decisions, decision)
	}
	return decisions, nil
}

func runtimeDecisionIdentities(runtimeDecision choice.Decision) (record.SHA256, []record.SHA256, error) {
	siteProjection := struct {
		Kind                 uint8               `json:"kind"`
		SiteOffset           record.Uint64String `json:"site_offset"`
		SiteMissing          bool                `json:"site_missing"`
		Data                 uint32              `json:"data"`
		Alternatives         uint32              `json:"alternatives"`
		AlternativeSetSHA256 record.SHA256       `json:"alternative_set_sha256"`
	}{
		Kind: uint8(runtimeDecision.Kind), SiteOffset: record.Uint64String(runtimeDecision.SiteOffset),
		SiteMissing: runtimeDecision.SiteMissing, Data: runtimeDecision.Data, Alternatives: runtimeDecision.Alternatives,
		AlternativeSetSHA256: record.SHA256FromSum(runtimeDecision.AlternativeSetDigest),
	}
	encodedSite, err := canonicaljson.CanonicalJSON(siteProjection)
	if err != nil {
		return "", nil, err
	}
	site := record.DomainHash("gomad3-runtime-choice-site/v1", encodedSite)
	alternatives := make([]record.SHA256, runtimeDecision.Alternatives)
	for rank := range alternatives {
		encodedAlternative, err := canonicaljson.CanonicalJSON(struct {
			SiteSHA256 record.SHA256 `json:"site_sha256"`
			Rank       uint32        `json:"rank"`
		}{SiteSHA256: site, Rank: uint32(rank)})
		if err != nil {
			return "", nil, err
		}
		alternatives[rank] = record.DomainHash("gomad3-runtime-choice-alternative/v1", encodedAlternative)
	}
	return site, alternatives, nil
}

func ResultForRecord(config simulationengine.Config, candidate simulationengine.Candidate, recordBytes []byte, runtimeDecisions []simulationengine.Decision) (simulationengine.Result, error) {
	if err := simulationengine.ValidateCandidate(config, candidate); err != nil {
		return simulationengine.Result{}, err
	}
	expectedPlan, err := PlanForCandidate(config, candidate)
	if err != nil {
		return simulationengine.Result{}, err
	}
	projected, err := projectRetainedRecord(recordBytes, expectedPlan)
	if err != nil {
		return simulationengine.Result{}, err
	}
	if projected.seed != config.BaseSeed {
		return simulationengine.Result{}, errors.Join(errors.New("simulation exploration record seed does not match its candidate"), err)
	}
	decisions := append([]simulationengine.Decision(nil), runtimeDecisions...)
	decisions = append(decisions, projected.decisions...)
	seen := make(map[string]struct{}, len(decisions))
	for index, decision := range decisions {
		if err := validateDecision(decision); err != nil {
			return simulationengine.Result{}, fmt.Errorf("simulation exploration decision %d: %w", index, err)
		}
		key := fmt.Sprintf("%s\x00%d", decision.Dimension, decision.Ordinal)
		if _, ok := seen[key]; ok {
			return simulationengine.Result{}, errors.New("simulation exploration decisions contain a duplicate dimension ordinal")
		}
		seen[key] = struct{}{}
	}
	result := simulationengine.Result{
		CandidateSHA256: candidate.SHA256, OutcomeSHA256: projected.outcomeSHA256,
		Failed: projected.failed, FailureSHA256: projected.failureSHA256, Diverged: projected.diverged, Decisions: decisions,
	}
	if err := simulationengine.ValidateResult(config, candidate, result); err != nil {
		return simulationengine.Result{}, err
	}
	return result, nil
}

func ProjectArtifact(config simulationengine.Config, candidate simulationengine.Candidate, planBytes, recordBytes []byte, runtimeDecisions []simulationengine.Decision, recordLimit uint64) (record.SimulationProfile, error) {
	expectedPlan, err := PlanForCandidate(config, candidate)
	if err != nil {
		return record.SimulationProfile{}, err
	}
	if !bytes.Equal(planBytes, expectedPlan) {
		return record.SimulationProfile{}, errors.New("simulation exploration artifact plan does not match its candidate")
	}
	result, err := ResultForRecord(config, candidate, recordBytes, runtimeDecisions)
	if err != nil {
		return record.SimulationProfile{}, err
	}
	profile := record.SimulationProfile{
		Name: "gomad3-simulation-exploration/v1", ControllerSHA256: config.ControllerSHA256,
		ExecutionSHA256: config.ExecutionSHA256, CandidateSHA256: candidate.SHA256,
		OutcomeSHA256: result.OutcomeSHA256, FailureSHA256: result.FailureSHA256,
		Plan: record.SimulationPlan{
			Schema: planSchema, File: "simulation/plan.json", SHA256: record.HashBytes(planBytes), Bytes: record.Uint64String(len(planBytes)),
		},
		Record: record.SimulationRecord{
			Schema: recordSchema, File: "simulation/record.json", SHA256: record.HashBytes(recordBytes),
			Bytes: record.Uint64String(len(recordBytes)), Limit: record.Uint64String(recordLimit),
		},
	}
	if err := ValidateArtifact(profile, planBytes, recordBytes); err != nil {
		return record.SimulationProfile{}, err
	}
	return profile, nil
}

func ValidateArtifact(profile record.SimulationProfile, planBytes, recordBytes []byte) error {
	if profile.Name != "gomad3-simulation-exploration/v1" || profile.Plan.Schema != planSchema || profile.Plan.File != "simulation/plan.json" || profile.Record.Schema != recordSchema || profile.Record.File != "simulation/record.json" {
		return errors.New("simulation exploration artifact identity is invalid")
	}
	for _, identity := range []record.SHA256{profile.ControllerSHA256, profile.ExecutionSHA256, profile.CandidateSHA256, profile.OutcomeSHA256, profile.Plan.SHA256, profile.Record.SHA256} {
		if _, err := identity.Bytes(); err != nil {
			return err
		}
	}
	if profile.FailureSHA256 != "" {
		if _, err := profile.FailureSHA256.Bytes(); err != nil {
			return err
		}
	}
	if len(planBytes) == 0 || len(planBytes) > maximumPlanBytes || record.HashBytes(planBytes) != profile.Plan.SHA256 || uint64(len(planBytes)) != uint64(profile.Plan.Bytes) {
		return errors.New("simulation exploration artifact plan identity changed")
	}
	if len(recordBytes) == 0 || len(recordBytes) > maximumRecordBytes || uint64(profile.Record.Limit) == 0 || uint64(profile.Record.Limit) > maximumRecordBytes || len(recordBytes) > int(profile.Record.Limit) || record.HashBytes(recordBytes) != profile.Record.SHA256 || uint64(len(recordBytes)) != uint64(profile.Record.Bytes) {
		return errors.New("simulation exploration artifact record identity or bound changed")
	}
	retainedPlan, err := validateRetainedPlan(planBytes)
	if err != nil {
		return err
	}
	if retainedPlan.ControllerSHA256 != profile.ControllerSHA256 || retainedPlan.ExecutionSHA256 != profile.ExecutionSHA256 {
		return errors.New("simulation exploration artifact plan execution identity changed")
	}
	projected, err := projectRetainedRecord(recordBytes, planBytes)
	if err != nil {
		return err
	}
	if projected.seed != retainedPlan.BaseSeed || projected.outcomeSHA256 != profile.OutcomeSHA256 || projected.failureSHA256 != profile.FailureSHA256 {
		return errors.New("simulation exploration artifact result identity changed")
	}
	return nil
}

func projectRetainedRecord(recordBytes, expectedPlan []byte) (retainedRecord, error) {
	if len(recordBytes) == 0 || len(recordBytes) > maximumRecordBytes {
		return retainedRecord{}, errors.New("simulation exploration record size is invalid")
	}
	var fields map[string]json.RawMessage
	if err := canonicaljson.StrictDecode(recordBytes, &fields); err != nil {
		return retainedRecord{}, fmt.Errorf("decode simulation exploration record: %w", err)
	}
	schema, err := requiredString(fields, "schema")
	if err != nil || schema != recordSchema {
		return retainedRecord{}, errors.Join(fmt.Errorf("simulation exploration record schema = %q, want %q", schema, recordSchema), err)
	}
	seed, err := requiredUint64(fields, "seed")
	if err != nil {
		return retainedRecord{}, err
	}
	if _, err := requiredSHA256(fields, "identity"); err != nil {
		return retainedRecord{}, fmt.Errorf("simulation exploration record identity: %w", err)
	}
	actualPlan, ok := fields["exploration_plan"]
	if !ok || !bytes.Equal(actualPlan, expectedPlan) {
		return retainedRecord{}, errors.New("simulation exploration record plan does not match its candidate")
	}
	var recorded []simulationengine.Decision
	if raw, ok := fields["exploration_decisions"]; ok {
		if err := json.Unmarshal(raw, &recorded); err != nil {
			return retainedRecord{}, fmt.Errorf("decode simulation exploration decisions: %w", err)
		}
	}
	seen := make(map[string]struct{}, len(recorded))
	for index, decision := range recorded {
		if err := validateDecision(decision); err != nil {
			return retainedRecord{}, fmt.Errorf("simulation exploration decision %d: %w", index, err)
		}
		key := fmt.Sprintf("%s\x00%d", decision.Dimension, decision.Ordinal)
		if _, ok := seen[key]; ok {
			return retainedRecord{}, errors.New("simulation exploration decisions contain a duplicate dimension ordinal")
		}
		seen[key] = struct{}{}
	}
	outcome, err := requiredString(fields, "outcome")
	if err != nil {
		return retainedRecord{}, err
	}
	projected := retainedRecord{seed: seed, decisions: recorded}
	switch outcome {
	case "completed":
	case "scenario_failed", "oracle_failed":
		projected.failed = true
	case "replay_diverged":
		projected.diverged = true
	default:
		return retainedRecord{}, fmt.Errorf("simulation exploration outcome %q is invalid", outcome)
	}
	semantic := make(map[string]json.RawMessage, len(fields))
	for name, value := range fields {
		semantic[name] = append(json.RawMessage(nil), value...)
	}
	for _, name := range []string{"identity", "spec_sha256", "exploration_plan", "exploration_decisions"} {
		delete(semantic, name)
	}
	encodedSemantic, err := canonicaljson.CanonicalJSON(semantic)
	if err != nil {
		return retainedRecord{}, fmt.Errorf("encode simulation exploration outcome: %w", err)
	}
	projected.outcomeSHA256 = record.DomainHash("gomad3-simulation-exploration-outcome/v1", encodedSemantic)
	if projected.failed {
		projected.failureSHA256, err = requiredSHA256(fields, "failure_identity")
		if err != nil {
			return retainedRecord{}, fmt.Errorf("simulation exploration failure identity: %w", err)
		}
	}
	return projected, nil
}

func validateDecision(decision simulationengine.Decision) error {
	var canonical simulationengine.Decision
	var err error
	if decision.Dimension == simulationengine.DimensionRuntime {
		canonical, err = simulationengine.CanonicalControlledDecision(
			decision.Dimension, decision.Ordinal, decision.SiteSHA256, decision.Alternatives, decision.AlternativeControls, decision.Selected,
		)
	} else {
		canonical, err = simulationengine.CanonicalDecision(decision.Dimension, decision.Ordinal, decision.SiteSHA256, decision.Alternatives, decision.Selected)
	}
	if err != nil {
		return err
	}
	if canonical.Dimension != decision.Dimension || canonical.Ordinal != decision.Ordinal || canonical.SiteSHA256 != decision.SiteSHA256 || !slices.Equal(canonical.Alternatives, decision.Alternatives) || canonical.AlternativeSetSHA256 != decision.AlternativeSetSHA256 || canonical.Selected != decision.Selected || canonical.Identity != decision.Identity {
		return errors.New("simulation exploration decision identity does not match its contents")
	}
	return nil
}

func requiredString(fields map[string]json.RawMessage, name string) (string, error) {
	raw, ok := fields[name]
	if !ok {
		return "", fmt.Errorf("simulation exploration record field %q is missing", name)
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("decode simulation exploration record field %q: %w", name, err)
	}
	return value, nil
}

func requiredUint64(fields map[string]json.RawMessage, name string) (uint64, error) {
	raw, ok := fields[name]
	if !ok {
		return 0, fmt.Errorf("simulation exploration record field %q is missing", name)
	}
	var value uint64
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, fmt.Errorf("decode simulation exploration record field %q: %w", name, err)
	}
	return value, nil
}

func requiredSHA256(fields map[string]json.RawMessage, name string) (record.SHA256, error) {
	value, err := requiredString(fields, name)
	if err != nil {
		return "", err
	}
	return record.ParseSHA256(value)
}
