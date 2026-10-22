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
make model-time    # time the actual gate; note overruns of 10 s inner-loop / 60 s gate soft targets (R30)
```

The gate also checks resolved core capability restrictions and their negative controls,
the generic source boundary, both protobuf modules' freshness, the wasm32-wasip1 core
and guest builds, committed Temporal guest freshness, and CGO-disabled Go host parity for both modules.
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
make umpire-check-model-gaps
make umpire-check-live-tests
```

Slice and gaps use `temporal.Guest()` from the committed
`temporal/hosts/go/umpire_temporal_guest.wasm`; they build no Rust guest.
The slice requires Complete, Fail, HostRefusal and Cancellation to pass. The umbrella invokes both before its existing
legacy selector. Seven inherited legacy failures remain explicitly red; see DESIGN's
Mini activity slice section. This bespoke proof does not execute a Testpilot Case.

The Temporal host module's `go test -tags test_dep ./...` needs only Go, with no
guest environment variable or Cargo target lookup. Model authors refresh the embedded
artifact with `make umpire-guest`. The generic builder's `--check` compares bytes during
`make model-check`, reporting stale sizes and the first differing offset. Missing pinned
build prerequisites warn and skip only this freshness step when `CI` is unset; set `CI`
(including an empty value) fails. Actual compile failures and stale bytes always fail.
The full Rust authoring gate and the umbrella's legacy IR bridge still require Rust.
Umpire ships no prebuilt stock/control guests. See [GUEST_BUILDS.md](GUEST_BUILDS.md).

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
    pub struct World { initial: u8, limit: u8 }
    #[derive(State)]
    pub struct Model { pub value: u8 }
    #[derive(Action)]
    pub enum User { Advance }

    impl Model {
        #[starts]
        fn starts(w: &World) -> impl IntoIterator<Item = Self> {
            [Self { value: w.initial }]
        }
        #[on(User)]
        fn user(&mut self, _: User, w: &World) {
            if self.value < w.limit { self.value += 1; }
        }
        #[invariant]
        fn bounded(&self, w: &World) { check!(self.value <= w.limit); }
    }
}
```

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

Declared Worlds with admitted scalar/enum constants also support independent source lifting.
Only values cross from native code to the lifter: inside a declared-World `#[machine]`,
`derive(World)` and `derive(State)` add a hidden `Values` impl listing the named field values,
private fields included, and `derive(Action)` a hidden `ActionValues` impl giving each action's
variant and payload and each `#[domain]` field's values for a World. Field visibility is
unchanged; the export shows what the derived `Debug` already prints. `#[machine]` exports each
local fieldless enum's variant, as does `derive(Finite)`. Sequence (`imbl::Vector`) state is not
exported yet, so it cannot be bound; `Value::Sequence` is its slot. A field type without a value
leaves the Model compiling and is refused only where a binding is requested. Admission still reads the authored `model.rs` on its
own and translates every transition and claim itself. In a shared fixture,
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
independent Model. Follow the [annotation guide](temporal/annotate/README.md) for
derive mirrors, typed projections, constraints, roles and exhaustive bindings;
[Annotation catalog](DESIGN.md#annotation-catalog) explains admission and the API bump workflow.
Use the activity catalog as the concrete example. Request constraints need an expectation
including intentional server acceptances. Keep ignored fields
explicit with reasons and validate responses without rewriting their bytes.

Verify with `make model-check`; activity transport/catalog changes also require
`make umpire-check-model-slice` and `make umpire-check-model-gaps`. The latter compares
actual live probes with `temporal/feature/activity/expected/gaps.txt`. Review unexpected
Accepted/Mismatch/Unannotated rows and explicit mutation refusals before changing expectations
or goldens. Its external live golden is checked by the gap gate, not regenerated by
`make model-update`.

## Mini activity slice (fn-216)

The independent `temporal/feature/activity/model.rs` fixes a six-phase restriction
of the archived Product: Scheduled, Started, Completed, Failed, Canceled and TimedOut.
Its six actions now give one start, six reached states, 11 edges, 36 evaluated actions,
25 refusals and depth two without a cut. Cancellation is atomic and failure fatal-only;
there is no cancelRequested state, retry branch or real clock. Successful Start supplies
Scheduled outside this graph. This is not an exact port of the archived seven-phase Model.

`terminal_remains` and `closed_rejects` are registered postconditions, delegating to the
shared closure laws. Fn-221 adds a rejected cancel self-loop at each of the four terminal
phases, giving both laws terminal edges to check. The controls also enumerate every
terminal phase/action and prove that a hypothetical reopening, accepted closed step or
wrong rejection refutes the corresponding law.

The reusable `umpire-wasm` library owns state, replay, first-Unexplained freezing,
Reset, response-publication transactions and ABI buffers. A system-owned `Realize`
binding proposes paths; the runtime validates those paths through actual Model steps.
The activity binding uses Temporal-owned `#[realization]`, `#[observe]` and `#[answer]`
attributes over typed handlers. Their declared field paths distinguish observed responses,
incoming callbacks and produced partial SDK requests; omitted present fields refuse.
See the [realization guide](temporal/realization/README.md) for explicit state projection,
`Step`/`Reply` returns and the raw typed request packing contract.
The registered Temporal fixture consumer checks `expected/realization.snap.txt` against
the generated runtime metadata. It distinguishes observed, callback and produced contexts,
lists every classified path and ignore reason, and derives unhandled Model-action RPCs from
the admitted API bindings. [Declarative Realizations](DESIGN.md#declarative-realizations)
records the report contract and complete-file source counts. After committing a coverage
checkpoint, run its real added-catalog-field control from `model/` with
`mise exec -- cargo test -p umpire-temporal-tests --test fixtures added_catalog_field_fails_the_real_coverage_golden -- --ignored`.
It rebuilds in a disposable worktree and preserves the existing coverage baseline.
For public states, `#[machine]` exports `World` and `Action`, and `world()` returns
`World`. Non-public states retain the opaque interface with generated types following
the state's visibility. The public activity world removes
the binding's duplicate action carrier, delegating machine and claim-index re-wrap.
Generic Any helpers derive canonical type URLs from generated `prost::Name`; only the
Describe and cancel RPC method paths are binding constants. The policy parser and reply constructor
keep the remaining binding specific to Temporal.

Exactly three envelopes live under `temporal/proto`: Observation carries a method,
a response Any or error, and an optional actual-request Any at tag 4, Callback a task Any,
Reply a request Any. The real upstream API descriptor
closure is compiled natively and for WASM, with generated well-known types owned once
by `umpire_wasm::google::protobuf`. The closure's file count is not a schema invariant.
No Testpilot schema is moved, imported or copied. Shared WKT ownership means checker
rebuilds can trigger generated-API compilation; a separate owner is a possible later
build optimization, not a new dependency admitted here.

Describe projects actual Info.Status and Info.RunState: Running/Scheduled or
Running/Started, and terminal status/Unspecified. There is no Started execution-status
enum. Only a valid callback evidences Poll. Repeated current observations are zero-step
Explained; premature terminal and stale Scheduled observations are Unexplained, freezing
the run until Reset. The SDK callback supplies a partial poll projection containing
exactly one binary/plain Complete or Fail payload. Answer selects a partial completed
or non-retryable failed RPC request, leaving the Model Started until terminal observation.
Task token, identity, timestamps and whole-request equality are outside this projection.

The live owner judges Scheduled with its registered worker stopped, answers delivery
through that run's one Model instance, then synchronously reads and judges Describe.
This ordering prevents a pre-delivery read from arriving after delivery; it does not
solve general concurrent read-window inference. Complete and Fail, reset/isolation,
repeat observations, opposite-terminal mutation, host refusal and cancellation all have
real server/SDK controls. The worker returns only decoded Model reply data and uses the
SDK failure converter. This is a bespoke SDK/server/Model proof, not execution through
Testpilot Prepare/PreparedCase.Run, Case, Program or Contract.

Remaining DSL gaps include source lifting of `Finite` and World-bound inputs, facts and typed
projection authoring.
Fn-221 closes the [rejecting-outcome gap](DESIGN.md#rejecting-outcomes-and-live-cancellation-fn-221). No Model transition
was changed to fit server evidence. Fn-213 must include the Temporal fixture root;
fn-218 retains Cases(8), resolving impossible alternatives as a downstream grammar
decision rather than inventing edges or silently shrinking coverage.

For R33 this proves the Model protocol and opaque payload seam, preserving old wire
bytes and refusal codes. Broader Case identity/versioning, Program replay and removal
of Contract/CEL remain open. R34's size reduction, R35's CEL removal, R36's deferred
lifting decision, R37's general callback replacement, R38's unified host loop, R39's
Run/storage redesign and R40's polyglot forwarding remain the obligations in DESIGN's
[Case-plan table](DESIGN.md#recommendation-and-next-slices); this slice does not discharge them.

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

The unchanged-scope umbrella invokes the dedicated slice and retains all legacy checks.
Its seven inherited failures are AssessRecordedRuns, ExplorationDiscoversUnpinnedExecution,
GeneratedCases, NexusControlForgedCompletionIsViolated, NexusControlReplaysThroughTheCommand,
NexusPairCase and UmpireRunRunsACheckedInCaseAgainstAnyEndpoint (all TestTestpilot-prefixed).
The comparison in `.flow/tmp/fn-216-4/legacy-comparison.md` retains this task's
before/after identities and assertion causes, matching predecessor .3's comparison.
This umbrella is red; the dedicated slice passing does not make it green. The frozen
Scala tree was not revived to repair those legacy expectations.

## Testing

Models are read once, in `umpire-front` (`umpire/front/`). `declare/` reads the declarations:
attribute kinds and their order, registration, the State/World/actor schema, starts, input
domains and `&World` parameters, and `path.rs` the `path!` language. The macros in
`umpire-dsl-macros` only generate from what `declare/` read, and report its refusals as compile
errors. `shape.rs` takes the same declarations and admits the subset, delegating each body to
`body/` with a typed environment; these refusals are soft, since the Model still checks natively.
Body lowering reads no module items; `tree.rs` holds the admitted tree and `expr.rs` its typed
expressions. Its `Diagnostic` carries line, column and message; `umpire-lift` reads the source
files, calls `umpire_front::read`, names the file in each refusal, gives the tree its TLA+ names
(`names.rs`), binds the selected World (`bind.rs`) and emits TLA+.

End to end, by golden files (fn-210 R8, R23, R29). A test is a fixture, an input plus the output it
produces (`model.rs`, `tests.rs` and `expected/`). "Fixture", not
"case": Case is Testpilot's term.

- behaviour: each fixture's `expected/report.snap.txt`, `states.snap.txt`, `rows.snap.txt` and `paths/<claim>.snap.txt`
  (`insta`; review with `cargo insta review`);
- refusals: `trybuild` `.rs` + `.stderr` under `umpire/dsl/tests/ui/`;
- expansion: insta snapshots of each macro's implementation function, formatted by
  `prettyplease`;
- one property test against a brute-force reference explorer, the check that goldens are right
  rather than merely stable.

Goldens are deterministic, human-readable and timing-free. `make model-update` rewrites them; a
shared fixture folder without `model.rs`, `tests.rs` or `expected/` fails the build naming it. Unit tests only when a defect cannot be
written as a fixture.

`umpire-tests` at `umpire/tests/` owns 12 shared fixtures in `fixtures/<name>/`, all
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
refusals remain in `umpire/checker/tests/fixtures/`, together with the manual-trait Models the
attribute surface cannot state (DESIGN.md lists each with its gap). Among them,
`sample_kettle_trait` is the independent manual-trait engine reference: its Model is
`kettle.rs`, which the stock wasm guest includes, and its report, states and rows goldens are
the Go host's expected payloads. DSL formatting fixtures remain with the DSL.
The historical authoring sketches have been retired after preserving their comparison in
[DESIGN.md](DESIGN.md#historical-authoring-options); there is no samples crate.

`umpire-suite` at `umpire/suite/` is what a Model's own `tests.rs` uses (fn-213). An example is
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
in `umpire/suite/tests/fixtures/`.

### Test-space patterns

`find`, `cover` and `Run::along` accept the full `path!` grammar. Adjacent state predicates
consume no steps; each action consumes exactly one step. Only an explicit gap admits
intervening actions. Search may reach the first waypoint from a Space's starts; an initial
concrete state anchors `Run::along`. A story must have one authored alternative; alternatives
refuse with “use cover”.

| Pattern | Meaning |
| --- | --- |
| `state`, `true`, `\|s: &Model\| s.closed()` | Exact state or state predicate |
| `Worker::Poll` | One action |
| `User::Send { amount: 0..=3, from: (king \| queen) }` | One alternative per finite input value/combination |
| `User::Send { amount: 2, from: _, .. }` | Exact input; `_` and omitted fields use declared domains |
| `Deadline::*` | Each action/input value of this actor |
| `Client::Cancel => Outcome::Rejected(Rejection::NotFound)` | Action with a typed outcome filter |
| `Client::Cancel => (Outcome::Accepted \| Outcome::Rejected(_))` | One alternative per outcome pattern |
| `(Worker::RespondCompleted \| Worker::RespondFailed)` | Authored alternatives, in order |
| `[Worker::Poll; n]`, `[Worker::Poll; 0..=1]` | Exact repetition or one alternative per inclusive count |
| `unordered![a, b, c]` | Every ordering of the listed patterns |
| `..`, `..=n`, `..!(Client::Cancel, Worker::Poll)` | Any steps, at most n steps, or steps excluding the named actions |

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
`cd model && mise exec -- cargo test -p umpire-tests`:

```rust,ignore
let covered = cover(&Space::from([room()]), Cases((80 + 1) * 2),
    path![room(), User::PressButton, [Heater::Heat; 0..=80], ..,
          (User::DepressButton | Heater::Failover)])?;
```

The explicit gap supplies additional Heat actions before Failover for prefixes shorter than
80. The 162 alternatives describe explicit prefixes, not total heat counts; expansions show
every actual step. The literal adjacent form is retained as a refusal control. Activity uses
`[Worker::Poll; 0..=1], ..` followed by Cancel, RespondCompleted, RespondFailed or StartToClose:
eight choices. Zero explicit Poll can still need a gap-supplied Poll before completion, failure
or StartToClose. ScheduleToStart is not a fifth closing choice. Run it with
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

To write a Model's tests, add `mod examples` and `mod explorations` to its `tests.rs`, each with
`use umpire_suite::*;` and plain `#[test]` functions returning `Result<(), Refusal>` so a refusal
fails the test with its message; `cargo test` and the gate run them, nothing registers them
(`umpire-suite` is a dev-dependency of the crate that compiles the fixture). Kettle's
`umpire/tests/fixtures/sample_kettle/tests.rs` is the example: it asks the questions of the archived
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
roots (`umpire/tests/fixtures/`, `temporal/feature/`) also require `tests.rs`, while synthetic
control roots keep the hook in `model.rs`. Expected files
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

For a declared World, admit the source first, then call `lifted.bind(&binding)` before emission.
`Binding` contains ordered named World literals and ordered complete start rows; missing,
duplicate, extra, wrong-type and out-of-range values refuse at the source declaration. TLC
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
length guard; arbitrary collection methods, indexing and panicking `unwrap` refuse.
Action inputs without a declared World, generic or aliased input types and custom `Finite`
implementations, generic delegating adapters and arbitrary Rust remain outside the subset. `Phased`,
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
