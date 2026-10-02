---
satisfies: [R2, R12, R18, R19]
---
# fn-115-make-the-scala-model-the-model-and.7 Separate reader admission concerns and trim live checker and lowering surfaces

## Description
Separate reader admission concerns and trim live checker and lowering surfaces. Implements R2, R12, R18, R19 using the reviewed parent contracts.

**Size:** M
**Files:** tools/umpire/model loader/validator and internal checker; tools/umpire/lower; focused tests
**Touches:** [model/umpire/**, tools/umpire/model/**, tools/umpire/lower/**, tools/umpire/conformance/**, tools/umpire/export/**, tools/umpire/explore/**]

### Approach
- First deliver the owner-approved early fn113 Part A / R1 dead-code removal after task6 relocation: confirm callers, remove unused Canonical.scala, Lower.scala and Alterer plumbing under model/umpire, and retain declarations still read by tests, Coverage, Search, Refine or Compose. Record borderline retained declarations briefly. Preserve original archive copies and surviving comments. Validate with existing Scala DSL/Model/lifter checks and unchanged IR goldens; native evaluator retirement remains fn113 Part C. This is implementation of fn113 R1 early, not a second independent cleanup.
- The conductor owns shared module-map and migration-manifest updates after this task returns; return any required changes in the task-specific handover instead of editing those shared files.
- Split loading from admission/validation along the existing concerns identified in load.go; preserve diagnostics and error ordering rather than adding validation. Give each concern focused tests through the intended interface.
- Audit exports and actual production/test callers in the moved checker and lowering. Remove unreachable implementation, unexport package-only declarations and merge forwarding-only helpers; retain functionality with live consumers.
- Record live tooling line counts and the exported-surface inventory before/after. Keep aliases only where they implement the reviewed reader surface, not as historical package compatibility wrappers.
- Apply the owner-requested verification speedups to measured duplicate setup where they fit this cleanup. The instrumented task6 first run (.flow/tmp/fn115-6/full-first.jsonl) measured reader migration goldens at 144.46s, projection semantics at 72.18s and export tampered-dump checks at 48.7s (overlapping package times, not wall-time totals). In export/quint_test.go, each tamper subtest rebuilds the same untampered dump via encodeDump; evaluate building once and independently decoding each mutation. Reader migration helpers also bind the same input repeatedly. Preserve every assertion, original/mapped independence and mutation isolation; compare equivalent focused timings before claiming improvement. Do not add global caches or a new framework.
- Run mutation-sensitive goldens and all reader/lowering/conformance/export/exploration callers at the required gate; use affected focused tests during edits and after fixes per MILESTONES.md.

### Investigation targets
**Required** (current paths at planning time; follow the recorded move map after relocation):
- `model/scalav2/goir/load.go:18`
- `model/scalav2/goir/load.go:41`
- `model/scalav2/goir/load.go:1552`
- `model/scalav2/goir/load.go:1891`
- `model/go/umpire`
- `model/go/caseproducer`

### Quick commands
CC=/usr/bin/clang mise exec -- go test -tags test_dep ./tools/umpire/...; make lint-code-fast

### Execution constraints
Preserve the authorized uncommitted baseline and comments except the explicit R25 historical-attribution change. No staging, commits, worktrees or recursive deletion. Once task 2 exists, run the complete golden verification after every task. Resolve task-1 map choices before using projected destination names; capture any change in the map and downstream task briefs before work.
## Acceptance
- [ ] Loading and validation have separate concern files and focused tests without changed admission/diagnostic behavior.
- [ ] Every remaining exported checker/lowering declaration has a real outside-package caller or is removed/unexported; no forwarding-only package remains.
- [ ] Before/after line counts and public interface inventory are recorded, and all goldens and affected callers pass.

## Done summary
Removed the unused Scala framework code (fn-113 Part A): `Canonical.scala`, `Lower.scala`, the `Alterer` plumbing and helpers left without a caller; `model/umpire` non-test went from 2,911 to 2,415 lines. Split `tools/umpire/model/load.go` into `load.go`, `validate.go`, `validate_keys.go`, `validate_realization.go`, `operand.go` and `payload.go` as a line-for-line move with focused tests. Trimmed the checker, reader and producer surfaces: `tools/umpire` production Go went from 27,799 to 25,890 lines, checker exports from 248 to 126. The typed checker declaration layer is test support now.

Full `./tools/umpire/...` Go suite (1,916 passed), functional compile, `lint-code-fast`, `lint-model` and the model gate pass; goldens, checked-in IR and Cases are unchanged. Reader golden tests are faster (projection 63.8 s to 43.1 s, single samples).

Independent review (Claude Fable, fresh context) returned SHIP in round 1; its P2 follow-ups are applied and verified with focused tests and lint. Open follow-up: production branches that only typed test fixtures reach remain in the checker (listed in the handover). `Table.Stuck` and the export package's 46 exports stay exported as recorded exceptions. Handover: .flow/tmp/fn115-7-summary.md; evidence: .flow/tmp/fn115-7-evidence.json; review: .flow/tmp/fn115-7-review/round1-review.md. No agent commits.
## Evidence
- Commits:
- Tests: CC=/usr/bin/clang mise exec -- go test -count=1 -json -tags test_dep ./tools/umpire/... (exit 0, 282 s wall, 16 packages, 1916 passed, 12 opt-in skips; .flow/tmp/fn115-7/full-go.jsonl), CC=/usr/bin/clang mise exec -- go test -tags 'test_dep integration canary_harness' -run '^$' ./tests (exit 0; functional-compile.log), GOLANGCI_LINT_FIX=false CC=/usr/bin/clang mise exec -- make lint-code-fast (exit 0, 0 issues; lint-code-fast.log), CC=/usr/bin/clang mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (exit 0 after the Scala removal and again after the last Go change; check-model.log, check-model-final.log), mise exec -- make lint-model (exit 0; lint-model.log), git diff --check (exit 0), review fix: CC=/usr/bin/clang mise exec -- go test -count=1 -json -tags test_dep ./tools/umpire/model/... ./tools/umpire/conformance/... (exit 0, 1284 passed; review-fix/focused-go.jsonl); make lint-code-fast (exit 0, 0 issues; review-fix/lint-code-fast.log), Independent review round 1 SHIP (claude-fable-5-1); .flow/tmp/fn115-7-review/round1-review.md
- PRs: