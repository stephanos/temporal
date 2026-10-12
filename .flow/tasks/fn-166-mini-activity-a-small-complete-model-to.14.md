# fn-166-mini-activity-a-small-complete-model-to.14 Lifter suites compile each fixture once per run and lift in a shared JVM

## Description
Conductor decision under the owner's standing instruction to follow its own recommendations, 2026-10-12. This is part 5 of the change-scoped loop proposal. It runs before fn-169.1 and fn-141, whose tasks all change the lifter.

Measured (fn-166.13): a lifter change runs all 36 lifter suites in 633–901 s (723 s typical), and a mini-only change spends 340 s in 11 mini lifter suites. Most of this is per-test fixture builds: each test copies a fixture, compiles it with its own `scala-cli --server=false` and lifts it in a fresh JVM (`model/irgen/test/Fixtures.test.scala`: "A test builds a fixture with scala-cli and lifts it in a JVM of its own"). The heaviest single tests build whole projects only to see them refused (50–77 s each).

Do:
1. Compile each fixture project once per test run, shared across every test that reads it (a run-scoped cache keyed by the fixture's source hash), instead of per test.
2. Lift in-process, or in one long-lived lifter JVM per run, wherever a test only needs the lift output (`InProcess.lift` already exists for some suites).
3. Batch the compile-refusal tests: compile many deliberately broken sources in one scala-cli invocation where the diagnostics can still be attributed per file and line, or check them through the compiler API in-process.
4. Keep every assertion and refusal message; any test whose meaning would change is left as is and listed.
5. Measure the full `model/irgen` suite and `make model-check-changed` on a lifter-only change and on a mini-only change, before and after. Target: a lifter change under 5 minutes.

## Acceptance
- Fixture projects compile once per run; lifts reuse one JVM or run in-process where possible; refusal tests batched where attribution is kept.
- Same tests and assertions pass (274+ pass, 0 fail, the recorded sweep-held skips).
- Before/after wall times recorded for the full irgen suite and for model-check-changed on a lifter change and a mini change.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
