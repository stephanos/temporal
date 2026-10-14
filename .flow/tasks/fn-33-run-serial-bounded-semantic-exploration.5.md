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
- [x] Two runs over identical inputs and observations produce identical candidate order, credit and summary bytes; an early stop produces a prefix.
- [x] Pinned regressions are not selected, prepared or counted against any campaign cap.
- [x] A counterexample renders, compiles and compares to the same promotion source SHA-256 across runs, written only where the caller names, never installed.
## Done summary
Determinism pinned on both sides (Lean campaign replay of a scripted observation stream through two campaigns, the bridge's frames byte for byte, a cut stream as the completed prefix; Go Drive and umpire-fuzz twice, a stop as the completed prefix), pinned regressions shown outside the campaign and its caps, and each counterexample compiled by Umpire.Exploration.Promotion into a review-only source that seals the same SHA-256 every run, carried by the bridge's summary and written by umpire-fuzz only under --promotion-root outside the model.
## Evidence
- Commits: 76855df2d35d333b54dc0d1f66d8332d04611387
- Tests: cd model && lake build && lake exe umpire-explore-tests, LEAN_NUM_THREADS=1 make lint-model, make umpire-check-goldens umpire-check-case-runtime-conformance umpire-check-inventory umpire-check-model-module-index umpire-check-exploration-bridge, go test -count=1 -tags test_dep ./tools/umpire/campaign/ ./tools/umpire/cmd/umpire-fuzz/, GOLANGCI_LINT_BASE_REV=HEAD make lint-code-fast
- PRs: