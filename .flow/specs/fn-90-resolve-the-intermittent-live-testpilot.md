# fn-90-resolve-the-intermittent-live-testpilot Resolve the intermittent live Testpilot failures

## Goal & Context
<!-- scope: business -->

`make umpire-check-live-tests` (selection `^TestTestpilot`, empty expected-failure set) is the live
half of `make umpire-check-regression`, and every task receipt cites it. Three failures fire at base
commits in about one or two suite runs in ten. Workers re-run the gate until it passes and record
the failure as a "known flake". A gate that is red for no reason trains people to re-run it, and a
re-run can also hide a real regression. The failures, as the fn-87 receipts recorded them
(2026-09-12 and 2026-09-13):

1. **Namespace-delete timeout.** `TestTestpilotUmpireRunRunsACheckedInCaseAgainstAnyEndpoint`
   failed with a namespace-delete `DeadlineExceeded` (fn-87.1, twice in three gate runs).
2. **Evidence-ordering mismatch.** `TestTestpilotTypedNexusOperationsCase` failed with an
   evidence-ordering mismatch described as "history interleaving". It failed in 2 of 4 gate runs
   (fn-87.6) and passed 5/5 when run alone. It also failed alongside (1) in fn-87.1.
3. **Async-Nexus Run not SATISFIED.** An async-Nexus Run ended INCONCLUSIVE, or with Run
   disposition INCOMPLETE, instead of SATISFIED. This was "known flake (c)" in fn-87.7, .8, .9,
   .13, .15 and .16. It hit `TestTestpilotAsyncNexusCase`,
   `TestTestpilotAsyncNexusCaseRunsFromItsFixtureNameAlone`, the plain peer Run in
   `TestTestpilotWorkerOutageCaseLeavesAnotherQueueAlone`, and the umpire-run test. Measured
   rates: `-count=5` over two tests failed 1 of 10 at base and 2 of 10 at HEAD (fn-87.16).
   Repeating in-process with `-count=4` raised the rate (fn-87.7).

Since then fn-86 renamed or retired these identities. The typed Nexus test became
`TestTestpilotNexusPairCase` (the `nexusPairTests-bothComplete` fixture). The async Case became
`TestTestpilotNexusCallerAsyncCompletion`, which runs under the hsm and chasm switch values with
four concurrent Runs each; the umpire-run test drives the same `nexusCallerTests-asyncCompletion`
fixture. No receipt after 2026-09-13 records these failures, yet `UMPIRE4_ORDER.md` still carries
them forward. This spec first re-measures the failures on today's identities, then resolves each
one at its cause.

## Architecture & Data Models
<!-- scope: technical -->

- **Reproduction harness.** A developer command runs a named `^TestTestpilot...` selection N times
  and emits a per-iteration record plus a summary grouped by failure signature. A **failure
  signature** has these fields: test identity, failing assertion text, Run disposition, Verdict
  status, the status and terminal state of each rule that did not satisfy, the Run diagnostics
  (kind and code), and the stderr leak lines. Two modes: one process per iteration (`-count=1`),
  and one process with `-count=N`. The harness only observes and never changes the suite. It
  lives outside the gate.
- **Signature-bearing failures.** Every live assertion on a Run disposition or Verdict in the
  affected tests prints the Run disposition, the Run diagnostics and each rule's status. A single
  failing run is then diagnosable without a rerun. Some pair and caller assertions already print
  diagnostics; this spec makes that uniform for the affected identities.
- **Current code paths per failure.**
  - (1) The umpire-run `--create` binding provisions a namespace and deletes it on exit. The
    operator `DeleteNamespace` handler starts the system delete-namespace workflow and waits for
    its result. The functional cluster these tests use runs no system worker (`testcore` turns it
    off by default, and `WithWorkerService` is required to turn it on). The delete therefore always
    blocks until the binding's 30 s per-release teardown budget ends; every passing run of this test
    takes about 35 s. The test says it does not assert this deletion, so the failure comes from a
    side effect of the stuck delete, not from the delete result.
  - (2) Two operations are scheduled in one workflow task. Their completions reach history in an
    order set by timing. The retired test asserted the lifted `CorrelatedEvidence` kinds in one
    fixed global order. The pair Model's Scenario is an exact sequence over two instances.
  - (3) The recorder closes INCONCLUSIVE on two paths. (a) Run INCOMPLETE: an execution error, for
    example a declared 5000 ms instruction timeout (`finish-workflow`, `respond-async`) running out
    under load. (b) Run COMPLETED with an unresolved rule: the correlated monitor answers
    INCONCLUSIVE when an obligation is still pending or when not every processed evidence item was
    accepted. The async Case completes the operation from the controller as soon as the handler's
    completion authority exists, which can be before the server records the start. The server then
    takes the completion-before-start path: CHASM synthesizes the started event, dated with the
    callback's start time.

## API Contracts
<!-- scope: technical -->

- **Harness invocation.** Inputs: a test selection regex, an iteration count, and a mode
  (`process` or `in-process`). Output is one line per iteration, `<iteration> PASS|FAIL
  <signature-hash>`, then one line per distinct signature: `<signature-hash> <count>/<N>`, the
  test identity, and the first failing assertion. The exit status is 0 only when every iteration
  passed.
- **Failure signature (canonical order).** `{test, assertion, run_disposition, verdict_status,
  unresolved_rules:[{rule_id,status,terminal_state_id}], diagnostics:[{kind,code}], leaks:[line]}`.
  Iteration-specific values such as Run ids, timestamps and ports are excluded, so repeated
  occurrences of one cause hash equal.
- **Quarantine entry.** Allowed only under R7. It holds exactly: `{test, signature-hash,
  issue_url, max_retries: 1, allowed_outcomes: [INCONCLUSIVE, INCOMPLETE, <named infrastructure
  error>]}`. Nothing else is added.

## Edge Cases & Constraints
<!-- scope: technical -->

- **SEM-16.** Runtime code must not add implicit retry. Any retry lives in the live test and never
  inside Testpilot, the Driver or the evaluator.
- **QLF-05.** A proved VIOLATED Verdict stays violated. No retry, re-evaluation or quarantine may
  turn a VIOLATED attempt into a pass, and no fix may weaken the evaluator (pending-to-satisfied,
  dropping an obligation, widening a window) to hide INCONCLUSIVE.
- **Case authority.** Instruction timeouts and Contract deadlines belong to the Case. A budget that
  is too tight is fixed in the Model or Producer that derives it, and the fixture is regenerated.
  It is never overridden at runtime or in the test.
- **The fn-83 CLI contract stays unchanged.** With `--create`, umpire-run deletes both the namespace
  and the endpoint on exit, and prints one stderr line per resource it could not remove.
- **Failure rates.** The measured rates are low, around 10% per affected test per suite run, and the
  in-process mode raises them. A handful of runs therefore cannot settle anything; see R1 and R8.
- **Retired identities.** A failure whose original test was retired may have gone with it (the
  retired test's fixed-order assertion is gone). That outcome is a measured result, not an
  assumption.
- **Live suite cost.** The live suite takes about 30 minutes, and other agents share the host. The
  harness must allow a narrow selection.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** A reproduction harness implements the Harness contract in both modes and prints per-signature
  failure rates. Errors: a build failure or a selection matching no test exits non-zero and counts
  zero iterations, never passes; a failure the signature parser cannot read is reported as its own
  `unparsed` signature, never merged into a known one; an interrupted loop reports the iterations
  it completed.
- **R2:** Each failure is re-measured at the spec's base commit on its current identities. (1)
  `TestTestpilotUmpireRunRunsACheckedInCaseAgainstAnyEndpoint` runs at least 50 process-mode
  iterations. (2) `TestTestpilotNexusPairCase` runs at least 200 iterations. (3)
  `TestTestpilotNexusCallerAsyncCompletion` and `TestTestpilotWorkerOutageCaseLeavesAnotherQueueAlone`
  run at least 200 iterations each, plus at least 50 in-process iterations of `-count=4`. The
  rates and signatures are recorded in the task receipt. Errors: a failure with zero occurrences
  across its iterations closes as "not reproduced on the successor". The record names the retired
  identity, its successor and the iteration count, and it never claims a root cause. A signature
  outside (1) to (3) is recorded and left out of scope (see Boundaries).
- **R3:** The affected tests' disposition and Verdict assertions print a failure signature (Run
  disposition, diagnostics, and each rule's status and terminal state). One failing run then tells
  path (3a) from path (3b), and an ordering mismatch from a Contract violation. No error surface
  beyond test failure output.
- **R4:** Resolve (1) at its cause. The umpire-run live test runs against a cluster where the
  deletion that `--create` issues can finish. The preferred way is a test cluster with the system
  worker service. The test then asserts that both the endpoint and the namespace are gone after
  exit, and its run time no longer includes a 30 s teardown timeout. Errors: a namespace or endpoint
  that is still present fails the test and names the resource. A deletion that exhausts the teardown
  budget still prints its stderr leak line and fails the test. The unreachable-endpoint test keeps
  exit 3.
- **R5:** Resolve (2) at its cause. Every live assertion over the pair Case correlates each
  instance's evidence by its key (the scheduled event id a completion references), never by
  global position. If R2 or R3 shows that the pair Case's Scenario or Contract admits only one
  completion order that the server does not guarantee, the Model or Producer is fixed so the Case
  admits every order the server may record, and the fixture is regenerated. Errors: a completion
  that references the other instance's scheduled event still yields VIOLATED (the existing
  forged-completion control keeps passing); a missing completion still yields INCONCLUSIVE.
