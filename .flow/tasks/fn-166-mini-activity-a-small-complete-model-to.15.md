# fn-166-mini-activity-a-small-complete-model-to.15 Lifter suites: share the TASTy read across lifts, in-process conventions lint, bounded parallel suites

## Description
Follow-up from fn-166.14 (2026-10-12). A lifter-change check still takes 506 s against the 5-minute target. The remaining cost is about 403 lifts per run at about 0.8 s each, because each lift reads the whole Models jar's TASTy (about 320 s), plus `lintConventions` subprocesses (36 s) and the remaining scala-cli packaging (Gaps, Facets, Bloop refusal compiles). Runs after fn-141, which rewrites the lift path.

Do:
1. Share the TASTy read: one inspected Models classpath per JVM, reused across lifts.
2. Run `lintConventions` in-process.
3. Run the lifter suites side by side in one JVM with a bounded runner, keeping memory within budget on a shared machine.
4. Measure `make model-check-changed` on a lifter change and on a mini-only change, before and after. Target: a lifter change under 5 minutes.

## Acceptance
- Lifter-change check under 5 minutes, all tests passing; before/after recorded.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
