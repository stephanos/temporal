package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/campaign"
	"go.temporal.io/server/common/testing/testpilot/replay"
	"go.temporal.io/server/common/testing/testpilot/temporal"
	"go.temporal.io/server/tools/umpire/explore"
	"go.temporal.io/server/tools/umpire/ir"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func TestIRBridgeProtocol(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	require.NoError(t, err)
	binary := filepath.Join(t.TempDir(), "umpire-ir-bridge")
	build := exec.CommandContext(t.Context(), "go", "build", "-tags", "test_dep", "-o", binary, "./tools/umpire/cmd/umpire-ir-bridge")
	build.Dir = root
	output, err := build.CombinedOutput()
	require.NoError(t, err, "%s", output)
	m, err := ir.Load(filepath.Join(root, "model", "ir", "nexus-caller.json"))
	require.NoError(t, err)
	var query *umpirespb.Query
	for _, q := range m.Queries {
		if q.Name == "syncCompletion" {
			query = q
		}
	}
	require.NotNil(t, query)
	var scenario *umpirespb.Scenario
	for _, s := range m.Scenarios {
		if s.Name == query.Scenario.Name && s.Machine == query.Scenario.Machine {
			scenario = s
		}
	}
	require.NotNil(t, scenario)
	query.Exploration = &umpirespb.Exploration{Name: "protocol", Runs: 2, Edits: 100, DropPrefix: true, Variations: []*umpirespb.Variation{{Index: 0, Choices: []*umpirespb.Alternative{
		{Name: "missing-schedule", Priority: 20}, {Name: "original", Priority: 10, Actions: scenario.Actions[:1]},
	}}}}
	plan, err := explore.New(m, "protocol")
	require.NoError(t, err)
	require.Len(t, plan.Candidates, 2)
	require.NotEmpty(t, plan.Candidates[0].Rejection)
	candidate := plan.Candidates[1]
	require.Empty(t, candidate.Rejection)
	modelRoot := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(modelRoot, "ir"), 0755))
	encoded, err := protojson.Marshal(m)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(modelRoot, "ir", "model.json"), encoded, 0644))
	options := campaign.Options{Executable: binary, Dir: modelRoot}

	t.Run("campaign", func(t *testing.T) {
		bridge, err := campaign.Start(t.Context(), options)
		require.NoError(t, err)
		t.Cleanup(func() { _ = bridge.Close() })
		initialized, err := bridge.Initialize(t.Context(), plan.Name, "protocol-test")
		require.NoError(t, err)
		require.Equal(t, query.Limits.Name, initialized.Budget)
		require.EqualValues(t, query.Limits.Steps, initialized.Limits.Steps)
		require.Equal(t, []string{plan.Candidates[0].Key, candidate.Key}, initialized.Targets)
		next, err := bridge.Next(t.Context())
		require.NoError(t, err)
		require.Len(t, next.Skipped, 1)
		require.Equal(t, plan.Candidates[0].Key, next.Skipped[0].Target)
		require.NotEmpty(t, next.Skipped[0].Reason)
		require.NotNil(t, next.Candidate)
		require.Equal(t, candidate.Digest, next.Candidate.Identity)
		require.Equal(t, candidate.Key, next.Candidate.Target)
		require.Contains(t, next.Candidate.Covers, next.Candidate.Target)
		require.Equal(t, []byte(candidate.Bytes), []byte(next.Candidate.Case))
		source, err := testpilot.DecodeCaseProtoJSON(next.Candidate.Case)
		require.NoError(t, err)
		require.Equal(t, next.Candidate.CaseID, source.CaseId)
		catalog, err := temporal.NewWorkflowServiceCatalog()
		require.NoError(t, err)
		profile, err := temporal.DeriveProfile(source, catalog, temporal.Environment{Identity: "protocol-test", Namespace: "namespace", TaskQueue: "queue", HandlerTaskQueue: "handler", NexusEndpoint: "endpoint"})
		require.NoError(t, err)
		prepared, err := testpilot.Prepare(source, profile)
		require.NoError(t, err)
		require.True(t, proto.Equal(source, prepared.Snapshot()))
		_, err = bridge.Next(t.Context())
		require.ErrorIs(t, err, campaign.ErrCandidateOutstanding)
		detail := "no deployment"
		credited, err := bridge.Observe(t.Context(), next.Candidate.Identity, campaign.Result{PrepareRejected: &detail})
		require.NoError(t, err)
		require.Equal(t, "prepare-rejected", credited.Observation)
		require.Empty(t, credited.Credited)
		finished, err := bridge.Finish(t.Context(), "stopped")
		require.NoError(t, err)
		require.Equal(t, "stopped", finished.Status)
		require.Equal(t, 1, finished.Summary.Selected)
		require.NoError(t, bridge.Close())
	})

	t.Run("sequence", func(t *testing.T) {
		frames := []string{
			`{"frame":"next","seq":1,"set":"protocol"}`,
			`{"frame":"initialize","seq":1,"set":"protocol","profile":"p"}`,
			`{"frame":"initialize","seq":1,"set":"protocol","profile":"p"}`,
			`{"frame":"finish","seq":5,"set":"protocol"}`,
			`{"frame":"finish","seq":2,"set":"protocol"}`,
		}
		command := exec.CommandContext(t.Context(), binary)
		command.Dir = modelRoot
		command.Stdin = strings.NewReader(strings.Join(frames, "\n") + "\n")
		var stdout, stderr bytes.Buffer
		command.Stdout, command.Stderr = &stdout, &stderr
		require.NoError(t, command.Run(), "%s", stderr.String())
		var replies []struct {
			Frame  string
			Seq    int
			Reason string
		}
		decoder := json.NewDecoder(&stdout)
		for decoder.More() {
			var reply struct {
				Frame  string
				Seq    int
				Reason string
			}
			require.NoError(t, decoder.Decode(&reply))
			replies = append(replies, reply)
		}
		require.Len(t, replies, 5)
		require.Equal(t, []string{"rejected", "initialized", "rejected", "rejected", "finished"}, []string{replies[0].Frame, replies[1].Frame, replies[2].Frame, replies[3].Frame, replies[4].Frame})
		require.Contains(t, replies[0].Reason, "initialize or admit")
		require.Contains(t, replies[2].Reason, "sequence")
		require.Contains(t, replies[3].Reason, "sequence")
		require.Equal(t, 2, replies[4].Seq)
	})

	t.Run("replay", func(t *testing.T) {
		bridge, err := replay.StartBridge(t.Context(), options)
		require.NoError(t, err)
		t.Cleanup(func() { _ = bridge.Close() })
		hash := sha256.Sum256(candidate.Bytes)
		identity := hex.EncodeToString(hash[:])
		require.Equal(t, candidate.Identity, identity)
		admitted, err := bridge.Admit(t.Context(), plan.Name, "protocol-test", replay.Named{Target: candidate.Key}, identity)
		require.NoError(t, err)
		require.Equal(t, candidate.Case.CaseId, admitted.CaseID)
		require.Equal(t, candidate.Digest, admitted.Subject)
		require.Len(t, admitted.Edits, len(candidate.Actions)-1)
		require.NotEmpty(t, admitted.Edits)
		next, err := bridge.Next(t.Context())
		require.NoError(t, err)
		require.True(t, next.Exhausted)
		require.Len(t, next.Skipped, len(admitted.Edits))
		for i, skipped := range next.Skipped {
			require.Equal(t, admitted.Edits[i], skipped.Edit)
			require.Equal(t, "invalid", skipped.Fate)
			require.NotEmpty(t, skipped.Reason)
		}
		finished, err := bridge.Finish(t.Context(), "")
		require.NoError(t, err)
		require.Equal(t, "irreducible", finished.Status)
		require.Equal(t, admitted.Subject, finished.Retained)
		require.NotNil(t, finished.Proposal)
		require.Equal(t, admitted.Subject, finished.Proposal.Digest)
		recovered, err := explore.ReadProposal([]byte(finished.Proposal.Source))
		require.NoError(t, err)
		require.Equal(t, candidate.Bytes, recovered.Bytes)
		temporary, err := filepath.EvalSymlinks(t.TempDir())
		require.NoError(t, err)
		publicationRoot := filepath.Join(temporary, "proposals")
		written := replay.WriteProposal(publicationRoot, finished.Proposal)
		require.Equal(t, replay.ProposalWritten, written.Status, written.Error)
		require.Equal(t, filepath.Join(publicationRoot, finished.Proposal.Path), written.Written)
		require.NotEqual(t, replay.ProposalWritten, replay.WriteProposal(publicationRoot, finished.Proposal).Status)
		crossed, err := replay.StartBridge(t.Context(), options)
		require.NoError(t, err)
		t.Cleanup(func() { _ = crossed.Close() })
		_, err = crossed.Admit(t.Context(), plan.Name, "protocol-test", replay.Named{Target: candidate.Key}, strings.Repeat("0", 64))
		var crossedError *replay.CrossedError
		require.ErrorAs(t, err, &crossedError)
	})
}