- **R6:** Resolve (3) at the cause R3 names. On path (3a), the Model's declared bound is corrected
  and the fixture regenerated, and the receipt states the measured latency the new bound covers.
  On path (3b), either the async Model admits the server's completion-before-start path (the
  synthesized started event) as the product behavior it is, or the Program orders the completion
  after the start is observed; the Decision Context records which, and why. Errors: a Run that is
  truly incomplete (worker stopped, deadline passed) still closes INCONCLUSIVE or INCOMPLETE; a
  completion violating the Contract still closes VIOLATED; hsm and chasm must still agree (a
  Verdict that differs between the switch values still fails, naming both).
- **R7:** A signature may be quarantined only when R3 evidence shows its cause is outside this
  repository's control, for example a server or SDK race with an upstream issue. The quarantine
  entry must match the API contract, and its retry must live in the live test, not the runtime. The
  retry fires only when the failing attempt's signature equals the entry's signature and its outcome
  is in `allowed_outcomes`, at most once, and every retry is logged with the first attempt's
  signature. Errors: any attempt with a VIOLATED Verdict fails immediately with no retry; a
  signature that does not match fails with no retry; an entry with no issue link or with
  `max_retries` above 1 is rejected by an offline unit test that also proves the first two
  errors. With no quarantine entry, no retry code exists.
- **R8:** Definition of done: every signature from (1) to (3) is resolved or quarantined, and these
  all pass on the final commit. First, the R2 iteration counts again, per identity, with zero
  failures (with 200 iterations the 95% upper bound on the per-test failure rate is 1.5%). Second,
  five consecutive `make umpire-check-live-tests` runs exit 0 with no retry logged, unless a
  quarantine is logged and linked. Third, `make umpire-check-regression` once. The receipt carries
  the per-signature rates before and after. Errors: one failure resets its identity's count. A new
  signature found during the loops is triaged under R2 and is never retried away.
- **R9:** `UMPIRE4_ORDER.md` no longer carries the "Intermittent live failures" entry, and any
  quarantine is listed with its issue link. No error surface beyond doc review.

## Boundaries
<!-- scope: business -->

- Out of scope: runtime retry in Testpilot, the Driver or the evaluator (SEM-16), and any change
  that turns a pending rule into a satisfied one.
- Out of scope: changes to server Nexus or namespace-deletion behavior. A server race found here
  gets an upstream issue, not a server patch.
- Out of scope: a suite-wide retry, `-failfast` changes, or a non-empty expected-failure set in
  `umpire-check-live-tests`.
- Out of scope: failures outside (1) to (3), such as the canary harness tests and the
  `context canceled` noise at cluster shutdown. They are only recorded.
- Out of scope: running the harness in CI or on a schedule.
- Out of scope: the "one Contract monitor per entity" rework.

## Decision Context
<!-- scope: both -->

Root cause first, because every workaround is costly. A retry or a looser assertion can mask the
regression the gate exists to catch, and both SEM-16 and QLF-05 forbid the forms that could turn a
violation into a pass. A quarantine is the last resort, and it is scoped to one exact signature so
that it cannot absorb a new failure.

For (1), running the test cluster with the system worker is preferred over two alternatives:
setting `RetainNamespace` from the test, or adding a hidden CLI flag. Both would stop exercising
the `--create` deletion that fn-83 promises, and the flag would change the CLI contract. The worker
service costs one dedicated cluster, which this test already has, and it should remove the fixed
30 s teardown timeout from its run time.

The gate target follows from the rates. At about 15% red suite runs, five green suite runs alone
happen about 45% of the time with no fix at all. The per-identity loops carry the statistical
weight, and the suite runs show the fixes compose. Re-measuring comes first because the
identities changed under fn-86. Resolving a failure that no longer exists would waste the spec,
and so would assuming it is gone.

## Parked unknowns

- (1): the exact failing assertion is not recorded. It could be the testlogger failing on an
  error-level log from the stuck delete (the server-side versus client-side deadline race), the
  process's 2-minute bound, or an assertion on output. The first R3-instrumented failure resolves
  it.
- (2): whether the failure survived the move from `TestTestpilotTypedNexusOperationsCase` to
  `TestTestpilotNexusPairCase`. R2 resolves it.
- (3): whether the cause is path (3a) or (3b), and under which switch value. The first
  R3-instrumented failure resolves it.
