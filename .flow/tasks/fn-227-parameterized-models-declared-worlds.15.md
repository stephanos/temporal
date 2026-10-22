---
satisfies: [R3]
---
# fn-227-parameterized-models-declared-worlds.15 Compact emitted TLA+ for conditional returns before activity retries

## Description
From fn-227.11's review (`.flow/tmp/fn22711-review.md`). Run after .11 and before .3, because .3's retries add more conditional rejections to the activity.

**M2 (main):** sequential conditional returns emit repeated `IF c THEN x ELSE x` per field; one `Client::Drop` arm emits about 8 KB (`phased_rejection` `Job@tla.snap.tla:29`). Fold branches whose arms are equal and emit shared subexpressions through `LET` bindings so emitted size stays linear in the number of returns. TLC verdicts, state counts and replay must stay identical; the TLA+/cfg text goldens of affected fixtures change and each change is justified. Record emitted sizes before and after for `phased_rejection` and the activity.

**M1:** add one deliberately false postcondition after `Worker::Claim` in `phased_rejection` so a TLC counterexample replays through a rejected payload edge.

**L1:** refuse an inherent method that shadows a trait method used by a library law or `Phased`'s default (front would otherwise read the inherent one silently); L3: give unknown `closable::` laws and an unparsable library file a specific refusal; L4: test a non-literal rejection value; L6: README says qualified `umpire_dsl::reject!`/`path!` are refused. Optional L5 (apply the post-return guard where assignments are recorded) if it shrinks code.

### Verification
Quick default and `UMPIRE_BACKENDS=require`; manifest diff limited to listed TLA+/cfg text and the new fixture claim; Full gates after review.

## Acceptance
- [ ] Emitted TLA+ size is linear in sequential conditional returns; sizes before/after recorded; TLC verdicts, state counts and replay unchanged.
- [ ] A TLC counterexample replays through a rejected payload edge.
- [ ] L1, L3, L4, L6 resolved.
- [ ] Only listed TLA+/cfg goldens change, each justified.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
