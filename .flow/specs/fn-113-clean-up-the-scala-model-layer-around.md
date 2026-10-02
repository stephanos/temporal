# Clean up the Scala model layer around the IR

## Goal & Context
<!-- scope: business -->

Scala is where Temporal features are authored from now on. The Lean model is the past, and the Scala implementation has no duty to look like it. The Umpire IR and the Testpilot IR are what connect the parts: Scala definitions lift to the Umpire IR, and generic Go consumers check it and lower it to Testpilot Cases.

`model/scalav2/scala` was ported from Lean and still carries that history. A review on 2026-10-01 measured it.

- `scala/umpire` is 2,873 lines. Most of it is a second evaluator: it builds tables, searches Queries, checks refinements and composes machines at runtime. Go does all of that from the IR. The Scala evaluator already refuses channels, holes, monitors and progress claims, which only the IR interpreter reads, and only the 32 munit tests call it.
- About 390 of those lines have no caller at all (`Canonical.scala`, `Lower.scala`, the `Alterer` plumbing).
- `UmpireSet` and `Coverage` have no IR form. The lifter never reads them, so no Go consumer sees them.
- The Nexus caller keeps a "kernel" written in the subset Stainless accepts, with a prelude, a dispatch function for lemmas and a test that the dispatch agrees with the machines. The Stainless proofs were never brought into `model/scalav2`.
- The lifter builds the IR through 108 protobuf-java builder chains.

This spec makes the Scala layer the smallest and most expressive authoring front end for the IR. It serves two readers. The feature developer writes and reviews Models under `scala/temporal` and wants each fact stated once, by value. The framework developer maintains `scala/umpire` and the lifter and wants less code to keep in step with the IR.

The owner prefers a maintained library over hand-written code where the library reduces effort. Libraries are welcome in `scala/umpire` and the lifter. The Models under `scala/temporal` use the `umpire` DSL. Keeping them free of libraries is guidance, not a hard rule. A Model gains a library only where a spec names it, as `fn-117-type-the-temporal-api-in-the-models` does for the generated Temporal API classes.

## Architecture & Data Models
<!-- scope: technical -->

One rule decides what the Scala layer holds: **Scala declares, the lifter reads, Go evaluates.** A declaration earns its place when the lifter emits it into the IR or when the compiler uses its type to reject a wrong Model. Code that computes at runtime what Go derives from the IR does not.

The spec has four parts. Each is independent unless the order below says otherwise. A fifth part, stating every declaration once across all Models, moved to `fn-114-state-every-scala-model-declaration-once` on 2026-10-01.

| Part | What it does | Depends on |
| --- | --- | --- |
| A. Dead code | Delivered early by fn-115.7 after relocation: delete unused `Canonical.scala`, `Lower.scala` and `Alterer` plumbing; this spec verifies R1 against that result | fn-115.6 |
| B. ScalaPB | Replace protobuf-java in the lifter and in the gate's IR class generation | nothing |
| C. One evaluator | Retire the native Scala evaluator, after its tests are covered on the IR | A |
| D. Lean and Stainless residue | Fold the Nexus kernel into ordinary Scala, drop Lean-mirroring code and citations | C |

**After Part C, `scala/umpire` is a typed declaration DSL.** It keeps the types that make a wrong Model fail to compile (`Action[I]` and the typed `~>`, `Property[S]`, `Scenario[S]`, the query chain, `Finite` as evidence that a domain is finite), the `Step` type and the step helpers, and the realization declarations. It loses `Table`, the search, the runtime refinement check, the runtime composition table, key spelling, catalog ordering and Definition ID construction. Step functions stay ordinary executable Scala, so a native test can still call one and compare its result.

**Model problems are reported by Go, at the Scala line.** A start outside the state domain, a stuck state, a class bound twice, a refinement that fails: `goir` reports each from the IR with the source position the lifter recorded. The Scala layer no longer reports them a second time.

