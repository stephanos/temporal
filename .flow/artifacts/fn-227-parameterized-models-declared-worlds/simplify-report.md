# fn-227 simplification scout (read-only), 2026-10-21 at e68f3d4a20

Scope: the four built tasks (.1, .7, .2, .8). I read the code behind every candidate and
re-derived the prior audit's numbers where I use them (`git diff --numstat a76c358162..HEAD`,
`wc -l`, line ranges quoted below). Constraints honoured throughout: plain Rust first; TLC and
Apalache scaling matter; the lifter (`model/umpire/lift`) stays and keeps translating
transitions and claims from source on its own; only values cross the native/source boundary.

## Verdict: too complex?

Yes, in two places, and not in the semantics.

- The semantics are about the right size. Declared World, computed starts, `Finite`,
  `#[domain]`, checked catalog admission, bound CONSTANTs, whole-state Init, finite existential
  inputs and exact-payload fairness are each a few hundred lines on each side and are what the
  requirements ask for. TLC agreement on both fixtures holds for all five CFG families.
- The excess is in two mechanisms the planning process invented, not in anything a consumer
  asked for:
  1. **The value handoff from native to lifter (.7/.8)** is done by text-generating an
     augmented copy of `model.rs` in `OUT_DIR` at build time, splicing a projection function
     into the machine module, rewriting `tests.rs` around its `fixture!` hook, and routing both
     through discovery "overrides". The job it does is: read a World's fields, each start's
     fields, each catalog action's payload and each `#[domain]` function's values as
     `Literal`s. A derived `#[doc(hidden)]` method does that in about a tenth of the code, with
     no build-time code generation at all. Roughly 600 lines go, including the thread-local
     `SELECTION`/`CONSTRUCTIONS`/`ENUMERATIONS` scaffolding in the fixtures.
  2. **Generic `#[machine]` modules (.1)**. About 560 of `declared.rs`'s 892 lines plus
     `readonly.rs` and roughly 600 test lines exist to let `Model<T>`, `World<T>` and
     `User<T>` carry generics through the generated adapters. No Model uses it (every hit for
     `struct Model<`/`struct World<` is a test), and the lifter refuses generic source
     (`shape.rs:171`, README:665). It came from one spec line ("Generics. `#[machine]` accepts
     generics...", spec:29) that no consumer (fn-228, fn-211, fn-233) cites. Removing it is
     what makes the two machine generators mergeable.

Everything else is ceremony measured in tens of lines; some is worth taking along, none is
worth its own task.

Estimated total if the ranked plan below is done: about **1,300-1,500 source lines and
900-1,100 test lines removed**, with checker/report goldens, TLA+/cfg goldens and TLC
agreement unchanged. Only macro *expansion* snapshots for the declared-World fixture change.

## Candidates

### 1. The World handoff to the lifter (.7, extended by .8)

**What it does.** `lift/src/build.rs:105-303` (`Bound::read`) parses the fixture's `model.rs`
and `tests.rs` with `syn`, finds a free identifier prefix by trying every
`__umpire_bound_{n}_` up to the file length (`build.rs:121-124`), injects two generated
functions into the machine module (`#project`, reading private World/state fields into
`umpire_lift::Binding`, and `#identity`, mapping each `Action` to `umpire_lift::Concrete`),
replaces the `fixture!` invocation in `tests.rs` with six generated functions (`select`,
`bind`, `module_fn`, `native`, `run`, `backend_fn`), writes both files to `OUT_DIR`, and
returns them as discovery overrides. `testing/src/discovery.rs:72-95` validates and applies the
overrides; `testing/tests/discovery.rs:85-200` tests them. `build.rs:317-463` holds the
projection helpers (`project_fields`, `project_value`, `identity_arms`, `domain_values`,
`negative_runtime`, `contains_prefix`). `testing/src/backend.rs:86-96` adds
`check_typed_exploration` to carry the generated identity closure. In the fixtures, because
`run()`/`lifted(name)`/`render_tlc(run, folder)` are name-keyed and take no World, each test
selects a World through a `thread_local!` cell and counts constructions/enumerations to prove
"one selection per execution" (`parameterized_world/model.rs:6-15`, `tests.rs:5-34`;
`action_inputs/tests.rs:5-27`).

**Why it exists.** .1 Approach 4 and its done summary: "Authored field privacy is preserved with
no reflection/getter catalog", "expose existing field names/types rather than a second mirror
catalog or serialization subsystem". .7 Approach 2-4 then derived the only way to read private
fields without a getter: compile a copy of the module with the reader inside it. The
"selection stays in its fixture lexical scope" and "one owned World, one enumeration" rules
produced the `tests.rs` splicing and the counters.

**The simpler alternative.** The lifter already consumes plain data (`Lifted::bind(&Binding)`,
`tree.rs:102`); only the *producer* of `Binding` is baroque. Replace it with a derived
conversion:

- In `umpire_dsl`: a small `Value` enum (`Bool`, `Int(i128)`, `Enum(&'static str)`,
  `Seq(Vec<Value>)`) and a `Values` trait (`fn values(&self) -> Vec<(&'static str, Value)>`),
  with `IntoValue` impls for `bool`, the admitted integer widths, `imbl::Vector<T>`, and a
  derive for fieldless enums (ride on `derive(Finite)`, or a two-line derive).
- `derive(World)` and `derive(State)` emit `#[doc(hidden)] impl Values`. `derive(Action)` emits
  a hidden `fn concrete(&self) -> (&'static str, Vec<(&'static str, Value)>)` and, for
  `#[domain]` fields, `fn domains_in(world: &W) -> Vec<(&'static str, &'static str,
  Vec<Value>)>`. These are the same shapes `project_value`/`identity_arms`/`domain_values`
  generate today, moved from build-time text into the proc macro where they are typed and
  hygienic by construction.
- `umpire_testing` gains one function,
  `bound::<M>(world, budget, negative) -> Bound` (World values, `Machine::starts` enumerated
  once, `checked_catalog(CATALOG_LIMIT)` projected once, domains), and `fixture!` grows one arm
  that also emits `pub(crate) fn bound() -> umpire_testing::Bound` for declared-World fixtures.
  `lifted(name)` in `build.rs` becomes
  `"{name}" => Lift::source(path).and_then(|l| l.bind(&{id}_tests::bound().binding())).and_then(Tla::emit)`.
- The fixture tests then call `umpire_testing::bound(World::new(0, 1), 64, claim)` directly
  for their second and third Worlds; the thread-locals and counters go away. "One selection,
  one enumeration" becomes a property of a ten-line function instead of a test obligation.

The lifter's independence is untouched: `Lifted::bind` still validates names, counts, types,
order and the catalog product against its own reading of source (`tree.rs:102-335`), and
transitions/claims never see native data. Privacy: `derive(State)` already emits `Debug` and
`Display` printing every private field (`state.rs:96-108`); a hidden `values()` exposes
nothing `Debug` does not.

**Lines.** Source: `build.rs` −360 (lines 105-463; `generate_bound` collapses into
`generate`), `discovery.rs` −30, `backend.rs` −15 (`check_typed_exploration` folds into
`check_exploration` taking `&dyn Fn(&Action) -> Concrete`), `tests/build.rs` −8;
plus roughly +120 in `dsl/macros` (`state.rs`, `action.rs`) and `dsl/src` (`Value`,
`Values`), +40 in `testing`. Net about **−250 source**. Tests: `testing/tests/discovery.rs`
−116, the two fixtures' thread-local/counter scaffolding −60, `build.rs` unit tests −50
(replaced by one smaller test); net about **−200 test**. Build-time compile cost also drops
(no second compile of each bound `model.rs`).

**What changes / risk.** Checker, rows, states, paths, negative report, TLA+, cfg and
`tlc.txt` goldens: unchanged (same World, same starts, same catalog order, same `Binding`).
`tree.rs` and `emit.rs`: unchanged. Reverses .1's "no reflection/getter catalog" wording;
record the decision. Risk: low; the `Binding` validators in `tree.rs` and `lift/tests/world.rs`
(`binding_refuses_malformed_world_and_complete_state_values`) keep guarding the contract.

**Owner decision folded in (via the coordinator, 2026-10-21).** The rule "no native
reflection/serialization interface" is dropped for values only: `derive(World)` and
`derive(State)` expand inside the type's own module, so they generate a small method exporting
ordered field values in the lifter's existing Type/value representation; field visibility stays
the author's. The lifter still admits and translates `model.rs` on its own; only the selected
World's values and the start states cross. On that basis:

- **Deleted.** `lift/src/build.rs:14-21, 40-57, 105-463` (`generate_bound`, `Bound::read`, the
  projection/identity/domain generators, `negative_runtime`, `contains_prefix`, the prefix
  search): −360. `testing/src/discovery.rs:72-95` (`generate_with_overrides`) and the override
  wiring in `tests/build.rs`: −35. `testing/tests/discovery.rs:85-200` (override tests): −116.
  `build.rs` unit tests: −50. The `SELECTION`/`CONSTRUCTIONS`/`ENUMERATIONS` thread-locals and
  the `selected()`/`selected_once()` helpers in both fixtures: −60 test (moot: one World is
  built and enumerated once inside one plain function, visible by inspection).
  `backend.rs:86-96` (`check_typed_exploration`) folds into `check_exploration`: −15.
- **Added.** In `umpire_dsl`, a `Value` enum mirroring `umpire_lift::Literal` (`Bool`,
  `Int(i128)`, `Enum(&'static str)`, `Seq`) and a `Values` trait (`fn values(&self) ->
  Vec<(&'static str, Value)>`), with `IntoValue` for `bool`, the admitted integer widths,
  `imbl::Vector<T: IntoValue>` and fieldless enums (via `derive(Finite)` or a two-line
  derive): about +60. Hidden `impl Values` from `derive(World)`/`derive(State)` in `state.rs`
  (one `quote!` block over the field list it already iterates): +25. In `derive(Action)`, a
  hidden `concrete(&self)` (variant name plus ordered payload `Value`s) and, for `#[domain]`
  fields, `domains_in(&World)`: +40. In `umpire_testing`, `bound(world, budget, negative)`
  building `umpire_lift::Binding` (`Value` → `Literal` conversion lives here, so Model crates
  never depend on `umpire_lift`), plus a `fixture!` arm emitting `bound()` for declared-World
  fixtures: +50. `lifted(name)` in `build.rs` binds at runtime through that function: +10.
  Net about **−250 source, −220 test**.
- **Validation stays.** `Lifted::bind(&Binding)` (`tree.rs:102-143`) keeps checking World and
  start rows for missing/extra/duplicate fields and admitted types (`bind_fields`,
  `tree.rs:438-469`), TLC `.cfg` range (`cfg_representable`, `:471`), World-domain values
  against the field's admitted type and the catalog limit (`bound_domains`, `:152-226`), and
  the native catalog against the source's declaration-order product, ordinal by ordinal
  (`bind_catalog`/`compare`, `:229-399`), plus assumption membership (`scoped_catalog`). The
  `Binding` type and these tests (`lift/tests/world.rs::binding_refuses_malformed_world_and_complete_state_values`,
  `lift/tests/inputs.rs::malformed_bindings_refuse_at_the_source_declaration`) are untouched.
  The derive only replaces who fills `Binding`; the lifter's schema remains the arbiter.
- **Goldens/TLC.** None change: same World values, same `Machine::starts` order, same
  `checked_catalog` order, same `Binding` → same CONSTANTs, Init, input relations, `tlc.txt`.
  Checker goldens are untouched by construction (native side unchanged).

**Verdict: do it (owner-approved).**

### 2. Two machine generators (`machine.rs` 969 + `machine/declared.rs` 892)

**What it does.** `machine.rs:874` branches to `declared::expand` when a `derive(World)` struct
is present. The declared path re-emits the `Machine` impl with: a typed World, computed
starts, World-aware claim adapters, `ActorCatalog`-based catalog routes, and `Associated`
instead of `Registered`. Of its 892 lines, the parts that differ *because a World is declared*
are small: `starts()` (223-268), `declarations()` (270-323), `catalog_routes()` (640-678),
`invariants()` (718-734), the `assumptions` closure and the `impl Associated` (537-580).
The rest exists to support generics on the module's types: `merge_bounds`, `ParameterNames`,
`application_parameters`, `implementation_names` (45-221, 177 lines), `machine_generics`,
`resolved_bounds`, `implementation_bounds`, `actor_declarations` (325-483, 159 lines),
`SelfWorld` (705-716), the hand-written `Clone/PartialEq/Eq/Hash/Debug/Display` carrier impls
(`action_carrier`, 736-781, needed only because `#[derive]` cannot express the per-type
bounds), and `GenericHelpers`/`without_defaults`/`helpers` (783-892, 110 lines, which rewrites
the shared progress helpers to carry generics). `machine/readonly.rs` (55 lines) and the
`check_receivers_with_generics`/`state_generics` plumbing in `machine.rs:193-348` exist for
the same reason (f0adb5afdb "lexical readonly probes").

**Why it exists.** Two rules. (a) .1 Approach 1, "Keep the existing absent-World expansion
branch literally unchanged", protects the six unit expansion snapshots byte for byte
(`machine` 520, `rejecting_machine` 187, `staying_machine` 178, `state` 97, `action` 181,
`path` 82 lines). (b) spec:29 "Generics. `#[machine]` accepts generics on `Model`, `World` and
action enums". (b) is what makes (a) expensive: without generics, the declared path is a
`world: Option<&ItemStruct>` switch inside one generator.

**Simpler alternative, two steps.**

1. **Drop module-level generics in `#[machine]`.** Keep the standalone derives' existing
   generic support (`state.rs` `bounded()`, `action.rs` `#types`/`#constraints`), which predate
   fn-227. Refuse generic `Model`/`World`/actor declarations inside `#[machine]` with one
   located error, matching what the lifter does already. This deletes about 560 lines of
   `declared.rs`, `readonly.rs` (55), and the `parameterized`/`state_generics` branches in
   `machine.rs` and `registration.rs` (about 60).
2. **Merge.** With generics gone, `declared::expand` is `machine::expand` with a `world`
   option: `struct World;` vs the authored struct; `#state::starts()` vs
   `<#state>::#starts_method(self)`; `#actors::actions()` vs
   `<#actor as ActorCatalog<Self>>::actions(self)` per dependent actor; `|_, state|` vs
   `|__umpire_world, state|` adapters (the `world_methods` set already drives this in
   `postconditions`/`progress_claims`/`stepping`, which are shared today); `impl Registered` +
   `fn world()` vs `impl Associated`. The unit tokens can be kept identical, so the six unit
   snapshots need not change; I would still allow them to change rather than contort the
   code.

**Lines.** Source: `declared.rs` 892 → folded, `readonly.rs` −55, `machine.rs` −60; the
merged generator is about 1,050-1,100 lines (vs 1,916 today), i.e. about **−800 source**.
Tests: `dsl/tests/generic_world.rs` −285, `generic_bounds.rs` −155, `generic_readonly.rs`
−62, `ui/world_generic_*` −100, `dsl/macros/tests/fixtures/world/model.rs` rewritten
non-generic (snapshot 69 lines regenerates); about **−600 test**, plus one new trybuild case
"generic declarations are not admitted in `#[machine]`".

**What changes / risk.** Checker and TLA+ goldens: none (no fixture or Model is generic).
Expansion snapshots: `declared_world` changes (derived carrier impls replace the hand-written
ones; shorter); unit snapshots unchanged if the merge keeps tokens, else regenerated once.
Spec:29 and README:197 are amended; DESIGN.md:503 row ("Accept generics...") is retired.
Reverses a spec line, so it needs the owner's nod; the owner's rule "Author ergonomics over
plugin convenience" favours it, since no author has asked for it and the lifter cannot take
it. Risk: low; the generic tests are the only consumers.

**Verdict: do it** (step 1 is the decision; step 2 follows mechanically).

### 2b. Coordinator addition: method-level `#[starts]`

**Owner decision.** Computed starts move from a separate `#[starts] impl Model { fn
starts(&World) }` block to a method-level `#[starts]` on an associated fn of any name inside
the main `impl Model`, beside `#[on]`, `#[assumption]`, `#[scope]`. Field `#[starts(range)]`
stays; declaring both stays a compile error.

**Macro change.** `declared.rs:223-268` (`starts()`, which finds the `#[starts]` impl, checks
its single `fn starts(world: &World)`, strips the attribute and tags the State struct with
`#[__umpire_computed_starts]`) is replaced by one more kind in `registration.rs`: add
`"starts"` to `KINDS` (rank 0, so `check_order` keeps it first; the handler/postcondition
arms become 1/4/5/6), accept it on a no-receiver associated fn whose only parameter is the
trailing `&World` (`world_parameter` already classifies that), record the method name in
`Registrations`, and refuse a second one with the existing "declare starts once". The `impl
Machine` emits `fn starts(&self) { <Model>::#name(self).into_iter().collect() }`. The State
tagging and the field-`#[starts]` conflict check stay as they are. About −45 +30 lines; the
`check_impl(.., true)` relaxation for the separate impl (`machine.rs:126-142`) is no longer
needed.

**Lifter.** `shape.rs:203-252` locates starts by scanning for an `Item::Impl` with the
`starts` attribute and validates it as a separate block; `shape.rs:261` and `:422` skip that
impl when reading methods. Replace with: in the main-impl method loop (`shape.rs:~430`),
`has(&method.attrs, "starts")` → validate signature (no receiver, no positional parameters,
`world` present, not async/unsafe/generic; the same checks as today's 239-251), require a
declared World, refuse a second occurrence, mark `Start::Computed`. Remove `"starts"` from the
allowed impl attributes at `shape.rs:109`. `Init` emission is unaffected: it comes from the
bound `complete_starts` (`emit.rs:95`), never from the method body. About −50 +30 lines.

**Tests and docs.** Trybuild: `world_starts_mixed.rs`, `world_starts_empty_mixed.rs` move the
method into the main impl (stderr regenerates with new spans, same message); add one case
"`#[starts]` on a method with a receiver" and one "`#[starts]` twice". Fixtures:
`parameterized_world/model.rs:37-50` and `action_inputs/model.rs:30-35` move one method each.
Other `#[starts]` impl sites to migrate: `lift/tests/world.rs` (6), `lift/tests/inputs.rs` (1),
`dsl/tests/declared_world.rs` (1), `suite/tests/declared_world.rs` (1),
`suite/tests/world_domains.rs` (1), `checker/tests/catalog_admission.rs` (2), and the generic
tests if they survive candidate 2. Docs: README:176-181 and :199-203, DESIGN.md:500 (the
"`#[starts]` impl block" row), spec "API Contracts" bullet "`#[starts] impl Model { fn
starts(w: &World) -> .. }`" and the R1 wording. The `.flow` edits belong to the agent editing
planning files now; I did not touch them.

**Delta and goldens.** Net about −35 source; tests about ±0 after migration. Checker goldens:
none (same start sets). TLA+/cfg goldens: none. Expansion snapshot `declared_world`: changes
(it already changes under candidate 2).

**Verdict: do it, inside the same task as candidate 2.**

### 3. Requirement-driven machinery elsewhere

| Item | What / why | Alternative | Lines | Verdict |
| --- | --- | --- | ---: | --- |
| "One enumeration per execution" counters (`parameterized_world/model.rs:6-15`, `tests.rs:7-34`) | Test scaffolding proving .7 Approach 4's rule. | Disappears with candidate 1 (the rule becomes a property of `umpire_testing::bound`). | −35 test | do (with 1) |
| `Checker::explore_from` / `refuted_from` (`explore.rs:92-99`, `:526-543`) | Starts-taking twins of `explore`/`refuted`, added so bound glue does not re-enumerate. | `explore_from` duplicates `explore_within(.., CATALOG_LIMIT)`; `refuted` could take starts. ~25 lines. Essential semantics are "explore these starts"; the twins are naming. | −25 src | maybe; fold into candidate 1's cleanup |
| `Space<S, W = ()>` with two `SpaceBinding` impls (`space.rs:16-199`) | b5b91b4f82 "preserve legacy spaces": a `Space` for an *unregistered* state must still be constructible and configurable (`suite/tests/retained.rs:139-144`), and `Space<S>: Send + Sync` without `World: Send`. | `Space<S>` always holds `Rc<S::World>`; `From<[S; N]>` requires `S: Registered`. Drops the `W` parameter, one impl and the sealed trait's second implementor. The unregistered-state test becomes a compile-fail doc test. | −45 src, −10 test | maybe; low value, touches a public type |
| `Space::with_world` + `SpaceBinding::starts()` | Explicit-start construction so the bound route can hand over already-enumerated starts. | Needed as long as a caller wants explicit starts; keep. | 0 | keep |
| `ActorValue` / `Actor` split (`dsl/src/action.rs`) | A World-dependent actor cannot implement static `Actor::actions()`; .2 forbade faking it. | Alternative is to make `Actor::actions` take `&World` everywhere, touching `path_build.rs` (`StaticDomain`, wildcard patterns, outcome filters) and all manual `Actor` impls. Larger than what it removes. | 0 | keep (note the `.internal()` ambiguity, .2 L5) |
| `Associated` blanket over `Registered` (`machine.rs:109-126`) | Lets unit states keep `Space::from([state])` while declared states need a World. | 12 lines; the alternative (everything `Associated`, `Registered` only for unit) is what it already is. | 0 | keep |
| `catalog_limit` threading (38 sites: `Space::catalog_limit`, `SpaceBinding::catalog_limit`, `Checker::explore_within`, `Result.catalog_limit`, `coverage`/`covered`/`search` params) | .2 #6 "a small additive checker/suite option supplies another limit". Only tests use a non-default limit (`checker/tests/catalog_admission.rs:241-297`). | One constant. Keep `explore_within` as a test seam or construct a 4,097-action Model (one test already does). | −70 src, −15 test | maybe; defer until a consumer wants a limit |
| `#[scope]` as a second spelling of `#[assumption]` (`machine.rs:112`, `registration.rs`) | .1 Approach 2 compatibility spelling. | One spelling. | −8 src | trivial; fold in if touching registration |
| `Machine::checked_catalog` default + `ActionCatalog::checked` (`catalog.rs`) | Count-before-materialize admission, overflow vs over-limit refusals. | Essential: it is the scaling guard (no 4,097-action product gets built). | 0 | keep |
| `tree.rs::compare` (`tree.rs:337-399`) | Per-ordinal, per-field located diagnostics when the native catalog disagrees with the source product. | `binding.actions != self.concrete_expected()` with one diagnostic naming the first differing ordinal. | −45 src; `lift/tests/inputs.rs::malformed_bindings_refuse_at_the_source_declaration` asserts on messages, −20 test | maybe |
| `build.rs:121-124` prefix search over `0..=len` | Collision-free generated names. | Gone with candidate 1. | — | do (with 1) |

### 4. Author-facing warts

- **Fair/only/without lists must be literal arrays.** This is native, not only the lifter: the
  generated helpers take `[A; N]` (`machine.rs:675`, `declared.rs:869`), and the lifter requires
  an array (`body/progress.rs:93`). For `Poll{expected_attempt}` with a World-sized domain the
  author must spell every payload. Cheap fix: helpers take `impl IntoIterator<Item = A>` (arrays
  still work, +0 lines), and the lifter admits one more form, `<domain fn>(world).map(Actor::Variant)`
  for a single-field variant, resolving to every concrete value of that variant in the bound
  catalog (`action_list` in `progress.rs`, about +40 lines). This keeps .8's rule that a bare
  variant never silently means "all inputs": the expansion is authored. **Verdict: do it when
  the activity retries task (.3) first needs it; not blocking.**
- **World-dependent actors lose wildcard `User::*` and `=> outcome` filters in `path!`**
  (README:507-513, .2 follow-up M1). The explicit `Pattern::action(User::actions_in(&world)...)`
  route works. Relaxing outcome filters to `ActorValue` is the recorded M1 fix (small,
  regenerates one stderr). **Verdict: maybe, with M1.**
- **`fixture!` selection by `thread_local!`** in both fixture `tests.rs`. Gone with candidate 1.
- **`#[starts]` as a separate impl with a forced name.** Covered by 2b.

### 5. Anything else disproportionate

- **`lift/tests/world.rs` (447) and `lift/tests/inputs.rs` (354)** hand-build `Binding`s and run
  TLC on each CFG family. They are the right tests for the `Lifted::bind` contract and stay.
- **`names.rs` +104**: collision-safe generated TLA+ names for World constants and input
  symbols (`allocate`, `validate`). Needed once source may declare `__umpire_world_x`-like
  names; proportionate.
- **`backend.rs` `typed_action` (`:233-279`)**: replay by catalog ordinal, cross-checked against
  typed metadata and the native identity. Proportionate to .8 #5's "never identify inputs by
  display text". Stays; its `identity` argument becomes the derived `concrete()` under
  candidate 1.
- **Planning weight** is the real disproportion: 142 KB of spec/task prose and 345 `fn227*`
  scratch files for about 4.9k source lines. Not code; noted for the owner.

## Ranked plan

Two tasks, scoped so checker/report goldens, TLA+/cfg goldens and TLC agreement stay
byte-identical. Expansion snapshots for the declared-World fixture are the only goldens
expected to change.

**Task A — one `#[machine]` generator, no module generics, method-level `#[starts]`**
(candidates 2, 2b, and the `#[scope]` trivia)

1. Refuse generic `Model`/`World`/actor declarations inside `#[machine]` with one located
   error; delete the generic plumbing (`declared.rs:45-221, 325-483, 705-716, 736-892`,
   `readonly.rs`, `parameterized`/`state_generics` branches in `machine.rs`/`registration.rs`).
2. Register `#[starts]` as a method kind in the main impl; emit `fn starts(&self)` from the
   registered name; keep the field/computed conflict check.
3. Fold `declared::expand` into `machine::expand` behind `world: Option<&ItemStruct>`, keeping
   unit tokens identical.
4. Lifter: locate `#[starts]` as a method (`shape.rs:203-252` → method loop); drop the impl
   skips at `:261`/`:422` and `"starts"` from `:109`.
5. Migrate the fixtures, tests and trybuild cases listed under 2b; delete the generic tests;
   add the two new trybuild cases. Update README:176-203, DESIGN.md:500-503, spec:29 and the
   R1 API bullet (planning-file edits go to the agent that owns `.flow/` now).
6. Gate: `cargo test -p umpire-dsl-macros -p umpire-dsl -p umpire-lift -p umpire-tests`,
   then the two `model-check` variants; diff every `expected/` outside
   `dsl/macros/tests/fixtures/{declared_world,world}` against HEAD and require zero changes.

Expected: about **−900 source, −600 test**.

**Task B — derived value handoff; delete the augmented-copy route** (candidate 1 as the
owner decided it, plus the `explore_from`/`refuted_from` fold and the counters)

1. `umpire_dsl`: `Value` (mirror of `umpire_lift::Literal`), `Values`, `IntoValue`; hidden
   `impl Values` from `derive(World)` and `derive(State)` inside the type's own module (reads
   private fields, visibility unchanged); hidden `concrete()` and `domains_in(&World)` from
   `derive(Action)`; fieldless-enum `IntoValue`.
2. `umpire_testing`: `bound(world, budget, negative) -> Bound` (one World, one
   `Machine::starts`, one `checked_catalog`, `Value` → `Literal` conversion here);
   `fixture!` emits `bound()` for declared-World fixtures; `check_exploration` takes the
   identity function directly.
3. `lift/src/build.rs`: delete `Bound::read`, the projection/identity/domain generators, the
   prefix search and `generate_bound`; the `lifted(name)` arm for a declared-World fixture
   runs `Lift::source(path)?.bind(&{id}_tests::bound().binding())` and `Tla::emit` at test
   time (unit fixtures keep the eager OUT_DIR artifacts and `check_machine`).
4. `testing/src/discovery.rs`: delete overrides and their tests; `tests/build.rs` back to
   plain discovery.
5. Fixtures: replace `thread_local!` selection with direct `bound(World::new(..), ..)` calls
   in the second/third-World tests; drop the counters.
6. `Lifted::bind` and all its refusals stay exactly as they are; no lifter semantic file
   (`shape.rs`, `body/*`, `emit.rs`, `tree.rs`) changes.
7. Record the decision: a hidden derived `values()` replaces the "no reflection/serialization"
   rule for values only (same exposure as the existing derived `Debug`).
8. Gate as in A, plus `UMPIRE_BACKENDS=require` on `umpire-tests` for both fixtures and all
   five CFG families; `tlc.txt`, `*.tla`, `*.cfg`, checker/report goldens byte-identical.

Expected: about **−250 source, −220 test**, no golden or TLC change, and one fewer compile
of each bound `model.rs`.

Order: A before B (A shrinks the macro crate B adds to, and B's derives slot into the merged
generator). Total expected reduction: **about 1,200 source and 850 test lines**, with the
`maybe` items (`Space<S, W>`, `catalog_limit`, `tree.rs::compare`) worth another 150 source
if taken opportunistically.

Not proposed: dropping or bypassing the lifter, native-graph export, or relaxing any
independence rule. Those stay as the owner set them.
