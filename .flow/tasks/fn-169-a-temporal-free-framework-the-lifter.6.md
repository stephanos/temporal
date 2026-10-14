# fn-169-a-temporal-free-framework-the-lifter.6 Port the Python interpreter-table comparator to Scala in framework/check/equivalence; pins to temporal/check

## Description
Owner decision, 2026-10-12: fold the Python interpreter-table comparator in `model/testdata/table-pins/` (fn-156) into the fn-169 layout, so the model tooling is Scala and Go only. Runs after fn-169.1 and before the sweep needs the full interpreter route.

Today: `replay.py` runs the sealed fn-155 Go table dumper over generated IR and compares every record (catalogs, rows, ordered alternatives, outcomes, facts, explanations) with `pins.json` (61 tables, seven activity/Nexus artifacts, 13.9M records); `mutants.py` seeds named mutations to prove the comparator catches them; `test_replay.py` holds its tests. `Gate.scala` calls `python3 replay.py` only on the full equivalence route (`--dumper`, `make umpire-check-equivalence DUMPER=...`).

Do:
1. Port the comparator to Scala in `framework/check/equivalence/`, beside the step-table comparison, sharing its rename-map handling; the gate calls Scala instead of `python3`. Keep every failure class (missing/extra artifacts or subjects, incomplete dumps, catalog/row/result/order changes, changed interpreter scope, input mutation during observation) and the heavy-gates lock.
2. Port the mutation controls and `test_replay.py` into that module's colocated Scala tests.
3. Move `pins.json` and its README to the Temporal side (`temporal/check/`), as the sweep's "before" data for the real features. Delete the Python files.
4. Prove parity: on the fixture streams the Python tests use, and on a mini dump if a dumper is available, the Scala comparator reaches the same verdicts as the Python one did before deletion. A full real-feature replay stays with the sweep.

## Acceptance
- No Python remains under `model/`; the gate's full equivalence route calls the Scala comparator.
- Every mutation control fails as before; the ported tests pass; parity with the Python verdicts is recorded.
- `pins.json` lives under `temporal/check/` with its README.


## Done summary
The model's full equivalence route now compares interpreter tables in Scala, with shared rename accounting, streaming records and bounded diagnostics. Historical pins and their README live under `model/temporal/check/table-pins`; `pins.json` moved byte-identically, and no Python source or bytecode remains under `model/`.

Tier: session (jev-unavailable(no_key))

baseline: green via the preceding task's `.flow/tmp/checks/20261012-231353/record.md`, reused under the owner's 2026-10-13 faster-tasks policy. The unchanged Python regression suite also passed eight tests before implementation. The strict `FEATURES=activity_mini make model-mini` inherited IR drift remains held for the sweep; this task did not run that strict gate or change generated artifacts.

`InterpreterTablesSuite` preserves the Python fixture's exact bytes and literal SHA-256. Python evaluated 28 cases before deletion, including every `mutants.py` seed. Scala consumed the same gzip fixtures and matched every verdict and the reported rename/removal uses. The parity input is `.flow/tmp/fn169.6-parity/manifest.json`; `.flow/tmp/fn169.6-final-focused.log` records 51 passing affected tests, including parity. Initial comparator and native-mutator tests ran red before their implementations (`.flow/tmp/fn169.6-red.log`, `.flow/tmp/fn169.6-native-red.log`).

The colocated tests cover row/result/alternative/fact/binding/explanation/pair controls, paired native records, nine rejected mutation observations, family-qualified and duplicate subjects, removed/new subjects, inventory scope and declaration/file mismatches, incomplete/unavailable streams, missing/changed dumpers, failed processes, IR input mutation, shared native flock exclusion, and bounded diagnostics over 20,000 changed rows. `GateEquivalenceOutputsSuite` proves that an inconclusive native error propagates and leaves an inconclusive receipt instead of passing.

`make model-check-changed BASE=15643567e3` passed once at the end in 479.5 seconds. Its record is `.flow/tmp/checks/20261012-235203/record.md`; 75 selected suites passed 472 tests, along with syntax/Scalafix lint, the Temporal-free build boundary, samples, mini compile/lift and mini step-table equivalence. The record explicitly holds the checked-in mini IR drift and sweep-only Go/real-feature stages. Changed Scala files were formatted with the repository's Scalafmt configuration, and `git diff --check` passed.

Mini native parity was unavailable because the sealed fn-155 dumper is absent at both `.flow/tmp/fn-155/task1/tables/dump` and `.worktrees/fn-155-name-the-standalone-activitys-repeated/.flow/tmp/fn-155/task1/tables/dump`. No replacement dumper or broad real-feature replay was built or run. Full interpreter replay remains the sweep's obligation. The immutable `dispatch-original` source fixture keeps its historical pin path spelling; live callers use the relocated pins. Deleted Python sources remain recoverable from git history.

stage: impl-review - skipped(policy: MILESTONES.md source-batch policy defers independent review to the Scala batch end; REVIEW_MODE=none)
stage: plan-sync - skipped(config: planSync.enabled=false)
## Evidence
- Commits: 3e2da95b37add9d1bd04c310749ca34104341e4f
- Tests: python3 model/testdata/table-pins/test_replay.py, env INTERPRETER_PARITY=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn169.6-parity mise exec -- scala-cli run --server=false --test model/framework/check model/temporal/check --suppress-outdated-dependency-warning -M org.junit.runner.JUnitCore -- framework.check.equivalence.InterpreterTablesSuite temporal.check.GateEquivalenceOutputsSuite framework.check.equivalence.EquivalenceSuite framework.check.ToolsSuite temporal.check.CoverageSuite, make model-check-changed BASE=15643567e3
- PRs: