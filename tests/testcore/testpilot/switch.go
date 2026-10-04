package testpilot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/chasm/lib/activity"
	chasmnexus "go.temporal.io/server/chasm/lib/nexusoperation"
	"go.temporal.io/server/common/dynamicconfig"
)

// SwitchSetting is one dynamic config setting an environment sets: the setting, the value it takes,
// typed as the setting's Go type (bool, int, float64, string or time.Duration), and what asks for
// it, so two sources asking for one key differently are named together.
type SwitchSetting struct {
	Setting dynamicconfig.GenericSetting
	Value   any
	Source  string
}

// SwitchValue is one value of the switch a functional set repeats over: its name, and the dynamic
// configuration an environment running under it sets. A switch is a rollout flag between two
// implementations of one behavior, not a Model parameter: the Case bytes do not depend on it, the
// Profile records it, and a Verdict that differs between values is a divergence.
type SwitchValue struct {
	Name     string
	Settings []SwitchSetting
}

// Configuration is the value's settings as the derived Profile records them, keyed by each
// setting's key and spelled in the registry codec's text form (`true`, `100`, `1s`). The settings
// name each key once; ResolveSettings is what makes them so.
func (v SwitchValue) Configuration() map[string]string {
	configuration := make(map[string]string, len(v.Settings))
	for _, setting := range v.Settings {
		configuration[setting.Setting.Key().String()] = formatSettingValue(setting.Value)
	}
	return configuration
}

// formatSettingValue spells a typed value the way dynamic config's text form does.
func formatSettingValue(value any) string {
	switch value := value.(type) {
	case bool:
		return strconv.FormatBool(value)
	case int:
		return strconv.Itoa(value)
	case float64:
		return strconv.FormatFloat(value, 'g', -1, 64)
	case time.Duration:
		return value.String()
	case string:
		return value
	default:
		return fmt.Sprint(value)
	}
}

// ResolveSettings joins groups of settings into the configuration one cluster is constructed with:
// each key once. A key two sources give the same value is kept once; a key given two different
// values is refused, naming the key, both values and both sources, because which one the server
// should run with is a decision, not an order. A value the setting's type rejects is refused too.
func ResolveSettings(groups ...[]SwitchSetting) ([]SwitchSetting, error) {
	var resolved []SwitchSetting
	byKey := map[dynamicconfig.Key]int{}
	for _, group := range groups {
		for _, setting := range group {
			key := setting.Setting.Key()
			if err := setting.Setting.Validate(setting.Value); err != nil {
				return nil, fmt.Errorf("%s asks for %s=%v, which the setting refuses: %w", setting.Source, key, setting.Value, err)
			}
			index, seen := byKey[key]
			if !seen {
				byKey[key] = len(resolved)
				resolved = append(resolved, setting)
				continue
			}
			first := resolved[index]
			if formatSettingValue(first.Value) != formatSettingValue(setting.Value) {
				return nil, fmt.Errorf("dynamic config key %s is given two values: %s by %s and %s by %s",
					key, formatSettingValue(first.Value), first.Source, formatSettingValue(setting.Value), setting.Source)
			}
		}
	}
	return resolved, nil
}

// NexusImplementationSwitchName is the name of the Nexus implementation switch. The functional tests
// declare the switch and its value names; no Model, IR or Case declares it, and the Case bytes do not
// depend on it.
const NexusImplementationSwitchName = "implementation"

// NexusImplementationSwitch is the Nexus implementation switch: the HSM implementation and the CHASM
// one, each the six settings the upstream Nexus workflow suite sets at environment construction
// (tests/nexus_workflow_test.go). The rollout percent is among them: it defaults to 0, and below
// 100 a workflow's operations may stay on HSM with CHASM workflow operations enabled
// (chasmnexus.UseChasmForWorkflow).
func NexusImplementationSwitch() []SwitchValue {
	implementation := func(name string, chasm bool, rolloutPercent int) SwitchValue {
		source := NexusImplementationSwitchName + "=" + name
		return SwitchValue{Name: name, Settings: []SwitchSetting{
			{Setting: dynamicconfig.EnableChasm, Value: chasm, Source: source},
			{Setting: dynamicconfig.EnableCHASMCallbacks, Value: chasm, Source: source},
			{Setting: chasmnexus.Enabled, Value: chasm, Source: source},
			{Setting: chasmnexus.EnableChasmWorkflowOperations, Value: chasm, Source: source},
			{Setting: chasmnexus.ChasmWorkflowOperationsRolloutPercent, Value: rolloutPercent, Source: source},
			{Setting: dynamicconfig.EnableCHASMSignalBacklinks, Value: chasm, Source: source},
		}}
	}
	return []SwitchValue{implementation("hsm", false, 0), implementation("chasm", true, 100)}
}

