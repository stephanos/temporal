# model: Umpire in Rust

The Umpire framework as a Cargo workspace mirroring `model/archived/framework`'s layout. Its current scope is
what `model/archived/framework/samples/Kettle.scala` uses today; see `options/README.md` for the
authoring-surface options under discussion.

`archived/` holds the former Scala tree unchanged for reference, including its retained build
caches. Its internal paths describe the old layout, and its Scala targets are unmaintained and
not expected to build. The Cargo workspace and Rust gate do not build the archive.

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
make model-check   # fmt --check, clippy -D warnings on all targets, docs with warnings denied, tests
make model-fmt     # rustfmt the workspace
make model-time    # time the actual gate; warns over the 3 s inner-loop and 30 s gate budgets (fn-210 R30)
```

The gate also checks resolved core capability restrictions and their negative controls,
protobuf freshness, the wasm32-wasip1 core and guest builds, and CGO-disabled Go host parity.
It provisions the pinned `protoc-gen-go v1.36.6` through the repository's existing Make
target on first use; warm runs need no network. Rust generation uses prost-build and
the mise-pinned protoc 29.5. To regenerate Go after editing the single schema, run
`bash model/scripts/proto-go.sh` from the repository root.

The guest's three application exports are `umpire_buffer_alloc`, `umpire_buffer_release`
and `umpire_call`; see
`umpire/proto/README.md` for ownership and refusal contracts, and `umpire/hosts/go/README.md`
for reproduction and measurements. Rust exports these functions directly with the stock
linker. Go checks the built module's exact application exports and signatures, including
negative controls for missing, unexpected and mis-typed exports.
The ABI crate keeps the workspace lint rules with `unsafe_code = deny` and local
exceptions only on the three named export attributes. DSL/checker retain `forbid`.

From inside `model/`, `cargo check-all` and `cargo doc-all` are aliases for the clippy and doc steps.
`model-time` measures the actual `make model-check`, including every current gate step. Its inner
loop defaults to touching `umpire/checker/src/lib.rs` and testing `umpire-checker`; select another
crate and source with `UMPIRE_INNER_CRATE` and `UMPIRE_INNER_SOURCE` (relative to `model/`). Timed
command failures fail the helper; exceeding a time budget prints a warning and succeeds.

## Writing a Model

Inside a `#[machine]` module, one `impl Model` holds methods with per-item attributes.
`#[on(Actor)]` selects an actor enum derived with `Action`; its method takes that actor type
and matches its actions. Kettle's user transitions are:

```rust
impl Model {
    #[on(User)]
    fn user(&mut self, action: User) {
        match action {
            User::PressButton if !self.heating_on => {
                self.heating_on = true;
                self.beeping = false;
            }
            User::DepressButton if self.heating_on => {
                self.heating_on = false;
                self.beeping = false;
            }
            _ => {}
        }
    }
}
```

Each match arm combines the action, its guard and its effect. An action is enabled only when the
final state changes; `_ => {}` disables unmatched actions. Without a fallback, rustc checks match
exhaustiveness. Attributes preserve authored statements and method/impl headers.

Methods of each kind are contiguous, in `#[on]`, optional `#[effect]`, `#[invariant]`,
`#[postcondition]`, `#[assumption]`, `#[progress]` order. A postcondition may filter its action
with `#[postcondition(after = Heater::Failover)]`. Shared effects are ordinary `#[effect]`
helper methods; Kettle keeps its effects inline. Only `#[on]` and `#[effect]` methods may take
`&mut self`. Other `Model` methods take `&self` or no receiver, including unmarked helpers in
qualified, nested or aliased impls; compiler-resolved checks leave unrelated helper types alone.
Compiler-only checking declarations may be added inside local function or const scopes to resolve
aliases and trait projections; they add no runtime guard.
`#[machine]` forbids unsafe code throughout its module.

Attribute progress methods return `LeadsTo`. The existing
`whenever(..).eventually(..).assuming(..)` builder remains; `path!` is deferred to fn-215.
See [DESIGN.md](DESIGN.md) for the Scala/Rust
comparison and capability boundaries.

## Testing

End to end, by golden files (fn-210 R8, R23, R29). A test is a fixture, an input plus the output it
produces (`model.rs` and `expected/`). "Fixture", not
"case": Case is Testpilot's term.

- behaviour: each fixture's `expected/report.snap.txt`, `states.snap.txt`, `rows.snap.txt` and `paths/<claim>.snap.txt`
  (`insta`; review with `cargo insta review`);
- refusals: `trybuild` `.rs` + `.stderr` under `umpire/dsl/tests/ui/`;
- expansion: insta snapshots of each macro's implementation function, formatted by
  `prettyplease`;
- one property test against a brute-force reference explorer, the check that goldens are right
  rather than merely stable.

Goldens are deterministic, human-readable and timing-free. `make model-update` rewrites them; a
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
fixture fails naming its folder, including during updates. Seed a new fixture with a snapshot
payload such as `report.snap.txt` before running `make model-update`; renderers and consumer
registrations still use logical names such as `report.txt`.

Behaviour goldens use insta binary snapshots with text payloads. Consumers read `report.snap.txt`,
`states.snap.txt`, `rows.snap.txt` and `paths/<claim>.snap.txt` directly; the adjacent `.snap`
stores insta's review metadata. `cargo insta review` (or `cargo insta accept`) updates the payloads.
Run `make model-update` to create or refresh snapshots.
Metadata, payloads and pending snapshots belong to one logical expected file. Non-text
extensions use insta's snapshot suffix to avoid collisions (`Model@tla.snap` and
`Model@tla.snap.tla` back `Model.tla`; `Model@cfg.snap` separately backs `Model.cfg`). A pending
snapshot is accepted with `cargo insta accept`, then checked by the same runner.
The gate needs only the Rust toolchain and workspace dependencies. The optional `cargo-insta`
CLI supports interactive review and acceptance (`cargo install cargo-insta --locked`). Its real
pending-to-accept integration control runs explicitly from `model/` with
`mise exec -- cargo test -p umpire-testing --test consumers accepts_generic_pending_snapshots_with_cli -- --ignored`.

## Dependencies

Only admitted crates (fn-210 R18): a release in the last 12 months, or finished and very widely used.
They are declared once in `[workspace.dependencies]`; a crate opts in with `name.workspace = true`.
`deny.toml` bans the unmaintained alternatives (`im`, `derivative`, `proc-macro-error`,
`proc-macro-error2`, `deluxe`, `index_vec`, `stateright`); `cargo deny check bans` enforces it. A new
dependency needs the same check and a line in the spec.
