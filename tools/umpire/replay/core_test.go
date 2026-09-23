package replay

import (
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"google.golang.org/protobuf/proto"
)

// The core is what the violated rules' supporting sequences name, and the Run's other instruction
// events lie outside it, named by instruction id; neither the Run nor the Verdict changes.
func TestEvidenceCoreNamesTheViolatedRulesSupportAndOmitsTheRest(t *testing.T) {
	prepare := preparer(t)
	_, run, _ := recordedRunOf(t, prepare, profileName, loadCorpusCase(t, "violated"))
	before := proto.CloneOf(run)
	core := EvidenceCore(run.GetVerdict())
	require.Equal(t, run.GetVerdict().GetRules()[0].GetSupportingEventSequences(), core)
	require.NotEmpty(t, core)
	for _, sequence := range core {
		require.Equal(t, testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED, run.GetEvents()[sequence-1].GetKind())
	}
	outside := OutsideCore(run, core)
	require.NotEmpty(t, outside, "the Run carries instruction events that support no violated rule")
	for _, event := range outside {
		require.NotContains(t, core, event.Sequence)
		require.NotEmpty(t, event.InstructionID)
	}
	require.Equal(t, testpilotspb.RUN_EVENT_KIND_INSTRUCTION_STARTED, outside[0].Kind, "the violating instruction's start supports nothing")
	require.True(t, proto.Equal(before, run), "the core reads the Run and rewrites nothing")

	// A satisfied Verdict has no core, and every instruction event is outside it.
	satisfied := &testpilotspb.Verdict{Status: testpilotspb.VERDICT_STATUS_SATISFIED, Rules: []*testpilotspb.RuleVerdict{{RuleId: "r", Status: testpilotspb.RULE_VERDICT_STATUS_SATISFIED, SupportingEventSequences: []int64{4}}}}
	require.Empty(t, EvidenceCore(satisfied))
	require.Len(t, OutsideCore(run, nil), len(outside)+len(core))
	// Two violated rules naming one event: the core names it once.
	twice := &testpilotspb.Verdict{Rules: []*testpilotspb.RuleVerdict{
		{RuleId: "a", Status: testpilotspb.RULE_VERDICT_STATUS_VIOLATED, SupportingEventSequences: []int64{4, 2}},
		{RuleId: "b", Status: testpilotspb.RULE_VERDICT_STATUS_VIOLATED, SupportingEventSequences: []int64{4}},
	}}
	require.Equal(t, []int64{2, 4}, EvidenceCore(twice))
}
