# The Umpire IR: breakdown, what the planned specs do to it, and how to shrink it

Status: analysis with recommendations, 2026-10-17. No source was changed.

Paths are relative to the repository root. `proto/` means
`proto/internal/temporal/server/api/umpire/v1/`. This note continues
`ir-simplification.md` (2026-10-13, the IR against Quint's, measured on the lifted samples) and
takes its figures as given where nothing has changed. What has changed since then is the frame:

- **fn-196 (2026-10-17): the Scala checker is the oracle.** The Go `check`, the Go interpreter's use
  as a checker and the planned IR evaluator go. The IR "remains the export and lowering format".
- **fn-190/fn-195: claims and Queries leave the Model.** Claims are `check` sections; `Limits`,
  `assume`, `queries` for anything but witnesses, `Exploration`, `.live` and `.explore` go; Cases come
  only from a ModelSuite's `conformance` section, each path replayed on the Model first.
- **fn-192: builtins.** One closed table, one generic `BuiltinCall`, no `Inbox` message, no
  `OP_CONTAINS`/`OP_CONCAT`, no `Collection` node.
- **fn-189 R5/R11: positions on root functions only; helpers inlined** as `Let`s at the call site.
- **fn-146: CEL for runtime predicates** (realization operands and Case guards), finite Model
  expressions untouched.

The 10-13 note asked "why is it bigger than Quint's". This note asks the narrower question the
new frame allows: **what must the IR carry once nothing executes it**, and what can go with no or
only minor trade-offs.

**Method.** Nine proto files read in full; the 26 open specs that touch the IR read and tabulated
(§2); Go readers mapped field by field (a getter and `oneof`-arm grep over `tools/umpire`, non-test,
with packages split into "retires under fn-196" and "survives"); the lifter's message use counted
(`grep -o 'ir\.[A-Z]\w+'` over `model/framework`); MILESTONES.md as of 2026-10-17 for phasing.
Quint facts are from `ir-simplification.md` §1 (verified there against `quintIr.ts` at
`b6db567bd5`) and are not re-derived here.

---

## 0. Answer in brief

1. **The IR is three IRs in one message.** 120 messages, 21 enums, ~427 fields in nine files. By
   layer: model core 48 messages (`value`, `expression`, `machine`), claims 12 (`claim`), live driver
   59 (`realization`, `script`, `operand`), plus the root and `Position`. Quint's IR corresponds to the
   first layer only, and that layer is already within 1.0-1.5× Quint's size without positions.
2. **After the open specs land, the readers of the IR are: the Quint/TLA exporter, the Case lowering,
   and lint.** Nothing evaluates it. Fields that exist for the evaluator have no reader left
   (§1.3): channel bounds, fairness selections, `Progress.within`, refinement visibility,
   `Scenario.start`, `TypeRef.channel`, `Machine.assumes`, `Model.assumptions`. The claims layer
   loses its reason to exist in the Model: a Query is a test, not a declaration (fn-195).
3. **The specs already remove more than the 10-13 note proposed**, but they disagree with each
   other about who reads the IR (§2.3). Eight open specs still assume a Go reader that fn-196
   deletes. That has to be settled before any schema cut, or the cut will be made twice.
4. **Shrinking is mostly deletion, not redesign.** With the Go lowering kept (the minor-trade-off
   line), the schema goes from 120 messages to about 80 and the core from 48 to about 32; sample
   files shrink by roughly half again on top of the positions saving (§4). If the measurement note of
   fn-196.3 moves lowering to Scala, the realization layer leaves the IR and the core alone remains:
   about 32 messages, Quint-shaped (§5). That is the larger trade-off and is not recommended on its
   own; it falls out of a decision fn-196 already schedules.
5. **Do not adopt Quint's IR as the core.** The 10-13 reasons weaken (no Go evaluator, no builtin
   count to worry about) but the decisive one strengthens: Quint's IR JSON is the `quint` CLI's
   internal format, unversioned, and the plugin would pin to a Quint release. Emit a
   **Quint-shaped** core instead: definitions, a `name`/`app`/`lambda`/`let` spine, one builtin
   table, metadata beside the tree. That is where fn-131, fn-189 and fn-192 already point.

---

## 1. Breakdown of the IR today

### 1.1 The three layers

