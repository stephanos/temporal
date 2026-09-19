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
Added the `rule_events` bound to `ContractHorizonDefinition` so a bounded-liveness rule can expire on evaluated Run Events instead of the recording host's clock, with admission requiring exactly one positive bound and one helper (`horizonReached`) owning the counter for both the online and the offline evaluator paths.

Notes for downstream tasks:
- The spec's "counting continues while execution is incomplete" is not implementable at the documented site: `Evaluator.Observe` returns before `changes` whenever execution is incomplete (the banked "Freeze Contract transitions when execution becomes incomplete" behaviour), so the whole rule freezes. Incompleteness is sticky, so a frozen counter and a ticking one are observationally identical. The proto comment and the test now state the freeze; task .9's horizon rule text should say the same.
- `contract.proto` changing at all re-fingerprints every typed operation, because each `RpcSchema` carries the generator's whole descriptor input closure (fn-77 follow-up). `typed-nexus-case.json` and `typed-unary-case.json` were regenerated through `make umpire-gen-case-runtime-conformance`; only `behaviorFingerprint` bytes moved. The task's "no fixture regenerates" note was wrong about the functional tree.
- Swept into the first commit: `.flow/tasks/fn-80-....4.json` — this run's `flowctl task reset` unblocking task .4, whose block reason ("unblock when fn-77.10 is done") was already satisfied.

stage: impl-review - ran (claude backend, model claude-fable-5-1 at high) - SHIP with two P3 findings, both fixed in the follow-up commit
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 0a6d6eeafa13e5f858b2ada6ac98738e0cf30b77, 6e8eaefa18f15da39a7719aa0466ca4b21d4bcc5
- Tests: CGO_ENABLED=0 go test -tags test_dep ./common/testing/testpilot/..., CGO_ENABLED=0 go test -tags test_dep ./tests/testcore/testpilot/..., cd model && lake build Temporal TemporalModelTests UmpireTests TestpilotTests, make umpire-check-testpilot-protocol, make umpire-check-testpilot-authoring, make umpire-check-case-runtime-conformance, make umpire-check-lean-api, make umpire-check-semantic-inventory, make umpire-check-retired-vocabulary, make lint-model (169 findings = recorded baseline, all in generated Temporal/API/{Types,Proto}.lean), GATE_SKIPPED:live-tests:disk - make umpire-check-live-tests needs a live cluster and whole-repo build; task .1 declares no live acceptance
- PRs: