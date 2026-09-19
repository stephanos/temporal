---
satisfies: [R3, R4, R5, R9]
---
# fn-64-umpire-case-runtime.15 Implement append-only recorder and Monitor barrier

## Description
Implement the recorder and shared Monitor/admission barrier for R3/R4/R5/R9 without scheduling a DAG.
Task 4 drives admission/publication; task 9 supplies termination, drain, cleanup and final disposition.

**Size:** M
**Files:** `tools/umpire/internal/execution/recorder.go`, recorder/barrier tests, `tools/umpire/verification/evaluator.go`, focused failure-ordering tests and package documentation
**Touches:** [tools/umpire/internal/execution/recorder*, tools/umpire/internal/execution/README.md, tools/umpire/verification/evaluator.go, tools/umpire/verification/evaluator_failure_test.go, tools/umpire/verification/README.md]

### Approach
- Own per-Run central sequence, monotonic elapsed time, source-ID deduplication, diagnostics and
  open/closed publication state. Accept producer coordinates/causes and immutable typed fact batches;
  assign authoritative sequence/time centrally, including timeout and closure facts.
- Compare duplicate producer content before assigning a new sequence/elapsed coordinate: source ID,
  kind, activation/instruction/attempt/index, causes, outcome, Observations and failure marker are
  semantic. Recorder-assigned sequence/arrival elapsed are excluded. Identical redelivery appends
  nothing and invokes no Monitor; conflicting reuse latches a stable invariant failure.
- Expose one serialized admission/publication boundary: append and synchronous Observe cannot race
  admission of controller effects or reservations. A guarded admission operation retains ownership
  of returned handles before releasing the boundary; blocking Wait/drain occurs outside it. Tests
  use injected admission operations, not a second scheduler.
- Preserve the approved failure-before-violation precedence in the shared live/offline evaluator:
  incompleteness freezes new transitions, captures and support before processing the marked event;
  only violations committed before failure survive. Cover both orderings and a same-event failure.
- Latch execution incompleteness before horizon processing. Record Observe's returned-error event
  in `evaluation_failure_sequence`; a successful callback return remains committed even if its
  context is cancelled afterward. Reuse private Monitor contracts without importing verification.
- Bound recording/diagnostics and latch capacity failures once without recursive error-event
  generation or exceeding ceilings. Preserve the committed prefix and earlier proof when no more
  events fit. Close accepts a supplied terminal outcome, invokes Monitor.Close once, transfers
  independent frozen Run/Verdict data, and prevents later publication/admission even on callback error.
- Offer a closure critical section for sealing ordinary stores with recording; task 4 wires task
  14's store, while task 9 closes Host sessions/opaque bridges. Post-close facts go only to a bounded
  Host diagnostic sink; sink error/full capacity cannot mutate returned data or trigger recursive retries.

### Investigation targets
**Required**:
- `tools/umpire/internal/execution/contracts.go:20` — callback/Host ownership contracts
- `tools/umpire/verification/README.md` — callback commit/failure-coordinate handoff (no import)
- `proto/internal/temporal/server/api/umpire/v1/run.proto` — immutable recorded facts and diagnostics
- `tools/umpire/internal/execution/program.go:45` — prepared Run limits
- `.flow/memory/bug/runtime-errors/monitor-closure-must-honor-cancellation-2026-09-04.md`
- `.flow/memory/bug/integration/portable-execution-boundaries-must-2026-09-03.md`

### Early proof
Block a synchronous fake Observe callback while another caller tries admission; prove it cannot
enter until Observe returns and is rejected after Stop. No Host wait is held inside the barrier.

## Acceptance
- [ ] Recorder tests cover central sequence/elapsed on timeout/closure, duplicate redelivery with
  changed arrival time, conflicting content, bounded capacity/diagnostics and immutable input/output.
- [ ] Barrier race tests prove append/Observe excludes concurrent admission and Stop prevents later
  effects/reservations; callback error records the exact uncommitted coordinate, while cancellation
  after successful return preserves the committed prefix. No goroutine timeout wrappers are used.
- [ ] Execution/recorder/limit failures latch incomplete before applicable horizon processing;
  exhaustion cannot recurse or overrun limits. Supplied closure outcomes preserve prior proof,
  context-cancelled/failed Close is terminal, and repeated Close cannot transfer mutable data again.
- [ ] Closure atomically rejects later publication/admission; injected store sealing and bounded
  post-close diagnostics are tested, including failing/full sinks, without mutating serialized
  Run/Verdict. Task 9 retains real effect/bridge closure, cleanup and quarantine orchestration.
- [ ] Failure before or on a potentially violating event remains incomplete/inconclusive, while
  an earlier committed violation survives later failure. Shared evaluator tests prove identical
  live/offline decisions, supporting sequences and Verdicts; recorder tests prove failure handoff.
- [ ] The module remains private and imports neither root, verification nor concrete Temporal Hosts;
  documentation names the task 4/9 integration obligations. Tagged execution and verification race tests, formatting
  and authorized scoped no-fix lint pass before configured implementation review.


## Done summary
Implemented the private append-only recorder and shared synchronous Monitor/admission/closure barrier. The approved review fix also freezes shared live/offline Contract transitions at execution incompleteness, preserving only violations committed before failure; task15 now covers that R3 integration seam as well as its scoped R4/R5/R9 responsibilities.

Baseline: green focused execution/verification tests, execution race tests, formatting and authorized scoped no-fix lint, recorded incrementally in `.flow/tmp/fn64-task15-baseline-results.json`. Required task/spec/UMPIRE4/contracts/schema/readme/memory investigation completed before implementation; reused the existing Monitor contract, IR surface validation and injected value-store seam. No scheduler, public Run placeholder, root/verification imports into execution, concrete Host integration, libraries, worktrees, generated schema changes or worker commits.

Recorder tests cover centrally assigned sequence/elapsed including timeout and closure, arrival-independent semantic deduplication, source-content conflicts, immutable inputs/outputs and Monitor snapshots, bounded event/diagnostic/copy-work capacity, no partial publication on failed store commit, failure-before-horizon handoff, exact Observe failure coordinate, and successful-return cancellation semantics. Blocking fake Observe proves admission exclusion; injected admission retains partial handles before unlock without Wait/drain. Close seals the injected store in the critical section, transfers once even on cancellation/error, preserves earlier proof, rejects further admission/publication, and limits successful/failing post-close sink calls without changing serialized results. README states task4 scheduler/reservation/value-batch integration and task9 cleanup/drain/quarantine/Host/bridge closure obligations.

Review round1 identified failure-before-violation precedence. Focused red regressions reproduced a later positive violation and a violation on the first incomplete event. The conductor approved the narrow shared evaluator/test/README change and task/spec R3 coverage update. The evaluator now freezes transitions, captures and support before processing an incomplete event while retaining validated coordinate progression and prior proof. `TestEvaluatorFailureViolationOrderingAndReplay` covers failure-before, same-event failure and violation-before-failure, with live/offline Verdict/support/trace equality; `TestRecorderFailureViolationOrdering` covers recorder handoff. Existing failed-callback cutoff, horizon and successful-return cancellation tests remain green. No recorder-only masking or fabricated failure coordinates were introduced.

Final authoritative gates: `.flow/tmp/fn64-task15-final3-results.json` and its tests/race/format/lint logs. Execution+verification tagged tests and tagged race tests, `make fmt-imports`, and scoped no-fix `make lint-code GOLANGCI_LINT_BASE_REV=4c4e26ebdb15100387107f5d03daf5ce5fc01111 GOLANGCI_LINT_FIX=false` all exited0 before final review at 2026-09-05T00:50:27.868096+00:00. Go uses the physical repo `.flow/tmp/go-test-tmp`; normal CGO_ENABLED=0, race CGO_ENABLED=1 CC=/usr/bin/clang. Final1 lint findings were authored unchecked error returns and test-switch defaults, fixed before final2/final3; no inherited global lint success is claimed. Full live integration, Lean and regression cutover gates remain assigned to task10 because their code was not changed.

Gate classification: FULL. NO_RECEIPT: dirty staged worktree prevents a truthful green HEAD receipt; command logs and exit codes are authoritative. Reviewer's read-only sandbox could not run race tests; writer gate results are the execution proof. Final SHIP has no open findings. Memory captured as `bug/runtime-errors/freeze-contract-transitions-when-2026-09-05`; its issue is distinct from existing cancellation-during-Close memory despite module overlap.

stage: impl-review - ran (codex:gpt-5.6-sol:high; NEEDS_WORK → SHIP; final verdict 2026-09-05T00:51:44.032801Z; /tmp/impl-review-receipt-fn-64-umpire-case-runtime.15.json)
stage: plan-sync - skipped(config: planSync.enabled != true)

Task-start HEAD/base commit: `0fc128eb50a8c8bdc1db3eb1dfdf1445130910bf`. Actual HEAD: `0fc128eb50a8c8bdc1db3eb1dfdf1445130910bf`. Scoped review compared task-start tree `5b5df9e5648f12bb6c89217a929384f07b5b18e9` to owned staged tree `55306d076f6b732b08a55b053607a90bf4314556`; source matched that reviewed tree before lifecycle completion. The snapshot helper overrides only bundled flowctl tree capture, preserves normal read-only review/round counters/receipts, and records actual HEAD separately. All implementation, memory and lifecycle receipts remain staged for the user; no commits or pushes were authored.
## Evidence
- Commits:
- Tests: go test -count=1 -tags test_dep ./tools/umpire/internal/execution/... ./tools/umpire/verification/..., go test -count=1 -race -tags test_dep ./tools/umpire/internal/execution/... ./tools/umpire/verification/..., make fmt-imports, make lint-code GOLANGCI_LINT_BASE_REV=4c4e26ebdb15100387107f5d03daf5ce5fc01111 GOLANGCI_LINT_FIX=false, baseline: green (.flow/tmp/fn64-task15-baseline-results.json), final gates: all exit 0 (.flow/tmp/fn64-task15-final3-results.json), NO_RECEIPT: dirty staged worktree; captured command results are authoritative
- PRs: