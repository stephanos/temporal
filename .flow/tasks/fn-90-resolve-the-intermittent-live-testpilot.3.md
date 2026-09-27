---
satisfies: [R2]
---
# fn-90-resolve-the-intermittent-live-testpilot.3 Baseline re-measurement of the three failures

## Description
Re-measure the three failures on today's identities (R2), before any fix lands. Measurement only:
no source change. Its receipt is the evidence fn-90.4, .5 and .6 start from, and the "before"
column of R8. This is the spec's early proof point.

**Size:** M (wall-clock heavy, token light)
**Files:** none in the repository; record files live outside it (for example under `$TMPDIR/fn-90/`), and the results go into this task's receipt
**Touches:** []

### Approach
- Start from a commit that contains fn-90.1 and fn-90.2 and no fix from fn-90.4 to .6, with a clean tree under `tests/`, `common/testing/testpilot/` and `tools/umpire/`. Note the commit.
- Run loops one at a time (never two live loops at once on this host), each with its own record file, via `make umpire-repeat-run`:
  1. `^TestTestpilotUmpireRunRunsACheckedInCaseAgainstAnyEndpoint$`, process mode, 50.
  2. `^TestTestpilotNexusPairCase$`, process mode, 200.
  3. `^TestTestpilotNexusCallerAsyncCompletion$`, process mode, 200.
  4. `^TestTestpilotNexusCallerCaseRunsFromItsFixtureNameAlone$`, process mode, 200.
  5. `^TestTestpilotWorkerOutageCaseLeavesAnotherQueueAlone$`, process mode, 200.
  6. `^(TestTestpilotNexusCallerAsyncCompletion|TestTestpilotNexusCallerCaseRunsFromItsFixtureNameAlone)$`, in-process, `COUNT=4`, repeated to at least 50 iterations (record files merged with `summarize`).
