package explore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	modelirspb "go.temporal.io/server/api/modelir/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/campaign"
	"go.temporal.io/server/common/testing/testpilot/replay"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
)

func TestReplayBridgeRejectsCrossedIdentity(t *testing.T) {
	m, err := umpiremodel.Load("../../../model/ir/nexus-caller.json")
	require.NoError(t, err)
	for _, q := range m.Queries {
		if q.Name == "syncCompletion" {
			q.Exploration = &modelirspb.Exploration{Name: "space", Runs: 1, Edits: 1, DropPrefix: true, Variations: []*modelirspb.Variation{{Index: 0, Choices: []*modelirspb.Alternative{{Name: "missing-schedule"}}}}}
		}
	}
	var output bytes.Buffer
	err = Serve(bytes.NewBufferString(`{"frame":"admit","seq":1,"set":"space","profile":"p","target":"missing-schedule","identity":"crossed"}`+"\n"), &output, []*modelirspb.Model{m})
	require.NoError(t, err)
	var reply map[string]any
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(output.Bytes()), &reply))
	require.Equal(t, "crossed", reply["frame"])
	require.NotContains(t, reply, "proposal")
}

func TestUnreproducedRuntimeFailureNeverProducesProposal(t *testing.T) {
	m, err := umpiremodel.Load("../../../model/ir/nexus-control.json")
	require.NoError(t, err)
	plan, err := New(m, "nexusControl")
	require.NoError(t, err)
	candidate := plan.Candidates[0]
	require.Empty(t, candidate.Rejection)
	input, requests := io.Pipe()
	replies, output := io.Pipe()
	finished := make(chan error, 1)
	go func() {
		finished <- Serve(input, output, []*modelirspb.Model{m})
		_ = output.Close()
		_ = input.Close()
	}()
	t.Cleanup(func() { require.NoError(t, requests.Close()); require.NoError(t, replies.Close()) })
	bridge := replay.NewBridge(requests, replies, 0)
	admitted, err := bridge.Admit(t.Context(), plan.Name, "test", replay.Named{Target: candidate.Key}, candidate.Identity)
	require.NoError(t, err)
	reducer := replay.Reducer{Bridge: bridge, Admitted: admitted, Subject: &replay.Subject{Case: candidate.Case}, Binder: unusedBinder{}, Prepare: func(string, *testpilotspb.Case) (*testpilot.PreparedCase, error) {
		return nil, errors.New("must not prepare an unreproduced subject")
	}, Limits: replay.DefaultLimits}
	result, err := reducer.Reduce(t.Context(), &replay.Reruns{Class: replay.ClassNotReproduced})
	require.NoError(t, err)
	require.Equal(t, replay.ReductionNotAttempted, result.Status)
	require.Nil(t, result.Proposal)
	require.Equal(t, replay.ProposalNone, replay.WriteProposal(t.TempDir(), result.Proposal).Status)
	require.NoError(t, <-finished)
}

type unusedBinder struct{}

func (unusedBinder) Bind(context.Context, string, *testpilotspb.Case) (campaign.Bound, error) {
	return nil, errors.New("must not run an unreproduced subject")
}

func TestProposalReanswersTheOriginalQueryAndRejectsTampering(t *testing.T) {
	m, err := umpiremodel.Load("../../../model/ir/nexus-control.json")
	require.NoError(t, err)
	plan, err := New(m, "nexusControl")
	require.NoError(t, err)
	candidate, err := plan.Reduce(plan.Candidates[0], 4)
	require.NoError(t, err)
	proposed, err := plan.Proposal(candidate)
	require.NoError(t, err)
	recovered, err := ReadProposal([]byte(proposed.Source))
	require.NoError(t, err)
	require.Equal(t, candidate.Bytes, recovered.Bytes)
	again, err := plan.Proposal(candidate)
	require.NoError(t, err)
	require.Equal(t, proposed, again)
	var changed proposal
	require.NoError(t, json.Unmarshal([]byte(proposed.Source), &changed))
	changed.Drops = nil
	forged, err := json.Marshal(changed)
	require.NoError(t, err)
	_, err = ReadProposal(forged)
	require.ErrorContains(t, err, "exact identities and Case")
}

func TestCampaignBridgePreservesExactCaseBytes(t *testing.T) {
	m, err := umpiremodel.Load("../../../model/ir/nexus-caller.json")
	require.NoError(t, err)
	plan, err := New(m, "nexusDeadlines")
	require.NoError(t, err)
	var output bytes.Buffer
	input := `{"frame":"initialize","seq":1,"set":"nexusDeadlines","profile":"test"}` + "\n" + `{"frame":"next","seq":2,"set":"nexusDeadlines","profile":"test"}` + "\n"
	require.NoError(t, Serve(bytes.NewBufferString(input), &output, []*modelirspb.Model{m}))
	decoder := json.NewDecoder(&output)
	var initialized map[string]any
	require.NoError(t, decoder.Decode(&initialized))
	var candidate struct {
		Case json.RawMessage `json:"case"`
	}
	require.NoError(t, decoder.Decode(&candidate))
	require.Equal(t, plan.Candidates[0].Bytes, candidate.Case)
}
