# model: Umpire in Rust

The Cargo workspace contains the system-independent framework under `umpire/` and the
Temporal extension under `temporal/`. The Kettle fixtures establish the generic authoring
and checker surface; the activity slice additionally hosts one real SDK run through a Model.
See [DESIGN.md](DESIGN.md) for the scope, measurements and retained authoring decisions.

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
make model-time    # time the actual gate; note overruns of 10 s inner-loop / 30 s gate soft targets (R30)
```

The gate also checks resolved core capability restrictions and their negative controls,
the generic source boundary, both protobuf modules' freshness, the wasm32-wasip1 core
and guest builds, and CGO-disabled Go host parity for both modules.
It provisions the pinned `protoc-gen-go v1.36.6` through the repository's existing Make
target on first use; warm runs need no network. Rust generation uses prost-build and
the mise-pinned protoc 29.5. To regenerate Go after editing the single schema, run
`bash model/scripts/proto-go.sh` from the repository root. The three Temporal envelopes
use `bash model/scripts/proto-temporal-go.sh`; both scripts support `--check`.

`umpire/` owns the DSL, checker, reusable testing tools, Model protocol, transactional
runtime/ABI library, stock/control guests and transport-only Go host. `temporal/` owns
the activity Model and goldens, generated upstream API closure, realization, guest and
its separate Go envelope module. The source scanner is generic; its deny policy and
negative controls live in `temporal/testing/boundary.rs`, outside the scanned tree.
Every `.rs`, `.proto`, `.go` and `.toml` file is scanned, including comments and literals.
Only the exact `temporalio` organization segment in a `github.com/temporalio/<module>`
path is allowed. Offenses report deterministic relative file/line diagnostics.

Cargo resolves all crates through local workspace `path` dependencies. The server's
`go.mod` requires/replaces `umpire.local/model` and `umpire.local/temporal` with the
two in-tree host modules; the latter replaces the former locally as well. No publication
is needed for uncommitted local changes. Extraction remains a plan in `../MIGRATE.md`;
Temporal's Models, generated API and realization stay in the server repository.

The real in-process server/SDK check is separate from the fast model gate:

```sh
make umpire-check-model-slice
make umpire-check-live-tests
```

The first target builds `umpire_temporal_guest.wasm` and requires Complete, Fail,
HostRefusal and Cancellation to pass. The umbrella invokes it before its existing
legacy selector. Seven inherited legacy failures remain explicitly red; see DESIGN's
Mini activity slice section. This bespoke proof does not execute a Testpilot Case.

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

`State` derives `Display` and `Debug`, rendering named fields with standard `Debug`
formatting, including collections and nested Debug-only values. `Action` derives standard
`Debug` and a qualified `Display` name such as `User::PressButton`. Manual `Machine`
implementations provide `Display` for their states and actions and `Debug` for domain
outcomes and facts. Reports and paths implement `Display`; native and Wasm rows use the
same formatting contracts.

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

For a public Model, the macro exports `World` and its `Action` carrier; `world()` returns `World`, so a
realization and guest can name the registered machine without a delegating adapter.
Non-public states retain the opaque return interface and their generated types follow
the state's visibility, preserving module-local Models without leaking private types.
Each match arm combines the action, its guard and its effect. An action is enabled only when the
final state changes; `_ => {}` disables unmatched actions. Without a fallback, rustc checks match
exhaustiveness. Attributes preserve authored statements and method/impl headers.

A module containing `reject!(Rejection::NotFound)` uses the shared step `Outcome`: changed states
emit `Accepted`, and rejected requests emit a `Rejected(rejection)` self-loop. An unchanged handler
without rejection remains disabled. `reject!` records the first rejection and returns from its
enclosing function; a rejecting helper returns only from the helper. Rejection outside evaluation
has no effect. A postcondition can take `(&self, before: &Self, outcome: &Outcome)`.
Engines call `Machine::checked_step`, which refuses a rejecting handler that changed state before
publishing an edge. Direct `step` calls are unchecked and panic on that invalid implementation.
Modules without `reject!` keep unit outcomes and their existing rows.

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
`whenever(..).eventually(..).assuming(..)` builder accepts a positive transition bound with
`.within(n)`. Reaching the first target at exactly `n` transitions passes; the transition that
made the trigger true is excluded. `path!` is deferred to fn-215.
Assumptions compose `only([..])`, `without([..])` and explicit `fair([..])` action lists.
Nothing is fair by default: `#[internal]` marks environment-independent actions only.
`only` replaces the selection, `without` and `fair` accumulate, and fair membership does not
admit an excluded action. Numeric bounds receive no benefit from fairness.

After `Checker::explore`, select a registration from `world.progress()` and call
`result.verify_progress(&world, &claim)`. The query returns a fresh `ClaimResult` using retained
states and rows, without stepping again or changing cached reports. A selector may replace the
assumption with `.assuming(..)` and the bound with `.within_steps(n)` independently. Manual
`Progress::within(scope)` and `Scope::without(&[..])` remain compatible. The query resolves the
unique original registration by name and uses its original callbacks; it refuses unknown or
ambiguous names and changed ordered action catalogs. The caller must supply the same unchanged
world used for exploration; public descriptors cannot prove world identity or unchanged behavior.
Available metadata is validated even with zero starts, but no progress method is invoked without
a reached state. Reached assumption/bound disagreement is Unknown unless that metadata was replaced.

`ProgressVerdict` distinguishes deadlock, cycle, missed deadline, verified with an optional longest
first-target distance, and unknown. Caught predicate/effect defects keep their existing outcomes
without a temporal classification; the aggregate `CheckOutcome` retains `Verified`, `Refuted` and
`Unknown`, and default report text is unchanged.
The supplementary `kettle_progress*` fixtures pin the room/cold longest distances (81/141), cycle,
deadlock, deadline and budget questions using the original Kettle predicates. Fn-213 retains the
ergonomic Kettle `tests.rs` presentation. The TLC emitter maps explicit `fair(A)` entries
to `WF_vars(A)` and introduces no other fairness.
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

`umpire-tests` at `umpire/tests/` owns 24 shared fixtures in `fixtures/<name>/`.
`sample_kettle` and `sample_kettle_trait` are the canonical attribute and manual-trait authoring
examples; focused controls are named for the behavior they test. `progress_dead_end` refutes
progress on a finite maximal path in the assumption-filtered graph, with no enabled internal
action to reach the target. The crate's sole test target is `fixtures.rs`.
The Kettle Models live in their own fixture's `model.rs`;
native and wasm hosts can include that source by path. The trait Kettle has executable native
report, states and rows goldens. Checker-specific failures, panics, overflows, holes, budget cuts
and refusals remain in `umpire/checker/tests/fixtures/`. DSL formatting fixtures remain with the DSL.
The historical authoring sketches have been retired after preserving their comparison in
[DESIGN.md](DESIGN.md#historical-authoring-options); there is no samples crate.

`umpire-testing` is a development-only dependency. It owns rendering, comparison, snapshot
updates, discovery and the consumer registry. Build scripts share its dependency-free discovery
source, avoiding a checker dependency cycle. Fixtures are discovered in name order and must
contain `model.rs`, `expected/`, and at least one consumer-owned expected file. Expected files
select consumers: the checker owns `report.txt`, `states.txt`, `rows.txt`, `negative_report.txt`
and `paths/*.txt`; a selected consumer renders and checks its outputs twice for determinism.
The DSL registers its formatting consumer for `values.txt`. Future backends register their own
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
The default gate builds and checks independently source-lifted TLA+ artifacts and located
subset refusals, and explicitly reports that TLC execution was skipped. To run the real
backend, fetch the pinned official tool once, then require its agreement checks:

```sh
bash model/scripts/fetch-tlc.sh
UMPIRE_BACKENDS=require make model-check
```

The backend admits the official `tla2tools` 1.7.4 artifact by SHA-256 and its TLC 2.19
version, using the repository's mise-pinned Temurin 27. Required runs fail for a missing
or changed tool, unselected admitted fixtures, incomplete exploration, a state-set or
property difference, or a counterexample that does not replay through the Model.
The build-time `syn` lifter reads authored source independently of native exploration;
unsupported source produces located warnings and refuses emission. Each admitted shared
fixture owns emitted module/config and `tlc.txt` agreement goldens. Progress uses explicit
authored fairness and private observers to distinguish real Model edges from idle stuttering.

For a larger Model, the typed backend entry point needs no native exploration:

```rust,ignore
let module = umpire_lift::Tla::emit(&umpire_lift::Lift::source(source_path)?)?;
let result = umpire_tlc::Tlc::discover(repository_root)?.check_module(&module)?;
```

This checks the source-lifted scope independently. Complete native set/verdict agreement is
a separate fixture obligation; a cut native exploration cannot establish it. The admitted
source subset includes scalar/enum attribute Models and the shared manual transition-table,
choice, and progress controls. Checked integer operations use signed TLC-representable
Rust types (`i8`, `i16`, `i32`, `u8`, `u16`); wider types refuse by source location.
Attribute Models also admit explicitly imported `imbl::Vector<T>` with scalar/enum elements:
`new`/`default`, `len`, `is_empty`, guarded `push_back`, and safe `get` queries
(`is_some`/`is_none` or `copied().unwrap_or(default)`). Growth requires an authored finite
length guard; arbitrary collection methods, indexing and panicking `unwrap` refuse.
Generic delegating adapters and arbitrary Rust remain outside the subset. `Phased`,
`reject!`, and imported generic `closable` laws also currently refuse; that source shape
has no system TLC agreement evidence. System fixture roots may retain located refusal
goldens while the required shared gate still mandates attribute Kettle execution.

The default gate needs only the Rust toolchain and workspace dependencies. The optional `cargo-insta`
CLI supports interactive review and acceptance (`cargo install cargo-insta --locked`). Its real
pending-to-accept integration control runs explicitly from `model/` with
`mise exec -- cargo test -p umpire-testing --test consumers accepts_generic_pending_snapshots_with_cli -- --ignored`.

## Dependencies

Only admitted crates (fn-210 R18): a release in the last 12 months, or finished and very widely used.
They are declared once in `[workspace.dependencies]`; a crate opts in with `name.workspace = true`.
`deny.toml` bans the unmaintained alternatives (`im`, `derivative`, `proc-macro-error`,
`proc-macro-error2`, `deluxe`, `index_vec`, `stateright`); `cargo deny check bans` enforces it. A new
dependency needs the same check and a line in the spec.
