---
satisfies: [R2, R3, R4, R5]
---
# fn-167-gaps-declared-in-the-model-gap-replaces.5 Held sweep: shared IR transport and metadata identity exclusions

## Description
Held sweep: serialize collected metadata through the existing shared IR and finish transport/inertness proofs (R2–R4, transport portion of R5). Do not start before the owner opens schema, Go and generation work.

**Size:** M
**Files:** shared proto, generated schema, Scala lifter/gate/equivalence, Go reader/identity/lower tests.
**Touches:** [proto/internal/temporal/server/api/umpire/v1/**, api/umpire/v1/**, model/irgen/**, model/check/**, tools/umpire/ir/**, tools/umpire/conformance/**, tools/umpire/lower/**]

### Approach
- Add metadata to the one existing schema; regenerate its actual ScalaPB/Go closure using established routes. Emit task 2/3 collections without a second exporter or sidecar.
- Exclude gaps and derivation from each identity projection independently: Scala equivalence, fingerprints, Go model identity and table reads. Cover nonempty metadata and default-empty fields, not just absent fields.
- Switch capability waivers to gap transport and retire the transient file/copy path only when the downstream consumer can use the same keys. Coordinate this switch with task 6 or keep it staged until that task is ready; real acceptances are not deleted here.

### Investigation targets
Required:
- `proto/internal/temporal/server/api/umpire/v1/ir.proto` — Model schema.
- `proto/internal/temporal/server/api/umpire/v1/machine.proto` — machine metadata.
- `model/check/Gate.scala:134` — schema closure generation.
- `model/check/Equivalence.scala:369` — Scala identity projection.
- `model/irgen/Lift.scala:129` — Model assembly.
- `tools/umpire/conformance` — existing semantic model identity.

### Verification
Test old/default-empty and nonempty metadata through reader validation and ProtoJSON round trips. Capture before/after exact fingerprint, equivalence, model identity and table-pin equality with independent negative controls that alter semantic rows. Run focused Scala/Go tests and scratch lift/lower without rewriting real feature artifacts; finish the waiver-switch seam with task 6 before claiming R4 complete.

## Acceptance
- [ ] Owner has explicitly opened schema, Go and generation work before admission.
- [ ] Shared wire round trips and independent default/nonempty metadata identity pins pass.
- [ ] Capability transport switches only with its working consumer; real-feature acceptance files are not retired here.

## Done summary
Superseded by fn-200/fn-201 (one-language series, 2026-10-18): no IR transport of gaps; @gap is resolved natively

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: