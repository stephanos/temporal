# The Model constructs its own IR: DSL types build `ir` messages, capture points lift function bodies, the lifter retires

## Conversation Evidence

> user: "let's apply this idea ; the cons seem acceptable"
> user: "stalness had a solution (the clean one; I think using macro annotations - accepted)"
> user: "byte-for-byte matching is not required during the migraiton"
> user: "we can live without position and names in IR" … "well, retaining names would actually be nice"
> user: "and re-work the plan with how fn-141 Part B fits into this"

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
lifted at compile time by a macro at the capture point with the same `quotes.reflect` code the lifter
uses, with identical output, and §8 of that document describes the end state this spec builds: each
DSL type constructs its own `ir` message, the capture point returns an `ir.Function`, and lift is
"construct, then print". No exporter, no lifter JVM, no TASTy for IR.

**Owner decisions, 2026-10-13.** Adopt §8. Staleness is solved the clean way, at the definition site,
with macro annotations (SIP-63, experimental in Scala 3.9; accepted). Byte-for-byte IR identity is
not required during the migration: the oracle is the Go checks' answers. Declaration positions may be
dropped from the IR; names are retained (they already come from reflection on the declaring `val` or
object, so this costs nothing). fn-172's toolchain fixes land first.

**How fn-141 Part B fits.** This spec is Part B's successor: same obligations, different mechanism.

| fn-141 | Here |
| --- | --- |
| .1–.4 (lints out, unused spellings, sugar once): done | Prerequisites, unchanged. The generic sugar expansion (.3) moves into the macro module as is. |
| .5 capture points and lookup by span | **Superseded.** The capture point lifts the body it holds; no span is recorded and no lookup is written (.1 here). |
| .6 refusal ledger | **Stays in fn-141**, a prerequisite of .8 here (ledger rows decide each refusal's home). |
| .7 realization factories build complete values | **Stays in fn-141**, a prerequisite of .2 here. |
| .8–.13 one exporter per kind | **Superseded** by .2–.7 here, same kind order (realizations, signature, machines, compositions, claims, capabilities), with the conversion on the DSL type instead of in an exporter. |
| .14 close | **Superseded** by .8 here. fn-141 closes when its .6 and .7 are done and this spec's .8 has retired the lifter. |
| R6, R7 (names), R8 (functions captured), R9, R12, R13, R14, R16 | Carried, restated as R-IDs below. |
| R7 (positions), R8 (span lookup), R11 (byte identity per step) | Retired by the owner's decisions; R10 (two-run determinism) is kept. |
| fn-169.2 "follows fn-141.8" | Follows .2 here; it shrinks to layout conventions and API descriptors, since the kit's types convert themselves. |
| fn-173.5 "DSL and lifter: native collections" | Becomes "DSL and macro": the spike's §2.4 shows the recognition table ports unchanged. |

## Architecture & Data Models

**Modules.**

- `model/framework/reify` (new): the macro module. It holds the body lifter moved from
  `model/framework/export` (`Expressions`, `Types`, `Expansion`, `Roles`, `Equality`, `Constants`,
  the name and position parts of `Context`; about 2,300 lines), written against `Quotes` as it is
  today, with two seams: "definition of a symbol" (`Symbol.tree` instead of `Index.defs`) and
  "refuse" (`report.errorAndAbort` instead of `LiftError`). It exposes `Reify.function`, which a
  capture point calls on its `inline` argument and which returns an `ir.Function`, and the `@lifted`
  macro annotation. It is compiled before the DSL and packaged as its own jar, like the lifter today.
- `model/framework/dsl` depends on `reify` and on the generated IR classes (`build/ir-scalapb.jar`).
  The DSL already depends on the ScalaPB runtime; the Temporal Models already compile against
  `build/api-scalapb.jar`, so this is a new jar, not a new kind of dependency.
- Models compile with `-Yretain-trees` and `-experimental` (`model/project.scala`,
  `model/framework/dsl/project.scala`, the samples and the fixtures).
- `model/framework/export` ends as the driver: discover roots (`IrFile.discover`), construct them
  (`IrFile.construct`), collect the messages they hold into one `ir.Model` per IR file, print with the
  existing printer. Under 100 lines plus the multi-file and `--ir` plumbing that stays.
- `model/framework/inspect` and `model/framework/lint` keep the TASTy inspector for the order,
  structure and marker lints. The inspector reads no IR.

**Function capture.** A capture point is an `inline` method whose function argument is `inline`
(`Rules.on`, `effect`/`is` blocks, `holds`/`holdsAcross`, monitor `next`/`after`, `leadsTo`
`from`/`to`, `end`, refinement maps, evidence, capability function fields: fn-141 R8's list). It
lifts the argument's tree to an `ir.Function` at expansion and stores it on the constructed value.
The function's name is derived as today's lifter derives it; its position comes from the macro for
free.

**No non-local reads (staleness).** A capture lifts trees of its own compilation unit only. A def in
another file that a body reaches must carry `@lifted`, a macro annotation that lifts the def's own
body at its definition site and stores the `ir.Function` beside it; the capture then references it as
a call. A `@sugar` definition in the framework or a kit jar is expanded from the jar's TASTy, which
cannot go stale (a classpath change recompiles every Model). Any other reached def is refused at
compile time naming the def and the fix. This removes the measured incremental-staleness defect
(design §6) without a hash guard.

**Declarations construct their messages.** Each DSL declaration type holds, or builds on demand, its
`ir` message: `ActionDecl` → `ir.Action`, channels, assumptions, holes, `Machine` → `ir.Machine` with
`StepBinding`s from its rules, monitors, refinement, derivations, compositions and syncs, Properties,
Scenarios, Queries, progress, capability expansions, realizations. The conversion is written beside
the type, in the framework or in the kit that owns the type (`model/temporal/realize`). Derived IDs
are computed where the name is known, as the lifter computes them today.

**Names and positions.** Names come from reflection on the declaring `val` or object, as today
(`IrFile.scala`, `Action.name`, `objectName`). Function IR keeps positions. Declaration positions are
empty unless the construction site has one at no cost. Go already accepts an empty position.

## API Contracts

- `Reify.function[F](inline f: F): ir.Function` (macro). Refusals use today's text, at the position
  of the refused tree when it lies inside the argument.
- `@lifted` on a `def`: lifts the def's body at its definition and makes it referable from captures.
- Every DSL declaration type has one conversion to its `ir` message. The driver reads only those.
- The IR schema (`proto/internal/temporal/server/api/umpire/v1`) does not change. `Position` may be
  empty for declarations.
- Lift's command line and output layout (`OutputPaths`, `--ir`, one file per feature) do not change.

## Edge Cases & Constraints

- A refusal inside a `@lifted` def is reported at the def's position (the annotation runs there). A
  refusal inside a capture argument is reported at the refused tree. There is no "inline stack trace
  only" case left.
- A def reached by several captures is lifted once, at its definition.
- A function-valued parameter of a helper is opaque to the macro and refused, as fn-141 R8 states.
- Nested inline calls inside a captured body (the composition `records` capture inside a Property)
  are seen expanded; the macro must recognise the expansion or `records` must become a `@sugar`.
  Decide in .1 with a fixture.
- JVM string constants are limited to 64 KB; the macro constructs the `ir.Function` with generated
  code, or emits one chunked constant per function. The largest function today is 27 KB compact.
- Determinism: no conversion may read object identity, hash order, initialization order, time or
  environment. The two-run comparison stays.
- Initialization order: the order lint runs before construction; a `null` or half-made value reached
  during construction is refused naming the declaration that holds it.
- A change to the IR schema or to `reify` recompiles every Model (measured 3.4 s for Kettle on a warm
  server; accepted).
- `-experimental` is required for macro annotations on Scala 3.9. Record the flag's reason beside it;
  if the annotation becomes stable, drop the flag.

## Acceptance Criteria

- **R1:** Function bodies are lifted to `ir.Function` at compile time by capture points, through one
  body lifter in `model/framework/reify` that is the lifter's code moved, not rewritten. The IR of
  every function of the six framework samples and of `activity_mini` is equal, as a message with
  positions stripped, to the TASTy lifter's before the switch. Errors: a construct outside the
  liftable subset is a compile error with today's refusal text; inside a capture argument it is
  reported at the refused tree.
- **R2:** No capture reads a tree outside its compilation unit except a `@lifted` def's stored IR and
  a jar's `@sugar` definitions. A `@lifted` def is lifted at its definition. Any other reached def is
  refused at compile time naming it and the fix. A regression test edits a helper's body in another
  file under a warm Bloop server and sees the IR change. Errors: the refusal names the def and the
  capture.
- **R3:** Every declaration kind's IR comes from the constructed Model, by a conversion on the DSL
  type, in this order: realizations, signature (actions, inputs, entities, observations, channels,
  assumptions, holes, Limits, types), machines (header, rules, monitors, refinement, derivations),
  compositions and syncs, claims (Properties, Scenarios, Queries, progress), capability expansions.
  After each kind moves, no kind is produced by both the tree path and the constructed path. Errors:
  a root that throws while constructed is refused naming the root and the declaration; nothing is
  written once any root failed.
- **R4:** Names are retained for every declaration and function, equal to today's. Function IR keeps
  positions. Declaration positions may be empty. Go's readers, the Cases generator and the Quint and
  TLA+ exports accept the output unchanged. Errors: a name that differs from today's is a defect of
  the step.
- **R5:** The oracle for each step is the Go checks, not IR bytes: the six samples' receipts, each
  feature's Cases and the answers of `make umpire-check-backends` are unchanged after regenerating
  the IR, apart from positions. IR diffs are reviewed in the task's done summary, not forbidden.
  Errors: a changed receipt, Case identity or backend answer stops the step until traced.
- **R6:** Export is deterministic: the same sources give byte-identical IR on every run, checked by
  exporting each fixture in two separate JVM runs. Errors: the first differing line is named.
- **R7:** The refusal ledger (fn-141.6) has an outcome for every refusal kind: deleted (the spelling
  now exports), kept at compile time (macro), kept at construction (factory), kept in the lint, or
  left to Go. The initialization-order lint runs before construction. Errors: a kept refusal whose
  fixture no longer refuses at its line fails the fixture test.
- **R8:** At the end, `model/framework/export` emits no IR from trees; the TASTy inspector serves the
  lints only; the lifter's fixtures are Models that compile and run in-process, with refusal fixtures
  checked through `scala.compiletime.testing.typeCheckErrors` or a compile of the fixture; the rules
  of record are rewritten (`.plans/DSL_OPERATORS.md` rule 5, `.plans/SCALA.md`, `model/README.md`'s
  "works on TASTy, not as a macro" paragraph, `model/SEMANTICS.md` §21, the `Lift.scala` header);
  the design document's §7 and §8 record the outcome; and the source lines of `export`, `reify`,
  `inspect` and `lint` are recorded before and after. Errors: a doc that still states the retired
  reason fails review.
- **R9:** No Model loses a spelling it uses. The only Model edits are `@lifted` on helpers reached
  across files and, where .1 decides it, the `records` capture. Errors: any other Model edit is put
  to the owner first.

## Boundaries

- Not in this spec: native collections in the macro (fn-173.5 extends the recognition table, which
  the spike showed ports unchanged); the compiler-plugin host (not taken; `@lifted` gives exact
  positions without it); an embedded expression DSL for bodies (rejected: native `match`, `if` and
  `copy` are the point); positions for declarations; changes to the IR schema, Go, Quint or TLA+.
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
accepted", which is design §6 fix (a), capture at the definition site; `-experimental` is accepted
with it. [user] "byte-for-byte matching is not required during the migration": fn-141 R11 is
retired, the Go checks are the oracle (R5), and two-run determinism (fn-141 R10) stays as R6. [user]
Positions may go; names stay ("retaining names would actually be nice"): names already come from
reflection, so R4 keeps them at no cost. [user] The plan must say how fn-141 Part B fits: the table
in Goal & Context does; this spec is Part B's successor with the exporter folded into the DSL types
and the span lookup replaced by the capture macro.

[inferred] Kind order follows fn-141 R11's (realizations first) because fn-141.7 makes realization
values complete and they hold no functions, so .2 proves the driver before functions are involved.
[inferred] The body lifter is moved, not rewritten (R1), so the function-level IR stays equal and the
331 fixtures keep their value as sources. [inferred] Rejected bigger designs: a plugin host (internal
compiler API), a hash-based staleness guard (manages the defect rather than removing it), deriving
conversions with `Mirror` (the DSL's 80 types and the IR's 120 messages do not line up field for
field; hand-written conversions beside each type are short and explicit).

## Requirement coverage

| Req | Task |
| --- | --- |
| R1 | .1, .4 |
| R2 | .1 |
| R3 | .2, .3, .4, .5, .6, .7 |
| R4 | .2, .3, .4, .5, .6, .7 |
| R5 | .2, .3, .4, .5, .6, .7, .8 |
| R6 | .2, .8 |
| R7 | .8 |
| R8 | .8 |
| R9 | .1, .8 |
