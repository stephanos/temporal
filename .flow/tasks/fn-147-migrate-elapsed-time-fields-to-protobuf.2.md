---
satisfies: [R1, R2, R4]
---
# fn-147-migrate-elapsed-time-fields-to-protobuf.2 Replace elapsed-time schemas and migrate Umpire producers

## Description
Replace integer elapsed-time fields directly and migrate Scala authoring, lifting and Umpire lowering. The tree may be temporarily red until Task 3 updates consumers; no parallel old fields or compatibility branches are introduced.

**Size:** M
**Files:** Testpilot and Umpire protobuf schemas and generated APIs, Scala realization vocabulary and lifter, Go realization admission and lowering
**Touches:** [proto/internal/temporal/server/api/testpilot/v1/*.proto, proto/internal/temporal/server/api/umpire/v1/*.proto, api/testpilot/v1/*.go, api/umpire/v1/*.go, model/temporal/realize/**, model/irgen/**, tools/umpire/realization/**, tools/umpire/lower/**]

### Approach
- Replace elapsed-time fields with Duration and update the schema ledger; reserve retired field coordinates only where new numbers are needed.
- Update Scala and Go producers plus `tools/umpire/realization` source admission (`GetTimeoutMs`, `GetIntervalMs`, `GetAtMostMs`, `GetDeadlineMs`) with the checked conversion contract. Preserve source defaults, timer hints and derived polling rules.
- Defer consumer-dependent realization and lowerer tests until Task 3 restores compilation, and managed-artifact tests until Task 4 regenerates. This task may finish with a documented red consumer tree; no compatibility fields are added to make intermediate gates green.
- Keep runtime polling policy and scalar-presence cleanup in Task 3.

### Investigation targets
**Required** (read before coding):
- `proto/internal/temporal/server/api/umpire/v1/ir.proto:783-847` - realization time hints
- `proto/internal/temporal/server/api/umpire/v1/ir.proto:1105-1200` - command timeout and read interval
- `common/testing/testpilot/internal/execution/dataflow.go:245-329` - node bounds and wait hints
- `common/testing/testpilot/internal/execution/evidence.go:292-350` - read policy admission
- `tools/umpire/lower/realization.go` - Umpire-to-Case conversion
- `model/temporal/realize/Behavior.scala` - authored behavior hints

### Key context
Use the post-fn-145 split file paths when implementation starts. Old formats are unsupported; all checked-in artifacts migrate under the current schema.


### Quick commands

```bash
make proto
go test -tags test_dep ./api/umpire/v1 ./api/testpilot/v1
```

The schema stage replaces old definitions directly and retains no compatibility branches. Runtime policy and singleton presence switching belong to Task 3.
## Acceptance
- [ ] Schema and generated APIs use Duration directly; superseded integer fields are removed.
- [ ] Scala authoring, lifting, realization admission and Go lowering use checked field-specific conversions.
- [ ] Counts, logical bounds and timestamps remain unchanged.
- [ ] Schema generation and generated-API checks pass; consumer-dependent tests are explicitly deferred to Task 3 and artifact-dependent gates to Task 4.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
