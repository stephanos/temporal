---
satisfies: [R1]
---
# fn-228-runner-and-run-the-one-go-host-loop.1 Renames alone: umpire-wasm to umpire-runtime, umpire-tests to umpire-fixtures, Realization in prose

## Description
Rename the crate umpire-wasm to umpire-runtime and umpire-tests to umpire-fixtures across Cargo manifests, Makefile targets, scripts, guest imports, README, DESIGN, MIGRATE and Go host comments; replace the word binding with realization in Umpire prose. Add a gate grep that fails on the old names. Byte-identical goldens and wire goldens; make model-check and UMPIRE_BACKENDS=require pass; model-time pasted. Runs alone, no other task in flight.

## Acceptance
- [ ] No remaining umpire-wasm/umpire_wasm or umpire-tests/umpire_tests; gate grep in place
- [ ] Prose says realization, not binding
- [ ] All goldens byte-identical; gate passes; model-time pasted

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
