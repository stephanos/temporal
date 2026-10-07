---
satisfies: [R1, R2, R7]
---
# fn-148-consolidate-testpilot-evidence-and.1 Generalize evidence declarations and extract evidence.proto

## Description
Define the one declaration shape for R1, R2 and R7 before either existing lift representation is removed. Move evidence identity, value and lifting declarations to a leaf schema while preserving the current inline and referenced forms during this task.

**Size:** M
**Files:** Makefile protocol inputs, protocol tests, generalized evidence binder and fixture harness, `proto/internal/temporal/server/api/testpilot/v1/program.proto`, `correlated.proto`, new `evidence.proto`, generated APIs and protocol tests
**Touches:** [Makefile, common/testing/testpilot/protocol_test.go, common/testing/testpilot/internal/execution/evidence.go, common/testing/testpilot/internal/execution/dataflow.go, common/testing/testpilot/internal/execution/*evidence*_test.go, api/testpilot/v1/*.go, proto/internal/temporal/server/api/testpilot/v1/program.proto, proto/internal/temporal/server/api/testpilot/v1/correlated.proto, proto/internal/temporal/server/api/testpilot/v1/evidence.proto, api/testpilot/v1/*evidence*.go, common/testing/testpilot/protocol_test.go]

### Approach
- Resolve the remaining `ModelValue` declaration owner after CEL: move it beside correlated consumers only if evidence imports remain acyclic; otherwise retain the neutral model-atom leaf and record the ownership reason. Never merge it into runtime CEL values.
- Implement generalized declaration binding for dynamic scopes, admitted guards and constant fields in `internal/execution/evidence.go`, reusing scope handling from `dataflow.go`. Production caller conversion remains Task 2.
- Add a before/after fixture harness comparing every supported inline/reference lift against its generalized declaration: dynamic scopes, constants, ordered overlap and absent paths. The passing fixture projection is required before Task 2 retires inline fields.
- Add the evidence schema to Makefile's protocol documentation inputs and `protocolFiles`, and regenerate all affected Program/correlated APIs.
- Give declarations source identity, projected schema, operation key, scope and field expressions, guards and constants.
- Represent response lifting as an ordered declaration-ID list plus its Observation destination.
- Keep source-specific selection policy outside the common declaration shape.

### Investigation targets
**Required** (read before coding):
- `proto/internal/temporal/server/api/testpilot/v1/program.proto:1-120` - current evidence declarations
- `proto/internal/temporal/server/api/testpilot/v1/correlated.proto:1-120` - inline lift and evidence identities
- `common/testing/testpilot/internal/execution/evidence_lift_test.go:1-240` - supported inline lift surface
- `common/testing/testpilot/internal/execution/evidence_source_test.go:1-180` - source and ambiguity policies
- `.plans/TESTPILOT_SCHEMA_RESEARCH.md:40-48` - target declaration and retained policies


### Quick commands

```bash
go test -tags test_dep -run 'Test.*(Evidence|Correlated|Contract|Instruction|ReadSource|Protocol)' ./common/testing/testpilot/...
```
## Acceptance
- [ ] R1's complete declaration vocabulary is generated and descriptor-checked; one lift mixing inline data with declaration IDs remains invalid.
- [ ] R2's ordered and ambiguous selection policies stay representable.
- [ ] R7's evidence ownership produces an acyclic import graph.
- [ ] Protocol and evidence schema tests pass.
- [ ] Executable before/after lift fixtures pass for every supported inline/reference capability before Task 2 rollout.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
