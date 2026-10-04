package explore

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
)

func TestReplayBridgeRejectsCrossedIdentity(t *testing.T) {
	m, err := umpiremodel.Load("../../../model/ir/nexus-caller.json")
	require.NoError(t, err)
	for _, q := range m.Queries {
		if q.Name == "syncCompletion" {
			q.Exploration = &umpirespb.Exploration{Name: "space", Runs: 1, Edits: 1, DropPrefix: true, Variations: []*umpirespb.Variation{{Index: 0, Choices: []*umpirespb.Alternative{{Name: "missing-schedule"}}}}}
		}
	}
	var output bytes.Buffer
	err = Serve(bytes.NewBufferString(`{"frame":"admit","seq":1,"set":"space","profile":"p","target":"missing-schedule","identity":"crossed"}`+"\n"), &output, []*umpirespb.Model{m})
	require.NoError(t, err)
	var reply map[string]any
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(output.Bytes()), &reply))
	require.Equal(t, "crossed", reply["frame"])
	require.NotContains(t, reply, "proposal")
}

func TestReplayBridgeAdmitsTheCandidateIdentity(t *testing.T) {
	m, err := umpiremodel.Load("../../../model/ir/nexus-control.json")
	require.NoError(t, err)
	plan, err := New(m, "nexusControl")
	require.NoError(t, err)
	candidate := plan.Candidates[0]
	require.Empty(t, candidate.Rejection)
	admit, err := json.Marshal(map[string]any{"frame": "admit", "seq": 1, "set": plan.Name, "profile": "test", "target": candidate.Key, "identity": candidate.Identity})
	require.NoError(t, err)
	var output bytes.Buffer
	require.NoError(t, Serve(bytes.NewReader(append(admit, '\n')), &output, []*umpirespb.Model{m}))
	var reply map[string]any
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(output.Bytes()), &reply))
	require.Equal(t, "admitted", reply["frame"])
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
	require.NoError(t, Serve(bytes.NewBufferString(input), &output, []*umpirespb.Model{m}))
	decoder := json.NewDecoder(&output)
	var initialized map[string]any
	require.NoError(t, decoder.Decode(&initialized))
	var candidate struct {
		Case json.RawMessage `json:"case"`
	}
	require.NoError(t, decoder.Decode(&candidate))
	require.Equal(t, plan.Candidates[0].Bytes, candidate.Case)
}
