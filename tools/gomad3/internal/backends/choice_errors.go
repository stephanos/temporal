package backends

import (
	"errors"
	"fmt"

	"go.temporal.io/server/tools/gomad3/internal/model/choice"
)

const (
	MinimumChoiceTraceBytes = choice.MinimumTraceBytes
	// The native runner maps its trace and tape, so it keeps its historical
	// 64 MiB bound below the codec's whole-program maximum.
	MaximumChoiceTraceBytes      = 64 << 20
	MaximumChoiceReplayPlanBytes = MaximumChoiceTraceBytes - (choice.MaximumTraceBytes - choice.MaximumReplayPlanBytes)
)

var (
	ErrChoiceTraceMalformed    = errors.New("choice trace malformed")
	ErrChoiceTraceOverflow     = errors.New("choice trace overflow")
	ErrChoiceReplayDivergence  = errors.New("choice replay divergence")
	ErrChoiceTraceUnterminated = errors.New("choice trace unterminated")
)

type ChoiceReplayDivergenceError struct {
	Divergence choice.Divergence
}

func (err *ChoiceReplayDivergenceError) Error() string {
	return fmt.Sprintf("%v: ordinal=%d reason=%s", ErrChoiceReplayDivergence, err.Divergence.Ordinal, choice.DivergenceReasonName(err.Divergence.Reason))
}

func (err *ChoiceReplayDivergenceError) Unwrap() error {
	return ErrChoiceReplayDivergence
}
