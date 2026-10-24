---
satisfies: [R1]
---
# fn-199-native-refinement-replacement-and.1 Refinement simulation over two reachable graphs

## Description
Implement native refinement checking in the checker: explore the refining and refined instances under the budget, apply the typed state map, and classify failures as catalog, initial, unmatched, visible stutter or incomplete (budget cut or unread hole). Each refutation carries a concrete path and the product witness. Expose `Checker.refines(instance, refinement, budget)` and include refinement results in `verifyAll`.

**Files:** model/framework/checker/NativeCheck.scala, model/framework/checker/Checker.scala, model/framework/checker/Result.scala, new model/framework/checker/RefinementRuntime.scala, tests under model/framework/checker/.

**Quick commands:** `mise exec -- scala-cli test --server=false model/framework/checker --require-tests`

## Acceptance
- Verified, refuted (with path and failure kind) and incomplete answers for sample refinements and for synthetic controls of each failure kind.
- A map throwing on a reachable state refutes with that state; a missing refined start set refuses at declaration.
- Lifetimes: the refined graph is released with the Result.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: