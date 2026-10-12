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
Change-scoped model check: `make model-check-changed [BASE=<rev>]` (default BASE=HEAD~1, plus working tree and untracked files). Commits 7f37be9d2f, 1b9effea96, 901c8693d3.

Requirement evidence
- One command (model/check/ChangeCheck.scala, Makefile target): scope from `git diff --name-only --no-renames BASE` + untracked; lint (SyntaxRule/CommentRule/BlockFormRule over the selected sources; `lint-model-check` when model/check changed), `make compile-model`, `make lift-model LIFT_OUT=<record>/lift`, `make umpire-check-equivalence STEPS_ONLY=1` against the feature's baseline under the committed map model/check/baselines/activity_mini.map (copy of fn160/task3/steps.map), then exactly the covering suites per module via `scala-cli run --test <module> --main-class org.junit.runner.JUnitCore`. Stops at the first failing stage.
- Path-to-scope data in one place, model/check/Coverage.scala: glob rules (incl. planned fn-169.1 homes model/{framework,temporal}/{export,check,migrate}), current features (activity_mini), baselines, modules, and the registry of all 112 suites (Models 47, lifter 36, check 15, migrate 14). Suites reading behind features are skipped with the reason. irgen SuiteCoverage.mini now reads the registry (irgen project includes ../check/Coverage.scala).
- Unregistered suite / stale entry: CoverageSuite fails and the command refuses (tested).
- Record: .flow/tmp/checks/<time>/record.md (scope and why, suites with reasons, stages with command, exit, wall, JUnit counts, held items, not-run items) plus a live log per stage.
- Acceptance scopes (CoverageSuite + real runs): mini edit -> activity_mini + 22 suites (5 Model, 11 mini lifter, 2 check, 4 migrate); lifter edit -> activity_mini + 36 lifter + 4 check suites, no Model suite; docs-only (AGENTS.md, model/README.md) -> "nothing in scope", exit 0; behind-feature edit -> nothing checked, named behind, its suites skipped.
- Deliberate failing change (mini `attempts < 1`): exit 2 at equivalence, 10 table lines differ (deliberate-fail.log).
- Stale committed IR: fresh lift differing from features/activity_mini/generated/ir.json is reported "held" (Coverage.committedIrHeld); make model-mini keeps the strict compare.
- Docs: model/README.md "Writing a Model" and AGENTS.md model line name the command as the default per-change check; full gates are batch-end/sweep tier.

Lint debt fixed (7f37be9d2f)
- Rule corrected: SyntaxRule read `flags.is(Flags.Module)` (quotes-reflect API) in FacetSource/InputSource/RuleSource:44 as the `is` sugar; a member selection now counts only for sugar defined as an extension method (new SyntaxRuleSuite test).
- Sources fixed: RuleSource.disabled -> disabledLabels (sugar name outside Syntax.scala); mini Entering.test rule in block form; Gate.build `var completed` and StepLedger `var size` removed; Gate.test stub drops a needless `s` interpolator (scalafix's suggested fix would have kept `$$`); PropertyGroupMigration.test uses full package names so mini-scoped builds exclude it (make test-model FEATURES=activity_mini failed to compile before).
- Mini lift byte-identical to fn-166.12's lift-out after the rename.

Wall times (stage-times.txt)
- Before (old routine: model-mini skip-lint + equivalence + test-model mini + full irgen/check/migrate suites): 731.8s.
- After, mini-only edit: 463.3s (mini lifter suites 339.6s dominate). After, lifter edit: 797.9s (36 lifter suites 722.7s; machine noise, irgen suite measured 633-901s across runs). Docs-only: ~5s.

Verification
- make model-check-changed BASE=94732a8379 (this task's whole change): exit 0, 728s; Models 30 suites 139 tests, lifter 36 suites 275, check 15 suites 127, migrate 5 suites 14; 17 behind suites skipped (record .flow/tmp/checks/20261012-155050).
- Old routine on the final tree: irgen 274 pass/6 sweep-held, check 127, migrate 64, test-model mini 139, all pass.

Failures retained / not run
- Scalafix lint of Models (lint-model-models: Witness.test etc.) and lifter (lint-model-irgen: 28 DisableSyntax findings) is inherited debt; the command names it not-run (batch-end tier).
- Go checks and the strict committed-IR/Cases compare stay held (make model-mini still exits 2 at the IR compare).
- CapabilityPropertiesTest reads real features: registry skips it under the mini scope.

Deferred gates: full gates and independent review at batch end.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 7f37be9d2f, 1b9effea96, 901c8693d3
- Tests: make model-check-changed BASE=94732a8379: exit 0 (record .flow/tmp/checks/20261012-155050), after-mini 463.3s exit 0, after-lifter 797.9s exit 0, deliberate meaning change exit 2, before routine 731.8s
- PRs: