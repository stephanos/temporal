package execution

import (
	"errors"
	"fmt"

	"go.temporal.io/server/tools/gomadv3/choice"
)

const (
	MinimumChoiceTraceBytes      = choice.MinimumTraceBytes
	MaximumChoiceTraceBytes      = choice.MaximumTraceBytes
	MaximumChoiceReplayPlanBytes = choice.MaximumReplayPlanBytes
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
