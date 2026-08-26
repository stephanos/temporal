---
satisfies: [R1, R3, R4]
---
# fn-26-local-qualification-receipts-and-staged.1 Define reusable qualification profiles and the exact local policy

## Description
**Size:** M
**Files:** model/Umpire/Qualification/**, model/Temporal/System/Qualification/Local.lean
**Touches:** qualification vocabulary and local policy identity

Define the Temporal-free Umpire.Qualification vertical package and the single Temporal local-ephemeral profile. Freeze v1 only to the environment, claim, cleanup, and not-provided formal values exercised locally; later profiles must version the schema to add vocabulary. Freeze exact profile fields, validation, canonical digest, pilot binding shape, complete accumulating reason table/precedence, and local profile identity. Prove every requirement mutation changes or invalidates the digest and that no endpoint, credential, path, Temporal, or Nexus value enters reusable Umpire.

## Acceptance
The generic package validates the local-only closed canonical v1 independently, the local profile binds exactly fn-19/fn-20 declarations and declares formal evidence not-provided, and focused tests reject future/contradictory/empty/duplicate/unknown requirements. The exact accumulating reason/status matrix and all identity formulas are executable and reusable Umpire remains domain-neutral.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
