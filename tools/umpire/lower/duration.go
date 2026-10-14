package lower

import (
	"google.golang.org/protobuf/types/known/durationpb"
)

// durationMilliseconds reads a bound already checked by realization admission.
// Its protobuf range fits int64 milliseconds, including bounds beyond time.Duration.
func durationMilliseconds(value *durationpb.Duration) int64 {
	return value.GetSeconds()*1000 + int64(value.GetNanos()/1000000)
}
