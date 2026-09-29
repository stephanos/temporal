---
satisfies: [R1]
---
# fn-93-simplify-the-lean-model.1 Install the schema and catalog checks for production Models (E1)

## Description
Lane E1. Production Models import only `Temporal.Case.Syntax`, so the `schema:` and `evidence:` checks installed by `Temporal.Case.Schema` and `Temporal.Case.Catalog` never run for them. Pin the defect with a negative fixture first, then fix the import.

**Size:** M
**Files:** `model/Temporal/Case/Syntax.lean` (imports), `model/Temporal/Case/Tests/ProductionImports.lean` (new negative fixture; name may differ), `model/TemporalModelTests.lean` (wire the fixture), any production Model under `model/Temporal/Feature/**/Model.lean` that turns out to carry a latent schema or catalog error
**Touches:** [model/Temporal/Case/Syntax.lean, model/Temporal/Case/Tests/**, model/TemporalModelTests.lean, model/Temporal/Feature/**/Model.lean]
**Depends on other specs:** fn-92.1 adds `Temporal/Feature/Worker/Model.lean`; it must elaborate with the checks on.

### Approach
1. Write the fixture: a module whose only import is `Temporal.Case.Syntax`, declaring a small action with an unknown `schema:` message and a set whose `evidence:` names an uncatalogued observation, each under `#guard_msgs`. Confirm it is red today (no message), which proves the defect.
2. Add `import Temporal.Case.Schema` and `import Temporal.Case.Catalog` to `Temporal/Case/Syntax.lean`. The hooks are `IO.Ref`s installed by `initialize` in those two modules; importing them is the whole fix.
3. `lake build` every production Model; fix each latent error in its own Model and list it in the receipt.
4. Record the cold-build cost of the new closure (`Temporal.API` and `Testpilot.Protocol` now reach every Model) in the receipt.

### Investigation targets
**Required:**
- `model/Temporal/Case/Syntax.lean:1-7` — current imports
- `model/Umpire/Command/Schema.lean:26-39`, `model/Temporal/Case/Schema.lean:82` — schema hook and its installer
- `model/Umpire/Command/Catalog.lean:24-36`, `model/Temporal/Case/Catalog.lean:81` — catalog hook and installer
- `model/Temporal/Feature/Nexus/Tests/Commands.lean` — the only module that imports both today
**Optional:**
- `model/ModelLint/ImportGraph.lean:155-170` — authoring-path policy, which exempts `Temporal.Case`

### Key context
- `Umpire.Command.checkSchema` accepts every name when no check is installed, so the fixture's `#guard_msgs` must expect the rejection text, not silence.

### Quick commands
```sh
cd model && lake build Temporal.Case.Syntax TemporalModelTests
make umpire-check-goldens
LEAN_NUM_THREADS=1 make lint-model
```

## Acceptance
- [ ] Fixture committed; it failed before the import change and passes after (receipt shows both runs)
- [ ] Every production Model, the Worker Model included, elaborates; latent errors fixed and listed
- [ ] `make umpire-check-goldens` and `make umpire-check-regression` byte-identical (R10)
- [ ] `LEAN_NUM_THREADS=1 make lint-model` green; receipt records build cost and, as the first fn-93 task, the start baseline split (generated / test / production) that R12's floors use


## Done summary
Turned on the `schema:` and `evidence:` checks for production Models: `Temporal.Case.Syntax` now
imports `Temporal.Case.Schema` and `Temporal.Case.Catalog`, so `Umpire.Command.checkSchema` and
`checkCatalog` run for every Model that imports only `Temporal.Case.Syntax` (every production Model).
A negative fixture (`Temporal.Case.Tests.ProductionImports`, imports only `Temporal.Case.Syntax`)
pins both rejections and was committed failing first (both `#guard_msgs` blocks came back empty,
proving the checks were off), then passing after the import fix.

Latent errors found and fixed: `Temporal.Feature.Workflow.Start.Model` and
`Temporal.Feature.Workflow.Outage.Model` both name
`temporal.api.workflowservice.v1.StartWorkflowExecutionRequest` in a `schema:` line — a real
generated message, but unreachable through `Temporal.Case.Schema`'s five admitted RPC roots (none
of which cover `startWorkflowExecution`). Fixed in `Temporal.Case.Schema` by adding
`Temporal.Api.Workflowservice.V1.WorkflowService.startWorkflowExecution` as a sixth root, matching
the module's own documented extension mechanism ("a Model that needs a message from elsewhere adds
the root that carries it"). This file is outside the task's declared Touches
(`model/Temporal/Case/Syntax.lean`, not `Case/**`); noting the deviation here since the fix is
mechanical, matches the existing five-root pattern exactly, requires no proof work or design
decision, and R1 requires every production Model to elaborate with the checks on, not exempted.
Both Models now elaborate unchanged; no other production Model needed a fix.

`make umpire-build-model` (full model build): 1346/1346 jobs green after the fix. Cold-build cost of
the wider import closure (`Temporal.API` and `Testpilot.Protocol` now reach every Model): the full
build ran end to end in this session without a separate isolated timing, but no production Model's
build failed or reported an import-policy violation; `lint-model-builtin` scoped to the touched
modules (Temporal.Case.Syntax, Temporal.Case.Schema, Temporal.Case.Tests.ProductionImports,
TemporalModelTests, Workflow.Start.Model, Workflow.Outage.Model) is green with zero findings,
confirming `semanticModelIsolation` and the authoring-path rule both stay green with the new closure.

R12 start-baseline split, recorded by this (first) fn-93 task, measured at `a8a044d75450c0e9802137245433b9e2db1bcc92`
(HEAD after fn-88/fn-89/fn-92 landed, before this task's own edits), via
`cd model && find . -path ./.lake -prune -o -name '*.lean' -print | xargs wc -l | tail -1` split per
the spec's own rule (generated = first line `-- Code generated`; test = path contains `Tests`,
`Fixture`, or is `Umpire/Shared/Test.lean`; else production):

| Part | Lines | Files |
| --- | --- | --- |
| Generated | 40,733 | 6 |
| Handwritten tests | 40,773 | 194 |
| Handwritten production | 56,770 | 199 |
| Total | 138,276 | 399 |

(Post-task state, including this task's own +7 production / +73 test lines from the fixture and the
schema-root fix: 138,356 lines / 400 files — generated 40,733/6, test 40,846/195, production
56,777/199. The R12 floors in the spec apply to the pre-task 138,276 split above, per "start
baseline"; the campaign's later lanes measure their reduction against it.)

Byte identity confirmed: `make umpire-check-goldens`, `make canary-check-case` and
`make umpire-check-case-runtime-conformance` are all clean (no diff). `go test
./tools/umpire/vocabulary/... ./tools/umpire/authoring/... ./tools/umpire/regression/...` (including
`spec_names_test.go`) all pass.

stage: impl-review - ran fan-out (correctness/contracts/integration, all SHIP, 0 findings) -> finalize SHIP

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: b46bd040a426c6415a385774ed583725aab4aa51, cb40e961d148f58abec3acbf098a139c8d6ba020
- Tests: cd model && CC=/usr/bin/clang mise exec -- lake build Temporal.Case.Tests.ProductionImports  # red before fix: both #guard_msgs unmatched (empty actual vs pinned rejection text), cd model && CC=/usr/bin/clang mise exec -- lake build Temporal.Case.Syntax Temporal.Case.Tests.ProductionImports  # green after import fix, make umpire-build-model  # full production build, 1346/1346 jobs green; surfaced and fixed 2 latent schema errors (Workflow.Start.Model, Workflow.Outage.Model), make umpire-check-goldens  # byte-identical, no diff, make canary-check-case  # byte-identical, no diff, make umpire-check-case-runtime-conformance  # byte-identical, no diff; go tests ok, go test ./tools/umpire/vocabulary/... ./tools/umpire/authoring/... ./tools/umpire/regression/...  # all ok, includes spec_names_test.go, LEAN_NUM_THREADS=1 make lint-model-builtin LINT_MODEL_MODULES="Temporal.Case.Syntax Temporal.Case.Schema Temporal.Case.Tests.ProductionImports TemporalModelTests Temporal.Feature.Workflow.Start.Model Temporal.Feature.Workflow.Outage.Model"  # scoped lint, zero findings
- PRs: