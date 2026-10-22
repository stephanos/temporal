# Gomad: context-sized Rust crates with boundary tests

## Conversation Evidence

> user: "I have this idea of cutting fairly small Rust crates. small enough so an agent can fit the entire crate into its context without problems."
> user: "they'd need very clear boundaries. and their tests would all exclusively test that boundaries instead of being unit tests."
> user: "that's on top of the existing fixture tests that are end to end we want to prioritize for testing effort."
> user: "yes write flow spec"

## Goal & Context

The Rust side of gomad is two crates: `tools/gomad/rust/gomad` (~35k lines, 17 top-level modules) and `tools/gomad/internal/backends/wasm/wasmhost` (~57k lines, 33 top-level modules). A worker assigned a task in `replay/` or `volume/` today has no crate-level statement of what it may touch and routinely reads into neighbours to find the seam. The owner wants crates sized so one fits in an agent's context with room to work, each with a boundary that is written down and enforced by the compiler, and tests that exercise that boundary rather than internals.

Survey on 2026-10-22 (grep over `crate::` paths and `wasmtime` imports):

- An agent holds roughly 10–15k lines of Rust with room to work, so the target is **8–12 crates**, not dozens. A task that touches four thin crates costs more context than one that touches a single well-cut crate.
- `evidence/` in `gomad` references no sibling module; it is a leaf consumed by `campaign` (25 references), `replay`, `qualify`, `stress` and `minimize`.
- `campaign/` depends on `evidence`, `qualify`, `replay` and `fixture`; it sits above them, not beside them.
- In `wasmhost`, `clock/`, `network/` and `volume/` reference none of `engine`, `model`, `world`, `cache`, and import no `wasmtime`. `namespace/` does import `wasmtime` (`namespace/mod.rs`, `namespace/transport.rs`) and so is not pure simulation.
- `wasmtime` is imported by 29 files across `engine`, `model`, `linked*`, `cluster/execution`, `abi`, `protocol`, `cache`, `storage_transport`, `transcript`. It is the heavy dependency; a crate that needs it pays its compile cost regardless of how the rest is cut.
- `wasmhost` already has seven integration tests under `tests/` that use only the public API. `gomad` has none; its tests are inline.

Testing policy (already decided, 2026-10-20): end-to-end fixture tests take priority; a unit test must name the invariant it protects. A crate boundary test is a named invariant at a narrower seam and uses the same fixture-in, golden-out style.

## Architecture & Data Models

One Cargo workspace rooted at `tools/gomad/rust/Cargo.toml` (it exists; `wasmhost` joins it as a path member or stays a path dependency, decided in task .1 by whichever keeps the Go `cgo`/build-script wiring unchanged). Each extracted crate is a directory beside `gomad/` with:

- `src/lib.rs` whose crate doc comment states the crate's job in one paragraph and lists what it deliberately does not do. This is the first thing a worker reads and is the context saver.
- `#![warn(missing_docs)]` so every public item is justified; the public surface is kept to what consumers and boundary tests use.
- `tests/` containing only boundary tests: public API in, asserted output out, fixture-driven where the input has a serialised form. No `#[cfg(test)]` access to internals from `tests/`.
- Inline `#[cfg(test)]` tests are permitted only when they name an invariant that cannot be reached through the public API; each such test states the invariant in its doc comment.

Cut only along seams the survey confirms, in dependency order (leaf first), one crate per task, measuring after each:

| order | crate | source modules | approx lines | depends on |
| --- | --- | --- | --- | --- |
| 1 | `gomad-evidence` | `gomad/src/evidence/` | ~5k | serde, sha2 |
| 2 | `wasm-sim` | `wasmhost/src/{clock,network,volume}/` plus `event_order`, `stress_jitter` if the survey confirms they have no `wasmtime`/`engine` edge | ~12k | none heavy |
| 3+ | decided by the task .4 review | candidates: `wasm-abi` (`abi`, `protocol`); `wasm-engine` (`engine`, `model`, `world`, `cache`, `namespace`); `wasm-cluster` (`cluster`, `linked*`); `gomad-campaign`; `gomad-replay`/`qualify`/`stress`/`minimize` as consumer crates; `gomad` bin as glue | | |

No code moves that are not crate extractions. Behaviour, CLI, fixtures, evidence formats and the helper protocol are unchanged; every existing e2e and fixture test passes unchanged after each extraction.

### Measurement

Each extraction records, in `.flow/artifacts/fn-240-gomad-context-sized-rust-crates-with/<crate>.md`:

- lines per crate before and after (`find … -name '*.rs' | xargs wc -l`),
- `cargo build --timings` wall time for the workspace, clean and incremental, before and after,
- public item count (`cargo doc` item count or `grep -c '^pub '` over `lib.rs` re-exports),
- number of `pub` items added solely to make a boundary test possible (target: zero; each one is listed with its reason).

## API Contracts

- `cargo test --workspace` and the Go-side build (`go build ./...` under `tools/gomad`, plus the WASM helper build) pass unchanged after every task.
- Every extracted crate's `lib.rs` doc comment answers "what does this crate own" and "what does it refuse to own" in prose a worker can read in under a minute.
- `tests/` in each extracted crate compiles against the crate's public API only; `cargo test -p <crate>` runs its boundary tests in isolation without the workspace's other crates as dev-dependencies, except fixtures shared via a path to `testdata/`.
- No crate re-exports another crate's entire module tree (`pub use other::*`) to paper over a seam.

## Edge Cases & Constraints

- `wasmhost` is built from Go via its own `Cargo.toml` and `build.rs`; the Go invocation and the produced artifact path must not change. If joining the workspace moves `target/`, the shared cargo cache convention (`~/.cache/codex-build/cargo-target`) is preserved.
- `namespace/` imports `wasmtime`; it does not go into `wasm-sim`. If a sim-only core exists inside it, that is a later cut, not part of task .3.
- A boundary test that needs an internal is a signal the boundary is wrong, not a licence to widen `pub`. The task records the case and either moves the seam or keeps the test inline with its invariant named.
- Compile time may get worse with a deep chain of thin crates. If the task .4 measurement shows clean-build wall time up by more than 20%, further cuts stop until the DAG is widened.
- Workers run in parallel lanes on worktrees; an extraction is a large mechanical diff that conflicts with every open branch touching the moved modules. Each extraction task lands on its own branch and is merged before the next starts; the conductor checks open branches for overlap before claiming.
- No CI run, PR or push is authorized by this spec.

## Acceptance Criteria

- **R1:** A dependency survey artifact lists, for every top-level module of both crates, its `crate::` edges and whether it imports `wasmtime`, and derives the cut table above from it. Errors: a proposed crate whose modules have an unlisted edge is a finding.
- **R2:** `gomad-evidence` is a workspace crate with a `lib.rs` contract doc, `missing_docs` on, and boundary tests under `tests/` that cover store open/publish, symbolize and inspection through the public API; all prior inline tests either moved to `tests/` or retained with a named invariant; `cargo test --workspace` and the e2e fixture suite pass unchanged. Errors: a `pub` added only for a test and not listed with its reason is a finding.
- **R3:** `wasm-sim` is a crate with no `wasmtime` dependency, holding clock, network and volume with their existing corpus/conformance tests as boundary tests; the Go helper build is unchanged. Errors: a `wasmtime` import anywhere under `wasm-sim`, or a change to the helper artifact path, is a finding.
- **R4:** After R2 and R3 the measurement artifact compares lines, build times and public surface before and after, and a go/no-go decision for the next cut is recorded with its reason. Errors: a decision without the measurements is a finding.
- **R5:** Every crate created under this spec has a `lib.rs` doc that names what the crate owns and what it does not; a worker reading only that doc and `tests/` can state the crate's contract. Errors: a crate without that doc is a finding.

## Boundaries

- No behaviour change, no API redesign, no renaming beyond the crate paths; a module keeps its name inside the new crate.
- No cut inside a module (`engine.rs` is not split here).
- No new test framework, proc-macro or dev-dependency beyond what the workspace already has.
- No reduction of e2e fixture coverage; boundary tests are added beside them, not instead of them.
- Not more than two extractions before the task .4 review.
- No CI run, PR or push.

## Decision Context

Owner proposal (2026-10-22): many small crates, each context-sized, boundary tests only. Agreed on the contract-and-boundary-test discipline; narrowed the crate count to 8–12 because the context cost of a task is driven by the seams it crosses, not the size of the crate it sits in, and because thin chains compile serially. Rejected: a big-bang split (wrong seams are expensive to undo and conflict with every open lane); tests-only-through-public-API as an absolute rule (it inflates `pub` surface); cutting `namespace` into the sim crate (it imports `wasmtime`).

Sequencing: whole spec after fn-232 (Rust host rewrite chain) lands; spec-level dependency recorded. The fn-232 chain is the largest open diff over `wasmhost`, and the evidence crate cut is cheap enough that it is not worth a separate lane before then.

## Early proof point

Task .2, first hour: move `evidence/` into a crate with no changes except `use` paths and confirm `cargo test --workspace` passes. If more than `use` path edits are needed, the seam is not where the survey says it is; stop and report.

## Quick commands

```sh
cd tools/gomad/rust
cargo build --workspace --timings                       # baseline, record wall time
cargo test --workspace
cargo test -p gomad-evidence                            # boundary tests only, after task .2
grep -rn 'wasmtime' ../internal/backends/wasm/wasmhost/src/{clock,network,volume}   # expect empty
(cd ../ && go build ./...)                              # Go side unchanged
```

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Dependency survey and derived cut table. | fn-240-gomad-context-sized-rust-crates-with.1 | — |
| R2 | `gomad-evidence` extracted with contract doc and boundary tests. | fn-240-gomad-context-sized-rust-crates-with.2 | — |
| R3 | `wasm-sim` extracted without `wasmtime`; helper build unchanged. | fn-240-gomad-context-sized-rust-crates-with.3 | — |
| R4 | Measurement and go/no-go for further cuts. | fn-240-gomad-context-sized-rust-crates-with.4 | — |
| R5 | Every new crate carries its contract doc. | fn-240-gomad-context-sized-rust-crates-with.2, fn-240-gomad-context-sized-rust-crates-with.3 | — |
