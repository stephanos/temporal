---
satisfies: [R4]
---
# fn-196-the-scala-checker-is-the-oracle.8 Checker performance: define each claim section class once, then the per-step costs

## Description
Bank's verifyAll takes about 83 s for 4,624 states and 739,840 steps (about 112 us per step). A code read on 2026-10-18 attributes about 97% of that to `ClaimEvaluation.evaluate`: `Reload.initialise` creates a new ClassLoader, `defineClass`es the rewritten section bytes, runs `Class.forName` with initialization and falls back to a reflective constructor on every evaluation (once per state for `invariants`, once per edge for `postconditions`), so the claim body never JITs and metaspace churns (the Voting7 run's full GCs). Only the bytes are cached.

Do, in order, measuring Bank and Voting5 after each with the measurement harness:
1. Define the reloaded section class once per (section, owner): move the initializer body into a re-runnable method in `NativeClaimSections.prepare`, cache a MethodHandle or instance, call it per evaluation. Expected: Bank 83 s to about 2-4 s.
2. Memoize postcondition results by (before, after, outcome, facts) when the section reads no taken class (no `targeting`).
3. `VarRuntime.at`: emit getter/setter lambdas from VarSchemaMacro instead of reflective `Method.invoke`; skip the resume restore at depth 0.
4. Validate the input once when the catalog is built and pass the existing Class through `stepFunction` instead of `classOf(decl, input)` (input is validated three times per step today).
5. Dedup: cache the hash in `State`, key on `State` when not positioned, one `getOrElseUpdate`, drop the pending Edge.

Update the measurement note's table with the new numbers and keep the old rows as history. Semantics must not change: every native test and the samples' verdicts stay identical.

**Files:** model/framework/checker/ClaimEvaluation.scala (Reload, evaluate), model/framework/checker/NativeClaimSections.scala, model/framework/checker/VarRuntime.scala, model/framework/checker/Effects.scala, model/framework/checker/InputRuntime.scala, model/framework/checker/NativeCheck.scala (Search.reach, walk), model/framework/dsl/Vars.scala (State.hashCode), model/framework/design/scala-checker-measurement.md.

**Quick commands:** `mise exec -- scala-cli test --server=false model/framework/checker model/framework/samples --require-tests`; `env -u BASH_ENV UMPIRE_MEASUREMENT_OUT="$PWD/.flow/tmp/fn196perf/run1" bash model/framework/design/scala-checker-measurement.sh build` then `... bank time 1048576 3` and `... voting5 time 1048576 3`

## Acceptance
- Bank verifyAll under 5 s and Voting5 under 2 s on the measurement harness, same verdicts.
- No new class defined per claim evaluation (assert via a counting ClassLoader in a test, or JFR class-load count).
- All native tests pass; measurement note updated with before/after rows.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: