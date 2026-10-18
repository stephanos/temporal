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
Tasks .3 and .4 can now implement progress and derives without changing Cargo manifests or the lockfile. The workspace activates the admitted checker and DSL dependencies and adds umpire-dsl-macros with inherited lint policy, a documentation-only source file, the admitted macro dependencies, and the requested public DSL and shared fixture runner dev dependencies.

R18 wiring resolves with syn 3.0.9, darling 0.24.1, manyhow 0.14.1 with darling integration, and prettyplease 0.3.0. Cargo metadata and the feature tree are recorded in .flow/tmp/fn210-11-metadata.json and .flow/tmp/fn210-11-tree.txt. cargo check --locked --workspace --all-targets compiled the dev dependency graph. The documented skeleton implements no macros or re-exports. All 187 tracked fixture files have identical paths and Git blob IDs against the pre-edit base, verified by cmp of .flow/tmp/fn210-11-fixtures-{base,head}.txt. No Scala source changed.

baseline: green from make model2-check and cargo deny check bans before implementation. The initial nested-shell invocation observed the parent directory and was inconclusive; the corrected invocation explicitly entered temporal and passed before source edits. Final fresh make model2-check and cargo deny check bans both exited 0. The full gate covers formatting, Clippy with warnings denied, documentation, workspace tests and the WASI build. Logs are .flow/tmp/fn210-11-closure-gate.log and .flow/tmp/fn210-11-closure-bans.log; green receipts are .flow/tmp/green-receipts/8ed47c0c-unittest.json and .flow/tmp/green-receipts/8ed47c0c-bans.json.

stage: impl-review - ran (SHIP; three first-round lenses; codex:gpt-6.1-sol:high; same-family review)
Tier: project implementer gpt-6.1-sol at medium (judge unavailable no_key)

Review receipt: /tmp/impl-review-receipt-8f37faba39e2-fn-210-rust-prototype-kettle-on-a-trait-engine.11.json
The reviewers found no introduced findings. Their build attempts were inconclusive because the read-only sandbox could not open Cargo's build lock. The worker's fresh full gate passed afterward. Memory capture did not run because no NEEDS_WORK-to-SHIP transition occurred. The checkpoint also includes conductor-authorized Flow planning and MILESTONES changes that were ready in the shared checkout.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 8ed47c0c22d8ebedb9f43c1f51429b9e53fced1c
- Tests: make model2-check, cd model2 && mise exec -- cargo deny check bans, cd model2 && mise exec -- cargo check --locked --workspace --all-targets, cd model2 && mise exec -- cargo metadata --locked --format-version 1, cd model2 && mise exec -- cargo tree --locked -p umpire-dsl-macros -e normal,build,dev,features, cmp .flow/tmp/fn210-11-fixtures-base.txt .flow/tmp/fn210-11-fixtures-head.txt
- PRs: