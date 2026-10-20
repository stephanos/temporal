---
satisfies: [R10]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.11 install, doctor, discover, plan and recover in Rust; overlay invoked via the pinned Go toolchain

Touches: gomad/src/cli/{install.rs,doctor.rs,discover.rs,recover.rs}, gomad/src/install/{manifest.rs,toolchain.rs}

## Description
Port installation with a new manifest version: pin compiler, helper, runtime sources and configuration by digest, invoke the Go toolchain overlay generator through the pinned toolchain, never download; discover validates identities without opening build inputs; doctor and recover as documented; Go-version manifests refused. Resolution order `--installation`, env, manifest beside binary, no fallback from an invalid one.

## Acceptance
Fresh install on a clean checkout produces a manifest whose digests equal the Go install's for the same inputs (digest comparison, not bytes); missing or mismatched tools refused; Go-version manifest refused; discover works after build inputs are removed.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
