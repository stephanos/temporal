---
satisfies: [R4]
---
# fn-197-correlated-obligations.1 DSL completion: prototype the spelling of a correlated obligation claim and record the decision

## Description
**DSL completion phase (owner, 2026-10-17).** Under `model/design/obligation-probe/` (README, `run.sh`), write a dispatcher fixture with two request ids and a worker that may answer one id forever, and express its promise three ways: the hand-written `Set[Id]` monitor read at ends, `owes(Fact.requested).until(Fact.responded)`, and a quantified `whenever(recorded(Fact.requested(id))): eventually(recorded(Fact.responded(id)))`. Read the mini feature's `.scala.disabled` monitors (`attemptsNeverRewind`, the Nexus result-delivery obligations) and note how each would read in every spelling. Write `model/design/obligation-monitors.md` with the candidates, the readings, the checker shape (owed set in explored-state identity, per-payload progress semantics) and the owner's decision; if the owner picks another spelling, revise fn-197 R1's words in the spec via `spec set-plan`. No source under `model/framework` changes.

**Size:** M

**Files:** `model/design/obligation-monitors.md` (new), `model/design/obligation-probe/*` (new).

### Verification
- The probe's `run.sh` compiles the fixture with the pinned scala-cli and shows the three spellings side by side.
- The design note records a decision dated by the owner.

## Acceptance
- [ ] Probe with README and run.sh under model/design
- [ ] Design note with candidates, mini-feature readings, checker shape and the owner's decision
- [ ] fn-197 R1 revised if the spelling changed

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
