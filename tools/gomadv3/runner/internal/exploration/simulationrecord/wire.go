package simulationrecord

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"

	"go.temporal.io/server/tools/gomadv3/choice"
	"go.temporal.io/server/tools/gomadv3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomadv3/record"
	simulationengine "go.temporal.io/server/tools/gomadv3/runner/internal/exploration/simulation"
)

const planSchema = "gomadv3.simulation-exploration-plan/v1"
const maximumPlanBytes = 16 << 20

type plan struct {
	Schema           string         `json:"schema"`
	ExecutionSHA256  record.SHA256  `json:"execution_sha256"`
	ControllerSHA256 record.SHA256  `json:"controller_sha256"`
	BaseSeed         uint64         `json:"base_seed"`
	Overrides        []rootOverride `json:"overrides"`
	CandidateSHA256  record.SHA256  `json:"candidate_sha256"`
}

type rootOverride struct {
	Dimension            simulationengine.Dimension `json:"dimension"`
	Ordinal              uint64                     `json:"ordinal"`
	SiteSHA256           record.SHA256              `json:"site_sha256"`
	Alternatives         uint32                     `json:"alternatives"`
	AlternativeSetSHA256 record.SHA256              `json:"alternative_set_sha256"`
	Selected             uint32                     `json:"selected"`
	SelectedSHA256       record.SHA256              `json:"selected_sha256"`
	Identity             record.SHA256              `json:"identity"`
}

type CandidateExecution struct {
	SimulationPlan   []byte
	ChoiceMode       choice.Mode
	ChoiceReplayPlan *choice.ReplayPlan
}

func CandidateForArtifact(profile record.SimulationProfile, planBytes []byte, exactTape *choice.ReplayPlan) (simulationengine.Config, simulationengine.Candidate, error) {
	retained, err := validateRetainedPlan(planBytes)
	if err != nil {
		return simulationengine.Config{}, simulationengine.Candidate{}, err
	}
	if retained.ControllerSHA256 != profile.ControllerSHA256 || retained.ExecutionSHA256 != profile.ExecutionSHA256 {
		return simulationengine.Config{}, simulationengine.Candidate{}, errors.New("simulation exploration artifact execution identity changed")
	}
	config := retainedCandidateConfig(retained)
	overrides := make([]simulationengine.ForcedDecision, len(retained.Overrides))
	for index, override := range retained.Overrides {
		forced := simulationengine.ForcedDecision{
			Dimension: override.Dimension, Ordinal: override.Ordinal, SiteSHA256: override.SiteSHA256,
			Alternatives: override.Alternatives, AlternativeSetSHA256: override.AlternativeSetSHA256,
			Selected: override.Selected, SelectedSHA256: override.SelectedSHA256,
		}
		if override.Dimension == simulationengine.DimensionRuntime {
			if exactTape == nil {
				return simulationengine.Config{}, simulationengine.Candidate{}, errors.New("runtime simulation override requires an exact choice tape")
			}
			prefix, prefixErr := choice.BuildForcedRankPrefix(*exactTape, override.Ordinal, override.Selected)
			if prefixErr != nil {
				return simulationengine.Config{}, simulationengine.Candidate{}, fmt.Errorf("reconstruct runtime control at ordinal %d: %w", override.Ordinal, prefixErr)
			}
			forced.Control = prefix.Bytes
		}
		forced, err = simulationengine.CanonicalForcedDecision(forced)
		if err != nil {
			return simulationengine.Config{}, simulationengine.Candidate{}, fmt.Errorf("reconstruct forced decision %d: %w", index, err)
		}
		overrides[index] = forced
	}
	candidate, err := simulationengine.CanonicalCandidate(config, overrides, "")
	if err != nil {
		return simulationengine.Config{}, simulationengine.Candidate{}, err
	}
	if candidate.SHA256 != profile.CandidateSHA256 {
		return simulationengine.Config{}, simulationengine.Candidate{}, errors.New("reconstructed simulation candidate identity changed")
	}
	return config, candidate, nil
}

