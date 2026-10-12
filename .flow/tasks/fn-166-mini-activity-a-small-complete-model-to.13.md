# fn-166-mini-activity-a-small-complete-model-to.13 Change-scoped model check: scope from the diff, suite coverage registry, evidence record

## Description
Owner decision, 2026-10-12 (parts 1 and 3 of the change-scoped model loop; runs before fn-141).

Do:
1. One command, e.g. `make model-check-changed [BASE=<rev>]`, derives the scope from `git diff` against BASE (default: the last commit, plus the working tree): a path under a feature folder selects that feature; framework, lifter (`model/irgen`, `model/check`, `model/migrate`), shared code (foundations, actors, capabilities, realize, `Bounds`) select every feature that is not behind (today: `activity_mini`). It runs the scoped compile, lift, step-table equivalence against the selected feature's baseline, and exactly the test suites that cover the changed paths.
2. Every Scala test suite (framework, Models, lifter, migrate, check) declares what it covers (paths and, where it lifts Models, which features), in one registry the command reads; a suite covering a behind feature is skipped with that reason under a mini-only scope. fn-166.11 does the lifter suites' admission/retargeting; this task adds the registry for all suites and uses it.
3. The command writes a fixed evidence record (`.flow/tmp/checks/<timestamp>/record.md`: scope and why, commands, exit codes, counts, wall times) that a task receipt can cite instead of bespoke proof scripts.
4. Update the worker-facing docs (model/README.md "fast loop" section and AGENTS.md's model mandate line) so the command is the default per-change check; full gates stay the batch-end/sweep tier.

## Acceptance
- A change under `features/activity_mini` selects only mini and its covering suites; a lifter change selects the lifter suites plus every non-behind feature; a docs-only change selects nothing and says so.
- Every suite is in the registry (a check fails on an unregistered suite).
- Evidence record written with real exit codes and wall times; a deliberately failing change makes the command fail.
- Before/after wall time of a typical mini-only change and of a lifter change recorded.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
