---
satisfies: [R1, R2, R3]
---
# fn-150-bounded-liveness-across-composed.2 Represent and resolve scoped composition fairness

## Description
Represent and resolve scoped composition fairness. Advances R1, R2, R3 of the parent spec.

**Size:** M
**Files:** `proto/internal/temporal/server/api/umpire/v1/**`, `api/umpire/v1/**`, `model/check/**`, `tools/umpire/ir/validate.go`, `tools/umpire/ir/*test.go`, `tools/umpire/check/claims.go`, `tools/umpire/check/composed_progress_test.go`, `tools/umpire/internal/engine/compose.go`
**Touches:** [proto/internal/temporal/server/api/umpire/v1/**, api/umpire/v1/**, model/check/**, tools/umpire/ir/validate.go, tools/umpire/ir/*test.go, tools/umpire/check/claims.go, tools/umpire/check/composed_progress_test.go, tools/umpire/internal/engine/compose.go]

### Approach
- Inspect the schema layout left by fn-145 before editing the Assumption/Progress leaf. Add the smallest structured reference that preserves owner, member field or sync, and action-wide versus concrete class selection; reuse existing class/input value encoding.
- Retain machine fairness behavior and use the same resolved composed classes as the existing own/synced selection logic. Disambiguate two fields containing the same machine by member field; no handwritten concatenated keys.
- Reuse composedFair and mergeAssumption for inherited mappings and same-name unions. Apply replacement-member assumptions exactly as composition semantics already define. Make claim-specific assumptions use the same owner-aware resolution.
- Reject wrong-owner, unknown member/sync/action, ambiguous selection and invalid parameterized class before checking. Do not strengthen weak fairness or grant disabled synchronizations fairness eligibility from local enabledness.
- Regenerate the necessary protobuf/Scala schema closure using existing generation commands, reviewed against the completed schema baseline; preserve existing machine pins.

### Investigation targets
**Required:**
- `proto/internal/temporal/server/api/umpire/v1/ir.proto:452` - current Assumption representation; follow extracted successor leaf.
- `tools/umpire/check/claims.go:455` - existing machine assumption binding.
- `tools/umpire/internal/engine/compose.go:67` - assumption inheritance/merge.
- `tools/umpire/internal/engine/compose.go:101` - mapping to composed classes.
- `model/check/Gate.scala` - schema packaging closure.

### Key context
Re-anchor paths and interfaces against completed fn-140/fn-141 and the approved schema/package moves before editing. Keep this work outside the activity batch and serialize shared regeneration with it and the schema chain; no new spec-close dependency is implied.

### Quick commands
```bash
go test -tags test_dep ./tools/umpire/ir ./tools/umpire/check ./tools/umpire/internal/engine
```

## Acceptance
- [ ] Structured references resolve own actions, syncs and exact parameterized classes deterministically, including repeated machine members.
- [ ] Same-name union and replacement assumptions match existing composition rules; wrong-owner/ambiguous/unknown references fail with attribution.
- [ ] Machine fairness and generated descriptor/Scala consumers retain their existing behavior; no stronger scheduling guarantee is introduced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
