package protocol

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const validVeilConcreteResult = `{
  "formatVersion": "umpire3/backend-result/v1",
  "backend": "veil",
  "backendRevision": "300c305e945750ab3fb62de4a79c23161b24da39",
  "viewFormatVersion": "umpire3/first-order-view/v1",
  "target": "nexus-cancellation",
  "property": "nexus.cancellation.won-excludes-success",
  "world": "smoke",
  "variant": "sound",
  "semanticHash": "sha256:0000000000000000000000000000000000000000000000000000000000000000",
  "job": "concrete",
  "resultClass": "external-no-counterexample",
  "trustBadge": "tested-instance",
  "exact": false,
  "termination": "exhausted-instance",
  "bounds": {"concreteStateLimit": 512},
  "exploredStates": 26,
  "options": ["sequential"],
  "axioms": [],
  "omissions": ["veil-concrete-fingerprint-collisions-not-ruled-out"]
}`

func TestDecodeBackendResultPreservesCollisionQualifiedConcreteResult(t *testing.T) {
	result, err := DecodeBackendResult(strings.NewReader(validVeilConcreteResult), DefaultDecodeLimit)
	require.NoError(t, err)
	require.Equal(t, BackendResult{
		FormatVersion:     BackendResultFormatVersion,
		Backend:           BackendVeil,
		BackendRevision:   "300c305e945750ab3fb62de4a79c23161b24da39",
		ViewFormatVersion: FirstOrderViewFormatVersion,
		Target:            TargetIDNexusCancellation,
		Property:          PropertyIDNexusCancellationWonExcludesSuccess,
		World:             "smoke",
		Variant:           "sound",
		SemanticHash:      "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		Job:               BackendJobConcrete,
		ResultClass:       ResultClassExternalNoCounterexample,
		TrustBadge:        TrustBadgeTestedInstance,
		Exact:             false,
		Termination:       BackendTerminationExhaustedInstance,
		Bounds:            BackendBounds{ConcreteStateLimit: 512},
		ExploredStates:    26,
		Options:           []string{"sequential"},
		Axioms:            []string{},
		Omissions:         []string{VeilConcreteCollisionOmission},
	}, result)
}

func TestBackendResultRejectsUnearnedConcreteCompleteness(t *testing.T) {
	for name, changed := range map[string]string{
		"finite exhaustive": strings.Replace(validVeilConcreteResult,
			`"resultClass": "external-no-counterexample"`, `"resultClass": "finite-exhaustive"`, 1),
		"exact": strings.Replace(validVeilConcreteResult, `"exact": false`, `"exact": true`, 1),
		"missing collision qualification": strings.Replace(validVeilConcreteResult,
			`["veil-concrete-fingerprint-collisions-not-ruled-out"]`, `[]`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := DecodeBackendResult(strings.NewReader(changed), DefaultDecodeLimit)
			require.ErrorContains(t, err, "concrete no-counterexample")
		})
	}
}

func TestBackendResultRequiresCheckedCanonicalReplayForTraceWitness(t *testing.T) {
	result := BackendResult{
		FormatVersion: BackendResultFormatVersion, Backend: BackendVeil,
		BackendRevision:   "300c305e945750ab3fb62de4a79c23161b24da39",
		ViewFormatVersion: FirstOrderViewFormatVersion,
		Target:            TargetIDNexusCancellation, Property: PropertyIDNexusCancellationWonExcludesSuccess,
		World: "smoke", Variant: "stale-completion-guard-removed",
		SemanticHash: "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		Job:          BackendJobConcrete, ResultClass: ResultClassTraceWitness,
		TrustBadge: TrustBadgeCheckedCertificate, Exact: true,
		Termination: BackendTerminationViolationFound,
		Bounds:      BackendBounds{ConcreteStateLimit: 512}, ExploredStates: 17,
		Options:   []string{"sequential"},
		Axioms:    []string{"Classical.choice", "Quot.sound", "propext"},
		Omissions: []string{VeilTraceStateOmission},
		Trace: &ModelTrace{
			World: "smoke", Property: PropertyIDNexusCancellationWonExcludesSuccess,
			Violation: true,
			Steps: []TraceStep{
				{Action: ActionKindDispatchTask},
				{Action: ActionKindAcquireOwnership},
				{Action: ActionKindWorkerReturnsSuccess},
				{Action: ActionKindPersistSuccess},
			},
			Assumptions: []string{}, Bounds: BackendBounds{ConcreteStateLimit: 512},
			SourceMap: []TraceSource{
				{Action: ActionKindDispatchTask, BackendAction: "DispatchTask"},
				{Action: ActionKindAcquireOwnership, BackendAction: "AcquireOwnership"},
				{Action: ActionKindWorkerReturnsSuccess, BackendAction: "WorkerReturnsSuccess"},
				{Action: ActionKindPersistSuccess, BackendAction: "PersistSuccess"},
			},
			Replay: TraceReplayResult{Status: TraceReplayAccepted,
				TrustBadge: TrustBadgeCheckedCertificate,
				Axioms:     []string{"Classical.choice", "Quot.sound", "propext"}},
		},
	}
	require.NoError(t, result.Validate())

	result.Trace.Replay.Status = TraceReplayRequired
	require.ErrorContains(t, result.Validate(), "accepted canonical replay")

	result.Trace.Replay.Status = TraceReplayAccepted
	result.Trace.Steps[0].Action = ActionKind("unknown-action")
	require.ErrorContains(t, result.Validate(), "unknown trace action")

	result.Trace.Steps[0].Action = ActionKindDispatchTask
	result.Axioms = []string{}
	require.ErrorContains(t, result.Validate(), "axiom inventory")

	result.Axioms = []string{"Classical.choice", "Quot.sound", "propext"}
	result.Omissions = []string{}
	require.ErrorContains(t, result.Validate(), "state digest omission")
}

