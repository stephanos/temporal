---
satisfies: [R2, R3, R4, R5, R6]
---
# fn-160-one-ir-file-per-feature-collected-from.2 Collect one IR file per feature package; remove irFile and exports

## Description
Replace explicit `irFile` lists with one IR file per feature folder, collected by the IR generator from the feature's package. Then migrate every consumer that fn-160.1 inventoried.

**Size:** L
**Files:** `model/framework/IrFile.scala`, the lifter's IR-file entry point in `model/irgen`, `model/check` (gate), the `IrFiles` test, each feature file's `exports` object, `model/README.md`, `model/ir/*`, `model/cases/*`, the pinned fixture trees, and the Go tests and harnesses fn-160.1 lists
**Touches:** [model/framework/IrFile.scala, model/irgen/*.scala, model/check/*.scala, model/temporal/IrFiles.test.scala, model/temporal/features/**/*.scala, model/README.md, model/ir/*, model/cases/*, tests/testcore/testpilot/testdata/generated/*, tools/canary/casebinding/testdata/*, tools/umpire/**, tools/canary/**]

### Approach
- **Collection:** the lifter treats each feature folder (a folder with a feature file) as one IR file named after the feature, e.g. `activity-standalone`. Its roots are every machine object, composition, `queries` and `capabilities` section, progress claim and realization declared in the feature's package and sub-packages, plus what they reach.
- **Opt-out:** an explicit marker keeps a declaration out of `model/ir`. Report each opted-out declaration. Use it only where fn-160.1 found a reason.
- **Remove** `irFile` and every `exports` object. Update the feature-file layout rule (exports was step 5 of the reading order) and the README.
- **Gate:** lift per feature, settle `model/ir`, accept waivers, and have the `IrFiles` test construct each feature package's declarations.
- **Regenerate** the IR and Cases, and merge the per-file `.lint.json` acceptances. Check that every machine, Query, Case and claim is still present with the same behaviour, and that the only changes are file boundaries and names. Explain any other difference before accepting it.
- **Migrate** the Go tests, canary preflight, pinned fixture trees, and export README tables named in fn-160.1's inventory.

### Prerequisite
- fn-160.1's inventory, with any split the user decided to keep.
- Batch 5b's shared close (see MILESTONES.md): the collection tooling lands with the batch's tooling; this task closes the batch with the single regeneration of IR, Cases, lint acceptances and fixture trees, the equivalence seal against the batch's rename map, the full gate, review and live run.
- A working tree that builds the lifter and passes the Cases check (the in-progress IR `Duration` migration, plus `make protoc`).

### Quick commands
```sh
make umpire-gen-model
make umpire-check-model
make umpire-check-lint
make umpire-check-fixtures
```
## Acceptance
- [ ] Each feature folder produces exactly one IR file from its package; no Model declares `irFile` or `exports`.
- [ ] An explicit opt-out exists and the generator reports each declaration it keeps out.
- [ ] Regenerated IR and Cases keep every machine, Query, Case and claim; differences are limited to file boundaries and derived names.
- [ ] Every consumer from fn-160.1 is migrated; `make umpire-check-model`, `make umpire-check-lint` and `make umpire-check-fixtures` pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
