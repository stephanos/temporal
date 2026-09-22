---
satisfies: [R7]
---
# fn-22-deterministic-replay-semantic.6 Emit the checked review-only regression proposal

## Description
Lift fn-33 .5's proposal shape into `Umpire.Promotion.propose` (an admitted Query, its checked Model, the anchor read off its planning, fresh names keyed by the candidate's digest under the Model's family, a caller-named location) so the exploration campaign and the replay bridge share it; the bridge's `finish` frame carries the retained candidate's proposal (digest, path, bytes) or its error only for a `minimized` or `irreducible` result. Go writes the bytes only under `--promotion-root` outside the model, at the path the bridge named, checking every path before writing any, refusing to overwrite an existing destination, and reports the digest, the path and where it was written; a write failure or an existing destination never reruns.

### Approach
- `Umpire.Exploration.Promotion.propose` becomes a call into the shared function; its tests stay green.

### Quick commands
`cd model && lake build UmpireTests umpire-replay-bridge-tests umpire-explore-tests && lake exe umpire-replay-bridge-tests && lake exe umpire-explore-tests && cd .. && go test -count=1 -tags test_dep ./tools/umpire/replay/`

**Size:** M
**Files:** `model/Umpire/Promotion.lean`, `model/Umpire/Exploration/Promotion.lean`, `model/Umpire/PromotionTests.lean`, `model/Umpire/Exploration/Tests/Classed.lean`, `model/Temporal/Tool/ReplayBridge.lean`, `model/Temporal/Tool/ReplayBridgeTests.lean`, `tools/umpire/replay/proposal.go`, `tools/umpire/replay/proposal_test.go`
**Touches:** `model/Umpire/Promotion*.lean`, `model/Umpire/Exploration/Promotion.lean`, `model/Umpire/Exploration/Tests/Classed.lean`, `model/Temporal/Tool/ReplayBridge*.lean`, `tools/umpire/replay/proposal*.go`

### Re-plan note (2026-09-22)
Rewritten on fn-85, fn-86, fn-87 and fn-33 after the first plan's MAJOR_RETHINK; revised after plan review round one; see the spec's **Re-plan** and **Plan review** sections. Start only after the spec's fresh plan review.
## Acceptance
- [ ] An incomplete, not-reproduced or indeterminate result carries no proposal.
- [ ] The proposal compiles through `Umpire.Promotion` from the retained candidate's admitted Query, renders the Model's expected trace and never the observed Run, and seals the same digest across two reductions.
- [ ] Nothing is written under the model root; a write failure or an existing destination is reported as the proposal's status and triggers no rerun.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
