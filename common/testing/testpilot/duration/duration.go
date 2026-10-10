package duration

import (
	"fmt"
	"math"

	"google.golang.org/protobuf/types/known/durationpb"
)

// Milliseconds checks exact protobuf duration arithmetic before integer conversion. A nil value
// returns zero; callers retain presence and enforce their own positive/default rules.
func Milliseconds(field string, value *durationpb.Duration) (int64, error) {
	if value == nil {
		return 0, nil
	}
	if err := value.CheckValid(); err != nil {
		return 0, fmt.Errorf("%s: invalid Duration: %w", field, err)
	}
	if value.Seconds < 0 || value.Nanos < 0 {
		return 0, fmt.Errorf("%s: Duration must be nonnegative", field)
	}
	if value.Nanos%1_000_000 != 0 {
		return 0, fmt.Errorf("%s: Duration must contain exact whole milliseconds", field)
	}
	nanosMilliseconds := int64(value.Nanos / 1_000_000)
	if value.Seconds > (math.MaxInt64-nanosMilliseconds)/1000 {
		return 0, fmt.Errorf("%s: Duration overflows milliseconds", field)
	}
	return value.Seconds*1000 + nanosMilliseconds, nil
}

// FromMilliseconds constructs an exact Duration without overflowing time.Duration arithmetic.
// Admission still validates the resulting protobuf range and the owning field's presence rules.
func FromMilliseconds(value int64) *durationpb.Duration {
	return &durationpb.Duration{Seconds: value / 1000, Nanos: int32(value%1000) * 1_000_000}
}
