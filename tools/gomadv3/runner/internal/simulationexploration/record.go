package simulationexploration

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/runner/internal/combinedfrontier"
)

const recordSchema = "gomadv3.cluster-record/v7"
const maximumRecordBytes = 128 << 20

func ResultForRecord(config combinedfrontier.Config, candidate combinedfrontier.Candidate, record []byte, runtimeDecisions []combinedfrontier.Decision) (combinedfrontier.Result, error) {
	if len(record) == 0 || len(record) > maximumRecordBytes {
		return combinedfrontier.Result{}, errors.New("simulation exploration record size is invalid")
	}
	if err := combinedfrontier.ValidateCandidate(config, candidate); err != nil {
		return combinedfrontier.Result{}, err
	}
	var fields map[string]json.RawMessage
	if err := evidence.StrictDecode(record, &fields); err != nil {
		return combinedfrontier.Result{}, fmt.Errorf("decode simulation exploration record: %w", err)
	}
	schema, err := requiredString(fields, "schema")
	if err != nil || schema != recordSchema {
		return combinedfrontier.Result{}, errors.Join(fmt.Errorf("simulation exploration record schema = %q, want %q", schema, recordSchema), err)
	}
	seed, err := requiredUint64(fields, "seed")
	if err != nil || seed != config.BaseSeed {
		return combinedfrontier.Result{}, errors.Join(errors.New("simulation exploration record seed does not match its candidate"), err)
	}
	identity, err := requiredSHA256(fields, "identity")
	if err != nil {
		return combinedfrontier.Result{}, fmt.Errorf("simulation exploration record identity: %w", err)
	}
	_ = identity
	expectedPlan, err := PlanForCandidate(config, candidate)
	if err != nil {
		return combinedfrontier.Result{}, err
	}
	actualPlan, ok := fields["exploration_plan"]
	if !ok || !bytes.Equal(actualPlan, expectedPlan) {
		return combinedfrontier.Result{}, errors.New("simulation exploration record plan does not match its candidate")
	}
	var recorded []combinedfrontier.Decision
	if raw, ok := fields["exploration_decisions"]; ok {
		if err := json.Unmarshal(raw, &recorded); err != nil {
			return combinedfrontier.Result{}, fmt.Errorf("decode simulation exploration decisions: %w", err)
		}
	}
	decisions := append([]combinedfrontier.Decision(nil), runtimeDecisions...)
	decisions = append(decisions, recorded...)
	seen := make(map[string]struct{}, len(decisions))
	for index, decision := range decisions {
		if err := validateDecision(decision); err != nil {
			return combinedfrontier.Result{}, fmt.Errorf("simulation exploration decision %d: %w", index, err)
		}
		key := fmt.Sprintf("%s\x00%d", decision.Dimension, decision.Ordinal)
		if _, ok := seen[key]; ok {
			return combinedfrontier.Result{}, errors.New("simulation exploration decisions contain a duplicate dimension ordinal")
		}
		seen[key] = struct{}{}
	}
	outcome, err := requiredString(fields, "outcome")
	if err != nil {
		return combinedfrontier.Result{}, err
	}
	failed, diverged := false, false
	switch outcome {
	case "completed":
	case "scenario_failed", "oracle_failed":
		failed = true
	case "replay_diverged":
		diverged = true
	default:
		return combinedfrontier.Result{}, fmt.Errorf("simulation exploration outcome %q is invalid", outcome)
	}
	semantic := make(map[string]json.RawMessage, len(fields))
	for name, value := range fields {
		semantic[name] = append(json.RawMessage(nil), value...)
	}
	for _, name := range []string{"identity", "spec_sha256", "exploration_plan", "exploration_decisions"} {
		delete(semantic, name)
	}
	encodedSemantic, err := evidence.CanonicalJSON(semantic)
	if err != nil {
		return combinedfrontier.Result{}, fmt.Errorf("encode simulation exploration outcome: %w", err)
	}
	result := combinedfrontier.Result{
		CandidateSHA256: candidate.SHA256,
		OutcomeSHA256:   evidence.DomainHash("gomadv3-simulation-exploration-outcome/v1", encodedSemantic),
		Failed:          failed,
		Diverged:        diverged,
		Decisions:       decisions,
	}
	if failed {
		failure, err := requiredSHA256(fields, "failure_identity")
		if err != nil {
			return combinedfrontier.Result{}, fmt.Errorf("simulation exploration failure identity: %w", err)
		}
		result.FailureSHA256 = failure
	}
	return result, nil
}

func validateDecision(decision combinedfrontier.Decision) error {
	canonical, err := combinedfrontier.CanonicalDecision(decision.Dimension, decision.Ordinal, decision.SiteSHA256, decision.Alternatives, decision.Selected)
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

func requiredSHA256(fields map[string]json.RawMessage, name string) (evidence.SHA256, error) {
	value, err := requiredString(fields, name)
	if err != nil {
		return "", err
	}
	return evidence.ParseSHA256(value)
}
