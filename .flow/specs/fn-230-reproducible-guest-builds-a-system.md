# Reproducible guest builds: a system-agnostic wasm build step, the committed Temporal guest, and the freshness gate (model, temporal)

## Conversation Evidence

> user (2026-10-20): "when we build the model/ as WASM; we need to check that into VCS actually as we don't want uers to have to install Rust to be able to run tests"
> assistant: agreed. Commit only the Temporal guest, next to the Go module that loads it with `go:embed`; the gate rebuilds and byte-compares like the protobuf freshness checks; that needs a reproducible build (dedicated profile, no debug info, stripped, remapped paths) proven on two checkouts. Measured: temporal guest 2.9 MB with debug symbols; two clean builds on 2026-10-19 differ by a few hundred bytes, so reproducibility is real work. MIGRATE.md's "separate decision" line becomes the decision.
> user: "hve we captured the reproducisble, system-agnostic WASM output build step; if not let's"

## Goal & Context

Go tests in the Temporal repository read the Model guest from the Cargo target directory through an environment variable, so anyone running them needs the Rust toolchain. The owner's decision: the built guest is checked into version control, and Go is the only toolchain a test runner needs. CI rebuilds and compares the artifact. This spec proves Linux path independence; fn-232-linux-macos-guest-byte-identity owns the retained cross-OS certification separately. The build step is system-agnostic and lives in Umpire; the committed artifact and its freshness check are Temporal's.

## Architecture & Data Models

- **Guest profile in Umpire.** A Cargo profile `guest` in the workspace manifest, inheriting release: `debug = false`, `strip = "symbols"`, `lto = "fat"`, `codegen-units = 1`, `overflow-checks = true` (fn-210 R11), `panic = "unwind"` (panics are caught per step and refute it; abort would change verdicts). Path remapping through `--remap-path-prefix` for the workspace, the cargo registry and the toolchain sysroot, so absolute paths never enter the binary. Target `wasm32-wasip1`, `--locked`, the mise-pinned toolchain.
- **One script.** `model/scripts/guest.sh <crate> <output>` builds any guest crate with that profile and copies the result to `<output>`; `--check` builds and byte-compares against the existing file, failing with a short diff summary (size and first differing offset). It knows no crate names and no Temporal paths; the Makefile supplies them.
- **Committed artifact.** `model/temporal/hosts/go/umpire_temporal_guest.wasm`, embedded with `go:embed` by the Temporal host module and, after fn-228, by the Runner. The environment variable lookup and target directory path logic are removed from the Go tests. `.gitattributes` marks `*.wasm` binary. Umpire's stock and control guests are not committed; the Rust gate builds them as today.
- **Freshness gate.** `make model-check` runs `guest.sh --check` for the Temporal guest, so a stale commit fails in CI, where Rust is installed. `make umpire-guest` refreshes it for Model authors. The slice targets (`umpire-check-model-slice`, the umbrella) use the committed file, so the tests and the artifact cannot disagree.
- **Reproducibility proof.** Two clean builds from separate checkouts (different absolute paths, same pinned toolchain) produce identical bytes, recorded in DESIGN.md with the sizes. A failure of that proof blocks the freshness gate, since a non-reproducible artifact would make the check noise.

## API Contracts

- `[profile.guest]` as listed; no other profile changes.
- `model/scripts/guest.sh <crate> <output> [--check]`: exit 0 on success or fresh; 1 on build failure; 2 on stale with the summary on stderr.
- Make: `umpire-guest` (refresh), the freshness step inside `model-check`, slice targets reading the committed path.
- Go: `//go:embed umpire_temporal_guest.wasm` in the Temporal host module exporting `Guest() []byte`; the Runner reuses it.

## Edge Cases & Constraints

- The guest profile applies only to guest crates; the checker's release profile keeps its symbols for profiling.
- `panic = "unwind"` is kept; a panic inside a step still refutes it and the wasm report stays byte-identical to the Rust goldens (fn-210 R24).
- Size: stripped Temporal guest expected around 1 MB; recorded in DESIGN.md. Git history grows by the file size per Model change; LFS is not adopted now and is named as the fallback if the file exceeds 5 MB.
- The freshness check needs the pinned toolchain and the wasm target; it reports a missing toolchain as a skipped step with a warning only outside CI (`CI` unset), and fails in CI.
- R30 budgets: the guest build is already part of the gate; the check adds one compare. Report `make model-time`.
- Byte identity across operating systems belongs to fn-232-linux-macos-guest-byte-identity, following the owner's split decision. It remains required there and does not gate this spec or its consumers. This spec proves two-checkout Linux/arm64 identity. A third platform producing different bytes remains a documented limitation.

## Acceptance Criteria

- **R1:** The `guest` profile and `guest.sh` build any guest crate byte-identically from two clean checkouts at different absolute paths on the pinned toolchain; the proof with sizes is recorded in DESIGN.md. Errors: build failure exit 1; a non-identical pair fails the task.
- **R2:** The Temporal guest is committed beside the Temporal host module, embedded with `go:embed`, and the Go tests and slice targets use it with no environment variable and no Cargo target path; `go test` in the Temporal host module passes on a machine without Rust. Errors: a missing embedded file is a compile error.
- **R3:** `make model-check` fails on a stale committed guest with the size and first-differing-offset summary; `make umpire-guest` refreshes it; outside CI a missing toolchain skips with a warning. Errors: exit 2 on stale, 1 on build failure.
- **R4:** MIGRATE.md, README.md and the hosts README record the decision: the Temporal guest is committed and freshness-checked; Umpire ships no prebuilt wasm; `.gitattributes` marks wasm binary. No error surface beyond the gate.

## Boundaries

- No prebuilt wasm for Umpire's own guests, no release or distribution pipeline, no LFS.
- No change to the ABI, exports, protocol or Realization.
- No Component Model or alternative targets; `wasm32-wasip1` only.
- The Runner's use of the embedded guest is fn-228's; this spec provides the embed.

## Decision Context

The owner moved macOS verification to fn-232-linux-macos-guest-byte-identity so Linux implementation and embedding can proceed. The caught-panic requirement is unchanged and still requires resolution before task .1 can complete.

2026-10-20, owner: commit the built wasm so users running tests need no Rust. The root agent's assessment, accepted: commit only the Temporal guest beside its Go module, freshness-check it in the gate like the protobuf modules, and make the build reproducible first. The build step is system-agnostic by design so the extracted Umpire repository keeps it and the Temporal repository keeps only the committed artifact. Rejected: committing all three guests (the stock and control guests serve Umpire's gate, which has Rust); a download step at test time (reintroduces a toolchain and a network dependency); keeping the environment-variable path as a fallback (two sources of truth). Placement: no spec dependency; lands before fn-228.4, where the live slice runs through the Runner.
