package replay

import (
	"fmt"
	"slices"
	"strings"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
)

// CorrelatedViolated is the terminal state the Case Runtime records for a violated correlated
// rule: a runtime constant, not a Definition ID, taken as-is.
const CorrelatedViolated = "correlated.violated"

// RuleKey is one violated rule as the key names it: the rule by Definition ID, the terminal state
// it reached, and its violating evidence by Definition ID (a monitor rule's observation ids, or
// nothing when its deadline violated it; a correlated rule's evidence kind).
type RuleKey struct {
	Rule     string   `json:"rule"`
	Terminal string   `json:"terminal"`
	Evidence []string `json:"evidence"`
}

// ViolationKey is the Contract-relative identity of a violation: what two Runs of one Case, or
// of a subject and its reduced candidate, share when they violated the Contract the same way. It
// reads Definition IDs, never Case-local names, and never the Case identity, the Run identity,
// sequences, times, instruction ids, values or the Verdict's accumulated support.
type ViolationKey struct {
	Rules []RuleKey `json:"rules"`
}

// Equal says the two keys name the same violation.
func (k ViolationKey) Equal(other ViolationKey) bool {
	return k.String() == other.String()
}

// String is the key's one rendering, rules sorted, for the report and for comparison.
func (k ViolationKey) String() string {
	parts := make([]string, 0, len(k.Rules))
	for _, rule := range k.Rules {
		parts = append(parts, fmt.Sprintf("%s@%s[%s]", rule.Rule, rule.Terminal, strings.Join(rule.Evidence, ",")))
	}
	return strings.Join(parts, ";")
}

// KeyOf derives the key of one evaluated Run of source: for each rule the Verdict names violated,
// its Definition ID, its terminal state and the violating evidence the evaluation names for it.
// Local names resolve through the Case's provenance rows where one exists and are taken as-is
// otherwise; a correlated rule's terminal state is the runtime constant.
func KeyOf(source *testpilotspb.Case, verdict *testpilotspb.Verdict, evaluation *testpilot.Evaluation) ViolationKey {
	resolve := resolver(source)
	correlated := map[string]bool{}
	for _, rule := range source.GetContract().GetCorrelated().GetRules() {
		correlated[rule.GetRuleId()] = true
	}
	violations := map[string]testpilot.RuleViolation{}
	if evaluation != nil {
		for _, violation := range evaluation.Violations {
			violations[violation.RuleID] = violation
		}
	}
	var key ViolationKey
	for _, rule := range verdict.GetRules() {
		if rule.GetStatus() != testpilotspb.RULE_VERDICT_STATUS_VIOLATED {
			continue
		}
		entry := RuleKey{Rule: resolve(rule.GetRuleId()), Evidence: []string{}}
		violation := violations[rule.GetRuleId()]
		if correlated[rule.GetRuleId()] {
			entry.Terminal = CorrelatedViolated
			if violation.CorrelatedKind != "" {
				entry.Evidence = []string{resolve(violation.CorrelatedKind)}
			}
		} else {
			entry.Terminal = resolve(rule.GetTerminalStateId())
			for _, observation := range violation.ObservationIDs {
				entry.Evidence = append(entry.Evidence, resolve(observation))
			}
		}
		slices.Sort(entry.Evidence)
		entry.Evidence = slices.Compact(entry.Evidence)
		key.Rules = append(key.Rules, entry)
	}
	slices.SortFunc(key.Rules, func(a, b RuleKey) int { return strings.Compare(a.Rule+"@"+a.Terminal, b.Rule+"@"+b.Terminal) })
	return key
}

// resolver maps a Case-local name to its Definition ID through the provenance rows, and leaves a
// name with no row as it is, since such a name is its own Definition ID or a runtime constant.
func resolver(source *testpilotspb.Case) func(string) string {
	names := map[string]string{}
	for _, row := range source.GetProvenance().GetLocalNames() {
		names[row.GetLocalName()] = row.GetDefinitionId()
	}
	return func(local string) string {
		if definition, ok := names[local]; ok {
			return definition
		}
		return local
	}
}
