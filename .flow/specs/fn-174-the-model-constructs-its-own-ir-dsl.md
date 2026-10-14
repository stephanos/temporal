# The Model constructs its own IR: DSL types build `ir` messages, capture points lift function bodies, the lifter retires

## Conversation Evidence

> user: "let's apply this idea ; the cons seem acceptable"
> user: "stalness had a solution (the clean one; I think using macro annotations - accepted)"
> user: "byte-for-byte matching is not required during the migraiton"
> user: "we can live without position and names in IR" … "well, retaining names would actually be nice"
> user: "and re-work the plan with how fn-141 Part B fits into this"
> user (2026-10-14): "attempt a prototype of the compiler pligin to assess viability"
> user (2026-10-14): "adjust the existing specs or create a new one to superseed the macro based one if needed"

## Goal & Context

Every DSL construct is described twice. When a Model runs, `action(this)` builds an `ActionDecl`,
`object rules` fills its buffer of `Rule` values and the gate's `IrFile.construct` instantiates every
reachable machine and binds its rules. Then the lifter's own JVM reads TASTy, recognises the same
calls by name in the trees, rebuilds the declarations, lifts the function bodies, assembles an
`ir.Model` (`model/framework/export/Lift.scala:170`) and prints it. The lifter is 12,541 lines
without tests, and fn-141 names the duplication as the cause.

fn-141 Part B removes the tree-side description of declarations by adding an exporter that walks the
constructed objects, and lifts function bodies by looking up a recorded span in TASTy. The spike
recorded in `model/design/scala-macros-export.md` (2026-10-13) showed that function bodies can be
lifted at compile time with the same `quotes.reflect` code the lifter uses, with identical output,
and §8 of that document describes the end state this spec builds: each DSL type constructs its own
`ir` message, the capture point yields an `ir.Function`, and lift is "construct, then print". No
exporter, no lifter JVM, no TASTy for IR.

**Owner decisions, 2026-10-13.** Adopt §8. Byte-for-byte IR identity is not required during the
migration: the oracle is the Go checks' answers. Declaration positions may be dropped from the IR;
names are retained. fn-172's toolchain fixes land first.

**Host decision, 2026-10-14 (amendment).** The first implementation attempt, with capture macros and
a SIP-63 macro annotation storing helper IR, stopped on two design conflicts: a macro annotation
stores its result after `pickler`, where no caller can see it, so the caller-specific `choose`
refusal could not stay a compile error (`.flow/tmp/fn174-boundary-conflict.md`); and constructed
values lose which declaration a caller spelled (`scripted(original)` and `scripted(alias)` construct
one value but are named `original` and `alias` today; `.flow/tmp/fn1742-name-conflict.md`). A
compiler-plugin prototype solved both on the pinned compiler with no `-experimental`, no macro
annotation and no `-Yretain-trees`: `model/design/compiler-plugin-prototype.md`. The compile-time
host of this spec is therefore a **Scala compiler plugin**, `model/framework/reify`, and the
plugin exclusion of the original capture is lifted. The goal, the requirements, the kind order and
the fn-141 supersession below are unchanged; only the mechanism paragraphs are.

**How fn-141 Part B fits.** This spec is Part B's successor: same obligations, different mechanism.

| fn-141 | Here |
| --- | --- |
| .1–.4 (lints out, unused spellings, sugar once): done | Prerequisites, unchanged. The generic sugar expansion (.3) moves into the reify module as is. |
| .5 capture points and lookup by span | **Superseded.** The capture point lifts the body it holds; no span is recorded and no lookup is written (.1 here). |
| .6 refusal ledger | **Stays in fn-141**, a prerequisite of .8 here (ledger rows decide each refusal's home). |
| .7 realization factories build complete values | **Stays in fn-141**, a prerequisite of .2 here. |
| .8–.13 one exporter per kind | **Superseded** by .2–.7 here, same kind order (realizations, signature, machines, compositions, claims, capabilities), with the conversion on the DSL type instead of in an exporter. |
| .14 close | **Superseded** by .8 here. fn-141 closes when its .6 and .7 are done and this spec's .8 has retired the lifter. |
| R6, R7 (names), R8 (functions captured), R9, R12, R13, R14, R16 | Carried, restated as R-IDs below. |
| R7 (positions), R8 (span lookup), R11 (byte identity per step) | Retired by the owner's decisions; R10 (two-run determinism) is kept. |
| fn-169.2 "follows fn-141.8" | Follows .2 here; it shrinks to layout conventions and API descriptors, since the kit's types convert themselves. |
| fn-173.5 "DSL and lifter: native collections" | Becomes "DSL and reify": the spike's §2.4 shows the recognition table ports unchanged. |

## Architecture & Data Models

**Modules.**

- `model/framework/reify` (new): the compiler plugin. It holds the body lifter moved from
  `model/framework/export` (`Expressions`, `Types`, `Expansion`, `Roles`, `Equality`, `Constants`,
  the name and position parts of `Context`; about 2,300 lines), still written against `Quotes`
  (obtained inside a phase from `QuotesImpl`, as the spike's plugin host did), with two seams:
  "definition of a symbol" (the typed tree in the current run, or a stored `@liftedIr` from
  TASTy) and "refuse" (`report.error` at the refused tree). It registers three standard
  `PluginPhase`s, scheduled after `posttyper` and before `pickler`:
  - **store**: for each def with the author's `@lifted` marker, lift the body and add
    `@liftedIr(<ir.Function>)` to the def's symbol. Before `pickler`, so the carrier is in TASTy
    and in the API hash zinc invalidates by.
  - **capture**: at each capture point (below), lift the argument's tree and rewrite the call so
    the constructed value holds its `ir.Function`. At each `choose`, apply today's helper-body
    rules (`namedCopy`, step cardinality) against the stored IR of the defs the alternatives call.
  - **names**: at each call, replace an argument that is a reference to a declaring `val` of a
    nameable DSL type, passed where that type is expected, by `ref.named("<spelling>")`;
    parameters pass through unchanged, so a helper carries its caller's spelling.
  It is compiled against `scala3-compiler` before the DSL and packaged as its own jar, like the
  lifter today; every Model build loads it with `-Xplugin`.
- `model/framework/dsl` depends on the generated IR classes (`build/ir-scalapb.jar`) and on the
  small `reify` runtime library (the marker and carrier annotations, the `Named` slot). The DSL
  already depends on the ScalaPB runtime; the Temporal Models already compile against
  `build/api-scalapb.jar`, so this is a new jar, not a new kind of dependency.
- Models compile with `-Xplugin:<reify jar>` (`model/project.scala`, `model/framework/dsl/project.scala`,
  the samples and the fixtures). `-experimental` and `-Yretain-trees` are not needed and not added.
- `model/framework/export` ends as the driver: discover roots (`IrFile.discover`), construct them
  (`IrFile.construct`), collect the messages they hold into one `ir.Model` per IR file, print with the
  existing printer. Under 100 lines plus the multi-file and `--ir` plumbing that stays.
- `model/framework/inspect` and `model/framework/lint` keep the TASTy inspector for the order,
  structure and marker lints. The inspector reads no IR.

**Function capture.** A capture point is a DSL method whose function parameter is a capture
(`Rules.on`, `effect`/`is` blocks, `holds`/`holdsAcross`, monitor `next`/`after`, `leadsTo`
`from`/`to`, `end`, refinement maps, evidence, capability function fields: fn-141 R8's list),
marked for the plugin by an annotation on the parameter. The capture phase lifts the argument's
typed tree to an `ir.Function` at the call and rewrites the call to supply it beside the function
value, which stays an ordinary callable with its public type. The function's name is derived as
today's lifter derives it; its position is exact.

**No non-local reads (staleness).** A capture lifts trees of its own compilation unit only. A def in
another file that a body reaches must carry `@lifted`, a plain marker annotation; the store phase
lifts the def's own body at its definition and stores the `ir.Function` on the symbol as `@liftedIr`,
and the capture references it as a call and reads only the stored IR, from the same run or from a
jar's TASTy. A `@sugar` definition in the framework or a kit jar is expanded from the jar's TASTy,
which cannot go stale (a classpath change recompiles every Model). Any other reached def is refused
at compile time naming the def and the fix. Because the carrier is in the def's API, zinc recompiles
every dependent unit when a `@lifted` body's IR changes (measured in the prototype); no hash guard.

**Declarations construct their messages.** Each DSL declaration type holds, or builds on demand, its
`ir` message: `ActionDecl` → `ir.Action`, channels, assumptions, holes, `Machine` → `ir.Machine` with
`StepBinding`s from its rules, monitors, refinement, derivations, compositions and syncs, Properties,
Scenarios, Queries, progress, capability expansions, realizations. The conversion is written beside
the type, in the framework or in the kit that owns the type (`model/temporal/realize`). Derived IDs
are computed where the name is known, as the lifter computes them today.

**Names and positions.** Names of declarations come from reflection on the declaring `val` or object,
as today (`IrFile.scala`, `Action.name`, `objectName`). Names of values passed by reference (a
command's `Instruction`, anything today's exporter names by following a helper parameter to the
caller's reference) come from the carried spelling the names phase inserts: a nameable type has a
slot (`Named`: `named(spelling)`, `ref`) that the conversion reads. The owner chooses in .2 between a
field on each nameable type and one wrapper type. Function IR keeps positions. Declaration positions
are empty unless the construction site has one at no cost. Go already accepts an empty position.

## API Contracts

- `@lifted` on a `def`: a marker; the plugin lifts the def's body at its definition and stores it
  as `@liftedIr` on the symbol, visible in the same run and from the jar's TASTy. `@liftedIr` is
  written by the plugin only.
- Capture parameters: the DSL marks them; the plugin lifts the argument and supplies the
  `ir.Function` to the constructed value. Refusals use today's text, at the refused tree.
- Nameable values: `named(spelling)` and `ref` on the DSL type; the plugin inserts `.named(...)` at
  call sites. Authors never write it. An argument of a nameable type that is neither a declaration
  reference nor a parameter is refused at compile time where today's exporter refuses it.
- Every DSL declaration type has one conversion to its `ir` message. The driver reads only those.
- The IR schema (`proto/internal/temporal/server/api/umpire/v1`) does not change. `Position` may be
  empty for declarations.
- Lift's command line and output layout (`OutputPaths`, `--ir`, one file per feature) do not change.

## Edge Cases & Constraints

- A refusal is reported at the refused tree, inside a capture argument and inside a `@lifted` def
  alike: the plugin has exact positions everywhere (spike §2.2, prototype). There is no "inline
  stack trace only" case.
- A def reached by several captures is lifted once, at its definition.
- A function-valued parameter of a helper is opaque and refused, as fn-141 R8 states.
- Nested inline calls inside a captured body (the composition `records` capture inside a Property)
  are seen before inlining, since the phases run before `pickler`; `records` need not become a
  `@sugar`. Confirm in .1 with a fixture.
- JVM string constants are limited to 64 KB; the `@liftedIr` carrier holds the `ir.Function` as a
  list of chunked constants or bytes, and the capture rewrite constructs the message with generated
  code or chunked constants. The largest function today is 27 KB compact. Measure in .1.
- The names phase rewrites only an `Ident`/`Select` of a non-parameter `val` passed to a parameter
  whose declared type is not a top type (`Any`, `AnyRef`, `Matchable`, `AnyVal`), so `eq`, `==`,
  `println` and collections of `Any` leave references alone. Default, named, vararg and constructor
  arguments are covered; the prototype exercised `Apply` only.
- Determinism: no conversion may read object identity, hash order, initialization order, time or
  environment. The two-run comparison stays.
- Initialization order: the order lint runs before construction; a `null` or half-made value reached
  during construction is refused naming the declaration that holds it.
- A change to the IR schema or to `reify` recompiles every Model (measured 3.4 s for Kettle on a warm
  server; accepted).
- The plugin uses internal compiler API (`dotty.tools.dotc.plugins`, `Annotation`, `addAnnotation`,
  `QuotesImpl`); it is built against the pinned `scala3-compiler` and the TASTy inspector already
  requires a compiler at least as new as the Models'. Record the pin and the API surface beside it.
- Refusal fixtures: `typeCheckErrors` may not run plugin phases; where it does not, the fixture is
  compiled (R8 allows either). Decide in .1 with the first refusal fixture.
- Metals: Bloop carries plugin diagnostics (prototype); as-you-type behaviour is unverified and not
  a requirement.

## Acceptance Criteria

- **R1:** Function bodies are lifted to `ir.Function` at compile time by capture points, through one
  body lifter in `model/framework/reify` that is the lifter's code moved, not rewritten, running in
  the plugin's phases. The IR of every function of the six framework samples and of `activity_mini`
  is equal, as a message with positions stripped, to the TASTy lifter's before the switch. Errors: a
  construct outside the liftable subset is a compile error with today's refusal text at the refused
  tree.
- **R2:** No capture reads a tree outside its compilation unit except a `@lifted` def's stored IR and
  a jar's `@sugar` definitions. A `@lifted` def is lifted at its definition and stored before
  `pickler`. Any other reached def is refused at compile time naming it and the fix. The
  caller-specific `choose` rules (`namedCopy`, step cardinality) are applied to stored helper IR at
  compile time at the caller's alternative. A regression test edits a helper's body in another file
  under a warm Bloop server and sees the dependent unit recompile and its IR or refusal change.
  Errors: the refusal names the def and the capture.
- **R3:** Every declaration kind's IR comes from the constructed Model, by a conversion on the DSL
  type, in this order: realizations, signature (actions, inputs, entities, observations, channels,
  assumptions, holes, Limits, types), machines (header, rules, monitors, refinement, derivations),
  compositions and syncs, claims (Properties, Scenarios, Queries, progress), capability expansions.
  After each kind moves, no kind is produced by both the tree path and the constructed path. Errors:
  a root that throws while constructed is refused naming the root and the declaration; nothing is
  written once any root failed.
- **R4:** Names are retained for every declaration and function, equal to today's. A value named by
  reference carries the caller's spelling through ordinary helpers and `val` aliases: the
  `original`/`alias` fixture of `.flow/tmp/fn1742-name-probe` gives ids `original` and `alias`
  from constructed values. Function IR keeps positions. Declaration positions may be empty. Go's
  readers, the Cases generator and the Quint and TLA+ exports accept the output unchanged. Errors: a
  name that differs from today's is a defect of the step.
- **R5:** The oracle for each step is the Go checks, not IR bytes: the six samples' receipts, each
  feature's Cases and the answers of `make umpire-check-backends` are unchanged after regenerating
  the IR, apart from positions. IR diffs are reviewed in the task's done summary, not forbidden.
  Errors: a changed receipt, Case identity or backend answer stops the step until traced.
- **R6:** Export is deterministic: the same sources give byte-identical IR on every run, checked by
  exporting each fixture in two separate JVM runs. Errors: the first differing line is named.
- **R7:** The refusal ledger (fn-141.6) has an outcome for every refusal kind: deleted (the spelling
  now exports), kept at compile time (plugin), kept at construction (factory), kept in the lint, or
  left to Go. The initialization-order lint runs before construction. Errors: a kept refusal whose
  fixture no longer refuses at its line fails the fixture test.
- **R8:** At the end, `model/framework/export` emits no IR from trees; the TASTy inspector serves the
  lints only; the lifter's fixtures are Models that compile and run in-process, with refusal fixtures
  checked through `scala.compiletime.testing.typeCheckErrors` or a compile of the fixture; the rules
  of record are rewritten (`.plans/DSL_OPERATORS.md` rule 5, `.plans/SCALA.md`, `model/README.md`'s
  "works on TASTy, not as a macro" paragraph, `model/SEMANTICS.md` §21, the `Lift.scala` header);
  the design documents' §7 and §8 and `compiler-plugin-prototype.md` record the outcome; and the
  source lines of `export`, `reify`, `inspect` and `lint` are recorded before and after. Errors: a
  doc that still states the retired reason fails review.
- **R9:** No Model loses a spelling it uses. The only Model edits are `@lifted` on helpers reached
  across files and, where .1 decides it, the `records` capture. The `.named(...)` the plugin inserts
  is never written in a Model. Errors: any other Model edit is put to the owner first.

## Boundaries

- The owner currently authorizes framework source work and read-only sample regressions. A
  committed framework source checkpoint may release preparation for the next declaration kind
  while the original kind's held acceptance remains open. Preparation retains metadata and proves
  isolated conversions; it does not activate a second production producer. Production switching,
  reader retirement, mini/Temporal equivalence and R5 remain governed by the original integration
  tasks and their dependencies. The finite-schema foundation prepares the signature kind without
  switching signature emission. Its successor constructs Action/Input/Channel messages in isolated
  framework fixtures, retaining exact typed schema selection before generic erasure. Production
  signature activation, the remaining signature kinds and state-field Inbox ownership remain with
  the original integration task. The next source checkpoint retains exact composed member/sync
  references and constructs isolated fairness references, without complete composition export.
  The next source checkpoint constructs complete native assumptions, holes and Limits and retains
  exact omitted-versus-explicit declaration/reference names, including explicitly empty values.
  Intrinsic fairness and local name/bound conflicts are checked in isolated construction; saved
  TASTy and ordinary author evaluation remain unchanged. Machine-held ownership and consumer-specific
  fairness checks follow before activation. These source
  checkpoints release neither full integration nor the kit plug-in seam. A separate Actor naming
  source checkpoint proves exact old/native actor names while preserving public getter and native
  evaluation behavior. It retains compiler module spelling only where needed, qualifies old-refused
  constructor/helper forms, and leaves Inbox ownership and production activation separate.
- Not in this spec: native collections in the macro (fn-173.5 extends the recognition table, which
  the spike showed ports unchanged); macro annotations (SIP-63) and `-experimental` (not taken: the
  plugin stores at the definition site before `pickler`, where an annotation macro cannot);
  `-Yretain-trees` (not needed: the plugin sees its run's trees and reads stored IR otherwise); an
  embedded expression DSL for bodies (rejected: native `match`, `if` and `copy` are the point);
  positions for declarations; changes to the IR schema, Go, Quint or TLA+.
- fn-172's six task closures land locally first (owner). The conductor verifies their committed
  closures before executing this source lane; fn-172's independent review and whole-spec closure
  remain at the source-batch boundary, not a whole-spec scheduling dependency. Per MILESTONES order,
  fn-141.7 also closes locally before .1 starts. fn-141.6 and .7 remain prerequisites of .8 and .2
  here and stay in fn-141. These cross-spec task prerequisites are conductor-checked because Flow
  task dependency edges are restricted to one spec.
- fn-169.2 (plug-in seam) follows .2 here and no longer owns realization lifting.

## Decision Context

[user] The owner read the design document's §8 and its pros and cons and said "let's apply this idea;
the cons seem acceptable". [user] Staleness: "the clean one; I think using macro annotations -
accepted", which is design §6 fix (a), capture at the definition site. [user] "byte-for-byte matching
is not required during the migration": fn-141 R11 is retired, the Go checks are the oracle (R5), and
two-run determinism (fn-141 R10) stays as R6. [user] Positions may go; names stay ("retaining names
would actually be nice"): names already come from reflection, so R4 keeps them at no cost. [user] The
plan must say how fn-141 Part B fits: the table in Goal & Context does; this spec is Part B's
successor with the exporter folded into the DSL types and the span lookup replaced by the capture.

[user, 2026-10-14] After .1 and .2 blocked on the two design conflicts, the owner asked for a
compiler-plugin prototype "to assess viability", then to "adjust the existing specs or create a new
one to supersede the macro based one". The prototype (`model/design/compiler-plugin-prototype.md`,
`.flow/tmp/fn174-plugin-prototype/`) solved both conflicts: stored helper IR crosses the compile
boundary and is visible to a later unit's plugin phase and to a macro; the `choose` refusal stays a
compile error at the caller's alternative with today's text; a warm body-only helper edit
invalidates the caller through zinc; `scripted(original)`/`scripted(alias)` give ids `original`/`alias`
from constructed values. The macro-annotation host is replaced by the plugin; the definition-site
staleness fix the owner accepted on 2026-10-13 is kept, in the plugin's store phase.
[inferred] Amend this spec rather than supersede it: the goal, R1–R9, the kind order, the task
ladder and every cross-reference from fn-141, fn-169.2 and fn-173.5 stay; only the host paragraphs
change. [inferred] Plugin-only host rather than plugin carrier plus macro captures: one phase owning
both what a unit stores and what every other unit reads is the simplest form of the prototype's
hardest property; the hybrid stays the fallback if `QuotesImpl` inside a phase proves unworkable in .1.

[inferred] Kind order follows fn-141 R11's (realizations first) because fn-141.7 makes realization
values complete and they hold no functions, so .2 proves the driver before functions are involved.
[inferred] The body lifter is moved, not rewritten (R1), so the function-level IR stays equal and the
331 fixtures keep their value as sources. [inferred] Rejected bigger designs: a hash-based staleness
guard (manages the defect rather than removing it), deriving conversions with `Mirror` (the DSL's 80
types and the IR's 120 messages do not line up field for field; hand-written conversions beside each
type are short and explicit).

## Quick commands

The framework source lane runs affected suites directly while the owner holds the root gate.

```bash
mise exec -- scala-cli test model/framework/dsl model/framework/realize --server=false --test-only 'framework.realize.*'
```

## Requirement coverage

| Req | Task |
| --- | --- |
| R1 | .1, .4 |
| R2 | .1 |
| R3 | .2, .3, .4, .5, .6, .7, .9, .10, .11, .12, .13, .14 |
| R4 | .2, .3, .4, .5, .6, .7, .9, .10, .11, .12, .13, .14 |
| R5 | .2, .3, .4, .5, .6, .7, .8 |
| R6 | .2, .8, .9, .11, .12, .13, .14 |
| R7 | .8 |
| R8 | .8 |
| R9 | .1, .8, .10, .11, .12, .13, .14 |
