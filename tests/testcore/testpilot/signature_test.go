package testpilot

import (
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
)

// inconclusiveRun is an async Run whose finish instruction ran out of time, with the ids and the
// detail every occurrence of that cause spells differently.
func inconclusiveRun(runID string) (*testpilotspb.Run, *testpilotspb.Verdict) {
	verdict := &testpilotspb.Verdict{
		Status: testpilotspb.VERDICT_STATUS_INCONCLUSIVE,
		Rules: []*testpilotspb.RuleVerdict{
			{RuleId: "clause-two", Status: testpilotspb.RULE_VERDICT_STATUS_PENDING},
			{RuleId: "clause-one", Status: testpilotspb.RULE_VERDICT_STATUS_SATISFIED, TerminalStateId: "correlated.satisfied"},
			{RuleId: "clause-three", Status: testpilotspb.RULE_VERDICT_STATUS_PENDING, TerminalStateId: "awaiting"},
		},
	}
	return &testpilotspb.Run{
		RunId:       runID,
		Disposition: testpilotspb.RUN_DISPOSITION_INCOMPLETE,
		Verdict:     verdict,
		Diagnostics: []*testpilotspb.RunDiagnostic{
			{DiagnosticId: runID + "-2", Kind: testpilotspb.RUN_DIAGNOSTIC_KIND_MONITOR, Code: "pending", Detail: "rule clause-two pending in " + runID},
			{DiagnosticId: runID + "-1", Kind: testpilotspb.RUN_DIAGNOSTIC_KIND_EXECUTION, Code: "instruction-timeout", Detail: "finish-workflow ran out in " + runID},
		},
	}, verdict
}

const inconclusiveLine = `TESTPILOT-SIGNATURE {"test":"TestTestpilotSample/hsm","assertion":"verdict status",` +
	`"run_disposition":"Incomplete","verdict_status":"Inconclusive",` +
	`"unresolved_rules":[{"rule_id":"clause-three","status":"Pending","terminal_state_id":"awaiting"},{"rule_id":"clause-two","status":"Pending","terminal_state_id":""}],` +
	`"diagnostics":[{"kind":"Execution","code":"instruction-timeout"},{"kind":"Monitor","code":"pending"}],` +
	`"leaks":[]}`

func TestSignatureLineIsCanonical(t *testing.T) {
	run, verdict := inconclusiveRun("0199aa11-7e1c-4bd2-9f5e-4a7c1d2e3f40")
	for name, signature := range map[string]Signature{
		"a Run and its Verdict":          RunSignature("TestTestpilotSample/hsm", "verdict status", run, verdict),
		"a Run carrying its own Verdict": RunSignature("TestTestpilotSample/hsm", "verdict status", run, nil),
		// umpire-run reports the same Run across its process boundary.
		"an umpire-run report": ReportSignature("TestTestpilotSample/hsm", "verdict status",
			"run Incomplete\ncleanup Succeeded\nverdict Inconclusive\n"+
				"rule clause-two Pending \nrule clause-one Satisfied correlated.satisfied\nrule clause-three Pending awaiting\n"+
				"diagnostic Monitor pending\ndiagnostic Execution instruction-timeout\n",
			""),
	} {
		t.Run(name, func(t *testing.T) {
			line, err := signature.Line()
			require.NoError(t, err)
			require.Equal(t, inconclusiveLine, line)
		})
	}
}

func TestSignatureLineCarriesNoRunSpecificValue(t *testing.T) {
	first, firstVerdict := inconclusiveRun("0199aa11-7e1c-4bd2-9f5e-4a7c1d2e3f40")
	second, secondVerdict := inconclusiveRun("0199bb22-0000-4bd2-9f5e-4a7c1d2e3f41")
	a := RunSignature("TestTestpilotSample/hsm", "verdict status", first, firstVerdict)
	b := RunSignature("TestTestpilotSample/hsm", "verdict status", second, secondVerdict)
	require.True(t, a.Equal(b))
	aLine, err := a.Line()
	require.NoError(t, err)
	bLine, err := b.Line()
	require.NoError(t, err)
	require.Equal(t, aLine, bLine)
	require.NotContains(t, aLine, "0199aa11")
}

func TestSignatureReportCarriesLeaks(t *testing.T) {
	signature := ReportSignature("TestTestpilotUmpireRun", "exit status",
		"run Completed\ncleanup Succeeded\nverdict Satisfied\nrule clause-one Satisfied answered\n",
		"delete namespace umpire-run: context deadline exceeded\nsome other line\ndelete Nexus endpoint e: gone\n")
	require.Equal(t, Signature{
		Test: "TestTestpilotUmpireRun", Assertion: "exit status",
		RunDisposition: "Completed", VerdictStatus: "Satisfied",
		UnresolvedRules: []SignatureRule{}, Diagnostics: []SignatureDiagnostic{},
		Leaks: []string{"delete Nexus endpoint e: gone", "delete namespace umpire-run: context deadline exceeded"},
	}, signature)
}

func TestSignatureEqualComparesEveryField(t *testing.T) {
	run, verdict := inconclusiveRun("run-1")
	base := RunSignature("TestTestpilotSample/hsm", "verdict status", run, verdict)
	for name, change := range map[string]func(*Signature){
		"test":             func(s *Signature) { s.Test = "TestTestpilotSample/chasm" },
		"assertion":        func(s *Signature) { s.Assertion = "run disposition" },
		"run disposition":  func(s *Signature) { s.RunDisposition = "Completed" },
		"verdict status":   func(s *Signature) { s.VerdictStatus = "Violated" },
		"unresolved rules": func(s *Signature) { s.UnresolvedRules = s.UnresolvedRules[:1] },
		"diagnostics":      func(s *Signature) { s.Diagnostics = nil },
		"leaks":            func(s *Signature) { s.Leaks = []string{"delete namespace n: timeout"} },
	} {
		t.Run(name, func(t *testing.T) {
			changed := RunSignature("TestTestpilotSample/hsm", "verdict status", run, verdict)
			change(&changed)
			require.False(t, base.Equal(changed))
		})
	}
	// List order is not a difference.
	reordered := base
	reordered.UnresolvedRules = []SignatureRule{base.UnresolvedRules[1], base.UnresolvedRules[0]}
	require.True(t, base.Equal(reordered))
}
