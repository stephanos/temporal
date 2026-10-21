package simulation

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	"go.temporal.io/server/tools/gomad/internal/model/cluster"
)

func inspectionSearchFixture(t *testing.T) *cluster.WASMExecutionEvidence {
	t.Helper()
	handle := cluster.NodeHandle{Node: "node", Incarnation: 2}
	a, b := sha256.Sum256([]byte("a")), sha256.Sum256([]byte("b"))
	var records []choice.Record
	for i := uint64(0); i < 5; i++ {
		d, err := choice.CanonicalDecision(i, choice.KindRunnable, 10, false, [][32]byte{a, b}, a, 0)
		require.NoError(t, err)
		records = append(records, d.Record())
	}
	trace, err := choice.BuildTrace(records, choice.TerminalComplete)
	require.NoError(t, err)
	metadata := &cluster.WASMInspectionEvidence{Schema: cluster.WASMInspectionSchema, WorldEncoding: cluster.WASMInspectionWorldEncoding, Complete: true, World: []byte(cluster.WASMInspectionWorldHeader), Lifecycle: []byte(cluster.WASMInspectionLifecycleHeader), Incarnations: []cluster.WASMInspectionIncarnation{{Handle: handle, TerminalOrder: 10, Registry: cluster.WASMInspectionRegistry{Closed: true, Entries: []cluster.WASMInspectionRegistration{{Kind: "check", Name: "healthy", Monotonic: true}}}, Streams: cluster.WASMInspectionStreams{Choice: trace.Bytes}}}}
	for i := uint64(0); i < 5; i++ {
		choiceBytes, err := choice.TracePayloadBytes(i)
		require.NoError(t, err)
		metadata.World = binary.LittleEndian.AppendUint64(metadata.World, sha256.Size)
		world := sha256.Sum256([]byte(fmt.Sprintf("World boundary %d", i)))
		metadata.World = append(metadata.World, world[:]...)
		metadata.Lifecycle = append(metadata.Lifecycle, 2)
		metadata.Lifecycle = binary.LittleEndian.AppendUint32(metadata.Lifecycle, uint32(len(handle.Node)))
		metadata.Lifecycle = append(metadata.Lifecycle, string(handle.Node)...)
		metadata.Lifecycle = binary.LittleEndian.AppendUint64(metadata.Lifecycle, handle.Incarnation)
		witness := &cluster.WASMRuntimeWitness{Goroutines: 2, RuntimeState: make([]byte, 128)}
		binary.BigEndian.PutUint64(witness.RuntimeState[112:], 2)
		metadata.Boundaries = append(metadata.Boundaries, cluster.WASMInspectionBoundary{Handle: handle, Decision: i, Trace: i, Order: i + 1, WorldBytes: uint64(len(metadata.World)), LifecycleBytes: uint64(len(metadata.Lifecycle)), Cuts: []cluster.WASMInspectionCut{{StateEncoding: cluster.WASMInspectionStateEncoding, StorageState: make([]byte, 32), RecoveryState: make([]byte, 32), NetworkState: make([]byte, 32), Handle: handle, ChoiceRecords: i, Witness: witness}}})
		metadata.Boundaries[i].Cuts[0].ChoiceBytes = choiceBytes
	}
	empty := fmt.Sprintf("sha256:%x", sha256.Sum256(nil))
	return &cluster.WASMExecutionEvidence{Inspection: metadata, Incarnations: []cluster.WASMIncarnationEvidence{{Handle: handle, ChoiceTrace: trace.Bytes, WASISHA256: empty, NetworkSHA256: empty, StorageSHA256: empty}}}
}

func TestInspectionSearchScanDoesNotSkipRecoveredFailures(t *testing.T) {
	metadata := inspectionSearchFixture(t)
	request := WASMInspectionSearchRequest{Check: "healthy", First: 0, Last: 4, AttemptBudget: 10, WallTime: time.Second}
	var called []uint64
	result, err := SearchWASMInspection(t.Context(), metadata, request, func(ctx context.Context, resolved cluster.WASMInspectionResolution, check string) (WASMInspectionEvaluation, error) {
		require.Equal(t, "healthy", check)
		require.Equal(t, uint64(2), resolved.Target.Incarnation)
		called = append(called, resolved.Boundary.Decision)
		outcome := "pass"
		if resolved.Boundary.Decision == 1 || resolved.Boundary.Decision == 4 {
			outcome = "fail"
		}
		return WASMInspectionEvaluation{Outcome: outcome, PrefixVerified: true}, nil
	})
	require.NoError(t, err)
	require.Equal(t, "first_failure", result.Status)
	require.Equal(t, []uint64{0, 1}, called)
	require.Equal(t, uint64(1), *result.FirstFailure)
}

func TestInspectionSearchBudgetsAndPredicateErrorsStayInconclusive(t *testing.T) {
	for _, test := range []struct {
		name, outcome, status string
		budget                uint64
		verified              bool
	}{
		{"attempt budget", "pass", "incomplete", 2, true},
		{"panic", "panic", "invariant_evaluation_failure", 10, true},
		{"timeout", "timeout", "invariant_evaluation_failure", 10, true},
		{"divergence", "divergence", "replay_divergence", 10, true},
		{"unverified predicate", "pass", "replay_divergence", 10, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := SearchWASMInspection(t.Context(), inspectionSearchFixture(t), WASMInspectionSearchRequest{Check: "healthy", Last: 4, AttemptBudget: test.budget, WallTime: time.Second}, func(context.Context, cluster.WASMInspectionResolution, string) (WASMInspectionEvaluation, error) {
				return WASMInspectionEvaluation{Outcome: test.outcome, PrefixVerified: test.verified}, nil
			})
			require.NoError(t, err)
			require.Equal(t, test.status, result.Status)
			require.Nil(t, result.FirstFailure)
			require.NotNil(t, result.Remaining)
		})
	}
}

func TestInspectionSearchBisectDeclaresConditionalityAndRejectsObservedReversal(t *testing.T) {
	for _, test := range []struct {
		name   string
		values []bool
		status string
		first  *uint64
	}{
		{"transition", []bool{true, true, false, false, false}, "first_failure", searchUint64(2)},
		{"already failing", []bool{false, false, false, false, false}, "already_failing_at_first", searchUint64(0)},
		{"never failing", []bool{true, true, true, true, true}, "no_failure", nil},
		{"observed reversal", []bool{false, true, false, false, true}, "monotonicity_violation", nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := SearchWASMInspection(t.Context(), inspectionSearchFixture(t), WASMInspectionSearchRequest{Check: "healthy", Last: 4, Mode: "bisect", AttemptBudget: 10, WallTime: time.Second}, func(_ context.Context, resolved cluster.WASMInspectionResolution, _ string) (WASMInspectionEvaluation, error) {
				outcome := "fail"
				if test.values[resolved.Boundary.Decision] {
					outcome = "pass"
				}
				return WASMInspectionEvaluation{Outcome: outcome, PrefixVerified: true}, nil
			})
			require.NoError(t, err)
			require.True(t, result.ConditionalOnMonotonicity)
			require.Equal(t, test.status, result.Status)
			require.Equal(t, test.first, result.FirstFailure)
		})
	}
}

func TestInspectionSearchRejectsRequestsBeforeEvaluation(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*WASMInspectionSearchRequest, *cluster.WASMExecutionEvidence)
	}{
		{"unknown check", func(r *WASMInspectionSearchRequest, _ *cluster.WASMExecutionEvidence) { r.Check = "absent" }},
		{"invalid range", func(r *WASMInspectionSearchRequest, _ *cluster.WASMExecutionEvidence) { r.First = 5 }},
		{"zero attempts", func(r *WASMInspectionSearchRequest, _ *cluster.WASMExecutionEvidence) { r.AttemptBudget = 0 }},
		{"zero wall bound", func(r *WASMInspectionSearchRequest, _ *cluster.WASMExecutionEvidence) { r.WallTime = 0 }},
		{"unreachable choice", func(r *WASMInspectionSearchRequest, _ *cluster.WASMExecutionEvidence) { r.Last = 5 }},
		{"missing monotonic declaration", func(r *WASMInspectionSearchRequest, e *cluster.WASMExecutionEvidence) {
			r.Mode = "bisect"
			e.Inspection.Incarnations[0].Registry.Entries[0].Monotonic = false
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			execution := inspectionSearchFixture(t)
			request := WASMInspectionSearchRequest{Check: "healthy", Last: 4, AttemptBudget: 10, WallTime: time.Second}
			test.change(&request, execution)
			_, err := SearchWASMInspection(t.Context(), execution, request, func(context.Context, cluster.WASMInspectionResolution, string) (WASMInspectionEvaluation, error) {
				t.Fatal("preflight refusal executed replay")
				return WASMInspectionEvaluation{}, nil
			})
			require.Error(t, err)
		})
	}
}

func searchUint64(n uint64) *uint64 { return &n }

func TestInspectionSearchWallBoundIncludesEvaluatorCleanup(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	result, err := SearchWASMInspection(ctx, inspectionSearchFixture(t), WASMInspectionSearchRequest{Check: "healthy", Last: 4, AttemptBudget: 10, WallTime: time.Second}, func(context.Context, cluster.WASMInspectionResolution, string) (WASMInspectionEvaluation, error) {
		cancel()
		return WASMInspectionEvaluation{Outcome: "pass", PrefixVerified: true}, nil
	})
	require.NoError(t, err)
	require.Equal(t, "incomplete", result.Status)
	require.Equal(t, "timeout", result.Evaluations[0].Outcome)
	require.Nil(t, result.FirstFailure)
}

