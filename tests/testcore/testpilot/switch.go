package testpilot

import (
	"fmt"
	"strconv"
	"strings"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	chasmnexus "go.temporal.io/server/chasm/lib/nexusoperation"
	"go.temporal.io/server/common/dynamicconfig"
)

// SwitchSetting is one dynamic config setting a switch value sets, with the value it takes.
type SwitchSetting struct {
	Setting dynamicconfig.GenericSetting
	Value   bool
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
// setting's key.
func (v SwitchValue) Configuration() map[string]string {
	configuration := make(map[string]string, len(v.Settings))
	for _, setting := range v.Settings {
		configuration[setting.Setting.Key().String()] = strconv.FormatBool(setting.Value)
	}
	return configuration
}

// NexusImplementationSwitchName is the switch the Nexus realization declares, spelled as its
// `repeat:` names it.
const NexusImplementationSwitchName = "implementation"

// NexusImplementationSwitch is the Nexus implementation switch: the HSM implementation and the CHASM
// one, each the three settings the upstream Nexus suites set at environment construction. The value
// names are the ones the realization declares.
func NexusImplementationSwitch() []SwitchValue {
	implementation := func(name string, chasm bool) SwitchValue {
		return SwitchValue{Name: name, Settings: []SwitchSetting{
			{Setting: dynamicconfig.EnableChasm, Value: chasm},
			{Setting: dynamicconfig.EnableCHASMCallbacks, Value: chasm},
			{Setting: chasmnexus.EnableChasmWorkflowOperations, Value: chasm},
		}}
	}
	return []SwitchValue{implementation("hsm", false), implementation("chasm", true)}
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
