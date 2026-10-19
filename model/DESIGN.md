# Umpire in Rust: decision

The workspace now lives at `model/`, with the frozen Scala reference under `model/archived/`.
Run `make model-check` and `make model-time` from the repository root. Measurements below retain
the command names used when they were recorded, before the workspace move.

## TLC source backend (fn-212)

`umpire-lift` reads authored source with `syn`; the fixture build scripts publish emitted
modules, configurations and typed decoding metadata. It never calls the native explorer.
It reads one source form, the `#[machine]` attribute subset; source admission,
expression/type lowering, progress builders and emission have separate modules. fn-223
deleted the manual-trait lowering (`lift/src/manual*`): the lifter's source went from 5,048
lines to 3,022, and its review follow-ups to 2,978
(`cat model/umpire/lift/src/*.rs model/umpire/lift/src/*/*.rs | wc -l`). The follow-ups
removed what only the manual form produced: explicit start lists, tuple-state labels,
multi-file inputs, and per-branch choice indices and facts. The emitted observers keep
`__umpire_choice = 0` and `__umpire_facts = <<>>`, so modules and replay are unchanged;
choice and facts must be re-added with the attribute surface that states them. The tree keeps
`outcome` (`reject!`) and `preserve_self_loop`, which fn-224 sets for accepted self-loops.
Source that implements `Machine` by hand is refused at its `impl Machine` item
(`progress_dead_end`'s lift golden pins the message).
Unsupported syntax, references, conditional compilation and generated-name collisions
produce located warnings and refuse emission. `#[machine]` remains registration-only.

`umpire-tlc::Tlc::check_module` executes that module without requiring a native graph.
It retains the unrestricted complete reached set and separately checks every registered
safety, postcondition, scoped progress and supplemental negative property. Property-specific
observer scopes do not replace the unrestricted set. Native agreement is a separate testing
layer: it requires complete sets and every verdict to agree, and replays returned traces
through actual Model steps, including action, choice, outcome and facts. A native cut cannot
certify agreement. A control deliberately cuts native exploration at one state while the
same independently lifted source reaches its complete four-state set in TLC.

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

The shared fixture corpus admits 8 fixtures: attribute Kettle at 283 projected states,
`changed_state`, and six attribute progress graphs covering weak fairness (Kettle's `boils`
is the only scoped progress claim on TLC);
these are control variants, not 8 independent products. Four sources retain located refusals:
`kettle_cut`, `kettle_progress` and `progress_dead_end` use delegating adapters outside the
subset, and `rejecting_steps` uses a helper and rejecting outcomes outside the subset.
The system-owned fixture root additionally owns source-dependent artifacts. The newer
`Phased`/`reject!`/generic `closable` shape is outside this subset: it requires a located
refusal and has no system TLC agreement evidence. Older source evidence must not certify it.

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
| `edge_checks` | checker-only | `bool` outcomes and an authored self-loop (`Idle` at 2); unchanged handlers are disabled. `changed_state` keeps an `after`-filtered refuted postcondition with a replayed counterexample on TLC; the unfiltered refuted `never_decreases` and the `after`-filtered self-loop refutation are pinned nowhere on TLC. |
| `vacuity` | checker-only | Same `bool` outcomes and self-loop as `edge_checks`. Never-triggered `after` claims stay on TLC through `changed_state`. |
| `shared_starts` | checker-only | Duplicate starts `[0, 1, 0]`, `bool` outcomes and a self-loop; `#[starts]` enumerates a duplicate-free field range. TLC loses the duplicate-start control (TLA+ `Init` is a set). |
| `deep_chain` | checker-only | State carries a shared `Rc<Cell>` clone counter and the hook asserts linear clone work; a checker property, already refused by the lifter. |
| `interleaved_claims` | checker-only | Claim order interleaves kinds (`claim_order`) and the hook adds a supplemental manual claim; attribute claims are contiguous per kind. Already refused by the lifter. |
| `progress_graph` (cycle) | attribute | |
| `progress_continuous` | attribute | |
| `progress_intermittent` | attribute | |
| `progress_two_lobes` | attribute | Catalog order is `Enter, Escape, Left, Right` (was `Enter, Left, Right, Escape`); `Enter`/`Escape` never fire, so every golden is unchanged. |
| `progress_fulfilled` | attribute | |
| `progress_trigger_stops` | attribute | |
| `progress_self_loop` | checker-only | Authored self-loop. Holds the trimmed trait `Graph` the other moved progress fixtures use. |
| `progress_multiple_starts` | checker-only | Starts `[9, 0]` outside a field range, and a self-loop. |
| `progress_target_barrier` | checker-only | Self-loop after the target. |
| `progress_excluded_prefix` | checker-only | Self-loop inside the scope. |
| `progress_scope_exit` | checker-only | Self-loop beside an excluded fair escape. |
| `progress_action_choices` | checker-only | One fair action with two choices, one a self-loop. |

The eleven `progress_*` fixtures selected a mode of the trait `progress_graph` Model, so they
were trait-form too. TLC no longer pins self-loops, nondeterministic choice, non-`()` outcomes,
facts, starts outside a field range, an unfiltered refuted postcondition (`never_decreases`),
refuted progress under a `without` scope (`progress_excluded_prefix`) or a fair action that the
scope excludes (`progress_scope_exit`) anywhere in the shared corpus.

Real TLC execution is selected by `UMPIRE_BACKENDS=require`; the default gate checks source
artifacts/refusals and explicitly reports no TLC execution evidence. Required shared runs
mandate attribute Kettle and every admitted fixture's selected TLC goldens. Both renders in
the existing determinism contract execute the backend. The final required full gate took
144 s, including 101.60 s for the shared fixture double render. The final warm default gate
took 9.564 s and its edit/test inner loop 4.986 s (pre-edit baseline: 11.086 s / 4.295 s).
These single cached measurements distinguish the opt-in backend cost from the default
feedback loop; they are not a scaling benchmark or a claim that TLC is faster than native.

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

## Capability and panic boundaries

The real guest imports only `environ_get`, `environ_sizes_get`, `fd_write` and
`proc_exit` from `wasi_snapshot_preview1`, with exact checked signatures. The host
provides an empty environment, discarded descriptor 1/2 output and an exit trap.
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

WASI uses panic=abort. Unexpected guest panics trap and invalidate the host
instance; host errors name the logical method and failing ABI function. Native
checker catch_unwind behavior and panic/overflow fixtures remain unchanged.
Invalid budgets and state IDs remain generated typed refusals, not panic tests.

## Attribute Kettle and the authoring sketch (final flat form)

The canonical `umpire/tests/fixtures/sample_kettle/model.rs` uses the owner-approved
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

These anchors compare the final [attribute source](umpire/tests/fixtures/sample_kettle/model.rs)
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
| `umpire-wasm` | 2 | 220 | 200 | 6 / 5 |
| `umpire-tests` (fixture runner only) | 0 | 0 | 0 | 9 / 7 |

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

## Parameterized Models: readiness

The engine is ready by construction under R9: a world is a value implementing `Machine`, and
its parameters can be ordinary fields. `starts`, `actions` and `step` receive `&self`; state,
edge and progress predicates receive `&M` (`dsl/src/machine.rs`, `claim.rs`, `progress.rs`).
The associated `State: Clone + Eq + Hash + Display` admits generic state values satisfying those
bounds, without a whole-state enumeration. Manual worlds can therefore compute starts and
action catalogs from their parameters. The missing R17 engine-side convenience is `Finite`
and its derive/domain support; persistent collections are admitted dependencies, not a ported
Bank or Voting implementation.

This is a trait-contract conclusion, with an explicit inference caveat: Rust does not infer
the world type backwards through every associated-state projection. Manual registrations may
need `Claim::<Self>` / `StepClaim::<Self>` / `Progress::<Self>` or an explicitly typed closure parameter. That is
type annotation at the call site, not evidence that the attribute surface supports parameters.
Kettle itself proves only its unit-world, nongeneric, unit-action use case.

| Attribute gap in the final code | Settled future shape, delivered with the first sample needing it |
| --- | --- |
| No declared world struct: `#[machine]` emits `struct World;` and `world()` returns that unit value. | A module-local `#[derive(World)]` struct carries parameters; absence keeps the unit world and Kettle unchanged. |
| Generated handler and claim calls discard the world argument. | An optional explicit extra `w: &World<..>` parameter on handlers and claims, detected by arity as `before` is today. |
| `#[starts(range)]` supports static ranges, including `COLDEST_TEMPERATURE`, but not world-dependent/computed starts. | A `#[starts]` impl block supplies computed starts; keep field `#[starts(range)]` for literal/static domains. |
| `derive(Action)` requires unit variants; no input fields or domains. | Action variants carry inputs; domains come from `Finite` or the world. |
| `#[machine]` rejects generic registered impls and its generated adapters name bare nongeneric types. | Accept generics on `Model`, `World` and action enums and propagate their bounds. Standalone State derive's existing generic support does not close this gap. |

Numeric `.within(n)` is available on `LeadsTo`; the manual trait's `.within(scope)` retains its
environment-assumption meaning and uses `.within_steps(n)` for numeric bounds.
Once world access lands, `w.resource_managers.len()` needs no
additional authoring form to supply a runtime parameter-derived bound. This follows from the
planned explicit Rust parameter and expression argument; no such bound is evaluated by today's
Kettle or generated code.

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
| R36 | Keep `#[machine]` registration-only. Lifted trees, subset diagnostics and TLC's second opinion belong to fn-212, which first judges whether their machinery is worthwhile. |
| R37 | At every SDK callback ask the Model what to return; serialize callbacks into the run's one Model instance. Remove worker interpreter/expression/typed-script behavior. Record a callback the Model cannot answer in the Run for Model judgment. |
| R38 | Replace the separate run/fuzz/replay/assess/repeat commands with one host loop and modes/drivers for a Model question, exploration, reduction, recorded Run or many Runs. Remove the NDJSON campaign/replay bridge. |
| R39 | A Run contains its observation log and Model judgment. Remove evaluation profiles, receipts and provenance documents. Keep `publish` as a never-overwrite store and keep only the canary's lease and policy. |
| R40 | Host exactly one Model per Run in the Go runner. Go callbacks stay in process; other SDK workers use the runner's local gRPC/HTTP endpoint with protobuf messages and a thin forwarding shim returning result/failure/heartbeat/delay. Workers never embed another Model. Record unreachable endpoints and callback timeouts in the Run for Model judgment; refuse unknown shim protocol versions with Refusal. |

## Mini activity slice (fn-216)

The independent `temporal/features/activity/model.rs` fixes a six-phase restriction
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
For public states, `#[machine]` exports `World` and `Action`, and `world()` returns
`World`. Non-public states retain the opaque interface with generated types following
the state's visibility. The public activity world removes
the binding's duplicate action carrier, delegating machine and claim-index re-wrap.
Generic Any helpers derive canonical type URLs from generated `prost::Name`; only the
Describe and cancel RPC method paths are binding constants. The policy parser and reply constructor
keep the remaining binding specific to Temporal.

Exactly three envelopes live under `temporal/proto`: Observation carries a method and
a response Any or error, Callback a task Any, Reply a request Any. The real upstream API descriptor
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

Remaining DSL gaps include parameterized inputs/facts and typed projection authoring.
Fn-221 closes the rejecting-outcome gap described below. No Model transition
was changed to fit server evidence. Fn-213 must include the Temporal fixture root;
fn-218 retains Cases(8), resolving impossible alternatives as a downstream grammar
decision rather than inventing edges or silently shrinking coverage.

For R33 this proves the Model protocol and opaque payload seam, preserving old wire
bytes and refusal codes. Broader Case identity/versioning, Program replay and removal
of Contract/CEL remain open. R34's size reduction, R35's CEL removal, R36's deferred
lifting decision, R37's general callback replacement, R38's unified host loop, R39's
Run/storage redesign and R40's polyglot forwarding remain the obligations in the table
above; this slice does not discharge them.

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
two-field `umpire.temporal.v1.Error` because `google.rpc.Status` is absent from the actual
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

Measured 2026-10-19 at source checkpoint `7d1b9a5e93aa87b822d2227fbcd6b053a61e2f39`,
after the flat surface and sample renames. Subsequent changes in this task only record these
observations. The canonical sample SHA-256 is
`76bf39abd4a108f7d7b72054b731b6e9506cfa65c803487bcb2d8dc41788806d`.
No concurrent latency measurements ran. Host: Linux `dev-build 7.0.14`, aarch64, 12 CPUs;
Rust/rust-analyzer 1.99.1 (`58cf8f9`, 2026-10-13), Cargo 1.99.1, Go 1.27.0, wazero 1.12.0.
Raw CPU/OS data, exact argv/cwd/environment, artifact paths, hashes and every subprocess exit
are retained under `.flow/tmp/fn2107-measure-final/`; native profile metadata and raw samples
are in `fn2107-measure-final-release/` and `fn2107-measure-final-dev/` beside it.

### Native fresh exploration and all claims

Each of five fresh processes makes three calls, without warmup. Every call constructs a fresh
exploration at budget 1,048,576, then calls `verify_all`; it never reuses an explored Result.
World construction is outside the in-process timer; starts, claims, exploration and report
cloning are inside. Formatting, assertion, printing and destruction are outside. Every call
reports 81 starts, 283 states, 424 emitted edges, 1,132 evaluated actions, 708 refusals, depth
62, no cut, all four claims Verified and no vacuity. The table reports every sample in order.

| Profile / process | First call (µs) | Second call (µs) | Third call (µs) | Whole process wall (ms) |
| --- | ---: | ---: | ---: | ---: |
| release / 0 | 102.917 | 69.000 | 66.333 | 15.206625 |
| release / 1 | 200.084 | 74.250 | 73.333 | 23.011500 |
| release / 2 | 103.375 | 72.750 | 67.375 | 23.628125 |
| release / 3 | 133.750 | 70.750 | 67.083 | 7.820500 |
| release / 4 | 101.583 | 1,249.042 | 658.417 | 29.888334 |
| dev / 0 | 1,048.083 | 988.250 | 991.667 | 4.443792 |
| dev / 1 | 1,209.875 | 1,099.583 | 1,125.959 | 4.324500 |
| dev / 2 | 1,438.792 | 1,015.792 | 984.959 | 4.423708 |
| dev / 3 | 1,024.959 | 979.042 | 1,017.125 | 7.115750 |
| dev / 4 | 1,091.750 | 989.292 | 988.083 | 3.880875 |

Process wall includes spawn/startup, all three calls, output capture and shutdown, excluding
builds. It is not a startup-only estimate. The release outliers remain in the table; there was
no retry or minimum selection. The standalone driver includes the actual fixture and links
the exact rlibs reported by that invocation's Cargo JSON, rather than selecting a cache glob.
Both profiles use overflow checks and unwind. Release uses opt-level 3, debug 1, thin LTO,
one codegen unit and no debug assertions; dev uses opt-level 0, debug 2, no LTO, 256 codegen
units and debug assertions. These explicit driver flags match the workspace profiles;
environment rustflags were absent. Driver compilation is outside both timing boundaries.

Reproduction from `temporal/`, with the shared target idle (output directories must be new):

```sh
CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target python3 .flow/tmp/fn2107-measure-native.py --execute --out .flow/tmp/fn2107-measure-final-release
CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target python3 .flow/tmp/fn2107-measure-native.py --execute --profile dev --out .flow/tmp/fn2107-measure-final-dev
```

The frozen Scala note measured the first public `verifyAll` call as **0.365004916 s**, then
0.109971000 and 0.111102584 s in the same JVM, before instance-retention fixes. The first
call already includes lazy checker initialization and JIT effects. It excludes JVM startup,
build, Model construction, initial-state materialization and rendering; later calls reuse
the Model and caches. Rust times fresh exploration each call and has a smaller feature scope.
These are historical cross-implementation observations, not a controlled speed ratio.
Standalone JVM startup, Scala clean/incremental builds and controlled post-fix Scala timings
are **unavailable**. The archived source was neither rebuilt nor repaired.

### Test-binary and build wall time

Cargo JSON from `cargo test -p umpire-tests --release --no-run --locked --offline
--message-format=json` identifies the actual `fixtures` executable. Its `--list` contains
`fixtures: test`; each `--exact fixtures --nocapture` run reports one passed test, exercising
all 23 shared Models and deterministic rendering. Five whole-binary walls were
**117.228501, 66.000416, 67.033625, 65.912000 and 65.446000 ms**. This includes fixture I/O,
rendering and comparisons and is not a Kettle-only checker timing. Actual executable paths,
list output and all five logs are in `fn2107-measure-final/test-*`.

The build command was `mise exec -- cargo build -p umpire-testing --lib --release --locked
--offline`, covering the DSL, macros, checker and testing library. A new empty target from
`mktemp` was used; downloaded registry sources were already cached. No shared target was
cleaned. Whole-command walls, including mise/Cargo startup, were:

| Build boundary | Wall (s) |
| --- | ---: |
| Clean artifacts, empty temporary target | 5.612547878 |
| Same command, no source change | 0.119822000 |
| Checker source mtime touched, same bytes, rebuild dependents | 1.356116667 |

The third row models edit-triggered dependency recompilation without changing behavior;
it does not measure a particular code edit or Rust's incremental compiler cache (release
incremental compilation is off). `clean-target.txt` records the retained temporary target;
`clean-build`, `nochange-build` and `edit-rebuild` logs show the actual scope and commands.

### Parameterized-world execution proof

The final manual-trait probe compiled against the exact final dev-profile Cargo rlibs and
matched all three literal expectations in `fn210-parameter-readiness.expected`:

```text
u8 start=0 limit=1 actions=2 states=2 edges=1 verified=true
u16 start=2 limit=4 actions=3 states=3 edges=3 verified=true
u16 start=5 limit=4 refuted=true
```

Its generic `World<T>` carries starts, a limit and an input domain; `Input<T>::Set` carries
values. Invariant, postcondition and progress predicates read the world. This proves the
manual engine contracts above on two type instantiations and the invalid-start control;
it implements none of the five deferred attribute gaps. Source, expected output, exact
compiler argv and output are retained as `fn210-parameter-readiness.rs`, `.expected` and
`fn2107-measure-final/parameter-*` under `.flow/tmp`.

### Macro-enabled rust-analyzer

The actual LSP client used the `model/Cargo.toml` workspace, idle shared Cargo target,
locked/offline build scripts, enabled proc macros and attribute expansion, and native
diagnostics with experimental diagnostics enabled. Cargo check/check-on-save were disabled,
so the unsaved diagnostic below is native analyzer evidence. Workspace readiness was
5.725970 s after opening (6.169540 s from launch), using warm build artifacts. Expansion
proved generated `World`, `world()`, `Machine for World` and `type State = Model`.

