---
satisfies: [R3, R4]
---
# fn-25-optional-callerclosure-veil-binding-and.3 Add Umpire Verify Veil opt-in dependency and result admission

**Size:** L
**Files:** `model/lakefile.toml`, `model/lake-manifest.json`, `model/Umpire/Formal/**`, `model/Temporal/Feature/Nexus/CallerClosure/Veil.lean`
**Touches:** primary Lake dependency closure, reusable external-lean admission, v3 receipt evidence, production handwritten declaration

## Description
### Umpire4 reconciliation (normative)

Generic optional checker mechanics live under `Umpire.Verify.Veil`; family bindings live under `Temporal.Verify`. Pin and prove Lean 4.33.1 compatibility before adoption because current Veil main and the legacy trees do not share this toolchain. Ordinary facades/builds must remain free of Veil.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

In adopt mode, add only the exact gate-selected immutable Veil requirement and resolved closure to the primary Lake project, then add the handwritten production family declaration behind focused non-default imports. Extend Umpire.Formal with a backend-name-free external-lean admission factory and non-breaking umpire-verification-receipt/v3 support: exact checker-view, checker-binding, and compatibility-gate evidence schemas/order/sources/digests while preserving v2 bytes. Compatibility evidence consumes only task .1's stable Definition ID/Behavior Fingerprint and never raw receipt bytes or a gate rerun. The factory accepts exact request/view/binding/checker/trust/completeness evidence and cannot forge claims, upgrade trust, or define an external transition IR. In defer mode, complete as not applicable and prove the Lake files, optional roots, external admission additions, and dependency closure remain absent.
## Acceptance
Adopt mode uses exactly one selected pinned requirement, unchanged upstream source, the current toolchain, one primary Lake project, no generated Lean, and no ordinary/default import. External establishment is possible only for complete bidirectional binding, exact Limits/identities, the deterministically selected command capability, and trust no stronger than the gate. V2 fixtures remain byte-identical; v3 uses the exact expanded evidence order and rejects partial v2 decoding. Defer mode has byte-equivalent dependency surfaces and no optional source. Reusable Umpire contains no Veil, Temporal, Workflow, or Nexus vocabulary.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
