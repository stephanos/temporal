package model

// The source-derived Models the lifter's tests pin in model/lifter/testdata/lifts/expected,
// read as any other Model: admitted, then interpreted.

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
)

func lifted(t *testing.T, name string) *umpirespb.Model {
	t.Helper()
	m, err := Load(filepath.Join("..", "..", "..", "model", "lifter", "testdata", "lifts", "expected", name+".json"))
	require.NoError(t, err)
	return m
}

func TestLiftedModelsAreAdmitted(t *testing.T) {
	for _, name := range []string{"admission", "channels", "closereset", "declarations", "presence", "realizations"} {
		t.Run(name, func(t *testing.T) {
			lifted(t, name)
		})
	}
}
