---
satisfies: [R1, R2, R8]
---
# fn-166-mini-activity-a-small-complete-model-to.1 The feature as the unit: FEATURES on every model target, discovered by default, per-feature gate report

## Description
---
satisfies: [R1, R2, R8]
---
# fn-166-mini-activity-a-small-complete-model-to.1 The feature as the unit: FEATURES on every model target, discovered by default, per-feature gate report

### Description
Make every model operation take a feature set whose default is discovered, and make the gate report each feature's state (R1, R2, R8). This is the mechanism the mini feature and every later DSL change rely on; no parking, no deletion, no list of features.

**Size:** L
**Files:** `tools/umpire/ir` (path helper), `tools/umpire/cmd/umpire-gen-cases`, `tools/umpire/cmd/umpire-repeat` and their tests, the Go tests that name `model/ir` or `model/cases`, `model/framework/IrFile.scala`, `.gitattributes`, `CODEOWNERS`, `Makefile` (`MODEL_SOURCES` and every target that uses it: jar, gate, lint, test, equivalence), `model/check/Gate.scala` (source enumeration, the per-feature report), `model/temporal/IrFiles.test.scala` and `MachineTablePins.test.scala` (read the compiled set), `model/README.md` (one paragraph)
**Touches:** [Makefile, .gitattributes, .github/CODEOWNERS, .plans/UMPIRE_MODULES.md, model/OutputPaths.scala, model/OutputPaths.test.scala, model/check/*.scala, model/check/test/*.scala, model/framework/IrFile.scala, model/framework/IrFile.test.scala, model/irgen/*.scala, model/irgen/test/**, model/irgen/testdata/**, model/temporal/*.test.scala, model/temporal/features/**/generated/**, model/temporal/foundations/**/generated/**, model/temporal/actors/**/generated/**, model/ir/**, model/cases/**, model/README.md, tools/umpire/**/*.go, tools/umpire/lint/testdata/**, tools/umpire/export/README.md, tools/canary/**/*test.go, tests/**/*test.go, tests/testcore/testpilot/model_fixture.go, common/testing/testpilot/**/*test.go]

### Approach
- Discovery: a feature is a folder under `model/temporal/features/` that holds a feature file, the existing layout rule (`features/activity/standalone`, `features/nexus/workflow`, ...). The Makefile discovers them with `$(wildcard)`/`find` at run time; `FEATURES` (space-separated, relative to `features/`) narrows the discovered set. No variable, file or test holds a list of features.
- `MODEL_SOURCES` becomes project.scala, framework, foundations, actors, capabilities, `Bounds` and the selected feature folders; every target that lists Model sources uses it, including the gate's own enumeration (follow how `Gate.scala` lists sources for the lift and pass the selected folders through).
- Per-feature report in the gate: for each discovered feature, `current` (compiles, IR matches), `behind` (does not compile under the current framework, or IR does not match), `missing` (no IR checked in). A full-set run with a behind feature fails naming it; a narrowed run reports only its set.
- `IrFilesTest` and `MachineTablePins` compare against the compiled set under any `FEATURES`, and report an unlisted feature's files as not in scope rather than failing.
- Outputs in the feature: move each feature's IR, lint acceptances and Cases into `features/<feature>/generated/` (`ir.json`, `ir.lint.json`, `manifest.json`, `<query>.case.json`, the Case name without its IR-file prefix) and delete `model/ir/` and `model/cases/`. One path helper per language (Go: the IR package; Scala: `IrFile`) resolves a feature's tree, and every consumer goes through it: the gate's stale/missing/left-over checks, `umpire-gen-cases` (every kind), `umpire-repeat`, the lint, the Quint export, the inventory tests, the lifter's fixtures. Use fn-160.1's inventory for the list. Go tests: a run scoped to one feature changes nothing outside its tree; an unscoped run regenerates every tree. Add `**/generated/** linguist-generated=true` to `.gitattributes` and one CODEOWNERS line per feature folder (owner to fill names). This is the spec's one Go change, an owner-granted exception to the batch's only-`model/` rule.
- `make model-mini` is an alias for `make model FEATURES=activity_mini` and is added here so task 2 can use it.
- Owner approval (2026-10-11): repair the four functional/canary check/update Make entrypoints so `FEATURES` scopes their inputs by the existing feature folders. Narrow private Go publication logic may preserve the central pinned tree while updating only selected existing pins; no new CLI selector or public semantic API. Preserve pin identities and unselected payload bytes, manifest rows and modification times under the existing lock. A selected feature with no pins is a no-op. Add actual Make-entrypoint and partial-publication tests for check/update, no-pin selection and preservation. Do not regenerate protected real artifacts, repin Cases, restore protobufs or implement deferred Go behavior.
- `umpire-check-lint` takes `FEATURES` too (owner, 2026-10-11): the target names the selected features' IR files, `umpire-lint $(wildcard $(addsuffix /generated/ir.json,$(MODEL_FEATURE_PATHS)))`, as the gate's lint step names `selectedIr`. The default set still lints every feature that has an IR; a feature not yet lifted is skipped by the `wildcard`, not failed; a scoped run checks for orphan acceptance files only in its own folders. Naming files skips the frozen payloads under `generated/legacy/`, which the gate already skips and fn-160 retires, so the two routes agree. Fix the target's comment and the `make umpire-check-lint` row of `.plans/UMPIRE_MODULES.md`, which still say `model/ir`.
- Prove with one discovered real feature alone: compiles, lifts, passes the Scala suites; `git status` shows no change under the other features; the report covers only the selected feature. Historical IR drift must be reported as behind, not hidden by regenerating preserved payloads. The full-set behind failure is a required negative proof (R2), not evidence that the whole spec is complete.

### Acceptance
- [ ] Every model target takes `FEATURES`; the default is discovered from the tree and no list of features exists in the build or the tests; `make model-mini` is the alias.
- [ ] `make umpire-check-lint FEATURES=<feature>` lints only that feature's `generated/ir.json` and touches no other folder; the bare target lints every lifted feature; a listed feature without an IR is skipped.
- [ ] With one feature listed, only it and the shared code are compiled, lifted, checked and tested; the others are untouched on disk.
- [ ] The gate reports each discovered feature as current, behind or missing; a full-set run with a behind feature fails naming it.
- [ ] `IrFilesTest` and `MachineTablePins` pass under the default and under a single-feature set.
- [ ] Every feature's IR, lint acceptances and Cases live in `features/<feature>/generated/`; `model/ir/` and `model/cases/` are gone; every consumer resolves paths through the helper; a run scoped to one feature changes nothing outside its tree (Go test).
- [ ] `.gitattributes` marks `generated/` trees as generated and CODEOWNERS has one line per feature folder.

### Done summary
TBD

### Evidence
- Commits:
- Tests:
- PRs:
## Acceptance
- [ ] Every model target takes `FEATURES`; the default is discovered from the tree and no list of features exists in the build or the tests; `make model-mini` is the alias.
- [ ] `make umpire-check-lint FEATURES=<feature>` lints only that feature's `generated/ir.json` and touches no other folder; the bare target lints every lifted feature; a listed feature without an IR is skipped.
- [ ] With one feature listed, only it and the shared code are compiled, lifted, checked and tested; the others are untouched on disk.
- [ ] The gate reports each discovered feature as current, behind or missing; a full-set run with a behind feature fails naming it.
- [ ] `IrFilesTest` and `MachineTablePins` pass under the default and under a single-feature set.
- [ ] Every feature's IR, lint acceptances and Cases live in `features/<feature>/generated/`; `model/ir/` and `model/cases/` are gone; every consumer resolves paths through the helper; a run scoped to one feature changes nothing outside its tree (Go test).
- [ ] `.gitattributes` marks `generated/` trees as generated and CODEOWNERS has one line per feature folder.

