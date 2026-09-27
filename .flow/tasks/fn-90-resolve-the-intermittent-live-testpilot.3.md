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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
