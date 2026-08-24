package fault

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
)

func TestPreflightRejectsEscapingUnsupportedAndRestrictedFaults(t *testing.T) {
	t.Parallel()

	term := Term{
		Kind:       protocolcatalog.FaultKindDrop,
		Scope:      Scope{Namespaces: []string{"namespace"}, Services: []string{"frontend"}, Routes: []string{"StartWorkflowExecution"}},
		Occurrence: Occurrence{First: 1, Count: 1}, Interval: Interval{Start: 1, Stop: 2},
	}
	require.ErrorContains(t, Preflight(term, nil, false), "missing capabilities")
	require.NoError(t, Preflight(term, []protocolcatalog.CapabilityID{protocolcatalog.CapabilityIDFaultRpc}, false))

	term.Scope.Namespaces = []string{"one", "two"}
	require.ErrorContains(t, Preflight(term, []protocolcatalog.CapabilityID{protocolcatalog.CapabilityIDFaultRpc}, false), "one isolated namespace")

	restricted := term
	restricted.Kind = protocolcatalog.FaultKindClockSkew
	restricted.Scope = Scope{Namespaces: []string{"namespace"}, Participants: []string{"worker"}}
	require.ErrorContains(t, Preflight(restricted, []protocolcatalog.CapabilityID{protocolcatalog.CapabilityIDFaultClock}, false), "restricted")
}

func TestRunAlwaysReleasesAndCleansFault(t *testing.T) {
	t.Parallel()

	realizer := &fakeRealizer{}
	term := dropTerm()
	err := Run(context.Background(), term, realizer, Options{
		Capabilities: []protocolcatalog.CapabilityID{protocolcatalog.CapabilityIDFaultRpc}, CleanupTimeout: time.Second,
	}, func(context.Context) error {
		return errors.New("injected failure")
	})
	require.ErrorContains(t, err, "injected failure")
	require.Equal(t, []string{"install", "activate", "release", "cleanup"}, realizer.calls)
}

func TestRunCleansFaultAfterPanic(t *testing.T) {
	t.Parallel()

	realizer := &fakeRealizer{}
	require.PanicsWithValue(t, "panic", func() {
		_ = Run(context.Background(), dropTerm(), realizer, Options{
			Capabilities: []protocolcatalog.CapabilityID{protocolcatalog.CapabilityIDFaultRpc}, CleanupTimeout: time.Second,
		}, func(context.Context) error {
			panic("panic")
		})
	})
	require.Equal(t, []string{"install", "activate", "release", "cleanup"}, realizer.calls)
}

func TestLearnedFootprintsSelectDeterministically(t *testing.T) {
	t.Parallel()

	footprints := []Footprint{
		{Protocol: "grpc", Service: "matching", Route: "Poll", Risk: 2},
		{Protocol: "http", Service: "nexus", Route: "Start", Risk: 5},
		{Protocol: "grpc", Service: "matching", Route: "Poll", Risk: 2},
	}
	first := SelectFootprints(footprints, 7, 2)
	second := SelectFootprints(footprints, 7, 2)
	require.Equal(t, first, second)
	require.Len(t, first, 2)
	require.Equal(t, "Start", first[0].Route)
	require.True(t, first[0].RealizationEvidence)
}

func dropTerm() Term {
	return Term{
		Kind:       protocolcatalog.FaultKindDrop,
		Scope:      Scope{Namespaces: []string{"namespace"}, Services: []string{"frontend"}, Routes: []string{"route"}},
		Occurrence: Occurrence{First: 1, Count: 1}, Interval: Interval{Start: 1, Stop: 2},
	}
}

type fakeRealizer struct {
	calls []string
}

func (r *fakeRealizer) Install(context.Context, Term) (string, error) {
	r.calls = append(r.calls, "install")
	return "handle", nil
}
func (r *fakeRealizer) Activate(context.Context, string) error {
	r.calls = append(r.calls, "activate")
	return nil
}
func (r *fakeRealizer) Release(context.Context, string) error {
	r.calls = append(r.calls, "release")
	return nil
}
func (r *fakeRealizer) Cleanup(context.Context, string) error {
	r.calls = append(r.calls, "cleanup")
	return nil
}