func TestDecodeBackendResultRejectsUnknownAndTrailingInput(t *testing.T) {
	unknown := strings.Replace(validVeilConcreteResult, `"world": "smoke",`,
		`"world": "smoke", "unknown": true,`, 1)
	_, err := DecodeBackendResult(strings.NewReader(unknown), DefaultDecodeLimit)
	require.ErrorContains(t, err, "unknown field")

	_, err = DecodeBackendResult(strings.NewReader(validVeilConcreteResult+` {}`), DefaultDecodeLimit)
	require.ErrorContains(t, err, "multiple JSON values")
}

func TestTraceReplayInputHasStableBoundDigest(t *testing.T) {
	input := TraceReplayInput{
		FormatVersion: TraceReplayInputFormatVersion,
		Target:        TargetIDNexusCancellation,
		Property:      PropertyIDNexusCancellationWonExcludesSuccess,
		World:         "smoke",
		Variant:       "stale-completion-guard-removed",
		SemanticHash:  "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		Actions: []ActionKind{
			ActionKindDispatchTask,
			ActionKindAcquireOwnership,
			ActionKindWorkerReturnsSuccess,
			ActionKindPersistSuccess,
		},
	}
	canonical, err := input.CanonicalJSON()
	require.NoError(t, err)
	require.Equal(t,
		`{"formatVersion":"umpire3/trace-replay-input/v1","target":"nexus-cancellation","property":"nexus.cancellation.won-excludes-success","world":"smoke","variant":"stale-completion-guard-removed","semanticHash":"sha256:0000000000000000000000000000000000000000000000000000000000000000","actions":["dispatch-task","acquire-ownership","worker-returns-success","persist-success"]}`,
		string(canonical))
	digest, err := input.Digest()
	require.NoError(t, err)
	require.Equal(t, "sha256:acab4c8de082939fc9b38f80063a6c19687893f46c5c16245d060b3804e53c76", digest)
}

func TestDecodeTraceReplayReceiptRequiresAcceptedBoundCertificate(t *testing.T) {
	receiptJSON := `{
  "axioms": [],
  "formatVersion": "umpire3/trace-replay-receipt/v1",
  "status": "accepted",
  "traceDigest": "sha256:acab4c8de082939fc9b38f80063a6c19687893f46c5c16245d060b3804e53c76",
  "trustBadge": "checked-certificate"
}`
	receipt, err := DecodeTraceReplayReceipt(strings.NewReader(receiptJSON), DefaultDecodeLimit)
	require.NoError(t, err)
	require.Equal(t, TraceReplayReceipt{
		FormatVersion: TraceReplayReceiptFormatVersion,
		TraceDigest:   "sha256:acab4c8de082939fc9b38f80063a6c19687893f46c5c16245d060b3804e53c76",
		Status:        TraceReplayAccepted,
		TrustBadge:    TrustBadgeCheckedCertificate,
		Axioms:        []string{},
	}, receipt)

	rejected := bytes.ReplaceAll([]byte(receiptJSON), []byte(`"accepted"`), []byte(`"rejected"`))
	_, err = DecodeTraceReplayReceipt(bytes.NewReader(rejected), DefaultDecodeLimit)
	require.ErrorContains(t, err, "accepted checked-certificate")
}
