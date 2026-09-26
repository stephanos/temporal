package testpilot

import (
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
)

func TestNexusImplementationSwitchSetsEachImplementationsSettings(t *testing.T) {
	values := NexusImplementationSwitch()
	require.Len(t, values, 2)
	require.Equal(t, "hsm", values[0].Name)
	require.Equal(t, "chasm", values[1].Name)
	// A setting's key is already the catalog's spelling, lower-case, which is what the Profile
	// records and what the realization's switch names.
	require.Equal(t, map[string]string{
		"history.enablechasm":                          "false",
		"history.enablechasmcallbacks":                 "false",
		"nexusoperation.enablechasmworkflowoperations": "false",
	}, values[0].Configuration())
	require.Equal(t, map[string]string{
		"history.enablechasm":                          "true",
		"history.enablechasmcallbacks":                 "true",
		"nexusoperation.enablechasmworkflowoperations": "true",
	}, values[1].Configuration())
}

// An injected divergence fails naming the switch, both values and both Verdicts, rule by rule.
func TestCheckSwitchAgreementNamesBothValuesAndVerdicts(t *testing.T) {
	satisfied := &testpilotspb.Verdict{Status: testpilotspb.VERDICT_STATUS_SATISFIED, Rules: []*testpilotspb.RuleVerdict{
		{RuleId: "completion", Status: testpilotspb.RULE_VERDICT_STATUS_SATISFIED},
	}}
	violated := &testpilotspb.Verdict{Status: testpilotspb.VERDICT_STATUS_VIOLATED, Rules: []*testpilotspb.RuleVerdict{
		{RuleId: "completion", Status: testpilotspb.RULE_VERDICT_STATUS_VIOLATED},
	}}
	agreeing := []SwitchVerdict{{Value: "hsm", Verdict: satisfied}, {Value: "chasm", Verdict: satisfied}}
	require.NoError(t, CheckSwitchAgreement(NexusImplementationSwitchName, agreeing))

	diverging := []SwitchVerdict{{Value: "hsm", Verdict: satisfied}, {Value: "chasm", Verdict: violated}}
	err := CheckSwitchAgreement(NexusImplementationSwitchName, diverging)
	require.Error(t, err)
	require.Contains(t, err.Error(), `the Verdict diverges across the "implementation" switch`)
	require.Contains(t, err.Error(), "implementation=hsm it is Satisfied [completion:Satisfied]")
	require.Contains(t, err.Error(), "implementation=chasm it is Violated [completion:Violated]")

	// A rule that differs under the same Verdict status is a divergence too.
	sameStatus := &testpilotspb.Verdict{Status: testpilotspb.VERDICT_STATUS_SATISFIED, Rules: []*testpilotspb.RuleVerdict{
		{RuleId: "completion", Status: testpilotspb.RULE_VERDICT_STATUS_INCONCLUSIVE},
	}}
	require.Error(t, CheckSwitchAgreement(NexusImplementationSwitchName,
		[]SwitchVerdict{{Value: "hsm", Verdict: satisfied}, {Value: "chasm", Verdict: sameStatus}}))
	require.NoError(t, CheckSwitchAgreement(NexusImplementationSwitchName, nil))
}
