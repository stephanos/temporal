package observed

import (
	"crypto/sha256"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/determinism"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	"go.temporal.io/server/tools/gomad/internal/model/cluster"
)

func TestInspectionPreservesDecisionAndTraceCoordinates(t *testing.T) {
	t.Parallel()
	a, b := sha256.Sum256([]byte("a")), sha256.Sum256([]byte("b"))
	first, err := choice.CanonicalDecision(0, choice.KindRunnable, 10, false, [][32]byte{a, b}, a, 0)
	require.NoError(t, err)
	second, err := choice.CanonicalDecision(2, choice.KindRunnable, 20, false, [][32]byte{a, b}, b, 0)
	require.NoError(t, err)
	ready, err := choice.NewReadiness(1, 0)
	require.NoError(t, err)
	trace, err := choice.BuildTrace([]choice.Record{first.Record(), {Ordinal: 1, Kind: choice.KindSelectResult, Flags: choice.FlagObservation, Alternatives: 1, Data: 1, SiteOffset: 15, Origin: 1, Readiness: ready}, second.Record()}, choice.TerminalComplete)
	require.NoError(t, err)
	diagnostic, err := choice.BuildDiagnosticTrace([]choice.DiagnosticRecord{{Ordinal: 0}, {Ordinal: 1}, {Ordinal: 2}}, 4096)
	require.NoError(t, err)
	handle := cluster.NodeHandle{Node: "node", Incarnation: 1}
	evidence := &cluster.WASMInspectionEvidence{
		World:        []byte("world"),
		Incarnations: []cluster.WASMInspectionIncarnation{{Handle: handle, TerminalOrder: 10, Streams: cluster.WASMInspectionStreams{Choice: trace.Bytes, Stdout: []byte("earlylate")}}},
		Boundaries: []cluster.WASMInspectionBoundary{
			{Handle: handle, Order: 3, Decision: 0, Trace: 0, WorldBytes: 1, Cuts: []cluster.WASMInspectionCut{{Handle: handle, StdoutBytes: 5}}},
			{Handle: handle, Order: 4, Decision: 1, Trace: 2, WorldBytes: 2, Cuts: []cluster.WASMInspectionCut{{Handle: handle, ChoiceRecords: 2, StdoutBytes: 5}}},
		},
	}
	run := determinism.Run{Workload: "fixture", Seed: 7, Identities: map[string]string{"module": "m"}, Command: []string{"rerun"}, ArtifactDir: "/retained/campaign/success", Incarnations: []determinism.Incarnation{{Instance: "guest", Incarnation: 1, Diagnostic: diagnostic.Bytes}}}
	got, err := fromEvidence(run, evidence)
	require.NoError(t, err)
	require.True(t, got.BoundaryScoped)
	require.Equal(t, uint64(2), got.Boundaries[1].Trace)
	require.Equal(t, uint64(1), got.Boundaries[1].Decision)
	require.Equal(t, "node", got.Incarnations[0].Instance)
	require.Equal(t, diagnostic.Bytes, got.Incarnations[0].Diagnostic)
	require.Equal(t, uint64(3), got.Incarnations[0].Operations[0].At.Shared)
	require.Equal(t, "output:stdout", got.Incarnations[0].Operations[0].Op)

	changed := got
	changed.Incarnations = slices.Clone(got.Incarnations)
	changed.Incarnations[0].Choices = slices.Clone(got.Incarnations[0].Choices)
	changed.Incarnations[0].Choices[2].Record.SiteOffset++
	report, err := determinism.Compare(got, changed)
	require.NoError(t, err)
	require.Equal(t, "diverged", report.Status)
	require.Equal(t, "inspection-boundary", report.Ordering)
	require.NotEqual(t, report.First.Expected, report.First.Actual)
	require.NotNil(t, report.Reproducer)
	require.Equal(t, uint64(0), report.Reproducer.Stop.Stop.Value)
}

func TestInspectionRejectsMalformedMetadata(t *testing.T) {
	t.Parallel()
	_, err := FromInspection(determinism.Run{}, []byte(`{"schema":"unknown"}`))
	require.Error(t, err)
}
