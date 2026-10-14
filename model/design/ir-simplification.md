# Why the Umpire IR is bigger than Quint's, and what to simplify

Status: research, 2026-10-13. No source was changed.

Paths are relative to the repository root unless they start with `/tmp` or `quint/`.

- `proto/` means `proto/internal/temporal/server/api/umpire/v1/`.
- `export/` means `model/framework/export/`.
- `quint/` means `quint/src/` of `github.com/quint-co/quint` at `main` (commit `b6db567bd5`).

**Method.**

- I read the nine `.proto` files, the lifter (`export/Lift.scala`, `Expressions.scala`,
  `Declarations.scala`, `Claims.scala`, `Capabilities.scala`, `Realizations.scala`, `Roles.scala`) and
  the Go readers (`tools/umpire/{ir,interp,check,explore,conformance,export,lower}`).
- I measured the six lifted samples in `/tmp/kettle-check/out/*.json`, lifted 2026-10-13 from the
  working tree. I also measured the four checked-in `model/temporal/features/**/generated/ir.json`.
- Quint 0.33.0 was in the local npx cache. I compiled the six `*.qnt.test` originals with
  `quint compile --target json` and generated their source maps with `quint parse --source-map`.
- The measuring scripts are in `/tmp/irsimp/` (`stats.py`, `qstats.py`, `savings.py`, `posred.py`,
  `steps.py`).
- `model/design/reachable-exploration.md` did not exist when I finished. I checked twice, and also
  under `.plans/`. §4.2 therefore works from the direction as other documents describe it.

---

## Summary

1. **The expression language is not where the size comes from.**
   - Umpire has 15 `Expr` forms with 13 binary, 2 unary and 3 channel operators
     (`proto/expression.proto:33-53`). Quint has 7 expression kinds and 93 builtin opcodes
     (`quint/ir/quintIr.ts:126-252`).
   - Kettle's functions lift to 124 expression nodes. Quint's Kettle module has 109.
   - Go evaluates every form in a 58-line switch (`tools/umpire/interp/eval.go:297-354`).
   - Go already translates the core to Quint form for form (`tools/umpire/export/quint.go:361-422`).
2. **Where the size does come from.** Four sources account for it.
   - **Breadth.** One `Model` message carries three IRs: a model core, a claims and test-plan layer,
     and a live-driver layer. Of the schema's 120 messages, 59 are realization, script and operand
     messages, and 27 are machine and claim messages (§1.1). Quint's IR has only the first layer.
   - **Per-node positions.**
     - Every `Expr` carries a `Position`. That is 55% of the compact bytes of every sample.
     - 62-76% of those positions repeat their parent's position.
     - Three different identity hashes then have to strip positions again, or fail to.
   - **Copies instead of references.** In the checked-in activity IR:
     - realizations repeat the kit's behavior, rejection table, correlation and often whole scripts
       (39% of the file);
     - 16 machines repeat one machine's body (5%);
     - 58 Scenarios carry inline copies of only 3 distinct starts; those equal to their machine's start are 4% of the file.
   - **Lifter-shaped encodings.**
     - Role tests are unrolled into list-membership literals: 167 occurrences, 25% of the activity
       functions' bytes.
     - Function names mix JVM module paths (`samples.Kettle$.Model$.effects$.heat`) with derived names
       (`model.rules.heat`).
     - Every step record spells out default `[]` facts and `""` explanations.
     - The step type's name is a Scala package path. A package move already needs a `sed` before Go
       reads the samples.
3. **Metadata is woven into the core.** Totals, Property origins, choice names, positions and `source`
   are fields of semantic messages. Every identity therefore has to strip them first:
   - `WithoutTotals`, `WithoutOrigins`, `WithoutChoiceNames` and `clearPositions` exist for this;
   - the explore digest forgets positions (`tools/umpire/explore/explore.go:108`).
4. **Behavior Fingerprints and Definition IDs are robust to every encoding change proposed here.** They
   hash evaluated tables and names, never the proto (`tools/umpire/internal/engine/canonical.go:18`,
   `:118-148`, `:205-307`; `table.go:16-21`). The proto-level hashes are not robust:
   - `goir.model/v1` model identity (`tools/umpire/conformance/conformance.go:162-173`);
   - explore candidate digests (`explore.go:108`);
   - explore proposals (`explore/proposal.go:33`).

   `.plans/UMPIRE_MODULES.md:563-567` calls the `goir.model/v1` identity persisted compatibility data.
5. **Recommendation.**
   - Do not adopt Quint's IR as the core (§3.9).
   - Take three ideas from it:
     - a side table for source positions;
     - one generic operator node instead of per-family operator messages;
     - references rather than copies.
   - Order the work: cheap lifter-only clean-ups, then position elision, then de-duplication, then
     a metadata side table, then a generic operator node landed before bounded collections add a
     fourth operator family (§5).

---

## 1. Side-by-side inventory

### 1.1 Schema size

| File | Messages | Enums | Fields | Enum values | Layer |
| --- | ---: | ---: | ---: | ---: | --- |
| `ir.proto` | 1 | 0 | 16 | 0 | root |
| `common.proto` | 1 | 0 | 2 | 0 | positions |
| `value.proto` | 11 | 0 | 30 | 0 | model core |
| `expression.proto` | 21 | 3 | 49 | 20 | model core |
| `machine.proto` | 15 | 1 | ~84 | 3 | model core and fairness |
| `claim.proto` | 12 | 6 | 36 | 28 | claims and test plan |
| `realization.proto` | 24 | 8 | 100 | 35 | live driver |
| `script.proto` | 21 | 3 | 67 | 9 | live driver |
| `operand.proto` | 14 | 0 | 43 | 0 | live driver |
| **total** | **120** | **21** | **~427** | **95** | |

The counts come from a regex over the files with comments removed, so the field counts are ±2.

**Quint's IR** has two files.

- `quint/ir/quintIr.ts` (436 lines) defines:
  - 7 expression kinds: `name`, `bool`, `int`, `str`, `app`, `lambda`, `let` (`:252`);
  - 5 definition kinds: `def`, `const`, `var`, `assume`, `typedef` (`:392`);
  - 3 module-level kinds: `import`, `export`, `instance` (`:374`);
  - `QuintModule` (`:379`);
  - 8 operator qualifiers (`:72`);
  - 93 builtin opcodes (`:126-220`).
