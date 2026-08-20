package canary

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/evidence"
	umpire3runtime "go.temporal.io/server/tests/umpire3/execution"
	"go.temporal.io/server/tests/umpire3/profile"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func TestCanaryRejectsUnapprovedDigestBeforeWorkerExecution(t *testing.T) {
	t.Parallel()

	request := canaryRequest(t, "success")
	request.Approval.ExperimentDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	request.Approval, _ = Seal(request.Approval)
	_, err := (Controller{Store: NewMemoryStore()}).Run(context.Background(), request)
	require.ErrorIs(t, err, ErrUnsafeRequest)
}

func TestCanaryTerminatesBlockingExecutionAndStillCleans(t *testing.T) {
	t.Parallel()

	for _, phase := range []string{"prepare", "execute", "worker-wait", "observe"} {
		t.Run(phase, func(t *testing.T) {
			request := canaryRequest(t, "block-"+phase)
			started := time.Now()
			result, err := (Controller{Store: NewMemoryStore()}).Run(context.Background(), request)
			require.NoError(t, err)
			require.Less(t, time.Since(started), 2*time.Second)
			require.Equal(t, "deadline", result.PrimaryFailure)
			require.Empty(t, result.CleanupFailure)
			require.False(t, result.Complete)
		})
	}
}

func TestCanaryTerminatesBlockingCleanupAndRetainsRecovery(t *testing.T) {
	t.Parallel()

	store := NewMemoryStore()
	request := canaryRequest(t, "block-cleanup")
	result, err := (Controller{Store: store}).Run(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, "deadline", result.CleanupFailure)
	require.True(t, result.Recovery.CleanupPending)
	retained, err := store.Load(context.Background(), request.Approval.Identifier)
	require.NoError(t, err)
	require.True(t, retained.CleanupPending)
}

func TestCanaryResumesCleanupFromPersistedRecovery(t *testing.T) {
	t.Parallel()

	request := canaryRequest(t, "success")
	store := NewMemoryStore()
	recovery := RecoveryRecord{
		FormatVersion: FormatVersion, ApprovalID: request.Approval.Identifier,
		ApprovalDigest:   request.Approval.ApprovalDigest,
		ExperimentDigest: request.Approval.ExperimentDigest,
		Namespace:        request.Approval.Namespace, Tenant: request.Approval.Tenant,
		Resources: map[string]string{"namespace": request.Approval.Namespace}, CleanupPending: true,
	}
	require.NoError(t, store.Save(context.Background(), recovery))
	result, err := (Controller{Store: store}).ResumeCleanup(context.Background(),
		request.Profile, request.Approval, request.WorkerEnvironment)
	require.NoError(t, err)
	require.True(t, result.Complete)
	_, err = store.Load(context.Background(), request.Approval.Identifier)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestCanaryAcceptsQualifiedEvidenceWithSameSemanticDigest(t *testing.T) {
	t.Parallel()

	request := canaryRequest(t, "success")
	result, err := (Controller{Store: NewMemoryStore()}).Run(context.Background(), request)
	require.NoError(t, err)
	require.True(t, result.Complete)
	require.Equal(t, request.Approval.ExperimentDigest, result.Runtime.ExperimentDigest)
	require.Equal(t, umpire3runtime.ClaimConforming, result.Runtime.Claim.Kind)
}

func TestCanaryStopsOnEvidenceLoss(t *testing.T) {
	t.Parallel()

	request := canaryRequest(t, "evidence-loss")
	result, err := (Controller{Store: NewMemoryStore()}).Run(context.Background(), request)
	require.NoError(t, err)
	require.False(t, result.Complete)
	require.Equal(t, "evidence loss or omission", result.StopReason)
}

func TestCanaryRejectsActionOutsideImmutableAllowlist(t *testing.T) {
	t.Parallel()

	request := canaryRequest(t, "success")
	request.Approval.AllowedActions = request.Approval.AllowedActions[:1]
	request.Approval, _ = Seal(request.Approval)
	_, err := (Controller{Store: NewMemoryStore()}).Run(context.Background(), request)
	require.ErrorIs(t, err, ErrUnsafeRequest)
}

func TestCanaryArtifactsDoNotContainWorkerSecrets(t *testing.T) {
	t.Parallel()

	request := canaryRequest(t, "success")
	request.WorkerEnvironment = append(request.WorkerEnvironment, "UMPIRE3_CANARY_SECRET=top-secret")
	result, err := (Controller{Store: NewMemoryStore()}).Run(context.Background(), request)
	require.NoError(t, err)
	encoded, err := json.Marshal(result)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "top-secret")
}

func TestFileStorePersistsRecoveryWithProtectedFile(t *testing.T) {
	t.Parallel()

	store := NewFileStore(t.TempDir())
	record := RecoveryRecord{
		FormatVersion: FormatVersion, ApprovalID: "approved", ApprovalDigest: "digest",
		Resources: map[string]string{"namespace": "isolated"}, CleanupPending: true,
	}
	require.NoError(t, store.Save(context.Background(), record))
	loaded, err := store.Load(context.Background(), "approved")
	require.NoError(t, err)
	require.Equal(t, record, loaded)
}

func TestCanaryWorker(t *testing.T) {
	mode := os.Getenv("UMPIRE3_CANARY_MODE")
	if mode == "" {
		return
	}
	input, err := io.ReadAll(os.Stdin)
	require.NoError(t, err)
	var request WorkerRequest
	require.NoError(t, json.Unmarshal(input, &request))
	if ((mode == "block-prepare" || mode == "block-execute" || mode == "block-worker-wait" || mode == "block-observe") &&
		request.Operation == OperationExecute) ||
		(mode == "block-cleanup" && request.Operation == OperationCleanup) {
		for {
			runtime.Gosched()
		}
	}
	response := WorkerResponse{FormatVersion: FormatVersion, CleanupComplete: true}
	if request.Operation == OperationExecute {
		digest, digestErr := request.Experiment.Digest()
		require.NoError(t, digestErr)
		fact := evidence.Fact{
			Identifier: "fact", Kind: request.Experiment.Checkpoints[0].Observation, Value: true,
			SourceIdentity: "production-history", ClockDomain: "production-history-sequence",
			SourceSequence: 1, ObservedAtUnixNano: time.Now().UnixNano(), Reference: "event/1",
			EntityIdentity: "entity", Lineage: []string{request.Approval.Namespace, "entity"},
		}
		facts := []evidence.Fact{fact}
		if mode == "evidence-loss" {
			facts = nil
		}
		response.Result = umpire3runtime.Result{
			FormatVersion: umpire3runtime.ResultFormatVersion, ExperimentDigest: digest,
			ResultClass: protocol.ResultClassImplementationConforming,
			TrustBadge:  protocol.TrustBadgeTestedInstance,
			Environment: umpire3runtime.EnvironmentIdentity{
				BuildID:               request.Profile.Environment.BuildID,
				ConfigurationIdentity: request.Profile.Environment.ConfigurationIdentity,
			},
			Claim: umpire3runtime.Claim{
				Kind: umpire3runtime.ClaimConforming, Property: request.Experiment.Property.Identifier,
			},
			Evidence: evidence.Graph{
				FormatVersion: evidence.FormatVersion, Facts: facts,
				Claims: []evidence.Claim{{
					Property: request.Experiment.Property.Identifier,
					Verdict:  string(umpire3runtime.ClaimConforming),
				}},
			},
		}
		response.Resources = map[string]string{"workflow": "workflow-id"}
	}
	encoded, err := json.Marshal(response)
	require.NoError(t, err)
	_, err = os.Stdout.Write(encoded)
	require.NoError(t, err)
	//nolint:revive // The helper process must not append the Go test runner's PASS output to its protocol response.
	os.Exit(0)
}

func canaryRequest(t *testing.T, mode string) Request {
	t.Helper()
	file, err := os.Open("../testdata/update-lifecycle.json")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, file.Close()) })
	experiment, err := protocol.DecodeExperiment(file, protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	config := profile.Canary("https://temporal.example", "token", "build", "canary-namespace",
		"canary-task-queue", []string{os.Args[0], "-test.run=TestCanaryWorker"})
	definition, err := profile.Define(config)
	require.NoError(t, err)
	profileDigest, err := definition.Digest()
	require.NoError(t, err)
	experimentDigest, err := experiment.Digest()
	require.NoError(t, err)
	allowedActions := make([]string, len(experiment.Actions))
	for index, action := range experiment.Actions {
		allowedActions[index] = action.Kind
	}
	approval, err := Seal(Approval{
		Identifier: "approval-" + mode, ApproverIdentity: "release-controller",
		ExperimentDigest: experimentDigest, CatalogDigest: experiment.Model.CatalogHash,
		ProfileDigest: profileDigest, Tenant: "canary-tenant", Namespace: definition.Namespace,
		Mode: ModeSafeWrite, AllowedActions: allowedActions, AllowWrites: true,
		MaxActions: len(experiment.Actions), MaxFaults: len(experiment.Faults),
		MaxConcurrent: 1, MaxRatePerSecond: 1, MaxDuration: 100 * time.Millisecond,
		CleanupTimeout: 100 * time.Millisecond, MaxEvidenceBytes: 1 << 20, MaxOutputBytes: 2 << 20,
	})
	require.NoError(t, err)
	return Request{
		Experiment: experiment, Profile: definition, Approval: approval,
		WorkerEnvironment: []string{"UMPIRE3_CANARY_MODE=" + mode},
	}
}
