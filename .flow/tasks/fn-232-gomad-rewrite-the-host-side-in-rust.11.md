---
satisfies: [R10, R13, R14]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.11 install, doctor, discover, plan and recover in Rust; overlay invoked via the pinned Go toolchain

Touches: gomad/src/cli/{install.rs,doctor.rs,discover.rs,recover.rs}, gomad/src/install/{manifest.rs,toolchain.rs}

## Description
This task follows fn-232.10; the redundant transitive .6 edge does not permit starting before .10. Plan validation internals from .8 are reused when exposing the public plan verb here.
Port installation with a new manifest version: pin compiler, helper, runtime sources and configuration by digest, invoke the Go toolchain overlay generator through the pinned toolchain, never download; discover validates identities without opening build inputs; doctor, plan and recover as documented; preserve the WASM refusal for native-only analyze without porting native; Go-version manifests refused. Resolution order `--installation`, env, manifest beside binary, no fallback from an invalid one. Use the running binary's linked helper implementation by default. Preserve optional `--helper PATH` grammar, but authenticate an explicit path against the same linked Rust implementation identity and refuse any mismatch; never switch execution to that external path. Document the authorized required/default change from Go's mandatory external helper without changing flag grammar or refusal statuses.

## Acceptance
The owned behavior has reviewed golden fixtures, repeated-execution agreement and byte-identical second regeneration (R14); public verbs also run their fresh Go/Rust semantic parity lane (R13).
Fresh install pins the same unchanged compiler/runtime/input digests and actual new Rust implementation identities; all owned public verbs have parity and golden fixtures; missing or mismatched tools refused; Go-version manifest refused; discover works after build inputs are removed; omitted --helper derives the linked implementation, a matching explicit path is admitted and a mismatched/legacy helper is refused, with CLI documentation and golden coverage for this explicit default exception.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
