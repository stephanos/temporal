package main

import (
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/recordedrun"
)

func mustHash(t *testing.T, signature Signature) string {
	t.Helper()
	hash, err := signature.Hash()
	require.NoError(t, err)
	return hash
}

func TestSignatureHashIgnoresWhereTheAssertionFired(t *testing.T) {
	before := Signature{Test: "TestTestpilotNexusPairCase", Assertion: "pair_test.go:88: verdict Inconclusive, want Satisfied"}
	moved := Signature{Test: "TestTestpilotNexusPairCase", Assertion: "pair_test.go:120: verdict Inconclusive, want Satisfied"}
	other := Signature{Test: "TestTestpilotNexusPairCase", Assertion: "pair_test.go:88: verdict Violated, want Satisfied"}

	require.Equal(t, mustHash(t, before), mustHash(t, moved))
	require.NotEqual(t, mustHash(t, before), mustHash(t, other))
}

func TestSignatureHashNormalizesIterationSpecificValues(t *testing.T) {
	signature := func(runID, port, suffix string, rules []UnresolvedRule) Signature {
		return Signature{
			Test:            "TestTestpilotUmpireRunRunsACheckedInCaseAgainstAnyEndpoint",
			Assertion:       "run " + runID + " against 127.0.0.1:" + port + " left namespace umpire-run-ns-deleted-" + suffix,
			RunDisposition:  "Incomplete",
			UnresolvedRules: rules,
			Leaks:           []string{"namespace umpire-run-ns: context deadline exceeded after 30.002s"},
		}
	}
	one := []UnresolvedRule{{RuleID: "a", Status: "Pending"}, {RuleID: "b", Status: "Pending"}}
	reordered := []UnresolvedRule{{RuleID: "b", Status: "Pending"}, {RuleID: "a", Status: "Pending"}}

	require.Equal(t,
		mustHash(t, signature("testpilot-run-3f1c2a9e-7b4d-4e8a-9c1f-0a2b3c4d5e6f", "53211", "x7k2p", one)),
		mustHash(t, signature("testpilot-run-a0b1c2d3-e4f5-4a6b-8c7d-9e0f1a2b3c4d", "61874", "q9w8e", reordered)))
	require.NotEqual(t,
		mustHash(t, signature("testpilot-run-3f1c2a9e-7b4d-4e8a-9c1f-0a2b3c4d5e6f", "53211", "x7k2p", one)),
		mustHash(t, signature("testpilot-run-3f1c2a9e-7b4d-4e8a-9c1f-0a2b3c4d5e6f", "53211", "x7k2p", one[:1])))
}

// A reserved signature never merges into a parsed one with the same test and assertion.
func TestReservedSignaturesHashApart(t *testing.T) {
	parsed := Signature{Test: "TestTestpilotNexusPairCase"}
	unparsed := Signature{Test: "TestTestpilotNexusPairCase", Reserved: reservedUnparsed}
	process := Signature{Test: "TestTestpilotNexusPairCase", Reserved: reservedProcess}

	require.NotEqual(t, mustHash(t, parsed), mustHash(t, unparsed))
	require.NotEqual(t, mustHash(t, unparsed), mustHash(t, process))
}

// A signature line that does not decode is no signature: the assertion is read the fallback way and
// the raw line is kept.
func TestSignatureOfFallsBackFromAMalformedSignatureLine(t *testing.T) {
	signature := signatureOf("TestTestpilotNexusPairCase", []outputLine{
		{text: `TESTPILOT-SIGNATURE {"test":`},
		{text: "    pair_test.go:88: verdict Inconclusive", kind: "error"},
	})

	require.Equal(t, Signature{
		Test: "TestTestpilotNexusPairCase", Assertion: "verdict Inconclusive", Location: "pair_test.go:88",
		Detail: `TESTPILOT-SIGNATURE {"test":`,
	}, signature)
}

// The line a live test logs is read back field for field: the live tests' encoder and this parser
// are the two ends of one contract, so each is pinned against the other rather than a copy of it.
func TestSignatureOfReadsTheLineALiveTestLogs(t *testing.T) {
	verdict := &testpilotspb.Verdict{
		Status: testpilotspb.VERDICT_STATUS_INCONCLUSIVE,
		Rules: []*testpilotspb.RuleVerdict{
			{RuleId: "clause-two", Status: testpilotspb.RULE_VERDICT_STATUS_PENDING},
			{RuleId: "clause-one", Status: testpilotspb.RULE_VERDICT_STATUS_SATISFIED},
		},
	}
	run := &testpilotspb.Run{
		Disposition: testpilotspb.RUN_DISPOSITION_INCOMPLETE,
		Verdict:     verdict,
		Diagnostics: []*testpilotspb.RunDiagnostic{{Kind: testpilotspb.RUN_DIAGNOSTIC_KIND_EXECUTION, Code: "instruction-timeout"}},
	}
	line, err := recordedrun.RunSignature("TestTestpilotNexusPairCase", "verdict status", run, verdict).Line()
	require.NoError(t, err)

	signature := signatureOf("TestTestpilotNexusPairCase", []outputLine{
		{text: "    testpilot_signature_test.go:42: " + line + "\n"},
		{text: "    testpilot_signature_test.go:77: verdict Inconclusive", kind: "error"},
	})

	require.Equal(t, Signature{
		Test: "TestTestpilotNexusPairCase", Assertion: "verdict status",
		RunDisposition: "Incomplete", VerdictStatus: "Inconclusive",
		UnresolvedRules: []UnresolvedRule{{RuleID: "clause-two", Status: "Pending"}},
		Diagnostics:     []Diagnostic{{Kind: "Execution", Code: "instruction-timeout"}},
		Leaks:           []string{},
	}, signature)
}

func TestClopperPearsonMatchesTheExactInterval(t *testing.T) {
	for _, tc := range []struct {
		failures, trials int
		lower, upper     float64
	}{
		{failures: 0, trials: 200, lower: 0, upper: 0.018275},
		{failures: 5, trials: 10, lower: 0.187086, upper: 0.812914},
		{failures: 10, trials: 10, lower: 0.691503, upper: 1},
	} {
		lower, upper := clopperPearson(tc.failures, tc.trials)
		require.InDelta(t, tc.lower, lower, 1e-5)
		require.InDelta(t, tc.upper, upper, 1e-5)
	}
}
