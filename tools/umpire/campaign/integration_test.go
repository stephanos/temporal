//go:build integration

package campaign

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/binding"
)

// The proof point end to end: one candidate of the caller Model's exploratory set, produced by
// the real bridge, prepared and run once against a development cluster through the shared
// deployment binding, its cleanup observed, and its planned path credited by the bridge. The
// cluster is named by environment, and the test reports itself as not run without one.
func TestIntegrationOneCandidateCreditsItsPlannedPath(t *testing.T) {
	grpcAddress := os.Getenv("UMPIRE_FUZZ_GRPC")
	httpAddress := os.Getenv("UMPIRE_FUZZ_HTTP")
	if grpcAddress == "" || httpAddress == "" {
		t.Skip("UMPIRE_FUZZ_GRPC and UMPIRE_FUZZ_HTTP name no development cluster; the integration proof did not run")
	}
	executable, modelRoot := bridgeExecutable(t)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()

	suffix := time.Now().UTC().Format("20060102-150405")
	deployment := binding.Deployment{
		GRPCAddress:   grpcAddress,
		HTTPAddress:   httpAddress,
		Namespace:     "umpire-fuzz-" + suffix,
		TaskQueue:     "umpire-fuzz-" + suffix,
		NexusEndpoint: "umpire-fuzz-" + suffix,
		Create:        true,
	}
	campaign, err := binding.Open(ctx, deployment, deployment.TaskQueue+"-handler")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, campaign.Close(context.Background())) })

	var stderr bytes.Buffer
	bridge, err := Start(ctx, Options{Executable: executable, Dir: modelRoot, Stderr: &stderr})
	require.NoError(t, err)
	t.Cleanup(func() { _ = bridge.Close() })
	profile := "umpire-fuzz." + deployment.Namespace
	_, err = bridge.Initialize(ctx, "nexusCallerExploration", profile)
	require.NoError(t, err)
	next, err := bridge.Next(ctx)
	require.NoError(t, err)
	require.NotNil(t, next.Candidate)

	outcome, err := RunCandidate(ctx, bridge, CampaignBinder{Campaign: campaign}, next.Candidate)
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, outcome.Kind)
	require.NoError(t, outcome.ReleaseError)
	require.NotNil(t, outcome.Credited)
	t.Logf("candidate %s observed %s (%s): credited %v", outcome.Identity, outcome.Credited.Observation, outcome.Credited.Detail, outcome.Credited.Credited)
	require.Equal(t, "satisfied", outcome.Credited.Observation)
	require.Equal(t, next.Candidate.Covers, outcome.Credited.Credited)

	finished, err := bridge.Finish(ctx, "stopped")
	require.NoError(t, err)
	require.Equal(t, len(next.Candidate.Covers), finished.Summary.Covered)
}
