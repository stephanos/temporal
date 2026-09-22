---
satisfies: [R5]
---
# fn-33-run-serial-bounded-semantic-exploration.5 Prove determinism, regression independence and counterexample promotion

## Description
Run identical checked inputs, caps and decisive observations twice and compare candidate order, per-target credit and the canonical summary. Prove that wall-clock timing can change only the completed prefix, that pinned regressions consume no campaign limit, and that a counterexample compiles to the same promotion source every time and installs nothing.

**Size:** M
**Files:** `tools/umpire/campaign/integration_test.go`, `model/Umpire/Exploration/Tests/**`, `model/Temporal/Tool/ExplorationBridgeTests.lean`, `model/Umpire/Promotion.lean`
**Touches:** [tools/umpire/campaign/integration_test.go, model/Umpire/Exploration/Tests/**, model/Temporal/Tool/ExplorationBridgeTests.lean, model/Umpire/Promotion.lean]

### Approach
- A scripted observation stream (decisive verdicts by candidate identity) drives the campaign in Lean and the coordinator in Go; both are replayed twice and compared byte for byte.
- A stream that ends early (stopped) yields a prefix of the full run's summary with the same identities.
- The Switch example's functional set Cases are shown outside the campaign's caps.
- For a counterexample, render the proposal with `renderPromotionSource` over the candidate's witness trace, compile it with `compilePromotionSource` from the `AdmittedQuery` the campaign retained beside the candidate (one Model per campaign, so one index), a `PromotionBaseAnchor` (plan result, plan, expected trace, selection reason) and fresh namespaced promoted IDs with a source location, and compare the SHA-256 across two runs; the file is written only under a caller-named scratch root and never under `model/`.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Promotion.lean:216,300` — `checkPromotedQuery`, `compilePromotionSource`.
- Task .1's ledger and task .4's summary.

### Quick commands
`cd model && lake exe umpire-explore-tests && cd .. && go test -count=1 -tags test_dep ./tools/umpire/campaign/...`

### Re-plan note (2026-09-21)
Re-planned on fn-85's exploratory set after fn-86 R6 deleted the variation Space this task was first written against; see the spec's **Re-plan on fn-85** section. Start only after the spec's fresh plan review.
## Acceptance
- [ ] Two runs over identical inputs and observations produce identical candidate order, credit and summary bytes; an early stop produces a prefix.
- [ ] Pinned regressions are not selected, prepared or counted against any campaign cap.
- [ ] A counterexample renders, compiles and compares to the same promotion source SHA-256 across runs, written only where the caller names, never installed.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
