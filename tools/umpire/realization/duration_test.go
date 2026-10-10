package realization

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/testpilot/duration"
	"google.golang.org/protobuf/types/known/durationpb"
)

type durationAdmitter struct {
	rejectionAdmitter
	positions []*umpirespb.Position
}

func (a *durationAdmitter) Report(at *umpirespb.Position, format string, args ...any) {
	a.positions = append(a.positions, at)
	a.rejectionAdmitter.Report(at, format, args...)
}

func TestNativeDurationAdmissionPreservesBoundsAndOwningSource(t *testing.T) {
	at := &umpirespb.Position{File: "Realization.scala", Line: 37}
	for _, test := range []struct {
		name, field        string
		value              *durationpb.Duration
		required, positive bool
		milliseconds       int64
		problem            string
	}{
		{name: "missing default", field: "default.timeout", required: true, positive: true, problem: "is required"},
		{name: "missing optional", field: "command.timeout", milliseconds: 0},
		{name: "explicit zero optional", field: "command.timeout", value: &durationpb.Duration{}, milliseconds: 0},
		{name: "zero poll", field: "poll.interval", value: &durationpb.Duration{}, positive: true, problem: "must be positive"},
		{name: "negative bound", field: "wait.at_most", value: &durationpb.Duration{Seconds: -1}, positive: true, problem: "must be nonnegative"},
		{name: "fractional millisecond", field: "server-step.deadline", value: &durationpb.Duration{Nanos: 1}, positive: true, problem: "exact whole milliseconds"},
		{name: "malformed nanos", field: "wait.interval", value: &durationpb.Duration{Nanos: 1000000000}, positive: true, problem: "invalid Duration"},
		{name: "mixed signs", field: "command.timeout", value: &durationpb.Duration{Seconds: 1, Nanos: -1000000}, problem: "invalid Duration"},
		{name: "protobuf range", field: "default.timeout", value: &durationpb.Duration{Seconds: 315576000001}, problem: "invalid Duration"},
		{name: "integer construction overflow range", field: "wait.at_most", value: duration.FromMilliseconds(math.MaxInt64), problem: "invalid Duration"},
		{name: "whole milliseconds", field: "server-step.deadline", value: duration.FromMilliseconds(1501), positive: true, milliseconds: 1501},
		{name: "valid range beyond time Duration", field: "wait.at_most", value: &durationpb.Duration{Seconds: 315576000000}, positive: true, milliseconds: 315576000000000},
	} {
		t.Run(test.name, func(t *testing.T) {
			d := &durationAdmitter{}
			a := &realizing{d: d, r: &umpirespb.Realization{}, owner: "realization test"}
			got := a.duration(at, test.field, test.value, test.required, test.positive)
			if test.problem == "" {
				require.Empty(t, d.problems)
				require.Equal(t, test.milliseconds, got)
			} else {
				require.Equal(t, int64(-1), got)
				require.Len(t, d.problems, 1)
				require.Contains(t, d.problems[0], test.field)
				require.Contains(t, d.problems[0], test.problem)
				require.Equal(t, []*umpirespb.Position{at}, d.positions)
			}
		})
	}
}
