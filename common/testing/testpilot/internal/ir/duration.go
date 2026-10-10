package ir

import (
	pbduration "go.temporal.io/server/common/testing/testpilot/duration"
	"google.golang.org/protobuf/types/known/durationpb"
)

func DurationMilliseconds(field string, value *durationpb.Duration) (int64, error) {
	result, err := pbduration.Milliseconds(field, value)
	if err != nil {
		return 0, Invalid(Malformed, field, err.Error())
	}
	return result, nil
}

func MillisecondsDuration(value int64) *durationpb.Duration {
	return pbduration.FromMilliseconds(value)
}
