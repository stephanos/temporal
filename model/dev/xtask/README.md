# Native development tools

Build this host-only tool before setting guest compilation flags:

```sh
cd model
mise exec -- cargo build -p xtask --bin xtask --profile dev --locked
mise exec -- cargo xtask guest-lock Cargo.lock build/identities < metadata.json
```

The alias selects the development profile; callers must bootstrap without a guest `--target`, `RUSTC_WRAPPER`, `RUSTFLAGS` or `CARGO_ENCODED_RUSTFLAGS`. Subsequent compiler invocations call the built native executable directly as `xtask guest-rustc <rustc> <arguments...>`. They never launch Cargo recursively. Task .5 switches guest orchestration; the Python helpers and their existing callers stay active until then.

`guest-lock <Cargo.lock> <directory>` reads Cargo metadata JSON from stdin. It emits the published map's OS path followed by a newline. Publication uses a sibling temporary file, complete write and atomic rename. Its content-derived filename isolates different lock maps sharing a target directory. The native map contains `schema: 2` and `closures`, an object from `name version` to a lowercase SHA-256 string or null for ambiguous coordinates. Each digest covers native schema, lock version, sorted portable name/version/source coordinates, checksums, and every reachable non-development edge, including build and platform-conditional edges. Metadata's feature-unified graph is authoritative; host-only additions that enable shared optional edges may intentionally affect a closure. Checkout-bearing Cargo IDs never enter its digest.

`guest-rustc` accepts raw OS arguments, passes probes and ordinary host compilation through, and normalizes wasm32-wasip1 and proc-macro identities. It inherits the environment and jobserver descriptors. `UMPIRE_GUEST_PREVIOUS_WRAPPER`, when nonempty, executes before the compiler. Existing `UMPIRE_GUEST_{WORKSPACE,SYSROOT,CARGO_HOME,TARGET,LOCK,COMPILER}` and Cargo package/manifest/OUT_DIR environment values supply the compilation context. A successful target compilation replaces Cargo's metadata hash with its native portable identity while retaining Cargo's output filename; native macro arguments remain intact. Sidecars publish before the child starts for Cargo pipelining. A per-output file lock serializes publication, child completion and failure cleanup; persistent lock files must stay in place. Launch or compiler failure removes the sidecar. Compiler exit statuses pass through, with Unix signals represented as 128 + signal; driver errors return 1 with an `xtask:` diagnostic.

Native schema 2 intentionally replaces Python identity JSON, hash inputs, sidecars, formatting and diagnostics. Old schema-1 sidecars are invalid; the orchestration successor must invalidate its old cache. Sidecars contain `schema`, `identity` and transitive `host_variants`; hashes are exactly 64 lowercase hex digits. Compiler identity inputs preserve arbitrary Unix OS bytes without lossy Unicode conversion. Paths are lexically normalized under declared roots, with component-boundary matching. Host paths and Cargo host metadata are excluded from portable identity; dependency identities and native-macro collision detection remain mandatory. Standard-library exemptions require wasm, Cargo's build-std marker and either pinned sysroot sources or matching entries in rust-src's library lockfile.

Serialization passes through plain serde_json `Value` so nested object keys use the default BTreeMap ordering. `preserve_order` is disabled workspace-wide. SHA-256 uses in-process sha2 0.11. Native TOML is exactly the admitted toml 1.1.8+spec-1.1.0 with only std/parse/serde requested by xtask. Other existing workspace consumers retain their pre-existing TOML features. No unbounded feature or Python grammar, number, date, recursion, escaping or diagnostic-order emulation is provided. JSON/TOML parse and schema failures are errors; native parser resource limits apply. Current workspace Cargo, mise and Rust toolchain documents and the installed rust-src lockfile are controls in the focused native corpus.

Run the task's focused controls:

```sh
cd model
mise exec -- cargo test -p xtask --test guest_identity --locked
mise exec -- cargo clippy -p xtask --all-targets -- -D warnings
```

The canonical workspace gate will own these tests once, without a second package invocation. Whole-guest regeneration, freshness and the integrated TLC/Apalache pipeline belong to later tasks. Linux process controls cannot establish native macOS loader behavior; inherited loader bytes are checked on the available host, with native macOS execution unclaimed.

## Apalache installer

`fetch-apalache [workspace-root]` admits the official pinned Apalache jar. The
optional root defaults to the enclosing repository root identified by
`model/Cargo.toml`, or to the current directory outside a checkout. The root
contains `model/`; missing cache components are created. Success
prints the absolute jar path followed by a newline and exits 0. Failures exit 1
with `xtask: <native error>` on stderr; curl can also write its own diagnostics.
Paths are OS arguments, including non-UTF-8 paths on Unix. No Python diagnostic,
exception type or output encoding compatibility is promised.

The installer requires curl and Unix filesystem admission. It stores the jar at
`model/build/apalache/apalache-<version>/lib/apalache.jar`, locks a persistent
regular `.install.lock` using `File::lock()`, and refuses symlink components,
wrong existing jars and partial installations. Download and validation use a
private 0700 staging directory. The archive hash is checked before gzip/tar
parsing, the exact regular jar member is hashed before publication, and the
validated directory is renamed into place while holding the lock. Staging is
removed on success and failure. Installation pins live only in
`src/apalache_pins.rs`, included by the runtime backend as well as xtask.
The production command accepts no URL, version, archive or hash overrides.
