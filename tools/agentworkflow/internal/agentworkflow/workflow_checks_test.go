package agentworkflow

import (
	"strings"
	"testing"
)

func TestClassifyChecksRejectsOptionalMutation(t *testing.T) {
	outcome, message, repairable := classifyChecks([]CheckResult{{Name: "generator", Required: false, Outcome: "mutated"}})
	if outcome != OutcomeProjectFailed || !repairable || !strings.Contains(message, "mutated") {
		t.Fatalf("classifyChecks() = %q, %q, %t", outcome, message, repairable)
	}
}
