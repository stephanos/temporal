---
satisfies: [R3, R5]
---
# fn-64-umpire-case-runtime.3 Implement deterministic Contract evaluation

## Description
Extend task 12's generic verification module with the per-Run Evaluator (R3), independent of Temporal and
property-specific operators. Keep live monitoring and offline evaluation on one prepared transition
implementation and the internal execution MonitorFactory contract.

**Size:** M
**Files:** `tools/umpire/verification/**`, verification package documentation and focused tests
**Touches:** [tools/umpire/verification/**]

## Approach
- Consume task 12's admitted finite ordered transition machines, typed predicates, event-kind indexes, terminal
  states, supporting-event references, typed single-assignment scalar captures, and explicit
  safety/liveness horizons. Predicates read pre-transition captures; assignments and state changes
  commit atomically. Captures retain the source event and remain local to their rule and Run.
- Implement the internal execution MonitorFactory contract so every Run receives fresh state; it is
  the only production Monitor factory. Task 13 binds it through `PrepareCase`; this task proves
  factory construction using task 11's immutable Program Observation/bounds view.
- Before transitions on every recorded event, expire pending liveness when its elapsed coordinate
  is greater than or equal to the Run-relative deadline. Only earlier witnesses qualify; never arm
  an invisible timer or use target timestamps. Completed closure before a pending deadline is
  inconclusive. Once execution/evaluation is incomplete, time alone cannot establish a new absence
  violation; previously proved violations remain.
- Preserve the useful live/offline distinction from legacy Run Evaluation while deleting all
  scenario/property-specific semantics at cutover.

## Investigation targets
**Required** (read before coding):
- `tools/umpire/portableevaluation/property.go:14-247` — property-specific behavior to eliminate
- `tools/umpire/portableevaluation/evaluator.go:16-35` — current evaluation entrypoint
- `tools/umpire/runevaluation/run_evaluation.go:71` — legacy offline evaluation boundary
- `tools/umpire/runevaluation/result.go` — existing verdict/result representation
- `.plans/UMPIRE_CASE_RUNTIME_DESIGN.md` — Contract IR and approved monitor-machine semantics

**Optional** (reference as needed):
- `tools/umpire/runevaluation/README.md` — live/offline documentation to replace, not preserve as API

## Key context
Transition order is semantic. Indexing may reduce candidates by event kind but must preserve the
same first matching transition, bad prefix, and closure result in both modes.

## Acceptance
- [ ] Safety tests prove exact first-violation prefix and synchronous Stop. Liveness tests cover
  witnesses before/at/after deadline, including a 5s deadline, 6s witness and 7s closure; expiry is
  checked before transitions on observation, timeout and closure events.
- [ ] Completed early closure is inconclusive with completed disposition; incomplete observation
  at/after the deadline cannot manufacture an absence violation. Live/offline results agree.
- [ ] Capture tests correlate a prior scheduled event ID with matching and mismatched later IDs,
  retain source-event references, isolate rules/Runs, and reject unguarded missing reads, repeated
  writes, wrong types, and count/byte/work overflow at the appropriate phase.
- [ ] Live and offline evaluation yield byte-equivalent transition traces, supporting-event
  references, and Verdicts for completed, stopped, and incomplete Runs, including time horizons.
- [ ] Contracts cannot access Slots, opaque capabilities, raw payloads, or undeclared fields.
- [ ] Malformed states/transitions/predicates/captures/horizons and static work overflow fail
  preparation; Monitor error/timeout yields incomplete/inconclusive unless violation is already proved.
- [ ] Event-kind indexing stays within declared per-event work bounds without changing ordered
  semantics.
- [ ] No public API permits replacing the prepared Case's Contract Monitor.
- [ ] `go test -count=1 -tags test_dep ./tools/umpire/verification/...` passes.

## Done summary
Implemented the generic per-Run Contract Evaluator and shared bounded typed IR expression/path runtime, with atomic event commits, capture/source support, indexed ordered transitions, exact Stop prefixes, and shared live/offline evaluation. Added append-only failure-coordinate schema and Go/generated/authored Lean representations; no scheduler, root facade, legacy compatibility, libraries, worktrees, or worker-authored commits were introduced.

Baseline: green focused IR/execution/verification tests, formatting and authorized scoped no-fix Go lint before edits. Required investigation and approved design read; reused static Contract admission, immutable ProgramView and IR validation. Legacy property evaluators were consulted only for boundaries, not reused as scenario semantics.

Tests cover deadline before/at/after (including 5s/6s/7s), timeout/closure expiry, completed early closure, execution incompleteness, late witness exclusion, per-Run/rule captures, correlation mismatch/match, exact first-match Stop and supporting sequences, concurrent prepared reuse, atomic cancellation across rules, failed-callback cutoff, and successful-commit cancellation. IR tests cover typed comparisons including exact integer widths/naturals/float32/NaN/zero, presence/boolean short circuit, nested/map/wildcard projection and exact runtime budgets. Wildcard absence tests cover empty, all-present, mixed and all-missing inputs; absent branches do not silently filter or reorder values. Existing task12 admission regressions remain intact. Schema tests round-trip failure-status presence in wire/ProtoJSON; Lean checks preserve authored failure fields.

Validation: make proto; make umpire-gen-lean-api; final fix1 tagged IR/execution/verification/schema/API tests, race tests, make fmt-imports, scoped no-fix make lint-code all exited0. make umpire-build-model final2 exited0 (346 jobs). make lint-model retains only the known unchanged ActivationBinding.controller.injEq simpNF failure; inherited baseline proof is fn64-task11-lean-lint-baseline-proof.log. An authored Lean record syntax error was fixed. An overlapping builtin lint/build was stopped after missing-artifact failures; the owned serial builtin retry exited0, without cache cleanup. No global lint or full live/cutover gate is claimed.

Gate classification: FULL. Initial tree-base classify was unsupported (CLI expects commits), then actual-base classify succeeded. NO_RECEIPT: dirty staged worktree prevents a truthful green HEAD receipt. All changes remain staged for the user; snapshot review uses task-start tree plus owned files and excludes preserved external artifacts.

Review round1 returned NEEDS_WORK for cancellation during Close validation/result construction. Deterministic initial/mid-scan/final-boundary cancellation regressions reproduced it. The fix maintains Verdict data at atomic Observe commits, polls cancellation during closure validation and checks it at return, then transfers the frozen result once; cancelled/repeated Close and preserved proof/replay tests pass. No prior support history is cloned per event.

stage: impl-review - ran (codex:gpt-5.6-sol:high; NEEDS_WORK → SHIP; final verdict 2026-09-04T23:00:40.866042Z; /tmp/impl-review-receipt-fn-64-umpire-case-runtime.3.json)
stage: plan-sync - skipped(config: planSync.enabled != true)

Final review used owned tree 6061ff844493f6e2be2baefe4a1317d4cd46c4c0 against task-start tree 9809451f2a0230299333ab3786005d55bda9b769. Owned source matches that reviewed tree. Actual HEAD advanced externally from 156c253a57422518ba7e1266755ff5392a26698b to user WIP commit 0fc128eb50a8c8bdc1db3eb1dfdf1445130910bf during verification; the evidence commit list records that external advancement, not a worker-authored commit. Task README and lifecycle receipts remain staged for the user. The snapshot helper retains normal read-only review, round caps and receipt persistence and overrides only tree capture.

Authoritative final Go logs: .flow/tmp/fn64-task3-fix1-{tests,race,format,lint}.log, all exit0 before final review. Unchanged Lean/generator gates remain as recorded above. Reviewer test execution was blocked by its read-only sandbox; owned gate exit codes supply execution evidence. Final SHIP has no open findings. Memory capture: bug/runtime-errors/monitor-closure-must-honor-cancellation-2026-09-04.

Scheduler obligations for tasks4/9: record Observe's returned error coordinate without retroactively invalidating a successful return on later context cancellation; latch execution_incomplete before horizon processing; Close transfers its frozen result once, including on error, and rejects later callbacks. Root composition remains task13.
## Evidence
- Commits: 0fc128eb50a8c8bdc1db3eb1dfdf1445130910bf
- Tests: baseline: green (fn64-task3-baseline-tests.log, fn64-task3-baseline-format.log, fn64-task3-baseline-lint.log), make proto (exit 0; .flow/tmp/fn64-task3-proto.log), make umpire-gen-lean-api (exit 0; .flow/tmp/fn64-task3-lean-gen.log), CGO_ENABLED=0 TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-test-tmp go test -count=1 -tags test_dep ./tools/umpire/internal/ir/... ./tools/umpire/internal/execution/... ./tools/umpire/verification/... ./tools/umpire/cmd/umpire-gen-lean-api/... ./api/umpire/v1/... (exit 0; .flow/tmp/fn64-task3-fix1-tests.log), CGO_ENABLED=1 CC=/usr/bin/clang TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-test-tmp go test -race -count=1 -tags test_dep ./tools/umpire/internal/ir/... ./tools/umpire/internal/execution/... ./tools/umpire/verification/... (exit 0; .flow/tmp/fn64-task3-fix1-race.log), make fmt-imports (exit 0; .flow/tmp/fn64-task3-fix1-format.log), make lint-code GOLANGCI_LINT_BASE_REV=4c4e26ebdb15100387107f5d03daf5ce5fc01111 GOLANGCI_LINT_FIX=false (exit 0; .flow/tmp/fn64-task3-fix1-lint.log), make umpire-build-model (exit 0; .flow/tmp/fn64-task3-final2-model.log), cd model && lake --wfail lint --builtin-only --lint-only=.all,.extra,-.missingDocs (exit 0; .flow/tmp/fn64-task3-final2-lean-builtin.log), make lint-model (inherited failure: unchanged ActivationBinding.controller.injEq simpNF; .flow/tmp/fn64-task3-final-lean-lint.log; baseline .flow/tmp/fn64-task11-lean-lint-baseline-proof.log), gate classify: FULL; NO_RECEIPT: dirty worktree outside ignore set (.plans/FEATURE_AUTHORING_ASSESSMENT.md), Codex implementation review NEEDS_WORK → SHIP; .flow/tmp/fn64-task3-review2.log; owned source matches reviewed tree6061ff844493f6e2be2baefe4a1317d4cd46c4c0
- PRs: