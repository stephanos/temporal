# Umpire in Rust: decision

The workspace now lives at `model/`, with the frozen Scala reference under `model/archived/`.
Run `make model-check` and `make model-time` from the repository root. Measurements below retain
the command names used when they were recorded, before the workspace move.

## Front end and back ends (fn-238)

Both checkers read a Model through one front end and differ only in their back end (owner,
2026-10-21; [design and evidence](design/shared-front-end.md)).

- **Front end.** `umpire-front` reads each `#[machine]` module once, inside the macro: attributes,
  registration, schema, starts, input domains, `&World` and `path!` (`declare`, `path`), then the
  body subset (`shape`, `body/`). Structural refusals are compile errors; subset refusals are one
  soft located `Diagnostic`, the same for every consumer, and the Model still checks natively.
- **Admitted tree.** `umpire-ir` holds the typed tree and the module's `Source` identifiers.
  The macro embeds it as data (`__lifted::Lifts`, off `wasm32`, hidden, never checked in).
  `umpire-ir-derive::Print` derives constructors from its fields and variants; `ir::print`
  retains shared scalar/container printing. A new IR field needs no printer registration.
- **Native back end.** `umpire-dsl-macros` generates the executable Model from the front end's
  declarations; the native checker explores the compiled bodies.
- **Emission back end.** `umpire-lift` emits TLA+ and `.cfg` from the embedded tree with the
  same-process World, starts and catalog. It reads `tests.rs` for hook claims, and `model.rs` only
  to locate a refusal when the checked World embeds no `#[machine]` (`kettle_cut`,
  `kettle_progress`, `progress_dead_end`). fn-211's Apalache projection will be another emitter.
- **Compiled cross-checks.** The wasm guest and typed replay of every TLC counterexample run the
  compiled `checked_step`, so rustc's reading of each handler still checks the front end's.

**Independent source admission is retired for new work** (owner, 2026-10-21). fn-212, fn-227 and
fn-211 had the lifter admit source on its own with no shared interpretation; that is traded for
one reading, accepting that a front-end misreading reaches both checkers. Typed replay through
compiled bodies and TLC agreement on complete reached sets and verdicts are the cross-checks.

**Size.** Physical `src/` lines per crate, from the repository root:
`git ls-tree -r --name-only <rev> -- model/umpire/<crate>/src | grep '\.rs$' | while read f; do git show "<rev>:$f"; done | wc -l`.

| Crate | Before .2 (`29dbe75072`) | After .2 (`1511cf5aeb`) | After .3 (`3fa7390dad`) | After .4 (`0bbeb1ff35`) |
| --- | ---: | ---: | ---: | ---: |
| `dsl/macros` | 3,306 | 3,306 | 2,156 | 2,439 |
| `front` | — | 3,103 | 4,253 | 3,844 |
| `ir` | — | — | — | 327 |
| `lift` | 4,884 | 1,852 | 2,098 | 1,687 |
| **Core crates** | **8,190** | 8,261 | 8,507 | **8,297** |

The design estimated −850 to −1,600; the core crates grew **+107** (+239 with `testing`, 1,538
to 1,670). Its lifter duplication column mostly counted lowering with no macro counterpart, so .3
moved the macros' 1,150 reading lines into `front` and collapsed only about 150 lifter lines.
Body lowering (about 1,380) and emission stay by design; embedding costs about 420 (printer 277,
`Source`/`Embedded`, hooks, probe) against `build.rs` (−241) and binding re-validation (−200).

**Per-construct cost** halves only for declaration-level constructs (an attribute, claim kind,
World or input feature), now read once: the historical design estimated 55–60% of fn-227's
World/input costs, without measurement. Body-level constructs still need admission, lowering
and emission. fn-227.11 delivered `Phased`/`reject!`/closure laws through those layers;
constructor printing is derived since .9. Quint parity is mostly
body- and value-level; a new back end such as Apalache costs only its emitter.

**Derived printing** (fn-238.9). The 277-line handwritten printer is replaced by a
67-line proc macro and shared scalar/container implementations in `umpire-ir`.
Across the changed Rust files, production source shrank 60 lines and the field/variant
control added 42 test lines (18 net lines removed); this misses the roughly −190 estimate.
The retained scalar/container code, constructor path annotations and the derive offset most
of the deleted record registrations and enum arms. Existing expansion and semantic goldens
remain byte-identical; a scratch oracle compares old and derived constructor tokens for
26 Models, including admitted trees and located refusals.
The required-TLC full gate took 532.677 s and the inner loop 9.272 s. The gate
exceeded the 60 s soft target; the inner loop met 10 s. A first attempt timed out
at 600 s during cold guest compilation, after rebuilding trybuild closures.
The completed run still includes guest/host fixture compilation and is not a
steady-state comparison with the earlier default-gate timings.

**Option B deferred** (owner, 2026-10-22). An interpreted checker (.5, .6) would add an
interpreter case and a `Value` bridge per body-level construct on top of lowering and emission
(+35–55%, the archived Scala's "written three times"), while native speed limits no gate. It
returns on a concrete need: checker speed, or a construct rustc cannot run.

**Interpreter as a differential check** (fn-238.5, reactivated 2026-10-23). `Interpreted<M>` in
`umpire-checker` is a `Machine` that evaluates the embedded tree: branch guards in written
order, the checked-arithmetic checks with Rust's own overflow and division failures, updates,
and `reject!`/`stay!` outcomes, then claims, progress evaluations and assumptions. State stays
the compiled `M::State`; `#[machine]` generates `Interprets` accessors beside `Lifts`, off the
wasm guest, and a table of each `check!`'s text that the tree's new `ExprKind::Check` indexes
(emission reads through it, so TLA+ is unchanged). The catalog, claim order and computed or
World-bound starts stay compiled. Every fixture whose World embeds an admitted Model (18 shared
fixtures, two checker controls, the lifter's seven Models and a checked-arithmetic control)
explores identically both ways, and two seeded mutant trees are caught. First rough release
timing of `Checker::explore_within` (shared, loaded 12-core host, medians): Kettle 88 µs
compiled against 228 µs interpreted (2.6×), the 2,001-start wide shape 1.48 ms against 3.84 ms
(2.6×). That is above .6's 2× gate; see `.flow/tmp/fn-238.5/handoff.md`.

**Guest identity** (fn-238.8). .2 changed the Temporal guest without a code change because each
crate's `-C metadata` hashed the whole `Cargo.lock`; it now hashes the crate's own lockfile
closure (`scripts/guest-lock.py`), and a control shows an unrelated crate leaves both guests
identical. .3 and .4 refreshed it because the macros and `umpire-dsl` gained front/ir dependencies.
fn-239 renewed Linux/macOS certification at `a96a4c6550`; all twelve byte comparisons and both host suites passed.

**Timing.** Warm `make model-time` receipts (soft targets 60 s gate, 10 s inner loop):

| Receipt | Gate total | Inner loop | Log under `.flow/tmp/` |
| --- | ---: | ---: | --- |
| fn-227.14, before .2 | 50.532 s | 7.908 s | `fn22714-full2-model-time.log` |
| .2 (fn-227.12 lane running concurrently) | 119.588 s | 19.589 s | `fn2382-full2-model-time.log` |
| .8 merged | 50.199 s | 6.128 s | `fn2388-merge-model-time.log` |
| .3 | 51.307 s | 8.822 s | `fn2383-full-time.log` |
| .4 | 47.592 s | 7.272 s | `fn2384-full-model-time.log` |

No warm regression. Touching `dsl/macros/src/lib.rs` then `cargo build --tests --workspace`
took 14.56 s (`fn2384-compile.log`); no pre-.4 counterpart was measured.

## TLC source backend (fn-212)

`umpire-lift` emits modules, configurations and typed decoding metadata from the embedded
admitted Model at test time; it never calls the native explorer. fn-223 deleted the manual-trait
lowering (the lifter went from 5,048 to 2,978 lines) and what only that form produced: explicit
start lists, tuple-state labels, multi-file inputs, and per-branch choice indices and facts.
The emitted observers keep `__umpire_choice = 0` and `__umpire_facts = <<>>`, so modules and
replay are unchanged; choice and facts must be re-added with the attribute surface that states
them. The tree keeps `outcome` (`reject!`) and `preserve_self_loop`, which `stay!()` sets
(fn-224). Source that implements `Machine` by hand is refused at its `impl Machine` item
(`progress_dead_end`'s lift golden pins the message). Unsupported syntax, references,
conditional compilation and generated-name collisions produce located warnings and refuse emission.

`umpire-tlc::Tlc::check_module` executes that module without requiring a native graph.
It retains the unrestricted complete reached set and separately checks every registered
safety, postcondition, scoped progress and supplemental negative property. Property-specific
observer scopes do not replace the unrestricted set. Native agreement is a separate testing
layer: it requires complete sets and every verdict to agree, and replays returned traces
through actual Model steps, including action, choice, outcome and facts. A native cut cannot
certify agreement. A control deliberately cuts native exploration at one state while the
same lifted Model reaches its complete four-state set in TLC.

The observer encoding from [the decision](design/second-opinion.md) is exercised with real
TLC controls for idle versus maximal paths, acyclic progress without fairness, finite dead
ends, authored self-loops, rejecting outcomes and facts, continuous/intermittent enabling,
unrestricted prefixes and transition bounds. Only authored fair actions receive weak
fairness. Terminal completion keeps deadlock checking enabled without counting a Model edge.
Decoding removes exactly the declared private fields, refuses collisions and unknown fields,
and deduplicates the projected set. Arithmetic controls cover ordered updates, checked
overflow, signed division/remainder including minimum integers, and empty starts.
The fixed `imbl::Vector` table covers scalar/enum elements, empty construction, length,
emptiness, bounded guarded appends and explicitly safe lookup. A separately authored
collection control compares the complete native/TLC sets and verified/refuted properties;
unsupported methods and unguarded growth retain located refusals.

The pinned official artifact is `tla2tools` 1.7.4 (TLC 2.19), SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`, on mise Temurin 27.
Missing or changed tools, subprocess failures/limits, incomplete dumps and disagreement
are refusals, never passing or silently skipped checks. Each Java process owns its temporary
library extraction and state directory, avoiding concurrent JVM extraction races.

The shared fixture corpus now has 16 admitted TLC golden selections: attribute Kettle
at 283 projected states, `changed_state`, ten attribute progress graphs (four with
accepted self-loops), `parameterized_world`, `action_inputs`, `phased_rejection` and
`shared_returns`. These are controls, not 16 independent products. Five sources retain
located refusals: `kettle_cut`, `kettle_progress`, `progress_dead_end`, `rejecting_steps`
and `accepted_self_loops`. The first three lack an admitted machine; the latter two
still contain unsupported declarations or outcome expressions. `reject!` itself is
admitted, including supported helpers, phase access and stock closure-law calls.
The system-owned activity root also has actual TLC agreement for selected Worlds 1 and 3.

Trait-form shared fixtures (fn-223). Each manual `Machine` fixture that shared the corpus was
rewritten as a `#[machine]` Model or moved to `umpire/checker/tests/fixtures/` (no TLC
goldens), so the lifter needs to read one source form. A rewrite keeps every behaviour
golden byte-identical except two spellings: state `State { 0: n }` → `Model { value: n }`
and action `Action::X` → `User::X` (or `Environment::X` for the internal actor). Its TLC
agreement golden is unchanged; its emitted module and configuration are regenerated
(`State.tla`/`.cfg` → `Graph.tla`/`.cfg`). A move keeps its checker goldens byte-identical
and drops its lift, module, configuration and TLC goldens. Nothing was removed. The gaps
are attribute-surface facts for a later spec, not filled here:

| Fixture | Disposition | Reason or surface gap |
| --- | --- | --- |
| `sample_kettle_trait` | checker-only | Independent engine reference (owner, fn-210.7); the wasm guest includes its `kettle.rs`. Report byte-identical to `sample_kettle`; its TLC coverage (283 states, same four claims) stays pinned by `sample_kettle`. |
| `choices` | checker-only | One action yields three ordered choices with `bool` outcomes and string facts, two to the same target; handlers emit one edge with unit or `Outcome` outcomes and no facts. TLC loses choice/outcome/fact replay. |
| `edge_checks` | checker-only | `bool` outcomes; its self-loop (`Idle` at 2) is stateable with `stay!()` since fn-224, but a rewrite would change every row's outcome spelling. `changed_state` keeps an `after`-filtered refuted postcondition with a replayed counterexample on TLC; the unfiltered refuted `never_decreases` and the `after`-filtered self-loop refutation are pinned nowhere on TLC. |
| `vacuity` | checker-only | Same `bool` outcomes as `edge_checks` (its self-loop is stateable since fn-224). Never-triggered `after` claims stay on TLC through `changed_state`. |
| `shared_starts` | checker-only | Duplicate starts `[0, 1, 0]` and `bool` outcomes (its self-loop is stateable since fn-224); The old field-start form enumerated a duplicate-free range. `parameterized_world` now tests computed correlated starts with a duplicate; TLC `Init` remains a set. |
| `deep_chain` | checker-only | State carries a shared `Rc<Cell>` clone counter and the hook asserts linear clone work; a checker property, already refused by the lifter. |
| `interleaved_claims` | checker-only | Claim order interleaves kinds (`claim_order`) and the hook adds a supplemental manual claim; attribute claims are contiguous per kind. Already refused by the lifter. |
| `progress_graph` (cycle), `progress_continuous`, `progress_intermittent`, `progress_fulfilled`, `progress_trigger_stops` | attribute | |
| `progress_two_lobes` | attribute | Catalog order is `Enter, Escape, Left, Right` (was `Enter, Left, Right, Escape`); `Enter`/`Escape` never fire, so every golden is unchanged. |
| `progress_self_loop` | attribute (fn-224) | Authored self-loop, written with `stay!()`. |
| `progress_multiple_starts` | checker-only | Historical field-start limitation; computed starts now support non-range lists, but this manual fixture remains checker-only. |
| `progress_target_barrier` | attribute (fn-224) | Self-loop after the target, written with `stay!()`. |
| `progress_excluded_prefix` | attribute (fn-224) | Self-loop inside the scope, written with `stay!()`. |
| `progress_scope_exit` | attribute (fn-224) | Self-loop beside an excluded fair escape, written with `stay!()`; `Escape` is the internal `Environment::Escape`. |
| `progress_action_choices` | checker-only | One fair action with two choices, one a self-loop. Holds the trimmed trait `Graph` that `progress_multiple_starts` also uses. |

The eleven `progress_*` fixtures selected a mode of the trait `progress_graph` Model, so they
were trait-form too. TLC no longer pins nondeterministic choice, non-`()` outcomes, facts,
starts outside a field range or an unfiltered refuted postcondition (`never_decreases`) in those
progress fixtures. Computed whole-state starts and rejecting outcomes now have separate controls
in the parameterized fixtures below; nondeterministic choices and facts remain gaps. fn-224 moved the four fixtures whose only gap was a self-loop back with
`stay!()`, so self-loops, refuted progress under a `without` scope (`progress_excluded_prefix`)
and a fair action that the scope excludes (`progress_scope_exit`) are on TLC again; their
report, rows, states and paths goldens kept fn-223's two spellings and TLC agrees on each.

Real TLC execution is selected by `UMPIRE_BACKENDS=require`; the default gate checks source
artifacts/refusals and explicitly reports no TLC execution evidence. Required shared runs
mandate attribute Kettle and every admitted fixture's selected TLC goldens. Both renders in
the existing determinism contract execute the backend. At fn-212 the required full gate took
144 s (101.60 s for the double render) against a 9.564 s warm default gate; single cached
observations, not a scaling benchmark. Current receipts are under the front end section.

## Stock-toolchain wasm verdict (fn-210 R24)

Recommendation: **Rust**. On 2026-10-18 the trait-shaped Kettle compiled with stock
Rust 1.99.1 for wasm32-wasip1 and ran in wazero v1.12.0 with Go 1.27.0 and
`CGO_ENABLED=0`. The generated protobuf Explore report and all ordered Rows matched
the existing native trait fixture goldens byte-for-byte: 81 starts, 283 reached
states, 424 emitted rows and depth 62. This is the decisive R20 slice before the
attribute macros. The final attribute surface is compared below. Lifting and a
TLC second opinion remain deferred to fn-212; neither is part of this verdict.

The three application exports are `umpire_buffer_alloc`, `umpire_buffer_release`
and `umpire_call`. Both sides use
generated messages from `umpire/proto/umpire/model/v1/model.proto` (renamed from
the original oracle namespace by fn-216); Refusal is a
case of the common Response envelope, separate from a checker verdict or trap.
The guest includes `umpire/checker/tests/fixtures/sample_kettle_trait/kettle.rs` directly.

Rust exports these names directly using the stock linker. The buffer-release name
avoids WASI libc's internal `free` symbol. The default gate compiles and runs the
real guest through the Go host, validates its export signatures, and rejects
missing, unexpected and mis-typed exports through compiled negative controls.
Direct locked Cargo builds need no custom linker or post-link processing.

## Reproducible guest builds (fn-230)

Linux unwind proof, 2026-10-21 UTC, source `594c13b237b3e3e5defcd67e7457257fa1db21ae`.
Linux 7.0.14/aarch64; Rust 1.99.1 (`58cf8f9c35b355199727627d07ae34a4e2846d67`,
LLVM 23.1.1), Cargo 1.99.1, Go 1.27.0, protoc 29.5. Two clean clones at different
absolute paths used initially absent independent targets and retained clean status.

| Guest | Bytes in each clone | SHA-256 in each clone |
| --- | ---: | --- |
| Stock | 192255 | `5ec57e2e5371723adf318665c1048b5a4021e9f4203ed56884c5f849e7a2dfc1` |
| Temporal | 3326027 | `1d52b188f7f582a3d1b78d5ed19f4ba8af4753bf43343aff0524854f1d713800` |

Both byte comparisons passed. Actual Tiny effect-panic and overflow fixtures match
their native full responses and existing report goldens across repeated calls and Reset.
The default gate builds the same unwind profile. [Commands and timing](GUEST_BUILDS.md).
fn-232 certified source `3c314fb5b85427471a1525d6f190fe5d1fb9f3b2` on 2026-10-21 using four clean clones after fn-236's compiler-input fix.
Linux 7.0.14/aarch64 and macOS 25.6.0/arm64 matched stock (192271 bytes) and Temporal (3330006 bytes); all twelve comparisons and both native host suites passed.
[Certification evidence](../.flow/artifacts/fn-232-linux-macos-guest-byte-identity/certification-3c314fb5b8/proof.md) records source/toolchain guards, independent targets, exact commands, sizes and hashes.
Temporal embeds the current asset; [build details and renewed proof](GUEST_BUILDS.md) cover freshness, cache identities and cross-OS verification.
fn-239 renewed that proof after fn-238.8 at `a96a4c6550b468c5d7c3011a7961d4e118dacd93` on 2026-10-22. Four clean clones matched stock (192489 bytes) and Temporal (3332314 bytes); all twelve comparisons and both host suites passed on each OS. [Renewed certification](../.flow/artifacts/fn-239-renew-the-linuxmacos-guest-byte/certification-a96a4c6550/proof.md) retains closure maps, commands, toolchains and results.

## Capability and panic boundaries

The real guest imports exactly checked `environ_get`, `environ_sizes_get`, `fd_write`,
`proc_exit`, `fd_close` and `fd_seek` from `wasi_snapshot_preview1`. The host provides
an empty environment, discarded descriptor 1/2 output and an exit trap. Close and
seek always deny with BADF (8), without descriptor access or output-memory effects.
It links no general WASI implementation, directories, inherited I/O, clocks or
thread APIs. Unexpected imports and imported memory fail before instantiation.

Core-local Clippy policies reject resolved standard filesystem, clock and thread
methods/types, including aliases, in both DSL and checker. Host build scripts
retain narrowly scoped allowances for fixture discovery. Negative controls prove
that each policy rejects representative methods and types, and the gate lints
the actual native and wasm core libraries. The linked-import check complements
this source restriction: it covers surviving transitive guest code. Neither is
a universal analysis of dormant third-party code, cfg-disabled branches or every
future API. Dependency changes still require admission and review.

The pinned compiler rebuilds std with scoped bootstrap and unstable final Wasm EH;
wazero's interpreter executes it. Checker-caught panics preserve native judgments.
Uncaught panics still trap and invalidate the instance; errors name the method and ABI.
Invalid budgets and state IDs remain generated typed refusals, not panic tests.

## Attribute Kettle and the authoring sketch (final flat form)

The canonical `umpire/fixtures/fixtures/sample_kettle/model.rs` uses the owner-approved
flat surface derived from the historical `Kettle.attrs.rs` sketch, retaining its separate
`User` and `Heater` enums, transition bodies and claim predicates.

Remaining differences from the hypothetical sketch are deliberate:

- Imports name the actual `umpire_dsl` crate instead of a hypothetical prelude.
- `#[machine]` registers one inline public `kettle` module with one `impl Model`.
  Per-item attributes mark methods, grouped contiguously in on, effect, invariant,
  postcondition, assumption and progress order. There are no group attributes or
  marker traits; attributes preserve authored statements and method/impl headers.
- The state is named `Model` to match Scala's `object Model`, replacing the
  sketch's `Kettle` name while leaving its fields and predicates unchanged.
- The module discovers action enums in declaration order rather than taking
  an actor tuple, preserving press, depress, heat, failover catalog order.
- Actor-level `#[internal]` replaces the hypothetical `Internal` derive.
- `State` supplies clone, equality, hashing, `Debug`, `Display` and ordered
  starts with defaults for unannotated booleans; `Action` supplies clone,
  equality, hashing, `Debug`, qualified `Display` and ordered enumeration.
- `#[postcondition(after = Heater::Failover)]` selects the failover action;
  its unused `before` parameter is omitted while the warming claim keeps it.
- The named assumption is a static function returning a module-local `Assumption`
  value through `without([User::DepressButton])`, replacing the associated
  constant and hiding action adaptation.
- `path![self.heating_on, .., self.beeping].assuming(Self::never_switched_off())`
  builds a module-local `LeadsTo` value from a two-waypoint path and names its assumption
  in the expression. The `whenever(..).eventually(..)` builder it expands to remains public.
- Public generated `World` and `Action` implement the existing `Machine`
  protocol for public states; `world()` returns `World`. Non-public states retain
  the opaque return interface and their generated types follow the state's visibility.
  The activity guest names its public world directly.
- Each transition runs on a clone and emits one successor with unit outcome
  and no facts only when its final state differs from the original state.
  A stuttering transition emits no edge, so its filtered postconditions are not checked.
- State fields are public so fixture controls can construct their own starts;
  full and cut producers sit outside the public Model module and use `world()`.
- The `progress_dead_end` control is generic over that public machine interface,
  using `From<User>` and `From<Heater>` for actions while preserving its
  single 20-degree start, disabled heat step and scoped progress claim.
- The shared fixture runner replaces the sketch's hypothetical test APIs,
  verifying all claims including `boils` and rendering the `never_beeps`
  refutation of the sketch's `!beeping` predicate.
- Registration retains no syntax tokens and performs no lifting or subset
  analysis, following R36 rather than the sketch's lifting comment.

Scala's `transitions` block selects an action with `on` and guards its effect:

```scala
object transitions extends Transitions:
  on(user.pressButton):
    if !heatingOn then effects.pressButton()
  on(heater.failover):
    if heatingOn && temperature >= boilingTemperature then effects.failover()
  on(heater.heat):
    if heatingOn && temperature < boilingTemperature then effects.heat()
  on(user.depressButton):
    if heatingOn then effects.depressButton()
```

Rust groups the same actions into one match per actor, with guards and effects inline:

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

    #[on(Heater)]
    fn heater(&mut self, action: Heater) {
        match action {
            Heater::Heat if self.heating_on && self.temperature < BOILING_TEMPERATURE => {
                self.temperature += 1;
            }
            Heater::Failover if self.heating_on && self.temperature >= BOILING_TEMPERATURE => {
                self.heating_on = false;
                self.beeping = true;
            }
            _ => {}
        }
    }
}
```

`#[machine]` binds each `#[on(Actor)]` method to the named actor, checks its parameter type,
and dispatches `step` through it. The fallback leaves the state
unchanged and disables that action. Omitting it lets rustc check exhaustiveness. Actor declaration
order still determines catalog order, so grouping the arms changes no canonical payload.
The concrete unit-only actor enums also derive `Copy`, keeping owned action parameters idiomatic
under Clippy without changing `derive(Action)` or the lint policy.

Scala's `object effects` corresponds to optional `#[effect]` methods immediately after the
`#[on]` methods. These are ordinary shared helpers such as
`fn switch_off(&mut self, beep: bool)`; Kettle needs none. Only `#[on]` and `#[effect]` methods
may take `&mut self`. Every other `Model` method, including an unmarked helper, takes `&self`
or no receiver. Compiler-resolved read-only checks cover qualified, nested, imported and aliased
`Model` impls while leaving unrelated helper types usable. Under the approved checking exception,
compiler-only declarations may be added inside local function or const scopes to resolve aliases
and trait projections. Original authored statements and method/impl headers remain unchanged;
these declarations add no runtime guard. The generated module carries
`#[forbid(unsafe_code)]`.

Scala states the progress claim as a block with an `in` clause:

```scala
val boils =
  (whenever(heatingOn):
    eventually(beeping)
  ) in neverSwitchedOff
```

Rust writes it as a state progression, read "heating on, then any number of steps, then
beeping", with the assumption named in the expression:

```rust
#[progress]
fn boils(&self) -> LeadsTo {
    path![self.heating_on, .., self.beeping].assuming(Self::never_switched_off())
}
```

`path!` builds a `Path<S>`, a pattern of two or more waypoints (`bool`s, closures over the
state, or states) with a gap between each pair: `..` for any number of steps, `..=n` for at
most `n`. The pattern says nothing about whether it holds on every path or on some path; the
question asked with it does (see the reading rules in [README.md](README.md)). A claim takes
exactly two waypoints: `#[machine]` refuses a literal `path!` chain at compile time and
`Path::claim` panics on one built elsewhere when the claim is evaluated, which the checker
reports as that claim Refuted (pinned by the `progress_chain_path` checker fixture), because the universal reading of a chain is
ambiguous and no surveyed claim needed one. Adjacent waypoints without a gap are refused
rather than read as "next step"; `..=1` says that. The ceiling is concatenation and bounded
wildcards; alternation of sub-sequences, repetition and negation are not part of the pattern.
`#[machine]` wraps each progress body in `From::from`, so a body returns either a `Path` or a
built `LeadsTo`, and the lifter reads `path![p, ..=n, q]` as `whenever(p).eventually(q).within(n)`.

The `State: Hash` requirement already excludes standard interior-mutable state such as `Cell`
and `RefCell`. Thread-locals and atomics in statics fall under the capability Clippy policy,
rather than the receiver-signature check.

Invariant and postcondition methods are synchronous, return nothing and use `check!`
and `unknown!` statements, remaining ordinary methods when called directly.
A scoped per-evaluation recorder collects failed check expressions and holes without
panics or unwinding, and generated predicates turn the recorded result into a verdict.
Progress methods run once per real reached state; each claim's assumption and numeric bound must
stay fixed, with a mismatch reported as Unknown with a diagnostic. Exploration and queries validate
available registration metadata and explicit overrides even with zero starts, but invoke no
progress method without a real reached state. Valid zero-start claims are vacuously Verified.
The legacy `Scope` alias and trait builders remain for manual trait sources only;
the attribute surface exposes `Assumption`, `#[assumption]`, `LeadsTo` and `.assuming(..)`.
The core `Progress` builder preserves `.within(scope)` for manual trait Models; numeric bounds use
`.within_steps(n)` there.

Assumptions explicitly select `only`, exclude `without`, and declare weak fairness with `fair`.
Nothing is fair by default; `#[internal]` means environment-independent, not fair. Selection is
the last `only` list minus accumulated exclusions, regardless of chaining order. Fair entries
accumulate without admitting actions. Every list is validated, including excluded fair entries;
an explicit restriction admitting no catalog action is refused. A valid selection whose actions
are disabled in an obligation state produces a deadlock.

`LeadsTo::within(n)` counts transitions from an obligation state to the first target, excluding
the incoming transition that made the trigger true. The positive bound includes its endpoint:
a target at exactly `n` passes. Bounds do not benefit from fairness. Progress verdict precedence
is Deadlock, Cycle, MissedDeadline, Verified with an optional longest distance, then Unknown.
Reached holes and cuts prevent unproved success while concrete refutations survive. Safety and
caught predicate/effect defects have no fabricated temporal verdict. Default report spelling and
aggregate outcomes remain compatible.

`Result::verify_progress(&self, world, &registered_claim)` answers another question over the retained
graph through the same borrowed progress evaluator as exploration. It returns a fresh `ClaimResult`,
without re-exploration, retained-row cloning or cached-result mutation. A unique original registration
is selected by name; its canonical callbacks execute, with only the supplied explicit assumption
and bound overrides applied. Each replacement discards only that metadata and its consistency checks.
Unknown/ambiguous names and changed ordered action catalogs are refused before action-column indexing.
The caller must provide the same unchanged world used for exploration. Neither public descriptors
nor function-pointer equality establish world provenance or unchanged behavior.

Supplementary Kettle fixtures import the canonical attribute Model and use a generic delegating
adapter for room (20) and cold (-40) starts through its public state fields. They use the original
registered `boils` callbacks and the existing checker `report.txt`/`paths/*.txt` consumer. Canonical
sources and snapshots stay unchanged. The questions pin longest 81/141, the unrestricted
depress/press cycle, the press-only deadlock, bound-80 missed deadline and budget-10 Unknown.
Fn-213 still owns their ergonomic `tests.rs` presentation and illustrated `Model::boils` selection.

The TLC emitter translates each explicit `fair(A)` entry to `WF_vars(A)`; it emits no fairness
for internal actions or any undeclared action. The generic cycle hint suggests `fair` or fewer
admitted actions, but it is not proof that `fair(heat)` repairs Kettle: heat is disabled while the
element is off, so weak fairness permits its press/depress cycle.

The owner approved exactly five canonical payload substitutions on 2026-10-18:
`Act::UserPressButton` → `User::PressButton`, `Act::UserDepressButton` →
`User::DepressButton`, `Act::HeaterHeat` → `Heater::Heat`, and
`Act::HeaterFailover` → `Heater::Failover`, and state prefix `Kettle { ` →
`Model { `. This includes dependent `kettle_cut`
vacuity output and `progress_dead_end` rows and progress counterexample. Counts,
state values, catalog and state discovery order within each Model, claims,
verdicts and counterexample structure stay unchanged. Snapshot payloads and
metadata remain byte-identical under the fixture renames; symlink aliases were removed in fn-210.21.
The separate `sample_kettle_trait` fixture and
its native/WASM reference payloads remain byte-identical.

## Kettle construct comparison

These anchors compare the final [attribute source](umpire/fixtures/fixtures/sample_kettle/model.rs)
with frozen [Kettle.scala](archived/framework/samples/Kettle.scala), including its imports and
declarations. They compare authored constructs, not a claim of general front-end equivalence.

| Construct | Scala lines | Rust lines | Authoring tradeoff |
| --- | --- | --- | --- |
| Imports and container | 1–9 | 1–8 | Scala has one object; Rust has an attributed module and an explicit state struct. Ordinary methods remain visible to Rust tooling. |
| Constants | 11–13 | 10–11 | Both name the boiling and minimum temperatures; Rust requires explicit types. |
| State and starts | 23–26 | 13–23 | Scala's `var temperature = oneOf(...)` is shorter. Rust separates the field type, start attribute and default-false booleans; the named constant range is supported today. |
| Actors and actions | 15–21 | 25–42 | Scala's action vals are shorter; Rust enums make the catalog and actor types explicit. `#[internal]` marks the whole Heater enum. |
| Guards and effects | 29–55 | 44–72 | Scala separates four `on` entries from four effects. Rust's two actor matches put each guard next to its assignments, at the cost of `self.`, braces and fallback arms. Reusable `#[effect]` helpers are optional. |
| Invariant | 57–60 | 74–79 | Both use statement checks. Rust adds the method signature and kind attribute, and records the failed `check!` expression without unwinding. |
| Before-state postcondition | 62–65 | 81–86 | Rust makes `before: &Self` explicit; Scala supplies it implicitly. The comparison itself is unchanged. |
| Action-filtered postcondition | 66–69 | 88–92 | Scala nests `after(action)`; Rust places the filter on the method attribute and omits an unused before parameter. Both check boiling and beeping. |
| Environment assumption | 73–74 | 94–98 | Scala has a val; Rust has a named static method returning module-local `Assumption`. Both exclude the user's depress action. |
| Progress | 76–82 | 100–104 | Scala nests `eventually` inside a `whenever` block and attaches the assumption with `in`; Rust writes `path![self.heating_on, .., self.beeping].assuming(...) -> LeadsTo`, a list read as a state progression. Kettle's claim is unbounded; `..=n` would bound it. |
| Fixture entrypoint | Separate test file | `tests.rs` | Both keep tests beside the Model: Rust's golden hook and the negative `never_beeps` claim live in the fixture's `tests.rs`, outside the counts below. |

The final source has one invariant, two postconditions and one progress claim. No
`macro_rules!` transition, invariant, postcondition or progress wrapper earned a place: ordinary
matches and methods already express those constructs. `check!` and `unknown!` do earn their place
as small statement recorders for located failures and holes; `check!` is used by Kettle. The
`fixture!` helper is test registration, not a second Model grammar. The derives and `#[machine]`
supply catalogs and adapters, leaving the authored Rust statements and headers intact under
the lexical compiler-check exception described above.

## Source size and counting rules

Counts were read from the final source after fn-210.22 on 2026-10-19. Physical lines include
blank lines and comments; nonblank lines retain comments. Files are tracked `.rs` or `.scala`
sources. Rust runtime rows select each crate's `src/`; build scripts and tests are separate.
Dependency sources, target directories, generated protobuf Rust in `OUT_DIR`, manifests and
snapshot payloads are excluded. `awk` counts records, rather than language-aware statements.

| Rust workspace crate | `src/` files | Physical | Nonblank | Build script physical / nonblank |
| --- | ---: | ---: | ---: | ---: |
| `umpire-dsl` | 6 | 596 | 516 | 11 / 8 |
| `umpire-dsl-macros` | 4 | 910 | 866 | 11 / 8 |
| DSL plus macros | 10 | 1,506 | 1,382 | 22 / 16 |
| `umpire-checker` | 6 | 1,056 | 1,000 | 11 / 8 |
| `umpire-testing` (development support) | 4 | 398 | 373 | none |
| `umpire-runtime` | 2 | 220 | 200 | 6 / 5 |
| `umpire-fixtures` (fixture runner only) | 0 | 0 | 0 | 9 / 7 |

| Frozen Scala counterpart | Files | Physical | Nonblank |
| --- | ---: | ---: | ---: |
| `archived/framework/dsl` | 61 | 12,518 | 11,530 |
| `archived/framework/checker` | 21 | 4,901 | 4,599 |
| `archived/framework/samples` | 7 | 892 | 719 |

Scala excludes `*.test.scala`, `testdata/` and build-directive `project.scala` files. The samples
row also excludes the assertion-based `PublicQuestionsClient.scala` integration client
(141 physical, 121 nonblank); including it makes eight files and 1,033 / 840 lines. Scala's
macro implementations are included in DSL, hence the Rust DSL-plus-macros row. Scala implements
far more than Kettle: these are module sizes, not equal-feature implementation costs.

| Sample or support surface | Files | Physical | Nonblank | Nonblank excluding whole-line `//` comments |
| --- | ---: | ---: | ---: | ---: |
| Final `sample_kettle/model.rs` | 1 | 106 | 93 | 79 |
| Manual `sample_kettle_trait/kettle.rs` | 1 | 120 | 106 | 106 |
| Frozen `Kettle.scala` | 1 | 82 | 71 | 57 |
| All shared fixture Models (attribute Kettle and controls, after fn-223) | 12 | 534 | 458 | — |
| Checked-in formatted macro expansion snapshots | 3 | 561 | 561 | — |
| Go host handwritten source | 1 | 277 | 262 | — |
| Go host generated protobuf source | 1 | 625 | 538 | — |
| Go host tests | 1 | 628 | 610 | — |

The comment-filtered column is only an authoring proxy: it removes whole-line `//`, `///` and
`//!` comments, not inline comments or generated syntax. Rust's full 108-line count includes its
fixture entrypoint; removing that last line gives 107 physical, 93 nonblank and 79 comment-filtered
lines. No samples crate remains; shared fixtures include negative controls and are not seven
ported Scala samples. Test and fixture `.rs` source outside expected snapshots contributes
534 / 476 lines for DSL (9 files), 874 / 716 for macros (45), 1,206 / 1,100 for checker (29),
280 / 265 for testing (4), 102 / 93 for wasm (1), and 12 / 10 for the shared runner (1);
the shared Model sources above are additional to that runner. Macro expansion snapshots are
reported separately and do not inflate the runtime rows.

For example, from the repository root, `git ls-files 'model/umpire/checker/src/*.rs'` selects
the checker row and `xargs awk 'FNR == 1 {files++} {physical++} NF {nonblank++} END
{print files, physical, nonblank}'` counts it. Apply the same selector to each crate's `src/`
(DSL's macro crate has its own path). For the single-file authoring proxy use
`awk 'NF {n++} NF && !/^[[:space:]]*\/\// {c++} END {print NR, n, c}' <file>`.

## Parameterized worlds and inputs

Declared Worlds and inputs are delivered through the shared fixtures and retry-era activity
Model (fn-227.1/.2/.3/.7/.8/.11/.13/.14/.16), with actual Start admission and SDK retry proof
in .5. The [compiling README example](README.md#fixed-parameters-and-computed-starts) binds
its selected World explicitly and checks the exact documented source against a Rust fixture.

| Readiness | Delivered shape and boundary |
| --- | --- |
| Declared World | One module-local named-field `#[derive(World)]` struct is the `Machine`. `World::new` takes fields in declaration order, with the struct's visibility and no Copy/Default requirement. Absence retains the unit World and `world()`. |
| Typed callbacks | Handlers, claims, scopes and progress may take a trailing immutable `&World`. Its type is recognized before the old positional arity, preserving before-state/outcome forms regardless of parameter names. |
| Computed starts | One `#[starts]` associated method, first among registered methods in the main `impl Model`, returns complete states from an optional `&World`. Field `#[starts(range)]` remains; mixing forms or duplicate methods refuses. Complete-state duplicates deduplicate and empty starts stay empty. |
| Inputs | `derive(Action)` supports tuple/struct fields. `#[inputs(range)]`/`#[inputs([values])]` supplies explicit domains; otherwise `Finite` supplies bool, derived fieldless enums or a derived one-integer-field bounded newtype. Last field varies fastest. |
| World domains | Field-level `#[domain(path)]` names `fn(&World) -> impl Iterator<Item = FieldType>`. The actor exposes `actions_in(&world)`; explicit contextual patterns use that same World rather than a fabricated static catalog. Variant placement, duplicate domains and domain/inputs combinations refuse. |
| Generic declarations | `#[machine]` refuses generic Model, World and actor declarations with located errors (fn-227.13); standalone derives retain their generics. |
| Value handoff and emission | Hidden derived `Values`/`ActionValues` expose private scalar/enum field values, complete starts and concrete payloads without changing field visibility. The macro embeds one admitted front-end tree; the lifter binds selected values and emits it, including World CONSTANTs in every configuration and correlated Init states. Transitions are not inferred from native rows. |
| Suite ownership | `Space::new(world)`, `Space::with_world(world, starts)`, `Run::new` and `along_in` retain an explicitly owned World. Fixture `fixture!(World = World::new(..), budget)` supplies `bound()`/`bound_for(world)` for the same selected World, starts and checked catalog through native checks, emission and typed TLC replay. |

`Finite` is enumeration, not automatic validation. Its bounded scalar derive generates
`try_new(raw) -> Option<Self>` and `contains(raw)` from the one range; raw construction
remains possible. `World::new` assigns fields without generic admission or state validation.
Consumers admit their own policy: activity's live realization accepts maximum_attempts 1..=3.
Native states need Clone/Eq/Hash/Display, not a whole-state catalog. Persistent collections
remain admitted dependencies, without a Bank or Voting port. Native `imbl::Vector<T>` value
export preserves element order through `IntoValue` when `T: IntoValue + Clone`. This supports
binding and typed replay of admitted sequence state; the projection is absent on Wasm targets.

Generated input-bearing machines count domain products before materialization through
`checked_catalog(limit)`. The default ceiling is 4,096 concrete actions; exact excessive
counts yield `CatalogLimit`, arithmetic overflow `CatalogOverflow`, and Wasm uses `Budget`.
An empty factor empties its variant even beside an overflowing factor; its vacuity remains
visible. `Machine::actions` is unchecked; manual Machines use its length for default admission.
`Checker::explore_within` and `Space::catalog_limit` select another native ceiling. Binding
uses the checked default catalog. This is separate from state, path-alternative and walk bounds.
Supported source values/body forms remain a located subset; generic declarations, unsupported
field types, outcome expressions and methods keep their refusals. No blanket Rust admission is
claimed. Named local IR from .16 keeps conditional-return trees linear without a 10,000-node cap.
Numeric `.within(n)` on `LeadsTo` and manual `.within_steps(n)` can use World-derived bounds;
manual `.within(scope)` retains its assumption meaning. No time bound gains fairness implicitly.

### Selected-World native and TLC evidence

Counts below are complete reached sets for the actual selected fixture Worlds, not all possible
World values. Required tests check agreement and replay actual counterexamples through compiled
steps. [Retained source receipts](../.flow/tasks/fn-227-parameterized-models-declared-worlds.7.md),
[input receipts](../.flow/tasks/fn-227-parameterized-models-declared-worlds.8.md) and
[phase receipts](../.flow/tasks/fn-227-parameterized-models-declared-worlds.11.md) retain execution.

| Fixture and selected World | Native states | TLC states | Scope |
| --- | ---: | ---: | --- |
| `parameterized_world`: initial=1, limit=3 | 5 | 5 | Two distinct correlated starts; false never_three replayed. |
| `parameterized_world`: initial=0, limit=1 | 3 | 3 | Two starts; never_three Verified. |
| `parameterized_world`: initial=2, limit=1 | 0 | — | Native empty-start and binding control; no claim of executed TLC for this selection. |
| `action_inputs`: attempts=2, spares=0 | 6 | 6 | Four actions, empty Skip; current-payload fairness Verified, four counterexamples replayed. |
| `action_inputs`: attempts=3, spares=2 | 18 | 18 | Seven actions; current-payload progress Refuted, five counterexamples replayed. |
| `phased_rejection`: attempts=2 | 8 | 8 | Deliberate reopening/accepted-closed/wrong-status mutants refute laws; six counterexamples replayed. |
| Activity: maximum_attempts=1 | 6 | 6 | Closure laws Verified; no retry in the negative control. |
| Activity: maximum_attempts=3 | 18 | 18 | Closure laws Verified; retry-payload counterexample replayed. |

### Retry-era activity World and state

| Item | Current meaning |
| --- | --- |
| World | `maximum_attempts: u8`; supported live selections 1..=3, exploration fixture selects 3. |
| State | `phase` in Scheduled, Started, Completed, Failed, Canceled, TimedOut; `attempt: u8` starts at 1. One Scheduled start. |
| Delivery | `Worker::Poll(expected)` schedules Started only for the current attempt; a stale Poll on an open state rejects FailedPrecondition. |
| Failure | `RespondFailed { retryable }` from Started returns to Scheduled at attempt+1 iff retryable and below cap; otherwise Failed. |
| Closure | Completed/Failed/Canceled/TimedOut are terminal; `terminal_remains` preserves phase and `closed_rejects` requires unchanged state and NotFound on emitted terminal steps. Terminal Cancel supplies rejected self-loops. |
| Retained deadlines/cancel | ScheduleToStart and StartToClose directly enter TimedOut from Scheduled/Started; open Cancel atomically enters Canceled. No clock or cooperative cancellation. |

The default World reaches 18 states, emits 48 choices from 162 evaluations, refuses 114
disabled actions and reaches depth 6 without a cut. Coverage is `Cases(3 * 7) = 21`: seven
scheduled/started closing or retry alternatives per attempt, each with an exact witness.
[Compatibility](temporal/feature/activity/compatibility/README.md) retains all 15 immutable
artifacts pinned at `af0ebafc571d33651fdf2c008d4415ad72467a0d`. The explicit one-attempt
state/action/fact projection equals the original six-state/eleven-edge graph, including both
deadlines and four rejected terminal Cancel loops; a changed deadline fails that comparison.
This is the retry checkpoint, not old-byte equality. fn-237.3 owns final Tick-era compatibility.

### Live admission and retained verification

The Temporal realization transports optional actual `Observation.request` at tag 4 with its
same-invocation successful Start result. It binds namespace/activity/request/run identity and
immutable raw request bytes before publishing the World. Failed, crossed or missing evidence
cannot install it. An out-of-domain maximum_attempts yields field-named Unsupported and freezes
admission; first refusal remains sticky. An actual idempotent started=false response for the
retained Start request takes zero steps after progress or Reset. World replacement, first freeze and composed Reset publish
transactionally, including failure/oversize/allocation rollback. Annotate/Probe remain stateless.
Request timeout fields are recorded exactly, not admitted or quantized as World parameters;
present timeouts are not refused merely for being present. fn-237 owns their semantics,
Tick/heartbeat, deadline priority, cooperative cancellation and fair-worker assumptions.

The [actual .5 verification](../.flow/artifacts/fn-227-parameterized-models-declared-worlds/task5-final-verification.json)
and [retry race correction](../.flow/artifacts/fn-227-parameterized-models-declared-worlds/task5-review-correction.json)
retain the seven-scenario SDK proof: successful Start before worker startup, retryable failure
on attempt 1, actual failure-RPC acknowledgment, fresh Describe delayed beyond the retry
interval still Scheduled at 2 under a serialized actual-Poll permit, then released Poll 2
and Complete. This is request/server/SDK evidence, not a Testpilot Case or general conformance.
The current guest SHA-256 is `f832a33c3024a8752976d146be839b62a95afde4790fa0db60f1dac9004b4e7e`.
The .5 slice passed in 21.547 s; gaps/private-output controls passed at that guest in
361.572 s Go time (367.085 s outer). Its required model-time passed all 17 stages: 581.229 s
against the 60 s soft gate target (missed), inner 6.084 s against 10 s (met), outer 587.528 s.
These are previous shared-host observations, not isolated performance measurements.
[This task's final output and timings](../.flow/artifacts/fn-227-parameterized-models-declared-worlds/task6-final-verification.json)
record its own required model-time run and unchanged-production/guest reuse proof. Historical
fn-232 failure evidence and later fn-232/fn-239 certifications remain; the current guest is
not claimed freshly cross-OS certified. fn-238.5/.6 stay deferred absent an interpreted-checker need.

## Historical authoring options

The following complete seven-option comparison is retained from the removed `model/options/`
README. All seven files were noncompiling design sketches. Library versions, editor-support
and lifting cells are historical design estimates, not current dependency or tooling claims.

| File | Kind | One line | Macro effort | Editor support inside the Model | Lifting |
| --- | --- | --- | --- | --- | --- |
| `Kettle.grammar.rs` | custom A | one `model!` block with its own grammar; reads like the Scala | highest: a parser over tokens, own diagnostics | none inside the block | free, the macro has the tokens |
| `Kettle.macro_rules.rs` | custom B | plain items plus four `macro_rules!` helpers; bare var names by destructuring `self` | low: declarative macros, no proc-macro crate for the helpers | full: rust-analyzer expands `macro_rules!` | pass-through `#[machine]` attribute records the impl |
| `Kettle.attrs.rs` | custom C | plain functions with pass-through attributes (`#[on]`, `#[invariant]`, …), statig's mechanics | low-medium: attributes that only register | full | same |
| `Kettle.trait.rs` | custom D | a trait and plain Rust, no macros at all | none | full | `syn` pass over source in `build.rs` |
| `Kettle.statig.rs` | library E | statig 0.4 handlers as the surface, claims bolted on | none for authoring | full | would read statig's impl |
| `Kettle.smlang.rs` | library F | smlang 0.8 transition table | none for authoring | table only; guards and effects are functions elsewhere | table is already data |
| `Kettle.stateright.rs` | library G | stateright 0.31 `Model` trait as surface and engine | none | full | `syn` pass |

The C lifting cell's “same” referred to B's impl-token capture. The sketches claimed a common
expansion with `step(&mut self, Action) -> Outcome` and `_ => Refused`; R9/R10 superseded that
with a world implementing `Machine`, whose step returns a list and disables an action with an
empty list. R36 then removed token retention and lifting from this prototype altogether.

| Option | Disposition and complete reasons |
| --- | --- |
| A: grammar | Rejected. A custom parser over `syn` tokens creates a second compiler, with no completion/type checking inside the block and diagnostics implemented by the macro. Highest build/maintenance cost buys Scala-like bare names and immediate tokens for lifting. |
| B: declarative helpers | Rejected as the general surface because destructuring `*self` into bare fields breaks on collections (the recorded design reason, not a compiler experiment on this sketch). Individual helpers were allowed only if a construct clearly improved. Its low effort/editor claims remain historical estimates. |
| C: attributes | Selected, with no rejection reason. Ordinary callable functions preserve Rust diagnostics and tooling. The sketch accepted `self.` and boolean-returning claims in exchange; the final form restores statement `check!` claims, uses an inline module and one flat ordered impl, actor matches, named assumptions and `LeadsTo`. |
| D: manual trait | Selected engine foundation, rejected as preferred author surface because explicit enum/catalog lists and claim-name registration are too noisy. It is the no-DSL floor; a separate `syn` pass in `build.rs` was its proposed lifting route. The independent trait Kettle remains as the engine/parity reference. |
| E: statig | Rejected foundation, retained only as design reference. Named control states replace `heating_on`/`beeping`; claims must inspect generated State. The sketch supplies no native claims, enabled-action enumeration, start set or checker: exploration would call `handle` for every state/action and read State back. It substitutes handled/no-change for refusal and makes the original invariant disappear into “true by construction”, with an empty invariant list. |
| F: smlang | Rejected foundation, retained only as reference. Named control states displace variables; temperature becomes state data threaded through rows, while guards/effects move into separate context functions. No claims/checker or refusal value: missing rows return `process_event` errors. Generated enums permit enabled-action enumeration, but claims use generated KettleStates. Better suited to protocol-shaped machines than variable-shaped Models. |
| G: stateright | Rejected dependency/foundation, retained as checker design reference. State predicates lack `before`, `after(action)`, assumptions and fair progress; step claims need history in State or extensions. `sometimes("beeps", ..)` is only the closest built-in, not equivalent scoped fair progress. Its guarded `next_state` returning None was an explorer reference. The sketch also cited a single maintainer and July 2025 release/commit, advising reading/vendoring rather than depending on it; these are attributed historical claims, not newly checked facts. |

This preserves the comparison and each reason before deletion of all eight option-directory
files. Git history retains the sketches; the live authoring example is `sample_kettle`, and
the independent manual reference is `sample_kettle_trait`.

## Recommendation and next slices

Continue with **Rust**, following R24's stock-toolchain R20 verdict. Authoring size, diagnostics
and timings size the follow-up work; they do not introduce a second language-selection test.
Bank and Voting remain parked until the sample port is resumed; extend them only with their
needed parameter/domain gaps, rather than claiming Kettle establishes their readiness.

The owner-approved priority is fn-214, fn-216, fn-221, fn-213, fn-215, fn-212, fn-218, fn-217, then
fn-211. This is scheduling order, preserving each spec's prerequisites and approval boundaries.
The post-verdict Case plan must carry these requirements; none is implemented by fn-210:

| Requirement | Case-plan obligation |
| --- | --- |
| R33, later owner amendment | The owner superseded schema-copy guidance with `umpire.model.v1` and Reset/Judge/Answer over opaque Any, plus exactly three Temporal envelopes over the upstream API. Fn-216 delivers that protocol slice without copying/importing Testpilot's schema. The broader Program remains data the Model replays; remove Contract/correlated-clause/CEL verification and bind Case identity to the Model content hash, Model and protocol version in the later Case plan. |
| R34, tightened by R35/R37 | Target at most **5,000 generic** and **11,000 total non-test Go lines** in Testpilot, down from the historical approximately 20,000 / 29,000 inventory. Remove `internal/verification` (2,725) and `evaluation` (849); shrink `internal/ir` (4,105) to instruction decoding, `internal/execution` (6,575) to act-and-observe, `replay` (1,513) to Model search, and `campaign` (1,261) to the host loop. Retain needed recording/storage and most Temporal driver work (about 8,600 before callback simplification); report achieved sizes. Older 6,000/14,000 and 5,000/13,000 targets are superseded. |
| R35 | Remove all CEL, `cel`/`predicate`, the IR CEL bridge and `cel-go`. The Model decides read completion on each observation. Instructions contain interval/timeout/backoff mechanics; capture and redaction are field paths. A read timeout without Model acceptance is an incomplete read in the Run, judged by the Model. Execution checks still enforce RPC completion, evidence well-formedness and redaction. |
| R36 | Keep `#[machine]` registration-only. Lifted trees, subset diagnostics and TLC's second opinion belong to fn-212, which first judges whether their machinery is worthwhile. Superseded: fn-212 added the lifter, and since fn-238.4 `#[machine]` embeds the admitted Model. |
| R37 | At every SDK callback ask the Model what to return; serialize callbacks into the run's one Model instance. Remove worker interpreter/expression/typed-script behavior. Record a callback the Model cannot answer in the Run for Model judgment. |
| R38 | Replace the separate run/fuzz/replay/assess/repeat commands with one host loop and modes/drivers for a Model question, exploration, reduction, recorded Run or many Runs. Remove the NDJSON campaign/replay bridge. |
| R39 | A Run contains its observation log and Model judgment. Remove evaluation profiles, receipts and provenance documents. Keep `publish` as a never-overwrite store and keep only the canary's lease and policy. |
| R40 | Host exactly one Model per Run in the Go runner. Go callbacks stay in process; other SDK workers use the runner's local gRPC/HTTP endpoint with protobuf messages and a thin forwarding shim returning result/failure/heartbeat/delay. Workers never embed another Model. Record unreachable endpoints and callback timeouts in the Run for Model judgment; refuse unknown shim protocol versions with Refusal. |

## Mini activity slice (fn-216)

The independent six-phase activity Model uses the Reset/Judge/Answer protocol over opaque Any.
[Slice behavior, boundaries and measurements](README.md#mini-activity-slice-fn-216) retain the proof details.

## Declarative Realizations

The Temporal-owned `#[realization(activity)]` impl generates dispatch from `#[observe(WorkflowService::Method, reads(path), ignores(path = "reason"))]` and
`#[answer(Task, reads(path), ignores(path = "reason"), produces(Request, fields(path),
ignores(path = "reason")))]` handlers. The independent Model carries no system annotations.
Observation input is a response or `Result<Response, i32>` for raw gRPC status handling. Handlers return `Result<_, Refusal>`. `Projection<Action>` is used directly;
`Step<Actor> { action, expected }` proposes one converted action with its expected outcome.
A state-valued return requires `project(Self::mapping)`. Activity's mapping returns Current for equal phases, proposes only Completed or Failed, and leaves other changes unexplained.
Answer returns `Reply<Actor> { delivery, action, request }`; the raw packed request is checked then wrapped once. SDK-filled fields remain partial. `#[propose]` is reserved.

`feature/activity/expected/realization.snap.txt` names canonical RPCs/messages for each observed response, incoming callback and outgoing request.
Descriptor-ordered paths are `mapped`, `ignored` with a reason, or `unsupported`; outgoing mapped paths mean produced fields.
Classified parents cover subtrees; opaque parents stop traversal. The report lists Model-action RPCs missing observation handlers with their callback/request coverage,
then RPCs missing every realization context. Its action universe is the admitted `RPC_BINDINGS`.
The Temporal fixture consumer uses generated `Realization::REALIZATION`, the same classifications
as runtime guards. Shared catalog/descriptor admission rejects unknown, overlapping and opaque descendant paths and blank reasons.
Guards inspect decoded optional/oneof presence, including opaque fields. Proto3 scalar defaults have no decoded wire presence. Unused declared reads are not detected.
Generated-enum wildcard checks reject unresolved type coverage. The added-field control changes actual descriptor/catalog inputs, recompiles metadata
and requires the unchanged fixture golden to fail with the new unsupported field.

Complete-file physical counts for `feature/activity/temporal.rs`, including blanks/comments, are 110 before fn-231 (`b9f26782e39f815d740ea4b89925a723303c861d`) and 197 after the rewrite.
The 87-line increase retains explicit field reasons; this source is longer. The formatted expansion snapshot pins generated dispatch separately. Model behavior and existing goldens stay fixed.

## Annotation catalog

Authors classify the API fields a feature uses in its typed sidecar,
`temporal/feature/activity/api.rs`. Upstream protos remain unchanged. The catalog
supplies generation, validation, consequences and expected server errors; the independent
`model.rs` continues to own transitions. [Authoring syntax](temporal/annotate/README.md)
lives beside the `Catalog` derive.

### Layering and admission

`umpire/annotations` owns system-free catalogs, kinds, tags, presence, expectations, seeded `fill`, `validate` and gap classification.
Temporal owns derive mirrors, descriptors, declarations, transport and probes. The source-boundary gate rejects Temporal names in `umpire/`.
Each field has one kind (Uuid, Identifier, Timestamp, Duration, Counter, Flag, Number, Secret, Enum, Text or Bytes) and compatible tags. `View<K>` checks the prost field or typed projection.
Admission rejects contradictory roles (`Decided + Ignored`, `Secret + Observed`, `Random + Decided`),
incompatible tags, unexplained ignores and rules naming ignored fields. Ignored values retain their prost type.
Custom tags declare compatible kinds and a concrete consequence without hiding roles. Fallible setters refuse invalid bytes before mutation; runtime catalog admission also runs.
Constraints borrow protovalidate names (`string.uuid`, `duration.gt`, `enum.defined_only`, `repeated.unique`); presence borrows JSON Schema's required/any_of/one_of/requires/excludes;
Decided/Observed/Ignored borrow AIP vocabulary. Native sidecar checks need no CEL, external engine or upstream descriptor validation options.

### Four exhaustiveness levels and API updates

| Level | Build-time obligation |
| --- | --- |
| Fields | A struct pattern without `..` names every prost field; descriptor checking also names every source oneof alternative and requires nested coverage unless explicitly opaque. |
| Enum values | A reviewed `foundation/enum_inventory.rs` lists variants; descriptor checks and wildcard-free matches reject additions. Runtime unknown integers return `UnknownEnum`. |
| RPC methods | Generated service identities feed exhaustive `classify` matches: `Action`, `Observed` or `OutOfScope(reason)`; stale, duplicate and missing declarations fail. |
| Model actions | Exhaustive outer actor and inner action matches bind each action to an RPC, an observation or `Environmental(reason)`. RPC mappings must agree with service classifications. |

The current `proto/api.binpb` has no service descriptors. `temporal/api/build.rs` uses
`getproto --linked-api` to add the actual linked WorkflowService descriptors in
`OUT_DIR/annotation-api.binpb` for coverage checks, preserving upstream-owned files.
Rust API generation still reads the upstream image. The build tracks that image and feature
`api.rs` files recursively, including their directories and the shared enum inventory.

After an API update, `make model-check` requires classification of added fields, enums and RPCs. Review kinds, roles, expectations and reasons with aligned action mappings.
Regenerate affected Go envelopes with `bash model/scripts/proto-temporal-go.sh`. Activity changes also run `make umpire-check-model-slice` and `make umpire-check-model-gaps`; review findings before goldens.

### Validation and the first gap report

Request constraints/presence carry typed rejection, `Accepts` or `Unobservable` expectations.
Server rules use `#[rule]`, mapping `#[mirror]`. `validate` reads without mutation; seeded `fill`
preserves supplied values/roles and refuses unfillable fields. Activity fills only absent request
IDs. Request findings exclude Accepts; response findings retain violations and exact bytes.
Loose rejection modes redact messages and are explicitly counted.

The guest derives contextual probes, builds minimal negatives and validates decoded witnesses. Go supplies real prerequisites and transport, including tokens and retry-enabled resources.
Bounded recipes carry witnesses over 2 MiB within the 1 MiB individual/16 MiB live ABI limits. Each case sends a valid baseline. Results are Confirmed, Mismatch, Accepted or Unverifiable;
successful intentional Accepts are Confirmed. Mismatch/unexpected Accepted fail unless a reviewed
AcceptedGap has a calendar date and nonblank reason. Activity's registry is empty.

The original [golden](temporal/feature/activity/expected/gaps.txt) had 62 Confirmed (52 rejections, 10 acceptances) and zero Mismatch/Accepted/Unverifiable before descriptor mutations replaced
server-source mapping. Unknown Start id_reuse_policy/id_conflict_policy had incorrect Accepts;
the server rejects both. Corrected expectations confirmed them without Model/server changes.

Black-box reverse checks mutate every request field from raw protobuf descriptors, independent
of catalog constraints, including ignored fields. Typed Rust setters construct absent, empty,
limit-plus-one, negative and undefined-enum witnesses where applicable. Traversal is bounded;
cycles, depth limits and inapplicable mutations remain explicit. Unpredicted server rejections
are `Unannotated` failures, not validator-source coverage or automatically accepted gaps.

`Monotone`, `Immutable`, `Cached { ttl }`, `EventuallyConsistent { polls }`,
`IdempotencyKey` and the three roles proved system-free and already live in Umpire.
The macro's Temporal-side `Visibility` control demonstrates a custom tag mapping to the
generic poll-bound consequence; the activity catalog does not declare it. Promote only a
system-free consequence; Temporal scoping and names stay in `temporal/`.

## Accepted self-loops (fn-224)

An attribute handler is enabled only when it changes state, so an enabled action that leaves
the state unchanged (a heartbeat, an idempotent request answered OK) had no spelling except a
`reject!` self-loop. `stay!()` in a handler arm marks the step enabled even if unchanged. It
records a flag in the same scoped frame as `reject!` and does not return: an arm that also
changes state is an ordinary step, and a rejection in the same step wins. In a module with
`reject!` its outcome is `Accepted`, otherwise unit. Modules without `stay!` expand exactly as
before, so existing goldens and expansion snapshots are unchanged.

The name is a short plain verb in `reject!`'s imperative style that names the edge (the state
stays). `accept!()` was the obvious pair, but it would imply an outcome that unit modules lack
and read as permission for any step. Admission is lexical: `#[machine]` rewrites `stay!()` in
`#[on]` bodies to the macro's hidden handler arm, and the plain arm is a `compile_error!`, so an
`#[effect]`, claim, helper or use outside `#[machine]` refuses at compile time (trybuild pins
the effect, helper and outside forms). A helper is refused rather than followed as `reject!`'s
is, because a mark that silently depends on call depth is easier to misread than to forbid.

`umpire-lift` strips `stay!()` from an arm's top level and sets the branch's
`preserve_self_loop`, which drops the changed-state conjunct and emits the existing authored
self-loop observer step; weak fairness stays on authored fair actions only. A `stay!()` nested
in a condition is refused with a located warning. `accepted_self_loops` pins the `Accepted`
and precedence rows natively; its lift still refuses an unsupported outcome-match expression.
`reject!` and the stock closure laws are admitted by the separate parameterized controls.

## Rejecting outcomes and live cancellation (fn-221)

The first evidence-driven DSL expansion distinguishes an answered error from a disabled
action. In an ordinary unit-returning handler, `reject!(Rejection::NotFound)` records a
rejection and returns. A module containing `reject!` uses `Outcome::Accepted` for changed
states and `Outcome::Rejected(rejection)` for unchanged rejecting edges. An unchanged
handler without rejection remains disabled; a rejection that changes state is a Model
error. Modules without `reject!`, including Kettle, keep unit outcomes and their goldens.
Outcome-taking postconditions receive the emitted edge's outcome.

The hand-implemented `Phase` and `Phased` traits name terminal phases once.
`closable::terminal_remains` prevents leaving or changing a terminal phase, and
`closable::closed_rejects` requires unchanged state and the entity's declared rejection.
The activity delegates both laws with NotFound. This supplies a shared foundation without
a new attribute or capability registry. Request-ID deduplication, repeated cancellation
and late worker replies remain outside the six-phase Model.

The live slice retains Complete, Fail, HostRefusal and context Cancellation and adds
CancelAfterCompletion and CancelBeforeWorkerStart. After real completion and terminal
Describe, a fresh-request-ID cancel returns NotFound. The test transports the returned
service error's actual code/message using `serviceerror.ToStatus`; Judge explains exactly
one `Client::Cancel` with `Rejected(NotFound)`. Another genuine Describe remains Completed
with empty steps/outcomes. A cloned error changed to the recognized FailedPrecondition
code is Unexplained with exactly `Client::Cancel` expected; subsequent genuine error and
Describe observations retain that frozen result.

Before worker startup, a fresh-request-ID cancel returns the actual successful RPC response.
Judge explains one `Client::Cancel` with `Accepted`, then the real Canceled Describe and
its repeat match with empty steps/outcomes. Activity-body calls remain zero. Live Started
cancellation is excluded because the server passes through CancelRequested, omitted by
this atomic six-phase restriction. The Model and realization were unchanged for this proof.
The Go host continues to transport opaque Any values without Temporal imports.

The Observation result oneof keeps response tag 2 and adds error tag 3. It uses the local
two-field `umpire.realization.v1.Error` because `google.rpc.Status` is absent from the actual
upstream descriptor closure. `Explanation.outcomes` parallels `steps`; replay supplies the
final actual outcome to matching and supplies None for zero-step current checks.

The first successful expanded `make umpire-check-model-slice` run on 2026-10-19 took
339.680626 ms including cleanup, excluding compilation. Its Answer and Judge samples are
recorded below. These are individual calls on linux/arm64, Go 1.27.0, Rust 1.99.1,
SDK 1.48.0 and `CGO_ENABLED=0`, with the inherited Cargo target at
`/home/agent/.cache/codex-build/cargo-target`; they establish no scaling or percentile claim.
Raw command output is `.flow/tmp/fn2214/green-slice-status.log`. The initial adapter run
failed because `status.FromError` did not recognize the client's serviceerror; the corrected
run uses the existing serviceerror conversion and preserves the observed status.

| Live call | Latency (µs) |
| --- | ---: |
| Answer Complete | 62.750 |
| Answer Fail | 52.208 |
| Answer HostRefusal | 84.250 |
| Answer CancelAfterCompletion | 67.917 |
| Judge actual NotFound cancel after completion | 75.708 |
| Judge successful cancel before worker startup | 28.042 |
| Judge Canceled Describe after accepted cancel | 47.875 |

The R33-R40 and Case obligations above remain deferred. This proof covers these two RPC
scenarios through the existing SDK/server slice; it does not execute a Testpilot Case or
establish general Temporal conformance.

## Final measurements and editor evidence

[Retained measurements](design/rust-verdict-measurements.md) record the 2026-10-19
source checkpoint, native exploration, editor checks, guest/host costs and exact
reproduction commands. Current front-end timings remain above.

## Historical measurements

[Historical measurements](design/rust-verdict-measurements.md#historical-measurements)
retain the superseded fn-210 results.

## Shared catalog and envelope extraction (fn-233.1)

The framework owns catalog syntax, derive expansion, semantic views, recursive
inspection/fill, descriptor coverage and method/mutation generation. Context types,
API roots, service identities and request roots come from each caller. Temporal keeps
its catalog declarations, dynamic-config policy and upstream descriptor acquisition.
The independent Kettle catalog fixture consumes this implementation without a Temporal
dependency; canonical Model and semantic outputs are unchanged.

Realization envelopes now use `umpire.realization.v1`; generic annotation and probe
transport uses `umpire.annotations.v1`. Any identities deliberately migrate, while field
numbers and payload encodings remain fixed, including Observation.request at tag 4.
The stateless annotation dispatcher and Go annotation/probe transport are shared.
Temporal retains ProbeRequest's namespace/activity/queue/run/token controls and its
activity witness registry and recipes: these encode that API's prerequisites rather
than reusable transport. Its realization-handler source analysis and upstream graph
remain extension-owned because they resolve authored API types and service shorthand;
this extraction does not introduce another realization authoring surface. No required
generic catalog or transport implementation remains behind that boundary.

## Kettle second catalog consumer (fn-233.3)

Kettle's `reference/kettle/api.rs` mirrors the complete in-scope descriptor closure,
including the nested fill options, map entries and well-known duration. Descriptor
admission checks fields, enum variants and all service methods; generated Rust
patterns additionally check fields, oneof alternatives, enums and every canonical
Model action. The reference crate depends only on framework crates. The guest uses
`AnnotationRuntime`, and the healthy Go interceptor annotates both directions of
all six RPCs, fills an omitted idempotency key and retains the existing semantic
transcript. Response annotation preserves the actual wire bytes.

The second consumer exposed three reusable implementation gaps:

- Scalar oneof alternatives had been encoded and inspected as messages. Typed
  `OneofProjection` now retains string/bytes values and selected-default presence,
  while message alternatives keep transactional protobuf decoding. The generic
  scalar-carrier control and existing extension derive/projected-oneof controls
  exercise both paths.
- Maps assumed byte-valued metadata. String-valued entries now use a typed
  `TextEntry`, deterministic ordering and the same projection/inspection runtime;
  the byte-entry carrier and existing extension metadata controls stay unchanged.
- Resolved limit storage was still extension-owned. Shared `Limit<S>` and `Limits`
  now store declared defaults and resolved overrides. Each consumer owns its scope
  type and precedence policy: Kettle resolves compact-kettle tag overrides; the
  extension keeps its existing dynamic-configuration scope declarations.

Kettle-specific predicates use the derive's existing typed `via` projection:
water-kind/source correlation, nonblank taps, nonempty cartridges and nonempty
label keys/values. Nested message Rust types use the existing explicit `type`
parameter where protobuf nesting differs from module naming. These are consumer
policy and existing authoring controls, rather than new generic syntax.

The probe inventory comes from the shared descriptor wire traversal and mutation
setters. Each applicable case runs an independent healthy server baseline and
witness. Deliberate invalid bytes reach the server even when Annotate predicts a
rejection; the shared black-box classifier compares actual responses with those
predictions. Transport/unknown failures refuse classification. Unsupported,
unchanged and state-prerequisite mutations remain explicitly Unverifiable, and
the deprecated note's Ignored reason remains visible. `coverage.snap` records the
full catalog; `gaps.txt` is generated only from the actual Go server observations.
No Model or canonical behavior golden changes are needed.
