---
satisfies: [R2, R7]
---
# fn-251-one-probe-engine-and-one-catalog-build.2 System::generate replaces build-configuration layers

## Description
Implement **Build configuration (B10)**: `umpire_catalog_build::System{..}.generate(descriptor, out_dir)`; remove `services::Config`'s derived fields, the `temporal/api/{services,wire_schema,codegen}.rs` and `annotate/syntax/src/services.rs` wrappers, and the duplicated descriptor build in `realization/macros/build.rs`.

## Acceptance
- [ ] Kettle and Temporal build through `System::generate`.
- [ ] Re-accept goldens after reviewing their diffs; `make model-check` passes.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
