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
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: