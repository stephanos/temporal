package testpilot

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	chasmnexus "go.temporal.io/server/chasm/lib/nexusoperation"
	"go.temporal.io/server/common/dynamicconfig"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tools/umpire/lower"
)

func TestNexusImplementationSwitchSetsEachImplementationsSettings(t *testing.T) {
	values := NexusImplementationSwitch()
	require.Len(t, values, 2)
	require.Equal(t, "hsm", values[0].Name)
	require.Equal(t, "chasm", values[1].Name)
	// A setting's key is already the catalog's spelling, lower-case, which is what the Profile
	// records and what the realization's switch names; a value is the registry codec's text form.
	// Both values name the six keys the upstream Nexus workflow suite sets.
	require.Equal(t, map[string]string{
		"history.enablechasm":                                  "false",
		"history.enablechasmcallbacks":                         "false",
		"history.enablechasmsignalbacklinks":                   "false",
		"nexusoperation.enablestandalone":                      "false",
		"nexusoperation.enablechasmworkflowoperations":         "false",
		"nexusoperation.chasmworkflowoperationsrolloutpercent": "0",
	}, values[0].Configuration())
	require.Equal(t, map[string]string{
		"history.enablechasm":                                  "true",
		"history.enablechasmcallbacks":                         "true",
		"history.enablechasmsignalbacklinks":                   "true",
		"nexusoperation.enablestandalone":                      "true",
		"nexusoperation.enablechasmworkflowoperations":         "true",
		"nexusoperation.chasmworkflowoperationsrolloutpercent": "100",
	}, values[1].Configuration())
}

// generatedSources is every lowered Case under model/cases, decoded, by its run name.
func generatedSources(t *testing.T) map[string]*testpilotspb.Case {
	t.Helper()
	directory := filepath.Join("..", "..", "..", "model", "cases")
	entries, err := GeneratedCases(directory)
	require.NoError(t, err)
	sources := map[string]*testpilotspb.Case{}
	for _, entry := range entries {
		if entry.Standing != lower.Lowered {
			continue
		}
		encoded, err := os.ReadFile(filepath.Join(directory, entry.File))
		require.NoError(t, err)
		source, err := testpilot.DecodeCaseProtoJSON(encoded)
		require.NoError(t, err)
		sources[GeneratedCaseName(entry)] = source
	}
	return sources
}

// usesChasmForWorkflow is what the CHASM workflow command handler decides for a workflow's new
// Nexus operations, read from a collection that holds exactly settings.
func usesChasmForWorkflow(settings []SwitchSetting, namespaceName, workflowID string) bool {
	client := dynamicconfig.NewMemoryClient()
	for _, setting := range settings {
		client.OverrideSetting(setting.Setting, setting.Value)
	}
	collection := dynamicconfig.NewCollection(client, log.NewNoopLogger())
	return chasmnexus.UseChasmForWorkflow(
		chasmnexus.EnableChasmWorkflowOperations.Get(collection)(namespaceName),
		chasmnexus.ChasmWorkflowOperationsRolloutPercent.Get(collection)(namespaceName),
		namespaceName, workflowID)
}

// Under exactly the settings the generated-Case harness constructs each workflow-Nexus Case's
// cluster with, `chasm` selects CHASM for every workflow and `hsm` selects HSM. Without the rollout
// percent, `chasm` would still run workflow operations on HSM: the setting defaults to 0.
func TestNexusImplementationSwitchSelectsTheImplementation(t *testing.T) {
	sources := generatedSources(t)
	scheduled := 0
	for name, source := range sources {
		if !SchedulesWorkflowNexusOperation(source) {
			continue
		}
		scheduled++
		for _, value := range NexusImplementationSwitch() {
			settings, err := CaseSettings(source, value.Settings)
			require.NoError(t, err, "%s under %s", name, value.Name)
			for i := range 32 {
				namespaceName, workflowID := "umpire-"+name, "workflow-"+strconv.Itoa(i)
				require.Equal(t, value.Name == "chasm", usesChasmForWorkflow(settings, namespaceName, workflowID),
					"%s under %s=%s, workflow %s", name, NexusImplementationSwitchName, value.Name, workflowID)
			}
		}
	}
	require.Positive(t, scheduled, "no generated Case schedules a workflow Nexus operation")

	chasm := NexusImplementationSwitch()[1]
	withoutRollout := slices.DeleteFunc(slices.Clone(chasm.Settings), func(setting SwitchSetting) bool {
		return setting.Setting.Key() == chasmnexus.ChasmWorkflowOperationsRolloutPercent.Key()
	})
	require.False(t, usesChasmForWorkflow(withoutRollout, "umpire-namespace", "workflow"))
}

// The switch applies to the Cases that schedule a workflow Nexus operation, not to every Case that
// binds an endpoint: a standalone Nexus operation's Cases run once, under the standalone settings
// and what they require.
func TestSchedulesWorkflowNexusOperationSelectsTheSwitchedCases(t *testing.T) {
	var switched, standaloneNexus []string
	for name, source := range generatedSources(t) {
		if SchedulesWorkflowNexusOperation(source) {
			switched = append(switched, name)
			continue
		}
		if strings.HasPrefix(name, "nexus-operation-") {
			standaloneNexus = append(standaloneNexus, name)
			settings, err := CaseSettings(source, StandaloneSettings())
			require.NoError(t, err, name)
			require.Equal(t, "true", SwitchValue{Settings: settings}.Configuration()["nexusoperation.enablestandalone"], name)
		}
	}
	for _, name := range switched {
		require.True(t, strings.HasPrefix(name, "nexus-caller-") || strings.HasPrefix(name, "nexus-control-"), name)
	}
	require.NotEmpty(t, switched)
	require.NotEmpty(t, standaloneNexus)
}

// One key given two values is refused, naming the key, both values and both sources; the same value
// twice is kept once, and a value the setting's type rejects is refused.
func TestResolveSettingsRefusesOneKeyGivenTwoValues(t *testing.T) {
	hsm := NexusImplementationSwitch()[0]
	_, err := ResolveSettings(hsm.Settings, StandaloneSettings())
	require.EqualError(t, err,
		"dynamic config key history.enablechasm is given two values: false by implementation=hsm and true by the harness's standalone settings")

	chasm := NexusImplementationSwitch()[1]
	resolved, err := ResolveSettings(chasm.Settings, StandaloneSettings())
	require.NoError(t, err)
	require.Len(t, resolved, len(chasm.Settings)+2)

	_, err = ResolveSettings([]SwitchSetting{{Setting: dynamicconfig.EnableChasm, Value: "yes", Source: "a test"}})
	require.ErrorContains(t, err, "a test asks for history.enablechasm=yes, which the setting refuses")

	_, err = CaseSettings(&testpilotspb.Case{Program: &testpilotspb.Program{RequiredSettings: []*testpilotspb.RequiredSetting{
		{Key: "nexusoperation.enableStandalone", Value: "true"},
	}}}, hsm.Settings)
	require.EqualError(t, err,
		"dynamic config key nexusoperation.enablestandalone is given two values: false by implementation=hsm and true by the Case's required settings")
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
