# Gomad: context-sized Rust crates with boundary tests

**Owner acceptance amendment, 2026-10-24 (legacy compatibility retired).** Apply the [current compatibility policy](../artifacts/legacy-compatibility-policy-2026-10-24.md). Legacy Go/Python byte, schema, CLI/message and compatibility-only frozen-host requirements below are retired. Current behavior, integrity and same-candidate deterministic replay remain required; compatibility-only proof rows may retire explicitly without replacement proof.

## Conversation Evidence

> user: "I have this idea of cutting fairly small Rust crates. small enough so an agent can fit the entire crate into its context without problems."
> user: "they'd need very clear boundaries. and their tests would all exclusively test that boundaries instead of being unit tests."
> user: "that's on top of the existing fixture tests that are end to end we want to prioritize for testing effort."
> user: "yes write flow spec"

## Goal & Context

The Rust side of gomad is two workspace crates under `tools/gomad/rust/`: `gomad` (about 74k production and 37k test lines) and `wasmhost` (about 43k production and 14k test lines), per the 2026-10-23 survey in `.flow/artifacts/fn240-crate-survey-2026-10-23.md`; the figures below from the 2026-10-22 grep are stale. A worker assigned a task in `replay/` or `volume/` today has no crate-level statement of what it may touch and routinely reads into neighbours to find the seam. The owner wants crates sized so one fits in an agent's context with room to work, each with a boundary that is written down and enforced by the compiler, and tests that exercise that boundary rather than internals.

Survey on 2026-10-22 (grep over `crate::` paths and `wasmtime` imports):

- An agent holds roughly 10–15k lines of Rust with room to work, so the target is **8–12 crates**, not dozens. A task that touches four thin crates costs more context than one that touches a single well-cut crate.
- `evidence/` in `gomad` references no sibling module; it is a leaf consumed by `campaign` (25 references), `replay`, `qualify`, `stress` and `minimize`.
- `campaign/` depends on `evidence`, `qualify`, `replay` and `fixture`; it sits above them, not beside them.
- In `wasmhost`, `clock/`, `network/` and `volume/` reference none of `engine`, `model`, `world`, `cache`, and import no `wasmtime`. `namespace/` does import `wasmtime` (`namespace/mod.rs`, `namespace/transport.rs`) and so is not pure simulation.
- `wasmtime` is imported by 29 files across `engine`, `model`, `linked*`, `cluster/execution`, `abi`, `protocol`, `cache`, `storage_transport`, `transcript`. It is the heavy dependency; a crate that needs it pays its compile cost regardless of how the rest is cut.
- `wasmhost` already has seven integration tests under `tests/` that use only the public API. `gomad` has none; its tests are inline.

Testing policy (already decided, 2026-10-20): end-to-end fixture tests take priority; a unit test must name the invariant it protects. A crate boundary test is a named invariant at a narrower seam and uses the same fixture-in, golden-out style.

## Architecture & Data Models

One Cargo workspace rooted at `tools/gomad/rust/Cargo.toml`; `wasmhost` is already a member with one `Cargo.lock` (owner decision 2026-10-21), and there is no Go build of it any more. The helper artifact path used by `wasm-test-helper`/`publish-executable.sh` must stay unchanged. Each extracted crate is a directory beside `gomad/` with:

- `src/lib.rs` whose crate doc comment states the crate's job in one paragraph and lists what it deliberately does not do. This is the first thing a worker reads and is the context saver.
- `#![warn(missing_docs)]` so every public item is justified; the public surface is kept to what consumers and boundary tests use.
- `tests/` containing only boundary tests: public API in, asserted output out, fixture-driven where the input has a serialised form. No `#[cfg(test)]` access to internals from `tests/`.
- Inline `#[cfg(test)]` tests are permitted only when they name an invariant that cannot be reached through the public API; each such test states the invariant in its doc comment.

Cut only along seams the survey confirms, in dependency order (leaf first), one crate per task, measuring after each:

