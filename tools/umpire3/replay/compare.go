package replay

import umpire3execution "go.temporal.io/server/tools/umpire3/execution"

type DriftKind string

const (
	DriftSemantic    DriftKind = "semantic"
	DriftRealization DriftKind = "realization"
	DriftSchedule    DriftKind = "schedule"
	DriftObservation DriftKind = "observation"
	DriftEvidence    DriftKind = "evidence"
	DriftFootprint   DriftKind = "footprint"
)

type Drift struct {
	Kind   DriftKind `json:"kind"`
	Detail string    `json:"detail"`
}

func Compare(previous, current umpire3execution.Result) []Drift {
	if previous.ExperimentDigest != current.ExperimentDigest {
		return []Drift{{Kind: DriftSemantic, Detail: "experiment digest changed"}}
	}
	var drift []Drift
	if len(previous.Actions) != len(current.Actions) {
		drift = append(drift, Drift{Kind: DriftRealization, Detail: "realized action count changed"})
	} else {
		for index := range previous.Actions {
			before := previous.Actions[index]
			after := current.Actions[index]
			if before.Identifier != after.Identifier || before.Kind != after.Kind || before.Error != after.Error {
				drift = append(drift, Drift{Kind: DriftRealization, Detail: "realized action changed: " + before.Identifier})
			}
		}
	}
	if (previous.Footprint == nil) != (current.Footprint == nil) {
		drift = append(drift, Drift{Kind: DriftFootprint, Detail: "learned runtime footprint availability changed"})
	} else if previous.Footprint != nil &&
		(previous.Footprint.FootprintDigest != current.Footprint.FootprintDigest ||
			previous.Footprint.ReconciliationDigest != current.Footprint.ReconciliationDigest) {
		drift = append(drift, Drift{Kind: DriftFootprint, Detail: "learned runtime footprint changed"})
	}
	if len(previous.Observations) != len(current.Observations) {
		drift = append(drift, Drift{Kind: DriftEvidence, Detail: "observation count changed"})
		return drift
	}
	for index := range previous.Observations {
		before := previous.Observations[index]
		after := current.Observations[index]
		if before.CheckpointID != after.CheckpointID || before.Kind != after.Kind || before.Satisfied != after.Satisfied {
			drift = append(drift, Drift{Kind: DriftObservation, Detail: "normalized observation changed: " + before.CheckpointID})
			continue
		}
		if before.SourceSequence != after.SourceSequence {
			drift = append(drift, Drift{Kind: DriftSchedule, Detail: "source sequence changed: " + before.CheckpointID})
			continue
		}
		if before.Source != after.Source || before.CausalReference != after.CausalReference {
			drift = append(drift, Drift{Kind: DriftEvidence, Detail: "observation evidence changed: " + before.CheckpointID})
		}
	}
	return drift
}
