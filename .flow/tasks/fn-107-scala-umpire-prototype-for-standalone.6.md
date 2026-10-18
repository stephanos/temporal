---
satisfies: [R2, R6, R7, R8]
---
# fn-107-scala-umpire-prototype-for-standalone.6 Lower Scala scenarios and observations through Testpilot

Touches: [model/scalav2/scala/umpire/caseproducer/**, model/scalav2/scala/temporal/nexuscaller/Realization.scala, model/scalav2/scala/temporal/test/CaseBytes.test.scala, model/scalav2/scala/umpire/**, model/scalav2/lifter/**, model/scalav2/ir/**, model/scalav2/goir/testpilot/**, model/scalav2/goir/load.go, model/scalav2/goir/*_test.go, proto/internal/temporal/server/api/modelir/v1/**, api/modelir/v1/**, model/scalav2/SEMANTICS.md, model/scalav2/gen.sh, model/scalav2/README.md, model/scalav2/run.sh, Makefile, proto/internal/temporal/server/api/testpilot/v1/**, api/testpilot/v1/**, model/scalav2/gen.sh]

## Description
Add the producer adapter from admitted Scala IR scenarios to existing Testpilot Cases. Identify narrowly necessary protocol changes explicitly.

Cases are produced in Go from the IR, in `goir/testpilot`. Scala declares the scenario, observations, learned values and worker script; it does not build Cases. `model/scalav2/scala/umpire/caseproducer` is a Scala-side Case producer that task 16 carried over from the legacy `model/scala` experiment; only `nexuscaller/Realization.scala` and `CaseBytes.test.scala` use it, and nothing lifts it. It is a second producer beside the Go one, which the spec rules out, so this task retires it.

**Size:** M
**Files:** lifter scenario support; goir/testpilot lowering and admission fixtures; the Scala realization declarations, re-expressed as lifted declarations; removal of model/scalav2/scala/umpire/caseproducer and the v2 Testpilot proto jar; Testpilot protocol/generated outputs only when required.

### Approach
- Reuse the current Case/Program/Contract abstraction and producer-shaped declarations. Import relevant existing descriptors; generate source annotations for field identity/projection roles.
- Lower DAG dependencies, typed learned values, scripts, observations, and monitors mechanically. Reject unsupported lowering rather than synthesize feature semantics.
- Inventory every Testpilot gap from the reviewed sketches. Use existing primitives where possible; document and test the smallest schema/admission evolution where necessary.
- Exercise ordinary Prepare with existing fixtures and Cases lowered from the Scala IR; retain existing consumer compatibility and canonical artifact identity.
- Retire the Scala Case producer: delete `scala/umpire/caseproducer`, the producer calls in `Realization.scala`, `CaseBytes.test.scala`, and the `testpilot` jar from `gen.sh`, the Makefile's Scala jar prerequisites and `run.sh`. Byte parity with the existing `nexusCallerTests-*-case.json` fixtures is not required (owner decision, 2026-09-30): a Case lowered from the IR must be admitted by ordinary Prepare and carry the behaviour the Scala declares, not match those bytes. Reuse `model/go/caseproducer` where it fits rather than duplicating it.

### Investigation targets
**Required:** model/scalav2/scala/umpire/caseproducer/Producer.scala; model/scalav2/scala/umpire/caseproducer/Program.scala; model/scalav2/scala/temporal/nexuscaller/Realization.scala; model/go/caseproducer/; model/go/nexuscaller/realization.go and case_test.go; common/testing/testpilot/prepare.go:27; common/testing/testpilot/profile.go:84; proto/internal/temporal/server/api/testpilot/v1/program.proto:131.
**Optional:** common/testing/testpilot/internal/ir/descriptor.go; tests/testcore/testpilot/protobuf_lean_authoring_test.go.

### Quick commands
`make umpire-check-scala`; `make lint-scala`; `mise exec -- go test -tags test_dep ./model/scalav2/... ./common/testing/testpilot/... ./tests/testcore/testpilot/...`; run existing protocol generation gates and keep `model/scalav2/gen.sh` building only the jars that remain.

## Acceptance
- [ ] An admitted Scala scenario produces a Testpilot Case with typed learned values and independent branches.
- [ ] Valid public/internal descriptor projections preserve presence/validation, while crossed types/IDs/descriptors and cycles reject.
- [ ] Existing Testpilot consumers admit and evaluate after any documented narrow protocol tweak.
- [ ] Generated monitor/script behavior is traceable to Scala and no feature policy is introduced in Go.
- [ ] Cases are built only in Go from the IR, and `model/scalav2/scala/umpire/caseproducer`, its byte-parity test and the v2 Testpilot proto jar are gone. No test requires byte parity with the existing Nexus caller Case fixtures.

- [ ] Legacy model/scala files remain unchanged; v2 generation, lifting, native tests and lint keep passing with that tree absent.

## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: