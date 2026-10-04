# Task 16 source handover

Simulation callers now submit complete typed protocol transitions to one private
progress owner. Admission and forwarding validate the response identity,
participant incarnation, epoch and aggregate arrivals before changing any
progress. The owner holds the arbiter mutex through credit consumption,
reservation, response-phase changes, waiter wakes and settlement. Both required
historical-defect regressions fail against the old source and pass against the
new owner.

The source and tests are frozen at the identities in [evidence.json](evidence.json).
No live commands remain. HEAD remains `0dd05b313acd0986312da7fd3159520e6a21f1bf`
on branch `gomad`. The user owns commits, so `commits` is empty. Task 16 remains
`in_progress` for conductor source audits and native acceptance. The worker made
no Flow status, review, completion or history mutation.

## Ownership and migration

`simulation_progress.go` owns the closed typed event interface, validation,
aggregate credit changes and coordinator response phases. Its response keys
include the participant pointer and coordinator request ID. Model transport
request IDs remain in the transport's independent namespace. Aggregate arrival
credits never select a particular model operation or replay identity.

The eight legacy external-accounting methods, external `runnable` and `remove`
entrypoints, `simulationResponseBarrier`, `coordinator.responses`, and all
begin/release/reserve barrier helpers are removed. The migration covers the 25
actual old accounting sites, plus runnable and removal boundaries. The task's
historic 27-site count was not used as a reduction claim.

| Protocol boundary | Complete owner transition |
| --- | --- |
| Coordinator request admission | Validate credits and duplication, consume credits and reserve its response |
| Stop/crash control forwarding | Validate both participants and response identity, consume source credits and reserve both sides atomically |
| Wait acceptance | Validate credits and duplication, consume arrivals, wake the installed waiter and reserve the response |
| Wait suspension/resumption | Change the private response phase and its accounting in one transaction |
| Completion-only response reservation | Reserve once when absent; preserve an existing wait reservation or suspended phase |
| Coordinator response delivery | Deliver the private reservation, including reserve-and-deliver from a suspended wait, and retire its identity |
| Node/model request forwarding | Validate incarnation, epoch and acknowledgements before adopting time and transferring work |
| Model dispatch and unavailable dispatch | Deliver coordinator work or terminate its unavailable handling reservation |
| Model arrival and known abandoned late discard | Transfer legitimate credits to a live node, or consume only coordinator credits for an abandoned reply |
| Participant runnable/removal | Wake the waiter or error it and remove the incarnation under the same lock |

Wait completion now performs one resumption transaction rather than sequencing
completion and resumption accounting. The redundant coordinator
`completionPending` flag and `retainNodeCompletion` bookkeeping are removed.
Process completion channels, hard crash/reap and lifetime synchronization remain
with the coordinator and process owners.

`simulation_model.go` is byte-identical to the predecessor candidate. Its bounded
pending/abandoned correlation maps, cancellation race and writer lock remain.
Domain handlers still own committed and uncommitted model mutations. Runtime
native timers, generated time-wire consumers, schemas, overlay inputs and seed
separation are unchanged by task 16.

## Regression and characterization evidence

Before production changes, the focused stock baseline passed at 0.004 seconds
and the race baseline passed at 1.019 seconds. The strengthened two-test run
then failed with exit 1 in [regressions-red.log](regressions-red.log).

- `TestSimulationTimeProgressPreexistingMalformedWaitWakesQuiescence` failed
  immediately because the rejected wait woke its installed waiter. The corrected
  test requires that waiter to remain installed and subsequently advance with
  the other participant.
- `TestSimulationTimeProgressPreexistingDuplicateAdmissionConsumesArrival`
  observed External instead of Retry after the duplicate consumed a prior
  delivered credit. The corrected test requires Retry for that unconsumed reply.

The same selector passes with exit 0 in
[regressions-green.log](regressions-green.log). Neither failure was a compile
error or timeout.