- Early stop per R2: once one signature has 5 occurrences for an identity, that loop may stop; report the rate with its interval.
- A loop that cannot finish in one session is resumed with a new record file and summed.
- For every distinct signature, classify it: (1) teardown (name which candidate from the spec's code-path notes it matches), (2) ordering, (3a), (3b) with switch value, or out of scope. For (3a), also compute the bounded step's latency distribution (p50, p99, max) from the Run files Run capture wrote for passing and failing iterations (`RunEvent` elapsed times), for fn-90.6.
- Out-of-scope identities are not in these selections; a new signature on a selected identity is still recorded and classed.
- Write the receipt summary as a table: identity, iterations, failures, 95% interval, signatures with class, commit, load. For each of (1), (2) and (3) state "reproduced" or "not reproduced on the successor" (with the retired identity, successor and count).

### Investigation targets
**Required:**
- The spec's "Current code paths per failure" and API Contracts sections
- `tools/umpire/cmd/umpire-repeat/` (fn-90.1) usage

### Key context
- In-process iterations are not independent trials; use them only to show whether reuse raises the rate.
- A new signature outside (1) to (3) is recorded and not chased (Boundaries).
## Acceptance
- [ ] All six loops meet their R2 counts or the early-stop rule; record files are summarized in the receipt with commit and load.
- [ ] Each of (1), (2) and (3) has an explicit reproduced / not-reproduced verdict, and every reproduced signature has a class.
- [ ] If a signature cannot be classified from its fields, the receipt says which field was missing (feeds back to fn-90.2's helper) instead of guessing a cause.
## Done summary
Baseline re-measurement (R2) at `b9bb1a58adf931e0830bcb5917849ae4f1ea894f` (contains fn-90.1 and fn-90.2, no fix from fn-90.4 to .6). The loops ran in the measurement clone with a clean tree, which the harness recorded as `dirty:false` on every record line. All six loops met their R2 counts with zero failures, so none needed the early stop. None of the three failures reproduced on today's identities.

| # | Identity | Mode | Iterations | Failures | 95% CI (Clopper-Pearson) | Signatures | Load at start -> end (1/5/15 min), other go test procs |
|---|---|---|---|---|---|---|---|
| 1 | TestTestpilotUmpireRunRunsACheckedInCaseAgainstAnyEndpoint | process | 50 | 0 | 0.00%-7.11% | none | 20.51 19.06 17.49 -> 5.12 5.01 6.58, 0 |
| 2 | TestTestpilotNexusPairCase | process | 200 | 0 | 0.00%-1.83% | none | 5.12 5.01 6.58 -> 4.76 5.06 6.22, 0 |
| 3 | TestTestpilotNexusCallerAsyncCompletion | process | 200 | 0 | 0.00%-1.83% | none | 4.76 5.06 6.22 -> 5.38 5.17 5.91, 0 |
| 4 | TestTestpilotNexusCallerCaseRunsFromItsFixtureNameAlone | process | 200 | 0 | 0.00%-1.83% | none | 5.27 5.15 5.90 -> 5.71 5.27 5.80, 0 |
| 5 | TestTestpilotWorkerOutageCaseLeavesAnotherQueueAlone | process | 200 | 0 | 0.00%-1.83% | none | 5.42 5.22 5.78 -> 5.60 11.36 9.14 (25.23 at harness start), 0 |
| 6 | NexusCallerAsyncCompletion + NexusCallerCaseRunsFromItsFixtureNameAlone | in-process, COUNT=4 x 13 processes, merged with `summarize` | 52 each | 0 / 0 | 0.00%-6.85% each | none | 5.31 11.20 9.10 -> 5.15 6.88 8.02 (peak 15.90), 0 |

Wall clock: 03:10Z to 04:20Z on 2026-09-27. Loop 1 took about 41 s per iteration, consistent with the stuck 30 s namespace-delete teardown that R4 removes. The driver log gives the other loops' wall clock, harness build included: loop 2 took 4 min, loop 3 5 min, loop 4 3 min and loop 5 4 min, each for 200 iterations, which is about 1 to 1.5 s per iteration. Loop 6 took 17 min for 13 processes. The Run captures below confirm that every iteration really ran its Runs.

Verdicts:
- (1) Namespace-delete timeout: not reproduced in 50 process-mode iterations of TestTestpilotUmpireRunRunsACheckedInCaseAgainstAnyEndpoint, and no kind-(3) failure appeared in that loop either. R4 is unconditional, so fn-90.4 proceeds.
- (2) Evidence-ordering mismatch: not reproduced on the successor. Retired identity TestTestpilotTypedNexusOperationsCase, successor TestTestpilotNexusPairCase, 0/200. That is 400 Runs, two sequential Runs per iteration on one cluster in its default configuration. The test sets no Nexus implementation switch, so only the default switch value was covered, not hsm and chasm separately.
- (3) Async-Nexus Run not SATISFIED: not reproduced on the successors. Retired TestTestpilotAsyncNexusCase is now TestTestpilotNexusCallerAsyncCompletion: 0/200 in process mode and 0/52 in process with -count=4. Retired TestTestpilotAsyncNexusCaseRunsFromItsFixtureNameAlone is now TestTestpilotNexusCallerCaseRunsFromItsFixtureNameAlone: 0/200 and 0/52. TestTestpilotWorkerOutageCaseLeavesAnotherQueueAlone (unchanged identity): 0/200. The umpire-run test: 0/50. Only TestTestpilotNexusCallerAsyncCompletion runs under both switch values (hsm and chasm, 4 Runs each). The other three identities run under the default configuration only.

Classification: no signature was observed, so nothing needed a class and no class is guessed. All 3118 captured Runs closed RUN_DISPOSITION_COMPLETED with VERDICT_STATUS_SATISFIED. By loop:
- loop 1: 50
- loop 2: 400
- loop 3: 1600
- loop 4: 200
- loop 5: 400
- loop 6: 468

Latency for path (3a), which fn-90.6 uses. Samples are controller-side `RunEvent` elapsed times, all from passing Runs because no failing Run occurred. The sample is the 2718 captured Runs of the async-caller fixture and the worker-outage test; the pair Case has other steps. The 2518 async-caller Runs carry these steps, including the 200 plain peer Runs of the worker-outage test; the 200 outage Runs do not.
- await-scheduled (covers the workflow's start-nexus-operation): p50 263 ms, p99 279 ms, max 303 ms. hsm p99 280, chasm p99 281.
- await-completion-authority: p50 0, p99 1, max 4 ms.
- complete-nexus-operation: p50 1, p99 6, max 22 ms.
- await-close (covers finish-workflow after the completion): p50 6, p99 15, max 24 ms. hsm p99 14, chasm p99 16.
- Whole Run (RUN_CLOSED elapsed): p50 283, p99 317, max 345 ms.

Missing field, fed back to fn-90.2's Run capture: the two 5000 ms bounded steps, `finish-workflow` (workflow entrypoint) and `respond-async` (handler entrypoint), emit no RUN_EVENT_KIND_INSTRUCTION_STARTED or _COMPLETED events. A recorded Run holds controller-activation events only, so the bounded steps' own latency cannot be read from Run capture. The controller-side windows that enclose them stay under 350 ms, far below the 5000 ms bound, at host loads of about 5 to 25. `finish-workflow` falls inside await-close. `respond-async` falls inside await-scheduled plus await-completion-authority, because the controller's completion authority exists only after the handler has responded asynchronously. The Case program records that ordering through its causal structure, not through an explicit dependency field. A bounded-step p99 for fn-90.6 therefore needs Run capture to record workflow and handler instruction events, or a Driver-side timing field.

Caveat for fn-90.4 to .6, a condition and not a cause: the original failures fired in whole-suite gate runs (`^TestTestpilot`, about 30 min). These loops ran one identity per process, with no other go test process on the host. Under that isolation, the rates are below the upper bounds above.

Record files are in `/private/tmp/fn-90/`: loop1..5.jsonl, loop6-01..13.jsonl, the matching `*.runs/` Run captures, `*.out` harness output, and `driver.log` with load at each loop's start and end. The driver is `driver.sh` and the latency extractor is `latency.sh`.

Baseline: green. In the clone, `go test -tags test_dep ./tools/umpire/cmd/umpire-repeat/...` passed and `go vet -tags 'test_dep integration' ./tests` passed. No source change (Touches: []).

## Evidence
- Commits:
- Tests:
- PRs:
