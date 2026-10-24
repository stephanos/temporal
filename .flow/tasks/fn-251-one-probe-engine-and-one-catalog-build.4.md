---
satisfies: [R4, R7]
---
# fn-251-one-probe-engine-and-one-catalog-build.4 One generic probe engine for Kettle and Temporal

## Description
Implement **Probe engine (B1)**: `umpire_catalog::generic_probe::{Engine, ProbeSystem}` replacing `kettle/probes.rs:19-292`, `temporal/realization/src/probes/generic.rs:21-345` and `report.rs` `observation_result`. Temporal's fingerprint-keyed strict predictions go through the trait.

## Acceptance
- [ ] One engine; Kettle and Temporal implement `ProbeSystem` only.
- [ ] Probe goldens hold; unknown method, bad prerequisite and unapplicable cases refuse as before.
- [ ] Re-accept goldens after reviewing their diffs; `make model-check` passes.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
