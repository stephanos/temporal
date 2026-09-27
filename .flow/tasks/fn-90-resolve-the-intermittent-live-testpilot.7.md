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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
