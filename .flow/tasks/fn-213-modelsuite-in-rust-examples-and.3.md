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
Delivered fn-213 R3 with fn-215 R4 for `find` in `umpire-suite` (commit 00f0ce3bc8).

- `Space::from([..])` / `Space::from(vec![..])` with `.budget(n)` (default `Checker::BUDGET`).
- `verify_all(&space) -> Result<Report<World>, Refusal>`: the Model's own claims from the space's starts via the new additive `Checker::explore_from(world, starts, budget)`; a cut is an Unknown report outcome.
- `refuted(&space, fn(&S) -> bool) -> Result<Counterexample path, Refusal>`: the checker explores the claim-free `Rooted` world carrying the predicate as its one invariant; a counterexample before a cut stands; `Refusal::Holds { reached }` and `Refusal::Unknown { budget }` otherwise.
- `find(&space, path![..])`: reuses `search::through` with the space's starts and budget; `Refusal::Unbridged` names the waypoint index, last reached state and whether the budget cut it.
- Fixtures `space_verify_all`, `space_refuted`, `space_find` pin results and refusals (budget 0 checker refusal, cut, holds, unreachable, `..=n` too tight, start outside the space). README documents the explorations.

Deviations:
- Touched `umpire/checker/src/explore.rs` (additive `explore_from`, starts threaded through `seed`): claims are plain `fn(&M, ..)` pointers, so a wrapper world cannot re-root the Model's own claims; the checker has to accept starts.
- `find` takes the path by value (like `Run::along`); paths are the checker's `Path` type.
- Cost: each search leg explores everything reachable from its start (no early stop); fine for current fixtures. Per-leg shortest is greedy: with several starts matching the first waypoint, the first start in BFS order wins even if another gives a shorter whole path (documented in README).

make model-check: pass (default). make model-time:
```
[1;36mCheck model (fmt, clippy, docs, tests)...[0m
TLC backend skipped: no execution/agreement evidence; use UMPIRE_BACKENDS=require
timing-controls                   0.012s
capability-controls               0.078s
source-boundary                   0.577s
protobuf-freshness                0.061s
temporal-protobuf-freshness       0.060s
fmt                               0.186s
clippy                            2.166s
docs                             18.726s
tests                            14.773s
wasm32-wasip1                     0.216s
core-wasm-clippy                  0.329s
wasm-guest                        9.417s
go-host                           1.333s
temporal-go-host                  6.555s
inner source: umpire/checker/src/lib.rs
inner command: mise exec -- cargo test -p umpire-checker --quiet
inner loop (edit+test)            1.615s
gate total                       54.753s  (soft target 30s)
inner loop                        1.615s  (soft target 10s)
note: gate over soft target (fn-210 R30)
```

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 00f0ce3bc8
- Tests: make model-check, make model-time
- PRs: