package execution

import testpilotspb "go.temporal.io/server/api/testpilot/v1"

// Conclude is the one aggregation of a closed Run's rule statuses into its Verdict status, and the
// disposition that Verdict leaves the Run in. Any violated rule makes the Verdict violated and the
// Run stopped by its Monitor, whatever else the Run did; a completed Run whose rules are all
// satisfied (vacuously so with no rules) is satisfied; anything else, a pending, unspecified or
// inconclusive rule or a Run that did not complete, is inconclusive and leaves the disposition as
// it was. The evaluator, the recorder, recorded-Run agreement and replay all conclude through it.
func Conclude(disposition testpilotspb.RunDisposition, rules []testpilotspb.RuleVerdictStatus) (testpilotspb.VerdictStatus, testpilotspb.RunDisposition) {
	allSatisfied := true
	for _, status := range rules {
		if status == testpilotspb.RULE_VERDICT_STATUS_VIOLATED {
			return testpilotspb.VERDICT_STATUS_VIOLATED, testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR
		}
		allSatisfied = allSatisfied && status == testpilotspb.RULE_VERDICT_STATUS_SATISFIED
	}
	if allSatisfied && disposition == testpilotspb.RUN_DISPOSITION_COMPLETED {
		return testpilotspb.VERDICT_STATUS_SATISFIED, disposition
	}
	return testpilotspb.VERDICT_STATUS_INCONCLUSIVE, disposition
}
