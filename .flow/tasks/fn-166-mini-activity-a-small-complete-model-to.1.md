---
satisfies: [R1, R2, R8]
---
# fn-166-mini-activity-a-small-complete-model-to.1 The feature as the unit: FEATURES on every model target, discovered by default, per-feature gate report

## Description
Make every model operation take a feature set whose default is discovered, and make the gate report each feature's state (R1, R2, R8). This is the mechanism the mini feature and every later DSL change rely on; no parking, no deletion, no list of features.

**Size:** L
**Files:** `tools/umpire/ir` (path helper), `tools/umpire/cmd/umpire-gen-cases`, `tools/umpire/cmd/umpire-repeat` and their tests, the Go tests that name `model/ir` or `model/cases`, `model/framework/IrFile.scala`, `.gitattributes`, `CODEOWNERS`, `Makefile` (`MODEL_SOURCES` and every target that uses it: jar, gate, lint, test, equivalence), `model/check/Gate.scala` (source enumeration, the per-feature report), `model/temporal/IrFiles.test.scala` and `MachineTablePins.test.scala` (read the compiled set), `model/README.md` (one paragraph)
**Touches:** [Makefile, .gitattributes, .github/CODEOWNERS, .plans/UMPIRE_MODULES.md, model/OutputPaths.scala, model/OutputPaths.test.scala, model/check/*.scala, model/check/test/*.scala, model/framework/IrFile.scala, model/framework/IrFile.test.scala, model/irgen/*.scala, model/irgen/test/**, model/irgen/testdata/**, model/temporal/*.test.scala, model/temporal/features/**/generated/**, model/temporal/foundations/**/generated/**, model/temporal/actors/**/generated/**, model/ir/**, model/cases/**, model/README.md, tools/umpire/**/*.go, tools/umpire/lint/testdata/**, tools/umpire/export/README.md, tools/canary/**/*test.go, tests/**/*test.go, common/testing/testpilot/**/*test.go]

### Approach
- Discovery: a feature is a folder under `model/temporal/features/` that holds a feature file, the existing layout rule (`features/activity/standalone`, `features/nexus/workflow`, ...). The Makefile discovers them with `$(wildcard)`/`find` at run time; `FEATURES` (space-separated, relative to `features/`) narrows the discovered set. No variable, file or test holds a list of features.
- `MODEL_SOURCES` becomes project.scala, framework, foundations, actors, capabilities, `Bounds` and the selected feature folders; every target that lists Model sources uses it, including the gate's own enumeration (follow how `Gate.scala` lists sources for the lift and pass the selected folders through).
- Per-feature report in the gate: for each discovered feature, `current` (compiles, IR matches), `behind` (does not compile under the current framework, or IR does not match), `missing` (no IR checked in). A full-set run with a behind feature fails naming it; a narrowed run reports only its set.
- `IrFilesTest` and `MachineTablePins` compare against the compiled set under any `FEATURES`, and report an unlisted feature's files as not in scope rather than failing.
- Outputs in the feature: move each feature's IR, lint acceptances and Cases into `features/<feature>/generated/` (`ir.json`, `ir.lint.json`, `manifest.json`, `<query>.case.json`, the Case name without its IR-file prefix) and delete `model/ir/` and `model/cases/`. One path helper per language (Go: the IR package; Scala: `IrFile`) resolves a feature's tree, and every consumer goes through it: the gate's stale/missing/left-over checks, `umpire-gen-cases` (every kind), `umpire-repeat`, the lint, the Quint export, the inventory tests, the lifter's fixtures. Use fn-160.1's inventory for the list. Go tests: a run scoped to one feature changes nothing outside its tree; an unscoped run regenerates every tree. Add `**/generated/** linguist-generated=true` to `.gitattributes` and one CODEOWNERS line per feature folder (owner to fill names). This is the spec's one Go change, an owner-granted exception to the batch's only-`model/` rule.
- `make model-mini` is an alias for `make model FEATURES=activity_mini` and is added here so task 2 can use it.
- Prove with one discovered real feature alone: compiles, lifts, passes the Scala suites; `git status` shows no change under the other features; the report covers only the selected feature. Historical IR drift must be reported as behind, not hidden by regenerating preserved payloads. The full-set behind failure is a required negative proof (R2), not evidence that the whole spec is complete.

## Acceptance
- [ ] Every model target takes `FEATURES`; the default is discovered from the tree and no list of features exists in the build or the tests; `make model-mini` is the alias.
- [ ] With one feature listed, only it and the shared code are compiled, lifted, checked and tested; the others are untouched on disk.
- [ ] The gate reports each discovered feature as current, behind or missing; a full-set run with a behind feature fails naming it.
- [ ] `IrFilesTest` and `MachineTablePins` pass under the default and under a single-feature set.
- [ ] Every feature's IR, lint acceptances and Cases live in `features/<feature>/generated/`; `model/ir/` and `model/cases/` are gone; every consumer resolves paths through the helper; a run scoped to one feature changes nothing outside its tree (Go test).
- [ ] `.gitattributes` marks `generated/` trees as generated and CODEOWNERS has one line per feature folder.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
