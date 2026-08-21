package simulationexploration

import (
	"encoding/json"
	"errors"
	"fmt"

	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/runner/internal/combinedfrontier"
)

const planSchema = "gomadv3.simulation-exploration-plan/v1"
const maximumPlanBytes = 16 << 20

type plan struct {
	Schema           string                            `json:"schema"`
	ExecutionSHA256  evidence.SHA256                   `json:"execution_sha256"`
	ControllerSHA256 evidence.SHA256                   `json:"controller_sha256"`
	BaseSeed         uint64                            `json:"base_seed"`
	Overrides        []combinedfrontier.ForcedDecision `json:"overrides"`
	CandidateSHA256  evidence.SHA256                   `json:"candidate_sha256"`
}

func PlanForCandidate(config combinedfrontier.Config, candidate combinedfrontier.Candidate) ([]byte, error) {
	if err := combinedfrontier.ValidateCandidate(config, candidate); err != nil {
		return nil, err
	}
	value := plan{
		Schema: planSchema, ExecutionSHA256: config.ExecutionSHA256, ControllerSHA256: config.ControllerSHA256,
		BaseSeed: config.BaseSeed, Overrides: append([]combinedfrontier.ForcedDecision(nil), candidate.Overrides...), CandidateSHA256: candidate.SHA256,
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
