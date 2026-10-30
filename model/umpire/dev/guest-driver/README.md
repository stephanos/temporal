# Native guest driver

Build this host-only tool before setting guest compilation flags:

```sh
cd model/umpire
mise exec -- cargo build -p umpire-guest-driver --bin umpire-guest-driver --profile dev --locked
mise exec -- cargo run -p umpire-guest-driver --bin umpire-guest-driver -- guest-lock Cargo.lock build/identities < metadata.json
```

The alias selects the development profile; callers must bootstrap without a guest `--target`, `RUSTC_WRAPPER`, `RUSTFLAGS` or `CARGO_ENCODED_RUSTFLAGS`. Subsequent compiler invocations call the built native executable directly as `umpire-guest-driver guest-rustc <rustc> <arguments...>`. They never launch Cargo recursively. `scripts/xtask.sh` owns bootstrap and prints the absolute xtask path; the driver is its sibling `<target>/xtask-bootstrap/debug/umpire-guest-driver`. Guest launchers copy that native executable atomically as `guest-rustc`, which dispatches by argv[0] before parsing command arguments.

`guest <crate> <output> [--check]` builds or compares a guest. `guest-freshness <crate> <output>` validates pinned tools, metadata, Rust sources and target readiness before comparing. Both use the caller Cargo workspace. From `model/`, `mise exec -- cargo xtask guest` and `mise exec -- cargo xtask guest-freshness` route the five framework packages to `umpire/` and other packages to the system workspace, preserving caller-relative output and target paths. Already bootstrapped callers can execute the driver directly from their workspace. Missing tools fail for every CI value. See [native CLI](../xtask/README.md) for native JSON admission, statuses and bootstrap environment.

`guest-lock <Cargo.lock> <directory>` reads Cargo metadata JSON from stdin. It emits the published map's OS path followed by a newline. Publication uses a sibling temporary file, complete write and atomic rename. Its content-derived filename isolates different lock maps sharing a target directory. The native map contains `schema: 2` and `closures`, an object from `name version` to a lowercase SHA-256 string or null for ambiguous coordinates. Each digest covers native schema, lock version, sorted portable name/version/source coordinates, checksums, and every reachable non-development edge, including build and platform-conditional edges. Metadata's feature-unified graph is authoritative; host-only additions that enable shared optional edges may intentionally affect a closure. Checkout-bearing Cargo IDs never enter its digest.

`guest-rustc` accepts raw OS arguments, passes probes and ordinary host compilation through, and normalizes wasm32-wasip1 and proc-macro identities. It inherits the environment and jobserver descriptors. `UMPIRE_GUEST_PREVIOUS_WRAPPER`, when nonempty, executes before the compiler. Existing `UMPIRE_GUEST_{TOOL_ROOT,WORKSPACE,SYSROOT,CARGO_HOME,TARGET,LOCK,COMPILER}` and Cargo package/manifest/OUT_DIR environment values supply the compilation context. A successful target compilation replaces Cargo's metadata hash with its native portable identity while retaining Cargo's output filename; native macro arguments remain intact. Sidecars publish before the child starts for Cargo pipelining. A per-output file lock serializes publication, child completion and failure cleanup; persistent lock files must stay in place. Launch or compiler failure removes the sidecar. Compiler exit statuses pass through, with Unix signals represented as 128 + signal; driver errors return 1 with an `umpire-guest-driver:` diagnostic.

Native schema 2 intentionally replaces Python identity JSON, hash inputs, sidecars, formatting and diagnostics. Old schema-1 sidecars are invalid; guest orchestration invalidates its cache using the native binary, lockfile and compiler hashes. Sidecars contain `schema`, `identity` and transitive `host_variants`; hashes are exactly 64 lowercase hex digits. Compiler identity inputs preserve arbitrary Unix OS bytes without lossy Unicode conversion. Paths are lexically normalized under declared roots, with component-boundary matching. Host paths and Cargo host metadata are excluded from portable identity; dependency identities and native-macro collision detection remain mandatory. Standard-library exemptions require wasm, Cargo's build-std marker and either pinned sysroot sources or matching entries in rust-src's library lockfile.

Serialization passes through plain serde_json `Value` so nested object keys use the default BTreeMap ordering. `preserve_order` is disabled workspace-wide. SHA-256 uses in-process sha2 0.11. Native TOML is exactly the admitted toml 1.1.8+spec-1.1.0 with only std/parse/serde requested by xtask. Other existing workspace consumers retain their pre-existing TOML features. No unbounded feature or Python grammar, number, date, recursion, escaping or diagnostic-order emulation is provided. JSON/TOML parse and schema failures are errors; native parser resource limits apply. Current workspace Cargo, mise and Rust toolchain documents and the installed rust-src lockfile are controls in the focused native corpus.

Run the task's focused controls:

```sh
cd model/umpire
mise exec -- cargo test -p umpire-guest-driver --locked
mise exec -- cargo clippy -p umpire-guest-driver --all-targets -- -D warnings
```

The driver has no dependency on the general CLI. The CLI builds this package and executes its native binary for guest operations. Cache namespaces hash these exact driver bytes, the full current caller lockfile and compiler input; unrelated compiled CLI code is outside those inputs.

`context::ToolRoot` admits `UMPIRE_TOOL_ROOT` when supplied, otherwise the framework root that compiled the driver. It requires local Cargo, mise and Rust-toolchain files and never searches ancestors. Tool execution and Cargo launches use these local pins; metadata/build receive the caller's explicit manifest path. Relative outputs and target overrides resolve against the original caller workspace before tool execution.

The raw OS context codec serves orchestration and the compiler. A single canonical longest-root mapping covers the caller workspace, separately nested framework sources, sysroot, registry and target. Compiler identity normalization and rustc flags consume that mapping; undeclared source roots refuse. The framework mapping is omitted when it equals the caller workspace.

Cross-checkout host compiler caching remains deferred: evaluated sccache 0.19.0 does
not apply `SCCACHE_BASEDIRS` to Rust cache keys. Hash-only path normalization also
risks retaining another checkout's embedded `CARGO_MANIFEST_DIR`, which supplies this
driver's default tool root. Keep native bootstrap wrapper isolation and ordinary
incremental compilation; task fn-254.19 retains the reproduction and unmet controls
in its [investigation](../../../../.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task19/INVESTIGATION.md).
