---
satisfies: [R1]
---
# fn-253-reusable-deadline-countdowns-for-models.2 Spike: activity deadlines against the proposed countdown surface

Touches: [model/design/deadline-countdowns.md, .flow/tmp/fn253-spike/**]

## Description
Rewrite the activity's deadline code against the proposed `countdown` surface in a throwaway worktree. Let admission refusals list the lifter gaps, and record a verdict that fixes the surface for .3–.8. Nothing from the worktree merges.

**Size:** M
**Files:** `model/design/deadline-countdowns.md` (verdict); probe sources, README and `run.sh` under `.flow/tmp/fn253-spike/`.
**Touches:** [model/design/deadline-countdowns.md, .flow/tmp/fn253-spike/**]

### Approach
- Worktree `.worktrees/fn2531-spike` on `work/fn2531-spike`. Stub `umpire_dsl::countdown` with plain Rust bodies so the native Model compiles; the refusals come from the front end.
- Rewrite `deadline`, `decrement`, the arming in `worker`/`initial`, `state_bounds` (split per spec R4), `countdowns_decrement` and `tick_priority` (`model/temporal/feature/activity/model.rs`).
- Test whether a local helper alone suffices for the activity: an unmarked `fn arm(ticks: Option<Ticks>) -> Option<Remaining>` (and a `tick` counterpart) called from `initial`, `Poll`, retry, `Heartbeat` and `decrement`. The `armed(&self) -> bool` helper is already admitted (2026-10-25). If value-returning local helpers are admitted, record that the library is only needed for sharing across Models, and size .4 accordingly.
- Try both representations from the spec's Parked unknowns: library-owned countdown types, and a trait implemented by the Model's `Finite` wrappers (`Ticks`, `Remaining`). Record which one the front end can type without new nominal-type machinery.
- Try the Temporal-layer form: a shared helper module in `model/temporal` versus named per-Model effects (`time_out`, `time_out_attempt`). Record the front-end change a shared module would need.
- Answer the spec's three Parked unknowns, or name them as owner questions.
- Verdict format per `model/design/` convention: status line, answer in brief, refusals with spans and the owning front-end file, line counts before/after, what was not proved, recommendation, rerun section.
- Remove the worktree at handoff; keep the probe in `.flow/tmp/fn253-spike/`.

### Investigation targets
**Required:**
- `model/umpire/front/src/library.rs` — how `closable.rs` is read as source
- `model/umpire/front/src/body/claim.rs:165` — library calls admitted only as claim statements
- `model/umpire/front/src/body/optional.rs` — the Option subset (presence, equality, eager `unwrap_or`)
- `model/umpire/author/dsl/src/closable.rs` — law style to mirror
- `model/temporal/feature/activity/model.rs:244-340` — current deadline code

**Optional:**
- `model/umpire/dev/fixtures/fixtures/optional_ticks/` — smallest Model with optional ticks
- `model/design/` — existing verdict write-ups

### Key context
fn-246.15 is rewriting the activity's timeout-retry cells. Spike from the current `umpire` tip and note anchors it will move; this task does not wait for it.

### Acceptance
- [ ] `model/design/deadline-countdowns.md` chooses the representation and the Temporal-layer form, with reasons.
- [ ] Every admission refusal from the rewrite is listed with its source span and owning front-end file.
- [ ] The verdict states the final `countdown` function and law names and signatures for .3.
- [ ] Probe kept in `.flow/tmp/fn253-spike/` with README and `run.sh`; worktree removed.
## Acceptance
- [ ] `model/design/deadline-countdowns.md` chooses the representation and the Temporal-layer form, with reasons.
- [ ] Every admission refusal from the rewrite is listed with its source span and owning front-end file.
- [ ] The verdict states the final `countdown` function and law names and signatures for .3.
- [ ] Probe kept in `.flow/tmp/fn253-spike/` with README and `run.sh`; worktree removed.

### Acceptance
- [ ] TBD
- [ ] The verdict states whether value-returning local helpers are admitted today, with the refusal if not.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