Part B's pipeline change:

| Stage | Today | After |
| --- | --- | --- |
| Schema | `proto/internal/temporal/server/api/modelir/v1/ir.proto` | unchanged |
| Generation (the gate) | `protoc --java_out`, packaged as a jar | protoc with the ScalaPB plugin, generated Scala compiled into the same jar |
| Lifter dependencies | `protobuf-java`, `protobuf-java-util`, the jar | `scalapb-runtime`, a ScalaPB ProtoJSON printer, the jar |
| IR construction in the lifter | `ir.X.newBuilder().setA(a).build()` | `ir.X(a = a)` |
| Realization emitter | reflection over `FieldDescriptor` and `Message.Builder` | reflection over `scalapb.descriptors` |
| Output | ProtoJSON from `JsonFormat` | ProtoJSON from the ScalaPB printer |
| Go side | `protojson.Unmarshal` in `goir` | unchanged |

## API Contracts
<!-- scope: technical -->

The lifter's `lift: <file>:<line>: <message>` refusal format and the IR schema stay as they are.

**Lifted meaning is frozen, lifted text is not.** For every IR file, `goir` derives the same tables, Definition IDs, refinement rows, fingerprints and Query answers before and after each task, and `goir/testpilot` lowers the same Case bytes. The checked-in ProtoJSON may change in whitespace, field order, number spelling, source positions and lifter-internal function names.

## Edge Cases & Constraints
<!-- scope: technical -->