Definitions returned the authored file, field at zero-based (19,12) and helper at (95,11),
from calls at (102,26) and (102,78). Hovers showed `heating_on: bool` and
`fn never_switched_off() -> Assumption`. Individual request latencies in milliseconds:

| Request | First | Warm 1 | Warm 2 | Warm 3 |
| --- | ---: | ---: | ---: | ---: |
| Field definition | 8.216875 | 0.333500 | 0.251250 | 0.251416 |
| Field hover | 11.787375 | 0.362250 | 0.223792 | 0.246667 |
| Helper definition | 0.463125 | 0.317041 | 0.268959 | 0.212791 |
| Helper hover | 0.656000 | 0.477459 | 0.301791 | 0.232167 |

An unsaved version-2 overlay changed only that field reference to `heating_oon`. Explicit
`textDocument/diagnostic` returned native rust-analyzer E0559, “no field `heating_oon` on
type `&Model`”, at the edited range. Observation took **0.581092 s** including the deliberate
overlay hover/expansion checks before the pull. Restoring the exact version-3 text returned
empty diagnostic items in **0.051902 s**. Source hash remained identical; shutdown returned
0 with both reader threads stopped. These results establish the tested fields/helper,
not universal diagnostics for arbitrary macros or generic blanket traits.

The initial push-only client timed out waiting ten seconds for the typo, with both default
and experimental diagnostics. A third diagnostic investigation confirmed updated expansion
and a null typo hover but still no push. Those failed observations remain in
`fn2107-ra-final-result.json`, `fn2107-ra-experimental-result.json` and
`fn2107-ra-overlay-result.json`. The installed server advertises `diagnosticProvider`;
the corrected pull-protocol run is `fn2107-ra-pull-result.json`, including every raw
request/response, expansion and diagnostic. This was a client correction, with no production
macro change. The analyzer was stopped before benchmark and gate timing runs.

### Final stock guest and Go host

`cargo build -p umpire-wasm --target wasm32-wasip1 --release --locked --offline
--message-format=json` selected the actual guest artifact
`/home/agent/.cache/codex-build/cargo-target/wasm32-wasip1/release/umpire_wasm.wasm`.
It is **3,423,000 bytes**, including release debug information, SHA-256
`e6f119c9929c735a58012b2ee24370a970cf5d77b005295511bf1712893cbd12`.
The CGO-disabled Go host's uncached `go test -tags test_dep -count=1 ./...` passed,
retaining byte-for-byte native trait Kettle report/ordered-Rows parity and ABI/import controls.
That stock-toolchain parity is R24's recommendation criterion.

The actual benchmark commands, from `model/umpire/hosts/go`, set `UMPIRE_WASM` to that
artifact and `CGO_ENABLED=0`:

```sh
mise exec -- go test -tags test_dep -run '^$' -bench '^BenchmarkColdLoad$' -benchmem -benchtime=5x -count=5
mise exec -- go test -tags test_dep -run '^$' -bench '^BenchmarkRows$' -benchmem -benchtime=100000x -count=5
mise exec -- go test -tags test_dep -run '^TestRepeatedAndConcurrentCalls$' -count=5 -v
```

| Sample | Cold compile/instantiate/close (ns/op) | Go B/op | Go allocs/op | Rows (ns/op) | Go B/op | Go allocs/op |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| 1 | 67,639,233 | 21,625,740 | 32,991 | 15,792 | 1,442 | 28 |
| 2 | 83,999,283 | 21,624,760 | 32,990 | 17,130 | 1,443 | 28 |
| 3 | 130,485,142 | 21,624,689 | 32,989 | 17,557 | 1,438 | 28 |
| 4 | 120,552,000 | 21,624,840 | 32,991 | 15,672 | 1,437 | 28 |
| 5 | 91,333,683 | 21,624,744 | 32,990 | 14,685 | 1,437 | 28 |

