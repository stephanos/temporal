package execution

import (
	"errors"
	"fmt"

	"go.temporal.io/server/tools/gomad3/choice"
)

const (
	MinimumChoiceTraceBytes = choice.MinimumTraceBytes
	// The native runner maps its trace and tape, so it keeps its historical
	// 64 MiB bound below the codec's whole-program maximum.
	MaximumChoiceTraceBytes      = 64 << 20
	MaximumChoiceReplayPlanBytes = MaximumChoiceTraceBytes - (choice.MaximumTraceBytes - choice.MaximumReplayPlanBytes)
)

// validateChoiceTraceLimit applies the native runner's bound on top of the codec's.
func validateChoiceTraceLimit(limit uint64) error {
	if err := choice.ValidateTraceLimit(limit); err != nil {
		return err
	}
	if limit > MaximumChoiceTraceBytes {
		return fmt.Errorf("invalid choice trace limit %d", limit)
	}
	return nil
}

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

func projectChoiceSessionError(err error) error {
	if err == nil {
		return nil
	}
	var divergence *choice.ReplayDivergenceError
	switch {
	case errors.As(err, &divergence):
		return &ChoiceReplayDivergenceError{Divergence: divergence.Divergence}
	case errors.Is(err, choice.ErrOverflow):
		return ErrChoiceTraceOverflow
	case errors.Is(err, choice.ErrUnterminated):
		return ErrChoiceTraceUnterminated
	default:
		return errors.Join(ErrChoiceTraceMalformed, err)
	}
}
