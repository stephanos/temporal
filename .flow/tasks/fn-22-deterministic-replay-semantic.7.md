---
satisfies: [R7]
---
# fn-22-deterministic-replay-semantic.7 Emit the checked review-only regression proposal

## Description
Lift fn-33 .5's proposal shape into `Umpire.Command.Promotion.propose` (downstream of `Umpire.Command.Authoring`, so `Umpire.Promotion` keeps its imports) (an admitted Query, its checked Model, the anchor read off its planning, fresh names keyed by the candidate's digest, its Plan checksum, under the Model's family, a caller-named location) so the exploration campaign and the replay bridge share it; the bridge's `finish` frame carries the retained candidate's proposal (digest, path, bytes) or its error only for a `minimized` or `irreducible` result. Lift `umpire-fuzz`'s proposal writer (the promotion root's containment outside the model, the check of every path before any write, the written paths) into `tools/umpire/internal/cli`, resolving both the promotion root and the model root through `filepath.EvalSymlinks` before the containment check, creating each file exclusively (`O_CREATE|O_EXCL`) so an existing destination is never replaced, and have both commands use it; `tools/umpire/replay` reports the digest, the path and where it was written, and a write failure or an existing destination never reruns.

### Approach
- `Umpire.Exploration.Promotion.propose` becomes a call into the shared function; its tests stay green; `umpire-fuzz`'s tests stay green over the shared writer.

### Quick commands
`cd model && lake build UmpireTests umpire-replay-bridge-tests umpire-explore-tests && lake exe umpire-replay-bridge-tests && lake exe umpire-explore-tests && cd .. && go test -count=1 -tags test_dep ./tools/umpire/replay/ ./tools/umpire/internal/cli/ ./tools/umpire/cmd/umpire-fuzz/`

**Size:** M
**Files:** `model/Umpire/Command/Promotion.lean`, `model/Umpire/Command.lean`, `model/Umpire/Exploration/Promotion.lean`, `model/Umpire/PromotionTests.lean`, `model/Umpire/Exploration/Tests/Classed.lean`, `model/Temporal/Tool/ReplayBridge.lean`, `model/Temporal/Tool/ReplayBridgeTests.lean`, `tools/umpire/internal/cli/proposal.go`, `tools/umpire/internal/cli/proposal_test.go`, `tools/umpire/cmd/umpire-fuzz/run.go`, `tools/umpire/cmd/umpire-fuzz/run_test.go`, `tools/umpire/replay/proposal.go`, `tools/umpire/replay/proposal_test.go`
**Touches:** `model/Umpire/Command/Promotion.lean`, `model/Umpire/Command.lean`, `model/Umpire/Exploration/Promotion.lean`, `model/Umpire/Exploration/Tests/Classed.lean`, `model/Temporal/Tool/ReplayBridge*.lean`, `tools/umpire/internal/cli/**`, `tools/umpire/cmd/umpire-fuzz/**`, `tools/umpire/replay/proposal*.go`

### Re-plan note (2026-09-22)
Rewritten on fn-85, fn-86, fn-87 and fn-33 after the first plan's MAJOR_RETHINK and revised through the six plan review rounds the spec's **Plan review** section records (SHIP on round six, 2026-09-22).
## Acceptance
- [x] An incomplete, not-reproduced or indeterminate result carries no proposal.
- [x] The proposal compiles through `Umpire.Promotion` from the retained candidate's admitted Query, renders the Model's expected trace and never the observed Run, and seals the same digest across two reductions, the proposal's names, its file and the candidate's Case ID all keyed on the one Plan checksum digest.
- [x] Nothing is written under the model root, a symlinked root or path included; a write failure or an existing destination (exclusive create) is reported as the proposal's status and triggers no rerun; one writer serves both commands, pinned in `proposal_test.go`.
## Done summary
`Umpire.Command.Promotion` (downstream of `Umpire.Command.Authoring`, so `Umpire.Promotion` keeps its imports) owns the proposal: `digestOf` a Plan checksum, the `anchor` read off an admitted Query's planning, the fresh-name `spec` under the Model's family keyed by the digest, `propose` through `compilePromotionSource`, and the `<set>-<digest>.lean` `path` and `location`. `Umpire.Exploration.Promotion` is rebuilt on it (its `Proposal` is the shared structure) with the exploration pins and the exploration bridge's tests green. The replay bridge's `finished` frame carries the retained candidate's proposal (digest, SHA-256, path, bytes, or its error) only for a `minimized` or `irreducible` result, compiled from the candidate that keeps the retained positions, and `null` for an incomplete one; pinned for the control (its own digest), the caller's minimized reduction (the retained candidate's digest names the proposal, its file and its Case) and an incomplete reduction, and sealed identically across two reductions. `tools/umpire/internal/cli` gains `Resolve` (symlinks resolved along the existing part of a path), `Within`, `OutsideModel` and `WriteProposals` (every path checked before any write, each file created exclusively, a directory that resolves out of the root refused); `umpire-fuzz` uses them for `--promotion-root` and `--record-root`, its test now pinning that a second run into the same root is refused and leaves the file as it was. `replay.WriteProposal` reports the proposal's status (`none`, `not-compiled`, `compiled`, `written`, `write-failed`), and the live Go test writes the control's proposal under a scratch root.
## Evidence
- Commits: bf1b34b0364ae34a4b3e014ed4baabc614e06e57
- Tests: cd model && lake build, LEAN_NUM_THREADS=1 make lint-model, make umpire-check-goldens umpire-check-inventory umpire-check-model-module-index umpire-check-exploration-bridge umpire-check-replay-bridge, go test -count=1 -tags test_dep ./tools/umpire/replay/ ./tools/umpire/internal/cli/ ./tools/umpire/cmd/umpire-fuzz/, GOLANGCI_LINT_BASE_REV=HEAD make lint-code-fast
- PRs: