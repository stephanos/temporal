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
- Worktree `.worktrees/fn2532-spike` on `work/fn2532-spike`. Stub `umpire_dsl::countdown` with plain Rust bodies so the native Model compiles; the refusals come from the front end.
- Rewrite `deadline`, `decrement`, the arming in `worker`/`initial`, `state_bounds` (split per spec R4), `countdowns_decrement` and `tick_priority` (`model/temporal/feature/activity/model.rs`).
- Test whether a local helper alone suffices for the activity: an unmarked `fn arm(ticks: Option<Ticks>) -> Option<Remaining>` (and a `tick` counterpart) called from `initial`, `Poll`, retry, `Heartbeat` and `decrement`. The `armed(&self) -> bool` helper is already admitted (2026-10-25). If value-returning local helpers are admitted, record that the library is only needed for sharing across Models, and size .4 accordingly.
- Shared `Ticks`/`Remaining` (owner question 2026-10-25): try library-owned types with the range as a const parameter (e.g. `Ticks<3>`/`Remaining<3>`), so one bound ties the configured range (`1..=N`) to the live range (`0..=N`). Today the activity states them separately as `#[inputs(1..=3)]` and `#[inputs(0..=3)]`, and nothing checks that they agree. The front end currently refuses generic input types and custom `Finite` impls; record what admitting a const-generic wrapper costs.
- Try both representations from the spec's Parked unknowns: library-owned countdown types, and a trait implemented by the Model's `Finite` wrappers (`Ticks`, `Remaining`). Record which one the front end can type without new nominal-type machinery.
- Try the Temporal-layer form: a shared helper module in `model/temporal` versus a README pattern with named per-Model effects (`time_out`, `time_out_attempt`). For a shared module, name its actual compiled Temporal crate owner, module declaration and generic project-library registration path, plus the admission change needed. For the pattern, name the README authority and consuming effects; do not propose an uncompiled helper `.rs` file.
- Answer the spec's three Parked unknowns, or name them as owner questions.
- Verdict format per `model/design/` convention: status line, answer in brief, refusals with spans and the owning front-end file, line counts before/after, what was not proved, recommendation, rerun section. The committed verdict records the source revision, exact probe edits or reconstruction instructions, commands and required tool identities, so the result remains reproducible after scratch expires.
- Remove the worktree at handoff; keep the temporary probe in `.flow/tmp/fn253-spike/` for current-session reruns. That scratch may expire and is not the durable acceptance evidence.

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
fn-246.15 delivered the timeout-retry correction and fn-246.13 owns the approved cancellation correction. Spike from the current committed `umpire` tip, identify which corrections it includes, and note anchors that later closures may move; this read/probe lane does not wait for those closures.

## Acceptance
- [ ] `model/design/deadline-countdowns.md` chooses the representation and the Temporal-layer form, with reasons.
- [ ] Every admission refusal from the rewrite is listed with its source span and owning front-end file.
- [ ] The verdict states the final `countdown` function and law names and signatures for .3.
- [ ] Probe available in `.flow/tmp/fn253-spike/` at handoff with README and `run.sh`; worktree removed. The committed verdict contains sufficient reconstruction/rerun instructions to survive scratch expiry.
- [ ] The verdict states whether value-returning local helpers are admitted today, with the refusal if not.
- [ ] The verdict decides whether `Ticks`/`Remaining` become shared `umpire-dsl` types with a Model-selected bound.

## Done summary
Completed the measured countdown spike at baseline02d4f55660. Chose shared library-owned Ticks<N>/Remaining<N> with one Model-selected bound1..255 and a README Temporal authoring pattern with named per-Model effects. Fixed six function/law signatures for .3; no production Model, framework, golden or schema change.

Measured21 admission expectations (4 admitted controls,16 Model refusals,1 parse control), precise refusal positions/messages/owners, four full native Model typechecks, compiled project-helper-body control, generic/trait finite-domain and real claim-frame law controls, and1+2 standalone tests. Reconstructed from committed assets and Cargo.lock in a fresh disposable checkout with identical observations. Removed both disposable checkouts; canonical scratch plus durable sources/archives remain.

stage: impl-review - ran [02d4f55660..3eef3b3b75], configured claude-fable-5-1/high, cross-family from GPT writer; correctness/contracts/integration allSHIP. Applied two nonblockingP3 corrections and verified regeneration leaves all25 generated source hashes unchanged; rechecked21 expectations. No blocking or unaddressed R1 findings.

Not proved: interpreted/TLC equivalence or admission of rewritten Models. .3 native library, .4 front end and later fixture/migration/full-gate obligations remain. Protocol sharing of Temporal semantics stays the explicit owner decision permitted by this task; it does not block R1. Broader milestone work and fn254.12 remain paused; migration deferred. No subsequent task starts.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 3eef3b3b75a03c500afa51c5cd65a491a6bcdaa3, daba61c4313b718987822c4ef8da4ccda8fd5669, aa0faf536664c81ee2d5df239a686f2a758381e0
- Tests: In BASE disposable worktree: env -u BASH_ENV CARGO_TARGET_DIR=<task-owned-target> CARGO_BUILD_JOBS=1 CARGO_INCREMENTAL=0 CARGO_PROFILE_DEV_DEBUG=0 bash .flow/tmp/fn253-spike/run.sh (exit0;21 expectations,4+1 native typechecks,finite/law controls,1+2 tests), Repeat documented reconstruction from durable probes and Cargo.lock in a fresh BASE worktree (exit0;observations identical), python3 .flow/tmp/fn253-spike/check-results.py .flow/artifacts/fn-253-reusable-deadline-countdowns-for-models/task2/admission.jsonl (21 verified), Post-review prepare.py regeneration: all25 generated source SHA256 unchanged; shell/Python syntax checks pass, Artifact SHA256 audit; no disposable worktree; no diff to production model/umpire or model/temporal
- PRs: