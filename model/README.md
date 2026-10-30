# model: Umpire in Rust

Two Cargo workspaces separate the independent framework under `umpire/` from the
Temporal extension under `temporal/`. Each owns its lockfile, configuration and build profiles. The Kettle fixtures establish the generic authoring
and checker surface; the activity slice additionally hosts one real SDK run through a Model.
See [DESIGN.md](DESIGN.md) for the scope, measurements and retained authoring decisions.

`archived/` holds the former Scala tree unchanged for reference, including its retained build
caches. Its internal paths describe the old layout, and its Scala targets are unmaintained and
not expected to build. The Cargo workspace and Rust gate do not build the archive.

Each workspace owns its `mise.toml` toolchain pins (`rust = "1.99.1"`); run through mise:

```
mise exec -- cargo build --workspace --locked
mise exec -- cargo test --workspace --locked
(cd umpire && mise exec -- cargo build --workspace --locked)
(cd umpire && mise exec -- cargo test --workspace --locked)
```

## Crate map

Paths below are relative to `model/umpire/`. Role directories group crates; each crate's
proc macros, syntax, build support and derives live beneath that crate.

| Path | Crate | Job | Used by Model authors |
| --- | --- | --- | --- |
| `author/umpire` | `umpire` | Author facade for DSL declarations, suite questions and catalog derives. | Yes; shared fixtures use it. |
| `author/dsl` | `umpire-dsl` | Model, World, action, claim and path contracts. | Yes; available directly. |
| `author/dsl/macros` | `umpire-dsl-macros` | Generate declarations from the shared front end. | Through the DSL or facade. |
| `author/suite` | `umpire-suite` | Authored runs, searches, coverage and verification. | Through the facade or directly. |
| `front` | `umpire-front` | Read declarations and admit typed Model bodies once. | No; compiler support. |
| `ir` | `umpire-ir` | Admitted typed tree, expressions and embedded values. | No; compiler/checker interface. |
| `ir/derive` | `umpire-ir-derive` | Generate embedded tree construction. | No; IR implementation. |
| `backends/native` | `umpire-checker` | Explore and check Models natively. | Through the suite; direct engine use is available. |
| `backends/process` | `umpire-backend-process` | Own bounded backend processes, Java admission, capture and scratch cleanup. | No; backend support. |
| `backends/tla` | `umpire-tla` | Bind and emit TLA+ and admit shared typed values. | No; backend support. |
| `backends/tlc` | `umpire-tlc` | Execute TLC and decode complete state sets and counterexamples. | No; backend adapter. |
| `backends/apalache` | `umpire-apalache` | Execute bounded safety checks and confirm traces by native replay. | No; optional backend adapter. |
| `protocol/proto` | — | Shared protobuf schemas and ABI contract. | For host/realization integration. |
| `protocol/runtime` | `umpire-runtime` | Protocol messages, transactional dispatch and core-wasm ABI. | For guest/realization integration. |
| `protocol/runtime/guests/stock` | `umpire-stock-guest` | Stock independent checker guest. | No; ABI/parity control. |
| `protocol/runtime/guests/control` | `umpire-control-guest` | Stateful dispatch and transaction controls. | No; ABI control. |
| `protocol/runtime/guests/caught-panic` | `umpire-caught-panic-guest` | Native/wasm caught-panic comparison. | No; fault control. |
| `protocol/runtime/guests/overflow` | `umpire-overflow-guest` | Checked-arithmetic failure control. | No; fault control. |
| `catalog` | `umpire-catalog` | Typed field catalogs, validation, generation, gaps, inspection and derives. | Through the facade or directly. |
| `catalog/syntax` | `umpire-catalog-syntax` | Admit catalog declarations and descriptor coverage. | No; catalog compiler support. |
| `catalog/macros` | `umpire-catalog-macros` | Generate exhaustive catalogs and inspection. | Through catalog or facade. |
| `catalog/build` | `umpire-catalog-build` | Generate descriptor-checked API dispatch and wire mutations. | In API consumers' build scripts. |
| `dev/testing` | `umpire-testing` | Discover fixtures and compare deterministic golden outputs. | For fixture harnesses. |
| `dev/fixtures` | `umpire-fixtures` | Shared end-to-end Models, authored tests and goldens. | Reference examples and regression inputs. |
| `reference/kettle` | `umpire-reference-kettle` | Independent reference Model and realization, with its own Go server/tests. | Reference integration example. |
| `reference/kettle/guest` | `umpire-kettle-guest` | Kettle realization over the shared ABI. | Reference guest example. |

The `umpire` facade exports DSL and suite items at its root, including `machine`, `Action`,
`State`, `World`, `path`, `Space`, `Run` and `verify_all`, plus `Catalog` and `annotate`.
The same surfaces are available as `umpire::{dsl, suite, catalog}`. It reduces author
dependencies while the underlying crates remain independently usable. Backend adapters share
supervision through `umpire-backend-process` and typed assignments through `umpire-tla`;
`umpire-apalache` has no dependency on `umpire-tlc`. Process supervision is native Rust;
parent-side rustix/tempfile support surrounds an audited allocation-free raw-libc child path.
Checker supervision owns adopted descendants on Linux; on Darwin it prevents descendants
and refuses root. Native validation controllers use a separate orchestration policy with
inherited cancellation/completion custody; its Darwin runtime proof remains open. Other
platforms refuse. Apalache remains opt-in with `UMPIRE_APALACHE=require`; TLC requires `UMPIRE_BACKENDS=require`.
`author/umpire/tests/consumer` proves authoring with one renamed facade dependency.
`protocol/runtime/guests` contains stock/control guest packages; `reference/kettle/guest`
contains the reference guest. The Temporal guest and extension remain under `model/temporal/`.

The host-only `umpire/dev/xtask` command front door owns checker provisioning, native
gate stages, timing, protobuf generation and retired-name filtering. Guest commands
execute the independent
`umpire/dev/guest-driver` executable, which owns guest builds, compiler wrapping,
lock admission and the shared admitted tool/caller context. `model/dev/xtask`
retains the thin system workspace command bridge and its entry-point controls.
Root Make and the system `cargo xtask` bridge delegate to the framework tools.
See [GUEST_BUILDS.md](GUEST_BUILDS.md) and
[native tool contracts](umpire/dev/xtask/README.md).

## Hygiene

Matching lint policies for all crates live in each workspace `Cargo.toml` (`[workspace.lints]`: clippy
`all` and `pedantic` at warn, a few pedantic lints that fight the Model style allowed, `unwrap_used`,
`expect_used`, `dbg_macro`, `todo` and `print_*` warned, `unsafe_code` denied, `missing_docs` and
`unreachable_pub` warned). Each crate opts in with `[lints] workspace = true`. Thresholds are in
`clippy.toml`, formatting in `rustfmt.toml` (stable options only, 2024 style, 100 columns).
`rust-toolchain.toml` matches the mise pin and names the components (rustfmt, clippy, rust-src,
rust-analyzer). `deny.toml` is the cargo-deny policy for licenses, duplicate versions, sources and
advisories; it needs the tool installed and the network. Process support mirrors the workspace
lints with `unsafe_code=deny` and permits unsafe only in its audited syscall subtree;
other modules retain `forbid`. Safe crate roots (including binaries, build scripts and integration
tests) must declare `#![forbid(unsafe_code)]` to reject local overrides. Guests inherit the
workspace policy and allow only the three audited ABI export attributes. Java heap and polled
file limits are not kernel resource caps.

The gate, from the repository root (`temporal/`):

```
make model-validate # one TLC-required full gate, then one edited-checker benchmark
make model-check   # same full gate, TLC skipped by default; no benchmark
make model-fmt     # rustfmt both workspaces
make model-time    # time the actual gate; note overruns of 10 s inner-loop / 60 s gate soft targets (R30)
```

Root Make delegates framework checks once to `model/umpire` and retains the Temporal
checks. For framework-only feedback, run `mise exec -- cargo xtask gate` from
`model/umpire`; see its [gate commands](umpire/README.md). Both full gates disable
quick-cache reads before starting stage children.

The gate rejects retired crate names in active files, preserving archived source and historical
evidence. It also checks resolved core capability restrictions and their negative controls,
the generic source boundary, all three protobuf modules' freshness, the wasm32-wasip1 core
and guest builds, committed Temporal guest freshness, and CGO-disabled Go host parity for all three modules.
It provisions the pinned `protoc-gen-go v1.36.6` through the repository's existing Make
target on first use; warm runs need no network. Rust generation uses prost-build and
the mise-pinned protoc 29.5. To regenerate shared Go envelopes, run
`(cd model/umpire && mise exec -- cargo xtask proto-go core)` from the repository root.
Kettle uses `(cd model/umpire && mise exec -- cargo xtask proto-go kettle)`;
Temporal-specific probe controls use `(cd model && mise exec -- cargo xtask proto-go temporal)`.
All three selectors support `--check`; each workspace admits only its own selectors.

`umpire/` owns the DSL, checker, reusable testing tools, Model protocol, transactional
runtime/ABI library, stock/control guests and transport-only Go host. `temporal/` owns
the activity Model and goldens, generated upstream API closure, realization, guest and
its separate Go adapter module. Shared envelopes use `umpire.realization.v1` and
`umpire.annotations.v1` in the framework protocol; the actual-request Observation field
retains tag 4 and its absent-request encoding. The source scanner is generic; its deny policy and
negative controls live in `temporal/testing/boundary.rs`, outside the scanned tree.
Every `.rs`, `.proto`, `.go` and `.toml` file is scanned, including comments and literals.
Only the exact `temporalio` organization segment in a `github.com/temporalio/<module>`
path is allowed. Offenses report deterministic relative file/line diagnostics.

Cargo resolves all crates through local workspace `path` dependencies. The server's
`go.mod` requires/replaces `umpire.local/model` and `umpire.local/temporal` with the
two in-tree host modules; the latter replaces the former locally as well.
The generic Go module separates `hosts/go/host` (wasm transport), `hosts/go/runner`
(the host loop), and `hosts/go/gen/umpire` (generated protocol packages).
Kettle's Go server and host tests form a third module at `umpire/reference/kettle`,
with a local replacement onto `../../hosts/go`; the model gate tests all three modules. No publication
is needed for uncommitted local changes. Extraction remains a plan in `../MIGRATE.md`;
Temporal's Models, generated API and realization stay in the server repository.

The real in-process server/SDK check is separate from the fast model gate:

```sh
make umpire-check-model-slice
make umpire-check-model-gaps
make umpire-check-live-tests
```

Slice and gaps use `temporal.Guest()` from the committed
`temporal/hosts/go/umpire_temporal_guest.wasm`; they build no Rust guest.
The slice requires Complete, Fail, HostRefusal, Cancellation, CancelAfterCompletion,
CancelBeforeWorkerStart and RetryThenComplete to pass. The umbrella invokes both before its
`^TestTestpilot` selector, which also runs the canary harness over the Runner; fn-228.6 retired the
Case tests and their seven inherited legacy failures. No Testpilot Case remains.

The Temporal host module's `go test -tags test_dep ./...` needs only Go, with no
guest environment variable or Cargo target lookup. Model authors refresh the embedded
artifact with `make umpire-guest`. The generic builder's `--check` compares bytes during
`make model-check`, reporting stale sizes and the first differing offset. The shell entrypoint bootstraps native xtask before guest compiler settings, with
auto-install disabled before tool execution. The builder assumes mise and its pinned
tools, Rust sources and target are installed; command failures and stale bytes fail
regardless of `CI`.
The full Rust authoring gate still requires Rust.
Umpire ships no prebuilt stock/control guests. See [GUEST_BUILDS.md](GUEST_BUILDS.md).

The guest's three application exports are `umpire_buffer_alloc`, `umpire_buffer_release`
and `umpire_call`; see
`umpire/protocol/proto/README.md` for ownership and refusal contracts, and `umpire/hosts/go/README.md`
for reproduction and measurements. Rust exports these functions directly with the stock
linker. Go checks the built module's exact application exports and signatures, including
negative controls for missing, unexpected and mis-typed exports.
The ABI crate keeps the workspace lint rules with `unsafe_code = deny` and local
exceptions only on the three named export attributes. DSL/checker retain `forbid`.

From either workspace, `cargo check-all` and `cargo doc-all` are aliases for its clippy and doc steps. The combined gate covers both workspaces.
`model-validate` forces TLC-required `model-time` through environment/Make conflicts; add
`UMPIRE_APALACHE=require` for both. Legacy modes remain; required timings are separate from historical defaults.
`model-time` measures every `model-check` stage, then updates the modification time of
`umpire/backends/native/src/lib.rs` and runs the framework `test-run --package umpire-checker`
route. Source bytes remain unchanged, including when the benchmark is interrupted.
Select another crate/source with `UMPIRE_INNER_CRATE`/`UMPIRE_INNER_SOURCE` (relative to `model/`).
Timed command failures fail the helper; exceeding a time budget prints a warning and succeeds.

## Catalog tooling

`umpire-catalog` owns typed field catalogs, validation, generation, gap reports, inspection,
identifier fill, semantic views and `derive(Catalog)`. The struct `Catalog<M>` and the
`Catalog` derive occupy Rust's separate type and macro namespaces.
Mirror items select their generated API root and limit context with
`#[mirror(api = crate::api, context = crate::Limits, request = "example.v1.Request")]`;
without overrides the defaults are `crate` and `crate::CatalogContext`. Explicit `type`
continues to select one message type. Temporal's realization re-exports its generated
API namespace and aliases `CatalogContext` to its dynamic-config limits at the crate root.

`umpire-catalog-syntax` owns declaration admission and descriptor exhaustiveness, and
`proto::Graph::new(bytes, rust_root)`: the one descriptor graph (messages, fields, presence,
oneofs, maps, enums and RPC edges, named as prost generates them) that catalog/build's wire
mutations and Temporal's `#[realization]` admission both read.
`umpire-catalog-build`'s `System { service, prefix, rust_root, request_roots, catalog_root }`
checks a descriptor against its catalog root, and `generate(descriptor, out_dir)` writes
`services.rs` (method classification/dispatch), `wire_schema.rs` (typed wire mutations),
`claims.rs` (descriptor field claim metadata) and `mirrors.rs` (structural catalog macros).
Temporal authors keep semantic field kinds, projections and rules in `claims.rs` inputs.
The generator fills the remaining descriptor fields with auditable Ignored reasons, then
checks the generated mirrors against descriptor coverage and actual Rust message types.
Temporal's probe adapter suppresses inferred opaque mutations only when they carry no
explicit catalog finding; `wire_cases` stays shape-only and Kettle admits its full inventory.
Generated symbol names derive from the service and prefix. Descriptor acquisition stays
with each consumer; Temporal's API build and realization macros share one definition in
`temporal/api/system.rs`. The independent
Kettle catalog fixture builds without an extension dependency; refusal fixtures retain
missing-field and enum-variant checks alongside descriptor method/action controls.

`umpire_catalog::generic_probe::Engine<S: ProbeSystem>` owns descriptor probe inventory,
current-context materialization, lazy canonical metadata, prediction correlation and gap
classification. System adapters supply typed wire operations, catalog claim resolution and
prerequisite/status policy. Canonical caches contain metadata and findings, never request
bytes; every preparation validates its current baseline and predictions. Strict contextual
witnesses and source dispositions enter through the same adapter contract.

## Kettle service

The framework-owned Kettle service runs a loopback Go server and a fresh wasm guest
through Annotate, Judge and Answer. The healthy transcript pins the complete heating
cycle; `reference/kettle/expected/mutants.txt.snap` pins four differential witnesses:
overlong IDs and removed label validation produce Accepted, a wrong rejection code
produces Mismatch, and boiling before RespondHeating produces current-only Judge
Unexplained. Each witness uses identical prepared wire input against fresh healthy and
single-mutant servers. Healthy-equivalent no-op responses, missing witness registrations,
unknown mutants and transport failures cannot count as detection. Select a server-local
mutation with `kettle.Options{Mutant: name}`; `kettle.Mutants()` lists the supported names.
These controls run in `make model-check`. The focused differential run took 20.546 s;
the full Go host and Kettle suites took 39.087 s and `make model-check` took 546.668 s, exceeding
R30's 60 s soft target.

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

For a public Model without declared constants, the macro exports `World` and its `Action` carrier; `world()` returns `World`, so a
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

`stay!()` in a handler arm keeps the step enabled when the state is unchanged: an accepted
self-loop, such as a heartbeat or an idempotent request answered OK. It does not return; an arm
that also changes state is an ordinary step, and a rejection in the same step wins. Its outcome is
`Accepted` in a module with `reject!` and unit otherwise. Only `#[on]` handler bodies admit it;
in an `#[effect]`, claim, helper or outside `#[machine]` it is a compile error. The lifter admits
it at an arm's top level and refuses it nested in a condition.

A handler may return `Option<F>` for one local, nongeneric, nonempty fieldless enum
with derived `Debug`. All fact-returning handlers in a machine use that same enum;
unit handlers may coexist with them. Each enabled step carries either no facts or the
single returned variant. State change, rejection and `stay!()` retain their enabledness
rules: returning `Some` alone does not enable an unchanged handler.

The independent source subset admits `None`, `Some(F::Variant)` and exhaustive tail
`if/else` trees over those leaves, after supported effects and locals. Conditions read
the state and locals at their authored position. Effect helpers remain unit-returning;
a direct `reject!` in an optional handler fails Rust return typing. A completed optional
result after an unchanged rejecting helper retains both the rejection and the fact.
An ordinary step clears earlier facts; observer-only backend steps preserve them.

See the framework guide for [bounded optional values](umpire/README.md#bounded-optional-values)
and [World-selected assumptions](umpire/README.md#world-selected-assumptions).

### Fixed parameters and computed starts

A module may declare one named-field `#[derive(World)] struct World`. Its fields stay fixed
through an exploration. Like `State`, the derive supplies conditional Clone/Eq/Hash and
Debug/Display implementations; it adds `World::new` with arguments in field order and the
struct's visibility. It requires neither Copy nor Default. The declared World is the Machine;
there is no implicit `world()` constructor for its parameters.

```rust
#[umpire_dsl::machine]
mod counter {
    use umpire_dsl::{Action, State, World, check};

    #[derive(World)]
    pub(super) struct World {
        initial: u8,
        limit: u8,
    }

    #[derive(State)]
    pub(super) struct Model {
        value: u8,
    }

    fn targets(world: &World) -> impl Iterator<Item = u8> {
        world.initial..=world.limit
    }

    #[derive(Action, Copy)]
    pub(super) enum User {
        Set {
            #[domain(targets)]
            value: u8,
            #[inputs([true])]
            enabled: bool,
        },
    }

    impl Model {
        #[starts]
        fn starts(world: &World) -> impl IntoIterator<Item = Self> {
            [Self { value: world.initial }]
        }

        #[on(User)]
        fn user(&mut self, action: User, world: &World) {
            match action {
                User::Set { value, enabled: true } if value <= world.limit => self.value = value,
                User::Set { .. } => {}
            }
        }

        #[invariant]
        fn bounded(&self, world: &World) {
            check!(self.value <= world.limit);
        }
    }
}

use counter::World;

umpire_testing::fixture!(World = World::new(1, 3), 64);

#[test]
fn selected_world() -> umpire_testing::TestResult {
    let files = bound().run()?;
    assert_eq!(files["states.txt"].lines().count(), 3);
    let smaller = bound_for(World::new(2, 3)).run()?;
    assert_eq!(smaller["states.txt"].lines().count(), 2);
    Ok(())
}
```

This exact block compiles as [the DSL README fixture](umpire/author/dsl/tests/readme/parameterized.rs),
checked against the README by `mise exec -- cargo test -p umpire-dsl --test main readme::` from
`model/`. The fixture binds one selected World explicitly; `bound_for` selects another without
fabricating defaults. `World::new` assigns fields and does not validate their ranges or starts.

Handlers, invariants, postconditions, assumptions and progress methods may take a trailing
immutable `&World`. Postconditions retain their optional before-state and outcome arguments
before it; parameter names do not select the signature. Per-method `#[scope]` is also accepted
as an assumption spelling, in the same registration order. Grouped `#[scopes]` is refused.
`#[machine]` refuses generic Model, World and actor declarations; standalone derives keep theirs.

A `#[starts]` method supplies complete states, preserving correlations across fields. It is an
associated fn of any name in the main `impl Model`, first among the registered methods, taking
only an optional trailing `&World` and returning `impl IntoIterator<Item = Self>`. It replaces
State's generated static starts and their Default bounds and requires a declared World. Declaring
it twice, or field starts as well, is an error, even if their range is empty; so is the former
`#[starts] impl Model` spelling. The checker keeps the first occurrence of each complete
state, and an empty computed iterator supplies no starts.

Use `Space::<counter::Model>::new(counter::World::new(0, 2))` to explore its computed starts,
or `Space::with_world(world, starts)` to choose explicit starts. `verify_all`, `refuted`, `find`,
`cover` and `sample` all use that selected owned World. `Run::new(world, state)` starts an
authored run; `Run::along_in(world, path)` and `along_in_with_budget` search one. Covered and
sampled Runs retain the World after their Space leaves scope. Existing state-only shorthand and
manual `Registered` implementations remain compatible through the additive `Associated` trait.
Declared states implement only the association, so their constants must be supplied explicitly.

Declared Worlds with admitted scalar/enum constants also support TLA+ emission from the embedded tree.
Only values cross from native code to the lifter: inside a declared-World `#[machine]`,
`derive(World)` and `derive(State)` add a hidden `Values` impl listing the named field values,
private fields included, and `derive(Action)` a hidden `ActionValues` impl giving each action's
variant and payload and each `#[domain]` field's values for a World. Field visibility is
unchanged; the export shows what the derived `Debug` already prints. `#[machine]` exports each
local fieldless enum's variant, as does `derive(Finite)`. On native targets, `imbl::Vector<T>`
exports ordered `Value::Sequence` elements when `T: IntoValue + Clone`, supporting binding
and typed replay of admitted sequence state. A field type without a value
leaves the Model compiling and is refused only where a binding is requested. The `#[machine]` macro admits the Model once and
embeds the admitted tree; the lifter emits from it. In a shared fixture,
`fixture!(World = World::new(1, 3), 64, negative)` defines `bound()`, and `bound_for(world)` for
another World: one World, one start enumeration and one checked catalog feed binding, native
checking and TLC counterexample replay.

Methods of each kind are contiguous, in `#[on]`, optional `#[effect]`, `#[invariant]`,
`#[postcondition]`, `#[assumption]`, `#[progress]` order. A postcondition may filter its action
with `#[postcondition(after = Heater::Failover)]`. Shared effects are ordinary `#[effect]`
helper methods; Kettle keeps its effects inline. Only `#[on]` and `#[effect]` methods may take
`&mut self`. Other `Model` methods take `&self` or no receiver, including unmarked helpers in
qualified, nested or aliased impls; compiler-resolved checks leave unrelated helper types alone.
Compiler-only checking declarations may be added inside local function or const scopes to resolve
aliases and trait projections; they add no runtime guard.
`#[machine]` forbids unsafe code throughout its module.

Attribute progress methods return `LeadsTo`, written as a two-waypoint `path!`. Kettle's claim is:

```rust
/// Switched on and left on, the water boils and the kettle beeps.
#[progress]
fn boils(&self) -> LeadsTo {
    path![self.heating_on, .., self.beeping].assuming(Self::never_switched_off())
}
```

A claim's `path!` has two state waypoints with a gap between them. A state waypoint is a `bool` (evaluated
per state inside a Model method), a closure over the state, or a state. `..` is any number of
steps; `..=n` is at most `n` steps. Reaching the target at exactly `n` steps passes; the step
that made the trigger true is not counted. The list reads the same everywhere; the question
asked with it decides what it promises:

- In a `#[progress]` claim, `..` means on every admitted path: whenever the first waypoint
  holds, every path the assumption admits reaches the second. A claim has exactly two waypoints:
  `#[machine]` refuses a literal `path!` chain at compile time, and a chain built elsewhere
  panics when the claim is evaluated, so the check reports that claim Refuted.
- In a search for an example or witness (`find`, `Run::along`), `..` means
  some path: the checker takes the shortest path through all waypoints at once, each within its
  gap, and refuses only when no admitted path has them.
- A pattern says nothing about what happens between or after its waypoints.

Claims reject adjacent waypoints, leading/trailing gaps, actions and structural patterns; `..=1` is the next
step. Test-space patterns admit adjacency, but their first and last waypoints cannot be gaps.
`whenever(..).eventually(..).within(n)` builds the same claim and remains public.
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

## Annotating an API

Author each feature's sidecar in `temporal/feature/<feature>/api.rs`, alongside its
independent Model. Follow the [annotation guide](temporal/foundation/README.md) for
derive mirrors, typed projections, constraints, roles and exhaustive mappings;
[Annotation catalog](DESIGN.md#annotation-catalog) explains admission and the API bump workflow.
Use the activity catalog as the concrete example. Request constraints need an expectation
including intentional server acceptances. Keep ignored fields
explicit with reasons and validate responses without rewriting their bytes.

Verify with `make model-check`; activity transport/catalog changes also require
`make umpire-check-model-slice` and `make umpire-check-model-gaps`. The latter compares
actual live probes with `temporal/feature/activity/expected/gaps.txt.snap`. Review unexpected
Accepted/Mismatch/Unannotated rows and explicit mutation refusals before changing expectations
or goldens. Its external live golden is checked by the gap gate, not regenerated by
`make model-update`.

## Mini activity slice (fn-216)

The independent activity Model carries retry maximum 1..=3, four optional bounded
`Ticks` deadlines and atomic/cooperative cancellation. Its state records phase, attempt,
pending cancellation and four countdowns. One Tick decrements all live budgets and selects
ScheduleToClose > StartToClose > Heartbeat > ScheduleToStart, emitting one exact expiry fact.
Terminal transitions clear budgets and pending. Pending attempt expiry or retryable failure
cancels, while global expiry times out; completion and nonretryable failure may win.

Exact Poll(1)..Poll(maximum_attempts) and RespondCompleted weak fairness establishes
closure, including timeoutless Worlds. Registered countdown laws establish the spanning
actual-Tick bound across arbitrary interleavings. The [activity guide](temporal/feature/activity/README.md)
states arming/reset rules, cancellation boundaries, the induction and availability controls.
Its [compatibility projection](temporal/feature/activity/compatibility/README.md) validates
one atomic attempt with one-tick waiting/running deadlines against all 15 original artifacts,
retaining exactly one start, six states and eleven old visible edges.

The default offline three-attempt World reaches 18 states and emits 63 edges from 216
evaluations, with 153 disabled evaluations and depth six. Its 17 claims verify natively;
Cases(3 * 7) retains 21 atomic witnesses. Expanded cooperative Worlds are checked separately.
Live constructors keep request-derived maximum_attempts, absent timeouts and atomic
cancellation until the downstream binding work; checker Worlds grant no live capability.

`terminal_remains` and `closed_rejects` are registered postconditions, delegating to the
shared closure laws. Fn-221 adds a rejected cancel self-loop at each of the four terminal
phases, giving both laws terminal edges to check. The controls also enumerate every
terminal phase/action and prove that a hypothetical reopening, accepted closed step or
wrong rejection refutes the corresponding law.

The reusable `umpire-runtime` library owns state, replay, first-Unexplained freezing,
Reset, response-publication transactions and ABI buffers. A system-owned `Realize`
realization proposes paths; the runtime validates those paths through actual Model steps.
The activity realization uses Temporal-owned `#[realization]`, `#[observe]` and `#[answer]`
attributes over typed handlers. Their declared field paths distinguish observed responses,
incoming callbacks and produced partial SDK requests; omitted present fields refuse.
See the [realization guide](temporal/realization/README.md) for explicit state projection,
`Step`/`Reply` returns and the raw typed request packing contract.
The registered Temporal fixture consumer checks `expected/realization.txt.snap` against
the generated runtime metadata. It distinguishes observed, callback and produced contexts,
lists every classified path and ignore reason, and derives unhandled Model-action RPCs from
the admitted API mappings. [Declarative Realizations](DESIGN.md#declarative-realizations)
records the report contract and complete-file source counts. After committing a coverage
checkpoint, run its real added-catalog-field control from `model/` with
`mise exec -- cargo test -p umpire-temporal-tests --test fixtures added_catalog_field_fails_the_real_coverage_golden -- --ignored`.
It rebuilds in a disposable worktree and preserves the existing coverage baseline.
For public states without declared constants, `#[machine]` exports `World` and `Action`,
and `world()` returns `World`. Non-public states retain the opaque interface with generated types following
the state's visibility. The declared activity World uses an explicit six-field `World::new` and removes
the realization's duplicate action carrier, delegating machine and claim-index re-wrap.
Generic Any helpers derive canonical type URLs from generated `prost::Name`; only the
Describe and cancel RPC method paths are realization constants. The policy parser and reply constructor
keep the remaining realization specific to Temporal.

The shared envelopes live under `umpire/protocol/proto`, in `umpire.realization.v1`: Observation carries a method,
a response Any or error, and an optional actual-request Any at tag 4, Callback a task Any,
and Reply a request Any. Error retains the same code/message fields. The enclosing Any
type URLs migrate to the shared package; protobuf payload encodings and tags remain fixed.
The real upstream API descriptor
closure is compiled natively and for WASM; its well-known types are `prost-types`, re-exported
once as `umpire_runtime::prost_types`. The closure's file count is not a schema invariant.
No Testpilot schema is moved, imported or copied.

Describe projects actual Info.Status, Info.RunState and Info.Attempt: Running/Scheduled or
Running/Started, and terminal status/Unspecified. There is no Started execution-status
enum. Only a valid callback evidences Poll. Repeated current observations are zero-step
Explained; premature terminal and stale Scheduled observations are Unexplained, freezing
the run until Reset. The SDK callback supplies a partial poll projection containing
exactly one binary/plain Complete, Fail or RetryThenComplete payload. Answer selects a partial
completed or failed RPC request (retryable only for RetryThenComplete at attempt one), leaving the Model Started until terminal observation.
The facade binds actual namespace/activity/run identity and immutable raw Start request bytes.
Task tokens and timestamps stay outside the lifecycle projection. See the realization guide
for the per-field admission table, publication rollback and retained-receipt Reset contract.

The live owner judges Scheduled with its registered worker stopped, answers delivery
through that run's one Model instance, then synchronously reads and judges Describe.
This ordering prevents a pre-delivery read from arriving after delivery; it does not
solve general concurrent read-window inference. Complete and Fail, reset/isolation,
repeat observations, opposite-terminal mutation, host refusal and cancellation all have
real server/SDK controls. The worker returns only decoded Model reply data and uses the
SDK failure converter. This is a bespoke SDK/server/Model proof, not execution through
Testpilot Prepare/PreparedCase.Run, Case, Program or Contract.

Declared Worlds, computed starts, Finite inputs, facts, bounded optional values and
exact selected-World fairness support the activity policy. The retained retry, closure,
compatibility, deadline and cooperative matrices use the same authored source with
independent interpretation and required TLC controls. [Activity policy verification](../.flow/artifacts/fn-237-activity-timeouts-heartbeat-and/activity-policy-verification.md)
records measured Worlds, replay, mutations, timing and the live admission boundary.
The [earlier checkpoint](DESIGN.md#parameterized-worlds-and-inputs) retains the original
actual live retry proof. No Model transition was selected to fit server evidence.

At that historical checkpoint, R33's protocol seam was proven and the broader runtime
replacement was still open. The later [Runner and Run](DESIGN.md#runner-and-run)
checkpoint now discharges R33–R39, records the final package layout and exact Run/replay
schema, and reports comparable counts and actual gate timings. Temporal workers now
follow a Model-issued program from Reset without callbacks; generic Answer remains for
other systems. `umpire program --model <guest.wasm> --world <world.binpb> --out <program.binpb>`
exports that table. The raw and Go SDK executors live in
`temporal/hosts/go/reference/worker/{raw,sdk}`; `tools/umpire/temporaldriver` composes
their Validate/Follow functions with the opaque transport Driver in Testpilot's `temporal/act`.
The hold fault replaces the old worker gate; the proposed Shim was superseded.

Final R40/fn-228 R7 proof remains open in fn-243.5 after independent fn-237 cooperative
semantics and binding. Unheld open-state cancellation is refused before proposal or
Cancel RPC with `unheld cancellation of an open activity is outside the atomic cancellation Model`;
held-before-start and terminal cancellation remain supported. The recorded 21-Run
lifecycle admits Explained cases plus only exact audited scope refusals, not 21 Explained
Runs. [Non-Go execution remains unproven](../.flow/artifacts/fn-243-programmed-workers-the-model-issues/non-go-offline-gap.md)
in the offline environment. Both fn-228 and fn-243 remain open.

Predecessor .3's final dedicated run took 381.104 ms including cleanup, with Answer
calls of 55.417 µs Complete, 48.750 µs Fail and 31.166 µs HostRefusal. Its full/inner
model gate took 9.746/1.819 s. Task .4's pre-edit warm gate measured 8.061/1.738 s
against the 30/10 s soft targets; its dedicated real run took 269.983 ms. These are
individual observations on linux/arm64, Go 1.27.0, Rust 1.99.1 and SDK 1.48.0,
excluding compilation from the live test timer, not percentiles or idle-host claims.
The post-change warm gate measured 11.856 s full and 2.526 s inner while clean-build
and live verification were also active; both stay below the soft targets. Fresh
descriptor generation produced the same 57 file descriptors, equal by file name;
raw descriptor-set byte order differs because getproto iterates a Go map. Its ignored
cache is not a canonical wire artifact. The original descriptor/cache was preserved.
Raw commands and logs live under `.flow/tmp/fn-216-4/`; predecessor records remain
under `.flow/tmp/fn216-3-*`.

Task .4's final umbrella measured 502.668 ms for its dedicated slice and 296.593 ms
for the broad-selector invocation. Their Complete/Fail/HostRefusal Answer calls were
134.958/65.583/168.333 µs and 103.834/57.000/32.958 µs respectively. Concurrent clean
compilation and Go linting ran during verification; these samples make no idle-host claim.

At fn-216.4 the umbrella also carried seven inherited legacy failures (AssessRecordedRuns,
ExplorationDiscoversUnpinnedExecution, GeneratedCases, NexusControlForgedCompletionIsViolated,
NexusControlReplaysThroughTheCommand, NexusPairCase and
UmpireRunRunsACheckedInCaseAgainstAnyEndpoint, all TestTestpilot-prefixed; see
`.flow/tmp/fn-216-4/legacy-comparison.md`). fn-228.3 and fn-228.6 retired them with the Case
runtime and mapped their coverage to the Runner (`.flow/artifacts/fn-228.6/coverage-map.md`); the
umbrella now expects no failure identity.

## Testing

Models are read once, in `umpire-front` (`umpire/front/`). `declare/` reads the declarations:
attribute kinds and their order, registration, the State/World/actor schema, starts, input
domains and `&World` parameters, and `path.rs` the `path!` language. The macros in
`umpire-dsl-macros` only generate from what `declare/` read, and report its refusals as compile
errors. `shape.rs` takes the same declarations and admits the subset, delegating each body to
`body/` with a typed environment; these refusals are soft, since the Model still checks natively.
Body lowering reads no module items; `tree.rs` holds the admitted tree and `expr.rs` its typed
expressions. Its `Diagnostic` carries line, column and message; `umpire-tla` reads the source
files for hook claims and refusal locations, names the file in each refusal, gives the embedded tree its TLA+ names
(`names.rs`), binds the selected World (`bind.rs`) and emits TLA+.

End to end, by golden files (fn-210 R8, R23, R29). A test is a fixture, an input plus the output it
produces (`model.rs`, `tests.rs` and `expected/`). "Fixture", not
"case": Case is Testpilot's term.

- behaviour: each fixture's `expected/report.txt.snap`, `states.txt.snap`, `rows.txt.snap` and `paths/<claim>.txt.snap`
  (`insta`; review with `cargo insta review`);
- refusals: `trybuild` `.rs` + `.stderr` under `umpire/author/dsl/tests/ui/`;
- expansion: insta snapshots of each macro's implementation function, formatted by
  `prettyplease`;
- one property test against a brute-force reference explorer, the check that goldens are right
  rather than merely stable.

Goldens are deterministic, human-readable and timing-free. `make model-update` rewrites them; a
shared fixture folder without `model.rs`, `tests.rs` or `expected/` fails the build naming it. Unit tests only when a defect cannot be
written as a fixture.

`umpire-fixtures` at `umpire/dev/fixtures/` owns 12 shared fixtures in `fixtures/<name>/`, all
`#[machine]` Models or adapters of one. `sample_kettle` is the canonical authoring example;
focused controls are named for the behavior they test. `progress_dead_end` refutes
progress on a finite maximal path in the assumption-filtered graph, with no enabled internal
action to reach the target. The crate's sole test target is `fixtures.rs`.
A shared fixture's `model.rs` holds only the Model (what `#[machine]` registers, what the lifter
reads, what hosts include); its golden hook (`umpire_testing::fixture!` or a hand-written `run`)
lives in `tests.rs` beside it. The build declares both as sibling modules, `<name>` and
`<name>_tests`, so `tests.rs` reaches the Model as `crate::<name>` and state fields are `pub`.
Folder names that are not plain lowercase identifiers compile under an escaped `fixture_` module.
The lifter reads a hook's supplemental claims from `tests.rs`.
The attribute Kettle lives in its fixture's `model.rs`; native and wasm hosts can include
that source by path. Checker-specific failures, panics, overflows, holes, budget cuts and
refusals remain in `umpire/backends/native/tests/fixtures/`, together with the manual-trait Models the
attribute surface cannot state (DESIGN.md lists each with its gap). Among them,
`sample_kettle_trait` is the independent manual-trait engine reference: its Model is
`kettle.rs`, which the stock wasm guest includes, and its report, states and rows goldens are
the Go host's expected payloads. DSL formatting fixtures remain with the DSL.
The historical authoring sketches have been retired after preserving their comparison in
[DESIGN.md](DESIGN.md#historical-authoring-options); there is no samples crate.

`umpire-suite` at `umpire/author/suite/` is what a Model's own `tests.rs` uses (fn-213). An example is
an authored path: `Run::from(state)`, `.step(action)`, `.repeat(n, action)` and `.state()` replay it
through `Machine::checked_step`, and `Run::along(path![state, .., |s| ..])` fills in the gaps with
the shortest path through every waypoint, retaining exact explored rows. A disabled action, one emitting several
steps, an unbridgeable gap and a predicate as the first waypoint are `Refusal`s naming the action
or waypoint and spelling the state. `expect! { s = run.state(); a; b; }` fails the test at the
first false line with its source text and the state. A run starts from a state alone because
`#[machine]` registers the state with its world (`Registered`); a manual Model implements that
trait by hand. An exploration asks the checker about a `Space`, start states and a state budget
(`Space::from([state]).budget(n)`, by default the checker's): `verify_all(&space)` is the Model's
own claims from those starts, `refuted(&space, |s| ..)` the shortest path to a state where the
predicate fails, and `find(&space, path![..])` the shortest path from the starts through
every waypoint in order. A cut answers unknown (a report outcome, or a `Refusal` when no
counterexample was found before the cut), never verified; a predicate that holds everywhere and
an unbridgeable gap are `Refusal`s too. A skipped step panic refuses a witness only when its
source depth is less than the witness length, since the missing edge could hide a shorter answer;
without a witness it refuses the search. Waypoint predicate panics have their own refusal naming
the waypoint. Search explores distinct Model states once under the budget, then finds the shortest
pattern match over retained rows, keeping nondominated partial matches and their own predecessor
paths. Pattern progress spends no Model-state budget, even when a witness revisits states.
`Run::along_with_budget(path![..], n)` sets the same budget; `Run::along` keeps the default.
A cut graph can answer only when its unexplored frontier cannot hide a shorter witness; a bounded
pattern anchored at one concrete state can also settle absence before that frontier.
The refusal names the furthest waypoint reached. The suite's messages are pinned by its fixtures
in `umpire/author/suite/tests/fixtures/`.

### Test-space patterns

`find`, `cover` and `Run::along` accept the full `path!` grammar. Adjacent state predicates
consume no steps; each action consumes exactly one step. Only an explicit gap admits
intervening actions. Search may reach the first waypoint from a Space's starts; an initial
concrete state anchors `Run::along`. A story must have one authored alternative; alternatives
refuse with “use cover”.

| Pattern | Meaning |
| --- | --- |
| `state`, `true`, `\|s: &Model\| s.closed()` | Exact state or state predicate |
| `Worker::Poll(1)` | One concrete action payload |
| `User::Send { amount: 0..=3, from: (king \| queen) }` | One alternative per finite input value/combination |
| `User::Send { amount: 2, from: _, .. }` | Exact input; `_` and omitted fields use declared domains |
| `Deadline::*` | Each action/input value of this actor |
| `Client::Cancel => Outcome::Rejected(Rejection::NotFound)` | Action with a typed outcome filter |
| `Client::Cancel => (Outcome::Accepted \| Outcome::Rejected(_))` | One alternative per outcome pattern |
| `(Worker::RespondCompleted \| Worker::RespondFailed { retryable: false })` | Authored alternatives, in order |
| `[Worker::Poll(1); n]`, `[Worker::Poll(1); 0..=1]` | Exact repetition or one alternative per inclusive count |
| `unordered![a, b, c]` | Every ordering of the listed patterns |
| `..`, `..=n`, `..!(Client::Cancel, Worker::RespondCompleted)` | Any steps, at most n steps, or steps excluding the named actions |

Payload fields of an `Action` declare their complete finite input catalog with
`#[inputs(0..=3)]` or `#[inputs([Account::King, Account::Queen])]`. A field without either uses
its type's `Finite` domain: `bool`, a `#[derive(Finite)]` fieldless enum in declaration order, or
a `#[derive(Finite)]` newtype whose one integer field declares `#[inputs(lo..=hi)]`. That one
range also generates `try_new(raw) -> Option<Self>` and `contains(raw)`; enumeration is not
validation, and raw construction stays possible. Raw integers have no `Finite` domain and keep
the missing-domain refusal; other non-`Finite` fields fail to compile with a note naming the
choices. Products enumerate in field order, the last field fastest.

Inside a `#[machine]` with a declared World, `#[domain(sizes)]` on a field names a
`fn(&World) -> impl Iterator<Item = FieldType>`. Such an actor has no static catalog: it
implements `ActorValue` (identity and `internal`/`fault` classification) but not `Actor`, and
`User::actions_in(&world)` returns its lazy domain for that World. Concrete values still infer
typed `path!` steps, though an outcome filter (`=>`) still requires a static `Actor`; a `User::*` or `User::Set { .. }` pattern without a World refuses at
compile time, so contextual queries pass the selected World explicitly:
`Pattern::action(User::actions_in(&world).map(Action::from), None, None)` inside `Path::pattern`.
`#[domain]` outside a declared-World machine, on a variant, twice, or together with `#[inputs]`
is refused; a wrong World or item type is a type error.

`Machine::actions` stays the unchecked catalog. Engines call `Machine::checked_catalog(limit)`,
which generated input-bearing machines answer by counting every variant's domain before
building any action, with exact overflow detection; an empty factor makes its variant empty
even beside an overflowing range. The default limit is 4,096 concrete actions; a larger catalog
is a checker `Refusal::CatalogLimit` naming the count, an unrepresentable one
`Refusal::CatalogOverflow`, before any handler runs. `Checker::explore_within(world, starts,
budget, limit)` and `Space::catalog_limit(n)` select another limit, which progress queries and
every suite route reuse; the Wasm runtime refuses with the `Budget` code. A declared variant with
no concrete actions is reported as `vacuity: variant User::Pick has no actions`. Manual
Machines keep the default, which checks the length of `actions()`.

Requested impossible values remain alternatives and fail witness search. A fault is an ordinary action of an actor marked `#[fault]`; `#[internal]`
alone neither declares a fault nor fairness. Permutations default to at most six elements
(720 orders); `path.permutation_limit(n)` explicitly changes that admission limit.

`cover(&space, Cases(n), path)` first compares the authored count with `Cases(n)`, before
search. Alternatives multiply across independent choices, input values, repetition counts
and permutations. A mismatch lists the alternatives; a count beyond the alternative limit
refuses. Cover returns one shortest witness per alternative, in authored order, or refuses
the unwitnessed alternative with its index, choices and reached state. It never silently
drops an impossible choice. `Covered::runs()` holds ordinary Runs; `len()`, `all(..)`,
`expansion()` and `coverage()` expose the result.

The limits are independent: `Space::budget(n)` counts distinct Model states;
`alternative_budget(n)` admits authored alternatives (default 1,048,576);
`walk_bound(n)` bounds each sampled walk in Model edges (default 256);
`catalog_limit(n)` admits concrete catalog actions (default 4,096). Pattern/product and
experiment storage also have independent finite ceilings. Cover and sample require a complete
explored graph and refuse `Refusal::Cut` when the state budget cuts it. Unbounded gaps still
search only that finite graph; they do not enlarge its state budget.

The real Kettle demonstration is runnable with
`cd model && mise exec -- cargo test -p umpire-fixtures`:

```rust,ignore
let covered = cover(&Space::from([room()]), Cases((80 + 1) * 2),
    path![room(), User::PressButton, [Heater::Heat; 0..=80], ..,
          (User::DepressButton | Heater::Failover)])?;
```

The explicit gap supplies additional Heat actions before Failover for prefixes shorter than
80. The 162 alternatives describe explicit prefixes, not total heat counts; expansions show
every actual step. The literal adjacent form is retained as a refusal control. Activity instead constructs
21 explicit alternatives: for each attempt 1..=3, Cancel or ScheduleToStart from Scheduled,
and Cancel, RespondCompleted, either RespondFailed payload or StartToClose from Started.
The selected three-attempt World supplies the search graph; gaps reach each requested phase
and attempt before the final action. `temporal/feature/activity/tests.rs` checks each exact
witness, including retries returning to Scheduled. Run it with
`cd model && mise exec -- cargo test -p umpire-temporal-tests`.

`sample(&space, path, n, seed)` performs reproducible finite random walks, including possible
nonshortest paths, rather than selecting only cover's shortest witnesses. Its expansion records
the seed, requested experiment size, walk bound and chosen alternatives. An alternative that
cannot finish within the bound refuses; it is never silently replaced with an easier choice.
Kettle's seeded example uses `walk_bound(12)`, four walks and seed 42. Sampling establishes only
that finite experiment, not complete coverage of alternatives or all paths.

Every cover/sample demonstration records `expansion/<name>.txt` with concrete choices,
actions, outcomes, facts and final states, plus `coverage.txt`. Typed catalog coverage lists
touched and untouched actions, outcomes and faults; undeclared metadata is unknown, distinct
from a declared empty catalog. Claims report witness evaluations and progress triggers by
registration identity, not whole-graph verification. Untapped entries remain visible. The
`umpire_testing::SUITE` consumer owns these outputs; register it alongside CHECKER and seed an
expected payload before updating goldens. Kettle retains its native/negative hook through
`fixture!(world, budget, negative, extra = suite_files()?)`; duplicate output keys refuse.
The lifter reads a hook's negative claim from `tests.rs` against the admitted tree, so it may
read state fields, the World and module constants, but not call Model methods or helpers.

To write a Model's tests, add `mod examples` and `mod explorations` to its `tests.rs`, each with
`use umpire_suite::*;` and plain `#[test]` functions returning `Result<(), Refusal>` so a refusal
fails the test with its message; `cargo test` and the gate run them, nothing registers them
(`umpire-suite` is a dev-dependency of the crate that compiles the fixture). Kettle's
`umpire/dev/fixtures/fixtures/sample_kettle/tests.rs` is the example: it asks the questions of the archived
`Kettle.test.scala` that the surface admits (the press/heat/depress table from the coldest and room
starts, the 99-to-100 boundary and the `>=` failover guard refuting `fails_over_at_boiling` at 101,
the 81 steps from switching on to the beep, every start from -40 to 40, `never_beeps` refuted by a
press, 80 heats and the failover, a budget of 10 answering unknown and a counterexample found before
a cut, and `fails_over_at_boiling` never triggered with the heater excluded, asked of a small
`Machine` that delegates to Kettle's world without the heater's actions, since a `Space` chooses
starts, never actions). From room temperature the beep is the deepest state, so a cut one state short of the whole
graph misses it; the cut-but-refuted question starts heating at 99 as well. The Scala questions the
surface does not admit, and why:

- the boils cycle on the whole Kettle, the deadlock with only the press, the missed 80-step deadline and the unfair `fair(heat)` cycle:
  a `Space` chooses starts and a budget, never actions or assumptions (scopes are the Model's own),
  so these stay with the `kettle_progress*` fixtures (fn-214.3), which explore their own worlds;
- claim truth tables over hand-picked before/after pairs that are not steps (`warms_only_when_on`
  on quiet-to-hot by heat): a claim is judged only on explored steps; a state invariant can still be
  asked from chosen starts, as the quiet, hot and heating starts holding `beeps_only_when_off` and
  the heating, beeping start refuting it show;
- `boils`'s metadata (name, no bound, the admitted actions of `never_switched_off`): the suite has
  no claim descriptors;
- the report's prose line (`budget N states, 82 steps; exploration complete`) and the empty query
  list: the Rust report spells counts, not that line, and has no queries; the depth and cut are
  asked instead.

`umpire-testing` is a development-only dependency. It owns rendering, comparison, snapshot
updates, discovery and the consumer registry. Build scripts share its dependency-free discovery
source, avoiding a checker dependency cycle. Fixtures are discovered in name order and must
contain `model.rs`, `expected/`, and at least one consumer-owned expected file; shared fixture
roots (`umpire/dev/fixtures/fixtures/`, `temporal/feature/`) also require `tests.rs`, while synthetic
control roots keep the hook in `model.rs`. Expected files
select consumers: the checker owns `report.txt`, `states.txt`, `rows.txt`, `negative_report.txt`
and `paths/*.txt`; a selected consumer renders and checks its outputs twice for determinism.
The DSL registers its formatting consumer for `values.txt`. Future backends register their own
filenames and renderer. An unregistered filename, an orphan snapshot artifact, or an empty
fixture fails naming its folder, including during updates. Seed a new fixture with a snapshot
file such as `report.txt.snap` before running `make model-update`; renderers and consumer
registrations still use logical names such as `report.txt`.

Goldens use named insta text snapshots: one `<logical filename>.snap`, including
`report.txt.snap`, `Model.tla.snap` and `Model.cfg.snap`. Readers use the std-only
`snapshot_payload` helper (Go: `umpire.local/model/snapshot`) to preserve the text body
without the insta header. `make model-update` refreshes fixture snapshots; review the
snapshot diffs before accepting them. Pending `.snap.new` files refuse even during updates.
Optional `cargo insta review` or `cargo insta accept` can accept reviewed candidates; the
normal test pass requires no installed cargo-insta CLI. Skipped backends still validate
complete, reviewed snapshots and report no execution or agreement evidence.

The ordinary snapshot ownership test covers every current fixture root and named snapshot,
including cross-crate, Go and external live readers. Historical compatibility originals retain
their recorded format. Live Go tests remain independent producers: export candidates, then
import them through the explicit ignored Rust insta controls. An import is no live execution
or comparison evidence. For activity gaps, run the live gap gate, then from `model/`:

```sh
UMPIRE_GAP_CANDIDATE=../.flow/tmp/fn2174-live-gap-report.txt INSTA_UPDATE=new mise exec -- cargo test -p umpire-temporal-tests --test fixtures import_live_gap_candidate -- --ignored
```

The default gate builds and checks independently source-lifted TLA+ artifacts and located
subset refusals, and explicitly reports that TLC execution was skipped. To run the real
backend, fetch the pinned official tool once, then require its agreement checks:

```sh
(cd model/umpire && mise exec -- cargo xtask fetch-tlc)
make model-validate
```

The native fetch command delegates provisioning to the framework-owned mise HTTP provider
under `umpire/mise-tools`; Apalache is explicit opt-in and execution never auto-installs.
The backend admits the official `tla2tools` 1.7.4 artifact by SHA-256 and its TLC 2.19
version, using the repository's mise-pinned Temurin 27. TLC and Apalache share one admission,
`umpire_backend_process::PinnedJar::admit_mise`: a symlink-free exact mise install path, an
in-process `sha2` digest, the pinned Java, then the tool's version probe. Required runs fail for
a missing or changed tool, unselected admitted fixtures, incomplete exploration, a state-set or
property difference, or a counterexample that does not replay through the Model.
The Apalache backend's real-tool controls are not part of the required gate while fn-211 is
deferred; `UMPIRE_APALACHE=require make model-check` fetches the pinned release and runs them.
The `#[machine]` macro admits each Model through `umpire-front` and embeds the admitted tree
in the compiled Model (off the wasm guest); the lifter emits from it, and Model bodies outside
the subset carry a located warning that refuses emission. Each admitted shared
fixture owns emitted module/config and `tlc.txt` agreement goldens. Progress uses explicit
authored fairness and private observers to distinguish real Model edges from idle stuttering.

For a larger Model, the typed backend entry point needs no native exploration.
`fixture!`, or `umpire_testing::lifted!(world)` in a `tests.rs` without one, defines
`lifted(source)`: the module emitted from the Model that World's `#[machine]` embeds.

```rust,ignore
umpire_testing::lifted!(counter::world());
let module = lifted(std::path::Path::new("fixtures/counter/model.rs"))?;
let result = umpire_tlc::Tlc::discover(repository_root)?.check_module(&module)?;
```

Both also define `interpreted()`, the differential check (fn-238.5). For a World whose
`#[machine]` module is admitted, it explores the Model twice under the fixture's budget:
compiled, and as `umpire_checker::Interpreted`, which evaluates the embedded tree on the
compiled typed state through accessors `#[machine]` generates beside it. Every checker file,
the report, states, rows, negative report and paths, must be identical, and so must any
refusal, including a panic while enumerating starts. A failed `check!` reports the text
`check!` itself spells, which `#[machine]` records for each one the tree indexes. The shared
runner's `interpreted_agrees_with_compiled` runs it for every shared fixture and fails if an
admitted one is not interpreted. The Temporal activity's `run()` and the admitted checker
controls call `umpire_testing::interpreted` themselves. The `interpreted` test of
`umpire-fixtures` adds the lifter's and the backends' own Models (the Apalache arithmetic
fixture across 220 Worlds), lazy start domains, length arithmetic, checked-arithmetic failures,
the claim cap and seeded mutant trees that must disagree. The interpreter does not replace the compiled checker, and the guest
and typed replay keep running the compiled bodies. The catalog, claim order, computed starts
and declared-World starts stay the compiled Model's; a fixture's negative claim is judged by
the compiled World on interpreted states.

For a declared World, `lifted.bind(&binding)` precedes emission. `Binding` holds the same
compiled Model's named World values, complete start rows and checked catalog; a value outside
its admitted type refuses at the source declaration. TLC
configuration values must be integers in `0..=i32::MAX`, so negative World integers refuse too.
Whole-state Init preserves correlations, deduplicates identical rows and admits empty starts.
The selected World constants have the same assignments in the main, invariant,
postcondition, negative and progress configurations. `parameterized_world` exercises three
selections (two checked against TLC, one with empty starts), private-field values and
same-World replay of a deliberately false claim.

Under a declared World, the lifter also admits action inputs: tuple and named-field variants
whose `bool`, admitted-integer or local fieldless-enum fields declare `#[inputs(lo..=hi)]`,
`#[inputs([..])]`, a `#[derive(Finite)]` domain or a field `#[domain(f)]`. Source reads only the
signature of `fn f(&World) -> impl Iterator<Item = T>`; the binding supplies its values for the
selected World together with the checked native catalog, exported by the derives. Binding
refuses a missing, duplicate or mistyped domain, a domain or catalog above 4,096 actions, and
any catalog that differs from the source declaration-order product in count, variant, field
order or value. Positional duplicates stay in the catalog, and an empty
domain keeps its variant with zero actions. Arms bind payload fields by tuple or named
patterns (with `_`, `..` and literal or constant subpatterns), guards and effects see them
through their own typed names, and a payload name shadows a World parameter of the same
name. Each variant emits a `_Inputs` table of `<<ordinal, values..>>` rows, its relation at
one position, and finite existential choice over all positions. Every edge records the
concrete catalog ordinal in the private `__umpire_input` observer, so replay takes the native
action at that ordinal and requires its typed identity to equal the emitted one; display text
is never parsed. Assumption lists name exact actions, such as `fair([Worker::Poll(2)])` or
`User::Paint { color: Color::Green }`; a bare payload variant is refused, and a value outside
the selected catalog refuses as the native checker does. `only` and `without` restrict the
variant relation to the admitted positions, and each fair payload gets its own `WF_vars`
over that exact position; no obligation ever ranges over a variant's union. Shared
`action_inputs` exercises a World-sized `Poll` attempt, a `bool` retry input and an empty
skip domain in two Worlds, including stale self-loops that cannot discharge fairness owed to
the current payload and mutated emissions that TLC must disagree with.

This checks the source-lifted scope independently. Complete native set/verdict agreement is
a separate fixture obligation; a cut native exploration cannot establish it. The lifter admits
one source form, `#[machine]` attribute Models with scalar/enum fields; source that implements
`Machine` by hand is refused at its `impl Machine` item. Checked integer operations use signed TLC-representable
Rust types (`i8`, `i16`, `i32`, `u8`, `u16`); wider types refuse by source location.
Attribute Models also admit explicitly imported `imbl::Vector<T>` with scalar/enum elements:
`new`/`default`, `len`, `is_empty`, guarded `push_back`, and safe `get` queries
(`is_some`/`is_none` or `copied().unwrap_or(default)`). Growth requires an authored finite
length guard; arbitrary collection methods, indexing and panicking `unwrap` refuse. Rust
computes lengths in `usize` and TLC in 32-bit integers, so `+` and `*` on a length refuse at
their span (fn-242); comparison, `-`, `/` and `%` mean the same in both and are admitted.
Action inputs without a declared World, generic or aliased input types and custom `Finite`
implementations, generic delegating adapters and arbitrary Rust remain outside the subset.

Phase access, rejection and closure laws are admitted together (fn-227.11). `impl
umpire_dsl::Phase` for a local fieldless enum and `impl Phased` for the State admit their
pure methods, and `Phased`'s default `closed` is read from `umpire-dsl`'s own source; other
trait impls refuse at the impl. A module with `reject!` has the stock `Outcome`, emitted as
the finite strings `Outcome::Accepted` and `Outcome::Rejected(rejection)`. Only the unqualified,
imported `reject!(r)` is read (a qualified `umpire_dsl::reject!` refuses, as does a
qualified `path!`); it records the first
rejection, returns from its function (an `#[effect]` helper's rejection returns from the helper
alone), and later effects apply only where the step still runs. A rejection is an enabled self-loop, never a dropped branch, and a
rejection after a state change fails TLC with an assertion, as `checked_step` refuses it
natively. Postconditions take `&Outcome`, compare whole states field by field and build
`Outcome::Rejected(..)` values. A `closable::` law call is lowered from the law's source in
`umpire-dsl/src/closable.rs`, with its generic state, outcome and value arguments bound; no
law is recognized by name. Shared `phased_rejection` breaks three terminal transitions and claims that every
step leaving the job open is accepted, so TLC and the native checker agree on each refutation and replay the
counterexamples through rejected edges and exact payloads; stale rejecting claims cannot discharge current-claim
fairness. The activity Model is admitted and checked by TLC in the required gate, and its
tests show TLC refuting reopening, accepted-closed-step and wrong-rejection mutants of its
source. An inherent method that shadows a `Phase` or `Phased` method refuses, since library
bodies call the trait's.

Admitted steps and embedded constructor data stay linear in sequential conditional returns
and repeated updates (fn-227.16). Each branch carries ordered single-assignment definitions;
later statements reference those values through typed local leaves, and emission binds them
with hygienically allocated `LET` names inside the branch and its payload scope. The outcome
keeps the first reached rejection, and checks retain their original execution guards. The
former 10,000-node refusal is removed: compilation controls embed 32 sequential returns and
64 repeated updates without expanding earlier values. Large repeated expressions within a
step still use the emitter's existing shared `LET` definitions.

The default gate needs only the Rust toolchain and workspace dependencies. The optional `cargo-insta`
CLI supports interactive review and acceptance (`cargo install cargo-insta --locked`). Its real
pending-to-accept integration control runs explicitly from `model/` with
`mise exec -- cargo test -p umpire-testing --test consumers accepts_generic_pending_snapshots_with_cli -- --ignored`.

## Dependencies

Only admitted crates (fn-210 R18): a release in the last 12 months, or finished and very widely used.
They are declared once in `[workspace.dependencies]`; a crate opts in with `name.workspace = true`.
`deny.toml` bans the unmaintained alternatives (`im`, `derivative`, `proc-macro-error`,
`proc-macro-error2`, `deluxe`, `index_vec`, `stateright`); `cargo deny check bans` enforces it. A new
dependency needs the same check and a line in the spec. Native tools use the admitted libc,
rustix/tempfile, toml (std/parse/serde), sha2 and tar/flate2 packages (backend jar admission
reuses sha2); current lockfile and [fn247 inventories](../.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task6/dependency-inventory.json)
record versions/features, transitive R18 accounting and guest-closure separation.
fn-248 added darling (with ident_case), heck and prost-types and removed enum-iterator; its
[task4 inventory](../.flow/artifacts/fn-248-off-the-shelf-crates-and-consolidation/task4/dependency-inventory.json)
records them.

## Configured activity proposals

`Host.Propose` uses method 6. The Temporal guest accepts a typed
`StartActivityExecutionRequest` in `ResetRequest.configuration`; namespace, activity identity,
task queue, retry policy and application input remain caller-selected. Reset returns an opaque
`WorkerProgram` for that World. The table enumerates every scenario and admitted attempt through
one response policy. An unconfigured Reset returns no program, and an empty Reset on a configured
guest retains its configuration. `umpire program --model <guest.wasm> --world <any.binpb> --out
<program.binpb>` provisions the same program without starting a Run.

The first proposal is BOOTSTRAP with an empty action. Propose fills absent declared identifiers
using its seed and stamps the chosen scenario into the `umpire-scenario` header. Successful Start
evidence for that exact request installs the World and immutable per-run receipt. Missing, failed
or crossed evidence cannot admit. Configuration, admission, pending requests and Model state change
only after response publication.

Cancellation is an Action proposal wrapped in `TemporalInstruction`, whose seeded hold requests
that worker polls wait for the request acknowledgment. An `ObserveActivity` is a READ proposal
with no Model action; its poll interval, read timeout and backoff are 10, 30000 and 10 milliseconds.
Worker delivery and responses are projected from Describe observations. Temporal Answer refuses;
workers receive their replies from the program. Rejected Cancel self-loops remain eligible after
closure. The executor integration and Driver hold mechanics follow in fn-243 task 2; at this
protocol boundary the Driver refuses unsupported programmed instructions as Driver faults.

Realization authoring uses `#[propose]` on one `(&self, state, seed)` handler returning typed
`Proposal<Action>`. Optional `matches = method` and `continue_polling = method` arguments delegate
correlation and completion hooks; omitted hooks retain the core's fail-closed correlation and
final-observation defaults. The polling hook receives the prospective state and optional typed
pending action. This changes no Model syntax or transition semantics.

Trace observer names and predecessor-field names are owned by `umpire_tla::observers`.
`EdgeObservers::decode` and the shared `umpire_tla::replay::step` select the canonical
concrete input, execute the chosen native action once and compare typed successor,
outcome and ordered facts. Native callbacks keep the TLA crate independent of the
DSL and checker. TLC retains tick/cycle/progress projection; Apalache retains tagged
stutters, before-state history, arithmetic boundaries and selected-claim confirmation.
The DSL's generated `__umpire_*` Rust methods and the emitter's hygienic TLA locals
and arithmetic operators are internal helpers, separate from trace observer fields.

The combined gate uses one mise DAG: root tasks import the framework task file under `umpire/dev/xtask/src/gate/tasks.toml` and add Temporal stages. Public `model-check` enters the Temporal gate once; framework `cargo xtask gate` remains independently runnable. Cross-workspace `wait_for` edges constrain already selected work without causing a framework-only invocation to select Temporal tasks. Host admission and owned cleanup span the whole invocation, including the isolated child; stage receipts retain scoped identities for later budget enforcement.

The Temporal prerequisite stage invokes the existing Make producer leaf under native host admission: root Go generator pins and `proto/api.binpb` are prepared before Temporal guest/API consumers. The leaf runs with `make -j1`; no mise task freshness substitutes for execution. API build scripts retain their own descriptor freshness/link checks. Initial compilation of the native command itself remains caller preparation with the required-mode bootstrap job cap.
