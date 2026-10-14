package backend

import (
	"go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/evidence/record/softassert"
)

func observeSoftAssertions(result backends.ExecutionResult, reportOnly bool) *record.SoftAssertions {
	observation := softassert.Observe(result.Stdout.Bytes, result.Stderr.Bytes, result.Captured && result.GroupGone && !result.Stdout.Truncated && !result.Stderr.Truncated)
	if len(observation.Messages) == 0 && observation.Complete && !reportOnly {
		return nil
	}
	return &record.SoftAssertions{Messages: observation.Messages, ReportOnly: reportOnly, Inconclusive: !observation.Complete}
}
