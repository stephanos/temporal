---
satisfies: [R3, R4, R5, R6]
---
# fn-145-modularize-the-umpire-ir-schema.4 Prove artifact identity and document schema ownership

## Description
Run the full equivalence harness once, classify every expected descriptor delta, and update the rules of record for R3 through R5. Close the schema-modularization milestone only after generated artifacts and identities match their declared contracts.

**Size:** M
**Files:** `model/README.md`, `model/SEMANTICS.md`, `.plans/UMPIRE_MODULES.md`, `MILESTONES.md`, generated IR/Cases and compatibility fixtures
**Touches:** [model/README.md, model/SEMANTICS.md, .plans/UMPIRE_MODULES.md, MILESTONES.md, model/ir/**, model/cases/**, tests/testcore/testpilot/testdata/generated/**, tools/canary/casebinding/testdata/**]

### Approach
- Regenerate once after Tasks 2 and 3, then compare reader projections, Query answers, Case bytes and identities against the captured baseline.
- Classify file-descriptor and standard-Empty changes without absorbing unrelated Model or Case deltas.
- Document the nine owners, closure-generation rule and unchanged Umpire/Testpilot boundary.

### Investigation targets
**Required** (read before coding):
- `MILESTONES.md:19-49` - full-gate and regeneration discipline
- `model/README.md:117-165` - two-IR pipeline and schema ownership
- `.plans/UMPIRE_MODULES.md:20-55` - Umpire IR import boundary
- `tools/umpire/ir/schema_test.go:125-295` - compatibility evidence
- `common/testing/testpilot/recordedrun/recordedrun.go:40-90` - Case identity pairing


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

### Shared-leaf evaluation

Evaluate `RequiredSetting` and disposition/cleanup duplication against a concrete maintenance failure. Record retain-or-propose with an ownership rationale, including fn-144's setting relations, origins and scoped bindings. A new shared schema requires an explicit ownership decision (R6).

## Acceptance
- [ ] R3's final equivalence evidence covers Models, Queries, Cases and identities.
- [ ] R4's marker delta is the only semantic API change classified.
- [ ] R5's rules of record and milestone entry match the shipped layout.
- [ ] Full required gates pass with reusable evidence recorded for review.
- [ ] R6's shared-leaf evaluation records evidence and an ownership recommendation without adding a schema dependency.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
