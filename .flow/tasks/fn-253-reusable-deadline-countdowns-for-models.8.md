---
satisfies: [R5]
---
# fn-253-reusable-deadline-countdowns-for-models.8 Docs and full gates for countdowns

Touches: [model/README.md, model/umpire/README.md, model/DESIGN.md, model/temporal/feature/activity/README.md, MILESTONES.md]

## Description
Document how a Model declares its countdowns and deadlines, then run the full gates. The spec closes on this evidence.

**Size:** S
**Files:** `model/README.md`, `model/umpire/README.md` (bounded optional values section), `model/DESIGN.md`, activity README, `MILESTONES.md`.
**Touches:** [model/README.md, model/umpire/README.md, model/DESIGN.md, model/temporal/feature/activity/README.md, MILESTONES.md]

### Approach
- README: a short authoring example (countdown fields, `arm`/`tick`/`expired`, the laws), next to the closure-law paragraph.
- DESIGN: the two-layer split and the independence rule for refinement.
- After .7/.9/.10 resolve and their changes are integrated, run `make model-check` and `make model-validate` against the final combined source through the current fn-254 gate. Reuse its test inventory and required-TLC evidence; do not precede those gates with a duplicate workspace suite. Follow the active fn-254 validation cadence if its final checkpoint is still pending.
- Confirm that final World schemas, exact parameter contracts and feature/protocol adapters agree after .9's recorded outcome and .10's flag removal. Re-run the affected existing refinement/parameter controls and live cancellation gates against this combined source; preserve independent protocol semantics and record the selected Worlds and actual counts.
- Write the validation-cost review required by spec close.

## Acceptance
- [ ] README and DESIGN describe the abstraction with an example; the activity README matches the migrated Model.
- [ ] `make model-check` and the TLC-required `make model-validate` pass; goldens re-accepted after review.
- [ ] Final combined-source evidence covers .9's adoption or recorded non-adoption, .10's approved compatibility control, exact parameter/adaptor agreement, and affected live cancellation controls.
- [ ] MILESTONES.md updated.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
