package choice

import (
	"crypto/sha256"
	"encoding/binary"
	"slices"
	"strings"
	"testing"
)

func TestProjectInterruptedReplayPlanValidatesObservedPrefix(t *testing.T) {
	identity := ExecutionIdentity{TargetSHA256: sha256.Sum256([]byte("target")), ToolchainBuildKey: strings.Repeat("a", 64), GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte("runtime"))}
	a, b := sha256.Sum256([]byte("a")), sha256.Sum256([]byte("b"))
	decision, err := CanonicalDecision(0, KindRunnable, 19, false, [][32]byte{a, b}, a, 0)
	if err != nil {
		t.Fatal(err)
	}
	trace, err := BuildTrace([]Record{decision.Record()}, TerminalComplete)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := ProjectInterruptedReplayPlan(trace.Bytes, 8192, identity)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Decisions) != 1 || plan.Decisions[0] != decision || plan.SourceTraceSHA256 != trace.SHA256 {
		t.Fatalf("interrupted projection=%+v", plan)
	}
	if _, err := ValidatePrefixReplayPlan(plan, identity); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"truncated", "ordinal", "origin", "capacity", "identity"} {
		t.Run(name, func(t *testing.T) {
			payload := slices.Clone(trace.Bytes)
			limit := uint64(8192)
			candidate := identity
			switch name {
			case "truncated":
				payload = payload[:len(payload)-1]
			case "ordinal":
				binary.BigEndian.PutUint64(payload, 1)
			case "origin":
				readiness, err := NewReadiness(1, 0)
				if err != nil {
					t.Fatal(err)
				}
				observed, err := BuildTrace([]Record{decision.Record(), {Ordinal: 1, Kind: KindSelectResult, Flags: FlagObservation, Alternatives: 1, Selected: 0, Data: 1, SiteOffset: 19, Readiness: readiness, Origin: 1}}, TerminalComplete)
				if err != nil {
					t.Fatal(err)
				}
				payload = observed.Bytes
				binary.BigEndian.PutUint64(payload[96+32:], 2)
			case "capacity":
				limit = MinimumTraceBytes - 1
			case "identity":
				candidate.TargetSHA256 = [32]byte{}
			}
			if _, err := ProjectInterruptedReplayPlan(payload, limit, candidate); err == nil {
				t.Fatal("invalid interrupted prefix accepted")
			}
		})
	}
}