- **The Go checker's typed fixture layer.** fn-115.7 moved the checker's typed declaration layer into `tools/umpire/model/internal/checker/*_support_test.go` (1,348 lines), because most of the checker's tests build fixtures with it. Production still holds the branches only those fixtures reach: typed `holds`/`holds2` in `search.go` and `lower.go`, the typed `alterer`, `readState`/`readStep`, the typed `observe` arms, `Refinement.productStep`/`stepOf`/`keyLevel`, the `checkKeyRefined` cross-checks and the typed branch of `asking.accepts`. Part C ports those tests to fixtures built from IR and deletes the branches, or records why a branch stays.
- **No test is deleted before its claim is covered.** Part C starts with an audit that maps each of the 32 munit tests to a Go test over the IR that asserts the same thing, or ports the assertion to Go, or records why the claim no longer applies (for example, "the native table refuses a channel"). The evaluator is removed only after the audit is committed.
- **Declarations with no IR form.** `UmpireSet`, its purposes and bindings, and `Coverage` reach no consumer. They are removed. The task lists each removed declaration with the authored facts it carried (which Queries were canary, which parties were observed), so the owner can ask for an IR form in a later spec without archaeology.
- **`Finite` after Part C.** It must still reject a state or input type with a non-finite field at compile time, and the lifter must still read integer bounds. It no longer has to enumerate values at runtime.
- **The Nexus kernel.** `kernel/NexusActions.scala`, `umpire.prelude` and `NexusKernel.test.scala` exist for Stainless lemmas that are not in this tree. The step functions move to ordinary Scala (`List`, `Nil`, `Step(...)`, `derives Finite`), and the lifter's special case for `umpire.prelude` goes with them.
- **Comments.** A comment that explains a rule stays and loses its Lean citation. A comment whose only content is a Lean or Stainless reference is deleted. A comment whose code is deleted goes with it. This is a deliberate exception to the repository rule that refactors preserve comments, taken because the owner retired Lean as a reference.
- **Version fit for ScalaPB.** The lifter compiles with Scala 3.9.0 and protoc comes from `mise.toml`. A ScalaPB release must support both. This is unverified and is the first thing Part B checks.
- **Default-valued fields.** The realization declarations rely on "a default is the empty value, which the IR leaves unset". The ScalaPB port must not start emitting defaults.
- **Realization emitter stays generic.** It maps a constructor to the IR message of its name through descriptors. A hand-written mapping per message would add the code this spec removes.
- **Libraries.** A library is adopted in `scala/umpire` or the lifter when a task shows it removes more hand-written lines than its wiring adds. A library that a Model under `scala/temporal` can import is not added by a task of this spec. A Model gains a library only where a spec names it, as `fn-117-type-the-temporal-api-in-the-models` does for the generated Temporal API classes. After Part C the review expects no library beyond ScalaPB to qualify, because the code a library could replace (derivation, JSON, name capture at runtime) is gone or small. Tasks record the candidates they weighed.
- **fn-112 overlaps.** `fn-112-make-the-standalone-activity-scala` adds framework constructs and rewrites the standalone activity. It was written when the native evaluator had to agree with the lifted IR, when no library was allowed, and when Lean provenance comments were frozen. Parts C and D remove those three constraints, and fn-112 was amended on 2026-10-01 to match: it depends on this spec, and each of its constructs is built once, in the lifter. `fn-114-state-every-scala-model-declaration-once` rolls the constructs out to the other Models after fn-112. fn-112 starts after this spec closes, so Part B and fn-112 never edit the lifter at the same time.
- **Layout.** `fn-115-make-the-scala-model-the-model-and` runs before this spec, and fn-107 is closed by then. fn-115 moves the model to `model/` and the Go reader to `tools/umpire/`, archives the Lean-era trees, splits the lifter by concern, replaces the shell scripts with one gate program, removes every mention of Lean under `model/` and writes the baseline goldens. Where this spec names `scala/umpire`, `scala/temporal`, `goir` or `lifter`, it means the DSL, the Models, the Go model reader and the lifter at the places fn-115 gives them.
- **Gates.** Each task runs the scoped parts of the model gate, `make lint-scala` and the Go tests of the Umpire tooling. The closing task runs all three in full and `make lint-code-fast`. No task installs a Lean toolchain.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** `Canonical.scala`, `Lower.scala`, the `Alterer` type and `Table`'s `alter` field are gone, together with every declaration in `scala/umpire` that has no caller after their removal. Errors: a declaration that a test, `Coverage`, `Search`, `Refine` or `Compose` still reads stays; the task lists each borderline declaration it kept and why.
- **R2:** Part A changes no lifted output. The model gate passes without `--update`, and no checked-in IR file or expected lifter fixture output differs. Errors: any diff in those files means the deletion removed something the lifter reads, and the task stops and reports it.
- **R3:** This spec adds no library that a file under `scala/temporal` can import (no error surface beyond the build).
- **R4:** Before any port work, the task records which ScalaPB release, ProtoJSON printer and protoc plugin work with Scala 3.9.0 and the protoc version in `mise.toml`, proven by generating and compiling the IR classes. Errors: if no released combination works, Part B stops there, the spec records the reason, and the other parts still stand.
- **R5:** The gate generates the IR's ScalaPB classes from the unchanged IR schema into the jar the lifter reads, regenerates only when the schema changed, and fetches the plugin through the existing `mise`/scala-cli tooling with no manual install step. Errors: a missing or stale jar fails the gate with the jar and the schema named.
- **R6:** The lifter imports nothing from `com.google.protobuf`, and its build no longer depends on `protobuf-java` or `protobuf-java-util` directly (no error surface beyond the build).
- **R7:** For every checked-in IR file and every expected lifter fixture output, the output of the ported lifter decodes to a protobuf message equal to the one decoded from the file at the commit before the port. A Go test or script using `protojson` and `proto.Equal` proves it, and its command and result are in the done summary. Errors: any unequal file blocks the port; the checked-in JSON is rewritten with `--update` only after all files compare equal.
- **R8:** The lifter's refusals are byte-identical across Part B. The expected refusal texts do not change, and the lifter's refusal and must-not-compile tests pass unmodified. Errors: a refusal that changes text or line is a regression.
- **R9:** The realization emitter stays descriptor-driven. A constructor with no IR message of its name, or a parameter with no field of its name, is refused at its source line as it is today, and the realizations fixture covers it (R7 and R8 prove the positive and negative paths).
- **R10:** The model gate, `make lint-scala`, the Go tests of the Umpire tooling and `make lint-code-fast` pass at the closing task. Errors: a scalafix finding in generated code is fixed by excluding the generated sources, never by a blanket suppression of a rule for the lifter.
- **R11:** The done summary of the port states the lifter's line count before and after, and the count of oneof matches the compiler now checks for exhaustiveness. Errors: if the lifter grows, the task reports the number and the cause instead of claiming a reduction.
- **R12:** The model's README, the lifter's documentation and the gate's own comments describe the layer as it is after this spec: ScalaPB generation, one evaluator, no removed file named. The README stays readable for a newcomer, as fn-115 R24 requires (no error surface).
- **R13:** The baseline is the golden set of fn-115 R2: for every IR file, the tables, Definition IDs, refinement rows, fingerprints, Query answers and lowered Case bytes that Go derives. Every task passes it. Errors: a task that needs a difference states it, amends this criterion's allowed list in the spec, and gets the owner's agreement before landing.
- **R14:** A committed audit maps each munit test under `scala/temporal/test` and `scala/umpire/test` to one of three outcomes: the Go test over the IR that asserts the same claim, a new Go assertion the audit adds, or a recorded reason the claim no longer applies. Errors: a test with no outcome blocks R15.
- **R15:** `scala/umpire` contains no code that builds a table, answers a Query, checks a refinement, composes machines or spells a key or a Definition ID at runtime. `Table`, `Search`, `Keys`, the runtime halves of `Refine` and `Compose`, and `Model.table` are gone. Errors: a declaration the lifter reads by name keeps its name and type so that every lifter fixture lifts as before (R13).
- **R16:** The compile-time guarantees survive R15, each proven by a fixture that must not compile: a step function bound to an action with other inputs, a Query pairing a Property with a Scenario of an unrelated machine, a missing evidence case where evidence is total, and a state type with a non-finite field. Errors: a guarantee that cannot be kept without the runtime code is listed with the Go check that now reports it and the fixture that proves the Go report carries the Scala line.
- **R17:** `UmpireSet`, `Purpose`, `Binding` and `Coverage` are removed, and the done summary lists every set removed with its purpose, bindings and Queries. Errors: if a Go consumer is found to need a set, the task stops and the owner decides between an IR form and removal.
- **R18:** The Nexus caller has no `kernel` package, no `umpire.prelude`, no `NexusActions.scala` and no `NexusKernel.test.scala`. Its domains use `derives Finite`, its step functions use the same `Step` and list forms as every other Model, and the lifter has no `umpire.prelude` case. R13 holds across the move.
- **R19:** No code in `scala/` exists only to reproduce Lean's or Stainless's behavior, and no comment cites a Lean or Stainless file, line or command. fn-115 R25 has already removed every mention of Lean under `model/` by the time this spec runs; this criterion removes the code and the Stainless references. Errors: an ordering or spelling rule that Go reads from the IR and that the lifter must therefore preserve stays, with a comment that names the Go consumer.
- **R20 to R23:** moved to `fn-114-state-every-scala-model-declaration-once` on 2026-10-01 (its R2 to R8). The numbers are not reused.
- **R24:** At the closing task, `scala/umpire` (non-test) is at most 1,300 lines (2,873 today), and the done summary states the line counts of `scala/umpire`, `scala/temporal` (5,062 today) and the Scala tests (982 today) before and after. The 1,300 figure is an estimate from current file sizes. Errors: if the framework ends above it, the summary names what stayed and why instead of cutting a compile-time guarantee to hit the number.
- **R25:** Each task that keeps or writes generic machinery in `scala/umpire` or the lifter records the library it weighed against it and the line counts both ways. A library is adopted where it wins. Errors: a library that a Model under `scala/temporal` could import is not adopted by a task on its own; the task reports the saving and the owner decides.
- **R26:** The lifted IR contains no compiler-synthesized name. A placeholder lambda (`_.facts.contains(x)`, `_ => true`) lifts today with the parameter name `_$1`, which appears 47 times across the checked-in IR and fixture files. The lifter gives such a parameter a stable, readable name. Errors: a name that would shadow one in scope is disambiguated; the baseline (R13) passes, and the IR text changes only in the renamed parameters and their references.

