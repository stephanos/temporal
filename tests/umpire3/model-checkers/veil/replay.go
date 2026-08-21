package veil

import (
	"context"
	"time"

	"go.temporal.io/server/tests/umpire3/model-checkers/canonical"
	"go.temporal.io/server/tests/umpire3/protocol"
)

const canonicalReplayTimeout = 30 * time.Second

func Replay(
	ctx context.Context,
	command []string,
	input protocol.TraceReplayInput,
) (protocol.TraceReplayReceipt, error) {
	return canonical.ReplayFinite(ctx, command, input)
}
