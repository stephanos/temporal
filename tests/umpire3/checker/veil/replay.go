package veil

import (
	"context"
	"time"

	"go.temporal.io/server/tests/umpire3/checker/leanreplay"
	protocolchecker "go.temporal.io/server/tests/umpire3/protocol/checker"
)

const canonicalReplayTimeout = 30 * time.Second

func Replay(
	ctx context.Context,
	command []string,
	input protocolchecker.TraceReplayInput,
) (protocolchecker.TraceReplayReceipt, error) {
	return leanreplay.ReplayFinite(ctx, command, input)
}
