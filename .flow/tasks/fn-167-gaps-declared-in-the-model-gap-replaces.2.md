---
satisfies: [R1, R2, R3, R4, R7]
---
# fn-167-gaps-declared-in-the-model-gap-replaces.2 Resolve gap subjects, owners, waivers and expiry references in Scala

## Description
Resolve annotations into internal lifter metadata (groundwork for R1–R4 and R7). No new protobuf fields or sidecar output are introduced here.

**Size:** M
**Files:** gap collection, lifter context/integration, capabilities integration, gap fixtures and docs.
**Touches:** [model/irgen/Gaps.scala, model/irgen/Context.scala, model/irgen/Lifting.scala, model/irgen/Lift.scala, model/irgen/Capabilities.scala, model/irgen/test/Gaps.test.scala, model/irgen/testdata/gaps/**, model/README.md, model/SEMANTICS.md]

### Approach
- Reuse existing symbol identities, positions, facet consumer ownership and capability waiver resolution. Keep annotation placement separate from matched finding owner; normalize each constructor to the current kind's exact subject representation.
- Resolve object/val annotations once and preserve annotation/subject positions. Detect duplicate full finding keys after action-class expansion and machine attribution, including across realizations that share a machine owner.
- Collect `except`/`overriding` metadata alongside gaps, preserving existing waiver emission and the gate copy path until task 5.
- Resolve expiry existence via an injected read-only spec registry: exact canonical id, then unique short id. Cover missing, ambiguous and malformed references. Do not introduce requirement-level completion semantics or a public Flow configuration surface.
- Integrate collection/validation into the normal Scala lift. Existing ProtoJSON remains the transport; the collected value is private implementation data for task 5, not a parallel IR.

### Investigation targets
Required:
- `model/irgen/Context.scala:297` — positions and owner helpers.
- `model/irgen/Capabilities.scala:377` — resolved waiver identities.
- `model/irgen/Lift.scala:129` — current Model assembly and waiver output.
- `tools/umpire/lint/accept.go:88` — exact acceptance key.
- `tools/umpire/lint/holes.go:711` — concrete hole subjects.
- `tools/umpire/lint/kinds.go:149` — evidence aggregation.

### Verification
Extend GapsSuite with each supported placement, shared facets, realization/machine owner differences, Property versus Query subjects, defaults, duplicate positions and invalid typed references. Pin unchanged semantic protobuf output modulo existing position normalization and existing waiver output on no-gap and gap fixtures; record full wire/inertness proof as still deferred. Run the complete lifter suite at this task boundary with frozen sources.

## Acceptance
- [ ] Canonical owner/subject resolution and duplicate-position controls pass for all supported placements.
- [ ] Expiry existence/refusal fixtures and waiver collection pass without changing current wire/copy behavior.
- [ ] Complete lifter suite passes; full transport-dependent R2–R4/R7 proof remains explicitly open.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
