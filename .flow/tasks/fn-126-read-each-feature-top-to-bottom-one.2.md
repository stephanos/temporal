---
satisfies: [R1, R2, R3, R5, R9, R10, R13]
---
# fn-126-read-each-feature-top-to-bottom-one.2 Write the Nexus folders and shared Models as feature files, share the bounds, and retire the per-kind file names

## Description
Convert the remaining folders to feature files with today's declaration forms: `features/nexuscaller` (with the control Model), `nexuscaller/closepolicy`, `features/nexusoperation`, `shared/taskqueue` and `shared/worker` (R1-R3, R5). Add the shared bounds source (R13, bounds half). Make the layout test refuse the retired file names (R10). Bring the docs to the new layout (R9, layout half).

**Cross-spec entry gate:**
- Task 1 is done.
- Never alongside fn-124.8.
- Before fn-124.7.

**Size:** L
**Files:**
- `model/temporal/features/{nexuscaller,nexuscaller/closepolicy,nexusoperation}/**`, `model/temporal/shared/{taskqueue,worker}/**`;
- `model/temporal/shared/Bounds.scala` (new);
- `tools/umpire/model/layout_test.go` (or its fn-124.8 home);
- Go tests that assert Scala positions (`tools/umpire/lower/activity_test.go:742` and others found by grep);
- `tools/umpire/internal/golden/config.json`;
- `model/README.md`, `model/SEMANTICS.md`, `.plans/UMPIRE_MODULES.md`, `.plans/UMPIRE4_VISION.md`, `AGENTS.md`.

**Touches:** [model/temporal/features/**, model/temporal/shared/**, tools/umpire/model/layout_test.go, tools/umpire/lower/**, tools/umpire/internal/golden/**, model/ir/**, model/cases/**, model/README.md, model/SEMANTICS.md, .plans/UMPIRE_MODULES.md, .plans/UMPIRE4_VISION.md, AGENTS.md]

### Approach
- Follow task 1's pattern folder by folder.
- `shared/taskqueue` keeps borrowing the record's pin (`…System$package$`) and family.
- The close policy gets one module object for `rejectAfterClose` and its nine derivations here. Task 5 splits them into `Derived` objects.
- Bounds: move every `Limits` that two folders declare (today `three` ×3, `five`, `twelve`) to `shared/Bounds.scala`. Names stay, and the IR changes only in positions.
- R10: extend the layout test with the five retired file names (one case each proves detection) and with live files that name such a path. Fix the Go tests that assert old Scala paths.
- Docs (R9, layout): describe the feature-file layout and R2's order, with the activity as the example. Remove every reference to the per-kind files and to "its own `Capabilities.scala`". Task 5 updates the declaration shape, and task 6 the names.

### Investigation targets
**Required:**
- the per-kind files of each folder above
- `tools/umpire/model/layout_test.go`
- `model/README.md` "Writing a Model" and "Where things are"
**Optional:**
- `.plans/UMPIRE_MODULES.md:30,321,484`

### Quick commands
```bash
make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks && make lint-model
go test -count=1 -tags test_dep -p 2 ./tools/umpire/...
grep -rn "Model.scala\|Properties.scala\|Queries.scala\|IrFiles.scala" model/README.md model/SEMANTICS.md .plans/UMPIRE_MODULES.md .plans/UMPIRE4_VISION.md AGENTS.md
```

### Execution constraints
- R5 holds as in task 1. The R4 lint passes on every converted folder.

## Acceptance
- [ ] Every Model folder holds one feature file in R2's order. 30 per-kind files have become 8 feature files.
- [ ] Shared bounds come from `model/temporal/shared/Bounds.scala`. No bound is declared twice, and Limits names are unchanged.
- [ ] The layout test fails on each of the five retired file names (one case each) and on a live file that names one.
- [ ] The R9 docs describe the feature-file layout and name no retired file.
- [ ] R5 holds, with the deltas recorded. All gates of the spec's Verification pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
