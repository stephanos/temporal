# model2: Umpire in Rust

The Umpire framework as a Cargo workspace mirroring `model/framework`'s layout. Its current scope is
what `model/framework/samples/Kettle.scala` uses today; see `options/README.md` for the
authoring-surface options under discussion.

The toolchain is pinned in the repository's `mise.toml` (`rust = "1.99.1"`); run through mise:

```
mise exec -- cargo build --workspace
mise exec -- cargo test --workspace
```

## Hygiene

One lint policy for all crates lives in the workspace `Cargo.toml` (`[workspace.lints]`: clippy
`all` and `pedantic` at warn, a few pedantic lints that fight the Model style allowed, `unwrap_used`,
`expect_used`, `dbg_macro`, `todo` and `print_*` warned, `unsafe_code` forbidden, `missing_docs` and
`unreachable_pub` warned). Each crate opts in with `[lints] workspace = true`. Thresholds are in
`clippy.toml`, formatting in `rustfmt.toml` (stable options only, 2024 style, 100 columns).
`rust-toolchain.toml` matches the mise pin and names the components (rustfmt, clippy, rust-src,
rust-analyzer). `deny.toml` is the cargo-deny policy for licenses, duplicate versions, sources and
advisories; it needs the tool installed and the network.

The gate, from the repository root (`temporal/`):

```
make model2-check   # fmt --check, clippy -D warnings on all targets, docs with warnings denied, tests
make model2-fmt     # rustfmt the workspace
make model2-time    # time the actual gate; warns over the 3 s inner-loop and 30 s gate budgets (fn-210 R30)
```

The gate also checks resolved core capability restrictions and their negative controls,
protobuf freshness, the wasm32-wasip1 core and guest builds, and CGO-disabled Go host parity.
It provisions the pinned `protoc-gen-go v1.36.6` through the repository's existing Make
target on first use; warm runs need no network. Rust generation uses prost-build and
the mise-pinned protoc 29.5. To regenerate Go after editing the single schema, run
`bash model2/scripts/proto-go.sh` from the repository root.

The guest's three application exports are `alloc`, `free` and `call`; see
`umpire/proto/README.md` for ownership and refusal contracts, and `hosts/go/README.md`
for reproduction and measurements. Stock WASI libc owns the internal `free` symbol.
The target linker wrapper changes the guest's distinct `umpire_free` export name to
`free` without changing code or other sections; its regression controls run in the gate.
The ABI crate keeps the workspace lint rules with `unsafe_code = deny` and local
exceptions only on the three named export attributes. DSL/checker retain `forbid`.

From inside `model2/`, `cargo check-all` and `cargo doc-all` are aliases for the clippy and doc steps.
`model2-time` measures the actual `make model2-check`, including every current gate step. Its inner
loop defaults to touching `umpire/checker/src/lib.rs` and testing `umpire-checker`; select another
crate and source with `UMPIRE_INNER_CRATE` and `UMPIRE_INNER_SOURCE` (relative to `model2/`). Timed
command failures fail the helper; exceeding a time budget prints a warning and succeeds.

## Testing

End to end, by golden files (fn-210 R8, R23, R29). A test is a fixture, an input plus the output it
produces (`model.rs` and `expected/`). "Fixture", not
"case": Case is Testpilot's term.

- behaviour: each fixture's `expected/report.txt`, `states.txt`, `rows.txt` and `paths/<claim>.txt`
  (`insta`; review with `cargo insta review`);
- refusals: `trybuild` `.rs` + `.stderr` under `umpire/dsl/tests/ui/`;
- expansion: insta snapshots of each macro's implementation function, formatted by
  `prettyplease`;
- one property test against a brute-force reference explorer, the check that goldens are right
  rather than merely stable.

Goldens are deterministic, human-readable and timing-free. `make model2-update` rewrites them; a
fixture folder without `model.rs` or without `expected/` fails the build. Unit tests only when a defect cannot be
written as a fixture.

`umpire-tests` at `umpire/tests/` owns the eight shared Models in `fixtures/<name>/`: `kettle`,
`kettle_trait`, `kettle_cut`, `choices`, `deep_chain`, `edge_checks`, `shared_starts`, and `vacuity`.
Its sole test target is `fixtures.rs`. The Kettle Models live in their own fixture's `model.rs`;
native and wasm hosts can include that source by path. The trait Kettle has executable native
report, states and rows goldens. Checker-specific failures, panics, overflows, holes, budget cuts
and refusals remain in `umpire/checker/tests/fixtures/`. DSL spelling fixtures remain with the DSL.
Authoring sketches stay in `options/` through the design comparison; there is no samples crate.

`umpire-testing` is a development-only dependency. It owns rendering, comparison, snapshot
updates, discovery and the consumer registry. Build scripts share its dependency-free discovery
source, avoiding a checker dependency cycle. Fixtures are discovered in name order and must
contain `model.rs`, `expected/`, and at least one consumer-owned expected file. Expected files
select consumers: the checker owns `report.txt`, `states.txt`, `rows.txt`, `negative_report.txt`
and `paths/*.txt`; a selected consumer renders and checks its outputs twice for determinism.
The DSL registers its spelling consumer for `values.txt`. Future backends register their own
filenames and renderer. An unregistered filename, an orphan snapshot artifact, or an empty
fixture fails naming its folder, including during updates. Seed a new fixture with a declared
logical output filename before running `make model2-update`.

Behaviour goldens use insta binary snapshots with text payloads. Each `report.txt`, `states.txt`,
`rows.txt`, or `paths/<claim>.txt` is a symlink to its `.snap.txt` payload; the adjacent `.snap`
stores insta's review metadata. This keeps the required text paths readable by other hosts and
lets `cargo insta review` (or `cargo insta accept`) update their single source of expected content.
Run `make model2-update` to create or refresh snapshots and their aliases together.
Metadata, payloads, aliases and pending snapshots belong to one logical expected file. Non-text
extensions use insta's snapshot suffix to avoid collisions (`Model@tla.snap` and
`Model@tla.snap.tla` back `Model.tla`; `Model@cfg.snap` separately backs `Model.cfg`). A pending
snapshot is accepted with `cargo insta accept`, then checked by the same runner.
The gate needs only the Rust toolchain and workspace dependencies. The optional `cargo-insta`
CLI supports interactive review and acceptance (`cargo install cargo-insta --locked`). Its real
pending-to-accept integration control runs explicitly from `model2/` with
`mise exec -- cargo test -p umpire-testing --test consumers accepts_generic_pending_snapshots_with_cli -- --ignored`.

## Dependencies

Only admitted crates (fn-210 R18): a release in the last 12 months, or finished and very widely used.
They are declared once in `[workspace.dependencies]`; a crate opts in with `name.workspace = true`.
`deny.toml` bans the unmaintained alternatives (`im`, `derivative`, `proc-macro-error`,
`proc-macro-error2`, `deluxe`, `index_vec`, `stateright`); `cargo deny check bans` enforces it. A new
dependency needs the same check and a line in the spec.
