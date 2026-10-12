package runner

import (
	"errors"

	backend "go.temporal.io/server/tools/gomad3/backends"
)

func backendChoiceReplayDivergence(err error) bool {
	for err != nil {
		if failure, ok := err.(*backend.FailureError); ok && failure.Termination == backend.Divergence {
			return true
		}
		err = errors.Unwrap(err)
	}
	return false
}