| order | crate | source modules | approx lines | depends on |
| --- | --- | --- | --- | --- |
| 1 | `gomad-evidence` | `gomad/src/evidence/` | ~5k | serde, sha2 |
| 2 | `wasm-sim` | `wasmhost/src/{clock,network,volume}/` plus `event_order`, `stress_jitter` if the survey confirms they have no `wasmtime`/`engine` edge | ~12k | none heavy |
| 3+ | tasks .6–.13, gated by the .4 review | the target layout in Amendment 4 below (owner decision 2026-10-24) | | |

No code moves that are not crate extractions. Behaviour, CLI, fixtures, evidence formats and the helper protocol are unchanged; every existing e2e and fixture test passes unchanged after each extraction.

### Measurement

Each extraction records, in `.flow/artifacts/fn-240-gomad-context-sized-rust-crates-with/<crate>.md`:

- lines per crate before and after (`find … -name '*.rs' | xargs wc -l`),
- `cargo build --timings` wall time for the workspace, clean and incremental, before and after,
- public item count (`cargo doc` item count or `grep -c '^pub '` over `lib.rs` re-exports),
- number of `pub` items added solely to make a boundary test possible (target: zero; each one is listed with its reason).

## API Contracts

- `cargo test --workspace`, the retained Go side (`go build ./...` under `tools/gomad`: models, generator, overlay, archcheck) and the merge gate (`make -C tools/gomad test-harness test-rust-engine validate-wasm-determinism`) pass unchanged after every task.
- Every extracted crate's `lib.rs` doc comment answers "what does this crate own" and "what does it refuse to own" in prose a worker can read in under a minute.
- `tests/` in each extracted crate compiles against the crate's public API only; `cargo test -p <crate>` runs its boundary tests in isolation without the workspace's other crates as dev-dependencies, except fixtures shared via a path to `testdata/`.
- No crate re-exports another crate's entire module tree (`pub use other::*`) to paper over a seam.

## Edge Cases & Constraints

- `wasmhost` is built by cargo as a workspace member; the helper artifact path consumed by `wasm-test-helper`/`publish-executable.sh` must not change. The shared cargo cache convention (`~/.cache/codex-build/cargo-target`) is preserved.
- `namespace/` imports `wasmtime`; it does not go into `wasm-sim`. If a sim-only core exists inside it, that is a later cut, not part of task .3.
- A boundary test that needs an internal is a signal the boundary is wrong, not a licence to widen `pub`. The task records the case and either moves the seam or keeps the test inline with its invariant named.
- Compile time may get worse with a deep chain of thin crates. If the task .4 measurement shows clean-build wall time up by more than 20%, further cuts stop until the DAG is widened.
- Workers run in parallel lanes on worktrees; an extraction is a large mechanical diff that conflicts with every open branch touching the moved modules. Each extraction task lands on its own branch and is merged before the next starts; the conductor checks open branches for overlap before claiming.
- No CI run, PR or push is authorized by this spec.

## Acceptance Criteria

