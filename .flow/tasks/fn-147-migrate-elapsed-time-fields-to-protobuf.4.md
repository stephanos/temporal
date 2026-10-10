---
satisfies: [R4, R5]
---
# fn-147-migrate-elapsed-time-fields-to-protobuf.4 Regenerate artifacts and document Duration contracts

## Description
Classify identity changes from the Duration fields and presence cleanup and update the rules of record for R4 and R5. Regeneration of affected Models, Cases and Run companions, full gates, review and live-run evidence occur once at fn-148.7; fn-147 does not close before that evidence is recorded.

**Size:** M
**Files:** generated Model IR and Cases, functional and canary fixtures, `model/README.md`, `model/SEMANTICS.md`, `.plans/UMPIRE_MODULES.md`
**Touches:** [model/ir/**, model/cases/**, tests/testcore/testpilot/testdata/generated/**, tools/canary/casebinding/testdata/**, model/README.md, model/SEMANTICS.md, .plans/UMPIRE_MODULES.md]

### Approach
- Hand the milestone summary and shared-close evidence to the conductor. Only the conductor edits `MILESTONES.md` and completes its required update at the shared fn-148.7 close; workers do not edit that file.
- Prepare the Duration identity-delta ledger against the retained pre-migration baseline, accounting only for new fields, reserved predecessors and derived identities; compare generated artifacts at fn-148.7.
- Inventory affected Model, Case and Run companions and link their single format 4.0 regeneration/replay evidence from fn-148.7.
- Document units, defaults, precision, presence and monotonic elapsed semantics.

### Investigation targets
**Required** (read before coding):
- `MILESTONES.md:19-49` - regeneration and full-gate discipline
- `model/README.md:117-165` - Umpire to Testpilot artifact pipeline
- `model/SEMANTICS.md` - realization and generated Case semantics
- `common/testing/testpilot/recordedrun/recordedrun.go:40-90` - current pairing
- `.plans/UMPIRE_MODULES.md` - Module ownership and Exact artifacts and fixture compatibility (the Archive and intermediate-state rules are historical)


### Shared-close commands (run once at fn-148.7)

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
- [ ] R4 cross-layer behavior passes on generated and hand-authored fixtures, with shared fn-148.7 evidence linked before spec closure.
- [ ] R5's identity-delta ledger and docs are complete; final identity accounting and milestone/spec closure use fn-148.7's shared evidence.
- [ ] No uncategorized Model, Case, Run or identity delta remains.
- [ ] Required full gates, review and live run pass at fn-148.7 with reusable evidence linked here before spec closure; this task does not independently regenerate or close the spec.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
