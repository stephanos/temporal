---
satisfies: [R1, R2]
---
# fn-217-annotation-catalog-over-the-temporal.2 annotate! with four-level exhaustiveness and the descriptor-set build check

## Description
`annotate!` in `model/temporal` (a `macro_rules!` or small proc macro over the prost-generated `temporal.api` types): per annotated message, catalog entries plus an exhaustive struct pattern without `..`; compile-time kind-vs-type checks; tag/kind applicability and contradictory-pair refusals; `rules:` and `covers:` blocks; `build.rs` reads `proto/api.binpb` and requires every method of each named service to be classified (`Action`, `Observed`, `OutOfScope(reason)`); the action-binding check (every Model action bound, revealed or environmental). trybuild fixtures for each refusal and a build-script negative control with a mutated descriptor set (an added field, enum value and method each fail naming themselves). After .1. **Touches:** `model/temporal/annotate/**`, `model/temporal/api/build.rs`, `model/temporal/api/Cargo.toml`, `model/temporal/testing/**`, `model/Cargo.toml`, `model/Cargo.lock`. The actual descriptor build script is in the API crate. Keep annotation macros independent of generated API dependencies so they can be used from build scripts without a cycle. **Feedback loop (R30):** paste `make model-time`.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
