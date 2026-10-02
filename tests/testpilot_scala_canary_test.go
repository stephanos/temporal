//go:build test_dep && integration && canary_harness

package tests

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/protorequire"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tools/canary/authority"
	"go.temporal.io/server/tools/canary/casebinding"
	"go.temporal.io/server/tools/canary/policy"
	"go.temporal.io/server/tools/canary/preflight"
	"go.temporal.io/server/tools/canary/testharness"
	"go.temporal.io/server/tools/umpire/recordedrun"
	"google.golang.org/protobuf/proto"
)

func TestTestpilotScalaActivitySharedWithCanary(t *testing.T) {
	env := scalaActivityEnvironment(t)
	functional := scalaActivityFixture(t, "completion")
	canary := scalaActivityFixture(t, "completion")
	require.Equal(t, functional.Bytes, canary.Bytes)
	identity, err := recordedrun.CaseIdentity(functional.Bytes)
	require.NoError(t, err)
	pinned := casebinding.Case()
	lives := make([]testpilotLiveCase, 2)
	for i := range lives {
		name := fmt.Sprintf("scala-shared-%d-%s", i, uuid.NewString())
		lives[i] = bindCase(t, env, functional.Source, CaseBinding{Identity: name, Namespace: name, TaskQueue: name})
		if i == 0 {
			continue
		}
		coordinates := authority.Coordinates{GRPC: env.FrontendGRPCAddress(), Namespace: name, TaskQueue: name, HandlerQueue: name + "-handler", NexusEndpoint: name + "-endpoint"}
		canaryPolicy, err := policy.Embedded()
		require.NoError(t, err)
		canaryPolicy.AuthorityClass, canaryPolicy.EvaluationProfile = policy.AuthorityHarness, testharness.ProfileName
		canaryPolicy.CaseIdentity, canaryPolicy.CaseProfile = identity, lives[i].profile.Identity
		canaryPolicy.Coordinates = coordinates.Digests()
		values := map[string]string{
			preflight.VariableEventName: "workflow_dispatch", preflight.VariableRepository: canaryPolicy.Repository,
			preflight.VariableRef:         canaryPolicy.TrustedRef,
			preflight.VariableWorkflowRef: canaryPolicy.Repository + "/" + canaryPolicy.WorkflowPath + "@" + canaryPolicy.TrustedRef,
			preflight.VariableRunID:       "123", preflight.VariableRunAttempt: "1",
		}
		scope, err := testharness.PrepareCase(t.Context(), preflight.Input{
			Policy: canaryPolicy, Lookup: func(key string) (string, bool) { value, ok := values[key]; return value, ok },
			Coordinates: coordinates, Redactor: coordinates.Redactor(), Namespaces: env.FrontendClient(),
		}, canary.Bytes, lives[i].profile)
		require.NoError(t, err)
		lives[i].prepared = scope.Prepared
	}
	require.NotEqual(t, lives[0].prepared.Identity().Bindings, lives[1].prepared.Identity().Bindings)
	protorequire.ProtoEqual(t, lives[0].prepared.Snapshot(), lives[1].prepared.Snapshot())
	for _, live := range lives {
		for i, entry := range live.prepared.Snapshot().GetProgram().GetEntrypoints() {
			if entry.GetActivity() == nil {
				continue
			}
			want, err := proto.MarshalOptions{Deterministic: true}.Marshal(functional.Source.GetProgram().GetEntrypoints()[i])
			require.NoError(t, err)
			got, err := proto.MarshalOptions{Deterministic: true}.Marshal(entry)
			require.NoError(t, err)
			require.Equal(t, want, got)
		}
	}
	ids, learned := map[string]bool{}, map[string]bool{}
	for range 2 {
		results := runScalaCases(t, env, functional, lives)
		for i, result := range results {
			requireScalaAssessment(t, functional, lives[i], result, "completes", testpilot.PropertySatisfied)
			activityRun := requireScalaActivity(t, env, lives[i], result, []testpilotspb.ActivityAttemptResponse{testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED})
			require.NotContains(t, ids, result.run.GetRunId())
			require.NotContains(t, learned, activityRun)
			ids[result.run.GetRunId()], learned[activityRun] = true, true
		}
	}
	require.Equal(t, pinned, casebinding.Case())
}
