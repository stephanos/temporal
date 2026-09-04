package nexus

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
)

func TestNewBindingRejectsAnIncompleteEnvironmentFactory(t *testing.T) {
	var typedNil *bindingEnvironmentFactory
	for _, factory := range []umpireruntime.EnvironmentFactory{nil, typedNil} {
		binding, err := NewBinding(factory)

		require.EqualError(t, err, "nexus binding requires an environment factory")
		require.Equal(t, Binding{}, binding)
	}
}

func TestNewBindingRetainsOnlyTheSuppliedEnvironmentFactory(t *testing.T) {
	factory := &bindingEnvironmentFactory{}
	binding, err := NewBinding(factory)

	require.NoError(t, err)
	require.Same(t, factory, binding.EnvironmentFactory())
}

type bindingEnvironmentFactory struct{}

func (*bindingEnvironmentFactory) Prepare(
	context.Context,
	umpireruntime.CheckedRunRequest,
	umpireruntime.Command,
) (umpireruntime.Environment, umpireruntime.Receipt) {
	panic("the constructor must not prepare an environment")
}
