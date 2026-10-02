package model_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
)

func TestMonitorEvaluationHasPublicType(t *testing.T) {
	read := func(monitor umpiremodel.Monitor) umpiremodel.Evaluation { return monitor.At }
	var evaluation umpiremodel.Evaluation
	require.Equal(t, evaluation, read(umpiremodel.Monitor{At: evaluation}))
}
