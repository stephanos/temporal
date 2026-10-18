---
satisfies: [R4, R5, R9]
---
# fn-64-umpire-case-runtime.4 Implement generic DAG scheduling and Run recording

## Description
Integrate task 14's typed data plane and task 15's recorder/barrier into bounded controller DAG
scheduling for R4/R5/R9. Task 9 retains public Run, termination, drain/quarantine, cleanup and reuse.

**Size:** M
**Files:** `tools/umpire/internal/execution/scheduler.go`, scheduler fake-Host tests and narrow
integration changes to the typed stores/recorder
**Touches:** [tools/umpire/internal/execution/**]

## Approach
- Consume private compiled graphs, stable ready ordering and task 14 values; schedule ready
  controller nodes with permitted concurrency and stable entrypoint/activation/instruction/attempt
  coordinates. Each controller entrypoint activates once per Run. Bounds are ceilings, not an
  implicit retry policy: current controller instructions dispatch once, and repetition is authored
  in the DAG. Guard-false nodes complete dependency bookkeeping without effects or invented outcomes;
  successors can only inspect an absent result through admitted presence guards.
- Integrate typed request building, outcome snapshots, immutable Slot writes and response projections.
  Stage writes/events first, then publish through task 15's barrier with stable EmitEach indexes and
  causal/source IDs. Protocol non-success and timeout remain typed outcomes; missing unguarded
  values, malformed Host data and global-limit failure become incomplete with stable diagnostics.
- Reserve node-declared worker activations before their triggering controller effect. Validate exact
  count, uniqueness and identity of returned handles. Retain every nonnil returned effect/reservation
  handle, including partial/error results or malformed reservations, for task 9 cancellation/drain;
  fail admission without triggering the controller effect when reservation acquisition is incomplete.
- Hold task 15's admission boundary through reservation/effect acceptance and handle registration.
  Keep waits outside it. Stop admits no later controller dispatch/reservation; already accepted
  worker effects remain Host-owned, with replay-local DAG state and no per-SDK-instruction central
  scheduler or verification stream.
- Preserve in-flight ownership and seal ordinary stores under the recorder's closure boundary for
  task 9. Integrate recorder timeout/closure facts and post-close rejection without implementing
  cleanup orchestration or exporting a placeholder public Run.

## Investigation targets
**Required**:
- `tools/umpire/internal/execution/program.go:72` — compiled DAG and node internals
- `tools/umpire/internal/execution/prepare.go:348` — deterministic dependency order
- `tools/umpire/internal/execution/contracts.go:53` — Host effects/reservations and session boundary
- `tools/umpire/executor/portable_executor.go:115` — legacy orchestration seam to replace
- `tools/umpire/executor/portable_executor_test.go` — fake runtime patterns
- `.flow/memory/bug/integration/portable-execution-boundaries-must-2026-09-03.md`

## Acceptance
- [ ] Fake-Host integration tests cover dependency order, permitted concurrency, activation isolation,
  false guards, typed protocol/timeout branching, constructed requests, immutable Slots and ordered
  fan-out through the actual task 14/15 modules; no preparation or expression rebinding occurs.
- [ ] Reservation tests cover exact/partial/error/malformed returns and no triggering effect on
  failure; all acquired handles remain owned for task 9. Max-attempt bounds never imply hidden retries.
- [ ] Stop races allow neither new controller effects nor reservations, including the gap between
  returned handle and registration. In-flight waits remain outside the Monitor critical section.
- [ ] Integrated events have stable source/causal/attempt/index coordinates and central monotonic
  sequence/time. Duplicates/conflicts, missing values and recorder/global-limit errors produce the
  specified incomplete diagnostics; closure seals Slots and post-close input cannot mutate snapshots.
- [ ] No root/verification/concrete-Host import or public scheduler/store/Monitor construction is
  introduced. Package docs describe outstanding-handle/sealing ownership for task 9; tagged execution
  race tests, formatting and authorized scoped no-fix lint pass before implementation review.

## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: