# fn-90-resolve-the-intermittent-live-testpilot Resolve the intermittent live Testpilot failures

> HTML render lens (local): open `.flow/artifacts/fn-90-resolve-the-intermittent-live-testpilot/spec.html` — regenerable, markdown is the record. <!-- flow-next:artifact-link -->

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
fixture; `TestTestpilotAsyncNexusCaseRunsFromItsFixtureNameAlone` became
`TestTestpilotNexusCallerCaseRunsFromItsFixtureNameAlone`. No receipt after 2026-09-13 records
these failures. `UMPIRE4_ORDER.md` no longer lists them as carried-forward failures; its delivery
queue carries this spec in their place. This spec first re-measures the failures on today's
identities, then resolves each one at its cause.

## Architecture & Data Models
<!-- scope: technical -->

- **Reproduction harness.** A developer command runs a named `^TestTestpilot...` selection N times
  and emits a per-iteration record plus a summary grouped by failure signature. A **failure
  signature** has these fields: test identity, failing assertion text, Run disposition, Verdict
  status, the status and terminal state of each rule that did not satisfy, the Run diagnostics
  (kind and code), and the stderr leak lines. Two modes: one process per iteration (`-count=1`),
  and one process with `-count=N`. The test binary is built once before the first iteration. The
  tests still read things from the working tree at run time: Case fixtures, the Lean helper
  binaries, and the umpire commands some tests build with `go build`. So before every iteration
  the harness fingerprints those inputs and stops the loop, keeping the completed iterations, when
  the fingerprint differs from the one it started with. A loop therefore measures one commit or
  says it could not. The harness reads the
  `go test -json` event stream, not the text output. The harness only observes and never changes
  the suite. It lives outside the gate, behind its own make target that sets up the same
  prerequisites the gate does (the Lean helper binaries and the physical temporary directory).
- **Signature line.** The harness and the live tests meet at one line format (see API Contracts).
  A failing live assertion prints the signature line; the harness reads it from the test's output
  events. A failing test that prints no signature line still gets a signature, built from the test
  identity and its first failing assertion line with every other field empty. This split lets the
  harness and the instrumentation be built at the same time.
- **Run capture.** A signature says why a Run failed but carries no timing. To size a bound
  (path 3a) from passing and failing Runs alike, the affected tests write every closed Run to a
  directory when the harness names one in the environment, the same way the control and canary
  tests already record Runs on request. The harness keeps the captured Run paths in its record.
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
    side effect of the stuck delete, not from the delete result. The failing assertion was never
    recorded. The candidates are: the test logger failing on an error-level log from the stuck
    delete (the server-side versus client-side deadline race); the test's 2-minute process bound
    killing the CLI during teardown, since the Run's own `--timeout` is also 2 minutes and teardown
    adds 30 s on top; or an assertion on the CLI output. Releases run in reverse creation order, so
    the endpoint is deleted before the namespace. That order matters: the delete-namespace workflow
    refuses a namespace that a Nexus endpoint still targets.
  - (2) Two operations are scheduled in one workflow task. Their completions reach history in an
    order set by timing. The retired test asserted the lifted `CorrelatedEvidence` kinds in one
    fixed global order. The pair Model's Scenario is an exact sequence over two instances. The
    successor test already checks evidence per rule and correlates each completion to its
    instance by the scheduled event id it references, so the retired assertion's cause may be gone.
  - (3) The recorder closes INCONCLUSIVE on two paths. (a) Run INCOMPLETE: an execution error, for
    example a declared 5000 ms instruction timeout (`finish-workflow`, `respond-async`) running out
    under load. (b) Run COMPLETED with an unresolved rule: the correlated monitor answers
    INCONCLUSIVE when an obligation is still pending or when not every processed evidence item was
    accepted. The async Case completes the operation from the controller as soon as the handler's
    completion authority exists, which can be before the server records the start. The server then
    takes the completion-before-start path (upstream temporal#6821): it synthesizes the started
    event, dated with the start time the completion carries, and that date can precede evidence the
    recorder has already observed. CHASM does this when it handles the completion; the hsm
    implementation fabricates the missing started event on its own path. Both switch values must
    be confirmed to produce the same history shape before a Model admits it.

## API Contracts
<!-- scope: technical -->

- **Harness invocation.** Inputs: a test selection regex, an iteration count, a mode (`process`
  or `in-process`), and a record file. Output is one line per iteration, `<iteration> PASS|FAIL
  <signature-hash>...` (one hash per failing leaf test in that iteration, none on PASS). Then one
  line per test identity, `<test> <failed>/<ran>`, and one line per distinct signature,
  `<signature-hash> <count>/<ran>`, with the test identity and the first failing assertion. The
  exit status is 0 only when every iteration passed. In `in-process` mode an iteration is one
  repeat of the selection inside the single process, numbered by the order of the tests' start
  events, because repeats share a test name.
- **Record file.** Each finished iteration appends one JSON line: `{iteration, mode, selection,
  commit, fingerprint, outcome, failures:[{test, signature_hash, signature}], runs:[path]}`. A
  summary can be printed from one or more record files, so a loop split across sessions adds up to
  one count; files with different fingerprints are never summed.
- **Run capture.** When `UMPIRE_REPEAT_RUN_DIR` names a directory, each affected test writes every
  Run it closes there as a recorded Run file (the format the replay package already writes), and
  the umpire-run test passes the CLI's existing `--record` flag. Unset, nothing is written.
- **Signature line.** A failing live assertion prints exactly one line `TESTPILOT-SIGNATURE <json>`,
  where `<json>` is the failure signature below on one line. When one test prints several, the
  first one counts.
- **Failure signature (canonical order).** `{test, assertion, run_disposition, verdict_status,
  unresolved_rules:[{rule_id,status,terminal_state_id}], diagnostics:[{kind,code}], leaks:[line]}`.
  Iteration-specific values such as Run ids, namespace suffixes, timestamps, ports, addresses and
  durations are normalized out, and lists are sorted, so repeated occurrences of one cause hash
  equal. The assertion is hashed without its source line number, so an edit that moves a line does
  not split one cause into two signatures; the record keeps the location for reading. Two reserved
  signatures carry only `test` and `assertion`: `unparsed` (a failure with no readable signature
  line and no readable assertion) and `process` (the test process ended without a pass or fail
  event for a test that started: a crash, a kill or a test-binary timeout; the assertion is the
  last output line).
- **Quarantine entry.** Allowed only under R7. It holds exactly: `{test, signature, issue_url,
  max_retries: 1, allowed_outcomes: [INCONCLUSIVE, INCOMPLETE, <named infrastructure error>]}`,
  where `signature` is the canonical failure signature itself, compared field by field with the
  one the live test builds. The live test never recomputes the harness's hash. Nothing else is
  added.

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
  harness must allow a narrow selection. The R2 and R8 loops take hours of wall-clock time, and one
  caller-test iteration starts one cluster per switch value. The record file lets a loop stop and
  resume across sessions without losing counts.
- **Shared host.** Load from other sessions changes timing, and path (3a) depends on timing. Every
  loop records the commit, the host's load average and the number of other `go test` processes
  when it starts. Before and after rates are compared only under similar load.
- **Measured tree.** Loops run from a commit, with no uncommitted edits in what the live tests
  build or read (`tests/`, `common/testing/testpilot/`, `tools/umpire/`, the Case fixtures and the
  Lean helper binaries), because other sessions edit this working tree. The per-iteration
  fingerprint (Architecture) catches an edit that lands during a loop.
- **fn-88 overlap.** fn-88 (in progress in another session) pins checker state counts on the
  Nexus Caller and Pair Models. A fix under R5 or R6 that changes either Model, or the Nexus
  realization, must not land while fn-88 is open unless it is coordinated with that session;
  fn-88 then re-pins its counts.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** A reproduction harness implements the Harness, Record file and Signature line contracts
  in both modes and prints per-identity and per-signature failure rates. Errors: a build failure
  or a selection matching no test exits non-zero and counts zero iterations, never passes (Go
  reports an unmatched `-run` as a passing package, so the harness checks that a selected test
  started); a failure the signature parser cannot read is reported as the `unparsed` signature,
  never merged into a known one; an interrupted loop reports and records the iterations it
  completed and does not count the interrupted one; a changed input fingerprint stops the loop the
  same way and names what changed; a malformed record-file line, or record files with different
  fingerprints, fail the summary and name the file and line.
- **R2:** Each failure is re-measured on its current identities, at a recorded commit that
  precedes any R4 to R6 fix. (1) `TestTestpilotUmpireRunRunsACheckedInCaseAgainstAnyEndpoint`
  runs at least 50 process-mode iterations; failures of kind (3) in this test are counted from
  the same loop under their own signatures. (2) `TestTestpilotNexusPairCase` runs at least 200
  iterations. (3) `TestTestpilotNexusCallerAsyncCompletion`,
  `TestTestpilotNexusCallerCaseRunsFromItsFixtureNameAlone` and
  `TestTestpilotWorkerOutageCaseLeavesAnotherQueueAlone` run at least 200 iterations each, plus at
  least 50 in-process iterations of `-count=4` over the first two. A loop may stop early for an
  identity once one signature has 5 occurrences there; the rate is then reported with its 95%
  interval. The rates, intervals, signatures, commit and host load are recorded in the task
  receipt. Errors: a failure with zero occurrences across the full count closes as "not
  reproduced on the successor"; that record names the retired identity, its successor and the
  iteration count, and it never claims a root cause. A signature outside (1) to (3) is recorded
  and left out of scope (see Boundaries).
- **R3:** The affected tests' disposition, Verdict and rule-status assertions print the Signature
  line (Run disposition, diagnostics, and each unresolved rule's status and terminal state), in
  `TestTestpilotNexusPairCase`, the Nexus caller Query tests, both worker-outage tests and the
  umpire-run test. For the umpire-run test, whose Run lives in a child process, the CLI's report
  also prints the Run's diagnostics, and the test's failure message carries the process exit
  status, whether it was killed, and the elapsed time. One failing run then tells path (3a) from
  path (3b), an ordering mismatch from a Contract violation, and a teardown kill from an output
  mismatch. The same tests implement Run capture (API Contracts). The part that builds a signature
  from a Run and Verdict compiles without the `integration` tag, so an offline unit test pins it and
  an R7 quarantine can reuse it. Errors: no error surface beyond test failure output; with Run
  capture unset nothing is written, and a capture directory that cannot be written fails the test
  naming the path; the CLI's exit codes and stderr leak lines are unchanged.
- **R4:** Resolve (1) at its cause. The umpire-run live test runs against a cluster where the
  deletion that `--create` issues can finish. The preferred way is a test cluster with the system
  worker service. The test then asserts that the endpoint is gone and that describing the
  namespace by its original name answers not-found within a bounded wait, and its run time no
  longer includes a 30 s teardown timeout. This fix is made whether or not R2 reproduces (1),
  because every run of the test today pays the stuck delete. Errors: a namespace or endpoint that
  is still present at the end of the wait fails the test and names the resource. A deletion that
  exhausts the teardown budget still prints its stderr leak line and fails the test. The
  unreachable-endpoint test keeps exit 3.
- **R5:** Resolve (2) at its cause. Every live assertion over the pair Case correlates each
  instance's evidence by its key (the scheduled event id a completion references), never by
  global position. If R2 or R3 shows that the pair Case's Scenario or Contract admits only one
  completion order that the server does not guarantee, the Model or Producer is fixed so the Case
  admits every order the server may record, and the fixture is regenerated. Errors: a missing
  completion still yields INCONCLUSIVE. When R5 changes the pair Model, Producer or Contract, a
  live control proves that a completion referencing the other instance's scheduled event still
  yields VIOLATED, and the existing single-instance forged-completion control keeps passing.
- **R6:** Resolve (3) at the cause R3 names. On path (3a), the Model's declared bound is corrected
  and the fixture regenerated, and the receipt states the latency the new bound covers: the p99 of
  the bounded step over the R2 iterations, taken from the Runs that Run capture wrote (passing and
  failing), with the margin stated. On path (3b), either the async Model admits the server's completion-before-start path
  (the synthesized started event) as the product behavior it is, or the Program orders the
  completion after the start is observed; the Decision Context records which, and why. Errors: a
  Run that is truly incomplete (worker stopped, deadline passed) still closes INCONCLUSIVE or
  INCOMPLETE; a completion violating the Contract still closes VIOLATED; hsm and chasm must still
  agree (a Verdict that differs between the switch values still fails, naming both).
- **R7:** A signature may be quarantined only when R3 evidence shows its cause is outside this
  repository's control, for example a server or SDK race with an upstream issue. The quarantine
  entry must match the API contract, and its retry must live in the live test, not the runtime. The
  retry fires only when the failing attempt's signature equals the entry's signature field by field
  and its outcome is in `allowed_outcomes`, at most once, and every retry is logged with the first
  attempt's signature. The retry decision compiles without the `integration` tag. Errors: any attempt with a VIOLATED Verdict fails immediately with no retry; a
  signature that does not match fails with no retry; an entry with no issue link or with
  `max_retries` above 1 is rejected by an offline unit test that also proves the first two
  errors. With no quarantine entry, no retry code exists.
- **R8:** Definition of done: every signature from (1) to (3) is resolved, closed as not
  reproduced, or quarantined, and these all pass on the final commit. First, the R2 iteration
  counts again, per identity, with zero failures and no early stop (with 200 iterations the 95%
  upper bound on the per-test failure rate is 1.5%; with 50 it is 6%). Second, five consecutive
  `make umpire-check-live-tests` runs exit 0 with no retry logged, unless a quarantine is logged
  and linked. Third, `make umpire-check-regression` once. The receipt carries the per-signature
  rates before and after, compared by signature class, and the host load of each loop. Errors:
  one failure of an identity resets that identity's count only; a gate run that fails on any
  identity in (1) to (3), or on a test this spec changed, restarts the five-run sequence. A gate run
  whose only failures are identities outside that set (the canary harness tests, for example)
  neither counts toward the five nor resets them; its signatures are recorded in the receipt and
  listed as a follow-up in `UMPIRE4_ORDER.md`. A new signature found during the loops is triaged
  under R2 and is never retried away.
- **R9:** On close, the `UMPIRE4_ORDER.md` queue item for this spec is replaced by an entry in the
  delivered list, the gate baselines record the after rates, any quarantine is listed with its
  issue link, and the Testpilot provisioning notes say that a `--create` namespace deletion
  finishes only on a cluster that runs the system worker. No error surface beyond doc review.

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

The fix tasks for (2) and (3) are conditional on the measurement. Each one starts from the
baseline receipt and may close as "not reproduced on the successor" with that evidence, without a
code change beyond confirming its R-ID's test-side clause. The fix for (1) is not conditional: the
stuck delete is certain on every run, whatever the measured failure rate.

The harness reads `go test -json` events and a one-line signature the tests print, rather than
parsing assertion text. Assertion text changes with every message edit, while the signature fields
are the facts that tell the causes apart. It is a Go command beside the other Umpire developer
commands, not a shell loop: it needs the event parsing, the normalization and a unit-tested
signature hash. The repository's CI test runner re-runs every failed test blindly, so it is the
pattern this spec avoids, not one to reuse.

Rejected as overkill: sharing the signature type between the harness and the live tests as Go
code. The one-line JSON format is the whole interface, and a shared package would couple the
integration-tagged tests to the tool's build.

A loop detects a changed tree instead of running from a frozen copy. A `git archive` export would
have no built Lean helper binaries and would need a cold model build per commit, and worktrees are
not used in this repository. The fingerprint costs one hash of a few directories per iteration and
turns a silent mix of versions into a stopped loop.

A red gate run caused only by an out-of-scope test does not count as green, and it does not reset
the five either. Counting it green would hide a failure; resetting on it would make this spec's
done depend on flakes it leaves out of scope, and invite re-running until five green runs line up.
Recording it and listing it as a follow-up keeps it visible.

## Quick commands

```bash
# Offline: the harness and its signature parser
go test -tags test_dep ./tools/umpire/cmd/umpire-repeat/...
# Compile the live tests without a cluster
go vet -tags 'test_dep integration' ./tests
# A narrow live loop
make umpire-repeat-run SELECT='^TestTestpilotNexusPairCase$' COUNT=5 MODE=process RECORD=/path/to/record.jsonl
# The gate
make umpire-check-live-tests
```

## Early proof point

Task fn-90-resolve-the-intermittent-live-testpilot.3 validates the approach: it shows which of the
three failures still occur on today's identities, and with which signatures. If it finds none of
them, the spec reduces to the (1) teardown fix, the not-reproduced records and the final gate; if
its signatures cannot tell the causes apart, revisit the signature fields before any fix.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
|-----|-------------|---------|-------------------|
| R1 | Reproduction harness | fn-90-resolve-the-intermittent-live-testpilot.1 | — |
| R2 | Baseline re-measurement | fn-90-resolve-the-intermittent-live-testpilot.3 | — |
| R3 | Signature-bearing failures | fn-90-resolve-the-intermittent-live-testpilot.2 | — |
| R4 | Resolve (1): deletion finishes | fn-90-resolve-the-intermittent-live-testpilot.4 | — |
| R5 | Resolve (2) or close not reproduced | fn-90-resolve-the-intermittent-live-testpilot.5 | — |
| R6 | Resolve (3) or close not reproduced | fn-90-resolve-the-intermittent-live-testpilot.6 | — |
| R7 | Signature-exact quarantine | fn-90-resolve-the-intermittent-live-testpilot.5, fn-90-resolve-the-intermittent-live-testpilot.6 | Conditional: built only by the fix task whose cause R3 shows is external; otherwise closed as "no quarantine needed" in both |
| R8 | Definition of done loops and gates | fn-90-resolve-the-intermittent-live-testpilot.7 | — |
| R9 | Plan and provisioning docs | fn-90-resolve-the-intermittent-live-testpilot.4, fn-90-resolve-the-intermittent-live-testpilot.7 | — |

