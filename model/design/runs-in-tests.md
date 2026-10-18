# ModelSuite prototype verdict

The native prototype under [modelsuite-prototype](modelsuite-prototype/README.md) lets a Model
author read examples, explorations, conformance declarations and plain unit tests together.
It adapts the shipped Scala checker. It changes no public DSL, sample, checker, Registry or
realization implementation. Fn-195.2 owns the public implementation.

These are prototype recommendations from the 2026-10-18 experiment. The owner decisions below
remain requirements. A working adapter does not establish a production API or liftability.

## Verdict per current Architecture bullet

| Architecture bullet in fn-195 | Verdict and evidence | Remaining obligation |
| --- | --- | --- |
| ModelSuite, builder and four sections | Keep the four objects and `test("name"):` shape. Register sections on first `munitTests()`, before `super.munitTests()`. Concrete eager helpers are then initialized. Lifecycle probes check repeated and concurrent enumeration, order, transforms, setup, filtering, unit-only suites and registration failures. | The literal superclass-constructor touch is unsafe and is rejected as an implementation strategy. Fn-195.2 must adopt a post-construction enumeration seam. Prototype member checks are runtime checks, not the final source lint. |
| Primary machines use class Model | Keep a fresh stable instance per suite. Bank uses its shipped class and an independent-instance unit control. | Kettle still uses its shipped singleton. Class-only migration and singleton refusal belong to fn-195.6. |
| Authored paths, named record calls, whole actions, oneOf and expect | Keep the named record fields and positioned Boolean assertions. Bank replays the literal retained Quint sendTest. Kettle executes press, 80 heats and failover. oneOf retains both branches in a native ReplayStep. | Prototype action wrappers explicitly append shipped replay operations. They are not automatic lifting of arbitrary statement bodies. before/outcome use a prototype NativeClaims adapter because the shipped claim macros refuse suite context. Intermediate Contract assertions remain held. |
| Explorations, spaces, any, find, sample and refuted | Keep spaces as reusable vals, find's named horizon, seeded sampling and an unflipped refuted result. The small Bank checks overflow within four sends, reproducible nonempty six-step walks and the Leaky supply counterexample. | The prototype's any selects a whole action declaration. Arbitrary per-input any syntax and public replacement of NativeControls remain fn-195.2/.3 work. Sampling proves a finite experiment only. |
| Conformance holds Cases and replays first | Keep conformance beside examples. The prototype replays its authored path and samples through the same native checker. expectation and through are prototype metadata strings that make the opener readable. | This proves a native carrier/replay seam only. It creates no Case, Contract or server Run and validates no realization binding. Typed RunExpectation defaults, unique/multiple realization resolution, capability paths and all binding refusals remain implementation work. |
| oneOf enumerates, any searches, Case names | Keep oneOf's finite native alternatives and confine any declaration to explorations. Prototype registration rejects duplicate slugs. | Stable emitted Case/sample identities are held with Case production. The prototype prints section/slugs only; those are not Case identities. |
| Model loses queries and old live switches; expectation constructors move | Keep questions in the test file. The prototype suites author no queries sections. | No public API is removed in this task. Existing queries, NativeControls, witness APIs and realization constructors are unchanged until fn-195.3. |
| Lifting sections and expectations | Positioned ReplayExpectation values retain a useful future seam. One file is sufficient for this native experiment. | Test-file lifting, plugin/IR changes, Contract lowering, sample Case generation and recorded-Run/server conformance remain held. This prototype earns no lifting verdict. |

The architecture's phrase “three section objects” lists four. This note follows R1 and the owner's
four-section decision.

## Registration findings

The superclass constructor must not invoke subclass section getters. It would run before eager
helper vals in the concrete suite have initialized. MUnit 1.2.0's overridable `munitTests()`
provides the observed boundary after concrete construction and before the inherited snapshot
and suite transforms. `beforeAll` and fixtures occur after discovery and cannot register tests.

The prototype visits examples, explorations, conformance and unit in that order. It holds a
per-instance monitor through registration. Registration moves from new to loading, then ready or
failed. Re-entry while loading fails explicitly. An initializer failure is sticky on later
enumeration, so a partially appended munit buffer cannot become a successful suite. Registration
transforms run once; suite transforms retain MUnit's repeated-enumeration behavior.

Cross-section initialization that registers a different section early is refused. An eager helper
that touches a section before enumeration is also refused. Referencing an already initialized
exploration space from a conformance test body works. The prototype preserves ordinary MUnit
transforms, filtering, ignore/only and setup ordering.

String and TestOptions registrations share the same admission helper for section origin and
duplicate slugs. The original options overload bypassed both checks. Two focused controls
reproduce those failures, including mixed String/options names. The read-only audit is retained
in `.flow/tmp/fn1951/registration-audit.md`.

The concurrency probe compares two submitted enumerations without forcing overlap at a
registration rendezvous. The instance monitor covers both registration and inherited enumeration;
the experiment makes no timing or scheduler-contention claim.

