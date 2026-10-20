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
	require.Equal(t, "undiagnosable", report.Status)
	require.Equal(t, "inspection-boundary", report.Ordering)
	require.NotEqual(t, report.First.Expected, report.First.Actual)
	require.NotNil(t, report.Reproducer)
	require.Equal(t, uint64(0), report.Reproducer.Stop.Stop.Value)
	for _, test := range []struct {
		name   string
		mutate func(*cluster.WASMInspectionCut)
	}{
		{"recovery state", func(cut *cluster.WASMInspectionCut) { cut.RecoveryState = []byte("changed") }},
		{"runtime witness", func(cut *cluster.WASMInspectionCut) { cut.Witness = &cluster.WASMRuntimeWitness{Allocations: 1} }},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			changed := cluster.CloneWASMInspection(evidence)
			test.mutate(&changed.Boundaries[1].Cuts[0])
			actual, err := fromEvidence(run, changed)
			require.NoError(t, err)
			report, err := determinism.Compare(got, actual)
			require.NoError(t, err)
			require.Equal(t, "undiagnosable", report.Status)
			require.Equal(t, uint64(4), report.First.At.Shared)
		})
	}

	other := cluster.NodeHandle{Node: "other", Incarnation: 1}
	otherTrace, err := choice.BuildTrace([]choice.Record{first.Record()}, choice.TerminalComplete)
	require.NoError(t, err)
	otherDiagnostic, err := choice.BuildDiagnosticTrace([]choice.DiagnosticRecord{{}}, 4096)
	require.NoError(t, err)
	clusterEvidence := cluster.CloneWASMInspection(evidence)
	clusterEvidence.Boundaries[1].Order = 8
	clusterEvidence.Incarnations = append(clusterEvidence.Incarnations, cluster.WASMInspectionIncarnation{Handle: other, StartedOrder: 4, TerminalOrder: 20, Streams: cluster.WASMInspectionStreams{Choice: otherTrace.Bytes}})
	clusterEvidence.Boundaries = slices.Insert(clusterEvidence.Boundaries, 1, cluster.WASMInspectionBoundary{Handle: other, Order: 5, Trace: 0, WorldBytes: 2, Cuts: []cluster.WASMInspectionCut{{Handle: handle, ChoiceRecords: 2, StdoutBytes: 5}, {Handle: other}}})
	clusterRun := run
	clusterRun.Incarnations = []determinism.Incarnation{{Instance: "node", Incarnation: 1, Diagnostic: diagnostic.Bytes}, {Instance: "other", Incarnation: 1, Diagnostic: otherDiagnostic.Bytes}}
	decoded, err := fromEvidence(clusterRun, clusterEvidence)
	require.NoError(t, err)
	require.Equal(t, uint64(8), decoded.Incarnations[0].Choices[2].At.Shared)
	require.Equal(t, uint64(5), decoded.Incarnations[0].Choices[1].At.Shared)
	mutated := cluster.CloneWASMInspection(clusterEvidence)
	mutated.World = append(mutated.World, '!')
	actual, err := fromEvidence(clusterRun, mutated)
	require.NoError(t, err)
	report, err = determinism.Compare(decoded, actual)
	require.NoError(t, err)
	require.Equal(t, uint64(20), report.First.At.Shared)
	require.NotNil(t, report.VirtualTimeAvailable)
	require.False(t, *report.VirtualTimeAvailable)
}

func TestInspectionRejectsMalformedMetadata(t *testing.T) {
	t.Parallel()
	_, err := FromInspection(determinism.Run{}, []byte(`{"schema":"unknown"}`))
	require.Error(t, err)
}

func TestZeroChoiceIncarnationPresenceUsesRetainedStart(t *testing.T) {
	t.Parallel()
	diagnostic, err := choice.BuildDiagnosticTrace(nil, 4096)
	require.NoError(t, err)
	expected := determinism.Run{Workload: "fixture", Identities: map[string]string{"module": "m"}, Command: []string{"rerun"}, BoundaryScoped: true}
	actual := expected
	actual.Incarnations = []determinism.Incarnation{{Instance: "late", Incarnation: 1, Started: &determinism.Coordinate{Shared: 100}, Diagnostic: diagnostic.Bytes}}
	report, err := determinism.Compare(expected, actual)
	require.NoError(t, err)
	require.Equal(t, uint64(100), report.First.At.Shared)
	require.NotNil(t, report.VirtualTimeAvailable)
	require.False(t, *report.VirtualTimeAvailable)
}