- **R1:** A dependency survey artifact lists, for every top-level module of both crates, its `crate::` edges and whether it imports `wasmtime`, and derives the cut table above from it. Errors: a proposed crate whose modules have an unlisted edge is a finding.
- **R2:** `gomad-evidence` is a workspace crate with a `lib.rs` contract doc, `missing_docs` on, and boundary tests under `tests/` that cover store open/publish, symbolize and inspection through the public API; all prior inline tests either moved to `tests/` or retained with a named invariant; `cargo test --workspace` and the e2e fixture suite pass unchanged. Errors: a `pub` added only for a test and not listed with its reason is a finding.
- **R3:** `wasm-sim` is a crate with no `wasmtime` dependency, holding clock, network and volume (plus `event_order`, `stress_jitter` and the pure controller half of `linked_simulation`, as the survey requires) with their existing corpus/conformance tests as boundary tests; the helper artifact path is unchanged. Errors: a `wasmtime` import anywhere under `wasm-sim`, or a change to the helper artifact path, is a finding.
- **R4:** After R2 and R3 the measurement artifact compares lines, build times and public surface before and after, and a go/no-go decision for the next cut is recorded with its reason. Errors: a decision without the measurements is a finding.
- **R5:** Every crate created under this spec has a `lib.rs` doc that names what the crate owns and what it does not; a worker reading only that doc and `tests/` can state the crate's contract. Errors: a crate without that doc is a finding.
- **R6:** Below `wasm-engine` no crate depends on `wasmtime`: `cargo tree -p wasm-sim -p wasm-model -p wasm-hooks -p wasm-diagnostic -i wasmtime` is empty, and `wasm-conformance` is a bin that links `wasm-model` only. Errors: a `wasmtime` edge beneath the engine is a finding.
- **R7:** Every `build.rs` that emits an implementation SHA hashes the sources of every workspace crate it depends on, and a test flips one byte in a dependency crate and asserts the identity changes; the determinism inventory scanner walks every workspace crate root and no inventory row is dropped. Errors: an extracted source outside its consumer's identity closure, or an unscanned crate root, is a finding.
- **R8:** The feature-gated cross-cutting code (`diagnostic-profile`, `determinism-fixture`, `stress-jitter`, `conformance`) is reached through `wasm-hooks` traits or separate crates; the only remaining Cargo features are `reference-helper` on the helper bin and `guest-profile` on `wasm-engine`. Errors: a `cfg(feature)` in a library crate other than `wasm-engine`'s `guest-profile` is a finding.
- **R9:** The `bins/gomad` crate holds argv parsing and dispatch only; every verb body is a library function in its owning crate. Errors: a `cli` module containing campaign, replay, install or measurement logic is a finding.
- **R11:** No `pub` item of a library crate returns `Result<_, String>`; each crate's public functions return that crate's error enum, which carries the record classification kind (`debug_error`, `divergence`, `capacity`, …) as an enum with `as_str`, defined once. Messages pinned by goldens or CLI exit-status tests stay byte-identical. Errors: a `Result<_, String>` in a `pub` signature, or a kind literal defined outside the enum, is a finding.
- **R12:** `clippy::too_many_lines` and `clippy::cognitive_complexity` are workspace lints at `warn` with CI `-D warnings`; today's offenders carry explicit `#[allow]`s listed in `.flow/artifacts/fn-240/lint-allows.md`. The list never grows, and a task that touches an allowed function removes its allow. Errors: a new allow, or an allow kept on a function the task rewrote, is a finding.
- **R10:** The final layout matches Amendment 4 (crate set, dependency graph, `crates/`/`bins/`/`tools/` directories); `cargo build --workspace --timings` clean wall time is not more than 20% above the task .1 baseline, and no crate exceeds about 17k production lines. Errors: a crate outside the graph, or a dependency edge the graph does not show, is a finding.

## Boundaries

- No behaviour change and no API redesign. A module keeps its name inside the new crate. The only renames are crate paths, the `crates/`/`bins/`/`tools/` directory relayout in task .13, and the named `linked_*`/`fixture_*`/`live_*` sibling files folding into directories.
- No cut inside a module except the five named in Amendment 4 (the `HostError` substitution, the `linked_simulation` controller, the `cli` option parser and verb bodies, the `engine`→`cluster` inversion, and `measurement::stress_driver`/`cluster_driver`'s stress calls). Any other in-module cut is a finding.
- No new test framework, proc-macro or dev-dependency beyond what the workspace already has.
- No reduction of e2e fixture coverage; boundary tests are added beside them, not instead of them.
- Not more than two extractions before the task .4 review. After .4 records go, tasks .6–.13 proceed in dependency order, one crate per branch, merged before the next starts.
- No CI run, PR or push.

## Decision Context

Owner proposal (2026-10-22): many small crates, each context-sized, boundary tests only. Agreed on the contract-and-boundary-test discipline; narrowed the crate count to 8–12 because the context cost of a task is driven by the seams it crosses, not the size of the crate it sits in, and because thin chains compile serially. Rejected: a big-bang split (wrong seams are expensive to undo and conflict with every open lane); tests-only-through-public-API as an absolute rule (it inflates `pub` surface); cutting `namespace` into the sim crate (it imports `wasmtime`).

Sequencing: whole spec after fn-232 (Rust host rewrite chain) lands; spec-level dependency recorded. The fn-232 chain is the largest open diff over `wasmhost`, and the evidence crate cut is cheap enough that it is not worth a separate lane before then.

## Early proof point

Task .2, first hour: move `evidence/` into a crate. The survey already names three back-edges to invert first (`evidence/store/open.rs` → `minimize::source::read` and `campaign::origin::authenticate`; `evidence/search_context.rs` → `site_scope` and `choice::hex`) and two cross-module test files to move to `gomad`. Beyond those listed relocations, only `use` path edits are expected; if more is needed, the seam is not where the survey says it is; stop and report.

## Quick commands

```sh
cd tools/gomad/rust
cargo build --workspace --timings                       # baseline, record wall time
cargo test --workspace
cargo test -p gomad-evidence                            # boundary tests only, after task .2
grep -rn 'wasmtime' wasmhost/src/{clock,network,volume}   # expect empty
(cd ../ && go build ./...)                              # retained Go side unchanged
```

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Dependency survey and derived cut table. | fn-240-gomad-context-sized-rust-crates-with.1 | — |
| R2 | `gomad-evidence` extracted with contract doc and boundary tests. | fn-240-gomad-context-sized-rust-crates-with.2 | — |
| R3 | `wasm-sim` extracted without `wasmtime`; helper build unchanged. | fn-240-gomad-context-sized-rust-crates-with.3 | — |
| R4 | Measurement and go/no-go for further cuts. | fn-240-gomad-context-sized-rust-crates-with.4 | — |
| R5 | Every new crate carries its contract doc. | fn-240-gomad-context-sized-rust-crates-with.2, fn-240-gomad-context-sized-rust-crates-with.3, fn-240-gomad-context-sized-rust-crates-with.6–.12 | — |
| R6 | No `wasmtime` below `wasm-engine`; conformance is a plain bin. | fn-240-gomad-context-sized-rust-crates-with.5, fn-240-gomad-context-sized-rust-crates-with.7 | — |
| R7 | Identity closure and inventory scanner cover every crate. | fn-240-gomad-context-sized-rust-crates-with.5 | — |
| R8 | Hooks and side crates replace cross-cutting features. | fn-240-gomad-context-sized-rust-crates-with.6 | — |
| R9 | Bin crate is dispatch only. | fn-240-gomad-context-sized-rust-crates-with.9, fn-240-gomad-context-sized-rust-crates-with.10 | — |
| R10 | Final layout, build-time bound, crate size bound. | fn-240-gomad-context-sized-rust-crates-with.13 | — |
| R11 | Typed error enums at every library crate boundary. | fn-240-gomad-context-sized-rust-crates-with.5 (HostError), .2, .3, .6–.12 (each extracted crate), .13 (audit) | — |
| R12 | Size and nesting lints with a shrinking allow-list. | fn-240-gomad-context-sized-rust-crates-with.5 (enable, record), .6–.13 (shrink) | — |

## Amendments (2026-10-24, conductor)

1. **The survey is in.** `.flow/artifacts/fn240-crate-survey-2026-10-23.md` (commit d7e311b557) supersedes the 2026-10-22 grep in Goal & Context. Its findings that change this spec: neither crate has a layered module graph (one strongly connected component each); the owner's 8–12 target fits with 11 crates (3 helper-side, 8 host-side); both approved first extractions need the listed relocations, not `use`-path edits alone; `world` belongs in `wasm-sim`, not a `wasm-engine` crate; `engine` and `cluster` stay together on purpose. Task .1 is not closed: its build-time baseline (`cargo build --timings`, clean and incremental) has not been recorded, and that is what remains of R1.
2. **Ordering (owner cleanup-first amendment, 2026-10-24).** After the narrow fn-232.30 checkpoint, fn-242 completes the Rust driver implementation/cutover and duplicated-check reduction. Then complete this spec's full approved scope, including Amendment 4 and tasks .1–.13. Final qualification and receiving run once under fn-246 after both cleanup specs finish. fn-232 remains open for its original acceptance; no cycle pass or post-cycle window is a prerequisite to these cleanup implementations. Focused tests and conformance remain required throughout. The read-only survey already ran.
3. **`gomad-measure` size.** The survey's crate 10 (`measurement`, `stress`, `profile`) is about 19.7k production lines, over the context target. fn-242's measurement deduplication is expected to shrink it; task .4 measures it after fn-242 rather than cutting it blind.

## Amendment 4 (2026-10-24, owner): target crate and directory layout

The owner reviewed the 2026-10-23 survey and the current tree and fixed the end state. Tasks .2–.4 are unchanged in order; .4 now decides go/no-go against this target rather than choosing candidates. Three findings from the review drive the design:

- Below the engine, `wasmtime` is used only as `wasmtime::{Error, Result}` (28 sites in model, namespace, protocol, transcript, storage_transport, world, volume, network). One domain error type makes the whole lower half wasmtime-free.
- There are 309 `cfg(feature)` lines. `conformance` swaps out the library; `diagnostic-profile` is referenced from abi (24), engine (19), broker (12), volume (26), cache, protocol, reply, storage_transport, linked, linked_cluster, cluster execution; `determinism-fixture` and `stress-jitter` reach down from above. Features are being used as architecture.
- `wasmhost/src` has twelve `linked_*.rs` siblings and `gomad/src` ten `fixture_*`/`live_*` siblings that belong in directories.

### Target workspace

```
rust/
  Cargo.toml                 # workspace only; [workspace.dependencies], [workspace.lints]
  rust-toolchain.toml
  README.md                  # one paragraph per crate, dependency diagram
  crates/
    wasm-sim/                # L0  clock, network, volume, world (no wasmtime)
    wasm-hooks/              # L0  Observer + FaultHook traits with zero-sized no-op defaults
    wasm-model/              # L1  protocol, transcript, abi types, model, namespace, storage transport, metrics, module metadata
    wasm-diagnostic/         # L1  diagnostic profile schema v2; implements wasm-hooks::Observer
    wasm-engine/             # L2  the wasmtime boundary: engine, abi dispatch, cache, standalone linked API, linked_* helpers
    wasm-cluster/            # L3  handoff broker, cluster execution, cluster network, linked_cluster
    gomad-evidence/          # L3  records v2, store open/publish, symbolize, search context
    gomad-install/           # L4  install, config, backstop, slots, shared option parser, strict spec codec
    gomad-execution/         # L5  prepare, compilation cache, seed execution, ledger, launcher, scenario, shards
    gomad-shim/              # L5  pinned compiler wrapper and guest build classification
    gomad-replay/            # L6  replay, minimize
    gomad-qualify/           # L6  qualify, runtime_qualification
    gomad-sweep/             # L6  bounded WASM sweep developer command
    gomad-campaign/          # L7  exploration, counterfactual, simulation, choice, retention, failure policy
    gomad-measure/           # L8  measurement core: build identities, inputs, workloads, drivers, retention leases
    gomad-stress/            # L8  stress, profile (beside measure)
    gomad-fixture/           # L8  determinism-fixture controls and live backends; installs FaultHook impls
  bins/
    gomad/                   # gomad, gomad-measure, gomad-profile: argv parsing and dispatch only
    wasmhost/                # gomad-wasmhost reference helper (framed transport; feature reference-helper)
    wasm-conformance/        # Go-parity transport; links wasm-model only, never wasm-engine
  tools/
    driver/                  # Python qualification driver (fn-242 owns its removal)
    workload_tools/
    publish-executable.sh
    allocation_probe.rs
  docs/
    STRESS.md
```

Every crate has the same shape: `src/lib.rs` opening with the owns/refuses contract doc and `#![warn(missing_docs)]`, `tests/` holding boundary tests, `testdata/` beside the tests that read it. Inline tests only with a named invariant.

### Dependency graph

```
L0   wasm-sim        wasm-hooks
          \          /    \
L1        wasm-model      wasm-diagnostic
              |
L2        wasm-engine        <- the only crate whose public API mentions wasmtime
          /        \
L3  wasm-cluster   gomad-evidence
          \        /
L4       gomad-install
         /     |     \
L5 gomad-execution  gomad-shim
     /    |     \
L6 replay qualify sweep
     \    /
L7  gomad-campaign
     /      \
L8 measure  stress  fixture
        \     |     /
bins   gomad   wasmhost   wasm-conformance
```

Wide, not deep: two to four crates compile concurrently at each level. `wasmtime` is built once and only `wasm-engine` waits on it directly.

### Crate contracts

| Crate | Owns | Public surface | Refuses |
| --- | --- | --- | --- |
| `wasm-sim` | Deterministic simulated environment: clock, partitioned stream network, volume state and crash images, World scheduler model with its Go conformance corpora, the pure `linked_simulation` controller | `Clock`, `Network`, `Volume`, `World`, simulation `Controller`, their evidence encoders | Guests, wasmtime, Temporal semantics |
| `wasm-hooks` | Traits only: `Observer` (spans, owner attribution) and `FaultHook` (event order, jitter, fixture conditions), with zero-sized no-op defaults | Two traits, two defaults | Any implementation logic |
| `wasm-model` | The Temporal semantic model served to a guest: protocol framing, transcript, ABI message types, namespace state, replay plan, storage transport, metrics, module metadata, `HostError` | `Model::serve`, `Namespace`, frame codecs, `HostError` | wasmtime; executing anything |
| `wasm-diagnostic` | Profile schema v2 and span accounting | `Session: Observer` | Appearing in a qualification build |
| `wasm-engine` | Compiling and running one guest store: fuel, memory, epoch watchdog, output sinks, compile cache, ABI dispatch into the model, the standalone linked API and `linked_*` helpers | `linked::{Request, Limits, ModelInput, Prepared, Execution}`, `module_metadata` | Cluster scheduling; the host CLI |
| `wasm-cluster` | Multi-instance lifecycle: baton handoff, broker scheduling, shared cluster network, bytes-owned cluster execution | `linked_cluster` session API | Engine internals beyond the slot trait the engine exposes |
| `gomad-evidence` | Execution records v2, canonical bytes, failure grouping, authenticated store open/publish, symbolization, search context, the one base64 helper | `Record`, `Store`, `Opener`, `symbolize` | Campaign policy; replay and minimize logic |
| `gomad-install` | Installation discovery, config, process backstop, slot accounting, shared option parser, strict spec codec | `Installation`, `Config`, `Backstop`, `parse_explore_options`, `spec::strict` | Running guests |
| `gomad-execution` | Guest preparation, compilation cache, seed execution over the linked API, ledger, launcher, scenario, shards | `Preparation`, `run_seeds`, `Ledger`, `Scenario` | Exploration strategy |
| `gomad-shim` | The pinned compiler wrapper | `shim::run` | Anything else |
| `gomad-replay` | Tape admission and preflight, standalone and cluster replay, minimization with checkpoints | `replay::run`, `minimize::run` | CLI formatting beyond its JSON result |
| `gomad-qualify` | One-cell qualification, cost model, retained Result reader, repository gate seam | `qualify`, `report`, `runtime_qualification` | Measurement and stress |
| `gomad-sweep` | The bounded sweep developer command and classifier | `sweep::run` | Anything else |
| `gomad-campaign` | Exploration rounds, choice policy, counterfactuals, simulation, retention, failure policy | `explore`, `resume`, `merge`, `execute_shard` | Prepare and ledger internals |
| `gomad-measure` | Measurement input catalogue, build identities, workloads, warm and namespace drivers, retention leases | `measure`, `retention::{RetainedRun, ChildLease}` | Stress recipes |
| `gomad-stress` | Seeded perturbations, cluster stress oracles, the stress and cluster drivers, diagnostic profile capture | `stress::run`, `profile` | Qualification evidence |
| `gomad-fixture` | Determinism fixture controls and live backends | `install_hooks()` | Shipping in a release build |

### Relocations the layout requires

1. **`HostError`.** Replace `wasmtime::{Error, Result}` below the engine with a `HostError` in `wasm-model`; the engine converts at its boundary. Precondition for `wasm-sim`, `wasm-model` and the `wasm-conformance` bin.
2. **Invert `engine`→`cluster`** (7 sites: the store's optional broker member and network endpoint become a slot trait the cluster crate implements; the "cluster requested" frame dispatch moves to the `wasmhost` bin). The survey recommended keeping them merged; the owner takes the split so that a single-store engine does not know about brokers.
3. **Split `linked_simulation`**: the pure controller (about 385 lines) to `wasm-sim`, the execute half stays in the engine.
4. **Hooks instead of features**: every `diagnostic`, `event_order`, `stress_jitter` and `determinism_fixture` reference in a library crate becomes a call on an `Observer`/`FaultHook` held in state; release builds pass the zero-sized defaults so monomorphization keeps it free.
5. **Evidence back-edges**: `minimize::source`, `campaign::origin`, `exploration::site_scope`, `choice::hex` move into `gomad-evidence`; the two cross-module test files move up.
6. **CLI back-edges**: `cli::{install,replay}::run` and `retain_observed` become library functions in their owning crates; the option parser moves to `gomad-install`; `shim` and `sweep` become crates so the bin crate is dispatch only.
7. **Measure/stress cycle**: `measurement::stress_driver` and `cluster_driver`'s stress calls move into `gomad-stress`; `retention` stays in `gomad-measure`.
8. **One base64 helper** replaces `campaign::exploration::base64` and `qualify::cells::base64`.

### Cross-cutting rules

- **Typed errors travel with each extraction (R11).** The crate being extracted replaces `Result<_, String>` in its public API with its own error enum in the same task; `map_err(|error| error.to_string())` chains at that boundary become `From` impls. Evidence 2026-10-24: 1548 `Result<_, String>` signatures, 900 `to_string` map_errs, 1699 distinct messages, four error enums in the tree.
- **Size lints (R12).** Task .5 turns on `too_many_lines` and `cognitive_complexity` with the workspace lints and records today's allow-list (known offenders: `sweep` nesting to depth 20, `measurement::workloads::run_inner` 336 lines, `qualify::support::build` 190 lines). Later tasks shrink it.
- Identity closure (R7) and the inventory scanner are extended before the first move (task .5), with the flip-one-byte regression test.
- Clippy lints and pinned versions move to `[workspace.lints]` and `[workspace.dependencies]`; crates inherit with one line.
- Remaining Cargo features: `reference-helper` (helper bin) and `guest-profile` (`wasm-engine`, for wasmtime's own profiling flags). Everything else is a crate or a hook.
- Makefile: `--workspace` for fmt, clippy and tests; bins built by package name. `lint-rust` covers every package.

### Decisions recorded

- **Eighteen crates and bins against the 8–12 figure.** The owner prefers clear boundaries over the count. If the count must fall, fold in this order with least damage: `gomad-shim` and `gomad-sweep` into the `gomad` bin, `gomad-stress` into `gomad-measure`, `gomad-fixture` into the `gomad` bin behind one hook registry, `wasm-hooks` into `wasm-sim`, the `wasmhost` bin into `wasm-cluster`. That lands at twelve with the same layering.
- **Engine/cluster split** is taken; keeping the survey's merged 17k crate is the recorded fallback if .8 shows the inversion costs more than about twenty sites.
- **Hooks over `cfg`**: if release binaries must provably exclude diagnostic code rather than monomorphize it away, one `cfg` stays at the hook installation site in the bin crate.