## Done summary
# fn-166.1 reviewed feature-scoped implementation

FEATURES scopes model compilation, checks and output consumers to discovered feature folders and shared code. All historical outputs are relocated losslessly into their existing owners, with supplemental artifacts under generated/legacy/. Functional/canary Make check/update targets now use source-folder scope while preserving global pin locations and identities; selected no-pin features do nothing.

Tier: project routing. Review override codex:gpt-6.1-sol:high. Writer and reviewer are from the same model family. Original review/done base: 47a2f69a23ad9fd6bbe7998257ff6918e3e794d0. First round returned NEEDS_WORK; all seven findings are fixed and tested. Reviewed implementation checkpoint: 2826487b8b981e4cb5363151975324fe19afbe04. Round2 returned SHIP with no surviving blocking findings; the reviewer verified code and Make dry runs but could not independently run Go in its read-only sandbox. Worker runtime receipts below are separate.

Verification:
- Settled full lifter: exit0,195 tests, .flow/tmp/fn166/lifter-review-final.log. Irgen sources and fixtures have not changed since that frozen-tree run.
- Default Scala183 and selected workflow99 passed; full check111 passed at the prior checkpoint. Only MakeFeaturesSuite changed afterward: its new real-entrypoint table reproduced root-CWD failure then passed2 focused tests, /tmp/fn166-pinned-make-{red,green}.log.
- Partial publisher/no-pin tests reproduced missing-unselected-pin failures before implementation. Invalid manifest symlink/extra-entry and generated/file scope controls reproduced unintended success before guards. Final command package suite exit0, .flow/tmp/fn166/go-pinned-command-final.log; latest scoped cmd/IR suite exit0, go-pinned-settled-final.log.
- Existing lower whole-tree publication, selected-byte identity and refusal guards passed: go test -count=1 -tags test_dep ./tools/umpire/lower -run 'Test(CasePublicationDetectsDriftAndPreservesTheOldTreeOnInvalidInput|SelectedCasesAreTheCompleteTreesBytes|SelectingRefusesAQueryWithNoCase)$'; lower-pinned-final.log.
- Actual CLI tests build the command and run both pinned kinds in source-only no-IR feature folders: check/update exit0 without creating outputs; outside scope exits1. Publication tests retain unselected Case bytes, manifest rows and mtimes, reject unsafe trees and locks, and keep checks read-only.
- node .flow/tmp/fn166/recover-relocation.mjs: exit0,44 original non-manifest payloads byte-identical exactly once;340 query rows identical modulo approved paths; every referenced Case resolves. Production feature sources, lint bytes, pins and equivalence baseline remain unchanged.
- Direct full lift and expected-negative full-set behind assertion passed; all three preserved features are reported behind without publication. Evidence logs: full-lift-settled-final.log and behind-settled-final.log.
- Seven review fixes include shared functional readers, both active Case Make entrypoints, selected-package fixture isolation, complete shared/default equivalence baseline comparisons, all4270 original Position oracles, and exact neutral namespace/import boundaries. Original eight goldens and their strict non-position/identity invariants remain unchanged. Only owner-approved hints/hintsRefused behavioral snapshots and exact refusal coordinates changed.
- git diff --check exit0; Flow validate --spec fn-166-mini-activity-a-small-complete-model-to exit0,valid7. An incidental validate --all exits1 on unrelated historical Flow inconsistencies; no repository-wide validation pass is claimed.

Explicit limits:
The canonical make model FEATURES=nexus/workflow MODEL_GATE_ARGS=--skip-lint exits2 on owner-held stale claim.pb.go. Fresh real-feature Go reading exits1 on unknown Realization.queries, declared in proto field21 but absent from generated Go. Neither is lint or success. These are retained task1 behind/schema debt; mini realization and full-spec correctness remain unproven. Lint is owner-deferred, not passed; default global lint policy remains intact. Broader optional Go diagnostics exited1 on classified inherited structural/held-output debt, not a broad pass.

The scoped-publication authority blocker was explicitly resolved by the owner; no decision is pending. Task6 owns generated/cases/ relocation, task7 target renaming, fn160 real-source flattening/collection. Conductor-authored fn160.8/fn166.7/fn167 Flow/MILESTONES planning rides catch-all commits and is not extra task1 implementation. Root alone owns MILESTONES.

Review receipt: /tmp/impl-review-receipt-8f37faba39e2-fn-166-mini-activity-a-small-complete-model-to.1.json.
stage: impl-review - ran (NEEDS_WORK → SHIP; original47a2f69a23ad9fd6bbe7998257ff6918e3e794d0..2826487b8b981e4cb5363151975324fe19afbe04; codex:gpt-6.1-sol:high; same-family writer/reviewer)

Post-review verification: executable tree unchanged; git diff --check and44/340 proof passed. Gate classification reports FULL because the cumulative task changes executable paths; no false whole-HEAD green receipt or canonical Make credit was minted. Full source-specific195/183/99/111 runs and fresh focused Make/publication guards provide the bounded task1 proof. Nontrivial review fixes were captured in bug/integration/output-relocation-must-scope-shared-2026-10-11. No commands remain running.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: b6072d7e79db601bb1f72ab707059b4e6afcebbb, e94f6f8cf9a9a39579e85b5e887a7c60a6b6d674, 9cb3fad5a6a4fd6adfa139b32c7ddded97b4ec52, 17c3a4006bc71e41f9febb16e7bed33782b9bb4d, 2826487b8b981e4cb5363151975324fe19afbe04
- Tests: mise exec -- scala-cli test model/irgen — exit0,195 tests;.flow/tmp/fn166/lifter-review-final.log;irgen source tree unchanged since frozen run, mise exec -- scala-cli test --server=false model/project.scala model/framework model/temporal — exit0,183 tests;.flow/tmp/fn166/full-scala-resume.log, make test-model FEATURES=nexus/workflow — exit0,99 tests;.flow/tmp/fn166/scoped-scala-resume-final.log, mise exec -- scala-cli test model/check — exit0,111 tests;.flow/tmp/fn166/check-review-final.log;subsequent changed Make test separately covered, mise exec -- scala-cli test --server=false model/check --test-only umpire.check.MakeFeaturesSuite — intended red exit1 then green exit0,2 tests;/tmp/fn166-pinned-make-{red,green}.log, go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-cases — exit0;.flow/tmp/fn166/go-pinned-command-final.log;actual CLI no-pin/check/update and invalid-scope regressions, go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-cases ./tools/umpire/ir -run Test.*(Feature|Managed|ModelPath|ModelNames|Publication|AddedFeature|Cases|Selection|Pinned) — exit0;.flow/tmp/fn166/go-pinned-settled-final.log, go test -count=1 -tags test_dep ./tools/umpire/lower -run Test(CasePublicationDetectsDriftAndPreservesTheOldTreeOnInvalidInput|SelectedCasesAreTheCompleteTreesBytes|SelectingRefusesAQueryWithNoCase)$ — exit0;.flow/tmp/fn166/lower-pinned-final.log, node .flow/tmp/fn166/recover-relocation.mjs — post-review exit0;44 byte-identical payloads exactlyonce,340 manifest rows identical modulo approved paths, Direct full lift exit0 and required expected-negative all-three-feature behind assertion exit0;.flow/tmp/fn166/full-lift-settled-final.log and behind-settled-final.log, make model FEATURES=nexus/workflow MODEL_GATE_ARGS=--skip-lint — exit2,owner-held stale claim.pb.go;not canonical success, Fresh selected real-feature Go reader — exit1,unknown Realization.queries;disclosed behind/schema debt,not lint or correctness success, LINT_DEFERRED: owner instruction;existing companion bytes and global default preserved, Full-original-range review round2 SHIP;codex:gpt-6.1-sol:high;same-family writer/reviewer;reviewer sandbox could not independently run Go
- PRs: