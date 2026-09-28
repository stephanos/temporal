---
satisfies: [R3, R4, R11, R14, R15]
---
# fn-92-compose-entity-machines-into-one-system.2 The compose command: generated unions, synchronization, key resolution, reachable literal table

## Description
Add `compose` to `Umpire.Command` with `for:`, `state:`, `members:`, `sync:`, `starts:`, `ends:`; generate the tagged-union Action, Outcome, and Fact types; key unsynchronized actions `<field>_<member key>` and synchronized ones by their `sync:` name, resolving dotted author references to those keys; lower member state fields as `<field>_<memberField>` state fields; extend Scenario and Property key resolution; build the reachable literal table by a frontier walk; elaborate to `Umpire.Command.DeclaredModel`; reject a `set` over a composition; list the fixture Queries in the Umpire differential sweep (R3, R4, R11, R14, R15's lowering half). Proofs and the per-composition check are task .3; field-addressed requirements are task .7.

**Size:** M
**Files:** `model/Umpire/Command/Compose.lean` (new), `model/Umpire/Command/Syntax.lean` (keyword, keys, elaborator dispatch, Scenario `actions:` and `starts:` resolution, Property `when:` resolution, `set` rejection), `model/Umpire/Command/Registry.lean` (record compositions), `model/Umpire/Command/Tests/Compose.lean` (new: fixture compositions, located errors, fingerprint test, a whole-state `verify` Query, a `find` Query with a composed witness), `model/UmpireTests.lean` (import), `model/Umpire/Search/Tests/Differential.lean` (import the fixture module; expected lines in the `sweep [\`Umpire]` block at :570-582)
**Touches:** [model/Umpire/Command/Compose.lean, model/Umpire/Command/Syntax.lean, model/Umpire/Command/Registry.lean, model/Umpire/Command/Tests/Compose.lean, model/UmpireTests.lean, model/Umpire/Search/Tests/Differential.lean]

### Approach
- Keyword via `declarationKeyword` (`Syntax.lean:381-390`); `compose` is not the reserved party spelling (`:1529`, `:2721`). Machine keys are the `machineKey` category (`:1663-1674`); the composition gets its own category.
- Generate `<name>.Action`, `<name>.Outcome`, `<name>.Fact` inductives by quoting `inductive … deriving DecidableEq, Repr, …` into `elabCommand` (`Syntax.lean:148-150` shape), one constructor per member field wrapping the member's type; catalog keys `<field>_<member key>` and composed state keys `_`-joined in field order, as `slottedKey`/`slotsKey` spell them (`Instances.lean:49-53`), because `FiniteCatalog.validKey` (`Table.lean:31-33`) admits only alphanumerics, `-`, and `_`; a member field name containing `_` is a located error; member Definition IDs kept inside constructors as `instances` keeps them (`Instances.lean:72-190`, `origin.ownedId` 84-89); synchronized action constructor and key named by the `sync:` line; the `-compose-<name>` owner formed as `instancesOwner` (`:60`) forms its own.
- State fields: `DeclaredModel.stateFieldIds`/`stateFieldValues` (`Authoring.lean:189-195`, `names.stateFields` 313-316) list each member's fields under `<field>_<memberField>`, never the member as one field; `Predicate.lean:14-30` (a disjunction across fields is refused) is why. Fixing a field is task .7; this task's fixture `verify` Query names a unique whole composed state.
- Frontier BFS from composed start states over member tables (`reachableFrom`, `Finite.lean:167`, as the shape): unsynchronized action steps its owner; synchronized step enabled only when every participant has a row, results the ordered product sorted by `stepOrderKey` (`Instances.lean:102-104`), refused above 16; classless participants match every class; timers, `unobservable:`, `evidence:` lifted per member under `<field>.`; catalog = reachable states sorted by an injective total key built on `modelValueOrderKey` (`:91-97`), sorted at elaboration with `List.mergeSort`, then emitted as a literal `FiniteTable`; bound check reachable states × actions vs `enumerationBound` (`Finite.lean:123`); drop never-enabled actions with a located warning.
- Key resolution: the Scenario elaborator keeps the last dotted component of an action term (`actionKeyOf`, `Syntax.lean:246-257`, used at `:598-606`) and splits `starts:` on `-` (`:570`); extend both, and Property `when:` (`propertyWhen` `:393`, `:442-451`, bare or classed), to resolve the dotted `operation.handlerReply (async)`, bare `operation.handlerReply`, and `sync:` names to the `_` catalog keys; composed `starts:`/`ends:` name member-qualified values, replacing the one-carrying-field rule (`:2126-2158`, `endsAcrossFieldsMessage` 1743). Every path that reads `members (α := State)` for a machine (`:409-410, 429-430, 2105, 2326-2347, 2403-2406`) reads the catalog for a composition.
- Elaborate to `declareModel` (`Authoring.lean:284`, emitted at `Syntax.lean:2401-2408`) and record with `Registry.recordModel` (`:2470-2481`); a `set` naming a composition is refused beside `systemBoundMessage` (`:2721`).
- Located errors pinned with `#guard_msgs`: unsynchronized shared name, participant input mismatch, `sync:` naming a timer, member field name containing `_`, product above 16, bound exceeded with counts, `starts:`/`ends:` value naming no member field, `set` over a composition.
- Fingerprint test: reorder `members:` and `sync:` lines in a fixture composition and assert equal fingerprints; reorder the `state:` fields and assert a different one. Pin `FiniteCatalog.Valid` on the composed catalogs.
- Fixture Queries: one `verify` whose Property names a unique whole composed state (every field, as `succeededOnRetry` does in `Caller/Model.lean:485-494`) and one `find` whose witness is a composed trace (R14); import the fixture module in `Search/Tests/Differential.lean` and add their lines to the Umpire sweep's expected block, both reading `veil default`.

### Investigation targets
**Required:**
- `model/Umpire/Command/Instances.lean:60-104`
- `model/Umpire/Command/Syntax.lean:242-330, 393, 442-451, 555-638, 1663-1674, 1873-2211, 2326-2347, 2401-2408, 2470-2481, 2637-2721`
- `model/Umpire/Command/Authoring.lean:170-210, 284, 313-316`
- `model/Umpire/Command/Finite.lean:82-170`; `model/Umpire/Command/Predicate.lean:14-30`
- `model/Umpire/Search/Tests/Differential.lean:534-582`; `model/Umpire/Search/Selection.lean:38-59`

### Key context
- The author's state structure derives no `Finite`; the reachable list is the catalog.
- `UmpireTests` is an explicit import list (`model/UmpireTests.lean`); nothing globs `Command/Tests/`.
- `List.mergeSort` is stable: ties would carry `members:` order into the fingerprint, so the sort key is injective.

### Quick commands
```bash
cd model && lake build Umpire.Command Umpire.Command.Tests.Compose Umpire.Search.Tests.Differential
LEAN_NUM_THREADS=1 make lint-model
```
## Acceptance
- [ ] `compose` elaborates a fixture composition to an `Umpire.Command.DeclaredModel` usable by `property`, `scenario`, `limits`, `query`, with Scenario `actions:`/`starts:` and Property `when:` (bare and classed) resolving composed and synchronized keys, and `starts:`/`ends:` taking member-qualified values
- [ ] Generated unions with `validKey`-legal `<field>_<member key>` keys, composed catalogs pinned `Valid`; no catalog collision for two members with `accepted`; member state fields lowered as `<field>_<memberField>`; a whole-state fixture Property enumerates and verifies
- [ ] Reachable literal table; fingerprint unchanged under `members:`/`sync:` reordering and changed under `state:` field reordering (pinned)
- [ ] Every listed located error pinned; `set` over a composition rejected
- [ ] The fixture `verify` and `find` Queries appear in the Umpire differential's expected block as `veil default` with both backends agreeing, and the `find` witness passes the kernel replay gate (R14)
- [ ] `lake build UmpireTests` and `LEAN_NUM_THREADS=1 make lint-model` pass
## Done summary
Added the `compose` command to `Umpire.Command`. It takes `for:`, `state:`, `members:`, `sync:`, `starts:`, and `ends:`. It generates tagged-union Action, Outcome, and Fact types without injectivity lemmas. It keys member actions as `<field>_<key>` and synchronized actions by their `sync:` name. It lowers member state fields as `<field>_<memberField>`, or `<field>` for a one-field member. It walks the reachable states breadth-first over the member tables, in the new `Umpire.Command.Compose` module, and emits the result as a literal `FiniteTable` through `declareModel`. Every catalog, the start states, and each row's results are sorted by the lowered order key. Scenario `actions:`/`starts:` and Property `when:` resolve dotted and `sync:` references. A bare classed `when:` claims every class. A `set` over a composition is refused.

`model/Umpire/Command/Tests/Compose.lean` pins these:
- the composed catalogs, and `FiniteCatalog.Valid` on them;
- a fingerprint that is unchanged when `members:`/`sync:` lines are reordered and changes when the state fields are reordered;
- ten located errors, including the review-added `_`-in-member-state-key and sync-name collision cases;
- a never-enabled warning.

Four fixture Queries (three `verify`, one `find` with a composed witness) read `veil default` in the Umpire differential sweep.

Deviations:
- The owner key is `compose-<name>`, because a leading `-` would make an odd ID segment.
- Lifted timers, unobservable entries, and evidence are recorded in `Registry.CompositionEntry` under the composed `<field>_<key>` spelling, not `<field>.`, to match the catalogs.
- Nothing in the elaboration or enumeration needed a proof. Composition proofs are task .3's.

Review round 1 (Codex fan-out) was NEEDS_WORK with 5 findings, all fixed in 48a0ba914a:
- a bare classed `when:` was not expanded to every class;
- a sync name could collide with a member action key;
- member state keys containing `_` made composed keys non-injective;
- input compatibility compared spellings, not domains;
- Scenario start resolution matched only the first key segment.

Round 2 was SHIP. The first `lint-model` run failed on simpNF, because a union constructor whose payload domain has one member gets an `injEq` that simp proves by itself. 05fae56ae1 sets `genInjectivity false` on the generated unions, and round 3 was SHIP.

baseline: green (focused lake build, pre-edit). lint-model: rc=0.

stage: impl-review - ran [round 1 NEEDS_WORK (codex fan-out, rid f16b1b0f4b7d4b40acf13797aa29852c) .. round 3 SHIP]

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 881a7b413addb72615dae277337a0e8f25d8232d, 48a0ba914a33f6675fa7cdaa6052041c818a4202, 05fae56ae1228b56dfd3b7b81607810a3226dd13
- Tests: cd model && lake build Umpire.Command Umpire.Command.Tests.Compose Umpire.Search.Tests.Differential, cd model && lake build UmpireTests, LEAN_NUM_THREADS=1 make lint-model
- PRs: