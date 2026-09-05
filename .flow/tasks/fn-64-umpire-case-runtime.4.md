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
Implemented the private bounded controller DAG scheduler using task14's compiled typed data plane and task15's recorder/admission barrier. Controllers activate once per Run, enabled nodes dispatch once with stable ready ordering and concurrent bounded waits, false guards release dependencies without invented outcomes, and outcome/Slot/fanout facts publish atomically before successors become ready.

Reservation acceptance validates exact count, zero-based unique ordinals, unique IDs and matching origins before the triggering effect; every nonnil partial/error/malformed returned handle remains owned. Frozen reservation identity is independent of the Host's Consume activation coordinate: completion diagnostic facts use the controller origin and trigger cause, not an invented worker activation. Recorder Stop/failure/closure signals wake scheduling while waits remain outside its critical section. Value-Slot notifications, opaque Bridge readiness/consumption and post-close completion diagnostics are integrated. README specifies task9 ownership of bounded Run context, cancellation/drain/quarantine/cleanup/Host closure and mandatory publication of completions accepted before its drain boundary. No public Run placeholder, worker SDK instruction stream, root/verification/concrete Host import or library was introduced.

Baseline: green execution tests, execution race tests, formatting and authorized scoped no-fix lint, with exact commands/environments/timestamps/exits in `.flow/tmp/fn64-task4-baseline-results.json`. Required spec/UMPIRE4/contracts/compiled graph/legacy executor and boundary memories were investigated; existing typed values, immutable work ceilings, recorder and test fixtures were reused, with new scheduler code because no equivalent generic scheduler existed. The first focused test failed before scheduler implementation and then passed. Initial final1 lint found authored function complexity and a test switch default; focused refactoring fixed both without bypasses.

Acceptance tests cover actual request construction and response projection, immutable ordinary Slots, readiness, ordered fanout indexes, stable source/causal/attempt coordinates and monotonic sequence/time, dependency order, independent concurrent waits, repeated Run state isolation, absent false-guard outcomes, timeout/protocol branches and no hidden retry, reservation exact/partial/error/nil/duplicate/crossed returns and partial effect ownership, malformed outcomes, attempt/event ceilings, source conflict, worker failure, Stop before trigger/reservations and during acceptance, opaque readiness/completion and nil bridge/capability failure, and closure snapshot/sealing behavior. Existing data-plane and recorder tests retain bounded-copy, missing-value, duplicate, sealing and Monitor failure coverage.

Authoritative final gates: `.flow/tmp/fn64-task4-final2-results.json` and its tests/race/format/lint logs; all four commands exited0 before review. Go tests use `-tags test_dep`, normal CGO_ENABLED=0, race CGO_ENABLED=1 CC=/usr/bin/clang, physical repo `.flow/tmp/go-test-tmp`. Scoped lint is `make lint-code GOLANGCI_LINT_BASE_REV=4c4e26ebdb15100387107f5d03daf5ce5fc01111 GOLANGCI_LINT_FIX=false`; no inherited global lint success is claimed. Gate classification FULL. NO_RECEIPT: dirty staged work prevents a truthful HEAD green receipt. Broad live/Lean/regression gates remain task10's responsibility.

stage: impl-review - ran [2026-09-05T01:10:33.840477+00:00..2026-09-05T01:15:40.926021+00:00] | codex:gpt-5.6-sol:high | first-pass SHIP; /tmp/impl-review-receipt-fn-64-umpire-case-runtime.4.json
stage: plan-sync - skipped(config: planSync.enabled != true)

Review reports no surviving findings and R4/R5/R9 met within task4's assigned scope. Reviewer test execution was blocked by its read-only sandbox; writer exit-code evidence above is authoritative. The approved task-local launcher overrides only bundled flowctl snapshot capture, preserving normal review counters/receipts and real git index. Task-start tree `844b78c40f084011d68ba82841d5622c96f370b2` was compared to owned staged tree `43b6ac3dc2e89bfabcdb7ad118c58f8dae9ff847`; owned source matched that tree after SHIP. Start and actual HEAD are `0fc128eb50a8c8bdc1db3eb1dfdf1445130910bf`, recorded separately in `.flow/tmp/fn64-task4-review-snapshot.json`. No commits, pushes or worktrees: implementation and lifecycle receipts remain staged for the user.
## Evidence
- Commits:
- Tests: baseline: green (.flow/tmp/fn64-task4-baseline-results.json), go test -count=1 -tags test_dep ./tools/umpire/internal/execution/... (exit 0; .flow/tmp/fn64-task4-final2-tests.log), go test -count=1 -race -tags test_dep ./tools/umpire/internal/execution/... (exit 0; .flow/tmp/fn64-task4-final2-race.log), make fmt-imports (exit 0; .flow/tmp/fn64-task4-final2-format.log), make lint-code GOLANGCI_LINT_BASE_REV=4c4e26ebdb15100387107f5d03daf5ce5fc01111 GOLANGCI_LINT_FIX=false (exit 0; .flow/tmp/fn64-task4-final2-lint.log), gate classify: FULL, NO_RECEIPT: staged working tree; final2 command exit receipts are authoritative, Owned source comparison against reviewed staged tree43b6ac3dc2e89bfabcdb7ad118c58f8dae9ff847: exit0
- PRs: