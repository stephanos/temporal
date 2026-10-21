---
satisfies: [R1]
---
# fn-238-shared-front-end-one-model-reading-for.1 Design: shared admission front end versus shared interpreted IR, measured against today's duplication

## Description
Read-only design task producing `model/design/shared-front-end.md`. After fn-227.14.

### Approach

1. Measure duplication between `umpire-dsl-macros` and `umpire-lift` (attribute grammar, registration, World/state/action schema, domains, `&World` classification, admitted expression subset, diagnostics), with file:line and line counts.
2. Option A: shared admission crate; macros still emit executable Rust bodies; lifter emits TLA+ from the shared admitted structure.
3. Option B: shared semantic IR; the native checker interprets it (as the owner phrased it), lifter emits from it. Assess authoring limits (Models already restricted to the admitted subset), wasm guest and Realization execution, checker performance, debugging, and how fn-211 and future Quint-parity constructs plug in.
4. Estimate line reduction and per-construct cost for each; recommend one with a migration task breakdown that keeps goldens, TLC agreement and typed replay unchanged.

### Verification

The design is reviewed; no code changes.

## Acceptance
- [ ] Measured duplication with file:line evidence.
- [ ] Options A and B compared on code size, authoring, wasm/Realization, performance, backends and future constructs.
- [ ] A recommendation with a migration task breakdown and golden/replay preservation plan.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
