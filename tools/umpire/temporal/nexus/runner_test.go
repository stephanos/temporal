package nexus

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
)

func TestNewBindingRejectsAnIncompleteEnvironmentFactory(t *testing.T) {
	var typedNil *bindingEnvironmentFactory
	for name, factory := range map[string]umpireruntime.EnvironmentFactory{
		"nil interface": nil,
		"pointer":       typedNil,
		"map":           bindingEnvironmentFactoryMap(nil),
		"slice":         bindingEnvironmentFactorySlice(nil),
		"function":      bindingEnvironmentFactoryFunc(nil),
		"channel":       bindingEnvironmentFactoryChan(nil),
	} {
		t.Run(name, func(t *testing.T) {
			binding, err := NewBinding(factory)

			require.EqualError(t, err, "nexus binding requires an environment factory")
			require.Equal(t, Binding{}, binding)
		})
	}
}

func TestNewBindingRetainsOnlyTheSuppliedEnvironmentFactory(t *testing.T) {
	factory := &bindingEnvironmentFactory{}
	binding, err := NewBinding(factory)

	require.NoError(t, err)
	require.Same(t, factory, binding.EnvironmentFactory())
}

func TestZeroBindingHasNoEnvironmentFactory(t *testing.T) {
	require.Nil(t, (Binding{}).EnvironmentFactory())
}

type bindingEnvironmentFactory struct{}

type bindingEnvironmentFactoryMap map[string]string

type bindingEnvironmentFactorySlice []string

type bindingEnvironmentFactoryFunc func()

type bindingEnvironmentFactoryChan chan struct{}

func (*bindingEnvironmentFactory) Prepare(
	context.Context,
	umpireruntime.CheckedRunRequest,
	umpireruntime.Command,
) (umpireruntime.Environment, umpireruntime.Receipt) {
	panic("the constructor must not prepare an environment")
}

func (bindingEnvironmentFactoryMap) Prepare(
	context.Context,
	umpireruntime.CheckedRunRequest,
	umpireruntime.Command,
) (umpireruntime.Environment, umpireruntime.Receipt) {
	panic("the constructor must not prepare an environment")
}

func (bindingEnvironmentFactorySlice) Prepare(
	context.Context,
	umpireruntime.CheckedRunRequest,
	umpireruntime.Command,
) (umpireruntime.Environment, umpireruntime.Receipt) {
	panic("the constructor must not prepare an environment")
}

func (bindingEnvironmentFactoryFunc) Prepare(
	context.Context,
	umpireruntime.CheckedRunRequest,
	umpireruntime.Command,
) (umpireruntime.Environment, umpireruntime.Receipt) {
	panic("the constructor must not prepare an environment")
}

func (bindingEnvironmentFactoryChan) Prepare(
	context.Context,
	umpireruntime.CheckedRunRequest,
	umpireruntime.Command,
) (umpireruntime.Environment, umpireruntime.Receipt) {
	panic("the constructor must not prepare an environment")
}