[test-body-comparison.json](test-body-comparison.json) compares exact source
bytes bounded by each Go AST function body. All nine valid task-15 test bodies
and assertions are byte-identical. Only the two explicitly permitted historical
negative bodies differ. The retained full preimage,
`simulation_progress_test.before.txt`, hashes to
`4d91f01eb5e378e5aa4824c2af655862d9fe0fe57772bd74f2dfa648418c0880`,
the final task-15 identity. The design hash remains
`1f2fc94d417ad3ffe2d64a2b255787d3ad74e13701bc85a7294522a0629a60c9`.

The task-15 fixture changes only its model-dispatch and known abandoned-response
discard callback wiring. Its bounded frame I/O, context-Done handshake and
cleanup remain. Existing old-method tests migrate through owner events; their
pending-external setup uses an actual control-forward/delivery/consumption
sequence in test utilities.

Five additional behavioral lifecycle tests cover suspended-wait completion and
both delivery paths, idempotent completion reservation, invalid response steps
leaving quiescence installed, duplicate control forwarding preserving both
participants, and simultaneous numeric ID 1 in the coordinator and model
request namespaces. Task-15 tests retain both same-participant reply orders and
the real `serveSimulationModels` plus World committed-mutation cancellation,
removal and late-discard fixture.

## Verification

All test commands ran serially with stable source, stock module-selected
Go 1.27.1 on linux/arm64, unseeded host environment, `GOWORK=off` and
`-tags test_dep`. [evidence.json](evidence.json) records exact commands and exits.

| Final check | Exit | Evidence |
| --- | --- | --- |
| Focused time/model/coordinator/server tests | 0 | `final-focused.log`, 0.004 seconds |
| Expanded focused race tests | 0 | `final-race.log`, 1.027 seconds |
| Progress and lifecycle tests, 100 repeats | 0 | `final-repeat.log`, 0.075 seconds |
| Execution-package vet | 0 | `final-vet.log` |
| `TestPackageArchitecture` | 0 | `architecture.log`, 0.431 seconds |
| Exact valid-body comparison | 0 | `test-body-comparison.json` |
| Strengthened negative regressions | 0 | `regressions-green.log`, 0.003 seconds |
| gofmt listing and diff checks | 0 | Empty gofmt listing; `git diff --check` clean |

No task-16 shared schema, template, overlay, generated file or source-inventory
input changed. The predecessor task-13/14 generation and validation evidence
remains applicable to those same inputs. The new private host file is covered by
the architecture check. No runtime stand-in or generated-source workaround was
introduced.

The existing scoped linter exited 2 before analysis. Read-only Go `debug/macho`
inspection confirms the exact binary is Mach-O 64-bit `CpuArm64`, SHA-256
`61f380f1d4c0c57b6cc0a4df3b72183f067c860818943a258aa76171456761e8`.
[linter-platform.json](linter-platform.json) records the mismatch with this
linux/arm64 host. `file` is unavailable. The binary was not retried or replaced;
lint acceptance remains unavailable, while scoped vet passes.

## Open acceptance

All five canonical task-16 patched-toolchain Quick commands exit 127 because
`.toolchain/bin/go` is absent. Their separate `native-*-unavailable.log` files
retain that result. Required `TestRootProcessSimulationUsesRunnerTransport`,
root gomad3sim toolchain tests, whole Runner/host gates and both native platform
qualifications remain incomplete. Developmental stock results do not prove
native timer behavior, cross-process timing, hard isolation or exact replay.

The task-15 pre-edit broad stock Simulation baseline remains red with child exit
49 in `TestRunSupervisesSimulationNodeProcess` and
`TestRunHardCrashesAndReapsSimulationNodeProcess`. Its retained evidence is
`../task-15/baseline-unit.log`. The unchanged environment failure was not retried.
No expectation, gate, platform policy or existing model-delay watchdog/D12/D14
disposition was weakened.

The conductor owns independent source review and task completion. The configured
codex implementation-review backend remains deferred while required native and
process acceptance is incomplete. This handover makes no GREEN BASELINE_HANDOFF,
native acceptance, R11 completion or SHIP claim. All pre-existing task-13/14,
Flow and unrelated `.turbo` edits remain preserved.

Tier: session(jev-unavailable(no_key)); explicit pin wins.
