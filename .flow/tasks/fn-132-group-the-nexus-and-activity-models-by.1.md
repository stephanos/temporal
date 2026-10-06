---
satisfies: [R1]
---
# fn-132-group-the-nexus-and-activity-models-by.1 Nexus forms move under features/nexus/: workflow and standalone

## Description
**Size:** M
**Touches:** [model/temporal/features/nexuscaller/**, model/temporal/features/nexusoperation/**, model/temporal/features/nexus/**, model/ir/**, model/cases/**, tools/umpire/**, common/testing/testpilot/**, tools/canary/**, tests/*.go, tests/testcore/testpilot/**, Makefile, model/check/Gate.scala, model/check/test/Gate.test.scala, model/README.md, .plans/UMPIRE_MODULES.md, MILESTONES.md]

**Required investigation:** current Nexus trees and exports; `model/irgen/Structure.scala`; `tools/umpire/ir/layout_test.go`; `.flow/tmp/fn-126/tools/{run.sh,apply_map.py,compare.py,projtool/main.go}`; the separate Model-packaging commands in `Makefile` and `model/check/Gate.scala`.
Adapt the ignored proof helper's retired `tools/umpire/model` imports to the current ir/interp/check/realization owners before collecting comparable baselines. Preserve Nexus's root realization files and `system/TrustingCaller.scala`. Use the prerequisite's kind classifier, not a validation bypass.

The prerequisite proved that package-only kind headers emit no TASTy and expose a Bloop packaging failure, while the same sources package normally with `--server=false`. Verify the necessary normal-compiler mode at both canonical packaging call sites (the Makefile jar rule and the gate's own Models build), with a focused gate regression; investigate any other affected compiler invocation before changing it. Keep printed compiler-error handling strict, and do not add placeholder declarations or edit `Tools.orFail` to hide the failure.

Part A for Nexus. Move `features/nexuscaller` to `features/nexus/workflow` and `features/nexusoperation` to `features/nexus/standalone`, as folders and packages, with their `product/` and `system/` subfolders and the close policy as fn-126 left them. Feature files are named after their folder: `NexusCaller.scala` becomes `Workflow.scala`, `NexusOperation.scala` becomes `Standalone.scala`. Add `features/nexus/Nexus.scala` as the kind's general feature file (a header comment and package declaration; Part B fills it).

Rename the IR files to match (`nexus-caller` → `nexus-workflow`, `nexus-control` → its workflow-form name, `nexus-operation` → `nexus-standalone`, and the close-policy files); fix the exact names here and record them in the done summary. Update every reference to the old paths and IR file names: Go `tools/umpire` (layout test, tests naming Scala positions), Makefile targets, Case trees, canary, `model/README.md` and `.plans/UMPIRE_MODULES.md` path mentions.

No meaning change. Definition IDs change only by package path (fn-126 decision 23).
## Acceptance
- [ ] `features/nexus/{workflow,standalone}` exist with the spec's layout; `features/nexuscaller` and `features/nexusoperation` do not.
- [ ] A before/after projection with the path and IR-file-name map applied is identical: reader tables, Query answers, verdicts, fingerprints, lint findings, Case bytes.
- [ ] A path/package/export-aware old-name search finds no retired Model references outside history, archived plans and closed specs; live Temporal server component/flag names are not Model-path failures. The exact ledger includes every control/close-policy export and Case/canary reference, not only the two primary exports.
- [ ] Focused layout, reader/lifter, exact projection and regenerated Case/fixture checks pass. Share still-applicable baseline evidence; explicitly defer the broad full-model/Go/runtime/artifact/lint obligations to task 2's closing Part A batch, with no focused full-gate receipt.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
