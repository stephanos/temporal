# The reify plugin keeps only the body lifter: one compilation, one carrier, declared Machine shape

## Conversation Evidence

> user (2026-10-16): "how can we change the DSL to _significantly_ reduce effort of the compiler plugin? ie what are tradeoffs there that have huge effects on the complexity and size of that plugin?"
> user: "we won't support that" (a facet template or a lifted helper in another jar)
> user: "I meant we won't have another jar outside of what we define in model/; it will all be in there"
> user: "is this part absolutely needed? the `val init` reads much nicer" (Machine endpoints as constructor arguments)
> user: "would this also work without the @lifted ? that'll become cumbersome"
> user: "let's write a spec with 0, 1, 2 (if author never has to write @lifted), 4, 5, 6, 7, 8"

## Goal & Context

The `umpire-reify` compiler plugin (`model/framework/reify`) is about 8,500 lines. Roughly 3,500 of
them are the body lifter moved from the old TASTy lifter: the part that turns a Scala expression into
an `ir.Function`. That part is the product and its size tracks the expression language Models get.
The other 5,000 lines exist to *find* and *carry* what the lifter needs: eleven carrier annotations,
nine phases in three modes, a jar unpickler, a dataflow analysis for `family`, name-based recognition
of `init`, `end`, `evidence`, `effects` and `rules`, and schema computation inside the compiler.
Every fn-174 checkpoint since .9 has added 200 to 2,000 lines in exactly those areas.

The pass over the plugin on 2026-10-16 showed that each of those areas is paying to infer something
from where code sits, and that in each case a declaration, a compilation-boundary decision, or a
compiler feature that already exists makes the inference unnecessary. This spec removes them. The
author-facing DSL does not change except for three small things: a `given Family` line per feature
package, `Lifted` as the marker on a shared vocabulary object instead of a per-def annotation, and
uncurried helpers in lifted code. `val init` and `def end` stay as they are written today.

Target after this spec: the plugin is the body lifter plus one store phase, one capture phase and one
carrier, a few hundred lines around the lifter instead of 5,000.

Out of this spec (owner, 2026-10-16): closed facet templates (the contextual-phase redesign) and
declaration-site naming. Both change what authors write or what the IR means and get their own
design note first.

## Architecture & Data Models

**The compilation contract.** Every Model and every definition a Model lifts from are compiled in one
`scala-cli` compilation of sources under `model/`: the framework DSL and realize core, the Temporal
kit and the selected features, as the Makefile's model sources already are. The only jars on the
Models' classpath are the generated IR and API proto classes and the plugin itself. Neither holds a
liftable body. The plugin reads no TASTy from jars and treats nothing on the classpath as untrusted.

**One carrier.** Under incremental compilation (the editor), a file that did not change is not in the
run and its trees are not available. The plugin therefore still stores, before `pickler`, the lifted
IR of every definition that a capture in another file may reach, as one annotation `liftedIr` on the
definition's symbol. That is the mechanism the prototype proved (`compiler-plugin-prototype.md`):
the annotation is in the API hash, so zinc recompiles dependents when the body changes. It carries
base64 chunks of one `ir.Model` and nothing else: no version, no fingerprint, no position table, no
call-group proof. A full compile (the Makefile) reads the same annotations it just wrote.

**What is stored, decided at the definition.** The plugin stores IR for a def when, and only when:

- its owner derives from `framework.dsl.Lifted`, a marker trait, or
- its result type is a step list `List[Step[S, O, F]]`, or
- it overrides `Machine.end` or `Machine.evidence`.

`Machine`'s own `effects` object derives from `Lifted` through the framework, so a Model's effects
are stored without the author writing anything. A shared vocabulary object in another file writes
`extends Lifted` once. The per-def `@lifted` annotation is removed from the DSL. A capture that
reaches a def in another file that none of the three rules covers is refused at the capture with
the def's name and "define it in an object that extends Lifted" (R2). In a full compile every tree is
in the run, so the refusal is reported there as well, with the same text, to keep build and editor
identical.

**Family.** `family` becomes a value supplied by the feature: `given Family = Family.here`, where
`Family.here` is a small macro in the DSL that reads the enclosing package. Realization-kit
functions that need the family take `(using Family)`. The `lazy val family` that throws, its
`contextualFamily` twin, `FamilyPhase` and the `familyFlow` carrier are removed.

**Machine shape.** `Machine` declares `def init: S`, `def end(state: S): Boolean = true` and
`def evidence: PartialFunction[F, String] = PartialFunction.empty` as members. The plugin recognises
them as overrides of those symbols, never by spelling. `init` is not lifted: it is a value of a
`Finite` type and the DSL encodes it through the type's schema at construction. `Phased` supplies the
default `end` that today's plugin synthesises from the projection. Schemas of the S, O and F type
arguments, the Machine's header identity, its Actor's name and the enum-case fact registrations are
produced by the DSL at runtime from the `Finite` instances and `scala.deriving.Mirror`; the plugin
injects no `retain*` or `declaredFact` calls for them.

