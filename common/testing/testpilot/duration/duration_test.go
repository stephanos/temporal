package duration

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestMillisecondsRejectsInvalidDuration(t *testing.T) {
	t.Parallel()
	for name, value := range map[string]*durationpb.Duration{
		"negative":            {Seconds: -1},
		"negative nanos":      {Nanos: -1},
		"sub millisecond":     {Nanos: 1},
		"invalid nanos":       {Nanos: 1_000_000_000},
		"invalid sign":        {Seconds: 1, Nanos: -1_000_000},
		"invalid range":       {Seconds: 315_576_000_001},
		"conversion overflow": {Seconds: math.MaxInt64},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Milliseconds("instruction.timeout", value)
			require.ErrorContains(t, err, "instruction.timeout")
		})
	}
}

func TestMillisecondsExactRoundTrip(t *testing.T) {
	t.Parallel()
	for _, milliseconds := range []int64{0, 1, 999, 1000, 1001, math.MaxInt64 / 1_000_000, 315_576_000_000_000} {
		value := FromMilliseconds(milliseconds)
		got, err := Milliseconds("run.elapsed", value)
		require.NoError(t, err)
		require.Equal(t, milliseconds, got)
	}
	got, err := Milliseconds("optional", nil)
	require.NoError(t, err)
	require.Zero(t, got)
}