The source Registry has a separate problem. Its pattern recognizes direct FunSuite subclasses.
The inventory experiment uses the actual unchanged Registry on the prototype directory and
records which classes it misses. The explicit run script selects ModelSuite subclasses and
requires tests. Fn-195.2 must fix discovery; passing other native suites does not close that gap.

## expect is still a production blocker

The macro retains every Boolean statement independently with its source line and text. The
deliberately wrong first line followed by a true line is the negative control. The shipped replay
checker supplies the failed expression, line and concrete state.

Scala 3.9 checks unused statements before this ordinary quoted macro can rewrite them. The
unannotated experiment produced E176 “unused value of type Boolean” under the unchanged strict
compiler flags. The prototype suites carry a narrow
`@nowarn("msg=unused value of type .*Boolean.*")` annotation solely for this experiment.
The raw refusal is retained in `.flow/tmp/fn1951/prototype-fourth.log`; the annotations are in
`Bank.test.scala` and `Kettle.test.scala` beside the affected suites.

This suppression is a prototype-only limitation. Fn-195.2 still owes unannotated authored expect
blocks with strict flags intact. A different argument shape or compiler phase may be required.
This prototype does not establish either solution. It also does not provide power-assert
subexpression rendering or prove the general liftable subset.

The read-only compiler-source investigation is recorded in
`.flow/tmp/fn195-prep/expect-warning-seam.md`, against Scala 3.9.0 commit
`777528f19a58e794c9954a42f433373472ec57f8`. It identifies two unproven seams for fn-195.2.
A transparent inline expression annotation might register a caller-span suppression while typer
warnings are buffered, or a bounded transformation could consume the Booleans before typing.
Changing the argument/result type alone does not change typing of earlier statements. The report
ran no compiler experiment and authorizes no public plugin changes. Either candidate still needs
annotation-free source/library controls with all strict flags and unrelated warnings intact.

## Repeated steps and trailing parameters

Choose `repeat(80)(heater.heat)`. The Kettle example executes that spelling from a named room
start at 20 degrees, checks 100 degrees after the heats and checks beeping/off after failover.
The spelling states the count once without implying arithmetic on an action value.
`heater.heat * 80` is rejected as the recommendation; it is not another supported public spelling.

Keep `test("...", from = room, expectation = ...):`. The trailing named parameter sits with
the other run metadata before the body. It avoids an indentation-sensitive suffix after an
expect block. The prototype executes this opener. Its string metadata is scaffolding, not the
settled typed expectation constructors.

## Exact retained Quint comparison

`model/framework/samples/BankTest.qnt.test:100` supplies sendTest. It starts king with 10,000
bananas and burgers, sends 2,000 bananas to donkeykong and checks king=8,000,
donkeykong=2,000, unchanged king burgers=10,000 and donkeykong's denomination keys exactly banana.
It then tries 1,000 burgers from donkeykong to peaches and expects invalid coins or insufficient
funds. QuintBankTest retains those literal inputs and checks; its unchanged-balances assertion
after refusal is additional Scala coverage.

The separate small Bank has supply 2 and bounded inputs. Its exploration, oneOf and conformance
examples are prototype-authored coverage, not literal upstream runs. Sparse-zero cleanup and
self-send ordering remain the intentional differences documented in Bank.scala. Replaying the
literal sendTest establishes that run, not whole-model Quint equivalence.

The retained TwoPhaseCommit.qnt.test successTest expands to prepare then receive for rm1,
prepare then receive for rm2, prepare then receive for rm3, TM commit and RM commits rm1/rm2/rm3.
The current Scala positive test groups prepares before receives. TwoPC is outside this task's
Bank/Kettle scope and remains an exact-run obligation for fn-195.4. Neither retained Quint file
was edited.

## Owner decisions retained

These decisions come from the 2026-10-17 owner interview recorded in fn-195. Prototype choices
above are recommendations and are not new quotations from that interview.

| Decision | Required direction |
| --- | --- |
| Capability expectations | Move expectation bindings into conformance tests; the Model declares the capability. |
| Multiple starts | Name from when init is a set; use the sole start otherwise. |
| Variation | oneOf is the only finite variation; retire runs, edits, dropPrefix and priority. |
| File organization | Try one test file; a separate declarations file is the fallback if lifting requires it. |
| Boolean expectations | Bare Booleans belong in expect only; claim sections keep check for now. |
| Find horizon | find(steps = n) is part of the question; the execution budget remains the caller's. |
| Realizations | Bind automatically with one realization; require through with several and refuse a foreign one. |

## Remaining proofs

Claim-section bare Booleans remain parked pending experience. The one-file native choice does
not resolve future exporter classpath/lifting. All realization execution, Case generation,
Contract lowering, IR/export/schema/backend/generated routes and Temporal sources remain held.
Public DSL dependency placement is fn-195.2 work. The prototype run script supplies MUnit 1.2.0
as an ordinary dependency only to its own isolated invocation.