// SchedulesWorkflowNexusOperation reports whether a Case's Program schedules a Nexus operation from
// a workflow, the one path the Nexus implementation switch selects. Binding an endpoint is not
// enough: a standalone Nexus operation runs on CHASM whatever the switch says.
func SchedulesWorkflowNexusOperation(source *testpilotspb.Case) bool {
	for _, entrypoint := range source.GetProgram().GetEntrypoints() {
		for _, node := range entrypoint.GetInstructions() {
			if node.GetInstruction().GetWorkflowCommand().GetCommand().GetScheduleNexusOperationCommandAttributes() != nil {
				return true
			}
		}
	}
	return false
}

// standaloneSettingsSource names the harness's blanket settings in a conflict.
const standaloneSettingsSource = "the harness's standalone settings"

// StandaloneSettings are the settings a Case outside the switch runs under whatever its Program
// requires: standalone activity, its operator commands, and CHASM. Nothing declares them yet; a
// Case that needs another value for one of them is refused by ResolveSettings rather than run.
func StandaloneSettings() []SwitchSetting {
	return []SwitchSetting{
		{Setting: activity.Enabled, Value: true, Source: standaloneSettingsSource},
		{Setting: activity.EnableStandaloneActivityOperatorCommands, Value: true, Source: standaloneSettingsSource},
		{Setting: dynamicconfig.EnableChasm, Value: true, Source: standaloneSettingsSource},
	}
}

// requiredSettingKinds is each dynamic-configuration key a generated Case may require, by its
// lower-case key: its typed setting and how its text value parses. dynamicconfig has no public
// lookup from a key to its setting, so a Case that requires a key missing here fails rather than
// running against a server that does not set it.
var requiredSettingKinds = map[string]struct {
	setting dynamicconfig.GenericSetting
	parse   func(value string) (any, error)
}{
	strings.ToLower(chasmnexus.Enabled.Key().String()): {
		setting: chasmnexus.Enabled,
		parse:   func(value string) (any, error) { return strconv.ParseBool(value) },
	},
}

// requiredSettingsSource names a Case's Program in a conflict.
const requiredSettingsSource = "the Case's required settings"

// CaseSettings is the configuration a Case's cluster is constructed with: base (a switch value's
// settings, or StandaloneSettings for a Case outside the switch) and every setting the Case's
// Program requires, each key once.
func CaseSettings(source *testpilotspb.Case, base []SwitchSetting) ([]SwitchSetting, error) {
	var required []SwitchSetting
	for _, setting := range source.GetProgram().GetRequiredSettings() {
		key := strings.ToLower(setting.GetKey())
		kind, ok := requiredSettingKinds[key]
		if !ok {
			return nil, fmt.Errorf("the Case requires %s, which the suite cannot set", setting.GetKey())
		}
		value, err := kind.parse(setting.GetValue())
		if err != nil {
			return nil, fmt.Errorf("the Case requires %s=%s: %w", setting.GetKey(), setting.GetValue(), err)
		}
		required = append(required, SwitchSetting{Setting: kind.setting, Value: value, Source: requiredSettingsSource})
	}
	return ResolveSettings(base, required)
}

// SwitchVerdict is the Verdict one Case produced under one switch value.
type SwitchVerdict struct {
	Value   string
	Verdict *testpilotspb.Verdict
}

// CheckSwitchAgreement reports a divergence: two switch values under which one Case produced
// Verdicts of different status, or of the same status with rule verdicts that differ. The error
// names the switch, both values and both Verdicts, because a divergence is a finding about the
// implementations rather than a flake, and the reader decides which one is wrong. Nil when every
// value agrees.
func CheckSwitchAgreement(switchName string, results []SwitchVerdict) error {
	for i := 1; i < len(results); i++ {
		first, other := results[0], results[i]
		if describeVerdict(first.Verdict) == describeVerdict(other.Verdict) {
			continue
		}
		return fmt.Errorf("the Verdict diverges across the %q switch: under %s=%s it is %s, under %s=%s it is %s",
			switchName, switchName, first.Value, describeVerdict(first.Verdict),
			switchName, other.Value, describeVerdict(other.Verdict))
	}
	return nil
}

// describeVerdict spells a Verdict the way a divergence names it: its status, and each rule's
// status in Verdict order.
func describeVerdict(verdict *testpilotspb.Verdict) string {
	rules := make([]string, 0, len(verdict.GetRules()))
	for _, rule := range verdict.GetRules() {
		rules = append(rules, rule.GetRuleId()+":"+rule.GetStatus().String())
	}
	if len(rules) == 0 {
		return verdict.GetStatus().String()
	}
	return verdict.GetStatus().String() + " [" + strings.Join(rules, ", ") + "]"
}
