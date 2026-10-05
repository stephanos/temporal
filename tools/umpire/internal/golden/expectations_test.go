package golden

import (
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

// A baseline expected Run is read as its reader read it: an unwritten Contract is satisfied, a
// violated one stops the Run and any other completes it, cleanup succeeds, and each prose reason is
// the IR reason the delta lists for it. The manifest is rewritten line by line in GenerateCases's
// layout.
func TestDeclaredRunsReadTheArchiveAsItsReaderDid(t *testing.T) {
	runs := DeclaredRuns{Reasons: map[string]string{"they disagree": "REASON_EXPLANATIONS_DISAGREE", "all violate": "REASON_EVERY_EXPLANATION_VIOLATES"}}
	archived := map[string][]byte{
		OriginalIR + "m.json": []byte(`{"queries": [
			{"name": "plain", "expectedRun": {"property": "OUTCOME_INCONCLUSIVE", "reason": "they disagree", "conformance": "CONFORMANCE_CONFORMANT",
				"monitors": [{"name": "watch", "outcome": "OUTCOME_SATISFIED"}]}},
			{"name": "forged", "expectedRun": {"property": "OUTCOME_VIOLATED", "reason": "all violate", "conformance": "CONFORMANCE_INCONCLUSIVE", "contract": "OUTCOME_VIOLATED"}},
			{"name": "unexpected"}]}`),
		OriginalCases + "manifest.json": []byte(`{
  "queries": [
    {
      "expected": {
        "conformance": "conformant",
        "properties": [
          {
            "id": "plain",
            "status": "inconclusive",
            "reason": "they disagree"
          }
        ]
      }
    },
    {
      "expected": {
        "contract": "violated",
        "conformance": "inconclusive",
        "properties": [
          {
            "id": "forged",
            "status": "violated",
            "reason": "all violate"
          }
        ]
      }
    }
  ]
}
`),
		OriginalCases + "m-plain-case.json": []byte(`{"reason": "they disagree"}`),
	}
	declared, err := runs.Declare(archived)
	require.NoError(t, err)
	var m umpirespb.Model
	require.NoError(t, protojson.Unmarshal(declared[OriginalIR+"m.json"], &m))
	run := func(property, contract umpirespb.RunExpectation_Outcome, disposition umpirespb.RunExpectation_Disposition, reason umpirespb.RunExpectation_Reason,
		conformance umpirespb.RunExpectation_Conformance) *umpirespb.RunExpectation {
		return &umpirespb.RunExpectation{Property: property, Contract: contract, Disposition: disposition, Cleanup: umpirespb.RunExpectation_CLEANUP_SUCCEEDED,
			Reason: reason, Conformance: conformance}
	}
	plain := run(umpirespb.RunExpectation_OUTCOME_INCONCLUSIVE, umpirespb.RunExpectation_OUTCOME_SATISFIED, umpirespb.RunExpectation_DISPOSITION_COMPLETED,
		umpirespb.RunExpectation_REASON_EXPLANATIONS_DISAGREE, umpirespb.RunExpectation_CONFORMANCE_CONFORMANT)
	plain.Monitors = []*umpirespb.MonitorExpectation{{Name: "watch", Outcome: umpirespb.RunExpectation_OUTCOME_SATISFIED}}
	require.Equal(t, plain.String(), m.GetQueries()[0].GetExpectedRun().String())
	require.Equal(t, run(umpirespb.RunExpectation_OUTCOME_VIOLATED, umpirespb.RunExpectation_OUTCOME_VIOLATED, umpirespb.RunExpectation_DISPOSITION_STOPPED_BY_MONITOR,
		umpirespb.RunExpectation_REASON_EVERY_EXPLANATION_VIOLATES, umpirespb.RunExpectation_CONFORMANCE_INCONCLUSIVE).String(), m.GetQueries()[1].GetExpectedRun().String())
	require.Nil(t, m.GetQueries()[2].GetExpectedRun())
	require.Equal(t, `{
  "queries": [
    {
      "expected": {
        "contract": "satisfied",
        "disposition": "completed",
        "cleanup": "succeeded",
        "conformance": "conformant",
        "properties": [
          {
            "id": "plain",
            "status": "inconclusive",
            "reason": "explanations_disagree"
          }
        ]
      }
    },
    {
      "expected": {
        "contract": "violated",
        "disposition": "stopped_by_monitor",
        "cleanup": "succeeded",
        "conformance": "inconclusive",
        "properties": [
          {
            "id": "forged",
            "status": "violated",
            "reason": "every_explanation_violates"
          }
        ]
      }
    }
  ]
}
`, string(declared[OriginalCases+"manifest.json"]))
	require.Equal(t, archived[OriginalCases+"m-plain-case.json"], declared[OriginalCases+"m-plain-case.json"], "a Case is no expected Run")

	for name, test := range map[string]struct {
		runs  DeclaredRuns
		model string
	}{
		"an unlisted prose reason": {DeclaredRuns{Reasons: map[string]string{}},
			`{"queries": [{"expectedRun": {"property": "OUTCOME_INCONCLUSIVE", "reason": "they disagree"}}]}`},
		"a listed reason no Run writes": {runs, `{"queries": [{"expectedRun": {"property": "OUTCOME_INCONCLUSIVE", "reason": "they disagree"}}]}`},
		"an archived Run that declares its disposition": {DeclaredRuns{Reasons: map[string]string{"they disagree": "REASON_EXPLANATIONS_DISAGREE"}},
			`{"queries": [{"expectedRun": {"property": "OUTCOME_INCONCLUSIVE", "reason": "they disagree", "disposition": "DISPOSITION_COMPLETED"}}]}`},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := test.runs.Declare(map[string][]byte{OriginalIR + "m.json": []byte(test.model)})
			require.Error(t, err)
		})
	}
	require.Error(t, DeclaredRuns{Reasons: map[string]string{"prose": "REASON_SOMETHING"}}.check(), "a reason the IR does not have")
}
