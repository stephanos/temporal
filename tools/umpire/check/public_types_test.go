package check_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/check"
)

func TestMonitorEvaluationHasPublicType(t *testing.T) {
	read := func(monitor check.Monitor) check.Evaluation { return monitor.At }
	var evaluation check.Evaluation
	require.Equal(t, evaluation, read(check.Monitor{At: evaluation}))
}
