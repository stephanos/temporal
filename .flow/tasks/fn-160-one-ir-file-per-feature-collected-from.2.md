---
satisfies: [R2, R3, R4, R5, R6, R10, R11]
---
# fn-160-one-ir-file-per-feature-collected-from.2 Collect one IR file per feature package; remove irFile and exports

## Description
Replace explicit `irFile` lists with one IR file per feature folder, collected by the IR generator from the feature's package. Then migrate every consumer that fn-160.1 inventoried.

**Size:** L
**Files:** `model/framework/IrFile.scala`, the lifter's IR-file entry point in `model/irgen`, `model/check` (gate), the `IrFiles` test, each feature file's `exports` object, `model/README.md`, `model/ir/*`, `model/cases/*`, the pinned fixture trees, and the Go tests and harnesses fn-160.1 lists
**Touches:** [model/framework/IrFile.scala, model/irgen/*.scala, model/check/*.scala, model/temporal/IrFiles.test.scala, model/temporal/features/**/*.scala, model/README.md, model/ir/*, model/cases/*, tests/testcore/testpilot/testdata/generated/*, tools/canary/casebinding/testdata/*, tools/umpire/**, tools/canary/**]

### Approach
- **Collection:** the lifter treats each feature folder (a folder with a feature file) as one IR file named after the feature, e.g. `activity-standalone`. Its roots are every machine object, composition, `queries` and `capabilities` section, progress claim and realization declared in the feature's package and sub-packages, plus what they reach.
- **Findings from fn-160.1 (spec, Resolved via Codebase) to apply here:**
  - **No split has to stay.** No checker relies on a file holding only some Queries; the race file's comment is historical.
  - **Foundations and actors (owner, option a):** the task queue, the worker and the kind folders produce their own IR file when they declare Queries or realizations (R10). The task queue's 25 Queries move out of `activity-standalone-record` into e.g. `taskqueue`. A feature lifts the foundation machines it reaches but not their Queries.
  - **Exploration (R11):** check only the asked Query, and stop hashing the whole IR file into candidate ids (`tools/umpire/explore/explore.go` around 95-127, `tools/umpire/lower/lower.go` around 104-116). Otherwise merged files multiply candidate work (e.g. `nexusControl` from 1 Query to about 180) and churn ids.
  - **Go tests that pin a whole file's results** filter by machine: `check/activity_system_test.go`, `replay_test.go`, `nexus_close*_test.go`, the export composition test.
  - **Lint acceptances:** merge the per-file `.lint.json` files and deduplicate the 314 shared activity acceptances (identical reasons). Recheck findings judged across the whole file.
  - **Names and leftovers:** name files `<kind>-<form>`, so the main files keep their names and the canary tree is unchanged. Delete the old `.json` and `.lint.json` files in the same change (the gate's settle refuses leftovers). The functional tree changes one pin (the control Case), and the pinned control Run (`Makefile` around 580) must be re-recorded live.
  - **Quint export:** it has never been given the race, control or nexus-standalone machines. Check that the merged slices open, and record the result.
- **Opt-out:** an explicit marker keeps a declaration out of `model/ir`. Report each opted-out declaration. Use it only where fn-160.1 found a reason.
- **Remove** `irFile` and every `exports` object. Update the feature-file layout rule (exports was step 5 of the reading order) and the README.
- **Gate:** lift per feature, settle `model/ir`, accept waivers, and have the `IrFiles` test construct each feature package's declarations.
- **Regenerate** the IR and Cases, and merge the per-file `.lint.json` acceptances. Check that every machine, Query, Case and claim is still present with the same behaviour, and that the only changes are file boundaries and names. Explain any other difference before accepting it.
- **Migrate** the Go tests, canary preflight, pinned fixture trees, and export README tables named in fn-160.1's inventory.

### Prerequisite
- fn-160.1's inventory, with any split the user decided to keep.
- Batch 7's shared close (see MILESTONES.md): the collection tooling lands with the batch's tooling; this task closes the batch with the single regeneration of IR, Cases, lint acceptances and fixture trees, the equivalence seal against the batch's rename map, the full gate, review and live run.
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
