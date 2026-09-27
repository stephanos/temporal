package testpilot

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
)

// SignatureLinePrefix starts the one line a failing live assertion prints; the JSON after it is the
// whole interface between the live tests and the umpire-repeat harness that reads it.
const SignatureLinePrefix = "TESTPILOT-SIGNATURE "

// Signature is one failing live assertion's failure signature: the facts that tell its causes
// apart, and nothing that changes from one occurrence of a cause to the next (no Run id, no
// diagnostic detail). The field order is the canonical order the line is encoded in.
type Signature struct {
	Test            string                `json:"test"`
	Assertion       string                `json:"assertion"`
	RunDisposition  string                `json:"run_disposition"`
	VerdictStatus   string                `json:"verdict_status"`
	UnresolvedRules []SignatureRule       `json:"unresolved_rules"`
	Diagnostics     []SignatureDiagnostic `json:"diagnostics"`
	Leaks           []string              `json:"leaks"`
}

// SignatureRule is one rule the Verdict did not satisfy.
type SignatureRule struct {
	RuleID          string `json:"rule_id"`
	Status          string `json:"status"`
	TerminalStateID string `json:"terminal_state_id"`
}

// SignatureDiagnostic is one Run diagnostic by its kind and code; its detail names ids, so it is left
// out.
type SignatureDiagnostic struct {
	Kind string `json:"kind"`
	Code string `json:"code"`
}

// RunSignature is the signature of assertion failing in test over a closed Run and its Verdict. A
// Run that carries no Verdict of its own is read with the one given.
func RunSignature(test, assertion string, run *testpilotspb.Run, verdict *testpilotspb.Verdict) Signature {
	if verdict == nil {
		verdict = run.GetVerdict()
	}
	signature := Signature{
		Test:           test,
		Assertion:      assertion,
		RunDisposition: run.GetDisposition().String(),
		VerdictStatus:  verdict.GetStatus().String(),
	}
	for _, rule := range verdict.GetRules() {
		if rule.GetStatus() == testpilotspb.RULE_VERDICT_STATUS_SATISFIED {
			continue
		}
		signature.UnresolvedRules = append(signature.UnresolvedRules, SignatureRule{
			RuleID: rule.GetRuleId(), Status: rule.GetStatus().String(), TerminalStateID: rule.GetTerminalStateId(),
		})
	}
	for _, diagnostic := range run.GetDiagnostics() {
		signature.Diagnostics = append(signature.Diagnostics, SignatureDiagnostic{
			Kind: diagnostic.GetKind().String(), Code: diagnostic.GetCode(),
		})
	}
	return signature.canonical()
}

// ReportSignature is the signature of assertion failing in test over a Run that umpire-run reported
// from its own process: the Run, Verdict, rule and diagnostic lines of its stdout, and each
// resource it could not delete from its stderr.
func ReportSignature(test, assertion, stdout, stderr string) Signature {
	signature := Signature{Test: test, Assertion: assertion}
	for _, line := range strings.Split(stdout, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "run":
			signature.RunDisposition = fields[1]
		case "verdict":
			signature.VerdictStatus = fields[1]
		case "rule":
			if len(fields) >= 3 && fields[2] != testpilotspb.RULE_VERDICT_STATUS_SATISFIED.String() {
				signature.UnresolvedRules = append(signature.UnresolvedRules, SignatureRule{
					RuleID: fields[1], Status: fields[2], TerminalStateID: fieldAt(fields, 3),
				})
			}
		case "diagnostic":
			signature.Diagnostics = append(signature.Diagnostics, SignatureDiagnostic{Kind: fields[1], Code: fieldAt(fields, 2)})
		default:
			// The cleanup line, and anything else, names nothing the signature carries.
		}
	}
	for _, line := range strings.Split(stderr, "\n") {
		if line = strings.TrimSpace(line); strings.HasPrefix(line, "delete ") {
			signature.Leaks = append(signature.Leaks, line)
		}
	}
	return signature.canonical()
}

func fieldAt(fields []string, index int) string {
	if index < len(fields) {
		return fields[index]
	}
	return ""
}

// Line renders the signature as the one line a failing live assertion prints.
func (s Signature) Line() (string, error) {
	encoded, err := json.Marshal(s.canonical())
	if err != nil {
		return "", fmt.Errorf("encode the failure signature of %s: %w", s.Test, err)
	}
	return SignatureLinePrefix + string(encoded), nil
}

// Equal compares two signatures field by field, each list in its canonical order.
func (s Signature) Equal(other Signature) bool {
	a, b := s.canonical(), other.canonical()
	return a.Test == b.Test &&
		a.Assertion == b.Assertion &&
		a.RunDisposition == b.RunDisposition &&
		a.VerdictStatus == b.VerdictStatus &&
		slices.Equal(a.UnresolvedRules, b.UnresolvedRules) &&
		slices.Equal(a.Diagnostics, b.Diagnostics) &&
		slices.Equal(a.Leaks, b.Leaks)
}

// canonical sorts every list, on a copy, and spells an empty list as one, so two occurrences of one
// cause encode to the same bytes whatever order the Run recorded them in.
func (s Signature) canonical() Signature {
	s.UnresolvedRules = slices.SortedFunc(slices.Values(s.UnresolvedRules), func(a, b SignatureRule) int {
		return cmp.Or(cmp.Compare(a.RuleID, b.RuleID), cmp.Compare(a.Status, b.Status), cmp.Compare(a.TerminalStateID, b.TerminalStateID))
	})
	s.Diagnostics = slices.SortedFunc(slices.Values(s.Diagnostics), func(a, b SignatureDiagnostic) int {
		return cmp.Or(cmp.Compare(a.Kind, b.Kind), cmp.Compare(a.Code, b.Code))
	})
	s.Leaks = slices.Sorted(slices.Values(s.Leaks))
	if s.UnresolvedRules == nil {
		s.UnresolvedRules = []SignatureRule{}
	}
	if s.Diagnostics == nil {
		s.Diagnostics = []SignatureDiagnostic{}
	}
	if s.Leaks == nil {
		s.Leaks = []string{}
	}
	return s
}
