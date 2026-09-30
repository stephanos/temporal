---
satisfies: [R2, R6, R7, R8]
---
# fn-107-scala-umpire-prototype-for-standalone.6 Lower Scala scenarios and observations through Testpilot

Touches: [model/scala/umpire/caseproducer/**, model/scalav2/lifter/**, model/scalav2/goir/testpilot/**, proto/internal/temporal/server/api/testpilot/v1/**, api/testpilot/v1/**]

## Description
Add the producer adapter from admitted Scala IR scenarios to existing Testpilot Cases. Identify narrowly necessary protocol changes explicitly.

**Size:** M
**Files:** lifter scenario support; proposed goir/testpilot lowering and admission fixtures; existing caseproducer declarations; Testpilot protocol/generated outputs only when required.

### Approach
- Reuse the current Case/Program/Contract abstraction and producer-shaped declarations. Import relevant existing descriptors; generate source annotations for field identity/projection roles.
- Lower DAG dependencies, typed learned values, scripts, observations, and monitors mechanically. Reject unsupported lowering rather than synthesize feature semantics.
- Inventory every Testpilot gap from the reviewed sketches. Use existing primitives where possible; document and test the smallest schema/admission evolution where necessary.
- Exercise ordinary Prepare with existing and Scala-produced fixtures; retain existing consumer compatibility and canonical artifact identity.

### Investigation targets
**Required:** model/scala/umpire/caseproducer/Producer.scala; model/scala/umpire/caseproducer/Program.scala; common/testing/testpilot/prepare.go:27; common/testing/testpilot/profile.go:84; proto/internal/temporal/server/api/testpilot/v1/program.proto:131.
**Optional:** common/testing/testpilot/internal/ir/descriptor.go; tests/testcore/testpilot/protobuf_lean_authoring_test.go.

### Quick commands
`mise exec -- go test -tags test_dep ./model/scalav2/... ./common/testing/testpilot/... ./tests/testcore/testpilot/...`; run existing protocol generation gates if schema changes.

## Acceptance
- [ ] An admitted Scala scenario produces a Testpilot Case with typed learned values and independent branches.
- [ ] Valid public/internal descriptor projections preserve presence/validation, while crossed types/IDs/descriptors and cycles reject.
- [ ] Existing Testpilot consumers admit and evaluate after any documented narrow protocol tweak.
- [ ] Generated monitor/script behavior is traceable to Scala and no feature policy is introduced in Go.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
