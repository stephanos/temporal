package veil

import (
	"context"
	"fmt"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func TestReplayRunsBoundCanonicalChecker(t *testing.T) {
	t.Setenv("UMPIRE3_TRACE_REPLAY_HELPER", "1")
	input := protocol.TraceReplayInput{
		FormatVersion: protocol.TraceReplayInputFormatVersion,
		Target:        protocol.TargetIDNexusCancellation,
		Property:      protocol.PropertyIDNexusCancellationWonExcludesSuccess,
		World:         "smoke",
		Variant:       "stale-completion-guard-removed",
		SemanticHash:  "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		Actions: []protocol.ActionKind{
			protocol.ActionKindDispatchTask,
			protocol.ActionKindAcquireOwnership,
			protocol.ActionKindWorkerReturnsSuccess,
			protocol.ActionKindPersistSuccess,
		},
	}
	digest, err := input.Digest()
	require.NoError(t, err)

	receipt, err := Replay(context.Background(),
		[]string{os.Args[0], "-test.run=^TestCanonicalReplayHelper$", "--"}, input)
	require.NoError(t, err)
	require.Equal(t, protocol.TraceReplayReceipt{
		FormatVersion: protocol.TraceReplayReceiptFormatVersion,
		TraceDigest:   digest,
		Target:        input.Target,
		Property:      input.Property,
		World:         input.World,
		Variant:       input.Variant,
		SemanticHash:  input.SemanticHash,
		Actions:       input.Actions,
		Status:        protocol.TraceReplayAccepted,
		TrustBadge:    protocol.TrustBadgeCheckedCertificate,
		Axioms:        []string{},
	}, receipt)
}

func TestCanonicalReplayHelper(t *testing.T) {
	if os.Getenv("UMPIRE3_TRACE_REPLAY_HELPER") != "1" {
		return
	}
	separator := slices.Index(os.Args, "--")
	if separator < 0 || len(os.Args[separator+1:]) != 10 {
		os.Exit(3)
	}
	arguments := os.Args[separator+1:]
	if arguments[1] != "nexus-cancellation" ||
		arguments[2] != "nexus.cancellation.won-excludes-success" ||
		arguments[3] != "smoke" || arguments[4] != "stale-completion-guard-removed" ||
		arguments[5] != "sha256:0000000000000000000000000000000000000000000000000000000000000000" ||
		!slices.Equal(arguments[6:], []string{
			"dispatch-task", "acquire-ownership", "worker-returns-success", "persist-success",
		}) {
		os.Exit(4)
	}
	fmt.Printf(`{"actions":["dispatch-task","acquire-ownership","worker-returns-success","persist-success"],`+
		`"axioms":[],"formatVersion":"umpire3/trace-replay-receipt/v1",`+
		`"property":"nexus.cancellation.won-excludes-success",`+
		`"semanticHash":"sha256:0000000000000000000000000000000000000000000000000000000000000000",`+
		`"status":"accepted","target":"nexus-cancellation","traceDigest":%q,`+
		`"trustBadge":"checked-certificate","variant":"stale-completion-guard-removed","world":"smoke"}`,
		arguments[0])
	os.Exit(0)
}