- `quint/ir/quintTypes.ts` (210 lines) defines 13 type kinds plus 4 row kinds (`:121-153`).

### 1.2 Top-level sections

| Umpire `Model` (`proto/ir.proto:26-46`) | Quint `quint compile` output (`quint/cliCommands.ts:81-103`) |
| --- | --- |
| `source`, `version` | `stage`, `main`, `errors`, `warnings` |
| `types` | `typedef` declarations in `modules[].declarations` |
| `functions` | `def` declarations (`pureval`/`puredef`/`val`/`def`) |
| `actions`, `machines` (steps bound per action) | `var` declarations and `action` defs; `init`/`step` are ordinary defs, `q::init`/`q::step` added by `compile` (`cliCommands.ts:546-590`) |
| `channels`, `monitors`, `holes`, `compositions` | none. Written as ordinary definitions and instances |
| `assumptions` | `assume` declarations, and `weakFair`/`strongFair` opcodes |
| `properties`, `scenarios`, `queries`, `progress` | none in the IR: invariants and temporal properties are CLI flags folded into `q::inv` / `q::temporalProps` (`cliCommands.ts:579-590`); tests are `run` defs |
| `realizations` | none |
| positions on every node | `id` on every node; positions in a separate source map keyed by id (`*.map.json`) |
| no lookup tables | `table` (name resolution), `types`, `effects`, `modes`: derived analyses, written beside the modules |

### 1.3 Expression forms

| Umpire `Expr.kind` (`expression.proto:35-52`) | Quint |
| --- | --- |
| `literal` (`Value`: bool, int, enum, record, list, text) | `bool`, `int`, `str`. Records, variants and lists are `app` of `Rec`/`variant`/`List` |
| `var` | `name` |
| `field` | `app field(r, "f")` |
| `call` | `app <userOp>(...)` |
| `construct` (record or enum case, optional `choice`) | `app Rec("f1", v1, ...)` / `app variant("C", v)` |
| `copy` | `app with(r, "f", v)` |
| `unary` (2 ops), `binary` (13 ops) | `app not/iuminus/eq/and/iadd/...` |
| `if` | `app ite` |
| `match` with 5 pattern kinds | `app matchVariant(x, "A", lambda, "B", lambda, ...)` |
| `let` | `let` (holds a full `QuintOpDef`) |
| `list` | `app List(...)` |
| `lambda` | `lambda` (with qualifier) |
| `hole` | none |
| `inbox` (3 ops) | none. Channels are ordinary sets or lists |

Counts of expression nodes and operators in the six samples:

| Sample | Umpire Expr nodes | Umpire forms used | Quint nodes | Quint `app` opcodes used |
| --- | ---: | --- | ---: | ---: |
| Kettle | 124 | 11 | 109 | 14 |
| Bank | 296 | 12 | 387 | 47 |
| TwoPhaseCommit | 501 | 12 | 267 | 21 |
| TwoLayeredCache | 229 | 12 | 289 | 39 |
| ReliableBroadcast | 521 | 14 | 493 | 40 |
| Voting | 570 | 13 | 415 | 44 |

Quint's opcode counts include user operators, which are also `app` nodes. The two languages express
the same models in expression trees of about the same size. Umpire runs larger where its samples
hand-encode maps and sets as records and `match` ladders (TwoPhaseCommit, Voting).
`model/design/samples-vs-quint.md` finding 1 covers that, and `bounded-collections.md` addresses it.

### 1.4 Kettle in both IRs (abridged)

**Umpire.** This is one step, `heat`, as written to `Kettle.json`. The pretty-printed rule function is
107 lines, and every node carries `position`. Shown compact, without positions:

```json
{"name":"model.rules.heat","params":[{"name":"s","type":{"named":"samples.Kettle.Snapshot"}}],
 "body":{"if":{"condition":{"binary":{"op":"OP_AND",
     "left":{"field":{"base":{"var":"s"},"field":"heatingOn"}},
     "right":{"binary":{"op":"OP_LT","left":{"call":{"function":"samples.Kettle$.temperature","args":[{"var":"s"}]}},
                                   "right":{"literal":{"int":"100"}}}}}},
   "then":{"call":{"function":"samples.Kettle$.Model$.effects$.heat","args":[{"var":"s"}]}},
   "else":{"list":{}}}}}
{"name":"samples.Kettle$.Model$.effects$.heat","params":[{"name":"state","type":{"named":"samples.Kettle.Snapshot"}}],
 "body":{"list":{"items":[{"construct":{"type":"framework.dsl.Step","args":[
   {"literal":{"enum":{"type":"samples.Kettle.Outcome","case":"accepted"}}},
   {"copy":{"base":{"var":"state"},"updates":[
      {"name":"reading","value":{"binary":{"op":"OP_ADD","left":{"field":{"base":{"var":"state"},"field":"reading"}},"right":{"literal":{"int":"1"}}}}},
      {"name":"heatingOn","value":{"literal":{"bool":true}}},
      {"name":"beeping","value":{"literal":{"bool":false}}}]}},
   {"list":{}},
   {"literal":{"text":""}}]}}]}}}
```

The rest of the file, abridged:

- the action `{"id":"samples.Kettle.heater.heat","name":"heat","actor":"system","internal":true}`;
- one machine binding four actions to four `model.rules.*` functions, with `starts` and `ends` as
  inline expressions;
- 4 Properties, each naming a `model.property.*` function;
- 2 Scenarios, each with a copied start expression;
- 5 Queries, each with `limits` and a lifter-computed `total`.

**Quint** (`quint compile`, `kettle` module). There is no position: `id` keys a separate source map.

```json
{"id":93,"kind":"def","name":"heat","qualifier":"action","expr":{"id":92,"kind":"app","opcode":"actionAll","args":[
  {"id":77,"kind":"name","name":"heatingOn"},
  {"id":80,"kind":"app","opcode":"ilt","args":[{"id":78,"kind":"name","name":"temperature"},{"id":79,"kind":"int","value":100}]},
  {"id":85,"kind":"app","opcode":"assign","args":[{"id":84,"kind":"name","name":"temperature"},
     {"id":83,"kind":"app","opcode":"iadd","args":[{"id":81,"kind":"name","name":"temperature"},{"id":82,"kind":"int","value":1}]}]},
  {"id":88,"kind":"app","opcode":"assign","args":[{"id":87,"kind":"name","name":"heatingOn"},{"id":86,"kind":"bool","value":true}]},
  {"id":91,"kind":"app","opcode":"assign","args":[{"id":90,"kind":"name","name":"beeping"},{"id":89,"kind":"bool","value":false}]}]}}
{"id":111,"kind":"def","name":"step","qualifier":"action","expr":{"id":110,"kind":"app","opcode":"actionAny","args":[
  {"id":106,"kind":"name","name":"pressButton"},{"id":107,"kind":"name","name":"heat"}, ...]}}
```

The real differences, setting positions aside:

- **Umpire splits guard from effect** into two functions, and returns a list of step records.
  Quint conjoins guard and assignments in one action and lets `actionAny` choose.
  - Umpire's split is deliberate: SEMANTICS §6 says "The IR binds each action to one step function"
    (`model/SEMANTICS.md:358-363`).
  - The list of records is what makes outcomes, facts and named choices first-class.
- **Umpire names an outcome type and a step record.** Quint has neither.
- **Umpire types every parameter**, and the types are finite. Quint's type annotations are
  optional (`quintIr.ts:27-29`), and its `int` is unbounded.

### 1.5 Sizes of the six samples

All figures are bytes.

| Sample | Umpire as written (pretty) | Umpire compact | Umpire compact, no positions | Quint `modules` compact | Quint source map | Quint full `compile` output |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Kettle | 39,213 | 19,915 | 8,941 | 6,354 | 9,943 | 56,098 |
| Bank | 93,328 | 39,715 | 17,449 | 26,284 | 61,716 | 521,869 |
| TwoPhaseCommit | 167,503 | 82,491 | 37,078 | 24,857 | 28,507 | 473,357 |
| TwoLayeredCache | 80,641 | 39,593 | 17,970 | 20,760 | 30,781 | 205,197 |
| ReliableBroadcast | 186,052 | 88,032 | 39,613 | 34,801 | 47,800 | 363,030 |
| Voting | 174,313 | 80,230 | 35,762 | 24,159 | 41,324 | 298,838 |

Caveats on the comparison:

- Umpire's file also carries what Quint keeps outside the IR: Properties, Scenarios, Queries,
  totals and progress.
- Quint's full output is dominated by the derived `table`, `types` and `effects` maps.
- Bank's Quint is the test module `bankTest`, flattened with what it imports.
- TwoPhaseCommit's Quint is the 3-RM instance.

Without positions, Umpire's core is within 1.0-1.5× Quint's module. The 2× from pretty-printing and
the 2.2× from positions are encoding choices, not content.

The checked-in feature IR for comparison:

| File | Compact | Positions | Functions | Realizations |
| --- | ---: | ---: | ---: | ---: |
| `activity_mini/generated/ir.json` | 270,319 | 55% | 187 KB | none |
| `nexus/standalone/generated/ir.json` | 134,061 | 53% | 99 KB | 15 KB |
| `nexus/workflow/generated/ir.json` | 264,531 | 49% | 153 KB | 35 KB |
| `activity/standalone/generated/ir.json` | 2,375,228 (4.77 MB pretty) | 48% | 661 KB | 1,369 KB |

---

## 2. Where the extra complexity comes from

Each Umpire concept that Quint lacks, with the classes from the brief:

- **E**: essential to Umpire's purpose.
- **L**: an artifact of the lifter.
- **R**: redundant, because the reader can derive it.
- **H**: historical, kept for compatibility.

The "Read by" column relies on a reader map I took through the code (`tools/umpire/**`, non-test
files).

### 2.1 Model core

| Concept | Proto | Written by | Read by | Class |
| --- | --- | --- | --- | --- |
| Finite types, integer ranges | `value.proto:15-63` | `export/Types.scala` | `interp/machine.go:19-75` (catalogs are derived, not written) | E |
| Step record `{outcome,state,facts,because}` | a `Construct` of type `framework.dsl.Step` | `Expressions.scala:563-571`, name at `Context.scala:247` | `interp/eval.go:122-126`, `machine.go:608-627`; the type is spelled `framework.Step` there | E for outcomes and facts. **L** for the name: `model/framework/check/Samples.scala:172` rewrites `"framework.dsl.Step"` to `"framework.Step"` before Go reads the samples |
| Default facts `[]` and `because` `""` on every step | args 3-4 of every step `Construct` | `Expressions.scala:563-571` fills the defaults | `machine.go:621` requires 4 fields | **L/R**. Kettle 4/4 steps, ReliableBroadcast 21/21, activity 81/112 spell the default explanation |
| Rules lowered to `match input { c => if g then e(s) else [] }` | ordinary `Function`s named `<machine>.rules.<action>` | `Declarations.scala` (`lowered`, ~:632), SEMANTICS §6 | `interp` evaluates them like any function | E by design (SEMANTICS:358-363). The cost is a second function per action: 46 `if … else []` in activity |
| Role tests unrolled to `x OP_CONTAINS [c1,…,cn]` | `Binary` + `ListOf` of enum literals | `export/Roles.scala:168-170` | `eval.go` | **L**. SEMANTICS:103 says "the IR has no role construct". 167 occurrences, 379 items and 162 KB, which is 25% of activity's function bytes |
| Function names | `Function.name` | `Expressions.scala:381-420` | lookup only; not an ID or fingerprint input | **L**. Two schemes in one file: Voting has 26 of 37 names with JVM `$` and 10 derived; activity has 82 of 192 with `$` |
| `Construct.choice` | `expression.proto:75` | `Expressions.scala:767-850` | inert for keys and equality (`eval.go:40-43`, `engine/table.go:23-33`); stripped for identity (`ir/load.go:88-105`) | E as metadata (SEMANTICS:330-346). It sits inside a semantic node |
| `Function.requires` | `expression.proto:24` | `Expressions.scala:423-431` | `interp` | E (rare: 3 in features) |
| `Expr.hole`, `Model.holes` | `expression.proto:50`, `machine.proto:180-185` | `Expressions.scala:614-616` | `eval.go:343-348`, lint | E |
| `Inbox`, `TypeRef.channel`, `Channel`, `Action.delivers/loses` | `expression.proto:182-197`, `value.proto:56`, `machine.proto:99-120` | `Expressions.scala:605-612`, `Declarations.scala` | `interp/channels.go:42-205`; the Quint export refuses channels (`export/quint.go:84`) | E. It is a per-feature operator family, the precedent `bounded-collections.md` §1.2 cites |
| `Action.actor/timer/internal` | `machine.proto:24-34` | `Declarations.scala:73-83` | lint only (`lint/kinds.go:97`, `lint/holes.go:513`), and actor in `lower/lower.go:920` | E for live; inert for the table |
| `Action.on`, `creates`, `schemas`, `results` | `machine.proto:28-31` | `Declarations.scala:39-83` (on, creates) | **no Go reader** | **H/L**. `schemas` appears in no checked-in IR; `.schema[T]` is retired (`Declarations.scala:88`) |
| `Action.examples` | `machine.proto:32,41-45` | `Declarations.scala` | `check/claims.go:248-256` | E |
| `Machine.family`, `entity` | `machine.proto:53-56` | `Context.scala:341` | IDs and fingerprints (`interp/machine.go:282`, `engine/canonical.go:66-72`) | E |
| `Machine.unobservable` | `machine.proto:67` | `Declarations.scala` | validated only (`ir/validate.go:699`) | E for conformance intent. Open question: no semantic Go consumer found |
| `Machine.ends` as an inline `Lambda`; `starts` inline; `holds`, `from`/`to`, `next`/`violated`/`after`, refinement `map`/`visible` as function names | `machine.proto:61-63`, `claim.proto:30`, `:210-211`, `machine.proto:130-143` | `Claims.scala`, `Declarations.scala` | interp | **L**. Two conventions for "a function of the state": inline lambda and named function |
| Derived machines (`restrict`, `rebind`, `extend`, `refining`, `assuming`, `unmonitored`) written as full copies | `Machine` | `Declarations.scala:1292-1420` | interp. The reader dedups them again: its behavior cache keys on `{state,outcome,fact type,steps}` (`interp/machine.go:326-333`) | **R**. 16 identical machine bodies in checked-in activity (5%). Provenance is kept lifter-side only (`Lift.scala:68-69`) |
| `Model.version` | `ir.proto:33-35` | never set | `ir/validate.go:27-30` (must be 0) | H. SEMANTICS does not define version 0 |
| `Model.source` | `ir.proto:28` | `Lift.scala:171` | error positions, Quint header; cleared for identity (`conformance.go:164`) but **kept** in explore digests | metadata in core |
| `Position` on every `Expr`, declaration and realization item | `common.proto:14-17`, 12,497 occurrences in activity | `Expressions.scala:32` (`expr(at)` always sets it) | diagnostics; `Query.position.file` becomes the Case's `SourceLocation.Path` (`lower/lower.go:839-841`) and so feeds the Case fingerprint | E for diagnostics. **L** for per-node granularity: 55% of bytes, and 62-76% repeat the parent |

### 2.2 Claims and test plan

| Concept | Proto | Written by | Read by | Class |
| --- | --- | --- | --- | --- |
| Property, Scenario, Query, Limits, Progress, Assumption, Monitor | `claim.proto`, `machine.proto:124-176` | `Claims.scala`, `Capabilities.scala` | `check` (`checking.go:303-320`, `claims.go`) | E. Quint keeps these outside its IR; Umpire needs them for Definition IDs and Query fingerprints (`canonical.go:205-307`) |
| `Query.total` | `claim.proto:86-92` | the author, or the lifter when the author wrote none (`Lift.scala:156-161`, `Capabilities.scala:1227-1308`) | recomputed and compared (`ir/totals.go:95-155`); stripped for identity (`totals.go:83-92`) | E when the author asserts it, as a review figure (SEMANTICS:604-607). **R** when the lifter computes it: the same count is implemented twice, in Scala `staticTotal` and in Go `total` |
| `Scenario.start` when it is the machine's start | `claim.proto:52` | `Claims.scala:901-948` (`declaredStart` copies `machine.starts.head`) | `ir/validate_keys.go:15` | **R**. The 58 activity Scenarios carry 3 distinct starts; copies equal to the machine's start are 2-8% of each file |
| `Scenario.keys` (composed class keys as joined strings) beside `Scenario.actions` | `claim.proto:54-56` | `Compositions.scala` | `validate_keys.go:33-44`, `check/claims.go:732`; lowering refuses it (`lower/lower.go:851`) | **R/H**. The composed-key spelling is derived twice in Go (`validate_keys.go:271-312`, `engine/compose.go:158-205`) and once in Scala |
| `Assumption.fair` beside `fair_refs` / `ComposedActionRef` | `machine.proto:148-176` | `Declarations.scala` | both (`check/claims.go:457`, `:1007`, `:1025-1062`) | **H**. Two encodings of one selection; `fairRefs` appears in no checked-in IR |
| `Property.origin` | `claim.proto:32-44` | `Capabilities.scala:451-455` | only to clear it (`ir/origins.go:11-17`) | metadata in core |
| `Exploration`, `RunExpectation` | `claim.proto:95-190` | `Claims.scala` | `ir/exploration.go`, `conformance`, `lower` | E for live |
| `reserved 2`, `reserved 3` in expectations | `claim.proto:169`, `:185` | n/a | n/a | H |

### 2.3 Live driver

| Concept | Proto | Written by | Read by | Class |
| --- | --- | --- | --- | --- |
| Realization, Script, Command, Operand, Proto, Evidence, Role, Correlation, Control, ServerStep, settlements | `realization.proto`, `script.proto`, `operand.proto` (59 messages) | **reflectively**: "each constructor becomes the IR message of its name and each argument the field of its parameter's name" (`Lift.scala:16-17`; `Realizations.scala:24-60`) | `realization` (admission), `lower`, `conformance`, `lint`, `internal/runtimecel`, `tests/testcore/testpilot/model_fixture.go` | E for live Cases. **L** in shape: the schema is the Scala vocabulary, so every DSL addition is a proto addition |
| `ApiBehavior`, `rejection_codes`, `Correlation` repeated in every realization | `realization.proto:49-55`; the comment says "The Temporal realization kit attaches the complete table to every realization" | kit | `lower/realization.go:171`, `lower.go:1205` | **R**. In checked-in activity, behavior, rejection codes and correlation have 1 distinct value across 15 realizations, scripts 9 and evidence 9. Repeats are 927 KB, 39% of the file |

