# Umpire in Rust: decision

The workspace now lives at `model/`, with the frozen Scala reference under `model/archived/`.
Run `make model-check` and `make model-time` from the repository root. Measurements below retain
the command names used when they were recorded, before the workspace move.

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
generated messages from `umpire/proto/umpire/oracle/v1/oracle.proto`; Refusal is a
case of the common Response envelope, separate from a checker verdict or trap.
The guest includes `umpire/tests/fixtures/sample_kettle_trait/model.rs` directly.

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
- `State` supplies clone, equality, hashing, canonical `Spell` and ordered
  starts with defaults for unannotated booleans; `Action` supplies clone,
  equality, hashing, spelling and ordered enumeration.
- `#[postcondition(after = Heater::Failover)]` selects the failover action;
  its unused `before` parameter is omitted while the warming claim keeps it.
- The named assumption is a static function returning a module-local `Assumption`
  value through `without([User::DepressButton])`, replacing the associated
  constant and hiding action adaptation.
- `whenever(self.heating_on).eventually(self.beeping).assuming(Self::never_switched_off())`
  builds a module-local `LeadsTo` value with boolean trigger and target and
  names its assumption in the expression. This builder remains; `path!` is deferred to fn-215.
- Private generated world/action adapters implement the existing `Machine`
  protocol; the public `world()` function returns an opaque machine exposing
  its state and actor conversions without exposing generated type names.
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

The `State: Hash` requirement already excludes standard interior-mutable state such as `Cell`
and `RefCell`. Thread-locals and atomics in statics fall under the capability Clippy policy,
rather than the receiver-signature check.

Invariant and postcondition methods are synchronous, return nothing and use `check!`
and `unknown!` statements, remaining ordinary methods when called directly.
A scoped per-evaluation recorder collects failed check expressions and holes without
panics or unwinding, and generated predicates turn the recorded result into a verdict.
Progress methods run once per real reached state; each claim's assumption must stay
fixed within one exploration, with a mismatch reported as Unknown with a diagnostic.
An empty world evaluates no progress method and is vacuously Verified; manual fixed
assumptions are still validated before exploration.
The legacy `Scope` alias and trait builders remain for manual trait sources only;
the attribute surface exposes `Assumption`, `#[assumption]`, `LeadsTo` and `.assuming(..)`.
The core `Progress` builder remains unchanged for manual trait Models.

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
| Progress | 76–82 | 100–104 | Rust uses `whenever(bool).eventually(bool).assuming(...) -> LeadsTo`; Scala uses its block syntax and `in`. Neither the future `path!` form nor numeric bounds are implemented here. |
| Fixture entrypoint | Separate test file | 108 | Rust includes a fixture registration and the negative `never_beeps` claim in this file; it is included in the full-file counts below. |

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
| Final `sample_kettle/model.rs` | 1 | 108 | 94 | 80 |
| Manual `sample_kettle_trait/model.rs` | 1 | 126 | 111 | 111 |
| Frozen `Kettle.scala` | 1 | 82 | 71 | 57 |
| All shared fixture Models (including both Kettles and controls) | 23 | 984 | 851 | — |
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
The associated `State: Clone + Eq + Hash + Spell` admits generic state values satisfying those
bounds, without a whole-state enumeration. Manual worlds can therefore compute starts and
action catalogs from their parameters. The missing R17 engine-side convenience is `Finite`
and its derive/domain support; persistent collections are admitted dependencies, not a ported
Bank or Voting implementation.

This is a trait-contract conclusion, with an explicit inference caveat: Rust does not infer
the world type backwards through every associated-state projection. Manual registrations may
need `Claim::<Self>` / `Progress::<Self>` or an explicitly typed closure parameter. That is
type annotation at the call site, not evidence that the attribute surface supports parameters.
Kettle itself proves only its unit-world, nongeneric, unit-action use case.

| Attribute gap in the final code | Settled future shape, delivered with the first sample needing it |
| --- | --- |
| No declared world struct: `#[machine]` emits `struct World;` and `world()` returns that unit value. | A module-local `#[derive(World)]` struct carries parameters; absence keeps the unit world and Kettle unchanged. |
| Generated handler and claim calls discard the world argument. | An optional explicit extra `w: &World<..>` parameter on handlers and claims, detected by arity as `before` is today. |
| `#[starts(range)]` supports static ranges, including `COLDEST_TEMPERATURE`, but not world-dependent/computed starts. | A `#[starts]` impl block supplies computed starts; keep field `#[starts(range)]` for literal/static domains. |
| `derive(Action)` requires unit variants; no input fields or domains. | Action variants carry inputs; domains come from `Finite` or the world. |
| `#[machine]` rejects generic registered impls and its generated adapters name bare nongeneric types. | Accept generics on `Model`, `World` and action enums and propagate their bounds. Standalone State derive's existing generic support does not close this gap. |

Numeric `.within(n)` is planned in fn-214. The current manual trait's `.within(scope)` takes
an environment assumption, not a numeric bound; its legacy `Progress` compatibility remains.
The attribute `LeadsTo` has `whenever`, `eventually` and `assuming`, with no `within` method.
Once world access and the numeric bound builder land, `w.resource_managers.len()` needs no
additional authoring form to supply a runtime parameter-derived bound. This follows from the
planned explicit Rust parameter and expression argument; no such bound is evaluated by today's
Kettle or generated code. `path!` remains fn-215.

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

The owner-approved priority is fn-214, fn-216, fn-213, fn-215, fn-212, fn-218, fn-217, then
fn-211. This is scheduling order, preserving each spec's prerequisites and approval boundaries.
The post-verdict Case plan must carry these requirements; none is implemented by fn-210:

| Requirement | Case-plan obligation |
| --- | --- |
| R33, later owner amendment | Keep today's proto limited to Explore, Rows and Refusal. Umpire copies useful Testpilot messages into its own proto and owns the copy; no schema move/import. The `umpire.model.v1` namespace rename is fn-216.2 future work. The Program remains data the Model replays; observations enter the Model and its judgment supplies the Verdict. Remove Contract/correlated-clause/CEL verification. Case identity names the Model content hash, Model and protocol version. |
| R34, tightened by R35/R37 | Target at most **5,000 generic** and **11,000 total non-test Go lines** in Testpilot, down from the historical approximately 20,000 / 29,000 inventory. Remove `internal/verification` (2,725) and `evaluation` (849); shrink `internal/ir` (4,105) to instruction decoding, `internal/execution` (6,575) to act-and-observe, `replay` (1,513) to Model search, and `campaign` (1,261) to the host loop. Retain needed recording/storage and most Temporal driver work (about 8,600 before callback simplification); report achieved sizes. Older 6,000/14,000 and 5,000/13,000 targets are superseded. |
| R35 | Remove all CEL, `cel`/`predicate`, the IR CEL bridge and `cel-go`. The Model decides read completion on each observation. Instructions contain interval/timeout/backoff mechanics; capture and redaction are field paths. A read timeout without Model acceptance is an incomplete read in the Run, judged by the Model. Execution checks still enforce RPC completion, evidence well-formedness and redaction. |
| R36 | Keep `#[machine]` registration-only. Lifted trees, subset diagnostics and TLC's second opinion belong to fn-212, which first judges whether their machinery is worthwhile. |
| R37 | At every SDK callback ask the Model what to return; serialize callbacks into the run's one Model instance. Remove worker interpreter/expression/typed-script behavior. Record a callback the Model cannot answer in the Run for Model judgment. |
| R38 | Replace the separate run/fuzz/replay/assess/repeat commands with one host loop and modes/drivers for a Model question, exploration, reduction, recorded Run or many Runs. Remove the NDJSON campaign/replay bridge. |
| R39 | A Run contains its observation log and Model judgment. Remove evaluation profiles, receipts and provenance documents. Keep `publish` as a never-overwrite store and keep only the canary's lease and policy. |
| R40 | Host exactly one Model per Run in the Go runner. Go callbacks stay in process; other SDK workers use the runner's local gRPC/HTTP endpoint with protobuf messages and a thin forwarding shim returning result/failure/heartbeat/delay. Workers never embed another Model. Record unreachable endpoints and callback timeouts in the Run for Model judgment; refuse unknown shim protocol versions with Refusal. |

## Final measurements and editor evidence

Final-source native exploration/checking, process wall, clean/no-change/edit builds,
macro-enabled rust-analyzer observations, parameterized-world probe, guest size/cold load/Rows/
memory and warm `make model-time` observations are recorded by the measurement lane here.
The historical observations below are not substitutes for those final measurements.

## Historical measurements

The direct-export guest built for fn-210.16 on 2026-10-18 is 3,370,765 bytes,
including debug information, using the same release profile and toolchain below.

The following host latency and memory measurements were taken for fn-210.9 on
2026-10-18, before the direct-export rename in fn-210.16. They do not measure the
current binary; fn-210.7 will refresh them.

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
