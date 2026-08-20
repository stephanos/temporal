//nolint:revive // The package name is the public Umpire3 runtime.Run seam.
package runtime

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/environment"
	umpire3fault "go.temporal.io/server/tests/umpire3/fault"
)

func TestCompareReplaySeparatesDriftClasses(t *testing.T) {
	baseline := Result{
		ExperimentDigest: "sha256:one",
		Actions:          []ActionResult{{Identifier: "a1", Kind: "schedule-operation"}},
		Observations: []environment.Observation{{
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
	require.Equal(t, []Drift{{Kind: DriftSemantic, Detail: "experiment digest changed"}}, CompareReplay(baseline, semantic))

	realization := baseline
	realization.Actions = []ActionResult{{Identifier: "a1", Kind: "different"}}
	require.Equal(t, DriftRealization, CompareReplay(baseline, realization)[0].Kind)

	schedule := baseline
	schedule.Observations = append([]environment.Observation(nil), baseline.Observations...)
	schedule.Observations[0].SourceSequence = 2
	require.Equal(t, DriftSchedule, CompareReplay(baseline, schedule)[0].Kind)

	observation := baseline
	observation.Observations = append([]environment.Observation(nil), baseline.Observations...)
	observation.Observations[0].Satisfied = false
	require.Equal(t, DriftObservation, CompareReplay(baseline, observation)[0].Kind)

	evidence := baseline
	evidence.Observations = append([]environment.Observation(nil), baseline.Observations...)
	evidence.Observations[0].CausalReference = ""
	require.Equal(t, DriftEvidence, CompareReplay(baseline, evidence)[0].Kind)
}

func TestCompareReplayDetectsLearnedFootprintDrift(t *testing.T) {
	previous := Result{ExperimentDigest: "sha256:same", Footprint: &umpire3fault.Report{
		FootprintDigest: "sha256:before", ReconciliationDigest: "sha256:reconciled",
	}}
	current := Result{ExperimentDigest: "sha256:same", Footprint: &umpire3fault.Report{
		FootprintDigest: "sha256:after", ReconciliationDigest: "sha256:reconciled",
	}}

	require.Equal(t, []Drift{{Kind: DriftFootprint, Detail: "learned runtime footprint changed"}},
		CompareReplay(previous, current))
}