ColdLoad excludes reading the bytes, with five operations per row. Rows explores once
before timing, then cycles the 283 state IDs over 100,000 calls per row, including protobuf
encoding/decoding, allocation, transfer, guest call, copy and release. It is not a full checker
benchmark. Every Rows sample began and ended at **18 pages**, or **1,179,648 bytes** of guest
linear memory (65,536 bytes/page), with sink-bytes 17,705,239. All five repeated/concurrent
controls passed their stable-pages-after-warmup assertion. Guest pages are not RSS or host
memory; B/op and allocs/op are Go allocations, not guest allocations. Every row is retained,
without selecting the fastest result. Raw logs are `fn2107-measure-final/go-*`.

### Final feedback loop and verification

Exactly one warm `CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target mise exec --
make model-time` ran after the source checkpoint, exiting 0 without retry. The complete
output is `.flow/tmp/fn2107-model-time.log`; its actual step measurements are:

```text
step (warm)                         time
gate command: UMPIRE_TIME_STEPS=1 make -C .. model-check
timing-controls                   0.022s
capability-controls               0.144s
protobuf-freshness                0.099s
fmt                               0.158s
clippy                            0.136s
docs                              0.119s
tests                             1.198s
wasm32-wasip1                     0.080s
core-wasm-clippy                  0.090s
wasm-guest                        0.108s
go-host                           1.875s
inner source: umpire/checker/src/lib.rs
inner command: mise exec -- cargo test -p umpire-checker --quiet
inner loop (edit+test)            6.569s
gate total                        4.279s  (soft target 30s)
inner loop                        6.569s  (soft target 10s)
```

Both observations meet the current soft targets. This includes the real complete gate and
a separate checker source-mtime touch/rebuild/test loop. Earlier per-task timings remain
historical; fn-210.22's 3.038 s inner / 3.629 s gate observation was retained without retry
and also meets the revised targets.

The pre-edit release workspace baseline passed. Final `make model-check` and
`cargo test --workspace --release --message-format=json` both exited 0, covering the real
fixture/refusal suites, capability controls, protobuf freshness, formatting, all-target
Clippy, docs, stock WASM builds and uncached Go-host parity. The four fixture-owning crates'
active build-script outputs all point to this root checkout, including Cargo's separate
host/target release variants. The shared registry selects exactly 23 Models and the final
sample/control names. Logs are `fn2107-baseline-release.log`, `fn2107-model-check.log`,
`fn2107-final-release.jsonl`, `.stderr` and `fn2107-final-registry-proof.log` under `.flow/tmp`.

## Historical measurements

The direct-export guest built for fn-210.16 on 2026-10-18 is 3,370,765 bytes,
including debug information, using the same release profile and toolchain below.

The following host latency and memory measurements were taken for fn-210.9 on
2026-10-18, before the direct-export rename in fn-210.16. They do not measure the
current binary; the final measurements above supersede them for the current source.

Measured on Linux arm64 with stock Rust 1.99.1, Go 1.27.0, protoc 29.5,
prost/prost-build 0.14.4, protoc-gen-go v1.36.6 and protobuf Go v1.36.11.
The release profile uses overflow checks, debug=1, thin LTO and one codegen unit;
that earlier wasm file was 2,872,544 bytes including debug information.

Five cold compile/instantiate/close runs, excluding file reading, took 29.217,
28.059, 27.874, 26.774 and 26.766 ms (median 27.874 ms).
Five runs of exactly 100,000 real Rows calls took 14,003, 13,838, 13,686, 24,061
and 24,269 ns/op (median 14,003 ns/op; range 13,686–24,269). These are aggregate
per-operation means, not latency percentiles. Each call includes protobuf
marshal, alloc/write/call/read-copy/free and protobuf decode, cycles state IDs
modulo 283, and consumes row text. Each run consumed 17,705,239 bytes; guest
memory stayed at 17 pages before and after every run. Go allocation counts were
28 allocs/op and 1,437–1,447 B/op. Runs include ordinary host scheduling noise.

Reproduce from `umpire/hosts/go` inside `model/`, with `UMPIRE_WASM` naming the built guest:

```sh
CGO_ENABLED=0 mise exec -- go test -run '^$' -bench '^BenchmarkColdLoad$' -benchtime=1x -count=5
CGO_ENABLED=0 mise exec -- go test -run '^$' -bench '^BenchmarkRows$' -benchtime=100000x -count=5 -benchmem
```

In fn-210.9, the integrated `make model2-time` passed with a 5.069 s warm full gate (30 s
budget) and 1.210 s checker edit/rebuild/test loop (3 s budget). This includes
capability/export controls, generation freshness, native checks, wasm builds
and uncached CGO-disabled Go parity tests. Raw measurements and final gate
output are retained in the fn-210.9 task evidence under `.flow/tmp/fn210-9-*`.
