# fn-166-mini-activity-a-small-complete-model-to.15 Lifter suites: share the TASTy read across lifts, in-process conventions lint, bounded parallel suites

## Description
**Note 2026-10-18 (one-language series):** After fn-201's `export` to `lift` rename.

Follow-up from fn-166.14 (2026-10-12). A lifter-change check still takes 506 s against the 5-minute target. The remaining cost is about 403 lifts per run at about 0.8 s each, because each lift reads the whole Models jar's TASTy (about 320 s), plus `lintConventions` subprocesses (36 s) and the remaining scala-cli packaging (Gaps, Facets, Bloop refusal compiles). Runs after fn-141, which rewrites the lift path.

Do:
1. Share the TASTy read: one inspected Models classpath per JVM, reused across lifts.
2. Run `lintConventions` in-process.
3. Run the lifter suites side by side in one JVM with a bounded runner, keeping memory within budget on a shared machine.
4. Measure `make model-check-changed` on a lifter change and on a mini-only change, before and after. Target: a lifter change under 5 minutes.
## Acceptance
- Lifter-change check under 5 minutes, all tests passing; before/after recorded.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: