package backend

import (
	"testing"

	"go.temporal.io/server/tools/gomad3/backends"
	"go.temporal.io/server/tools/gomad3/record"
)

func TestSoftAssertionOverridesSuccessfulGuestExit(t *testing.T) {
	for _, reportOnly := range []bool{false, true} {
		result := backends.ExecutionResult{Captured: true, GroupGone: true, Termination: backends.TerminationExit, Stdout: backends.Output{Bytes: []byte("PASS\nfailed assertion: invalid state\n")}}
		result.SoftAssertions = observeSoftAssertions(result, reportOnly)
		classified := backends.Classify(result, false, record.WorldTerminal{})
		want := "soft_assertion"
		if reportOnly {
			want = "success"
		}
		if classified.Reason != want || classified.SoftAssertions == nil || len(classified.SoftAssertions.Messages) != 1 || classified.SoftAssertions.Messages[0] != "invalid state" {
			t.Fatalf("classification = %+v", classified)
		}
	}
}

func TestSoftAssertionTruncationIsInconclusiveEvenWithReportOnly(t *testing.T) {
	for _, reportOnly := range []bool{false, true} {
		result := backends.ExecutionResult{Captured: true, GroupGone: true, Termination: backends.TerminationExit, Stdout: backends.Output{Bytes: []byte("PASS\n"), Truncated: true}}
		result.SoftAssertions = observeSoftAssertions(result, reportOnly)
		classified := backends.Classify(result, false, record.WorldTerminal{})
		if classified.Reason != "soft_assertion_inconclusive" || classified.Domain == "success" {
			t.Fatalf("truncated evidence passed: %+v", classified)
		}
	}
}
