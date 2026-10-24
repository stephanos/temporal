---
satisfies: [R1]
---
# fn-245-gomad-typed-boundaries-for-the-rust-host.5 Guest path and limit constants in one place

## Description
166 production literals of /workspace, /tmp, /proc, /dev across runtime_qualification.rs, install/config.rs (202, 577, 597, 893), measurement, qualify and wasmhost; the guest working directory, writable directories and preopens are restated per site. Define them once in wasm-model (guest::paths) and use the constants; same for the handful of repeated size literals that are true constants (IMAGE_METADATA_BYTES already is). Byte-identical configuration output is the proof (config goldens and installation identity unchanged).

## Acceptance
At most one definition each of the guest workspace, tmp and preopen paths; config goldens and installation identities unchanged; grep for '"/workspace' outside wasm-model and tests is empty.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
