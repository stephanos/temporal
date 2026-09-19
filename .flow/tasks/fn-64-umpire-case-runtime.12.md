---
satisfies: [R2, R3]
---
# fn-64-umpire-case-runtime.12 Compile static Contract and capture admission

## Description
Compile static Contract admission and capture analysis (R2, R3) using the shared IR and prepared
Program observation view. Runtime transition evaluation remains task 3.

**Size:** M
**Files:** `tools/umpire/verification/{prepare,captures}.go` and focused tests
**Touches:** [tools/umpire/verification/**]

### Approach
- Bind rule/state/transition IDs, kinds, terminal states, horizons and closed expression references
  against the immutable Program Observation schema and task 2's typed binder.
- Consume only task 11's immutable observation/bounds view; no scheduler or mutable Slot state is
  available to admission. Task 3's factory binds that same view to each fresh evaluator.
- Prove deterministic selection under declared first-matching transition order and reject malformed
  or unsupported predicates before I/O. Overlapping predicates retain their explicit ordered
  precedence; indexing must never reorder them or introduce unordered selection.
- Analyze capture types, reachable-path definite assignment or explicit presence guards, and
  single assignment. Predicates read pre-transition state; assignments atomically retain the
  matching Observation value and source-event support for later evaluation.
- Bound capture counts/bytes, machine states/transitions, expression depth and per-event/total work
  with overflow-safe arithmetic and immutable prepared indexes. No capture can read Slots or opaque
  capabilities. Consume explicit incomplete/closure and expiry semantics from the parent spec.
- Keep admission independent from target I/O and mutable evaluation state; task 3 supplies the
  production factory and live/offline interpreter using these prepared machines.

### Investigation targets
**Required**:
- `proto/internal/temporal/server/api/umpire/v1/contract.proto:30` — captures, transitions and limits
- `proto/internal/temporal/server/api/umpire/v1/value.proto:185` — Observation/capture expressions
- `proto/internal/temporal/server/api/umpire/v1/run.proto:12` — event kinds and failure coordinates
- `tools/umpire/evaluationcontract/contract.go:186` — existing bounded proto-surface checking pattern
- `.plans/UMPIRE_CASE_RUNTIME_DESIGN.md:245` — approved Contract semantics
## Acceptance
- [ ] Tests accept valid safety/liveness machines and reject duplicate/missing states or transitions,
  invalid terminal/horizon targets, unsupported predicates and unknown Observations. Overlapping
  valid predicates preserve deterministic first-matching declaration order.
- [ ] Capture tests cover matching types, cross-event correlation declarations, required presence
  guards, read-before-assignment, repeated assignment on any reachable path, conflicting branches,
  cycles, forbidden Slot/capability reads and retained supporting-event requirements.
- [ ] Count/byte/depth/work bounds and arithmetic overflow reject before evaluation; source mutation
  cannot change prepared Contract indexes or Observation binding.
- [ ] `go test -count=1 -tags test_dep ./tools/umpire/internal/ir/...
  ./tools/umpire/internal/execution/... ./tools/umpire/verification/...` and format/lint pass.
## Done summary
Implemented immutable static Contract admission and reachable capture analysis through verification.Prepare(Contract, Catalog, execution.ProgramView, ContractLimits). Runtime evaluation/factory and the root facade remain with tasks 3 and 13; no scheduler, mutable Slots, target I/O, new libraries, worktrees, or commits were introduced.

Baseline: green existing tagged IR/execution tests and scoped no-fix lint before edits. New verification package tests initially failed for missing implementation. Required investigation/specs were read; shared IR binding and immutable ProgramView were reused. Parent-authorized applicable gates exclude future live/runtime/cutover commands assigned to later tasks.

R2/R3 admission tests cover safety/liveness topology, duplicate/missing/invalid IDs and states/transitions, horizons/terminal targets, closed references, unknown Observations, forbidden Slots/outcomes/capabilities, first-match declaration order, capture type/correlation/support, pre-transition reads, presence guards, branch merges, alternative branches, repeated atomic assignment, safe/unsafe cycles, contradictory presence, unreachable assignments, count/byte/depth/runtime/static-work ceilings, overflow, source/index immutability, and bounded configuration exploration. Compiled paths obey Program fanout. Intrinsic RunEventKind binds against minimal catalogs and conflicting definitions reject.

Validation: final3 tagged IR/execution/verification tests, all3 race tests, make fmt-imports, and scoped no-fix make lint-code all exited 0 before final review dispatch. Authoritative logs are .flow/tmp/fn64-task12-final3-{tests,race,format,lint}.log. Repository-wide lint backlog remains inherited; this is only the authorized scoped lint result. Gate classification requested FULL; the green receipt was correctly refused because the user-authorized staged worktree is dirty, so no stale HEAD receipt was minted. Final owned source matches reviewed tree fc8e89dadee75cc66b1eb4acab430b9d16d6d1c3.

Review round 1 returned NEEDS_WORK for whole-response work charges on fixed-width captures/comparisons. Type-sensitive bounds fixed that, with a 16 MiB-response int64 correlation regression and undersized-work rejection. Additional reproduced regressions separated static binding work from runtime work, preserved authored depth at 2/64 via shared-budget IR Conditions, and aligned bound path fanout at 127/128/129. Round 2 returned SHIP but was dispatched before the final lint failure had been observed: this gate sequencing error is recorded, and round 2 was not used as completion evidence. The remaining complexity issue was refactored; all final3 gates explicitly passed before round 3 returned SHIP with no open findings. Reviewer test execution was sandbox-blocked; worker exit-code receipts are authoritative.

stage: impl-review - ran (codex; NEEDS_WORK, SHIP, final SHIP; final round gated after all tests/race/format/scoped lint passed)
stage: plan-sync - skipped(config: planSync.enabled != true)

No commits were authored; work is staged for the user. Existing/external fn-65, Nexus2 design, FEATURE_AUTHORING, and UMPIRE4_ORDER content was preserved; catch-all staging swept external additions into the real index, reported to conductor. Review used a temporary index with the exact task-start tree plus owned IR/verification blobs, preserving the real index and excluding external artifacts. The normal Flow-Next review orchestrator, caps, read-only reviewer, and receipts were retained; only snapshot capture was adapted for the user’s no-commit instruction. The first snapshot helper attempt failed before dispatch and consumed no review verdict. Memory capture records the fixed type-sensitive bounds.
## Evidence
- Commits:
- Tests: baseline: green (tagged IR/execution tests and scoped no-fix lint), go test -count=1 -tags test_dep ./tools/umpire/internal/ir/... ./tools/umpire/internal/execution/... ./tools/umpire/verification/..., CC=/usr/bin/clang CGO_ENABLED=1 go test -race -count=1 -tags test_dep ./tools/umpire/internal/ir/... ./tools/umpire/internal/execution/... ./tools/umpire/verification/..., make fmt-imports, make lint-code GOLANGCI_LINT_BASE_REV=4c4e26ebdb15100387107f5d03daf5ce5fc01111 GOLANGCI_LINT_FIX=false, final3 tests/race/format/lint exit 0; logs .flow/tmp/fn64-task12-final3-{tests,race,format,lint}.log, flowctl gate classify: FULL; gate receipt: NO_RECEIPT (dirty staged worktree; no unwarranted HEAD receipt), git diff --exit-code fc8e89dadee75cc66b1eb4acab430b9d16d6d1c3 -- tools/umpire/internal/ir tools/umpire/verification, review sequencing exception: round2 dispatched before final lint failure observed; excluded from completion evidence; final round3 dispatched only after explicit green gates
- PRs: