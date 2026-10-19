# The path! progression notation for progress claims (model)

## Conversation Evidence

> user (2026-10-19): "is there a delcarative option; something like a list where you can say [state_A, ..., state_B] and the `...` means evntually? somethig that reads like what a human not knowing temporal logic would write? and its reads like the state progression"
> user (2026-10-19), after the notation was checked against every progress claim in the archive: "file spec/task for the path! syntax"; on chains `[a, .., b, .., c]`: "should we either make it what people excpect or forbid it instead?" — forbidden.

## Goal & Context

A progress claim is today written as a sentence, `whenever(p).eventually(q)`. The owner wants the same claim readable as a state progression by a developer who knows no temporal logic: a list with a gap, `path![self.heating_on, .., self.beeping]`, Rust's own slice-pattern `..` standing for "eventually", and `..=n` for "within n steps". A survey of every progress claim in the archived Scala (Kettle, TwoLayeredCache, TwoPhaseCommit, ReliableBroadcast, five Nexus protocol Models, the activity-mini protocol) found each one to be a single leads-to, optionally bounded, under zero to two assumptions, so the notation covers the whole known population with exactly one shape. Chains and adjacency are refused rather than given a meaning.

## Architecture & Data Models

- `path!` is a `macro_rules!` in `umpire-dsl` with exactly two arms: `path![$p:expr, .., $q:expr]` expands to `whenever($p).eventually($q)` and `path![$p:expr, ..= $n:expr, $q:expr]` to `whenever($p).eventually($q).within($n)`; `$n` is any expression (a literal, a `const`, `w.rms.len()`). Any other shape is a compile error from a catch-all arm: a third element, `a progression has one trigger and one target; write two claims`; no gap, `write [p, ..=1, q] for the next step, or [p, .., q]`.
- It produces the same `Progress` value as the builder (fn-214), so `.assuming(..)` chains on it and the checker, verdicts, goldens and fn-212's emitter are untouched. `whenever(..).eventually(..)` stays public as the expansion.
- Kettle's canonical claim becomes `path![self.heating_on, .., self.beeping].assuming(Self::never_switched_off())`; `README.md` states the three reading rules in one place: `..` means on every admitted path (the existential cousin is `find` in a test file), `..=n` is a bound in steps, and the list says nothing about what happens between or after the two states.

## API Contracts

- `path![p, .., q] -> Progress`; `path![p, ..=n, q] -> Progress`.

## Edge Cases & Constraints

- The macro adds no dependency and no proc-macro work; rust-analyzer expands it. R30 budgets hold trivially.
- A `path!` outside a `#[progress]` method is an ordinary `Progress` value (usable in fn-213 tests); only `#[machine]` decides what is a claim.

## Acceptance Criteria

- **R1:** `path![p, .., q]` and `path![p, ..=n, q]` expand to the fn-214 builder forms and accept any expression for `n`. Errors: a third element and a missing gap are compile errors with the messages above, pinned by trybuild `.stderr` fixtures.
- **R2:** Kettle's `boils` is written with `path!` and `.assuming(..)`; all Kettle goldens are byte-identical; the `machine` macro fixture uses it. No error surface.
- **R3:** `README.md` has the three reading rules beside the Kettle example; DESIGN.md's authoring comparison shows the claim in Scala and in `path!`. No error surface.

## Boundaries

- No chains, no adjacency meaning, no monitors (history safety is a separate construct), no "always eventually" or "until": each would be its own pattern with its own verdicts if a sample ever needs it.
- Does not start before fn-214 (it needs `.within(n)` and `.assuming(..)`).

## Decision Context

2026-10-19, owner: the list-with-gap notation is adopted as the canonical spelling of a progress claim because it reads as a state progression to a developer with no temporal logic; the builder remains as its expansion. Chains are forbidden rather than defined, since the two plausible readings (independent legs versus one path) differ subtly and no real claim needs one; adjacency is refused rather than meaning "next step" for the same reason. Named `path!` after the checker's `paths/` goldens, which show exactly such progressions.
