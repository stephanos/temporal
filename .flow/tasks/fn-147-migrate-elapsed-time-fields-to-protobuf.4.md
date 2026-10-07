---
satisfies: [R4, R5]
---
# fn-147-migrate-elapsed-time-fields-to-protobuf.4 Regenerate artifacts and document Duration contracts

## Description
Regenerate the affected Models and Cases once, classify identity changes from the versioned fields and presence cleanup, and update the rules of record for R4 and R5.

**Size:** M
**Files:** generated Model IR and Cases, functional and canary fixtures, `model/README.md`, `model/SEMANTICS.md`, `.plans/UMPIRE_MODULES.md`, `MILESTONES.md`
**Touches:** [model/ir/**, model/cases/**, tests/testcore/testpilot/testdata/generated/**, tools/canary/casebinding/testdata/**, model/README.md, model/SEMANTICS.md, .plans/UMPIRE_MODULES.md, MILESTONES.md]

### Approach
- Compare generated artifacts against the pre-migration baseline and account only for new fields, reserved predecessors and derived identities.
- Regenerate checked-in Case and Run companions and replay the current format.
- Document units, defaults, precision, presence and monotonic elapsed semantics.

### Investigation targets
**Required** (read before coding):
- `MILESTONES.md:19-49` - regeneration and full-gate discipline
- `model/README.md:117-165` - Umpire to Testpilot artifact pipeline
- `model/SEMANTICS.md` - realization and generated Case semantics
- `common/testing/testpilot/recordedrun/recordedrun.go:40-90` - current pairing
- `.plans/UMPIRE_MODULES.md:400-450` - artifact and runtime ownership


### Quick commands

```bash
make umpire-gen-model
make umpire-gen-fixtures
make canary-gen-case
make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks
go test -tags test_dep -p 2 -timeout 30m ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/...
make umpire-check-cases
make umpire-check-fixtures
make canary-check-case
make lint-model
make lint-code-fast
```

## Acceptance
- [ ] R4 cross-layer behavior passes on generated and hand-authored fixtures.
- [ ] R5 identity accounting, docs and milestone close are complete.
- [ ] No uncategorized Model, Case, Run or identity delta remains.
- [ ] Required full gates pass with reusable evidence recorded.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
