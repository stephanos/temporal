---
satisfies: [R3]
---
# fn-253-reusable-deadline-countdowns-for-models.6 Temporal deadline semantics as a named pattern

Touches: [model/temporal/feature/activity/README.md]

## Description
Store the Temporal deadline semantics (spec Architecture layer 2) once, in the form .2 chose: a shared module the front end reads, or a documented authoring pattern with named effects and laws. Name it, and cite its authority.

**Size:** M
**Files:** task .2 selected the documented pattern at `model/temporal/feature/activity/README.md`, with named consuming effects demonstrated by .7. This reanchors the provisional capabilities/system path to the committed spike verdict. No unused `deadline.rs`, project-library registration or framework edit is added.
**Touches:** [model/temporal/feature/activity/README.md]

### Approach
- Cover the timeout names, arming per phase, expiry priority, retry re-arm, heartbeat reset, fn-246.13's API-aligned pending-cancellation failure/expiry outcomes, and a global countdown that is never replenished.
- Cite the pinned API and docs already used by fn-237 and fn-246.15; no new policy.
- Keep `model/umpire` free of Temporal names: any registration hook is generic (a project library), and the Temporal source stays in `model/temporal`.
- For a documented pattern, store the authoritative pattern in README and demonstrate its named effects in .7. For a shared module, wire the file into the actual Temporal crate/module tree and its declared generic project-library admission path, and prove the consumer both compiles and lifts it. A free-standing uncompiled `.rs` file does not satisfy either form.
- If the protocol Model question (spec Parked unknowns) is still open, write the pattern so the feature Model can use it without the protocol Model depending on it.

### Investigation targets
**Required:**
- `model/design/deadline-countdowns.md` — chosen form
- `model/temporal/feature/activity/model.rs:244-296` — semantics being extracted
- `.flow/tasks/fn-246-realization-as-a-checked-refinement-a.15.md` — corrected retry contract

### Key context
Use fn-246.15's committed retry contract and fn-246.13's approved cancellation correction. Reanchor the pattern against both committed implementations before .7; do not restore the superseded cancel-wins-over-expiry policy.

## Acceptance
- [ ] One named location holds the Temporal deadline semantics, documented with authority links.
- [ ] No Temporal names enter `model/umpire`.
- [ ] Pattern form has a README authority and named consuming effects without a dead `.rs` file; shared-module form has an explicit compiled owner/module declaration, generic registration and consumer compile/lift evidence, with a refusal row for misuse.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
