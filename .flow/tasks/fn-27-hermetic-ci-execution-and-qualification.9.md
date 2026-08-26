---
satisfies: [R1, R2, R3, R4, R5, R6, R7, R8]
---
# fn-27-hermetic-ci-execution-and-qualification.9 Complete the CI regression matrix and synchronize documentation

## Description
Complete R1-R8 with aggregate regressions and accurately scoped contributor/roadmap documentation.

**Size:** M
**Files:** `model/Umpire/Qualification/**`, `model/Temporal/System/Execution/**`, `model/Temporal/System/Qualification/**`, `model/Temporal/Tool/ConformanceTests.lean`, `tools/umpire/artifact/**`, `tools/umpire/runtime/**`, `tools/umpire/conformance/**`, `tools/umpire/qualification/**`, `tools/umpire/ci/**`, `tools/umpire/cmd/umpire-qualify-ci/**`, `model/README.md`, `model/ARCHITECTURE.md`, `.plans/UMPIRE4_COMPONENTS.md`
**Touches:** [model/Umpire/Qualification/**, model/Temporal/System/Execution/**, model/Temporal/System/Qualification/**, model/Temporal/Tool/ConformanceTests.lean, tools/umpire/artifact/**, tools/umpire/runtime/**, tools/umpire/conformance/**, tools/umpire/qualification/**, tools/umpire/ci/**, tools/umpire/cmd/umpire-qualify-ci/**, model/README.md, model/ARCHITECTURE.md, .plans/UMPIRE4_COMPONENTS.md]

### Approach

- Complete independent exact-limit/one-over, version crossing, identity/binding/status/reason/omission/provenance, workflow-policy, accepted input-below-workspace, rejected output/workspace crossing, cancellation, and cleanup matrices not already owned by focused tasks.
- Re-run local v1/v2 byte fixtures, local direct/root behavior, aggregate Lean/Go tests, artifact publication recovery, and the stable regression gate after the CI extension.
- Document the CI-only production invocation, input/output, hermetic execution boundary, two-stage isolation, statuses, retention/redaction, self-reported trust, exact claim, and non-release limitation.
- Update C12 roadmap status only after all evidence passes; preserve DSL/vision/generated projections and keep remote, canary, authenticated provenance, and release aggregation separate.

### Investigation targets

**Required** (read before coding):
- `.plans/UMPIRE4_COMPONENTS.md` — C12 and Active Flow status ownership
- `model/README.md` — current execution/conformance contributor commands
- `model/ARCHITECTURE.md` — current runtime-to-Result boundary
- `.flow/tasks/fn-20-local-execution-semantic-conformance.7.md` — documentation/live-proof precedent
- `.flow/tasks/fn-26-local-qualification-receipts-and-staged.6.md` — staged qualification wording and regression boundary
- Task `.7` — bounded CI proof evidence to summarize

### Acceptance

- [ ] Every exact-limit/one-over, version, binding, status, reason, omission, provenance, workflow, and isolation mutation fails at the intended boundary.
- [ ] Local schemas/commands remain byte-compatible; focused, aggregate, recovery, and stable regression checks pass.
- [ ] Docs give only the production CI path and accurately state self-reported trust, hermetic boundary, retention/redaction, and non-release scope.
- [ ] Roadmap claims only the implemented CI profile and leaves later C12 profiles explicit; generated projections remain unchanged.

## Acceptance
- [ ] R1-R8 aggregate regression and scoped documentation obligations are complete.
- [ ] All focused, aggregate, local-regression, publication-recovery, and stable regression checks pass.
- [ ] Existing comments and generated projections remain preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