func TestInspectionChoiceMappingUsesDecisionsRatherThanObservations(t *testing.T) {
	execution := inspectionSearchFixture(t)
	old, err := inspectionChoiceRecords(execution.Incarnations[0].ChoiceTrace, 5)
	require.NoError(t, err)
	ready, err := choice.NewReadiness(1, 0)
	require.NoError(t, err)
	records := []choice.Record{old[0], {Ordinal: 1, Kind: choice.KindSelectResult, Flags: choice.FlagObservation, Alternatives: 1, Data: 1, Origin: 1, SiteOffset: 15, Readiness: ready}}
	for _, record := range old[1:] {
		record.Ordinal++
		records = append(records, record)
	}
	trace, err := choice.BuildTrace(records, choice.TerminalComplete)
	require.NoError(t, err)
	execution.Incarnations[0].ChoiceTrace = trace.Bytes
	execution.Inspection.Incarnations[0].Streams.Choice = trace.Bytes
	for i := 1; i < len(execution.Inspection.Boundaries); i++ {
		b := &execution.Inspection.Boundaries[i]
		b.Trace++
		b.Cuts[0].ChoiceRecords = b.Trace
		bytes, err := choice.TracePayloadBytes(b.Trace)
		require.NoError(t, err)
		b.Cuts[0].ChoiceBytes = bytes
	}
	require.NoError(t, ValidateInspectionChoices(execution))
	execution.Inspection.Boundaries[1].Trace = 1
	execution.Inspection.Boundaries[1].Cuts[0].ChoiceRecords = 1
	bytes, err := choice.TracePayloadBytes(1)
	require.NoError(t, err)
	execution.Inspection.Boundaries[1].Cuts[0].ChoiceBytes = bytes
	require.Error(t, ValidateInspectionChoices(execution))
	execution = inspectionSearchFixture(t)
	execution.Inspection.Boundaries = execution.Inspection.Boundaries[:1]
	require.Error(t, ValidateInspectionChoices(execution))
}

func TestInspectionInterruptedSelectPrefixKeepsDecisionMapping(t *testing.T) {
	execution := inspectionSearchFixture(t)
	a, b := sha256.Sum256([]byte("a")), sha256.Sum256([]byte("b"))
	poll, err := choice.CanonicalDecision(0, choice.KindSelectPoll, 10, false, [][32]byte{a, b}, a, 1)
	require.NoError(t, err)
	trace, err := choice.BuildTrace([]choice.Record{poll.Record()}, choice.TerminalComplete)
	require.NoError(t, err)
	execution.Incarnations[0].Interrupted = true
	execution.Incarnations[0].ChoiceTrace = trace.Bytes
	execution.Inspection.Incarnations[0].Streams.Choice = trace.Bytes
	execution.Inspection.Boundaries = execution.Inspection.Boundaries[:1]
	execution.Inspection.World = execution.Inspection.World[:execution.Inspection.Boundaries[0].WorldBytes]
	require.NoError(t, ValidateInspectionChoices(execution))
}

// A v4 runnable decision appends its creation records before the decision
// record, while the helper's boundary keeps the trace length at which the
// decision was presented. The mapping uses that predecision coordinate and
// still refuses a boundary that points at the decision record itself.
func TestInspectionChoiceMappingUsesPredecisionCoordinateForCreationSites(t *testing.T) {
	execution := inspectionSearchFixture(t)
	a, b := sha256.Sum256([]byte("a")), sha256.Sum256([]byte("b"))
	ordered := [][32]byte{a, b}
	if string(b[:]) < string(a[:]) {
		ordered = [][32]byte{b, a}
	}
	var records []choice.Record
	for range 5 {
		for index, identity := range ordered {
			records = append(records, choice.Record{Ordinal: uint64(len(records)), Kind: choice.KindGoroutineCreation, Alternatives: 2, Selected: uint32(index), SiteOffset: 7, SelectedIdentity: identity})
		}
		d, err := choice.CanonicalDecision(uint64(len(records)), choice.KindRunnable, 10, false, [][32]byte{a, b}, a, 0)
		require.NoError(t, err)
		records = append(records, d.Record())
	}
	trace, err := choice.BuildTrace(records, choice.TerminalComplete)
	require.NoError(t, err)
	execution.Incarnations[0].ChoiceTrace = trace.Bytes
	execution.Inspection.Incarnations[0].Streams.Choice = trace.Bytes
	retrace := func(i int, coordinate uint64) {
		b := &execution.Inspection.Boundaries[i]
		b.Trace = coordinate
		b.Cuts[0].ChoiceRecords = coordinate
		bytes, err := choice.TracePayloadBytes(coordinate)
		require.NoError(t, err)
		b.Cuts[0].ChoiceBytes = bytes
	}
	for i := range execution.Inspection.Boundaries {
		retrace(i, uint64(3*i))
	}
	require.NoError(t, ValidateInspectionChoices(execution))
	retrace(1, 5)
	require.Error(t, ValidateInspectionChoices(execution))
}