## Boundaries
<!-- scope: business -->

- No change to what any Model says. No new Property, Scenario, machine, fault or assumption.
- No IR schema change. If a part cannot be done without one, the task stops and the owner amends this spec.
- No change to `goir`, `goir/testpilot` or `backends` semantics. They gain tests under R14 and R16 and read the rewritten JSON.
- The archives fn-115 creates (`model0/`, `tools/umpire0/`) are untouched.
- The rewrite of the standalone activity Model and the framework constructs it introduces belong to fn-112. Rolling them out to the other Models and removing the old forms belongs to `fn-114-state-every-scala-model-declaration-once`.
- The lifter's structure is the one fn-115 gives it. This spec changes it only as Parts B, C and D require.
- No build-tool change. scala-cli stays.
- No new library for the Models in this spec. That they stay free of libraries is guidance, and `fn-117-type-the-temporal-api-in-the-models` is where they gain one.

## Decision Context
<!-- scope: both — conditionally substructured -->

**One evaluator.** The first version of this spec kept the native Scala evaluator and listed its removal as a separate decision. The owner then set the direction: Lean is the past, the IRs are the connective tissue, and the Scala layer is optimised for authoring. Under that direction the evaluator is the largest duplication in the tree. Every construct fn-112 adds would otherwise be written twice, once natively and once in the lifter. The evaluator is also already partial, since it refuses every machine that uses a channel, a hole or a monitor. The cost is that a Model error surfaces from the lift-and-check step a few seconds after compiling, with its Scala line, instead of from a munit test. The audit in R14 is what keeps the removal from losing a claim.

