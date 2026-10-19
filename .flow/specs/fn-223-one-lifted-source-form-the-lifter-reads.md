# One lifted source form: the lifter reads attribute Models only (model)

## Conversation Evidence

> user (2026-10-19), on learning that about 2,000 of `umpire-lift`'s 5,000 source lines lower the hand-written `Machine` trait form and the rest the `#[machine]` attribute form: "wait why do we need two ??"
> assistant: no real Model needs both; only eight shared test fixtures are written in the trait form, and shared fixtures must agree across the checker and TLC. Recommended rewriting expressible fixtures in the attribute form, making the rest checker-only, and deleting the trait lowering.
> user (2026-10-19): "yes file spec"

## Goal & Context

`umpire-lift` has two front ends: the attribute subset (`#[machine] mod`, the preferred author surface, used by both Kettles' attribute form and by the Temporal activity Model) and the manual subset (`impl Machine for World` with `starts`/`step`, under `lift/src/manual*`, about 2,000 lines). They share only the typed tree and the TLA+ emitter. The manual route exists because shared fixtures in `umpire-tests` are required to agree across every backend (fn-210 R29), and eight of them are trait-form Models: `sample_kettle_trait`, `choices`, `edge_checks`, `vacuity`, `shared_starts`, `progress_graph`, `deep_chain` and `interleaved_claims` (the last two are already refused by location). No product Model uses the trait form.

The `Machine` trait stays: it is the engine interface every `#[machine]` expands to and the plain-Rust floor. Only lifting trait-form source to TLA+ goes. The lifter then has one source form to admit, and fn-211's Snowcat annotations, later admission work (the activity Model) and every future construct pay for one front end, not two.

## Architecture & Data Models

- Each trait-form shared fixture gets one disposition, recorded in a table in `model/DESIGN.md`'s TLC section:
  - **Attribute:** rewritten as a `#[machine] mod` Model with the same states, actions, outcomes, facts, starts and claims. Behaviour goldens (`report`, `rows`, `states`, `paths/`, `tlc`) stay byte-identical except for the author-visible spelling substitutions the attribute surface already implies (state type `Model { .. }`, generated action names), which the task lists explicitly. TLA+/cfg goldens are regenerated.
  - **Checker-only:** moved to `umpire/checker/tests/fixtures/<name>/` (checker fixtures carry no TLC goldens), when the attribute surface cannot express what the fixture pins (for example nondeterministic choice with facts, multiple computed starts, or supplemental manual registrations). The fixture keeps its checker goldens unchanged.
  - **Removed:** only when another fixture already pins the same behaviour on both backends; the table names that fixture.
- `sample_kettle_trait` stays as the independent engine reference (owner, fn-210.7), checker-only, its report byte-identical to attribute Kettle's apart from the documented spellings.
- After the moves, `lift/src/manual.rs` and `lift/src/manual/` are deleted. Source that implements `Machine` by hand and is selected for lifting gets one located refusal: `the lifter reads #[machine] Models; this Model implements Machine by hand` (exact wording decided in the task, pinned by a refusal golden).

## API Contracts

- No author-facing API change. `umpire-lift`'s public entry points keep their signatures; the manual subset's admission rules disappear from `model/README.md`.
- Fixture runner selection is unchanged: expected files still select consumers.

## Edge Cases & Constraints

- TLC coverage must not shrink silently: every behaviour a trait fixture pinned on TLC is either still pinned on TLC by an attribute fixture or listed as checker-only with the reason (the surface gap) in the DESIGN.md table. A surface gap is information for a later spec, not something this spec fills.
- Real TLC execution is selected by `UMPIRE_BACKENDS=require`; the required run must pass for every admitted fixture after the change, and its result goes in the done summary. The default gate keeps its R30 budget.
- `model/temporal` is untouched unless one of its fixtures is trait-form (none is today).
- The no-Temporal-in-umpire gate and the dependency policy apply unchanged.

## Acceptance Criteria

- **R1:** Every trait-form shared fixture has a recorded disposition (attribute, checker-only or removed, with reason) in `model/DESIGN.md`; attribute rewrites keep behaviour goldens byte-identical up to the listed spellings, checker-only moves keep their checker goldens byte-identical, and `make model-check` plus the required TLC run (`UMPIRE_BACKENDS=require`) pass. Errors: a fixture whose rewrite changes any verdict, count or path is not rewritten; it moves checker-only with the difference recorded.
- **R2:** `umpire-lift` no longer contains the manual lowering (`lift/src/manual*` deleted, the dispatch removed); a hand-written `Machine` selected for lifting is refused by location with one message pinned by a golden; `model/README.md` and `model/DESIGN.md` describe one admitted source form and record the lifter's new size against the old. Errors: the refusal names the file and the `impl Machine` item.

## Boundaries

- Does not extend the attribute surface to cover a gap found during the survey; gaps are recorded for the owner.
- Does not remove or change the `Machine` trait, `Steps`, `Claim`, `StepClaim` or the checker.
- Does not touch the six fixtures already refused for reasons other than the trait form, beyond what their move requires.

## Decision Context

2026-10-19, owner: approved filing after the root agent's recommendation to keep one lifted source form. Placement (root agent's call under the owner's standing instruction to follow its own recommendations): right after fn-215, before fn-213.2, so that the suite fixtures of fn-213 and fn-218 and fn-211's emitter work all build on the single front end. The fn-212 design note had chosen one `syn` route reading both forms; in practice the forms needed separate lowering, which this spec removes.
