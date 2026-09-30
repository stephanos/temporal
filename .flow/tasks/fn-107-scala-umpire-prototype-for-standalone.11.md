---
satisfies: [R1, R7, R8, R10]
---
# fn-107-scala-umpire-prototype-for-standalone.11 Close the exploration, regression, and trace-inspection loop

Touches: [model/scalav2/explore/**, model/scalav2/README.md, model/scalav2/SEMANTICS.md, tools/umpire/replay/**, tests/testpilot_nexus_control_case_test.go]

## Description
Connect Scala variation/reduction declarations to existing exploration/replay and produce the final bounded demonstration artifacts.

**Size:** M
**Files:** proposed IR exploration/replay adapter and trace renderer; existing replay bridge seam/tests; prototype README/SEMANTICS and fixtures.

### Approach
- Reuse generic exploration, rerun, minimization, and proposal algorithms. Scala selects parameter classes, priorities, and legal reductions.
- Discover an unpinned bounded execution. Use the existing forged-completion runtime control to demonstrate reproduction/minimization if the real activity implementation conforms.
- Preserve DAG dependencies, learned values, scripts, and failure identity while reducing. Emit byte-stable check-in-ready proposals with physical resources bound only at execution.
- Render one local trace linking product/system steps, monitors, fault/evidence/holes and source definitions. Combine documentation and final validation here.
- Run the Go-developer authoring exercise and record steps, feature source size, and diagnostic latency.

### Investigation targets
**Required:** tools/umpire/replay/core.go; tools/umpire/replay/minimize.go; tools/umpire/replay/bridge.go; tools/umpire/replay/proposal.go; tests/testpilot_nexus_control_case_test.go:30.
**Optional:** tests/testpilot_nexus_control_case_test.go:116; model/scalav2/README.md.

### Quick commands
`mise exec -- go test -tags test_dep ./model/scalav2/... ./tools/umpire/replay/...`; final gate includes `make umpire-check-scala`, `make lint-scala`, `make lint-code-fast`, and the selected live demonstrations.

## Acceptance
- [ ] Declared variation priorities discover an execution absent from pinned regressions with exact finite/sample coverage.
- [ ] A controlled runtime failure reproduces, minimizes without breaking dependencies, and produces a replayable proposal; unreproduced failures do not promote.
- [ ] Repeat generation is byte-identical and the trace artifact exposes both levels, monitor state, evidence, faults, and holes.
- [ ] The feature-only Scala authoring exercise and all targeted/final demonstration commands are recorded with honest support limits.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
