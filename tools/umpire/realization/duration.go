package realization

import (
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/testpilot/duration"
	"google.golang.org/protobuf/types/known/durationpb"
)

func durationMilliseconds(value *durationpb.Duration) int64 {
	ms, err := duration.Milliseconds("elapsed bound", value)
	if err != nil {
		return -1
	}
	return ms
}

func (a *realizing) duration(at *umpirespb.Position, field string, value *durationpb.Duration, required, positive bool) int64 {
	ms, err := duration.Milliseconds(field, value)
	switch {
	case err != nil:
		a.report(at, "%s", err)
	case required && value == nil:
		a.report(at, "%s is required", field)
	case ms < 0 || positive && ms == 0:
		a.report(at, "%s must be %s", field, map[bool]string{true: "positive", false: "nonnegative"}[positive])
	default:
		return ms
	}
	return -1
}
