---
satisfies: [R2]
---
# fn-253-reusable-deadline-countdowns-for-models.5 Shared countdowns fixture with refuting mutants

Touches: [model/umpire/dev/fixtures/fixtures/countdowns/**]

## Description
A small, system-agnostic fixture that exercises every countdown function and law through compiled native, interpreter and TLC. Source-only mutants distinguish the interpreted mutant from the compiled good graph and give TLC a named-law counterexample (spec R2 errors).

**Size:** S
**Files:** `model/umpire/dev/fixtures/fixtures/countdowns/{model.rs,tests.rs,expected/}`. Existing `build.rs` discovery finds the new fixture; no `dev/fixtures/src` registration file exists.
**Touches:** [model/umpire/dev/fixtures/fixtures/countdowns/**]

### Approach
- Follow `optional_ticks` and `phased_rejection` for layout and TLC agreement; use the existing `build.rs` automatic discovery. Retain `optional_ticks` as the independent raw optional-value subset control; this new fixture covers the library surface.
- Two countdowns with a priority between them, an optional World configuration, a reset action and an absent case.
- Create source-only mutants by replacing selected fixture `countdown` calls with inline broken bodies in the admitted Rust subset. Cover no decrement, arming/decrementing an absent countdown, zero-boundary corruption, and an off-by-one `within` check that rejects a valid boundary. For the zero-boundary mutant, replace saturated zero with a representable positive remaining value; native unit tests separately check that the real tick never underflows. Keep mutations representable so the intended named law fails rather than merely causing a Rust underflow panic or unsupported syntax. Use the existing interpreted source path to demonstrate the changed transition or claim result against the independently compiled good graph, then require TLC to refute the named law and replay its witness through the interpreted mutant while the compiled good behavior disagrees. Record the exact replacement and expected law for each control. This requires no runtime mutation of the compiled library and no library-source override hook.

### Investigation targets
**Required:**
- `model/umpire/dev/fixtures/fixtures/optional_ticks/` — optional-tick fixture pattern
- `model/umpire/dev/fixtures/fixtures/phased_rejection/` — library-law fixture with mutants
- `model/umpire/dev/fixtures/build.rs` — automatic fixture discovery
- `model/temporal/feature/activity/proof_checks.rs:514-540` — source-mutant helper usage

## Acceptance
- [ ] The fixture passes native, interpreter and TLC with agreement; goldens reviewed.
- [ ] Each inline source-only mutant differs from the compiled good transition/claim behavior, TLC refutes its named law, and its counterexample replays through the interpreted mutant. Controls record exact source replacements without a library override hook.
- [ ] Passes under `UMPIRE_BACKENDS=require`.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
