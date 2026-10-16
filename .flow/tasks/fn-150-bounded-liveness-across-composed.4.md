---
satisfies: [R2, R3, R4]
---
# fn-150-bounded-liveness-across-composed.4 Pin composed progress boundaries and witness replay

## Description
Pin composed progress boundaries and witness replay. Advances R2, R3, R4 of the parent spec.

**Size:** M
**Files:** `tools/umpire/check/composed_progress_boundaries_test.go`, `tools/umpire/check/replay_test.go`, `tools/umpire/internal/engine/progress_test.go`, `tools/umpire/internal/engine/compose_test.go`
**Touches:** [tools/umpire/check/composed_progress_boundaries_test.go, tools/umpire/check/replay_test.go, tools/umpire/internal/engine/progress_test.go, tools/umpire/internal/engine/compose_test.go]

### Approach
- Build small generic IR/table fixtures independent of task 3's authoring goldens. Reuse engine/reader test builders and the existing witness replay path.
- Cover deadlock, fair non-progress cycle, deadline miss and success independently, including a sync blocked by its partner, an unrelated member consuming the bound and a fair path that still misses a finite deadline.
- Cover all start combinations, source-unreachable exercised=false, source already satisfying destination, hole-only continuations, search exhaustion and an already-proved counterexample retained when later exploration hits a limit.
- Pin inherited, replacement and same-name assumption behavior; check witnesses identify composed actions and reconstruct both member states without parsing separator-joined keys.
- Keep unsupported monitored compositions explicit, and verify independent safety receipts survive progress errors. Fix discovered bugs only within the existing checker contract.

### Investigation targets
**Required:**
- `tools/umpire/check/checking_test.go:968` - current progress cases.
- `tools/umpire/check/replay_test.go` - independent replay.
- `tools/umpire/internal/engine/progress_test.go` - bounded graph cases.
- `tools/umpire/internal/engine/compose.go:67` - inherited fairness semantics.
- `tools/umpire/check/compose.go:44` - start product.

### Key context
Re-anchor paths and interfaces against completed fn-140/fn-141 and the approved schema/package moves before editing. Keep this work outside the activity batch and serialize shared regeneration with it and the schema chain; no new spec-close dependency is implied.

### Quick commands
```bash
go test -tags test_dep ./tools/umpire/check ./tools/umpire/internal/engine
```

## Acceptance
- [ ] Each R4 disposition has an independent fixture with the expected evidence and replay outcome.
- [ ] Finite-bound, fairness/synchronization, all-start and hole/limit boundaries have negative controls; no incomplete path passes.
- [ ] Old machine/composition pins and identity collision protections remain intact; no broad new verification framework is introduced.

## Done summary
Boundary/replay/structured-fairness checkpoints integrated earlier (d01aefebf1 and task .2 commits).
Added the lifted-fixture pin: a fair path (flowing: transfer and right.send fair) still misses
`within = 2` because the unrelated left drain takes the last step; witness `left_fill-high,
transfer-high, left_drain` replays. Engine package and focused check tests pass in this tree. Full
gates/review remain for Batch 5b.

Blocked:
OWNER HOLD (2026-10-16): Umpire framework DSL is the top priority. This integration task is not admitted while fn-188.3 completes final native public-client verification and formal batch review. Preserve its existing source checkpoints, evidence, dependencies and unfinished acceptance obligations; this scheduling hold claims no implementation or proof completion. Completion of fn-188.3 does not automatically release this task: select later IR/exporter/schema/Go/backend work explicitly. Any model/temporal work remains independently excluded entirely, including disabled source contents, until the owner restores that scope.
## Evidence
- Commits: d01aefebf1 (integrated), f7fcb54a67
- Tests: engine PASS; focused check/ir as in task .2 (`.flow/tmp/fn150/go-focused.log`)
- PRs:
