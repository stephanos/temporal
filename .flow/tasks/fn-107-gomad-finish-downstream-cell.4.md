---
satisfies: [R1, R6, R9, R11]
---
# fn-107-gomad-finish-downstream-cell.4 Run a real Walker-backed workflow through the dedicated downstream smoke harness

## Description
Work in ../saas-temporal/localcell/gomad. Reuse actual Walker localcluster and Temporal testcore.NewTestClusterFactory().NewCluster with AdditionalServerOptions and in-memory SQLite for explicitly named auxiliary stores. Same SQLite backing must seed and serve namespace/cluster state. Workflow completion/result/history and real Walker storage effects are required; prove disabling/denying Walker storage fails the workload. Disable unrelated canary/Omes/archival/Nexus workloads. Configure schemas/input mounts and root/nested module replacements coherently for GOWORK=off without accidental go.sum download mutations. Files/Touches: localcell/gomad/**, root/nested go.mod and necessary integration option seams. Quick: native dedicated test plus Gomad analyze closure/linked using the final target, test_dep/gomad/integration tags and exported private-module settings.

## Acceptance
Native smoke exercises real Walker execution and history via CDS wrapper/controller and completes a workflow with verified result/history. A negative test proves Walker cannot be bypassed. Deterministic profile/substitutions and exact preparation inputs are explicit; no live external dependency is required.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