| Layer | Files | Messages | Enums | Fields | What it says | Quint analogue |
| --- | --- | ---: | ---: | ---: | --- | --- |
| root | `ir.proto`, `common.proto` | 2 | 0 | 18 | `Model` with 16 repeated sections; `Position` | `QuintModule`; source map |
| model core | `value.proto`, `expression.proto`, `machine.proto` | 48 | 4 | ~163 | finite types and values; functions as expression trees; actions, machines, channels, monitors, assumptions, holes, compositions | `typedef`, `def`, `var`, `action`, `assume`; builtins |
| claims | `claim.proto` | 12 | 6 | 36 | Property, Scenario, Query, Limits, Exploration, RunExpectation, Progress | none: invariants and temporal props are CLI flags, tests are `run` defs |
| live driver | `realization.proto`, `script.proto`, `operand.proto` | 59 | 11 | 210 | roles, learned values, evidence, correlation, controls, scripts, commands, operands, proto templates, API behaviour | none |

Counts are the 10-13 note's regex counts (±2 on fields) and still match the files.

### 1.2 Who reads what, today and after fn-196

Go, non-test lines per package, and whether the package survives fn-196 R5 ("the Go `check`
package and the IR evaluator are retired"; Testpilot, the Driver, the canary and the Quint/TLA
export stay):

| Package | Lines | After fn-196 | Reads |
| --- | ---: | --- | --- |
| `ir` | 2,187 | survives, shrinks (validation of forms nobody evaluates) | everything |
| `interp` | 2,093 | retires as evaluator; `export` and `lower` still import its value and machine types | core |
| `check` | 2,255 | retires | core, claims |
| `internal/engine`, `runtimecel` | 4,336 | engine retires; `runtimecel` (155 lines) stays | core; operands |
| `explore` | 819 | retires with Queries/Exploration (fn-195) | claims |
| `export` (Quint, ITF, agreement) | 3,961 | survives | core: `functions`, `machines` (`starts`, `ends`, `steps`), `types`, `actions`, `channels`, `compositions`, `holes`; claims: `properties`, `scenarios`, `queries`, `progress` |
| `lower` | 6,400 | survives, location open (fn-196 R4) | driver: `realizations`, `evidence`; claims: `queries`, `limits`, `expected_run`, `scenarios`; core: `starts`, `actions`, 3 `body` reads |
| `realization` (admission) | 2,420 | survives with lowering | driver |
| `conformance` | 2,043 | replay moves to Scala; assessment identities stay | driver, claims |
| `lint` | 2,677 | survives (fn-167 places gap judging in Go) | core, driver, claims |

Two facts matter for the cut:

- **The exporter is the only surviving reader of expression trees.** `lower` reads `body` three
  times (to spell a Scenario's start and classes), `conformance` five, `lint` eight. Everything
  else that walks an `Expr` is in `interp`, `check` and `engine`.
- **The lowering needs the interpreter today** (`lower` imports `interp` 17 times, `check` 15)
  because it lowers "one Query's witness": it searches the path. Under fn-195 a conformance test
  *is* a pinned path, replayed in Scala before lowering. The lowering's input becomes data (a path
  of classes with the states and outcomes along it) and needs no evaluator.

### 1.3 Fields with no surviving reader

From the grep (approximate; a `oneof` arm read through a type switch counts): fields read only
by `check`, `interp`, `internal` or `ir`, and by nothing that survives:

| Message | Fields | Why they exist |
| --- | --- | --- |
| `Action` | `on`, `creates`, `schemas`, `examples` | `on`/`creates`/`schemas` have **no reader at all** (10-13 note §3.4.5 agrees); `examples` are the check's Abstraction Claims, which fn-195 moves to the test file |
| `Assumption` | `fair`, `fair_refs` | fairness for the checker; fn-190 R13 moves fairness to a claim's Scenario (`only`/`fair`) |
| `Channel` | `capacity`, `order`, `lossy`, `duplicates` | channel semantics for the interpreter; the Quint exporter refuses channels (`export/quint.go:84`) |
| `ComposedActionRef` | all four selections | composed fairness, same as `Assumption` |
| `Machine` | `family`, `entity`, `unobservable`, `assumes` | IDs built by the engine; `unobservable` validated only; `assumes` names Assumptions |
| `Model` | `version`, `assumptions` | version is checked to be 0 and never written |
| `Progress` | `within`, `assumptions` | the bounded liveness check |
| `Property` | `origin` | cleared for identity, read by nothing (fn-131 R7 replaces it with `LawOrigin`) |
| `Query` | `total` | the review figure; fn-173 R3 retires totals everywhere |
| `Refinement` | `visible`, `visible_outcomes` | refinement check |
| `Scenario` | `start` | the checker's root; always the machine's start in every measured Model |
| `TypeRef` | `channel` | the inbox state field's type |

This is the evaluator's share of the schema. Each row is a decision, not an automatic deletion:
a row either leaves (its meaning lives in the Scala Model and checker only), or the exporter learns
to carry it to Quint (channels, fairness, progress bounds are expressible there). §4 takes a
position per row.

### 1.4 What the lifter builds

The framework lifter's message use (`model/framework`, counts of `ir.X` references): `Model` 407,
`Expr` 366, `TypeRef` 283, `Value` 179, `Type` 148, `Position` 111, `Param` 108, `Function` 104,
`Machine` 75, then a long tail. `Position` is the sixth most referenced message in the lifter;
fn-189 R5 (root-only positions) removes most of those sites. The realization messages
(`Realization` 16, `Command` 15, `Operand` 22, `ProtoValue` 22) are built reflectively from the
`realize` DSL (2,302 lines of Scala), which is why the driver schema mirrors the Scala vocabulary
one to one.

---

## 2. What the planned specs do to the IR

### 2.1 Per spec

Specs under `.flow/specs/`, all `open` unless marked. "Reader" is who the spec assumes reads the
result.

| Spec | Adds | Removes | Changes | Assumed reader |
| --- | --- | --- | --- | --- |
| fn-174 Model constructs its IR | `@liftedIr` carrier (compile-time) | — | IR built by the plugin, "construct then print"; declaration positions may be empty | Go checks as oracle (R5) |
| fn-189 plugin diet | — | expression positions (root `Function.position` only, R5) | helpers inlined as `Let` (R11): no helper `Function`, no `Call` to one; `requires` on roots only | "no change to Go consumers" |
| fn-190 plain Scala | synthesised `<machine>.State` record; `oneOf` starts as several `starts` | `Query` bounds (R14); `<field>_<fact>` text keys (R17); `Assumption` at the DSL | assignments lift to `copy`, reads to `field`; member facts as a tagged enum case | Go checks as oracle (L163), Scala IR evaluator (R7) |
| fn-191 sections | — | — | claims grouped by kind in the DSL; IR Properties unchanged | — |
| fn-192 builtins | `BuiltinCall {name, args}` | `Inbox`, `OP_CONTAINS`, `OP_CONCAT`; no `Collection` node | operators as one table | Go interpreter and Quint exporter |
| fn-193 plain compositions | — | — | IR work deferred to fn-190.12 | — |
| fn-194 parametric Models | later: "a Model plus its parameter values" | — | a parameter lifts as a literal; the IR stays instance-level (Quint's `instance` flattened) | — |
| fn-195 ModelSuite | — | from the Model: `queries`, `Query`, `witness`, `.live`, `.explore`, `Exploration`, capability live switch, `Realizes.runs` | Cases come only from `conformance` tests, replayed on the Model first | Scala checker |
| fn-196 Scala checker | — | Go `check`, IR evaluator; conformance replay moves to Scala | IR is "the export and lowering format"; lowering location decided by a measurement note | Quint/TLA export, lowering |
| fn-197 `owes`/`until` | — | — | "no IR node, no export": a model check only | Scala checker |
| fn-173 reachable exploration | `Set`/`Map`/`List`/checked `int64`; per-rule guards; `checkedBy`/`confirm` on a Query | Query totals everywhere | Behavior Fingerprint = IR structural hash over an allowlist (D2) | Go throughout ("Confirm Go only") |
| fn-131 learn from Quint | `Expr.id` + position side table; `TypeRef.str`; `doc`; `LawOrigin`; `Canonical(Model)` | `Action.on/creates/schemas/results`, `Property.transition`, `Channel.lossy`, maybe `Machine.unobservable`; `producer*` to lowering config | `Action` kind as one `oneof`; `Scenario.start` optional; per-field int ranges | Go reader checks levels (Part A) |
| fn-146 CEL | Case format 4.0 with canonical CEL | custom predicate walkers in Testpilot | Umpire operands lowered to CEL in Go; "no Umpire IR import of Testpilot schemas" | Go lowering (`runtimecel`) |
| fn-123 faults | fault kind and reason on `Action`, budgets, durability | crash rule functions (Go derives the crash row) | — | Go derives rows, `umpire-faults` command |
| fn-125 dynamic config | `Setting`, `setting(id)` expr, `Query.under`, `Realization.encodings`, `ApiBehavior` preconditions | `Realization.required_settings` | `Query.total` × valuations | Go tables per valuation; Quint `const` |
| fn-144 tags/domains | ~15 messages (`TagDefinition`, `Domain`, `Template`, `Operand.any/setting`, …) | — | — | new Go module `tools/umpire/values` |
| fn-167 `@gap` | `Model.gaps`, `Machine.derives` (inert) | `.lint.json` sidecar | — | Go `umpire-lint` |
| fn-162 Queries name realizations | `Realization.queries` (landed as field 21) | — | — | Go lowering, lint, conformance |
| fn-160 one IR per feature | — | — | `Model.source` lists roots; schema out of scope | Go lowering, lint, export |
| fn-150 composed liveness | progress on compositions with composed fairness | — | keeps explicit assumptions | Go `check`/`engine` |
| fn-152 situations | a Situation declaration with ID and fingerprint | — | — | model checking and lowering |
| fn-147 Durations | — | — | hints and operands lower to `google.protobuf.Duration` | Go lowering, Testpilot |
| fn-159, fn-161, fn-164, fn-165, fn-141, fn-149 | — | `Observation.read` (fn-159 R13) | names, bodies, hashes may change; "no IR schema change" | — |
| fn-116 (done) | — | — | spike: Model expressions do not become CEL | — |
| fn-120 (done) | `Construct.choice` | — | — | Go lint; Quint export keeps inert names |
| fn-145 (done) | — | — | nine files, bytes unchanged | — |

### 2.2 The net effect, if every spec lands as written

- **Core:** `BuiltinCall` in, `Inbox`/`Unary` partly out, positions mostly out, `Expr.id` + side
  table in, `str` in, `Set`/`Map` in, `Action` as a `oneof` kind, `Function` roots only, faults and
  settings in, `doc` in, gaps and derivations in. Net: about the same message count, far fewer
  bytes, and a core that is still typed per form.
- **Claims:** `Query`, `Limits`, `Exploration`, `Variation`, `Alternative`, `Query.total`,
  `Scenario.free` and the capability live switch lose their author (fn-190/fn-195/fn-173). Nothing
  in those specs says what replaces them *in the IR* for the Cases a `conformance` section produces.
  That gap is the one real design decision left (§4.2).
- **Driver:** `Operand` keeps its symbolic forms and is lowered to CEL in Go (fn-146 R5); `Setting`
  encodings and tag domains add about 20 messages (fn-125, fn-144, both deferred).

### 2.3 Where the specs disagree

These must be settled before a schema cut, or the cut happens twice:

1. **Who reads the IR.** fn-174 R5, fn-190 L163, fn-173 D4, fn-123 (crash rows, budgets,
   `umpire-faults`), fn-125 (tables per valuation), fn-131 Part A (a level checker in the Go reader),
   fn-150 and fn-167 (Go lint judges gaps) all assume Go reads and evaluates the IR. fn-196 deletes
   that reader. fn-197 already depends on fn-196.
2. **The IR evaluator.** fn-190 R7 adds a Scala evaluator over the IR; fn-196 retires it before it
   is written.
3. **Full tables.** fn-125 and fn-123 check "over every state of the catalog"; fn-173 drops
   catalogs.
4. **Totals and Limits.** fn-125 multiplies `Query.total`; fn-173 retires totals; fn-190 R14 removes
   the Model's bounds but fn-173 reuses `Limits` for the reach budget; fn-159 R11 centralises Bounds.
5. **Assumptions.** fn-123 R6 and fn-150 keep `Assumption`; fn-190 retires it and moves fairness to
   a claim's Scenario with no IR shape given.
6. **Operator typing.** fn-131 Boundaries: "the typed operator enums are not collapsed into one
   generic application node, because proto-typed operators suit a Go interpreter". fn-192 collapses
   the value operators, and fn-196 removes the Go interpreter the boundary was written for.
7. **Progress in the export.** fn-197 says no progress claim is exported; fn-173 R8 lists progress
   under export coverage; `Progress` is read by the exporter once today.
8. **Gaps.** fn-120 kept acceptances out of the IR on purpose ("would need another IR field");
   fn-167 adds `Model.gaps`.

The pattern: everything written before 2026-10-17 assumes Go evaluates; fn-196 is one day old. The
cheapest resolution is a one-line amendment to each of fn-173, fn-123, fn-125, fn-131, fn-150, fn-167
and fn-192: "the reader is the Scala checker; the IR carries this only if the exporter or the
lowering reads it".

---

## 3. Quint's IR, reduced to the habits worth taking

`ir-simplification.md` §1 and §3.9 hold the detail. The habits, restated against the new frame:

| Quint habit | What it buys Quint | Umpire status |
| --- | --- | --- |
| **7 expression kinds** (`name`, `bool`, `int`, `str`, `app`, `lambda`, `let`); everything else is a builtin in one table | one parser, one resolver, one type checker; back ends own meaning | fn-192 does this for value operators only; `Unary`/`Binary`/`If`/`FieldAccess`/`Copy`/`ListOf` stay as forms (§4.1 A2) |
| **Definitions, not sections**: `def`, `val`, `var`, `const`, `assume`, `typedef`, with a `qualifier` (`pureval`, `action`, `temporal`, `run`) | one list of declarations; a `run` is a def | Umpire has 16 sections on `Model`; after fn-195 a run is a test, not a def, so Umpire needs *fewer* kinds than Quint here, not more |
| **`id` on every node, metadata beside the tree** (source map, inferred types, effects, modes) | identity hashing ignores metadata by construction | fn-131 Parts B and C; fn-173 D2's structural hash depends on it |
| **`doc` on declarations** | provenance in the IR, no sidecar | fn-131 E |
| **Instances flattened by the compiler** (`const` + `instance`) | the checker sees one module | fn-194: "the lifter emits an instance as a Model plus its parameter values"; the IR stays instance-level |
| **Claims outside the IR** (`--invariant`, `--temporal`, `run` defs) | the IR is the model, not the test plan | fn-195 reaches the same split from the other side |
| **No realization layer** | — | Umpire's reason to exist; keep, but behind its own root (§4.3) |

What Quint lacks that Umpire must keep: finite types by construction, step records with outcomes
and facts (what a Case checks in history), channels with bounds (if exported), holes, named
choices (inert), evidence functions, and the driver layer.

---

## 4. The cut: no or minor trade-offs

Ranked within each layer. "Trade-off" is what is lost; "none" means the Scala Model and checker
keep the meaning and no surviving reader needed the field. Each item names the spec that owns it
where one exists, so that the cut is a set of amendments rather than a new spec.

### 4.1 Core (`value`, `expression`, `machine`)

| # | Change | Trade-off | Owner |
| --- | --- | --- | --- |
| A1 | **Positions: root functions and declarations only, plus a side table for the located nodes** (holes, unmatched matches, choice constructors). | Diagnostics on an inner expression report the root. fn-131 accepted this on 2026-10-11. | fn-189 R5, fn-131 C |
| A2 | **One `BuiltinCall` for every operator, including `Unary`, `Binary`, `If` and `FieldAccess`'s arithmetic cousins.** fn-192 stops at value operators because "`Binary`/`Unary` have no recognition cost in the lifter". With no Go interpreter the only places that know an operator are the lifter's table lookup and the exporter's map; two enums beside a table are two mechanisms for one thing. `Expr.kind` becomes: `literal`, `var`, `builtin`, `call`, `construct`, `copy`, `field`, `if`, `match`, `let`, `list`, `lambda`, `hole` (13, from 16; `If`, `field`, `construct`, `copy`, `match` stay structural because the exporter and TLA translation treat them specially and typed admission on them is cheap). | None for meaning. The proto loses enum exhaustiveness for operators; `model/builtins.json` and its two conformance tests are the replacement fn-192 already builds. | fn-192 R2 amendment |
| A3 | **One convention for "a function of the state".** `Machine.ends` is an inline `Lambda`; `holds`, `from`/`to`, `next`/`violated`, `map`/`visible` are function names. Make `ends` a named function and drop `Lambda` from `Expr`, or make all inline. Named wins: the exporter already refuses an `ends` that is not a one-parameter lambda of the state. | None. | fn-131 open question 7; none owns it |
| A4 | **`Action`: kind as a `oneof` (`actor`, `timer`, `internal`, `delivers`, `loses`); delete `on`, `creates`, `schemas`, `results`, `examples`.** `examples` go to the ModelSuite `examples` section. | None; nothing reads them. | fn-131 D, fn-195 |
| A5 | **Step record: `Step(outcome, state)` with facts and `because` defaulted; the step type a fixed IR name, not a Scala package path.** | None. Removes the `sed` in `check/Samples.scala`. | 10-13 note §3.4.3/.4; none owns it |
| A6 | **Derived facts leave:** `Property.transition` (arity of `holds`), `Channel.lossy` (a `loses` action exists), `Scenario.start` (unset = the machine's start). | None. | fn-131 D |
| A7 | **Fairness and assumptions: one encoding.** `Assumption.fair` and `fair_refs`/`ComposedActionRef` are two spellings of one selection, and `Scenario.keys` a third (joined strings). Under fn-190 R13 fairness is a property of a claim's Scenario. Keep one structured `ActionRef {machine or member, action, inputs?}` and use it in `Scenario`, fairness and `Sync`. `Assumption`, `Machine.assumes`, `Model.assumptions`, `Progress.assumptions` go if the Scala checker is the only consumer of fairness; the exporter can write `weakFair` from the claim's Scenario directly. | None if fairness is per claim (fn-190). fn-150 and fn-123 R6 want named assumptions a result can cite: that is a Result field of the checker, not an IR field. | fn-190 R13/R17, fn-131 |
| A8 | **Channels: decide once whether the exporter carries them.** Today only the retiring interpreter reads `capacity`, `order`, `duplicates` and `TypeRef.channel`; the Quint exporter refuses channels. Either (a) the exporter writes a channel as a bounded `List`/`Set` var with the bound as a `const` and the send/deliver/lose actions as Quint actions, and the `Channel` declaration stays (one message, six fields); or (b) channels are not exported and the declaration shrinks to `{id, name, message}`. (a) is the right answer for the ReliableBroadcast sample and costs one exporter case. | (b) loses Quint agreement on channel Models; (a) costs an exporter case. | fn-192 touches `Inbox`; nothing owns the declaration |
| A9 | **`Machine.family`/`entity`, `Composition.family`:** keep `family` only as the Definition ID prefix the Cases need (fn-160 R15 derives Case ids from the package root); `entity` is a Temporal kit notion (fn-169 R6) and leaves the framework IR. `unobservable`: lint reads it or it goes (fn-131 R5's choice; nothing reads it today). | None. | fn-131 D, fn-169 |
| A10 | **`Refinement.visible`/`visible_outcomes`:** the refinement check is the Scala checker's; the exporter writes `map` only. Keep `product` and `map`, drop the two visibility functions from the IR. | Quint agreement never checked visibility. | none |
| A11 | **`TypeRef.int` and `TypeRef.list`** are admitted as inputs and refused later (fn-131). With fn-173's `oneOf` domains the refusal is the lifter's, at the Scala line. Nothing to add; drop fn-131 R1's refusal. | None. | fn-131 R1 amendment |
| A12 | **`Model.version`:** set it, once, at the cut (`version = 1`); readers refuse 0. | None. | 10-13 note §3.6 |

Estimated core after A1-A12: `value` 11 messages (+`str`, +`Set`/`Map` arms), `expression` 14
(from 21: `Unary`, `Binary`, `Inbox`, `Lambda`, `NamedExpr` folded, `Alternatives` kept), `machine`
10 (from 15: `Example`, `Assumption`, `ComposedActionRef`, `SyncMove` folded into one `ActionRef`,
`Refinement` smaller). About 35 messages, 4 enums gone, roughly 60 fields fewer.

### 4.2 Claims (`claim.proto`): from 12 messages to 4, and a `Conformance` the tests own

After fn-190/fn-195 the Model declares `invariants`, `postconditions`, `monitors` and `progress`,
and nothing else. The IR's claims layer should say exactly that:

| # | Change | Trade-off | Owner |
| --- | --- | --- | --- |
| B1 | **Keep `Property` (`machine`, `name`, `when`, `holds`), `Monitor`, `Progress` (`from`, `to`, `within`), `ClaimRef`.** `Property.origin` goes (fn-131 R7's `LawOrigin` replaces it, as inert metadata). `Progress.assumptions` goes with A7. | None. | fn-131 |
| B2 | **`Query`, `Limits`, `Exploration`, `Variation`, `Alternative`, `Query.total`, `Query.through`, `Scenario.free` leave the Model IR.** They have no author after fn-195 (`queries` only for witnesses and realizations; fn-190 says Kettle has no `queries` section at all). | None at the Model. The *tests* need a shape for what they lower: see B3. | fn-195 R5, fn-173 R3 |
| B3 | **A `Conformance` message, owned by the ModelSuite, not the Model:** `{machine, name, realization, start?, path: repeated ActionRef, expected: RunExpectation}`. It is what a `conformance` test produces after the Scala checker has replayed the path and attached the states and outcomes along it. It is the lowering's whole input besides the realization. It can live in `claim.proto` renamed `conformance.proto`, or as a sibling root the suite writes next to the IR file. `RunExpectation` and `MonitorExpectation` move with it. | None: this is the shape fn-195 already describes in prose. It also removes the lowering's need for the interpreter (§1.2). | fn-195, fn-196.3 (lowering location) |
| B4 | **`Scenario` becomes the pinned path of B3**: `actions` as structured `ActionRef`s, `keys` gone, `free` gone, `start` optional. | None; `keys` is derived three times today. | fn-190 R17, fn-131 |

Claims after B1-B4: `Property`, `Monitor`, `Progress`, `ClaimRef`, plus `Conformance`,
`RunExpectation`, `MonitorExpectation` in the test-owned file. Four messages in the Model's IR.

### 4.3 Driver (`realization`, `script`, `operand`): CEL and references

The realization layer is Umpire's reason to exist and stays. Three cuts are cheap; one is the
fork fn-196.3 decides.

| # | Change | Trade-off | Owner |
| --- | --- | --- | --- |
| C1 | **`Operand` becomes CEL text.** Today `Operand` is a 12-arm `oneof` with seven messages (`PathOf`, `Present`, `Equal`, `All`, `Greater`, `Not`, `Name`), and `runtimecel/operand.go` (155 lines) translates each arm to a CEL node under fn-146. The symbolic references (`environment`, `learned_value`, `run`, `projected`, `named`) are five CEL identifiers or functions in the restricted environment fn-146.2 already pins. The `realize` DSL keeps its typed operand builders and prints CEL (a 40-line printer); the IR carries `string cel = 1; Position position = 2;`; admission parses and checks it against the pinned environment with today's refusal texts. `Proto`/`ProtoValue` templates stay as fn-146's Boundaries require. | Minor. A refusal names the command's position and a CEL offset instead of the operand's own position (fn-146 R5 asks for "the Umpire source position": the command's is one). Operand typing moves from proto shape to CEL check; the Scala DSL still types what an author can write. fn-146 says "no Umpire IR import of Testpilot schemas": CEL text imports nothing. | fn-146 R5 amendment |
| C2 | **Kit data by reference.** `ApiBehavior`, `rejection_codes`, `correlation` and shared scripts are repeated in every realization (39% of the standalone activity file). One `Kit` per Model, `Realization.kit` by id, `Effective(r)` accessors in `lower`. | None; Case bytes unchanged if `lower` resolves before writing. | 10-13 note §3.3b |
| C3 | **`producer`/`producer_version` to lowering configuration** (fn-131 F); `Realization.queries` becomes `Conformance.realization` (B3), so the field lands and leaves within the same cut. | None. | fn-131 R8, fn-162 R8 |
| C4 | **Settlements as scripts.** `ActivityExternalSettlement` and `ActivityResetSettlement` are two 10-field messages each hand-lowered (`lower/activity_external.go`, `activity_reset.go`). They are Temporal kit shapes in a framework schema (fn-169 R6: the IR is Temporal-free). Spell each as a script with `await_*` and `attempt_*` commands, which the schema already has, and delete the two messages. | Moderate: two lowering files are rewritten as script lowering. Deferred to the sweep with the rest of fn-169. | fn-169 |
| C5 | **The fork (fn-196.3): where the lowering runs.** If it stays in Go, the driver layer stays in the IR as above: about 52 messages. If it moves to Scala, the realization is Scala data the lowering reads directly, the Testpilot Case proto is the wire, and the driver layer leaves the Umpire IR entirely: 120 messages become the core's ~35 plus `Conformance`. | Not minor: `lower` (6,400 lines) and `realization` admission (2,420) are rewritten in Scala, and the canary's Case reading must not change. The payoff is one schema, one language for everything that is not Testpilot. This note does not recommend it on its own; fn-196.3's measurement note is the place to decide it, with this figure in hand. | fn-196 R4 |

### 4.4 Metadata and identity

| # | Change | Trade-off | Owner |
| --- | --- | --- | --- |
| D1 | **One `Canonical(Model)` and one inert mark.** Positions, `doc`, `LawOrigin`, gaps, derivations, `source` and choice names are inert; everything else is in the structural hash (fn-173 D2's allowlist is the same list from the other side). Three hand-written strippers go. | None. | fn-131 B, fn-173 §5.5 |
| D2 | **`doc` on declarations** (fn-131 E): Quint has it, costs bytes only, and retires the laws sidecar. Take it. | Bytes. | fn-131 R7 |
| D3 | **Split the root**: `Model {version, core: Core, claims: Claims, meta: Annotations}` and realizations under their own root message (or file). The exporter reads `Core` and `Claims`; the lowering reads `Realization` and `Conformance`; the hash reads `Core`. | All proto hashes change once; the IR has no compatibility promise (`ir.proto` header). | 10-13 note §3.6 |

### 4.5 Estimated schema after the cut (lowering kept in Go)

| File | Today | After | Note |
| --- | ---: | ---: | --- |
| `ir` + `common` | 2 | 3 | `Model`, `Core`, `Annotations`; `Position` |
| `value` | 11 | 12 | +`SetValue`/`MapValue` (fn-173), `str` arm |
| `expression` | 21 | 14 | `BuiltinCall` in; `Unary`, `Binary`, `Inbox`, `Lambda`, `NamedExpr`, `Alternatives`? out |
| `machine` | 15 | 10 | `Example`, `Assumption`, `ComposedActionRef`, `SyncMove` out; `ActionRef` in |
| `claim` | 12 | 4 (+3 test-owned) | `Property`, `Monitor`, `Progress`, `ClaimRef`; `Conformance`, `RunExpectation`, `MonitorExpectation` beside the tests |
| `operand` | 14 | 8 | `Operand` family (7) → `Cel`; `Proto` family (7) stays |
| `script` | 21 | 21 | unchanged |
| `realization` | 24 | 23 | `Kit` in; two settlements out (C4, later); `RequiredSetting` to fn-125's shape |
| **total** | **120** | **~95 (~83 without C4's two and the three test-owned)** | enums 21 → ~14 |

Bytes: positions (A1) are 18-42% of compact files; defaulted step fields, inherited starts and
kit references (A5, A6, C2) another 5-40% depending on how realization-heavy the file is. A Model
file after the cut is roughly a third of today's, and a realization-heavy one roughly a quarter.

---

## 5. Two directions not recommended, and why

**5.1 Adopt Quint's IR as the core.** The 10-13 reasons were: relations versus step functions,
unbounded types, derived tables in the output, no claims/driver counterpart, 93 builtins to
evaluate. With fn-196 the last two no longer apply and the first weakens (nothing in Go needs to
evaluate the encoding). What remains is decisive on its own: Quint's IR JSON is the `quint` CLI's
internal representation, has no version field and no stability promise, and changes between
releases (the 10-13 note measured 0.33.0). The plugin would pin to a Quint release and every
Quint upgrade would be an IR migration. Keep Quint as the export target and take its shape.

**5.2 Lower channels, options and steps to records and functions in the lifter** to shrink the
core further. `bounded-collections.md` §8.1's reasons (junk states, key spellings, Quint seeing
records where it should see lists) still hold, and with `BuiltinCall` the per-feature cost of a
channel operator is one table row, not a message.

---

## 6. Order

The owner's phasing (2026-10-17) puts every IR change after the DSL phase, batched. Within that
batch:

1. **Settle §2.3 first**, as one-line amendments: the reader after fn-196 is the exporter, the
   lowering and lint. Decide A8 (channels in the export) and the lowering location (fn-196.3)
   before the cut, because they decide whether `Channel` bounds and the driver layer stay.
2. **One regeneration, `version = 1`:** A1-A7, A9-A12, B1-B4, C1-C3, D1-D3. Each is small; what
   makes them one task is that every proto hash changes once and the retained fixtures under
   `export/testdata` are regenerated once (fn-192 already schedules that with fn-190.10).
3. **Lifter-only follow-ups**, no schema change: stable function names, the Step constant, default
   step fields (A5) where the regeneration did not already cover them.
4. **C4 and `entity`** with the Temporal sweep (fn-169).
5. **C5** only if fn-196.3's note chooses Scala for the lowering.

## 7. Open questions

1. **Channels in the Quint export (A8).** ReliableBroadcast is the one sample that needs it; the
   fixture `model/framework/samples/ReliableBroadcast.qnt.test` shows how Quint spells it by hand.
2. **Do `Monitor` and `Progress` stay in the IR for the export only?** fn-197 says progress is a
   model check only; fn-173 R8 exports it. If the export never writes `temporal` properties, both
   can leave with the claims layer and the IR's claims become `Property` alone.
3. **Lint's reader.** fn-167 places gap judging in Go lint over the IR; with the Scala checker as
   the oracle and `@gap` a Scala annotation, lint over Scala (the source checks already exist,
   SEMANTICS §20) may make Go lint's IR reading unnecessary. That decides whether `Model.gaps` and
   `Machine.derives` are IR at all.
4. **`family` after fn-189 R3.** The DSL drops `Family`; Case ids derive from the package root
   (fn-160 R15). Whether `Machine.family` survives as a field or as the prefix of a fully qualified
   `name` is a naming decision for fn-159.
