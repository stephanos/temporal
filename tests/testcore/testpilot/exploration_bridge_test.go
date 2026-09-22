package testpilot

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal"
)

// explorationBridgeBinary is the exploration bridge the model package builds; the check target
// builds it before this test runs, and a checkout without it skips.
func explorationBridgeBinary(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "..", "..", "model", ".lake", "build", "bin", "umpire-explore"))
	require.NoError(t, err)
	if _, err := os.Stat(path); err != nil {
		t.Skipf("exploration bridge is not built at %s: %v", path, err)
	}
	return path
}

// explorationFrame is the part of every bridge frame the coordinator reads: its kind, its
// sequence number, and, on a candidate, the identity, the Case and the targets the Case covers.
type explorationFrame struct {
	Frame     string   `json:"frame"`
	Seq       int      `json:"seq"`
	Set       string   `json:"set"`
	Reason    string   `json:"reason"`
	Candidate string   `json:"candidate"`
	Target    string   `json:"target"`
	Covers    []string `json:"covers"`
	CaseID    string   `json:"caseId"`
	Skipped   []struct {
		Candidate string `json:"candidate"`
		Target    string `json:"target"`
		Reason    string `json:"reason"`
	} `json:"skipped"`
	Case json.RawMessage `json:"case"`
}

func runExplorationBridge(t *testing.T, binary string, frames ...string) []explorationFrame {
	t.Helper()
	command := exec.Command(binary)
	command.Stdin = strings.NewReader(strings.Join(frames, "\n") + "\n")
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	require.NoError(t, command.Run(), "bridge stderr: %s", stderr.String())
	var decoded []explorationFrame
	for _, line := range strings.Split(strings.TrimSuffix(stdout.String(), "\n"), "\n") {
		var frame explorationFrame
		require.NoError(t, json.Unmarshal([]byte(line), &frame), "frame: %s", line)
		decoded = append(decoded, frame)
	}
	return decoded
}

// The first realizable row target of the caller Model's exploratory set crosses the bridge as a
// whole Case that Prepare accepts under the caller Profile: Go reads the frame's Case and identity
// and nothing else about the campaign. The first row target itself performs a schedule member the
// realization binds nothing for, so the bridge reports it as skipped and moves on.
func TestExplorationBridgeFirstCandidatePrepares(t *testing.T) {
	binary := explorationBridgeBinary(t)
	const set = "nexusCallerExploration"
	frames := runExplorationBridge(t, binary,
		`{"frame":"initialize","seq":1,"set":"`+set+`","profile":"`+asyncNexusArtifactNamespace+`"}`,
		`{"frame":"next","seq":2,"set":"`+set+`"}`,
		`{"frame":"finish","seq":3,"set":"`+set+`"}`,
	)
	require.Len(t, frames, 3)
	require.Equal(t, "initialized", frames[0].Frame)
	require.Equal(t, "candidate", frames[1].Frame)
	require.NotEmpty(t, frames[1].Skipped)
	require.True(t, strings.HasPrefix(frames[1].Skipped[0].Target, "row:"), frames[1].Skipped[0].Target)
	require.Equal(t, 2, frames[1].Seq)
	require.True(t, strings.HasPrefix(frames[1].Target, "row:"), frames[1].Target)
	require.Contains(t, frames[1].Covers, frames[1].Target)
	require.True(t, strings.HasPrefix(frames[1].Candidate, "sha256:"), frames[1].Candidate)
	require.Equal(t, "temporal.case."+set+"."+strings.TrimPrefix(frames[1].Candidate, "sha256:"), frames[1].CaseID)
	require.Equal(t, "finished", frames[2].Frame)

	source, err := testpilot.DecodeCaseProtoJSON(frames[1].Case)
	require.NoError(t, err)
	require.Equal(t, frames[1].CaseID, source.GetCaseId())
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	prepared, err := testpilot.Prepare(source, asyncNexusProfile(catalog))
	require.NoError(t, err)
	require.NotNil(t, prepared)
}

// A frame out of sequence is rejected before the campaign is touched, and the campaign goes on
// from the frame it expected.
func TestExplorationBridgeRejectsOutOfOrderFrames(t *testing.T) {
	binary := explorationBridgeBinary(t)
	const set = "nexusCallerExploration"
	frames := runExplorationBridge(t, binary,
		`{"frame":"next","seq":1,"set":"`+set+`"}`,
		`{"frame":"initialize","seq":1,"set":"`+set+`","profile":"p"}`,
		`{"frame":"initialize","seq":1,"set":"`+set+`","profile":"p"}`,
		`{"frame":"finish","seq":5,"set":"`+set+`"}`,
		`{"frame":"finish","seq":2,"set":"`+set+`"}`,
	)
	require.Len(t, frames, 5)
	require.Equal(t, "rejected", frames[0].Frame)
	require.Contains(t, frames[0].Reason, "no campaign is open")
	require.Equal(t, "initialized", frames[1].Frame)
	require.Equal(t, "rejected", frames[2].Frame)
	require.Contains(t, frames[2].Reason, "duplicate")
	require.Equal(t, "rejected", frames[3].Frame)
	require.Contains(t, frames[3].Reason, "out-of-order")
	require.Equal(t, "finished", frames[4].Frame)
}
