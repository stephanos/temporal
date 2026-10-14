---
satisfies: [R8, R9]
---
# fn-90-resolve-the-intermittent-live-testpilot.7 Definition-of-done loops, gates and plan retirement

## Description
Prove the definition of done (R8) on the final commit and retire the plan entry (R9).
Measurement plus a docs edit; no test or source change unless a loop finds a new signature.

**Size:** M (wall-clock heavy)
**Files:** `.plans/UMPIRE4_ORDER.md` (queue item 2 to the delivered list, gate baselines, harness pointer in environment notes)
**Touches:** [.plans/UMPIRE4_ORDER.md]

### Approach
- On the final commit with a clean tree for the measured packages, rerun the six fn-90.3 loops at their full R2 counts, no early stop, one at a time. A failure resets only that identity's loop; triage a new signature under R2 (never retry it away).
- Then five consecutive `make umpire-check-live-tests` runs, per R8: a run failing on a (1) to (3) identity or on a test this spec changed restarts the five; a run failing only on out-of-scope identities (for example `TestTestpilotCanaryHarness*`) neither counts nor resets, and its signatures go into the receipt and an ORDER follow-up. No retry may be logged unless a quarantine from fn-90.5/.6 exists, in which case each logged retry is quoted with its issue link.
- Then `make umpire-check-regression` once.
- Receipt: per-signature-class before (fn-90.3) and after rates with intervals, host load per loop, the five gate outputs' last lines.
- `.plans/UMPIRE4_ORDER.md`: move the fn-90 queue item (lines 30-36 today) into the delivered list (lines 9-11) and renumber the queue; record the after rates and the umpire-run test's shorter run time under the gate baselines (lines 58-75); add a one-paragraph harness pointer to the environment notes (lines 85-97): `make umpire-repeat-run`, its modes, and that it is not a gate; list any quarantine with its issue link and any out-of-scope gate failures as a follow-up. Re-read the file first: other sessions edit it (fn-88 .9).

### Investigation targets
**Required:**
- fn-90.3, .4, .5, .6 receipts
- `.plans/UMPIRE4_ORDER.md`

### Key context
- `.plans/index.json` changes only through `go run ./tools/planindex` if the ORDER edit requires it; edit only fn-90's entry.
## Acceptance
- [ ] Six loops at full count, zero failures, on one recorded commit.
- [ ] Five consecutive green `make umpire-check-live-tests` runs and one green `make umpire-check-regression`.
- [ ] `UMPIRE4_ORDER.md` updated as above; `go run ./tools/planindex` passes.
## Done summary
Definition of done (R8) passed on commit 7d0997b990, which carries the fn-90.8 pair Case fix and Lean 4.32.0. The measurement clone had a clean tree and a cold model build. All six loops ran at full count with zero failures and no early stop. Five consecutive `make umpire-check-live-tests` runs and one `make umpire-check-regression` exited 0. `.plans/UMPIRE4_ORDER.md` retires fn-90 (R9).

Before is fn-90.3 at b9bb1a58ad; after is this run. 95% intervals are Clopper-Pearson.

| # | Identity | Mode | Before | After | Load at start -> end (1/5/15 min) |
|---|---|---|---|---|---|
| 1 | TestTestpilotUmpireRunRunsACheckedInCaseAgainstAnyEndpoint | process | 0/50 (0-7.11%) | 0/50 (0-7.11%) | 3.45 4.42 4.98 -> 8.30 9.11 7.05 |
| 2 | TestTestpilotNexusPairCase | process | 0/200 (0-1.83%) | 0/200 (0-1.83%) | 8.30 9.11 7.05 -> 4.98 7.54 6.70 |
| 3 | TestTestpilotNexusCallerAsyncCompletion | process | 0/200 (0-1.83%) | 0/200 (0-1.83%) | 4.98 7.54 6.70 -> 3.70 5.69 6.11 |
| 4 | TestTestpilotNexusCallerCaseRunsFromItsFixtureNameAlone | process | 0/200 (0-1.83%) | 0/200 (0-1.83%) | 3.70 5.69 6.11 -> 3.45 4.81 5.68 |
| 5 | TestTestpilotWorkerOutageCaseLeavesAnotherQueueAlone | process | 0/200 (0-1.83%) | 0/200 (0-1.83%) | 3.45 4.81 5.68 -> 4.64 4.71 5.48 |
| 6 | the two caller tests, COUNT=4 x 13 processes | in-process | 0/52 each (0-6.85%) | 0/52 each (0-6.85%) | 4.64 4.71 5.48 -> 3.61 4.01 4.80 |

Other `go test` processes on the host: 0, except 1 during loop 5's end and parts 01 to 02 of loop 6. Captured Runs: 3118 (50, 400, 1600, 200, 400, 468).

By signature class:
- (1), namespace-delete timeout: 0 before, 0 after. The umpire-run test takes 4.2 to 4.4 s in the gate, down from about 35 s before fn-90.4.
- (2), evidence ordering: 0 before, 0 after.
- (3), async-Nexus not SATISFIED: 0 before, 0 after.
- New pair Case signature b467973f771b (Run INCOMPLETE, monitor `observe_failed`: "unauthorized operation transition"): found once in 200 iterations at be465e5966 during this task's first attempt, records in /private/tmp/fn-90-after/. fn-90.8 fixed it in 860ecde37d, and it is 0/200 after.

Gates, all at 7d0997b990:
- The five `make umpire-check-live-tests` runs took 346 to 366 s each at load 1.2 to 3.0. Each ended with "Live Testpilot failure identities match the empty expected set across 45 passing identities." None logged a retry, and no quarantine exists.
- `make umpire-check-regression` took 827 s, exit 0, with the same final line for its live half.

ORDER edit:
- fn-90 moved to the delivered list and the queue was renumbered. fn-89's text now says fn-90 is delivered.
- The gate baselines hold the before and after table and the gate results.
- The environment notes carry the `make umpire-repeat-run` pointer: its modes, that it is not a gate, and that a `--create` namespace deletion finishes only with the system worker. The CC note is qualified.
- Five follow-ups were added to the carried-forward list:
  - the worker ignores Finish and NexusHandlerReply timeouts;
  - Run capture records no worker instruction events;
  - agent shells miss mise's CC fix;
  - the server writes local time labelled as UTC on synthesized started events;
  - a multi-instance Case with a retryable handler error would run `pending-attempts` before the moved scheduled read.
- `go run ./tools/planindex` is valid, so no index change was needed.

Records are in /private/tmp/fn-90-after2/: loop*.jsonl, *.out, *.runs, gate1..5.log, regression.log, and driver.log with load per loop and gate.

GATE_SKIPPED:unittest:docs-only - task commits touch only .plans/UMPIRE4_ORDER.md and the fn-90 review ledger; the live gates and regression above ran in the clone at the measured commit

stage: impl-review - ran [claude:opus:high; SHIP with two P3 wording findings, fixed in a2b91050eb, re-review SHIP]
## Evidence
- Commits: f1269ef211b8028e63ac5634b1c846920d576d07, a2b91050ebfc67ba180f6c7725fcbd1a398c53d6
- Tests: measured commit 7d0997b9903242466ed76a6c196d8d8f8432c0f3 (clone umpire-fn90-measure, clean tree, cold model build on Lean 4.32.0), baseline: green (go test -tags test_dep ./tools/umpire/cmd/umpire-repeat/... ; go vet -tags 'test_dep integration' ./tests, CC=/usr/bin/clang), make umpire-repeat-run SELECT='^TestTestpilotUmpireRunRunsACheckedInCaseAgainstAnyEndpoint$' COUNT=50 MODE=process -> 0/50, make umpire-repeat-run SELECT='^TestTestpilotNexusPairCase$' COUNT=200 MODE=process -> 0/200, make umpire-repeat-run SELECT='^TestTestpilotNexusCallerAsyncCompletion$' COUNT=200 MODE=process -> 0/200, make umpire-repeat-run SELECT='^TestTestpilotNexusCallerCaseRunsFromItsFixtureNameAlone$' COUNT=200 MODE=process -> 0/200, make umpire-repeat-run SELECT='^TestTestpilotWorkerOutageCaseLeavesAnotherQueueAlone$' COUNT=200 MODE=process -> 0/200, make umpire-repeat-run SELECT='^(TestTestpilotNexusCallerAsyncCompletion|TestTestpilotNexusCallerCaseRunsFromItsFixtureNameAlone)$' COUNT=4 MODE=in-process x13 + umpire-repeat summarize -> 0/52 each, make umpire-check-live-tests x5 consecutive -> exit 0 each, 45 passing identities, empty failure set, no retry, make umpire-check-regression -> exit 0, 45 live identities, go run ./tools/planindex -> valid, earlier attempt at be465e5966: loop2 TestTestpilotNexusPairCase 1/200 (signature b467973f771b), fixed by fn-90.8 in 860ecde37d
- PRs: