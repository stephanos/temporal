# Native identity harness

`identity_harness.execute(config)` is one closed harness building block below
`GLOBAL/stages/ordinary/identity-harness`. It supplies facts for the five original
identity roots. It does not implement the complete ordinary, provider, functional
or retained-cluster stage.

Before freezing, provision the GNU compiler envelope with the actual static tool
plan and canonical native GCC executable:

```sh
python3 tools/gomad/rust/driver/native_gnu.py prepare \
  --tools-plan /absolute/static-plan.json \
  --linker /usr/bin/aarch64-linux-gnu-gcc-15 \
  --root /absolute/fresh-gnu-envelope
```

The final binding is `identity_harness: {toolchain_plan:
/absolute/fresh-gnu-envelope/gnu-toolchain.json}`. The freezer expands that plan,
its actual raw probes, source controller, aliases, Rust distribution and GNU
compiler/delegate/header/library/loader closure, plus the offline Cargo registry
cache. No harness artifact or future output hash appears in these inputs.

The GNU envelope supports native Linux aarch64/x86_64 system GCC, with the default
system sysroot. It derives the compiler's machine, specs, internal tools, program
and library searches and C/C++ header searches from fixed real probes. It captures
loader configuration and the actual ELF interpreter and DT_NEEDED closure,
including matching native loader-cache candidates and `$ORIGIN` searches. Changed
delegates, startup objects, libraries, headers, loader configuration or symlink
aliases refuse independently. Global loader preloads and unsupported toolchain
shapes refuse. This restriction applies to this source-build recipe; the product's
performance-host policy remains separate.

The final managed recipe invokes the pinned Rust 1.99.1 Cargo executable with:

```text
test --release --locked --offline --no-default-features
-p gomad-wasmhost --test implementation_identity --no-run --message-format=json
--manifest-path REPO/tools/gomad/rust/Cargo.toml
```

The owned target is fresh. The closed environment fixes the native compiler,
linker, archive tools, offline cache, one Cargo job, no incremental compilation,
empty wrappers/flags, private HOME/TMPDIR and a PATH containing the sealed Rust
and GNU tools. Existing `driver.command` callers retain their prior environment
semantics; only this recipe requests an exact environment. Ambient environments
are not copied into receipts.

The reader authenticates the actual Cargo test artifact's package, source,
release/test profile, empty features and owned executable path. It compares the
build script's source inventory, compiler stdout and feature/target captures with
frozen files and separately retained compiler queries. It reopens the admitted
copy and every actual process journal and parent/monotonic interval, then rederives
all eighteen raw source/compiled/owner-mutation/missing-owner observations from the
five fixed tests. The derived executable admission remains local to this cycle;
it never changes the frozen input table. No caller supplies a test command,
compiler string, feature context, future binary digest or PASS row.