**Sugar.** `enter`, `stay`, `reject`, `disabled` and `in` become `inline def`s. The compiler expands
them in typer, so the lifter meets their core form inside an `Inlined` node and uses the node's call
for the name in diagnostics. The plugin's own expansion engine and the `@sugar` annotation are
removed.

**Positions.** A stored `ir.Model` carries a position only on its root function. Compile-time
refusals are reported by the compiler at the tree and are unaffected.

**Curried helpers.** A lifted body may call helpers with one term parameter list. `using` clauses are
allowed. The call-group proof and its carrier are removed.

**Captured arguments (after the old lifter retires).** A capture point's parameter is typed
`Captured[F]` and the capture phase wraps the argument in place before `pickler`:
`Captured(value, root, chunks)`. The hidden per-arity overloads (`capturedRule0..n`,
`capturedBinding0`, `capturedRebind`, `capturedExtend`, `capturedDisabledRules`), the post-pickler
retargeting phase and both retention phases go. This waits for fn-174.8, because until then the old
reader needs the original trees in TASTy.

**Phases after this spec:** `reifyStore` (store IR at definitions) and `reifyCapture` (lift at
capture points, run the `choose` cardinality check). One mode. Two annotations in the DSL:
`liftedIr` (carrier) and `capture` (parameter marker).

## API Contracts

Author-facing, the complete set of changes:

```scala
// a feature package supplies its family once
given Family = Family.here

// a shared vocabulary object in another file
object temperatures extends Lifted:
  def boiling(s: Snapshot) = s.reading >= 100

// a Machine, unchanged from today
object Model extends Machine[Snapshot, Outcome, Nothing]:
  val init = Snapshot(reading = 20, heatingOn = false, beeping = false)
  def end(state: Snapshot) = state.beeping
  object effects:
    def heat(s: Snapshot) = enter(s.copy(reading = s.reading + 1))
  object rules extends Rules:
    on(heater.heat): where(_.heatingOn) ~> effects.heat
```

Kit-facing: a realize function that forms a Definition ID takes `(using f: Family)` and reads
`f.id`. Nothing else in the kit changes.

Compiler refusals, exact texts are the contract:

- a capture reaches `pkg.helper` defined in another file outside the storage rules:
  `pkg.helper is reached from another file: define it in an object that extends Lifted`
- a lifted body calls a curried def `pkg.warmBy`:
  `pkg.warmBy has two parameter lists: write it with one`
- a Machine without `init`: the ordinary Scala "object creation impossible" error, no plugin text.

## Edge Cases & Constraints

- A def covered by two storage rules (a step function inside a `Lifted` object) is stored once.
- A `Lifted` object nested in a Machine is stored; the Machine's own `effects` is covered the same
  way, so moving effects between the two changes nothing.
- A helper in the same file as its capture is lifted from its tree and also stored if a rule covers
  it; the capture prefers the tree. The two agree by construction, and a test asserts it.
- Recursive helpers keep today's refusal "calls itself, directly or through another function".
- `Family.here` inside the framework or the kit packages is a compile error: a family names a
  feature.
- The editor: under incremental compilation a capture that reaches a def whose file is not in the
  run reads `liftedIr`; if the def's file was compiled without the plugin (no annotation), the
  refusal of R2 fires, naming the def. There is no stale-annotation case to detect: zinc recompiles
  the dependent when the API hash changes.
- The `in` extension with its `TypeTest` overload must still resolve its given at the call site
  once `inline`. A compile of the Facets fixture is the check.
- The DSL's framework-alone compile stays, as a boundary lint that the framework needs no Model.
  Nothing compiles a Model against that jar.
- Backward compatibility: none owed. fn-174's "equivalence except positions" comparisons against the
  old reader shrink to "equivalence except positions on non-root functions"; its retained old
  outputs are not regenerated by this spec.

## Acceptance Criteria

- **R1:** The plugin contains no code that reads TASTy or trees from a jar, no trusted-artifact or
  source-path identity checks, and no carrier version or fingerprint. The DSL and plugin build
  against the generated proto jars only. Errors: a capture whose reached definition has no tree in
  the run and no `liftedIr` annotation is refused with the R2 text; no other error surface.
