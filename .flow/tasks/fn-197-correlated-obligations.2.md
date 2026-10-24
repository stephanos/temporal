---
satisfies: [R1, R2, R3, R6]
---
# fn-197-correlated-obligations.2 DSL completion: the owes/until word and its checker semantics, with the fixture tests

## Description
**Note 2026-10-18 (one-language series):** `export` is `lift` (fn-201). Run conformance against `owes` waits for fn-204's descriptor (or the fn-209 oracle), not fn-196's conformance replay.

**DSL completion phase (owner, 2026-10-17).** After .1. In `model/framework/dsl/Assume.scala` and the Syntax macros, add `owes(open: K => F)` and `.until(close: K => F)` / `.until(close, within = n)`, a `LeadsTo`-shaped claim named after its `val` with `in` taking a Scenario; compile errors for `until` outside `owes`, differing payload types, another Machine's fact type, a multi-parameter constructor, `within` under one and `owes` outside `progress`. In the checker (`NativeCheck`, or fn-196's module if .1 of fn-196 has landed), explore the owed `Set[K]` with the vars as part of state identity, and report per payload deadlock or end while owed, fair non-progress cycle, missed deadline, with the payload, the opening step and the path; verified with the longest open interval on a complete exploration; unknown on a cut; a close while not owed changes nothing. Add `dsl/Owes.test.scala` with the R3 cases, including the `whenever(pending.nonEmpty)` claim that passes where `owes` refutes, and one compile-refusal test per error. Confirm the lifter leaves the claim out as it leaves `whenever` out.

**Size:** L

**Files:** `model/framework/dsl/Assume.scala`, `model/framework/dsl/Syntax.scala`, `model/framework/dsl/NativeCheck.scala` (or `model/framework/checker/*`), `model/framework/dsl/Owes.test.scala` (new), `model/framework/check/lint/BlockFormRule*.scala`.

### Verification
- `mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests` passes with the new suite counted.
- The block-form lint suite passes with an `owes` fixture.
- Quick: the README Native checks recipe.
## Acceptance
- [ ] owes/until declared in progress with within optional and in taking a Scenario; all six compile refusals pinned
- [ ] Owed set in explored-state identity; per-payload verdicts with payload, opening step and path; unknown on a cut
- [ ] Owes.test.scala covers R3 including the whenever-passes-owes-refutes contrast
- [ ] Lifter and export unchanged; a Model with owes lifts as before

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: