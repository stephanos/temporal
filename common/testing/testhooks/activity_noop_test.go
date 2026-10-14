//go:build !test_dep

package testhooks

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/namespace"
)

func TestActivityDispatchHookIsAbsentInProduction(t *testing.T) {
	hook, present := Get(NewTestHooks(), ActivityDispatch, namespace.ID("namespace"))
	require.False(t, present)
	require.Nil(t, hook)
}
