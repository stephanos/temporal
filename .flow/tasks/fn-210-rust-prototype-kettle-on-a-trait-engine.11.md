---
satisfies: [R18]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.11 Prepare shared Cargo wiring for parallel progress and derive lanes

## Description
Serial preparation for the owner-requested parallel .3/.4 implementation. After .10, opt checker into admitted petgraph and thiserror; opt dsl into admitted enum-iterator and its path dependency on the new umpire-dsl-macros crate. Create only a documented empty proc-macro crate at umpire/dsl/macros with the admitted syn 3, quote, proc-macro2, darling 0.24 and manyhow 0.14 (darling feature) dependencies, plus insta and prettyplease dev dependencies. Add dsl and checker trybuild dev dependencies; add trybuild and umpire-dsl as macro-crate dev dependencies so owning-crate UI fixtures can compile against the public DSL. Add the existing umpire-testing crate as a dev dependency of both dsl and macros for their later owning-crate fixture runners. Verify the resulting dev-dependency graph with Cargo. Wire workspace members/path dependencies and generate Cargo.lock once. No derives, progress semantics, fixture changes or re-exports are implemented here. This is dependency/skeleton preparation only; .3 and .4 remain open. Later lanes may not edit Cargo manifests or lockfiles. Primary compatibility evidence: .flow/tmp/fn210-macro-dependencies-research.md.

**Touches:** model2/Cargo.toml, model2/Cargo.lock, model2/umpire/checker/Cargo.toml, model2/umpire/dsl/Cargo.toml, model2/umpire/dsl/macros/Cargo.toml, model2/umpire/dsl/macros/src/lib.rs

**Quick commands:** make model2-check; cd model2 && mise exec -- cargo deny check bans
## Acceptance
Shared workspace/dependency wiring is committed and make model2-check plus cargo deny check bans pass. The proc-macro skeleton has no implemented derives or machine attribute. Existing fixture payloads and semantics are unchanged. Tasks .3 and .4 can work within their declared disjoint source/test surfaces without changing any Cargo file.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