func retainedCandidateConfig(retained plan) simulationengine.Config {
	limits := simulationengine.DimensionLimits{Runtime: 1, Scenario: 1, Network: 1, Storage: 1, Fault: 1, Crash: 1}
	for _, override := range retained.Overrides {
		limit := override.Ordinal + 1
		switch override.Dimension {
		case simulationengine.DimensionRuntime:
			limits.Runtime = max(limits.Runtime, limit)
		case simulationengine.DimensionScenario:
			limits.Scenario = max(limits.Scenario, limit)
		case simulationengine.DimensionNetwork:
			limits.Network = max(limits.Network, limit)
		case simulationengine.DimensionStorage:
			limits.Storage = max(limits.Storage, limit)
		case simulationengine.DimensionFault:
			limits.Fault = max(limits.Fault, limit)
		case simulationengine.DimensionCrash:
			limits.Crash = max(limits.Crash, limit)
		default:
			continue
		}
	}
	return simulationengine.Config{
		ExecutionSHA256: retained.ExecutionSHA256, ControllerSHA256: retained.ControllerSHA256, BaseSeed: retained.BaseSeed,
		Parallel: 1, MaxExecutions: 1, MaxForcedDecisions: max(1, uint64(len(retained.Overrides))),
		MaxExplorationBytes: ^uint64(0), MaxResultBytes: ^uint64(0), FailureBudget: 1, Limits: limits,
	}
}

func PlanForCandidate(config simulationengine.Config, candidate simulationengine.Candidate) ([]byte, error) {
	if err := simulationengine.ValidateCandidate(config, candidate); err != nil {
		return nil, err
	}
	overrides, rootCandidate, err := rootOverrides(config, candidate)
	if err != nil {
		return nil, err
	}
	value := plan{
		Schema: planSchema, ExecutionSHA256: config.ExecutionSHA256, ControllerSHA256: config.ControllerSHA256,
		BaseSeed: config.BaseSeed, Overrides: overrides, CandidateSHA256: rootCandidate,
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode simulation exploration plan: %w", err)
	}
	if len(encoded) == 0 || len(encoded) > maximumPlanBytes {
		return nil, errors.New("simulation exploration plan exceeds its bound")
	}
	return encoded, nil
}

func validateRetainedPlan(encoded []byte) (plan, error) {
	if len(encoded) == 0 || len(encoded) > maximumPlanBytes {
		return plan{}, errors.New("simulation exploration plan size is invalid")
	}
	var decoded plan
	if err := canonicaljson.StrictDecode(encoded, &decoded); err != nil {
		return plan{}, fmt.Errorf("decode simulation exploration plan: %w", err)
	}
	reencoded, err := json.Marshal(decoded)
	if err != nil {
		return plan{}, fmt.Errorf("encode simulation exploration plan: %w", err)
	}
	if string(reencoded) != string(encoded) {
		return plan{}, errors.New("simulation exploration plan is not canonical")
	}
	if decoded.Schema != planSchema {
		return plan{}, fmt.Errorf("simulation exploration plan schema = %q, want %q", decoded.Schema, planSchema)
	}
	for _, identity := range []record.SHA256{decoded.ExecutionSHA256, decoded.ControllerSHA256, decoded.CandidateSHA256} {
		if _, err := identity.Bytes(); err != nil {
			return plan{}, err
		}
	}
	var identityOverrides []any
	if decoded.Overrides != nil {
		identityOverrides = make([]any, len(decoded.Overrides))
	}
	for index, override := range decoded.Overrides {
		if !validDimension(override.Dimension) || override.Alternatives < 2 || override.Selected >= override.Alternatives {
			return plan{}, fmt.Errorf("simulation exploration plan override %d shape is invalid", index)
		}
		for _, identity := range []record.SHA256{override.SiteSHA256, override.AlternativeSetSHA256, override.SelectedSHA256, override.Identity} {
			if _, err := identity.Bytes(); err != nil {
				return plan{}, fmt.Errorf("simulation exploration plan override %d: %w", index, err)
			}
		}
		encodedOverride, err := canonicaljson.CanonicalJSON(map[string]any{
			"alternative_set_sha256": override.AlternativeSetSHA256, "alternatives": override.Alternatives,
			"dimension": override.Dimension, "identity": "", "ordinal": override.Ordinal, "selected": override.Selected,
			"selected_sha256": override.SelectedSHA256, "site_sha256": override.SiteSHA256,
		})
		if err != nil {
			return plan{}, err
		}
		if record.DomainHash("gomadv3-simulation-exploration-forced-decision/v1", encodedOverride) != override.Identity {
			return plan{}, fmt.Errorf("simulation exploration plan override %d identity changed", index)
		}
		identityOverrides[index] = map[string]any{
			"alternative_set_sha256": override.AlternativeSetSHA256, "alternatives": override.Alternatives,
			"dimension": override.Dimension, "identity": override.Identity, "ordinal": override.Ordinal,
			"selected": override.Selected, "selected_sha256": override.SelectedSHA256, "site_sha256": override.SiteSHA256,
		}
	}
	candidateBytes, err := canonicaljson.CanonicalJSON(map[string]any{
		"base_seed": decoded.BaseSeed, "controller_sha256": decoded.ControllerSHA256,
		"execution_sha256": decoded.ExecutionSHA256, "overrides": identityOverrides,
	})
	if err != nil {
		return plan{}, err
	}
	if record.DomainHash("gomadv3-simulation-exploration-candidate/v1", candidateBytes) != decoded.CandidateSHA256 {
		return plan{}, errors.New("simulation exploration plan candidate identity changed")
	}
	return decoded, nil
}

func validDimension(dimension simulationengine.Dimension) bool {
	switch dimension {
	case simulationengine.DimensionRuntime, simulationengine.DimensionScenario, simulationengine.DimensionNetwork,
		simulationengine.DimensionStorage, simulationengine.DimensionFault, simulationengine.DimensionCrash:
		return true
	default:
		return false
	}
}

func ExecutionForCandidate(config simulationengine.Config, candidate simulationengine.Candidate, identity choice.ExecutionIdentity) (CandidateExecution, error) {
	planBytes, err := PlanForCandidate(config, candidate)
	if err != nil {
		return CandidateExecution{}, err
	}
	execution := CandidateExecution{SimulationPlan: planBytes, ChoiceMode: choice.ModeRecord}
	controls := make([]choice.ReplayPlan, 0, len(candidate.Overrides))
	forced := make([]simulationengine.ForcedDecision, 0, len(candidate.Overrides))
	for _, override := range candidate.Overrides {
		if override.Dimension != simulationengine.DimensionRuntime {
			continue
		}
		tape := choice.ReplayPlan{Identity: identity, Bytes: append([]byte(nil), override.Control...), SHA256: sha256.Sum256(override.Control)}
		validated, err := choice.ValidatePrefixReplayPlan(tape, identity)
		if err != nil {
			return CandidateExecution{}, fmt.Errorf("validate runtime control at ordinal %d: %w", override.Ordinal, err)
		}
		if override.Ordinal >= uint64(len(validated.Decisions)) || len(validated.Decisions) != int(override.Ordinal)+1 {
			return CandidateExecution{}, fmt.Errorf("runtime control at ordinal %d is not its exact rank prefix", override.Ordinal)
		}
		if !runtimeOverrideMatches(override, validated.Decisions[override.Ordinal]) {
			return CandidateExecution{}, fmt.Errorf("runtime control at ordinal %d does not match its forced decision", override.Ordinal)
		}
		controls = append(controls, validated)
		forced = append(forced, override)
	}
	if len(controls) == 0 {
		return execution, nil
	}
	longest := 0
	for index := 1; index < len(controls); index++ {
		if len(controls[index].Decisions) > len(controls[longest].Decisions) {
			longest = index
		}
	}
	selected := controls[longest]
	for _, override := range forced {
		if override.Ordinal >= uint64(len(selected.Decisions)) || !runtimeOverrideMatches(override, selected.Decisions[override.Ordinal]) {
			return CandidateExecution{}, errors.New("runtime controls do not compose into one forced prefix")
		}
	}
	execution.ChoiceMode = choice.ModePrefix
	execution.ChoiceReplayPlan = &selected
	return execution, nil
}

