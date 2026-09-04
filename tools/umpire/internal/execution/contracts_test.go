package execution

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
)

type factoryFunction func(context.Context, ProgramView) (Monitor, error)

func (f factoryFunction) New(ctx context.Context, view ProgramView) (Monitor, error) {
	return f(ctx, view)
}

type factoryMap map[string]int

func (factoryMap) New(context.Context, ProgramView) (Monitor, error) { return nil, nil }

type factorySlice []int

func (factorySlice) New(context.Context, ProgramView) (Monitor, error) { return nil, nil }

type factoryChannel chan int

func (factoryChannel) New(context.Context, ProgramView) (Monitor, error) { return nil, nil }

type factoryPointer struct{}

func (*factoryPointer) New(context.Context, ProgramView) (Monitor, error) { return nil, nil }

type testMonitor struct{}

func (*testMonitor) Observe(context.Context, *umpirespb.RunEvent) (Decision, error) {
	return Continue, nil
}
func (*testMonitor) Close(context.Context, *umpirespb.Run) (*umpirespb.Verdict, error) {
	return &umpirespb.Verdict{}, nil
}
func TestMonitorFactoryRejectsEveryNilCapableForm(t *testing.T) {
	c, catalog, p := fixture(t)
	prepared, err := Prepare(c, catalog, p)
	require.NoError(t, err)
	for _, factory := range []MonitorFactory{nil, factoryFunction(nil), factoryMap(nil), factorySlice(nil), factoryChannel(nil), (*factoryPointer)(nil)} {
		_, err := NewMonitor(t.Context(), factory, prepared.View())
		require.Error(t, err)
	}
	factory := factoryFunction(func(ctx context.Context, view ProgramView) (Monitor, error) {
		require.Equal(t, prepared.View().ProgramID(), view.ProgramID())
		require.Same(t, t.Context(), ctx)
		return &testMonitor{}, nil
	})
	monitor, err := NewMonitor(t.Context(), factory, prepared.View())
	require.NoError(t, err)
	require.NotNil(t, monitor)
	_, err = NewMonitor(t.Context(), factoryFunction(func(context.Context, ProgramView) (Monitor, error) { return (*testMonitor)(nil), nil }), prepared.View())
	require.Error(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = NewMonitor(ctx, factory, prepared.View())
	require.ErrorIs(t, err, context.Canceled)
}