- **R2:** Storage is decided at the definition by the three rules in Architecture: owner derives
  from `Lifted`, result type is a step list, or override of `Machine.end`/`Machine.evidence`. The
  public `@lifted` annotation does not exist in the DSL. A Model with effects in its Machine and a
  helper in a `Lifted` object in another file lifts under a full compile and under a warm Bloop
  session where only the helper's body is edited, and the dependent Model recompiles with the new
  IR (the prototype's warm test, on the real lifter). Errors: a reached def in another file that no
  rule covers → the exact text in API Contracts, at the capture; same text in a full compile.
- **R3:** `family` is a `given Family` supplied by the feature package through `Family.here`; kit
  functions take it as a `using` parameter; `FamilyPhase`, `familyFlow`, `contextualFamily` and the
  throwing `lazy val family` are gone. The six samples and the realize fixtures produce the same
  Definition IDs as before. Errors: `Family.here` in a framework or kit package → compile error
  naming the package; a kit function called without a `Family` in scope → the ordinary missing
  given error.
- **R4:** `Machine` declares `init`, `end` and `evidence` as members; the plugin identifies them by
  override, not by name; `init` is encoded from its value through the state's `Finite` schema and
  never lifted; `Phased` provides the projection-based default `end`. The plugin injects no
  `retainHeaderIdentity`, `retainHeaderSchema`, `retainActorName`, `retainEndpoints`,
  `retainEvidence`, `declaredFact` or input/last-input schema registration; the DSL derives the
  same messages at runtime from `Finite` and `Mirror`, and the six samples' headers, schemas,
  endpoints and evidence match the retained old reader output except positions. Errors: a Machine
  without `init` → Scala's own abstract-member error; a state type without a `Finite` instance →
  the ordinary missing given error at the Machine.
- **R5:** A stored `ir.Model` carries a position on its root function only; `ruleTypePosition`,
  the type-location walk and the source-label maps are gone. Compile-time refusal positions are
  unchanged, verified by the existing refusal fixtures. No error surface beyond R2.
- **R6:** A lifted body calling a def with more than one term parameter list is refused with the
  exact text in API Contracts; `using` clauses do not count. `ruleCallGroups` and the call-group
  proof are gone. The six samples compile unchanged.
- **R7:** The five sugar defs are `inline`; `@sugar`, the plugin's expansion engine and the jar
  sugar lookup are gone; the lifter's diagnostics name the sugar from the `Inlined` call. The six
  samples and the Facets fixture compile and lift to the same IR as before except positions.
  Errors: a sugar body the lifter cannot lift reports at the call site, as today.
- **R8:** After fn-174.8 has retired the old reader: capture parameters are typed `Captured[F]`,
  the capture phase wraps the argument in place before `pickler`, and the per-arity hidden
  overloads, `RuntimeCapturePhase` and both retention phases are gone. The six samples lift to the
  same IR as under R7. Errors: a capture whose argument is not a function literal or a reference
  keeps today's "a capture applies one marked function parameter at its call site".
- **R9:** Counts recorded in the task receipt at the end: plugin phases (two), plugin modes (one),
  annotations in the DSL's reify vocabulary (two: `liftedIr`, `capture`), and plugin lines outside
  the body-lifter files. The last is reported, not gated. No error surface.

## Boundaries

- Closed facet templates (explicit phase lens, composition as an IR call) are not in this spec.
  Facets keep today's contextual-phase mechanism until a design note in `model/design` decides it.
- Declaration-site naming and the fate of `scripted(alias)` spelling are not in this spec.
  `NamesPhase` and the name-related `retain*` rewrites stay as they are.
- No change to the body lifter's expression language.
- No change to Go consumers, the IR schema, Testpilot, or the old TASTy reader except that R8 is
  sequenced after its retirement.
- No support for lifted code outside the single `model/` compilation, now or later.
- Temporal sources stay disabled; the samples and the framework fixtures are the proving ground.

## Decision Context

The alternative of keeping the carriers and tightening them was rejected: their cost is not the
annotation but the jar case they defend against, and that case was declared unsupported by the
owner on 2026-10-16. Keeping exactly one carrier, rather than none, is deliberate: a full compile
does not need it, but the editor's incremental compilation does, and the prototype measured that it
costs about 150 lines.

Machine endpoints as constructor arguments were considered and rejected by the owner for
readability; abstract members give the same "declared, not recognised" property with today's
spelling. A per-def `@lifted` marker was rejected as cumbersome; the container marker and the
step-list type rule cover every case in the samples, and the refusal names the fix for the rest.

`inline` sugar was chosen over keeping the plugin's expansion because the compiler already
implements inlining and does it before the plugin runs, which also makes sugar independent of
whether the DSL's file is in the current run.

Positions on non-root functions were already allowed to be empty by the owner (fn-174, 2026-10-13);
this spec makes the code follow the decision. The Go evaluator's diagnostics for a helper or type
then point at the rule that reached them, which the owner accepted.

R8 is in this spec because it completes the shape, but it cannot start before fn-174.8: the
post-pickler split exists only so the old reader still sees source trees. Everything else here can
run during the framework-only phase.

**2026-10-16, after fn-190 was captured.** fn-190 (Models as plain Scala: state as vars, effects as
assignment blocks, no in-process execution) re-cuts R4 here: the State record, its schema and
`init` come from the var declarations in the plugin, so task .6 is superseded by fn-190.2 and the
facet item in Boundaries is closed by fn-190.6. R1, R2, R3, R5, R6, R7 and R8 stand unchanged;
fn-190.8 re-records R9.

Order of work, cheapest and least visible first: R1 and R5 together, then R6, R7, R3, R2, R4, and R8
when its prerequisite lands.

## Requirement coverage

| Req | Task(s) | Gap justification |
| --- | --- | --- |
| R1 | fn-189.1 | — |
| R2 | fn-189.5 | — |
| R3 | fn-189.4 | — |
| R4 | fn-189.6 | — |
| R5 | fn-189.1 | — |
| R6 | fn-189.2 | — |
| R7 | fn-189.3 | — |
| R8 | fn-189.8 | after fn-174.8 retires the old reader |
| R9 | fn-189.7, fn-189.8 | — |
