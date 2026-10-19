---
satisfies: [R3]
---
# fn-213-modelsuite-in-rust-examples-and.3 umpire-suite explorations: Space, verify_all, refuted, find

## Description
In `umpire-suite`: `Space::from(starts)` with `.budget(n)` (default the checker's), `verify_all(&Space) -> Result<Report, Refusal>`, `refuted(&Space, predicate) -> Result<Counterexample, Refusal>` whose path spells as the checker's `paths/` goldens, `find(&Space, predicate) -> Result<Path, Refusal>` (shortest path to a matching state; none within budget is a Refusal). All answers come from the checker's `explore`/`verify_all`/`refuted`; the suite adds no exploration logic. A cut exploration answers unknown, never verified; a counterexample found before the cut stands. Failure messages pinned by suite fixtures (unreachable find, cut budget).

**Touches:** `model/umpire/suite/**` only (parallel with .2 after .1; the crate skeleton belongs to whichever of .2/.3 starts first, the other rebases).

**Feedback loop (R30):** paste `make model-time` into the done summary.

**Added 2026-10-19 (fn-215 R4):** `find(&space, path: Path<State>)` replaces the single-predicate form: the shortest admitted path through the waypoints in order, leg by leg (BFS from each reached waypoint state to the next, `..=n` honoured), actions spelled as the checker's `paths/` goldens; an unbridgeable gap is a `Refusal` naming the waypoint index and the last reached state. A one-predicate `find` is `path![start, .., pred]`. Needs fn-215 R1 first.
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
