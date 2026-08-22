package replay

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/execution"
	umpire3fault "go.temporal.io/server/tests/umpire3/execution/fault"
)

func TestCompareReplaySeparatesDriftClasses(t *testing.T) {
	baseline := execution.Result{
		ExperimentDigest: "sha256:one",
		Actions:          []execution.ActionResult{{Identifier: "a1", Kind: "schedule-operation"}},
		Observations: []execution.Observation{{
			CheckpointID:    "checkpoint",
			Kind:            "created",
			Satisfied:       true,
			Source:          "history",
			SourceSequence:  1,
			CausalReference: "causal",
		}},
	}

	semantic := baseline
	semantic.ExperimentDigest = "sha256:two"
	require.Equal(t, []Drift{{Kind: DriftSemantic, Detail: "experiment digest changed"}}, Compare(baseline, semantic))

	realization := baseline
	realization.Actions = []execution.ActionResult{{Identifier: "a1", Kind: "different"}}
	require.Equal(t, DriftRealization, Compare(baseline, realization)[0].Kind)

	schedule := baseline
	schedule.Observations = append([]execution.Observation(nil), baseline.Observations...)
	schedule.Observations[0].SourceSequence = 2
	require.Equal(t, DriftSchedule, Compare(baseline, schedule)[0].Kind)

	observation := baseline
	observation.Observations = append([]execution.Observation(nil), baseline.Observations...)
	observation.Observations[0].Satisfied = false
	require.Equal(t, DriftObservation, Compare(baseline, observation)[0].Kind)

	evidence := baseline
	evidence.Observations = append([]execution.Observation(nil), baseline.Observations...)
	evidence.Observations[0].CausalReference = ""
	require.Equal(t, DriftEvidence, Compare(baseline, evidence)[0].Kind)
}

func TestCompareReplayDetectsLearnedFootprintDrift(t *testing.T) {
	previous := execution.Result{ExperimentDigest: "sha256:same", Footprint: &umpire3fault.Report{
		FootprintDigest: "sha256:before", ReconciliationDigest: "sha256:reconciled",
	}}
	current := execution.Result{ExperimentDigest: "sha256:same", Footprint: &umpire3fault.Report{
		FootprintDigest: "sha256:after", ReconciliationDigest: "sha256:reconciled",
	}}

	require.Equal(t, []Drift{{Kind: DriftFootprint, Detail: "learned runtime footprint changed"}},
		Compare(previous, current))
}
