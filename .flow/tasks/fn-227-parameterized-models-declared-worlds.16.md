---
satisfies: [R3]
---
# fn-227-parameterized-models-declared-worlds.16 Named intermediates in the IR so admitted trees stay linear in conditional returns

## Description
From fn-227.15's reviews (`.flow/tmp/fn22715-review.md` L1, `fn22715-review2.md` N2). Emitted TLA+ is linear in conditional returns since fn-227.15, but the front end's admitted tree and the macro-embedded `__lifted` data still grow about 7x per return when a later condition or update reads a field changed after a conditional return (and a run of updates after one early return doubles the tree per update). fn-227.15 added a 10,000-node budget so such steps refuse at the statement instead of exhausting rustc memory (measured: about 24k-64k nodes OOM), which caps the pattern at about four sequential returns.

Introduce named intermediates in the IR (`ExprKind::Local` plus per-branch definitions, emitted as TLA+ `LET`) so the tree, the embedded data and compile time stay linear. Then raise or remove the budget. Run before fn-227.3, whose retries add conditional rejections to the activity.

### Verification
Size tests over the admitted tree and embedded data (not only emitted TLA+) for sequential returns and repeated updates; TLC verdicts, state counts and replay unchanged on every fixture; listed TLA+ text goldens may change, justified. Quick then Full gates; record compile time for a pathological probe before/after.

## Acceptance
- [ ] Admitted tree and embedded data grow linearly with sequential conditional returns and repeated updates (size tests).
- [ ] Node budget raised or removed with a probe showing no rustc blow-up.
- [ ] TLC verdicts, state counts and replay unchanged; changed goldens justified.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
