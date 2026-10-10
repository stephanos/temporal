package lower

import (
	"go.temporal.io/server/common/testing/testpilot/duration"
	"google.golang.org/protobuf/types/known/durationpb"
)

func durationMilliseconds(value *durationpb.Duration) int64 {
	ms, err := duration.Milliseconds("admitted elapsed bound", value)
	if err != nil {
		panic(err)
	}
	return ms
}
