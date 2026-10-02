//go:build test_dep && integration

package tests

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/campaign"
	"go.temporal.io/server/common/testing/testpilot/recordedrun"
	"go.temporal.io/server/common/testing/testpilot/temporal/binding"
	"go.temporal.io/server/common/testing/testpilot/temporal/provision"
	"go.temporal.io/server/model/scalav2/explore"
	"go.temporal.io/server/model/scalav2/goir"
	"google.golang.org/protobuf/encoding/protojson"
)

func writeExplorationArtifact(t *testing.T, name string, data []byte) {
	t.Helper()
	root := os.Getenv("UMPIRE_EXPLORATION_DIR")
	if root == "" {
		return
	}
	require.NoError(t, os.MkdirAll(root, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(root, name), data, 0600))
}

func TestTestpilotScalaExplorationDiscoversUnpinnedExecution(t *testing.T) {
	env := newTestpilotTestEnvironment(t)
	modelRoot, err := filepath.Abs(filepath.Join("..", "model", "scalav2"))
	require.NoError(t, err)
	model, err := goir.Load(filepath.Join(modelRoot, "ir", "nexus-caller.json"))
	require.NoError(t, err)
	plan, err := explore.New(model, "nexusDeadlines")
	require.NoError(t, err)
	require.Len(t, plan.Candidates, 3)
	require.Equal(t, "startDeadline", plan.Candidates[0].Key)
	for _, c := range plan.Candidates {
		require.Empty(t, c.Rejection)
	}
	// An execution's action sequence, rather than its generated name, establishes novelty.
	chosen := plan.Candidates[0]
	realizer, err := goir.NewRealizer(model, goir.DefaultScope)
	require.NoError(t, err)
	chosenKeys := []string{}
	for _, a := range chosen.Actions {
		chosenKeys = append(chosenKeys, realizer.ClassKey(a))
	}
	pinned, err := filepath.Glob(filepath.Join("testcore", "testpilot", "testdata", "*-case.json"))
	require.NoError(t, err)
	generated, err := filepath.Glob(filepath.Join(modelRoot, "cases", "*-case.json"))
	require.NoError(t, err)
	var scenarioFingerprint string
	for _, definition := range chosen.Case.GetProvenance().GetDefinitions() {
		if definition.GetKind() == testpilotspb.DEFINITION_KIND_SCENARIO {
			scenarioFingerprint = definition.GetBehaviorFingerprint()
		}
	}
	require.NotEmpty(t, scenarioFingerprint)
	for _, path := range append(pinned, generated...) {
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		require.NotEqual(t, string(chosen.Bytes), string(data), path)
		var existing testpilotspb.Case
		require.NoError(t, protojson.Unmarshal(data, &existing))
		for _, definition := range existing.GetProvenance().GetDefinitions() {
			if definition.GetKind() == testpilotspb.DEFINITION_KIND_SCENARIO {
				require.NotEqual(t, scenarioFingerprint, definition.GetBehaviorFingerprint(), "pinned Scenario: %s", path)
			}
		}
	}
	for _, scenario := range model.Scenarios {
		keys := []string{}
		for _, a := range scenario.Actions {
			keys = append(keys, realizer.ClassKey(a))
		}
		require.NotEqual(t, chosenKeys, keys, "already pinned Scenario %s", scenario.Name)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	deployment := binding.Deployment{GRPCAddress: env.FrontendGRPCAddress(), HTTPAddress: env.HttpAPIAddress(), Namespace: "umpire-scala-discovery", TaskQueue: "umpire-scala-discovery", NexusEndpoint: "umpire-scala-discovery"}
	release, err := provision.Create(env.Context(), provision.Clients{Workflow: env.FrontendClient(), Operator: env.OperatorClient()}, provision.Resources{Namespace: deployment.Namespace, TaskQueue: deployment.TaskQueue, NexusEndpoint: deployment.NexusEndpoint, NexusTaskQueue: binding.HandlerQueue(deployment), RetainNamespace: true})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, release(context.Background())) })
	opened, err := binding.Open(ctx, deployment, binding.HandlerQueue(deployment))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, opened.Close(context.Background())) })
	bridge, err := campaign.Start(ctx, campaign.Options{Executable: buildUmpireCommand(t, "umpire-ir-bridge"), Dir: modelRoot})
	require.NoError(t, err)
	defer func() { require.NoError(t, bridge.Close()) }()
	initialized, err := bridge.Initialize(ctx, "nexusDeadlines", "scala-discovery")
	require.NoError(t, err)
	require.Equal(t, []string{"startDeadline", "scheduleDeadline", "unbounded"}, initialized.Targets)
	next, err := bridge.Next(ctx)
	require.NoError(t, err)
	require.NotNil(t, next.Candidate)
	require.Equal(t, chosen.Bytes, next.Candidate.Case)
	outcome, err := campaign.RunCandidate(ctx, bridge, campaign.CampaignBinder{Campaign: opened}, next.Candidate)
	require.NoError(t, err)
	require.NoError(t, outcome.ReleaseError)
	require.NoError(t, outcome.RunError)
	require.Equal(t, []string{"startDeadline"}, outcome.Credited.Credited)
	next, err = bridge.Next(ctx)
	require.NoError(t, err)
	require.True(t, next.Exhausted, "the declared one-Run budget applies")
	finished, err := bridge.Finish(ctx, "")
	require.NoError(t, err)
	require.Equal(t, "limit-reached", finished.Status)
	require.Equal(t, campaign.Summary{Targets: 3, Selected: 1, Covered: 1, Pending: 2}, finished.Summary)
	require.Empty(t, finished.Counterexamples, "discovery never promotes a single Run")
	sourceRoot, err := filepath.Abs("..")
	require.NoError(t, err)
	trace, err := explore.RenderTrace(chosen, plan.Query, outcome.Run, nil, sourceRoot)
	require.NoError(t, err)
	recordPath := filepath.Join(t.TempDir(), "run.json")
	require.NoError(t, recordedrun.Write(recordPath, chosen.Bytes, outcome.Driver, outcome.Run))
	record, err := os.ReadFile(recordPath)
	require.NoError(t, err)
	report, err := json.MarshalIndent(struct {
		Enumeration *explore.Plan     `json:"exactFiniteEnumeration"`
		Runtime     campaign.Finished `json:"sampledRuntimeCoverage"`
		Schedule    string            `json:"scheduleGuarantee"`
	}{plan, finished, "black-box repetition; no fixed runtime schedule"}, "", "  ")
	require.NoError(t, err)
	writeExplorationArtifact(t, "coverage.json", report)
	writeExplorationArtifact(t, "discovered-case.json", chosen.Bytes)
	writeExplorationArtifact(t, "discovered-run.json", record)
	writeExplorationArtifact(t, "discovered-trace.html", trace)
}
