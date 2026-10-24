---
satisfies: [R7, R9]
---
# fn-252-quick-consolidation-wins-in-model-code.7 Tooling: one proto-go script, wasm env variable, workspace lints for guests

## Description
Implement **Tooling (B11)** from the spec, touching only scripts fn-247 leaves in Bash.

## Acceptance
- [ ] One proto-go script; `guest-parity.sh` deleted; one wasm env variable.
- [ ] Guests use `lints.workspace = true`; unsafe confined to audited modules (gate evidence).
- [ ] Re-accept goldens after reviewing their diffs; `make model-check` passes.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
