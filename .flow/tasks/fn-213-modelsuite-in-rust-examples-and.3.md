---
satisfies: [R3]
---
# fn-213-modelsuite-in-rust-examples-and.3 umpire-suite explorations: Space, verify_all, refuted, find

## Description
In `umpire-suite`: `Space::from(starts)` with `.budget(n)` (default the checker's), `verify_all(&Space) -> Result<Report, Refusal>`, `refuted(&Space, predicate) -> Result<Counterexample, Refusal>` whose path spells as the checker's `paths/` goldens, `find(&Space, predicate) -> Result<Path, Refusal>` (shortest path to a matching state; none within budget is a Refusal). All answers come from the checker's `explore`/`verify_all`/`refuted`; the suite adds no exploration logic. A cut exploration answers unknown, never verified; a counterexample found before the cut stands. Failure messages pinned by suite fixtures (unreachable find, cut budget).

**Touches:** `model/umpire/suite/**` only (parallel with .2 after .1; the crate skeleton belongs to whichever of .2/.3 starts first, the other rebases).

**Feedback loop (R30):** paste `make model-time` into the done summary.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
