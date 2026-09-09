---
satisfies: [R3]
---
# fn-80-close-the-model-to-case-seam-and-harden.1 Add rule_events horizon to classic Contract rules

## Description
Implements R3 (spec §R3 event-count horizon). Adds the `rule_events` bound to the wire, admission, both evaluator paths, and the Lean authoring constructor. Split from the fault work because it touches only the verification package and contract.proto.

**Size:** M
**Files:** `proto/internal/temporal/server/api/testpilot/v1/contract.proto`, `common/testing/testpilot/internal/verification/prepare.go`, `common/testing/testpilot/internal/verification/evaluator.go`, `common/testing/testpilot/internal/verification/evaluator_test.go`, `common/testing/testpilot/internal/verification/prepare_test.go`, `model/Testpilot/Authoring.lean`, `model/Testpilot/Tests/Authoring.lean`, generated `api/testpilot/v1/*.pb.go`, `model/Testpilot/Protocol.olean` via lake stamp
**Touches:** [proto/internal/temporal/server/api/testpilot/v1/contract.proto, api/testpilot/v1/**, common/testing/testpilot/internal/verification/**, model/Testpilot/Authoring.lean, model/Testpilot/Tests/**]

### Approach
- Add `int64 rule_events = 3` as a plain field (no oneof; spec §R3 explains the Lean mirror reason). Comment the message per the docs-gap notes: counts evaluated events since the rule's last transition, reset on transition, continues under incompleteness, stops at terminal, host-clock caveat on elapsed.
- Regenerate: `make proto`, then `cd model && lake build Testpilot` (the lakefile `testpilotProtocolSchemas` stamp at `model/lakefile.lean:38-62` must show `Built Testpilot.Protocol`; memory pitfall "Track schema inputs before reusing generated Lean modules").
- Admission at `internal/verification/prepare.go:239-243` currently rejects `GetElapsedMilliseconds() <= 0`; rewrite to require exactly one positive bound, reject both-positive, both-zero, negative.
- Evaluator: the horizon check is `evaluator.go:253-257` (before transitions). Add a per-rule counter on `ruleChange`/rule state, reset in the transition-apply path, incremented in one helper called from both `Evaluator.Observe` and `PreparedContract.Evaluate` (`evaluator.go:470-484`). Mirror the scoped counter shape at `internal/verification/scoped.go:45,473-474,528`.
- Respect the incompleteness precedence: the early return at `evaluator.go:253` suppresses expiry but the counter still increments (memory: "Freeze Contract transitions when execution becomes incomplete").
- Lean: add `Monitor.horizonEvents` beside `Monitor.horizon` at `model/Testpilot/Authoring.lean:393-396`; add a `#guard` round-trip in `model/Testpilot/Tests/Authoring.lean`.

### Investigation targets
**Required** (read before coding):
- `common/testing/testpilot/internal/verification/evaluator.go:244-260` — `nextChange` and the horizon check
- `common/testing/testpilot/internal/verification/evaluator.go:470-484` — offline `PreparedContract.Evaluate`
- `common/testing/testpilot/internal/verification/prepare.go:230-250` — liveness admission
- `common/testing/testpilot/internal/verification/scoped.go:470-530` — operation-transition counter to mirror

**Optional** (reference as needed):
- `common/testing/testpilot/internal/verification/evaluator_test.go` — existing horizon tests
- `.flow/memory/bug/runtime-errors/freeze-contract-transitions-when-2026-09-05.md`

### Key context
- No conformance Case declares a horizon today, so no `expected.json` regenerates in this task. The checked-in Case that declares `rule_events` is the fault fixture in task .8.
- `internal/ir` has no horizon code; admission lives only in `verification/prepare.go`.

## Acceptance
- [ ] `contract.proto` has `rule_events = 3` with the semantics comment; `make proto` and `lake build Testpilot` succeed and the lake output shows `Built Testpilot.Protocol`
- [ ] `Prepare` rejects both bounds positive, both zero, and negative `rule_events`, with a `*PreparationError` category test in `prepare_test.go`
- [ ] Table test in `evaluator_test.go`: expiry after exactly N evaluated events since last transition; reset on transition; expiry on the satisfying event resolves as expiry; counter increments under incompleteness while expiry is suppressed; counting stops at terminal
- [ ] Differential test: `Evaluator.Observe` and `PreparedContract.Evaluate` produce equal verdicts over the same event sequence for a `rule_events` rule
- [ ] `Monitor.horizonEvents` exists in `model/Testpilot/Authoring.lean` with a `#guard` in `model/Testpilot/Tests/Authoring.lean`
- [ ] `CGO_ENABLED=0 go test -tags test_dep ./common/testing/testpilot/...` passes; `make umpire-check-testpilot-protocol` passes

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