### 2.4 Synthesis

Most of the "Umpire is more complicated" impression has five causes.

- **(a)** The `Model` message is three IRs with no seam. The core is close to Quint's size. The claims
  layer is necessary but separable. The driver layer is half the schema.
- **(b)** Per-node positions double the file.
- **(c)** Writer-side denormalization. The reader has to re-dedup (`interp/machine.go:326`) and
  re-derive (composed keys, totals).
- **(d)** Lifter conventions leak: JVM names, the Step type's package, role unrolling, default step
  fields, and two conventions for a function of the state.
- **(e)** Metadata sits in semantic messages, so each identity strips a different subset:
  - `conformance.go:162-173` strips choice names, totals, origins, `source` and positions;
  - `explore.go:108` strips choice names, totals and origins, but not positions or `source`;
  - `proposal.go:33` strips totals and origins.

Sugar is not leaking in the §21 sense: every form SEMANTICS:1274-1278 calls sugar is gone by IR time.
What leaks is Scala's naming and the DSL's data vocabulary.

---

## 3. Concrete simplifications

How to read the risk entries:

- "BF" means Behavior Fingerprints: target, scenario, property and query (`canonical.go`). They read
  evaluated tables and names, so an encoding change leaves them unchanged unless it changes a key, a
  name or a row.
- "IDs" means Definition IDs (`engine/table.go:16-21`), built from family, kind, owner and member.
- "Proto hashes" means `goir.model/v1`, explore digests and Case fingerprints.

### 3.1 Elide positions that equal the parent's (S)

Before:

```json
{"position":{"file":"model/framework/samples/Kettle.scala","line":83},
 "unary":{"op":"OP_NOT","operand":{"position":{"file":"model/framework/samples/Kettle.scala","line":83},
   "field":{"base":{"position":{"file":"model/framework/samples/Kettle.scala","line":83},"var":"after"},"field":"state"}}}}
```

After, with the rule "an `Expr` without `position` is at its parent's":

```json
{"position":{"file":"model/framework/samples/Kettle.scala","line":83},
 "unary":{"op":"OP_NOT","operand":{"field":{"base":{"var":"after"},"field":"state"}}}}
```

- **Gain.** The compact file shrinks by 34-42% for samples, 35% for activity_mini, 25% for nexus
  workflow and 18% for activity standalone. There is no schema change.
- **Consumers.**
  - Lifter: `Expressions.scala:32`, passing the parent's position down.
  - Go: one helper, "position of the nearest positioned ancestor", for `interp.ErrorAt`
    (`eval.go:164-168`), `ir/validate.go` reports, `lint/lint.go:268`, `export/quint.go:143` and
    `conformance/plan.go:82`. Every `x.GetPosition()` on an `Expr` must go through a scope that
    carries the current position.
  - Lowering, testpilot and engine: no change.
- **Risk.**
  - BF and IDs: none.
  - `goir.model/v1`: none, because it clears positions anyway.
  - Explore digests: they change once, since positions are in them. `HoleRow`/`Decision` positions
    must keep the same values, which the inherited position gives.
- **Cost: S.**

### 3.2 Move metadata into one side table (M)

Before: totals, origins, choice names, positions and `source` sit in semantic messages, and
identity code strips them one by one (`ir/totals.go:83-92`, `ir/origins.go:8-17`,
`ir/load.go:88-105`, `conformance.go:176-192`).

After, with the proto sketched:

```proto
message Model {
  // ... semantic sections only ...
  Annotations annotations = 17;   // nothing semantic reads it; identities hash Model without it
}
message Annotations {
  string source = 1;
  map<string, int64> query_totals = 2;            // by Query name
  map<string, PropertyOrigin> property_origins = 3;
  repeated SourceSpan spans = 4;                  // see below
}
message SourceSpan { string node = 1; Position position = 2; }  // node: a path like "functions/12/body/if/then"
```

`Construct.choice` stays where it is. It is read when lowering and explaining, so it is not pure
metadata. It is stripped for identity because "naming a result moves no identity"
(`conformance.go:157-161`).

- **Gain.**
  - One identity function, `proto.Marshal(model with annotations cleared)`, replaces three strippers
    and closes the explore positions gap.
  - Positions become a Quint-style source map. Quint keys it by node id (`quintIr.ts:15-21`); a
    stable tree path is the protobuf equivalent.
- **Consumers.**
  - Lifter: `Lift.scala:170-186` and every `pos(...)` site.
  - Go:
    - `ir` (load, totals, origins);
    - diagnostics in `interp`, `check`, `lint` and `export`, which need a path-to-position lookup.
      This is the costly part;
    - `conformance.modelIdentity`;
    - `explore`;
    - `lower` (`Query.position.file` becomes the Case `SourceLocation`).
- **Risk.**
  - BF and IDs: none.
  - `goir.model/v1` changes. `.plans/UMPIRE_MODULES.md:563-567` treats that string as compatibility
    data. One option keeps the old projection as `goir.model/v1` and adds `goir.model/v2`. Another
    accepts a one-time re-seal.
  - Explore digests change.
  - Case fingerprints do not change, if `SourceLocation.Path` is read from the side table.
- **Cost: M**, or **S** if only totals, origins and `source` move and positions stay as in 3.1.

### 3.3 Reference shared data instead of copying it (S to M)

Three independent parts.

**(a) A Scenario start that is the machine's start** (S). Before: `{"machine":"model","name":"anyRun",
"start":{…1.5 KB copy of machine.starts[0]…}}`. After: `start` is omitted, meaning the machine's
first start.

- Consumers:
  - lifter `Claims.scala:907`;
  - Go `validate_keys.go:15`, `engine` `ScenarioSemantic` (the start *key* is hashed, so it is
    unchanged), `explore` (which edits Scenarios) and `export`.
- Risk: BF unchanged, because the start key is evaluated. Proto hashes change.
- Gain: 2-8% per file.

**(b) Kit data once per Model** (S/M). Before: each `Realization` carries `behavior`,
`rejection_codes` and `correlation`. After:

```proto
message Model { ... repeated Kit kits = 18; }
message Kit { string id = 1; ApiBehavior behavior = 2; repeated RejectionCode rejection_codes = 3; Correlation correlation = 4; }
message Realization { ... string kit = 22; /* behavior, rejection_codes, correlation read from the kit when unset */ }
```

- Consumers:
  - lifter `Realizations.scala:1725+`;
  - Go `realization` admission (`validate_behavior.go:40-105`, `rejection_codes.go`), `lower`
    (`realization.go:171`, `lower.go:1205`), `conformance` and `lint/api.go`. Each needs a
    `Effective(r)` accessor.
- Risk: Case bytes are unchanged if `lower` resolves before writing. `inventory`/`accounting`
  (`lower.go:1029-1668`) must count kit fields as carried.
- Gain: up to 39% of the checked-in activity file. Scripts shared verbatim across realizations
  (7 copies of 125 KB) need a `script_refs` list as well.

**(c) Derived machines as a source and a delta** (M). Before: 16 full copies. After:
`Machine.derives = {source: "activitySystem", ops: [...]}` with only the changed steps.

- Not recommended yet. Every Go reader would have to apply the ops, and `interp` already recovers
  the sharing at runtime (`machine.go:326`).
- Open question: do the 16 copies survive the fn-160.6 move to facets? See §6.

### 3.4 Lifter-only clean-ups (S each, no schema change)

1. **One function per role.** Lift `phase.in[Closed]` as a call of
   `temporal.….Phase.role.Closed(phase)`, lifted once, instead of an inline literal list
   (`Roles.scala:168-170`).
   - Gain: up to 25% of function bytes.
   - Risk: BF and IDs none, since tables are identical. Proto hashes change.
2. **Stable function names.** Derive every function's name from its Definition ID path, as
   `<machine>.rules.*` already is, instead of the JVM `$` path.
   - Function names feed no ID or fingerprint (`engine/canonical.go`).
   - Open question: whether lint acceptances key on function names
     (`lint/lint.go:85` ignores positions; I did not trace names).
3. **Step type name.** Make `framework.Step` a constant of the IR, not a Scala package path
   (`Context.scala:247`). The DSL's move to `framework.dsl` then needs no `sed`
   (`model/framework/check/Samples.scala:172`).
4. **Omit default step fields.**
   - Allow a step `Construct` of 2 args (outcome, state), with facts `[]` and because `""` implied.
   - This is a small Go change in `interp/eval.go:497` and `machine.go:621`, and in
     `validate.go:778-834`.
   - Gain is small, but every step function reads as `Step(outcome, state)`.
5. **Stop writing `Action.on/creates/schemas/results`.** No Go code reads them. Then delete them from
   `machine.proto:28-31` and from `Declarations.scala:39-83`.
   - Open question: are they intended for a future reader (`.plans`)? I found none.

### 3.5 Derive what the reader computes

- **Lifter-computed totals.** `Lift.scala:156-161` writes `countedTotal` when the author wrote no
  total. Go recomputes it and compares (`ir/totals.go:95-155`), and `RequireTotals`
  (`totals.go:43`, used by `cmd/umpire-gen-cases`) refuses a Query without one.
  - Option A: keep the field, but treat a lifter-computed one as a display copy. That changes
    nothing.
  - Option B: stop the lifter computing it, and let Go fill it (`WithTotals`, `totals.go:63`) when
    writing reports. This removes `staticTotal`/`size`/`composedClasses` from Scala
    (`Capabilities.scala:1227-1308`).
  - The catch: SEMANTICS:604-607 makes the total a review figure *in the IR diff*. Under B, an
    author no longer sees the count in the checked-in file unless they assert it.
  - **Cost S. Payoff modest, but it grows with bounded collections** (§4.1), whose closed-form sizes
    would otherwise be implemented twice.
- **Composed keys.** `Scenario.keys` stores separator-joined composed class keys, which three code
  paths derive. Replace them with structured `ComposedActionRef`s, the form `fair_refs` already uses
  (`machine.proto:160-176`), and drop `Assumption.fair` in favour of `fair_refs` for compositions.
  - Cost M: `Compositions.scala`, `validate_keys.go`, `check/claims.go:732`, `lower` (which refuses
    keys today).
  - Risk: `ScenarioSemantic` hashes action keys (`canonical.go:205`), so the Go-derived keys must
    stay identical. They do, if they are derived from the refs.
- **Catalogs** are already derived by the reader (`interp/machine.go:19-75`). Nothing to do.

### 3.6 Split the core from claims and realizations (M)

Before: one flat `Model` with 16 repeated sections (`ir.proto:26-46`). After:

```proto
message Model {
  uint32 version = 1;
  Core core = 2;                    // types, functions, actions, machines, channels, monitors, holes, assumptions, compositions
  Claims claims = 3;                // properties, scenarios, queries, progress
  repeated Realization realizations = 4;
  repeated Kit kits = 5;            // §3.3b
  Annotations annotations = 6;      // §3.2
}
```

- **Gain.** Readability, mostly.
  - `interp`, `export/quint.go` and `internal/engine` (which already reads no proto) see only
    `Core`.
  - `lower`, `conformance` and `realization` see `Realization`.
  - A reader can hash `Core` alone for a "model behavior changed" identity.
- **Consumers.** Every `m.GetX()` call site. The change is mechanical, and `gopls` rename covers
  most of it. Lifter: `Lift.scala:170-186`.
- **Risk.** All proto hashes change, and `ProtoJSON` paths change for every checked-in file. BF and
  IDs are unchanged. This is the natural place to set `version = 1`
  (`ir/validate.go:27-30`). Both the IR header and README:153 call the IR "an internal wire with no
  compatibility promise".
- **Cost: M.** Most of the work is regenerating and re-sealing checked-in files.

### 3.7 One generic operator node (M)

Before: three operator families with their own messages and enums. These are `Unary`
(`expression.proto:89-97`), `Binary` (`:99-120`) and `Inbox` (`:182-197`).
`bounded-collections.md:365-387` proposes a fourth, `Collection`, with 21 ops plus `type`, `receiver`,
`args`, `body` and `over`.

After:

```proto
message Expr { oneof kind { ...; Apply apply = 17; } }   // replaces unary, binary, inbox (and collection)
message Apply {
  Op op = 1;                 // one enum: NOT, NEG, EQ, ..., CONCAT, CONTAINS, SEND, IS_EMPTY, IS_FULL, SET_OF, INCL, ...
  repeated Expr args = 2;
  TypeRef type = 3;          // where the op needs one (constructors, catalog quantifiers)
  string channel = 4;        // where the op names a channel
  Lambda body = 5;           // quantifiers, folds
}
```

- **What it is not.** It is not Quint's string `opcode`. Keeping a closed enum keeps admission
  exhaustive (SPEC ART-04: reject unknown behaviour-affecting fields).
- **Keep these as distinct forms.** `Construct`, `Copy`, `Field`, `Match`, `Let`, `If`, `Call`,
  `Lambda`, `Var`, `Literal`, `Hole` and `List`. Quint encodes records and matches as `app` with
  alternating string and value arguments (`Rec("f", v, ...)`, `matchVariant(x, "A", λ, ...)`;
  quintIr.ts `:130`, `:186`). That moves field names and case names into untyped argument
  positions, which Umpire's typed admission would have to re-parse. Quint's 7 forms are smaller
  because they are untyped, not because they are simpler.
- **Gain.**
  - One evaluator dispatch per op (`interp/eval.go:311-313`, `:349`, `channels.go:96-121`).
  - One typing table in `ir/validate.go`.
  - One translation table in `export/quint.go:443-484`.
  - Bounded collections add enum values, not a message.
- **Consumers.**
  - Lifter: `Expressions.scala:41`, `:225-237`, `:590-640`.
  - Go: `interp/eval.go`, `interp/channels.go`, `ir/validate.go`, `export/quint.go`, and
    `lint/kinds.go` where it inspects operators.
  - Lowering and engine are not affected.
- **Risk.** BF and IDs: none, because tables are identical. Proto hashes change.
- **Cost: M.** It is cheaper before `Collection` lands than after.

### 3.8 Move the remaining sugar into the lifter

This is already done: SEMANTICS:1274-1290 requires each sugar form to lift to the same IR as its core
spelling, and the IR has no rule, role, witness or claim-pattern construct. The remaining source
reductions (SEMANTICS:1308-1318, fn-141.8-.13) are lifter-internal and do not change the IR.

The opposite move would shrink the schema: making some core into library over a smaller core, such
as channels lowered to records and functions. `bounded-collections.md` §8.1 (`:773-802`) argues
against doing this for lists, and its reasons (junk states, key spellings, Quint seeing records)
apply equally to channels. **Not recommended.**

### 3.9 Reuse Quint's IR, or target it, as the core

- **For.**
  - Quint has a mature toolchain: the simulator, Apalache and TLC.
  - The Go Quint export already exists. It is 1,069 lines (`tools/umpire/export/quint.go`) plus
    replay and agreement code (`itf.go`, `agreement.go`, `composed.go`, 1,267 lines).
  - Quint keeps positions in a side table.
- **Against.**
  1. Quint's actions are relations: `assign` and `actionAll`/`actionAny`. Umpire's semantics are
     deterministic step functions returning ordered result lists with outcomes, facts and choice
     names (SEMANTICS §5-§7). Every table, Definition ID and Behavior Fingerprint is defined over
     those rows (`engine/canonical.go:118-148`). Encoding them in Quint takes conventions such as
     `Step` records, a `q::` naming scheme and finite-type annotations, which no Quint tool
     checks. The result is a Quint dialect, not Quint.
  2. Quint types are optional and unbounded (`quintIr.ts:27-29`; `int`, `Set`, `Map`). Umpire's
     admission depends on finite catalogs (`interp/machine.go:19-75`).
  3. Quint's IR stores derived tables (`table`, `types`, `effects`). These are 80-95% of the
     `compile` output (§1.5). That is the opposite of SEMANTICS:10 "the Go reader derives every
     table".
  4. The claims and driver layers, which hold most of Umpire's schema, have no Quint counterpart
     and would remain as they are.
  5. The Go interpreter would need to evaluate up to 93 builtins instead of today's 18.
- **Recommendation.** Keep Quint as the second-opinion target (README:13-14; UMPIRE_MODULES:64).
  Take its three good ideas instead:
  - a side table for source positions (§3.1, §3.2);
  - one application form for operators (§3.7);
  - no copies, because the Quint IR references definitions by name and id.
- **Cost of adoption: L, with a negative payoff.**

---

## 4. Interaction with the in-flight designs

### 4.1 `model/design/bounded-collections.md`

Its additions are all additive (`:333-391`):

- three `TypeRef` arms: `set`, `total`, `seq`;
- two `Value` arms: `SetValue`, `TotalValue`;
- one `Expr` arm, `Collection`, with 21 ops (`:365-387`).

Which simplifications it enables or makes urgent:

- **§3.7 (generic operator node) becomes urgent.** `Collection` would be the fourth operator
  family. Its message already looks like the generic `Apply` of §3.7: `op`, `type`, `receiver`,
  `args`, `body`, `over`. Landing `Apply` first, with `Collection`'s ops as enum values, means one
  node instead of four. The design's Phase 2 (IR and Go, `:1180-1267`) is the natural slot.
- **§3.5 (totals derived once) becomes more attractive.** The lifter's `size` must learn closed
  forms for `Set`/`Total`/`ListUpTo` (`:484-487`), duplicating Go. samples-vs-quint finding 13
  already shows the Scala count failing on channels.
- **§3.4.1 (one function per role) is compatible.** A role could later lift as a constant `Set`
  literal, but a function call is already as small. Open question 8 of the design (state-key
  spelling and `SetupKey`) is independent of this report.
- **Value-spelling commitments** (`:196-217`) are key spellings. They feed BF, so they are unrelated
  to the encoding changes here, which must not touch `Value.Key`.

### 4.2 Reachable exploration

The design document was not written when I finished (§Method). From
`model/design/samples-vs-quint.md` finding 4 (`:221-276`) and `.plans/UMPIRE_MODULES.md:553-555`,
the direction is to check over reachable rows rather than every catalog row, and to explore
candidates. For the IR, that means:

- **It needs no IR change** for reachable-row refinement. Finding 4 rates it as a Go/reader change,
  cost M.
- **It makes §3.2 urgent if exploration identities matter.** "Exploration Case IDs may vary because
  they hash the whole candidate IR" (UMPIRE_MODULES:553-555). The digest also includes positions
  and `source` (`explore/explore.go:108`). Any of §3.1-§3.7 therefore churns every exploration
  candidate. Three ways to avoid it:
  - hash a core projection, as `conformance.modelIdentity` does;
  - hash the Scenario plus the Target fingerprint;
  - land §3.2 before exploration identities are persisted anywhere.
- **It favours §3.3a.** Exploration edits Scenarios (`explore.go:95-104`), and copied starts make a
  candidate's IR larger for nothing.

Open question: the reachable-exploration design may introduce IR (for example, an `assuming`
invariant on refinements, finding 4's second fix). Re-check §5's ordering against it once it exists.

---

## 5. Ranking and phased plan

### 5.1 Ranked by payoff per cost

| Rank | Change | Payoff | Cost | Proto-hash churn | BF/ID churn |
| ---: | --- | --- | --- | --- | --- |
| 1 | §3.4.3 Step type a fixed IR constant | removes a `sed` and a class of drift | S | yes, samples only today | none |
| 2 | §3.1 Positions inherit from parent | 18-42% of bytes; much easier diffs | S | explore digests | none |
| 3 | §3.4.1 One function per role | up to 25% of function bytes; readable guards | S | yes | none |
| 4 | §3.4.2 Stable function names | readability; no `$` | S | yes | none (open question on lint keys) |
| 5 | §3.3a Default Scenario start | 2-8% | S | yes | none |
| 6 | §3.4.5 Drop unread `Action` fields | 4 fields, 1 lifter path | S | small | none |
| 7 | §3.7 Generic operator node, before `Collection` | avoids a 4th operator family; one typing table | M | yes | none |
| 8 | §3.3b Kit data and shared scripts by reference | up to 39% of realization-heavy files | S/M | yes; Case bytes unchanged | none |
| 9 | §3.2 Metadata side table | one identity function; closes the explore-positions gap | M | yes (`goir.model/v1`) | none |
| 10 | §3.6 Split core / claims / realizations, version 1 | comprehension; a core-only identity | M | all | none |
| 11 | §3.5 Totals derived once; structured composed refs | removes double implementations | S / M | yes | none, if keys are kept |
| 12 | §3.4.4 Omit default step fields | small | S | yes | none |
| – | §3.3c Derived machines as deltas | reader complexity rises | M | – | – |
| – | §3.9 Adopt Quint IR | negative | L | – | – |

### 5.2 Phases that keep the gate green

The gate requires each `generated/ir.json` to equal a fresh lift (README:173, :219-233). Lowered Cases
carry fingerprints in `manifest.json` and `*.case.json`.

**Phase 0: guard.**

- Add a test that lifts every feature and every sample, and compares BF, IDs and the lowered Case
  bytes before and after a change. The "Sealing a batch" equivalence projection (README:331-368)
  already leaves out positions and fingerprints.
- Fix the explore digest to hash a core projection (`explore.go:108`), so that later phases do not
  churn exploration identities.
- Cost: S.

**Phase 1: lifter-only changes (ranks 1, 3, 4, 5, 6).**

- For each change:
  - make it in Go first so both forms are accepted (an omitted Scenario start, the Step constant);
  - then switch the lifter;
  - then regenerate every `generated/ir.json` and the samples in one commit;
  - then check that Phase 0 reports no BF, ID or Case-byte change.
- `goir.model/v1` changes once, so re-seal per README:331-368.

**Phase 2: positions (rank 2).**

- Go first: an ancestor-position helper, with `GetPosition` call sites moved onto it.
- Then the lifter, then regeneration.
- Case `SourceLocation.Path` comes from `Query.position`, which is a declaration position and is
  unaffected.

**Phase 3: realization de-duplication (rank 8).**

- Go first: the `Effective(r)` accessors.
- Then the lifter writes kits.
- The inventory/accounting check in `lower` proves no field is lost.

**Phase 4: operators and collections (rank 7).**

- Land `Apply` with today's 18 ops, then convert `Unary`/`Binary`/`Inbox` in one regeneration.
- Bounded collections Phase 2 then adds enum values only.

**Phase 5: structure (ranks 9-11).**

- One schema change: the side table plus the split, `version = 1`.
- Go reads only version 1 after one regeneration. The IR has no compatibility promise (the
  `ir.proto` header, README:153).
- Keep `goir.model/v1` computable from the new shape (same projection), or mint `v2` and record the
  change in `.plans/UMPIRE_MODULES.md`.

---

## 6. Open questions

1. **Is the checked-in activity IR stale?**
   - `model/temporal/features/activity/standalone/generated/ir.json` (last commit `e94f6f8cf9`,
     2026-10-11) cites `model/temporal/features/activity/standalone/system/*.scala` 3,836 times.
     That directory no longer exists: it was renamed to `protocol/` in `dc29d0d3e3` (fn-160.6).
   - The 16 machine copies, and the realization repeats measured here, may differ in a fresh lift.
     I could not lift the feature: the local `model/build/model-scala.jar` holds only activity
     mini.
   - Does the gate currently pass?
2. **`Machine.unobservable`** is validated (`ir/validate.go:699`), but I found no semantic Go
   consumer. Is it meant for conformance?
3. **`Action.on/creates/schemas/results`** have no Go reader. Are they kept for a planned consumer?
4. **Lint acceptances.** Do accepted lint findings (`*.lint.json`) key on function names? If they do,
   §3.4.2 needs a migration of those files.
5. **`goir.model/v1`.** Is it persisted outside this repository, for example in stored Run
   assessments? That decides between a re-seal and minting `v2` (§3.2, §3.6).
6. **Reachable exploration.** Does the forthcoming design add IR? If it does, re-rank §3.2 and §3.6
   against it.
7. **Inline lambdas versus named functions.** `Machine.ends` and `starts` are inline expressions,
   while every other "function of the state" is a named `Function`. Should one convention win? It
   affects `export/quint.go:748-754`, which refuses an `ends` that is not an inline function of one
   state.
