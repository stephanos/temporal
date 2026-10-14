package choice

import (
	"crypto/sha256"
	_ "embed"
	"fmt"
)

//go:embed interrupted.go
var interruptedReplaySource []byte

// InterruptedReplayImplementationSHA256 identifies the optional interrupted
// prefix projection without changing the native complete-trace implementation.
func InterruptedReplayImplementationSHA256() string {
	return fmt.Sprintf("sha256:%x", sha256.Sum256(interruptedReplaySource))
}

// ProjectInterruptedReplayPlan projects a validated observed prefix after a
// modeled interruption. It makes no claim that the execution finished; callers
// must apply the returned tape in prefix mode and retain the interruption.
func ProjectInterruptedReplayPlan(payload []byte, limit uint64, identity ExecutionIdentity) (ReplayPlan, error) {
	if err := ValidateTraceLimit(limit); err != nil {
		return ReplayPlan{}, err
	}
	count, err := TraceRecordCount(payload)
	if err != nil {
		return ReplayPlan{}, err
	}
	// Reuse the complete-trace codec only to validate the raw records. This
	// temporary scaffold is neither returned nor exposed as completed evidence.
	trace, err := DecodeStoredTrace(Profile, payload, TerminalMetadata{State: TerminalComplete, Limit: limit, Records: count, SHA256: sha256.Sum256(payload)})
	if err != nil {
		return ReplayPlan{}, err
	}
	return ProjectReplayPlan(trace, identity)
}
