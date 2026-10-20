---
satisfies: [R1, R14]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.1 gomad binary crate: link the helper as a library, in-process execution with slots and bounds

Touches: tools/gomad/rust/Cargo.toml (workspace), tools/gomad/rust/gomad/src/{main.rs,execute.rs,slots.rs}, wasmhost/Cargo.toml (lib + bin targets)

## Description
Create a Cargo workspace with a `gomad` binary crate and turn the helper into a library target the binary links (keep the helper bin target until R12). Implement `execute`: build a fresh Wasmtime engine/store per execution with the existing fuel, memory and wall-clock bounds, run the guest, collect termination, streams, transcripts and choice evidence through the library's existing model types. Implement a fixed slot pool with `std::thread::scope`; a panic in a slot aborts the run with an explicit error. Use std and only the helper's existing exact-pinned dependencies/helpers (wasmtime, serde, serde_json, sha2, crc32fast, base64 and rustix). Preserve CLI grammar with a std parser, reuse existing codec/error/hex helpers and sorted BTreeMaps, and use existing pinned filesystem/locking APIs where available. Do not introduce additional third-party libraries without a specific owner request; preserve full locking/publication semantics rather than substituting weaker homemade protocols.

## Acceptance
Use the foundation's fixture-only driver for the execution/store behavior below; no later public verb is a prerequisite. Actual public-verb goldens land with their owning tasks.
Golden corpus lane: fixture guests reach exit, trap, fuel exhaustion and watchdog timeout with the same termination classes as the Go backend, diffed as golden `--json` output; two slots run two seeds concurrently with a golden ledger; `make validate` still passes for the helper library. Unit tests only for the slot-panic abort path, which names its invariant.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
