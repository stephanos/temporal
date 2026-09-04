package runtime_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
)

func TestPreflightErrorIs(t *testing.T) {
	_, matching := umpireruntime.NewPhaseLimit("", 0, 0, 0, 0)
	_, sameKind := umpireruntime.NewPhaseLimit("", 0, 0, 0, 0)
	require.Error(t, matching)
	require.Error(t, sameKind)
	var absent *umpireruntime.PreflightError
	for _, tc := range []struct {
		name   string
		err    error
		target error
		match  bool
	}{
		{"matching kind", sameKind, matching, true},
		{"different kind", &umpireruntime.PreflightError{}, matching, false},
		{"wrapped target", matching, fmt.Errorf("wrapped: %w", matching), false},
		{"wrapped receiver", fmt.Errorf("wrapped: %w", matching), matching, true},
		{"nil receiver", absent, matching, false},
		{"nil target", matching, nil, false},
		{"typed nil target", matching, absent, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.match {
				require.ErrorIs(t, tc.err, tc.target)
			} else {
				require.NotErrorIs(t, tc.err, tc.target)
			}
		})
	}
}
