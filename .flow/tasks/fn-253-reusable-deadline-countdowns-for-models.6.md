---
satisfies: [R3]
---
# fn-253-reusable-deadline-countdowns-for-models.6 Temporal deadline semantics as a named pattern

Touches: [model/temporal/capabilities/system/**, model/temporal/realization/src/lib.rs, model/umpire/front/src/library.rs]

## Description
Store the Temporal deadline semantics (spec Architecture layer 2) once, in the form .2 chose: a shared module the front end reads, or a documented authoring pattern with named effects and laws. Name it, and cite its authority.

**Size:** M
**Files:** `model/temporal/capabilities/system/deadline.rs` and README (new directory, fn-163 split); front-end registration only if the verdict chose a shared module.
**Touches:** [model/temporal/capabilities/system/**, model/temporal/realization/src/lib.rs, model/umpire/front/src/library.rs]

### Approach
- Cover the timeout names, arming per phase, expiry priority, retry re-arm, heartbeat reset, cancel winning over local expiry, and a global countdown that is never replenished.
- Cite the pinned API and docs already used by fn-237 and fn-246.15; no new policy.
- Keep `model/umpire` free of Temporal names: any registration hook is generic (a project library), and the Temporal source stays in `model/temporal`.
- If the protocol Model question (spec Parked unknowns) is still open, write the pattern so the feature Model can use it without the protocol Model depending on it.

### Investigation targets
**Required:**
- `model/design/deadline-countdowns.md` — chosen form
- `model/temporal/feature/activity/model.rs:244-296` — semantics being extracted
- `.flow/tasks/fn-246-realization-as-a-checked-refinement-a.15.md` — corrected retry contract

### Key context
If the corrected retry semantics are not yet committed, write the pattern against the committed fn-246.15 contract text, and reanchor in .7.

### Acceptance
- [ ] One named location holds the Temporal deadline semantics, documented with authority links.
- [ ] No Temporal names enter `model/umpire`.
- [ ] If a shared module: it is admitted and lowered like `countdown`, with a refusal row for misuse.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