func runtimeOverrideMatches(override simulationengine.ForcedDecision, decision choice.Decision) bool {
	site, alternatives, err := runtimeDecisionIdentities(decision)
	return err == nil && override.Ordinal == decision.Ordinal && override.SiteSHA256 == site && override.Alternatives == uint32(len(alternatives)) &&
		override.AlternativeSetSHA256 == combinedAlternativeSetIdentity(decision.Ordinal, site, alternatives) && override.Selected == decision.Selected && override.SelectedSHA256 == alternatives[decision.Selected]
}

func combinedAlternativeSetIdentity(ordinal uint64, site record.SHA256, alternatives []record.SHA256) record.SHA256 {
	encoded, err := canonicaljson.CanonicalJSON(struct {
		Dimension    simulationengine.Dimension `json:"dimension"`
		Ordinal      uint64                     `json:"ordinal"`
		SiteSHA256   record.SHA256              `json:"site_sha256"`
		Alternatives []record.SHA256            `json:"alternatives"`
	}{simulationengine.DimensionRuntime, ordinal, site, append([]record.SHA256(nil), alternatives...)})
	if err != nil {
		return ""
	}
	return record.DomainHash("gomadv3-simulation-exploration-alternative-set/v1", encoded)
}

func rootOverrides(config simulationengine.Config, candidate simulationengine.Candidate) ([]rootOverride, record.SHA256, error) {
	var overrides []rootOverride
	var identityOverrides []any
	if candidate.Overrides != nil {
		overrides = make([]rootOverride, len(candidate.Overrides))
		identityOverrides = make([]any, len(candidate.Overrides))
	}
	for index, override := range candidate.Overrides {
		root := rootOverride{
			Dimension: override.Dimension, Ordinal: override.Ordinal, SiteSHA256: override.SiteSHA256,
			Alternatives: override.Alternatives, AlternativeSetSHA256: override.AlternativeSetSHA256,
			Selected: override.Selected, SelectedSHA256: override.SelectedSHA256,
		}
		encoded, err := canonicaljson.CanonicalJSON(map[string]any{
			"alternative_set_sha256": root.AlternativeSetSHA256, "alternatives": root.Alternatives,
			"dimension": root.Dimension, "identity": "", "ordinal": root.Ordinal, "selected": root.Selected,
			"selected_sha256": root.SelectedSHA256, "site_sha256": root.SiteSHA256,
		})
		if err != nil {
			return nil, "", err
		}
		root.Identity = record.DomainHash("gomadv3-simulation-exploration-forced-decision/v1", encoded)
		overrides[index] = root
		identityOverrides[index] = map[string]any{
			"alternative_set_sha256": root.AlternativeSetSHA256, "alternatives": root.Alternatives,
			"dimension": root.Dimension, "identity": root.Identity, "ordinal": root.Ordinal,
			"selected": root.Selected, "selected_sha256": root.SelectedSHA256, "site_sha256": root.SiteSHA256,
		}
	}
	encoded, err := canonicaljson.CanonicalJSON(map[string]any{
		"base_seed": config.BaseSeed, "controller_sha256": config.ControllerSHA256,
		"execution_sha256": config.ExecutionSHA256, "overrides": identityOverrides,
	})
	if err != nil {
		return nil, "", err
	}
	return overrides, record.DomainHash("gomadv3-simulation-exploration-candidate/v1", encoded), nil
}
