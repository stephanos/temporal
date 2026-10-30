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
Added one named Temporal deadline pattern section to the feature-owned activity README selected by task2. It documents phase arming, Tick priority/single expiry fact, heartbeat reset, retry/global-budget preservation, terminal clearing, cooperative pending Failed/TimedOut outcomes and atomic compatibility. Effective checker Worlds/None remain distinct from raw defaults; protocol semantics stay independently authored. Actual helper/law owners are named; time_out/time_out_attempt/countdown consumer adoption remains task7 and mode removal remains task10. No dead module, project-library registration, framework or Model changes.

Verified8 local targets/fragments,16 helper/claim owners and exact README/Model source hashes; surrounding protocol provenance is unchanged. Pinned API cached files agree byte-for-byte with their archive, origin commit and go.sum; external web fetch returned cache miss and is not claimed successful. Source diff formatting passes.

stage: impl-review - ran (model: claude-fable-5-1 at high); one contracts lens for this single-area documentation-only scope returned SHIP. Stale preparation-handoff Touches sentence corrected to actual root reanchoring; optional direct-public-doc link and preexisting old citation FYIs remain in raw review.
stage: completion-review - skipped(policy: unfinished spec tasks).
Gates: conservative classifier reported FULL for nested Model README; owner cadence defers unrelated broad Model gates to fn254.8. Focused documentation/source-link/authority checks passed; no full gate or code-test execution claimed for this doc-only change. Protocol parent was reviewed, closed and integrated before branch refresh/adoption.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 138e293c61, 5457142ad0
- Tests: README exact source hash and unchanged Model hash checked after integrated protocol refresh, Eight local targets/anchors resolve, sixteen current helper/invariant/postcondition owners verified against Model, Pinned API cached files and archive bytes/hash/origin agree with go.mod/go.sum; external fetch not claimed, git diff --check for documentation and corrected handoff passes, flowctl gate classify --base00ac2429ea:FULL(unmatched nested README); unrelated full gates deferred to fn254.8 by owner policy
- PRs: