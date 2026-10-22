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
Emitted TLA+ grows linearly with sequential conditional returns: outcomes are one chain of the rejections a step reaches, updates after a return are guarded where recorded (`running /\ cond` for late conditions), equal arms fold only when the condition has no checked arithmetic or indexing, and subexpressions over 400 characters repeated within a step are emitted once as a `LET` with allocated `__umpire_s` names. Sizes: `phased_rejection` Job.tla 18,040 -> 12,799 bytes (its Drop arm 7,024 -> 1,593); probe 1-6 returns 2,356 -> 706,669,127 bytes before, 1,582 -> 5,789 after. TLC verdicts, state counts and replay unchanged; new `shared_returns` fixture runs shared LETs through TLC for unit and payload actions (18 states agree, counterexample replays through shared edges) and pins the late-condition guard; `phased_rejection` gains a payload counterexample. L1/L3/L4/L5/L6 resolved. A front-end node budget of 10,000 refuses steps whose admitted tree would exhaust rustc memory (measured about 24k-64k nodes OOM); the admitted tree still grows per return, filed as fn-227.16 (named IR intermediates) before .3.

Reviews: NEEDS_WORK (LET path untested in TLC; unguarded late conditions), then NEEDS_WORK (budget too high), fixed by the conductor in 3cb0412d8f (.flow/tmp/fn22715-review.md, fn22715-review2.md). Gates: Full at bf7936c4b0 (`UMPIRE_BACKENDS=require make model-check` 318s, model-time 52.955s/7.827s, slice 7s, gaps 366s); at c3dc7d72e0 model-check 273s, model-time 46.298s/6.618s; at 3cb0412d8f Quick default/required, model-check 2,868s (cold, shared machine), model-time 45.288s gate / 6.080s inner loop. Guest unchanged throughout.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: bf7936c4b0, c3dc7d72e0, 3cb0412d8f
- Tests: UMPIRE_BACKENDS=require make model-check, make model-time, make umpire-check-model-slice, make umpire-check-model-gaps
- PRs: