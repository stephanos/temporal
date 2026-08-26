---
satisfies: [R2, R7]
---
# fn-23-veil-toolchain-compatibility-and.2 Author the isolated handwritten compatibility probe

## Description
Add the fixed handwritten compatibility source at `model/TemporalVeilCompatibility/Probe.lean` and the package support that validates and digests it before acquisition. Express only the minimal two-state transition system, one positive invariant, its establishing case, and one nearby failing mutation through documented Veil syntax. Keep the source outside ordinary `Umpire` and `Temporal` library roots, free of generated or copied product semantics, and load it only through the disposable overlay assembled by the diagnostic harness. Preserve existing comments.

**Size:** M
**Files:** `model/TemporalVeilCompatibility/Probe.lean`, `tools/umpire/veilcompat/probe.go`, `tools/umpire/veilcompat/probe_test.go`
**Touches:** [model/TemporalVeilCompatibility/Probe.lean, tools/umpire/veilcompat/probe.go, tools/umpire/veilcompat/probe_test.go]

## Acceptance
The checked-in source has one stable digest, parses into the expected positive/mutation semantic markers, and cannot be mistaken for a Temporal claim or ordinary model import. Tests reject generated markers, product imports, missing or duplicate cases, output-text-only success hooks, source drift, unsafe overlay paths, and any primary Lake dependency or manifest change.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
