---
satisfies: [R2, R3]
---
# fn-214-progress-vocabulary-on-kettle-scopes.2 Bounded claims and named progress verdicts with hints

## Description
`Progress` gains `within: Option<u32>` and `.within(n)`; `n` must be positive (`within(0)` refused). The checker's progress check implements Scala's verdict order on the region: `Deadlock`, `Cycle` (no fair action leaves it), `MissedDeadline` (bounded: a path of `n` steps reaches no target, or a cycle a fair action leaves can still avoid the target), `Verified { longest }` (`None` when an unbounded region cycles), `Unknown` (cut or hole). `ClaimResult` carries `verdict` and `hint`; `Progress::assuming(..)` sets the claim's assumption (replacing the `#[within]` attribute); the aggregate `Outcome` is unchanged. Hint lines use Scala's wording with Rust names (see the spec). Checker-only fixtures pin each verdict kind and hint on a synthetic Model; Kettle's goldens are byte-identical.

**Touches:** `model/umpire/dsl/src/progress.rs`, `checker/src/{progress,report}.rs`, `checker/tests/fixtures/**`. Parallel with .1 after fn-210.19 (disjoint files except `progress.rs` in the checker: .1 edits the fairness source, .2 the verdicts; whichever lands second rebases). **Feedback loop (R30):** paste `make model-time` into the done summary.
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
