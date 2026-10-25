# Native development tools

Build this host-only tool before setting guest compilation flags:

```sh
cd model
mise exec -- cargo build -p xtask --bin xtask --profile dev --locked
mise exec -- cargo xtask guest-lock Cargo.lock build/identities < metadata.json
```

The alias selects the development profile; callers must bootstrap without a guest `--target`, `RUSTC_WRAPPER`, `RUSTFLAGS` or `CARGO_ENCODED_RUSTFLAGS`. Subsequent compiler invocations call the built native executable directly as `xtask guest-rustc <rustc> <arguments...>`. They never launch Cargo recursively. `model/scripts/xtask.sh` now owns bootstrap and prints the absolute executable path under `<target>/xtask-bootstrap/debug/xtask`. Guest launchers copy that native executable atomically as `guest-rustc`, which dispatches by argv[0] before parsing command arguments.

`guest <crate> <output> [--check]` builds or compares a guest. `guest-freshness <crate> <output>` validates pinned tools, metadata, Rust sources and target readiness before comparing. Both run from `model/`; shell callers use `guest.sh` and `guest-freshness.sh` to preserve caller-relative output paths. Missing tools fail for every CI value. See [GUEST_BUILDS.md](../../GUEST_BUILDS.md) for native JSON admission, statuses and bootstrap environment.

`guest-lock <Cargo.lock> <directory>` reads Cargo metadata JSON from stdin. It emits the published map's OS path followed by a newline. Publication uses a sibling temporary file, complete write and atomic rename. Its content-derived filename isolates different lock maps sharing a target directory. The native map contains `schema: 2` and `closures`, an object from `name version` to a lowercase SHA-256 string or null for ambiguous coordinates. Each digest covers native schema, lock version, sorted portable name/version/source coordinates, checksums, and every reachable non-development edge, including build and platform-conditional edges. Metadata's feature-unified graph is authoritative; host-only additions that enable shared optional edges may intentionally affect a closure. Checkout-bearing Cargo IDs never enter its digest.

`guest-rustc` accepts raw OS arguments, passes probes and ordinary host compilation through, and normalizes wasm32-wasip1 and proc-macro identities. It inherits the environment and jobserver descriptors. `UMPIRE_GUEST_PREVIOUS_WRAPPER`, when nonempty, executes before the compiler. Existing `UMPIRE_GUEST_{WORKSPACE,SYSROOT,CARGO_HOME,TARGET,LOCK,COMPILER}` and Cargo package/manifest/OUT_DIR environment values supply the compilation context. A successful target compilation replaces Cargo's metadata hash with its native portable identity while retaining Cargo's output filename; native macro arguments remain intact. Sidecars publish before the child starts for Cargo pipelining. A per-output file lock serializes publication, child completion and failure cleanup; persistent lock files must stay in place. Launch or compiler failure removes the sidecar. Compiler exit statuses pass through, with Unix signals represented as 128 + signal; driver errors return 1 with an `xtask:` diagnostic.

Native schema 2 intentionally replaces Python identity JSON, hash inputs, sidecars, formatting and diagnostics. Old schema-1 sidecars are invalid; guest orchestration invalidates its cache using the native binary, lockfile and compiler hashes. Sidecars contain `schema`, `identity` and transitive `host_variants`; hashes are exactly 64 lowercase hex digits. Compiler identity inputs preserve arbitrary Unix OS bytes without lossy Unicode conversion. Paths are lexically normalized under declared roots, with component-boundary matching. Host paths and Cargo host metadata are excluded from portable identity; dependency identities and native-macro collision detection remain mandatory. Standard-library exemptions require wasm, Cargo's build-std marker and either pinned sysroot sources or matching entries in rust-src's library lockfile.

Serialization passes through plain serde_json `Value` so nested object keys use the default BTreeMap ordering. `preserve_order` is disabled workspace-wide. SHA-256 uses in-process sha2 0.11. Native TOML is exactly the admitted toml 1.1.8+spec-1.1.0 with only std/parse/serde requested by xtask. Other existing workspace consumers retain their pre-existing TOML features. No unbounded feature or Python grammar, number, date, recursion, escaping or diagnostic-order emulation is provided. JSON/TOML parse and schema failures are errors; native parser resource limits apply. Current workspace Cargo, mise and Rust toolchain documents and the installed rust-src lockfile are controls in the focused native corpus.

Run the task's focused controls:

```sh
cd model
mise exec -- cargo test -p xtask --locked
mise exec -- cargo clippy -p xtask --all-targets -- -D warnings
```

The canonical workspace gate will own these tests once, without a second package invocation. The conductor owns final joined-input guest publication, two-clean-build reproducibility, committed-asset freshness, host parity and the integrated TLC/Apalache pipeline. Linux process controls cannot establish native macOS loader behavior; inherited loader bytes are checked on the available host, with native macOS execution unclaimed.
