package execution

import (
	"errors"
	"fmt"

	"go.temporal.io/server/tools/gomad3/internal/backends"
	"go.temporal.io/server/tools/gomad3/internal/model/choice"
)

// validateChoiceTraceLimit applies the native runner's bound on top of the codec's.
func validateChoiceTraceLimit(limit uint64) error {
	if err := choice.ValidateTraceLimit(limit); err != nil {
		return err
	}
	if limit > backends.MaximumChoiceTraceBytes {
		return fmt.Errorf("invalid choice trace limit %d", limit)
	}
	return nil
}

func projectChoiceSessionError(err error) error {
	if err == nil {
		return nil
	}
	var divergence *choice.ReplayDivergenceError
	switch {
	case errors.As(err, &divergence):
		return &backends.ChoiceReplayDivergenceError{Divergence: divergence.Divergence}
	case errors.Is(err, choice.ErrOverflow):
		return backends.ErrChoiceTraceOverflow
	case errors.Is(err, choice.ErrUnterminated):
		return backends.ErrChoiceTraceUnterminated
	default:
		return errors.Join(backends.ErrChoiceTraceMalformed, err)
	}
}