**No JSON library.** The review first weighed ujson or circe for `Canonical.scala`. The file has no caller, so it is deleted.

**ScalaPB.** It was first rated marginal because switching the ProtoJSON printer rewrites every checked-in IR file once. Byte stability is no longer required, so that cost is a single `--update` guarded by R7. The expected saving in the lifter is 100 to 150 lines, an estimate from the builder-call counts, which is why R11 asks for the measured result.

**Other libraries.** `sourcecode` would capture names at runtime, and after Part C nothing reads a name at runtime, because the lifter takes it from the `val`'s symbol. shapeless-3 or magnolia would shorten `Finite.derived`, which is about 40 lines today and smaller once it stops enumerating values. Iron would bound integers, and the lifter already reads bounds. None is expected to pay for its wiring. R25 keeps the question open per task instead of settling it here.

**Rejected:** generating both Java and Scala protobuf classes to keep `JsonFormat`. It keeps both dependencies. **Rejected:** replacing the TASTy inspector with another TASTy reader, a rewrite of the lifter with no line saving. **Rejected:** keeping the Stainless-subset kernel in case the proofs return. If they do, they can state the subset they need then.

**Order of work.** A first, since it is safe and unblocks nothing else. Then C and D, which shrink what every later change has to touch. B at any point: `fn-117-type-the-temporal-api-in-the-models` and fn-112 both start after this spec closes.

## Parked unknowns

- Which ScalaPB release and ProtoJSON printer support Scala 3.9.0 and the pinned protoc. R4's proof-of-generation resolves it.
- Whether the owner wants canary and exploratory sets to have an IR form. R17 removes them and records what they said; a later spec can bring them back as IR.
