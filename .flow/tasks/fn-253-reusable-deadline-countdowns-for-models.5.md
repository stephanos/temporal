---
satisfies: [R2]
---
# fn-253-reusable-deadline-countdowns-for-models.5 Shared countdowns fixture with refuting mutants

Touches: [model/umpire/dev/fixtures/fixtures/countdowns/**, model/umpire/dev/fixtures/src/**]

## Description
A small, system-agnostic fixture that exercises every countdown function and law through compiled native, interpreter and TLC. Its mutants show both checkers refuting broken countdowns (spec R2 errors).

**Size:** S
**Files:** `model/umpire/dev/fixtures/fixtures/countdowns/{model.rs,tests.rs,expected/}`, fixture registration.
**Touches:** [model/umpire/dev/fixtures/fixtures/countdowns/**, model/umpire/dev/fixtures/src/**]

### Approach
- Follow `optional_ticks` and `phased_rejection` for layout, registration and TLC agreement.
- Two countdowns with a priority between them, an optional World configuration, a reset action and an absent case.
- Source mutants (as in `phased_rejection` and activity `proof_checks.rs` `global_mutants`): `tick` without decrement, decrementing an absent countdown, decrementing below zero, `within` off by one. TLC and native both refute each one, and the counterexamples replay.

### Investigation targets
**Required:**
- `model/umpire/dev/fixtures/fixtures/optional_ticks/` — optional-tick fixture pattern
- `model/umpire/dev/fixtures/fixtures/phased_rejection/` — library-law fixture with mutants
- `model/temporal/feature/activity/proof_checks.rs:514-540` — source-mutant helper usage

### Acceptance
- [ ] The fixture passes native, interpreter and TLC with agreement; goldens reviewed.
- [ ] Each of the four mutants is refuted by TLC and native, with replayed counterexamples.
- [ ] Passes under `UMPIRE_BACKENDS=require`.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
