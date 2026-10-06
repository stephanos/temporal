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
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/campaign"
	"go.temporal.io/server/common/testing/testpilot/recordedrun"
	"go.temporal.io/server/common/testing/testpilot/temporal/binding"
	"go.temporal.io/server/common/testing/testpilot/temporal/provision"
	"go.temporal.io/server/tools/umpire/check"
	"go.temporal.io/server/tools/umpire/explore"
	"go.temporal.io/server/tools/umpire/ir"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
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

func TestTestpilotExplorationDiscoversUnpinnedExecution(t *testing.T) {
	env := newTestpilotTestEnvironment(t)
	modelRoot, err := filepath.Abs(filepath.Join("..", "model"))
	require.NoError(t, err)
	model, err := ir.Load(filepath.Join(modelRoot, "ir", "nexus-caller.json"))
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
	realizer, err := check.NewRealizer(model, check.DefaultScope)
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
	binder := &keptRunBinder{binder: campaign.CampaignBinder{Campaign: opened}}
	report, err := campaign.Drive(ctx, bridge, binder, campaign.Caps{}, nil)
	require.NoError(t, err)
	require.Equal(t, campaign.StatusExhausted, report.Terminal.Status, "the declared one-Run budget applies")
	require.Len(t, report.Outcomes, 1)
	require.Equal(t, campaign.OutcomeCompleted, report.Outcomes[0].Kind)
	require.Equal(t, []string{"startDeadline"}, report.Outcomes[0].Credited)
	require.True(t, proto.Equal(chosen.Case, binder.source), "the bridge hands out the enumerated Case")
	require.NoError(t, binder.releaseErr)
	require.NoError(t, binder.runErr)
	require.NotNil(t, report.Finished)
	finished := *report.Finished
	require.Equal(t, "limit-reached", finished.Status)
	require.Equal(t, campaign.Summary{Targets: 3, Selected: 1, Covered: 1, Pending: 2}, finished.Summary)
	require.Empty(t, finished.Counterexamples, "discovery never promotes a single Run")
	sourceRoot, err := filepath.Abs("..")
	require.NoError(t, err)
	trace, err := explore.RenderTrace(chosen, plan.Query, binder.run, nil, sourceRoot)
	require.NoError(t, err)
	recordPath := filepath.Join(t.TempDir(), "run.json")
	require.NoError(t, recordedrun.Write(recordPath, chosen.Bytes, binder.driver, binder.run))
	record, err := os.ReadFile(recordPath)
	require.NoError(t, err)
	coverage, err := json.MarshalIndent(struct {
		Enumeration *explore.Plan     `json:"exactFiniteEnumeration"`
		Runtime     campaign.Finished `json:"sampledRuntimeCoverage"`
		Schedule    string            `json:"scheduleGuarantee"`
	}{plan, finished, "black-box repetition; no fixed runtime schedule"}, "", "  ")
	require.NoError(t, err)
	writeExplorationArtifact(t, "coverage.json", coverage)
	writeExplorationArtifact(t, "discovered-case.json", chosen.Bytes)
	writeExplorationArtifact(t, "discovered-run.json", record)
	writeExplorationArtifact(t, "discovered-trace.html", trace)
}

// keptRunBinder keeps the one Run a campaign drives, which the campaign's report does not retain.
type keptRunBinder struct {
	binder     campaign.Binder
	source     *testpilotspb.Case
	driver     testpilot.DriverIdentity
	run        *testpilotspb.Run
	runErr     error
	releaseErr error
}

func (b *keptRunBinder) Bind(ctx context.Context, identity string, source *testpilotspb.Case) (campaign.Bound, error) {
	b.source = source
	bound, err := b.binder.Bind(ctx, identity, source)
	if err != nil {
		return nil, err
	}
	return &keptRun{Bound: bound, kept: b}, nil
}

type keptRun struct {
	campaign.Bound
	kept *keptRunBinder
}

func (r *keptRun) Run(ctx context.Context) (*testpilotspb.Run, *testpilotspb.Verdict, error) {
	run, verdict, err := r.Bound.Run(ctx)
	r.kept.driver, r.kept.run, r.kept.runErr = r.Bound.Identity(), run, err
	return run, verdict, err
}

func (r *keptRun) Release(ctx context.Context) error {
	r.kept.releaseErr = r.Bound.Release(ctx)
	return r.kept.releaseErr
}
