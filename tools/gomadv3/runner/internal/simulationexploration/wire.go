package simulationexploration

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"

	"go.temporal.io/server/tools/gomadv3/choice"
	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/runner/internal/combinedfrontier"
)

const planSchema = "gomadv3.simulation-exploration-plan/v1"
const maximumPlanBytes = 16 << 20

type plan struct {
	Schema           string          `json:"schema"`
	ExecutionSHA256  evidence.SHA256 `json:"execution_sha256"`
	ControllerSHA256 evidence.SHA256 `json:"controller_sha256"`
	BaseSeed         uint64          `json:"base_seed"`
	Overrides        []rootOverride  `json:"overrides"`
	CandidateSHA256  evidence.SHA256 `json:"candidate_sha256"`
}

type rootOverride struct {
	Dimension            combinedfrontier.Dimension `json:"dimension"`
	Ordinal              uint64                     `json:"ordinal"`
	SiteSHA256           evidence.SHA256            `json:"site_sha256"`
	Alternatives         uint32                     `json:"alternatives"`
	AlternativeSetSHA256 evidence.SHA256            `json:"alternative_set_sha256"`
	Selected             uint32                     `json:"selected"`
	SelectedSHA256       evidence.SHA256            `json:"selected_sha256"`
	Identity             evidence.SHA256            `json:"identity"`
}

type CandidateExecution struct {
	SimulationPlan   []byte
	ChoiceMode       choice.Mode
	ChoiceReplayPlan *choice.ReplayPlan
}

func PlanForCandidate(config combinedfrontier.Config, candidate combinedfrontier.Candidate) ([]byte, error) {
	if err := combinedfrontier.ValidateCandidate(config, candidate); err != nil {
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

func ExecutionForCandidate(config combinedfrontier.Config, candidate combinedfrontier.Candidate, identity choice.ExecutionIdentity) (CandidateExecution, error) {
	planBytes, err := PlanForCandidate(config, candidate)
	if err != nil {
		return CandidateExecution{}, err
	}
	execution := CandidateExecution{SimulationPlan: planBytes, ChoiceMode: choice.ModeRecord}
	controls := make([]choice.ReplayPlan, 0, len(candidate.Overrides))
	forced := make([]combinedfrontier.ForcedDecision, 0, len(candidate.Overrides))
	for _, override := range candidate.Overrides {
		if override.Dimension != combinedfrontier.DimensionRuntime {
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

func runtimeOverrideMatches(override combinedfrontier.ForcedDecision, decision choice.Decision) bool {
	site, alternatives, err := runtimeDecisionIdentities(decision)
	return err == nil && override.Ordinal == decision.Ordinal && override.SiteSHA256 == site && override.Alternatives == uint32(len(alternatives)) &&
		override.AlternativeSetSHA256 == combinedAlternativeSetIdentity(decision.Ordinal, site, alternatives) && override.Selected == decision.Selected && override.SelectedSHA256 == alternatives[decision.Selected]
}

func combinedAlternativeSetIdentity(ordinal uint64, site evidence.SHA256, alternatives []evidence.SHA256) evidence.SHA256 {
	encoded, err := evidence.CanonicalJSON(struct {
		Dimension    combinedfrontier.Dimension `json:"dimension"`
		Ordinal      uint64                     `json:"ordinal"`
		SiteSHA256   evidence.SHA256            `json:"site_sha256"`
		Alternatives []evidence.SHA256          `json:"alternatives"`
	}{combinedfrontier.DimensionRuntime, ordinal, site, append([]evidence.SHA256(nil), alternatives...)})
	if err != nil {
		return ""
	}
	return evidence.DomainHash("gomadv3-combined-frontier-alternative-set/v1", encoded)
}

func rootOverrides(config combinedfrontier.Config, candidate combinedfrontier.Candidate) ([]rootOverride, evidence.SHA256, error) {
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
		encoded, err := evidence.CanonicalJSON(map[string]any{
			"alternative_set_sha256": root.AlternativeSetSHA256, "alternatives": root.Alternatives,
			"dimension": root.Dimension, "identity": "", "ordinal": root.Ordinal, "selected": root.Selected,
			"selected_sha256": root.SelectedSHA256, "site_sha256": root.SiteSHA256,
		})
		if err != nil {
			return nil, "", err
		}
		root.Identity = evidence.DomainHash("gomadv3-combined-frontier-forced-decision/v1", encoded)
		overrides[index] = root
		identityOverrides[index] = map[string]any{
			"alternative_set_sha256": root.AlternativeSetSHA256, "alternatives": root.Alternatives,
			"dimension": root.Dimension, "identity": root.Identity, "ordinal": root.Ordinal,
			"selected": root.Selected, "selected_sha256": root.SelectedSHA256, "site_sha256": root.SiteSHA256,
		}
	}
	encoded, err := evidence.CanonicalJSON(map[string]any{
		"base_seed": config.BaseSeed, "controller_sha256": config.ControllerSHA256,
		"execution_sha256": config.ExecutionSHA256, "overrides": identityOverrides,
	})
	if err != nil {
		return nil, "", err
	}
	return overrides, evidence.DomainHash("gomadv3-combined-frontier-candidate/v1", encoded), nil
}
