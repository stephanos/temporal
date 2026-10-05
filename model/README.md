# The Temporal behavior model

This directory holds a description of how Temporal features are expected to behave, written so
that tools can use it. You describe a feature once, as a small state machine and the promises it
makes. From that one description you get:

- **A check of the description itself.** The Go evaluator walks every path the state machine allows,
  within stated limits, and reports any path on which a promise fails.
- **Generated tests.** A path that shows a promise at work becomes a test that runs against a real
  Temporal server. Nobody writes the test by hand.
- **A judgment of what the server did.** The record of such a test is checked twice: against the
  test's own pass conditions, and against the description as a whole.
- **Second opinions.** The same description is exported to two other checkers, Quint and P, and
  their answers are compared with ours.

Paths on this page are relative to the repository root.

## The terms, with one example

The example is one promise about a Nexus operation, which is a call a workflow makes to a handler
in another service: when the handler answers the call synchronously, the operation succeeds and the
workflow history records its completion. The example runs through the whole page.

A **Model** is the description of one feature: its state machines and everything declared about
them. A **machine** is a finite state machine: its states, the actions that can happen, and a step
function per action that says what the next state is and which facts the step records. The Nexus
caller Model is in `model/temporal/features/nexuscaller`, and its machine `nexusProtocol` is declared in
its feature file, `NexusCaller.scala`, as the object `NexusProtocol`, which is the machine: its rules
say when each action fires and its effects what it does.

A **Property** is one promise a machine makes: a condition its steps must meet. This is the
example's Property (`model/temporal/features/nexuscaller/NexusCaller.scala`, line 476, in
`NexusProtocol.properties`), named after its `val`; inside the machine's object, `property` is the
machine's own:

```scala
/** A synchronous reply settles the operation as succeeded, and the completed event records it. */
val syncSucceeds = property when handler.handlerReply(Reply.syncSuccess) holds { s =>
  s.state.phase == Phase.succeeded && s.records(ProtocolFact.nexusOperationCompleted)
}
```

A **Scenario** is a path through a machine: a start state and a sequence of actions. Here the
caller schedules the operation and the handler replies synchronously.

Each action is written with the party that takes it: the feature declares the caller's actions in
`object caller` and the handler's in `object handler`, so `caller.schedule` is the caller's.
The path starts where the machine does, in `unscheduled`, the state before the operation exists.
`caller.schedule()` leaves each of the operation's three optional timeouts at its first value,
`unset`; `caller.schedule(scheduleToStart := expires)` sets one by name.

A **Query** is a bounded question that joins the two: find a path of this Scenario on which the
Property is put to work, or verify that the Property holds on every path of it (line 559, in
`NexusProtocol.queries`, without its exploration settings). A Scenario one Query uses is written
inside it, under its name:

```scala
val syncCompletion = (query find properties.syncSucceeds in scenario("syncReplied").actions(
  caller.schedule(),
  handler.handlerReply(Reply.syncSuccess)
) limits two total 384)
  .expect(satisfied)
```

The path a `find` Query returns is its witness. `limits two` bounds the search to two steps.
`total 384` is the author's count of the Query's static combinations, which the reader checks; see
[Counting a Query's total](#counting-a-querys-total).
`.expect(...)` states what a real run of this path should be judged as, here `satisfied`, the
Temporal kit's expectation (`model/temporal/realize/Kit.scala`) of a Run that completes, cleans up
and satisfies its Contract, and that the model assessment finds conformant with the Property
satisfied; see [Following the example to a Verdict](#following-the-example-to-a-verdict).

A **realization** says how to act a path out on a real server: which API calls perform each
action, and which recorded events are evidence of each fact. The Nexus caller's realization is
`model/temporal/features/nexuscaller/Realization.scala`.

The remaining terms name what the tools produce:

- The **IR** (intermediate representation) is a language-neutral file format that tools read
  instead of source code. There are two, described in the next section.
- **Lifting** turns the compiled Scala Models into the first IR.
- **Lowering** turns a Query's witness, through its realization, into a Case.
- A **Case** is one executable test: a Program of instructions to run against Temporal, and a
  Contract of rules that the recorded evidence must satisfy.
- A **Run** is the record of one execution of a Case: an ordered list of events.
- A **Verdict** is the Contract's conclusion about one Run: `satisfied`, `violated` or
  `inconclusive`.

## The layers

Scala declares a Model, the IR generator reads its compiled declarations, and Go evaluates the resulting
IR. The Scala DSL supplies types and step functions for authoring; Go is the single Model evaluator.

```mermaid
flowchart TD
    scala["Scala Models<br/>model/temporal, written with model/umpire"]
    uir[("Umpire IR<br/>model/ir/*.json")]
    reader["Tables, Query answers, witnesses<br/>tools/umpire/model"]
    tir[("Testpilot IR: Cases<br/>model/cases/*-case.json")]
    run["Run and Contract Verdict<br/>common/testing/testpilot"]
    assessment["Model assessment<br/>tools/umpire/conformance"]
    other["Quint and P<br/>tools/umpire/export"]

    scala -->|"lifting: model/irgen"| uir
    uir -->|"reading and checking"| reader
    reader -->|"lowering: tools/umpire/lower"| tir
    tir -->|"running against Temporal"| run
    run -->|"assessing"| assessment
    uir --> assessment
    uir -->|"export"| other
```

Two IRs sit between the layers, and each has one writer side and one reader side:

- The **Umpire IR** is a lifted Model. Scala writes it (the IR generator); Go reads it (the reader,
  lowering, conformance, export and exploration under `tools/umpire`). Its schema is
  `proto/internal/temporal/server/api/umpire/v1/ir.proto`, and the checked-in files are
  `model/ir/*.json`. Go never runs Scala code: everything it knows about a Model comes from
  this file.
- The **Testpilot IR** is the Case, Run and Verdict format. Lowering writes Cases; Testpilot, the
  Case runtime, reads them and writes Runs and Verdicts. Its schema is
  `proto/internal/temporal/server/api/testpilot/v1`. Testpilot knows nothing about Models.

| Layer | What happens | Module | Command |
| --- | --- | --- | --- |
| Authoring | Models are written in Scala with a small DSL (a library of declarations such as `machine`, `property` and `query`) | DSL `model/umpire`, Models `model/temporal` | `make lint-model`, `make fmt-model` |
| Lifting | The compiled Models are translated to the Umpire IR, built as the ScalaPB classes of its schema and written as ProtoJSON. A construct outside the supported subset is refused at its source line | `model/irgen`, run by the gate `model/check` | `make umpire-gen-model` writes `model/ir`; `make umpire-check-model` requires it to be current |
| Umpire IR | The checked-in lifted Models | `model/ir`; schema in `api/umpire/v1` | `make protoc` after a schema change |
| Reading and checking | Go loads and validates the IR, builds each machine's table, and answers every Property, Query and refinement | `tools/umpire/model` | `go test -tags test_dep ./tools/umpire/model/...` |
| Lowering | A `find` Query's witness becomes a Case through its realization | `tools/umpire/lower` | `make umpire-gen-cases` writes `model/cases`; `make umpire-check-cases` requires it to be current |
| Testpilot IR | The checked-in Cases, and `manifest.json`, which accounts for every Query | `model/cases`; schema in `api/testpilot/v1` | |
| Running | Testpilot admits a Case, runs its Program through the Temporal Driver, records the Run and evaluates the Contract into a Verdict | `common/testing/testpilot`, `common/testing/testpilot/temporal` | `make umpire-check-live-tests` (in-process server); `make umpire-run` builds `.build/umpire-run` for any deployment |
| Assessing | The Run's evidence is compared with the whole Model | `tools/umpire/conformance` | part of the live tests above |
| Export | The IR is written for Quint and P, and their answers are compared with Go's | `tools/umpire/export` | `make umpire-check-backends` (needs the tools its README names) |

## Running the gate

The gate is the one command that checks the whole model pipeline:

```sh
make umpire-check-model
```

It is a Scala program, `model/check`. In order, it checks that the files under `model/` keep to the
model's own vocabulary, compiles and tests the DSL and the Models, runs the IR generator's own tests,
lifts every IR file the Models declare in one IR generator run (which first holds every Model to its
declaration order, [below](#the-reading-order-and-its-lint)), requires every file of `model/ir` and
`model/cases` to equal what it just produced, lints every IR file, and runs `go vet` and `go test` over `./tools/umpire/...`. It stops at the first failure
and changes no checked-in file. scala-cli, the JDK, protoc and Go come from the repository's
`mise.toml`.

After you change a Model, regenerate the checked-in files with the same program:

```sh
make umpire-gen-model     # the gate with --update: rewrites model/ir and model/cases
```

Then read the diff of `model/ir` and `model/cases` like any other code change.

The gate then lints every IR file (`make umpire-check-lint` runs the lint alone): what a Model
declares that nothing reaches, takes, asks, evidences or realizes, and its specification holes, each
at its Scala line. It fails on a finding until you fix the Model or accept the finding with a reason
in `model/ir/<file>.lint.json`, and on an acceptance that no longer matches a finding. The coverage
summary it prints per machine is informational; no count fails the gate.

Three kinds read a machine's table for what can happen in its reachable states, each at the Scala
line of what it names:

| Kind | Reported for | Fix |
| --- | --- | --- |
| `never-enabled` | an action class no reachable state enables | bind a rule that enables it, or drop it from the machine |
| `silent-rejection` | a party's action disabled in a reachable state that is no end, by class and phase | a rule whose outcome says how the system answers it |
| `stuck-state` | a reachable state that is no end and enables no action class, a timer's and an internal step's included, with a shortest path to it; a state with a hole is not stuck, since the hole declares that unmodeled behavior may happen there | the rule a timer or an internal step is missing, or the state in the machine's `ends` if it is final |

Like `never-enabled`, `stuck-state` reads each machine's table, not a composition's composed table.

Lint reads the law sidecar beside an IR file (`model/ir/<file>.laws.json`) for the law kinds, each at
the Scala line the sidecar records:

| Kind | Reported for | Fix |
| --- | --- | --- |
| `waived-law` | a law a declaration waives with `except` or `overriding` and a reason | nothing: the gate's update forwards the reason into `<file>.lint.json`, keyed `<machine>.<law>` |
| `law-waived-without-reason` | a waiver with an empty reason (the IR generator refuses one, so the sidecar was edited) | regenerate the sidecar |
| `reason-names-no-law` | a waiver of a law the sidecar's catalog does not bring | remove the waiver, or declare the capability that brings the law |
| `parameter-without-citation` | a binding of a parameter its law lists in `Law(parameters = …)`, written without `cited(value, "<server file>")` | cite the server code that answers it so |
| `law-with-one-instance` | a catalog law fewer than two machines with their own state types instantiate, across every sidecar of `model/ir` | take the law out of the catalog until a second machine declares it |

The waivers' reasons have one source, the sidecar: `make umpire-gen-model` rewrites each forwarded
`waived-law` acceptance from it, after the acceptances an author wrote, and a check fails on a
`<file>.lint.json` that does not carry them, and on a forwarded acceptance whose waiver is gone.

Each file of `model/ir` is declared once, in Scala, beside the Models it holds, in the feature
file's `object exports`: its name and its roots, named by value, in a `val` named after the file.

```scala
object exports:
  val nexusControl =
    irFile("nexus-control")(ForgedCompletion.queries.forgedCompletion, NexusRealization.forgedCompletion)
```

A root is a machine, a composition, a Query, a list of Queries, a progress claim or a realization;
the file holds it and everything it reaches. A root that names nothing does not compile. A
declaration may be a root of several files and is lifted into each, and one that no file names
stays out of `model/ir`, so a design can be kept out of the checked files. The IR generator reads the
compiled Models once and lifts every file apart, with nothing carried from one to the next; a
refusal follows a line naming the file it was lifting. `lift --ir <jar=prefix> <classpath file>
<directory> [name...]` is that run, and `lift <jar=prefix> <classpath file> <out.json> <root>...`
lifts the roots named by their fully qualified names, as the IR generator's fixtures do.

The gate packages the linked Temporal API, Testpilot and well-known ScalaPB classes in
`model/build/api-scalapb.jar`. Its descriptor and tool stamp is checked before a build; editing a
Model reuses that jar. If the gate reports a missing or stale `proto/api.binpb`, run the named
`make proto/api.binpb` target; if it reports stale generated internal protobuf code, run
`make protoc`. The gate reports the declaration's source line when a typed selection cannot be
lifted. The compiler reports misspelled fields, wrong request or response roots and wrong value
types before lifting.

`make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks` leaves out the `go vet` and `go test`
step, for when you run the Go tests separately; it still lints the IR. `go test -tags test_dep ./tools/umpire/...` runs the
Go side alone from the checked-in IR and needs no JVM.

## Following the example to a Verdict

**Lifted.** `make umpire-gen-model` lifts the Query into `model/ir/nexus-caller.json`. This is its
entry there, without the `exploration` field:

```json
{
  "name": "syncCompletion",
  "position": {
    "file": "model/temporal/features/nexuscaller/NexusCaller.scala",
    "line": 585
  },
  "form": "FORM_FIND",
  "property": {
    "machine": "nexusProtocol",
    "name": "syncSucceeds"
  },
  "scenario": {
    "machine": "nexusProtocol",
    "name": "syncReplied"
  },
  "limits": {
    "name": "two",
    "steps": 2,
    "actions": 2,
    "search": 512
  },
  "expectedRun": {
    "property": "OUTCOME_SATISFIED",
    "conformance": "CONFORMANCE_CONFORMANT",
    "contract": "OUTCOME_SATISFIED",
    "disposition": "DISPOSITION_COMPLETED",
    "cleanup": "CLEANUP_SUCCEEDED"
  }
}
```

The Property, the Scenario, the machine and its step functions are lifted into the same file the
same way, each with its Scala source position. Step functions become expression trees, which
[SEMANTICS.md](SEMANTICS.md) defines how to evaluate.

**Checked.** The Go reader builds the table of `nexusProtocol` from the IR and answers the Query:
it finds the two-step path and confirms that `syncSucceeds` holds on its last step. That path is
the witness.

**Lowered.** `tools/umpire/lower` turns the witness into the Case
`model/cases/nexus-caller-syncCompletion-case.json`, with the Case ID
`temporal.case.scala.nexus-caller.syncCompletion`. Its Program has three parts: a workflow that
schedules a Nexus operation, a handler that replies synchronously, and a controller that starts
the workflow and reads its history. Its Contract holds one rule. This is the rule, with the two
comparisons left out:

```text
"ruleId": "fact-nexusOperationCompleted",
"trigger": { … },
"response": { … },
"clock": "CORRELATED_CLOCK_OPERATION_TRANSITIONS",
"bound": "1",
"ending": "TRACE_ENDING_PARTIAL"
```

The trigger matches the step whose action is `schedule-unset-unset-unset`, and the response matches
a step whose fact is `nexusOperationCompleted`. In words: once the Run shows the operation
scheduled, the completed fact must follow within one more transition of that operation.
`model/cases/manifest.json` lists the Query as `lowered` and repeats what `.expect(...)` declared.

**Run.** `tools/canary/assessment/testdata/nexus-caller-syncCompletion-run.json` is a recorded Run
of exactly this Case against the in-process test server. It holds 27 events and ends with this
Verdict:

```json
{
  "status": "VERDICT_STATUS_SATISFIED",
  "rules": [
    {
      "ruleId": "fact-nexusOperationCompleted",
      "status": "RULE_VERDICT_STATUS_SATISFIED",
      "terminalStateId": "correlated.satisfied",
      "supportingEventSequences": ["9", "19"]
    }
  ],
  "supportingEventSequences": ["9", "19"]
}
```

Event 9 is the evidence that the operation was scheduled, and event 19 is the history's
`EVENT_TYPE_NEXUS_OPERATION_COMPLETED` event. To produce such a Run yourself,
run every lowered Case against an in-process server:

```sh
go test -tags 'test_dep integration' ./tests -run '^TestTestpilotGeneratedCases$'
```

**Two judgments, kept apart.** The Verdict above is the *Contract Verdict*. It says the Case's own
rule was met, and nothing more: the rule checks that the completion was recorded, not that the
operation's phase was `succeeded`, because a Property's clause about one field of the state does
not lower to a Contract rule yet.

The *model assessment* is the second judgment. It asks whether some execution the Model allows
explains all the evidence in the Run (conformance: `conformant`, `nonconformant` or
`inconclusive`), and what the Query's Property is on every execution that does (`satisfied`,
`violated` or `inconclusive`). The assessment is not stored in the Run. For `syncCompletion` the
declared expectation is a completed Run with a succeeded cleanup and a satisfied Contract, assessed
`conformant` and `satisfied`, and `TestTestpilotGeneratedCases` requires exactly that of a live
Run and of its replay, each value by equality (`lower.ExpectedRun.Check`). Conformance short of
`conformant` names the judge's reason the same way, as `forgedCompletion` declares
`conformanceReason = Some(Reason.incomplete)`: it is set exactly when conformance is not
conformant, the reader refuses an expectation that breaks that either way, and `Check` compares it
by id.

**What is not supported.** These limits are current and recorded, not hidden:

- `syncCompletion` is the only one of the Nexus caller's seven `find` Queries whose Property the
  assessment settles. The other six Cases run and satisfy their Contracts, but their Property stays
  `inconclusive`, because the Model also allows another execution that explains the same evidence
  and on which the Property fails or is never evaluated. Each Query's `.expect(...)` names the
  reason by the judge's id (`Reason.explanationsDisagree`, `Reason.neverEvaluated`), whose wording
  is the judge's own (`tools/umpire/conformance/conclude.go`).
- Of the 264 Queries in the checked-in IR, 16 lower to a Case. 150 are `verify` Queries, which have
  nothing to run. 95 belong to machines that declare no realization yet. 3 standalone activity
  Queries are `unsupported`: their path needs something a Case cannot do or record in order yet,
  and the manifest names it at its source line.
- A Run samples one execution of the server. A satisfied Verdict is evidence about that Run, not a
  proof about every schedule.
- The recorded Run above was made on a test server. No Run of a real deployment is checked in.

## Where things are

| Path | What it holds |
| --- | --- |
| `model/umpire` | The DSL: what an author writes a Model with. It names no Temporal concept. Realization declarations any system needs, the open traits a system's kit extends and the script helpers are in `umpire/realize` |
| `model/temporal` | The Models: `features`, one folder per feature (`nexuscaller`, `nexusoperation`, `standaloneactivity`); `shared`, the entities features compose (`taskqueue`, `worker`); `capabilities`, the laws stated once for every entity; and `realize`, Temporal's realization vocabulary (`Realize.scala`) and the shared Temporal realization kit (`Kit.scala`) |
| `model/irgen` | The IR generator. `testdata` holds Models it must lift and Models it must refuse |
| `model/ir` | The checked-in Umpire IR, one file per `irFile` the Models declare |
| `model/cases` | The checked-in Cases and `manifest.json` |
| `model/check` | The gate program. It generates the IR's ScalaPB classes from the schema into `model/build` (`--generate-ir`) and runs the IR generator once over every IR file the Models declare. Its second entry point, `umpire.check.metrics`, prints the source metrics of the Model folders it is given |
| `model/project.scala` | The build settings of the DSL and the Models |
| [SEMANTICS.md](SEMANTICS.md) | The evaluation rules of the Umpire IR: what every construct means |
| [Known bugs](../.plans/UMPIRE4_VISION.md#known-bugs-knownbugs) | Vision for acknowledging a known bug; not implemented |
| `model/build` | Build output of the gate; ignored by git |

The framework, `model/umpire`, names no Temporal concept, in its prose or its identifiers, so a
Model of another system could be written in it. What a system has of its own its realization kit
declares, by extending the framework's open traits `Addressee`, `Activation`, `Instruction`,
`Recorded`, `Setting`, `Behavior` and `SystemStep`. Temporal's are in `temporal.realize`: `Role` and
`RoleKind`, the worker activations `WorkerActivation.{Workflow, NexusHandler, Activity}`, the
instructions `WorkerInstruction.{AttemptFailure, AttemptCanceled, Fault, WorkflowCommand,
NexusReply, NexusCompletion}` with `FaultKind`, the history read `WorkflowHistory.event`, the
dynamic-configuration `RequiredSetting`, and the API behavior hints `ApiBehavior` and `ServerStep`
with `WaitBound`, `Visible`, `CauseKind`, `AttemptNumbering` and `InstructionLimit`. `TestFrameworkNamesNoTemporal` in `tools/umpire/model`
fails when a file under `model/umpire` names a Temporal term, the six capability kinds among them; a
mention stays only under an allowance that states its reason, and none has one today. The
tooling downstream of the DSL is Temporal's driver tooling by design: the IR generator matches the kit's
vocabulary by fully qualified name and writes it into the IR's realization messages, whose names are
Temporal's, and lowering turns a Query into a Testpilot Case the Temporal Driver runs.

The module map, [.plans/UMPIRE_MODULES.md](../.plans/UMPIRE_MODULES.md), states each module's job,
its public interface and what it may import. Each module outside this directory has its own README:

- [tools/umpire](../tools/umpire/README.md): the Go tooling that reads the Umpire IR, and its commands.
- [tools/umpire/export](../tools/umpire/export/README.md): the Quint and P comparison.
- [common/testing/testpilot](../common/testing/testpilot/README.md): the Case runtime and the Testpilot IR.
- [common/testing/testpilot/temporal](../common/testing/testpilot/temporal/README.md): the Temporal Driver.
- [tests/testcore/testpilot](../tests/testcore/testpilot/README.md): the Cases the functional tests pin.
- [tools/canary](../tools/canary/README.md): the canary, which runs one pinned Case against a
  production deployment when an operator dispatches it.

## Writing a Model

`make lint-model` checks formatting and lint rules for the DSL, the Models, the IR generator and the
gate; `make fmt-model` formats them and `make fix-model` applies the lint rewrites. A construct a
lint rule forbids where no rewrite keeps the behavior carries a line-scoped
`// scalafix:ok <rule>`.

Model lint reads what the IR generator made of a Model, not its Scala. After `make umpire-gen-model`,
`make umpire-check-lint` lints every file of `model/ir` as the gate does: each finding at its Scala
line, then each machine's coverage summary. `go run ./tools/umpire/cmd/umpire-lint --tables
model/ir/<file>.json` adds each machine's table by class in the terms of model/SEMANTICS.md
("Modalities"), each cell MAY with its results, MUST NOT with its guard or `?` for a hole, beside the
claims that pin it. Fix a finding in the Model, or accept it with an entry of
`model/ir/<file>.lint.json` that names its `kind`, its `owner`, its `subjects` as lint prints them
and, under `because`, the reason; the gate fails on a finding neither fixed nor accepted and on an
acceptance that matches none.

A Model writes a declaration's type only where inference would give a different one, such as a
step function whose body is `disabled` or `stay(s)` (inferred with no facts), a `Long` written as
an `Int` literal, or a call whose type argument or `given` the expected type decides, or where the
IR generator needs it; the IR generator reads an inferred type as it reads a written one.

The IR generator reads what an author wrote, as written:

- **Declarations:** machine objects, `object M extends Machine[S, O, F]`, with their header members
  (`init`, `end`, `entity`, `evidence`, `unobservable`) and their sections (`states`,
  `refinement`, `effects`, `monitors`, `rules`, `properties`, `implements`, `queries`); derived
  machines, `object D extends Derived(m.op(…))`, of the derivations `restrict`, `rebind`,
  `extend`, `refining`, `assuming` and `unmonitored`; composition objects; action chains
  (`action`, `timer`, `internal`, `on`, `creates`, `input[T]`, `schema`, `results`, `example`);
  input tokens, `val scheduleToStart = input[Timeout]` declared on an action with
  `.input(scheduleToStart)`, each input named after its token's `val`; classes, `flip`, the
  positional `start(unset, expires, unset)` and `start()`, every input at its domain's first value
  (`start(unset, unset, unset)`, and for an action with no input its one class);
  Properties, Scenarios, Queries, Limits, monitors, assumptions, holes, channels, compositions,
  progress claims and realizations. A composition names its members by field selector
  (`_.activity -> CurrentRecord`), and so do its `sync`, `replaces` and `withMember`; a `sync`
  with no name is named after its first member's action; a Scenario
  or `whenAction` of it names a sync by one member action it pairs, `c.synced(_.activity ->
  history.dispatch)`, and a member's own action by `c.own(_.activity, caller.control(Control.pause))`.
- **Shared claims:** `property` and `scenario` are declared on `Declares[S]`, the supertype of a
  machine and a composition, so one function over `m: Declares[S]` declares a Property on either.
  A function-valued argument of such a function names a def of the lifted sources, which the
  IR generator binds where the function is called, as it binds a machine argument; a type parameter
  reads as the type the call applies it to. A case class whose every field is a Property, Scenario
  or Query bundles claims: built by its constructor in such a function, it lifts as its claims, and
  `x.field` reads one back. The laws such functions state once for every entity are lifted sources
  too, in `model/temporal/capabilities`; the framework holds none.
- **Types:** enums with and without case fields, case classes, and bounded counters `UpTo[N]`: a
  field `attempts: UpTo[2]` has the values 0, 1 and 2, lifted as the IR int range 0..2, and a step
  writes one with `UpTo(n)`. It replaces integer fields bounded by the per-record
  `given Finite[Int] = Finite.upTo(…)` beside the state's `Finite`, which Models not yet migrated
  keep.
- **Step functions:** `if`, `match` with case, alternative, binding and wildcard patterns, local
  `val`s, `copy`, constructors, comparisons, arithmetic, list literals and `++`, and calls of other
  functions; `Step(…)` and `steps.because("…")` on steps written out, and named choices,
  `choose(committed -> …, redelivered -> …)` over tokens `val committed = choice`. `require`
  becomes the function's precondition; `ensuring` is not lifted. Only a step function, and a
  function it calls that gives steps, makes a step: the IR generator refuses `Step(…)`, `enter`, `stay`
  or a call of a step function at its line in a start, an `ends`, evidence, a refinement, a
  monitor, a Property, a progress claim or a Scenario's start (model/SEMANTICS.md, "Levels").
- **Realization script helpers** (core, `umpire/realize/Scripts.scala`): `script(id, activation)`
  of `everyCase(command)`, `onPath(classes*)(command)` and `perform(step -> command, …)` items;
  `command(instruction, …)`; `rpc(role, method) { … }`, `readUntil(…) { … }` and
  `call.withFields { … }`, which appends assignments and keeps the call's name;
  `statusTable(fact -> value, …)`. A command is named after its `val` in kebab case
  (`val pauseActivity` is `pause-activity`) unless written out as `Command(id, …)`. A declaration
  referred to by value is written as its id, a monitor as its name
  (`MonitorExpectation(terminalFinality, …)` in a Query's expected Run), and a fact as
  its enum case, or the companion of a case with fields. A lookup `table(fact)` is resolved when
  the IR generator lifts. The IR generator refuses a command no `val` declares, a `perform` or `onPath` with no
  class, a lookup of a fact its table lists twice or not at all, a request-scope line of an `rpc`
  or a `readUntil` that is neither `field(_.x) := v` nor `Assignment.typed(…)`, and evidence that
  records a case of another enum than the facts its machine records.
- **Sugar** (`umpire/Syntax.scala`, lifted by `irgen/Syntax.scala`): `enter(state, facts*)`,
  `stay(s)`, `disabled`, `x.in(a, b, …)`, `a implies b` and `after.records(fact)`. Each is lifted to
  the IR its core form lifts to, and nothing else: `List(Step(accepted, state, List(facts*)))`,
  `List(Step(accepted, s))`, `Nil`, `List(a, b, …).contains(x)`, `!a || b` and
  `after.facts.contains(fact)`. `enter` and `stay` answer the outcome one
  `given Ok[Outcome] = Ok(Outcome.accepted)` names; `in` is written dotted and takes at
  least one member; `implies` reads its right side only where its left side holds. A composition's
  `after.records(_.member, fact)` lifts as `after.facts.contains("member_fact")`, the composed key
  it records. A named call `start(scheduleToStart := expires)` lifts as the positional
  `start(unset, expires, unset)`. The kit's `field(_.name) := operand`, in
  `temporal/realize/Syntax.scala`, lifts as `Assignment.typed(Field[Req, V](_.name), operand)`,
  where `Req` is the request type of the scope the enclosing `rpc` or `readUntil` opened.
- **Claim patterns** (sugar, the same files): `once(over).keeps(_.x)`, `never(to)`,
  `never(to).from(before)`, `stays(p)` and `stays(p).unless(release)` on a Property builder, each
  lifted to the Property its lambda declares: `holdsAcross((before, after) => !over(before) ||
  after.state.x == before.x)`, `holds(after => !to(after))`, `holdsAcross((before, after) =>
  !before(before) || !to(after))`, `holdsAcross((before, after) => !p(before) || p(after.state))`
  and the same with `|| release(after)`. Each predicate is lifted as `holds` lifts its lambda and
  called from the Property's function; `keeps` takes a field path, or a def whose body is one.
- **The monitor pattern** (sugar, the same files): `val retainedOutcome = sticky(outcomePreserved)`
  and `val ownerAcknowledgment = stickyAcross(ackOnlyWhenKept)` declare a monitor of a promise that,
  once a step breaks it, stays broken, named after its `val`. Each lifts as the monitor
  `monitor[S, O, F, Boolean](false)((broken, before, after) => broken || !p(after))(broken =>
  broken)`, with `!p(before, after)` for `stickyAcross`, read after every step. A promise about
  another state type than the monitor's does not compile. A monitor that needs history, such as
  one counting the outcomes recorded, is written with `monitor`.

A declaration takes its name from the `val` or object that declares it, and its family from the
`given Family` in scope. A machine is an object named after it, with its first letter lowered,
which states its three types once, in `extends Machine[S, O, F]`; its members write the state type
as `State`:

```scala
given Family = Family("fixture.store")

val put = action(Party("client"))
object Store extends Machine[StoreState, Outcome, Fact]:            // the machine `store`
  val init = StoreState(Kept.nothing)
  def end(s: State) = s.kept == Kept.stored
  object effects extends Section:
    def keep(s: State) = enter(s.copy(kept = Kept.stored), Fact.stored)
  object rules extends Rules:
    when(s => s.kept == Kept.nothing)(put ~> effects.keep)
  object properties extends Section:
    val putStores = property when put holds (after => after.records(Fact.stored))
  object queries extends Section:
    val putStoresOnce = query find properties.putStores in scenario("putOnce").actions(put) limits two total 2
object PutOnly extends Derived(Store.restrict(put))
val two = Limits(steps = 2, actions = 2, search = 64)
```

The same holds for `timer`, `internal`, `monitor[S, O, F, M](initial)…`, `assume`, `hole`, `channel[M](capacity = …, …)` and
`Realization(machine = …, …)` without `name`, and for `Party()`, `Entity(key = …)` and
`Observation(on = …, read = …)`, whose `name`, read as `attemptCount.name` in an evidence line or a
kit body, is their `val`'s.

A name is written as a string only where it differs from the `val`'s, or where no `val` declares
it, and each kind has one form for that: `action(name, party)`,
`monitor[S, O, F, M](name, initial)`, `assume(name)`, `property(name)`, `scenario(name)`,
`query(name)` and `Limits(name, …)`. A machine, timer, internal step, hole, channel, derived machine and
composition has none: each is named after its object or `val`, and a composition names its members, syncs and
Scenario classes by field selector, never by a string key. A progress claim is always named,
`m.leadsTo(name)(…)`. A Property or Scenario with no `val`, built in a list or in a function over a
machine argument, keeps `property("…")` or `scenario("…")`, except where the function's body ends
in it and a `val` declares the function's call: it then takes that `val`'s name, as a law's instance
does (`val terminalStays = terminalStatesAreFinal(m)(Admission.phase, Admission.terminal)`). A Query
that neither a `val` nor `query("…")` names is named
`<machine>.<scenario>.<property>`, after the machine its Scenario is declared on, its Scenario and
its Property: `query verify notPaused in any` over a design `m` is `m.any.notAdmittedWhilePaused`.
Any other captured form with no `val`, or with a name the compiler made up such as an anonymous
given's, is refused at its line. So are two declarations that would share a name: two machines or
compositions, two Properties or two Scenarios of one machine, two Queries (two unnamed Queries of
one Scenario and Property among them), two syncs of one composition, monitors, assumptions, holes,
channels or realizations, Limits of one name with different bounds, and two actions one machine
binds. A monitor a Query's expected Run names by value is refused where no `val` declares it or the
Query's machine does not watch it, and a `given Family` whose root is not a string literal.

A feature reads top to bottom in one feature file per folder, named after the folder, beside its
`Realization.scala` and its tests; its larger subjects are subfolders laid out the same way. Every
Model folder is laid out so, and the standalone activity is the example:

```text
features/
  standaloneactivity/
    StandaloneActivity.scala   types, signature; ActivityProduct, ActivityProtocol, ActivityWorker, StandaloneActivity; exports
    Realization.scala          the realization
    record/
      Record.scala             the record and its designs: CurrentAdmission, StaleAdmission, HeldAdmission, AdmissionResponseLoss
    withTaskQueue/
      WithTaskQueue.scala      the designs over the task queue: CurrentRecord, StaleRecord, CurrentOverQueue, …
  nexuscaller/
    NexusCaller.scala          NexusProduct, NexusProtocol, HandlerWorker, NexusCaller, ForgedCompletion; exports
    Realization.scala
    closepolicy/
      ClosePolicy.scala        RejectAfterClose and the nine designs derived from it; exports
  nexusoperation/
    NexusOperation.scala       NexusOperation; exports
    Realization.scala
shared/
  Bounds.scala                 the bounds more than one folder's Queries run under
  taskqueue/
    TaskQueue.scala            DispatchQueue, the opaque contract, and its storage-loss variant; MatchingQueue,
                               the provider that refines it, and the lossy, forgetful and volatile providers
  worker/
    Worker.scala               Polling
```

A Model folder holds no file named by kind (`Model.scala`, `Properties.scala`, `Queries.scala`,
`Capabilities.scala`, `IrFiles.scala`): `TestRetiredModelPathsStayRetired` in `tools/umpire/model`
fails on one, and on a live file that names one. A bound that two folders run their Queries under
is declared once, in `shared/Bounds.scala`, and keeps its name, which a Query's receipt reads; a
bound whose name another folder gives another budget stays in its own feature file.

A feature file reads in this order:

1. a header comment naming its objects in order, the imports, the file's `DefinitionScope` pin and
   its family;
2. its types: every enum, state case class and type alias, at the top level, so each keeps its
   `pkg.Type` IR name;
3. its signature: entities, inputs, the actor and section objects that hold its actions and
   timers, observations, choices, bounds and the top-level `given`s;
4. one object per machine or composition, in dependency order: a refined machine before the one that
   refines it, a base machine before those derived from it, the members of a composition before it;
5. `object exports`, its `irFile` roots.

A machine object is the machine: `object ActivityProduct extends Machine[ProductState, Outcome,
ProductFact]` (`umpire.Machine`), named after its object with the first letter lowered
(`activityProduct`). It reads in this order:

1. its header: `val init`, the state it starts in (`init` as Quint and TLA+ name it), `def end(s)`,
   the states it may end in, and where declared `val entity`, `val evidence` and, for a machine that
   refines nothing, `val unobservable`, its timers whose step records nothing a Run can read;
2. `object states`, its vocabulary: the named state sets and projections its guards, effects and
   claims read (`states.terminal`, `states.held`) and its constants;
3. `object refinement extends Refinement(ActivityProduct)`, where it refines another machine:
   `toProduct`, the map onto the refined machine's states, and where declared `visible`,
   `visibleOutcomes` and `unobservable`;
4. `object effects`: what each action does, plain defs named for it (`startAttempt`, `complete`),
   which never give `disabled` or `Nil`;
5. `object monitors`: the monitors that watch it and the assumptions it makes; an assumption no
   machine makes of its own, which a derivation adds with `assuming` or a progress claim names with
   `under`, sits in the feature's signature;
6. `object rules extends Rules(_.phase)` (`umpire.Rules`): when each action fires, one rule per line
   under a heading, `in(scheduled) { worker.attemptStart ~> effects.startAttempt }` or `when(g) {
   … }`; a rule fires a whole action or one class of it, `caller.control(Control.pause) ~>
   effects.pause`, and `disabled(process.workerStop)` binds an action no state enables. The rules
   of one action class hold in no common state: the rules object refuses an overlap as it is
   constructed, over every state and class, naming the machine, the class, both rules and a
   witness state, and the gate constructs every IR file's roots (`model/temporal/IrFiles.test.scala`).
   Each action's rules lower to one step function, `<machine>.rules.<action>`;
7. `object properties`, its claims, which name the machine implicitly: `property when … holds …`;
8. `object implements`, its capabilities, `val all = capabilities(limits = three)(…)`;
9. `object queries`, its Scenarios, then its Queries; a Scenario one Query uses is written inside it,
   `scenario("completed").actions(…)`.

A derived machine is an object too, `object StaleAdmission extends Derived(CurrentAdmission.rebind(
…))`, and adds only its own `states`, `properties`, `implements` and `queries`; `rebind(action ~>
effect)` keeps that action's rules and replaces their effect, and `rebind(when(g) { … })` replaces
its rules. A composition is `object CurrentOverQueue extends Composition[OverQueue](_.activity ->
CurrentRecord, _.queue -> DispatchQueue)` with its `def end(s)` and `object syncs extends Syncs`;
one with a member replaced is `object StaleOverQueue extends Composition(CurrentOverQueue.withMember(
_.activity -> StaleRecord))`. A machine object that holds a monitor, assumption, hole or channel
pins its former owner, as `record/`'s `CurrentAdmission` pins `…System$package$` beside its file's
own pin of the same owner. A section object is initialized on first use: an `implements` that reads
the realization, as the protocol's `Describable` does, leaves the machine object free for the
realization to read. A machine that is a failure model, the real design under a fault the
environment can cause, mixes in `FailureModel`, and a negative control, a deliberately wrong design
the checks must refuse, `NegativeControl` (`object StaleAdmission extends Derived(...),
NegativeControl`); the IR generator holds each to what it is for:

- a negative control is something the run checks can refute: a Query of it that is a `verify` or
  whose Run is expected violated, or a refinement check, of a refinement it keeps from the design
  it derives from or of a composition member standing in for another machine; nothing refines it,
  and it declares no refinement of its own;
- a failure model binds a fault, an action of the party `fault` or of a `faults` section that some
  state enables (a composition, through its members), and not every Query of it expects its Run
  violated;
- a machine or composition object that binds a fault and is marked neither is refused.

These rules are weak by construction. The IR holds no expected check answer, so the IR generator
can see only that a check able to refute a negative control exists, not that it does refute it; the
Go tests that pin each Query's answer and each refinement's receipt (`tools/umpire/model`, such as
`activity_system_test.go` and `nexus_close_baseline_test.go`) are the guarantee.

The markers change no ID, name or line of the IR. The core of a machine's rules, its step functions
bound by hand, `object rules extends Bindings(a ~> f, …)`, is the spelling of the IR generator's core
fixtures, and a Model may not use it. A folder
is a subpackage (`package standaloneactivity; package record`), so it reads its parent's vocabulary
without imports. A section that declares Properties over a machine argument returns them as a bundle
(below), so its Queries read them by field rather than declaring them.

Two families in one package cannot both be package-level givens, since each file would see both.
Each then lives in an object of its own, `object SystemFamily: given family: Family = …`, and each
file imports the one its declarations take (`import SystemFamily.given`), as the standalone
activity's files do: its feature file declares `ActivityFamily` and `SystemFamily`, and the files of
its `record/` and `withTaskQueue/` import `SystemFamily.given`.

An action, monitor, assumption, hole, channel with the actions it derives, or realization takes its
Definition ID from its `val`'s owner and name. Declarations moved to a new owner keep their IDs
through one `given DefinitionScope = DefinitionScope("pkg.Former$package$")` there, the compiler's
name for the former owner: each ID is `<former owner>.<val name>`. An owner pins once, not inside
an owner that pins and not to itself, and no two declarations may share an ID. Owners nested in a
pinned one keep their own IDs. No declaration names an ID of its own.

A feature declares its actions grouped by who takes them, so every call site shows the actor:
`caller.start()`, `worker.attemptStart`, `deadline.scheduleToStart`. A party that takes actions is
an actor object, `object caller extends Actor` (`umpire.Actor`), named after its object with the
first letter lowered, whose members are its actions (`val start = action(this)…`). Steps that are
no party's own are grouped in sections, `object timers extends Section` (`umpire.Section`):
`timers`, `deadline`, `history` for internal steps, `queue` and `faults`. Actor and section objects
are transparent to Definition IDs: a member takes the ID it would take as a direct member of the
section's enclosing owner, which at a file's top level is the file's package object, under the
file's pin, and directly in a machine's object is that object, under its pin
(`ForgedCompletion.caller.inspect` keeps `…Control$.inspect` under the control's pin). The IR generator refuses a section inside a
section, a section anywhere else, a section that pins, and two members that would share an ID, each
at its line. Actions a feature adds to a party another declares sit in a section that names the
party: the standalone activity's poll and answer are its `object worker extends Section`, taken by
the shared worker party it imports as `process` (`import shared.worker.{worker as process}`), whose
own actions read `process.workerStop`; the close policy's designs add `callerSide` and
`handlerSide` to the Nexus caller's `caller` and `handler`. A section is not named like a type of its
package but for the case, since the two would compile to class files whose names differ only in
case, which a case-insensitive file system cannot hold: beside the close policy's types `Caller` and
`Handler`, its sections are `callerSide` and `handlerSide`. The inputs an action declares by token
sit at the top level, apart from one named like an action of the object that takes it, which `object
Inputs` holds (`Inputs.control`, since inside `object caller` the name `control` is the action).

Type names follow the same pin. A top-level type belongs to its package, not its file, so a type at
the top level of a file whose declarations pin a former file owner `pkg.File$package$` takes the IR
name `pkg.<Type>` it had there; a pin of an object owner `pkg.Obj$` leaves type names alone. Two
types one lift reads that would share an IR name are refused.

A derived machine is another machine's declaration with one thing changed, an object named after
itself in the `given Family`. Chained, the derivation is one expression:

```scala
object StiffLamp extends Derived(Lamp.rebind(press ~> Lamp.effects.pressStiff)) // keeps press's guards
object FaultyLamp extends Derived(
  Lamp
    .extend(when(s => s.light == Light.on)(burnOut ~> Lamp.effects.burnOut)) // rules for an action the source does not bind
    .refining(ViewUnderFaults)(seen)                    // replaces the refinement, keeping what it lets through
    .assuming(burnOutAssumed)                           // appends assumptions, each once
), FailureModel
object PlainLamp extends Derived(Lamp.unmonitored)      // drops the monitors and the refinement
```

Each keeps everything else its source declares: starts, ends, evidence, unobservable timers,
monitors, assumptions and refinement. `rebind(action ~> effect)` keeps the guards and classes of the
action's rules and gives each the new effect; rules with different effects are refused unless each
fires one class, whose input the new effect reads. `rebind(when(g) { … })` replaces an action's
rules. The IR generator refuses rebinding an action the source does not
bind, extending by one it binds, binding one action twice, an assumption named twice, replacing a
refinement the source does not declare or by a machine of other state, outcome or fact types, and
a machine derived from or aliased to itself.

A composition is an object written with field selectors, and one derives from another by replacing
a member:

```scala
object CurrentOverQueue extends Composition[OverQueue](_.activity -> CurrentRecord, _.queue -> DispatchQueue):
  def end(s: State) = CurrentAdmission.end(s.activity)
  object syncs extends Syncs:
    sync(_.activity -> history.dispatch, _.queue -> queue.enqueue)
    sync("admit", _.activity -> worker.attemptStart, _.queue -> queue.deliver)
  object queries extends Section:
    val stale = scenario.actions(
      synced(_.activity -> history.dispatch),
      own(_.activity, caller.control(Control.pause))
    )
object StaleOverQueue extends Composition(CurrentOverQueue.withMember(_.activity -> StaleRecord))
```

`->` pairs a member with its value; it never means a transition. A sync with no name is named
after its first member's action, here `dispatch`; one written with its name, `admit`, is that name.
A selector, a sync and a member's action are resolved by the field and the action's declaration,
not by the strings the IR keys them with: `synced` finds the one sync that pairs that member's
action, and `own` an action that member binds and no sync pairs. `withMember` keeps the syncs, ends and member order and names
the derived composition after its object; a member that replaces a machine replaces, in the derived
one, the machine its new machine declares it refines. The IR generator refuses a selector that names no
field or no member, a member of another state type, a sync of an action the member does not bind,
a `synced` that matches no sync or several, an `own` of a paired or foreign action, and a
replacement of a replacing member by a machine that refines nothing.

A law over a composition reads a member's status set with `through(select, read)`, where it would
name a def: the composition's capability fields and a declaring function's function-valued
arguments. `through(_.activity, Admission.paused)` is `s => Admission.paused(s.activity)`, so a
composition needs no def of its own that restates its member's:

```scala
def overQueueCapabilities(c: Composition[OverQueue]) = capabilities(c, limits = five)(
  Closable(status = through(_.activity, Admission.phase), terminal = Admission.terminal, rejected = closedAnswer),
  Pausable(…, paused = through(_.activity, Admission.paused)),
  Pollable(dispatch = c.synced(_.activity -> worker.attemptStart), running = through(_.activity, Admission.running))
)

atMostOneActive(c)(through(_.activity, Admission.twoActive))
```

The IR generator lifts each `through` as one function of the composed state,
`<state>.through.<path>.<def>` (here
`temporal.features.standaloneactivity.withTaskQueue.OverQueue.through.activity.temporal.features.standaloneactivity.record.Admission$.paused`),
which the law calls as it calls a def. `select` is a field path, `_.activity` or `_.left.phase`, and
`read` a def of the lifted sources; each other selector and a lambda for `read` are refused at their
line, as a lambda is, and a `read` over another type than the member's does not compile. Its two
arguments share one parameter list: Scala infers the composed state from where the function is
passed only then. `through` is core, not sugar: no def of the lifted sources says which member a
law reads.

The queue in that example is the task queue, `model/temporal/shared/taskqueue`: a shared entity
(`taskQueue`, keyed by the queue's name) that a feature composes by synchronizing its own actions
with `enqueue`, `deliver` and `acknowledge`, and that imports nothing of any feature. It owns the
opaque contract `DispatchQueue` and its storage-loss variant, the providers that refine it
(`MatchingQueue` and the lossy one, failure models, and the forgetful and volatile negative
controls), the laws `queueLaws` every provider is held to, and the provider Queries.
A feature keeps its own syncs and its cross-entity claims, as the standalone activity's
`withTaskQueue/` does. The queue is a bounded abstraction, not a general queue: one message at a
time, delivered at most twice before its acknowledgment. The detailed provider's table, and a
design composed with it, has depth ten, so its free Queries run within `twelve`. Its declarations
keep the family, Definition IDs and type names they had in the standalone activity's system
contract, through the pin and the type-name rule above.

A claim over several designs is one function over `Declares[S]` whose state-dependent parts are
parameters, and each call passes defs of the lifted sources. A law is such a function, written once
for every entity in `model/temporal/capabilities`, and each instance is named by the
`val` that declares its call:

```scala
object pausedIsNotDispatched extends Law(cites = Seq("chasm/lib/activity/tasks.go"), promises = "…", doesNotPromise = "…"):
  def apply[S](m: Declares[S])(paused: S => Boolean, running: S => Boolean): Property[S] =
    m.property.never(s => running(s.state)).from(paused)

val notAdmittedWhilePaused = pausedIsNotDispatched(m)(Admission.paused, Admission.running)
```

A law is an object named after it: its `apply` states it, and its `Law` arguments are the server
code it rests on, what it promises and what it does not. A lambda passed for a function-valued
parameter is refused at its line, naming the def to write (a member's def read with `through`, above,
passes for a def), and so is a claim pattern after `when` (a
pattern reads every step) and a `keeps` projection that is not a field path. Any other value
parameter, such as an outcome, a fact or an action class, reads as the value the call passes: an
expression, a class or `when` reads `rejected` as `Outcome.notFound`.

A machine that declares capabilities receives such laws without calling them: see
[Capabilities and their laws](#capabilities-and-their-laws).

Such a function returns several claims as a bundle, a case class whose every field is a Property,
Scenario or Query. The IR generator folds the constructor's call to its claims, and a field read to the
one claim it names, as `providerQueries` reads `laws.delivers`:

```scala
final case class QueueLaws(delivers: Property[QueueDetail], committedStays: Property[QueueDetail])

def queueLaws(m: Machine[QueueDetail, QueueOutcome, QueueFact]): QueueLaws = QueueLaws(
  m.property("delivers") when deliver holds (after => after.records(QueueFact.delivered)),
  m.property("committedStays")
    .stays(_.custody != Custody.nowhere)
    .unless(_.records(QueueFact.acknowledged))
)
```

Sugar is kept apart from the core. The core is what the IR needs declared: `machine`, the
derivations `rebind`, `extend`, `refining`, `assuming` and `unmonitored`, `given Family`, `action`
and its classes (`start()` among them), `input`, `UpTo`, `steps` and `~>`, `Step` and `because`,
`Declares[S]`, `property` with `holds`, `holdsAcross` and `when`, `monitor`, `leadsTo`, `compose`
with `sync` (named or after its first member's action), `synced`, `own` and `withMember`,
`scenario`, `query` (named or after its Scenario and Property), `Limits`, `.total`,
`DefinitionScope`, `choose`, `irFile`, and the realization declarations, among them the script helpers `rpc`,
`readUntil`, `withFields`, `perform`, `onPath`, `everyCase`, `script`, `command` and `statusTable`, `Actuator`,
`MonitorExpectation` and the kit's roles and bindings. Sugar is a form whose meaning a core form
already says: `implies`, `in`, `records`, `enter`, `stay`, `disabled`, the claim patterns (`once`,
`keeps`, `never`, `from`, `stays`, `unless`), the monitor pattern (`sticky`, `stickyAcross`) and
both spellings of `:=`, named inputs and request
fields. It lives in the `Syntax.scala` files of the DSL (`umpire/Syntax.scala`), the kit
(`temporal/realize/Syntax.scala`) and the IR generator (`irgen/Syntax.scala`). Each form is documented
with `Core form:` and the core spelling it stands for, and an IR generator fixture lifts it beside that
spelling and requires the same IR. No core file imports sugar, and a sugar word is defined in no
other file; `make lint-model` checks these three rules.

Three symbols, each with one meaning: `~>` binds an action to its step function, `->` pairs a key
with its value, `:=` gives a named slot a value; everything else the DSL adds is a word.

`:=` is one operator (`@targetName("set")`) for both kinds of named slot, an action's input token
and a request's field. A named call `start(scheduleToStart := expires)` is sugar for the positional
`start(unset, expires, unset)`: the supplied inputs take the action's declaration order, and an
omitted input takes its domain's first value, here `unset`. `start()`, which omits every input, is
core: the class of every input at its first value, `start(unset, unset, unset)`, and the one class of
an action with no input. The IR generator refuses, at the call's line, a token that is no input of the
action, a token supplied twice and an omitted input whose domain has no value. A value of the wrong type for its
token is a compile error.

A step that can go more than one way names each result with `choose`, which is core: it says
what no other form does, the names of the alternatives, which the IR records on each step record
as its `choice`.

```scala
val committed = choice
val redelivered = choice

def admitted(s: AdmissionState): List[AdmissionStep] = choose(
  committed -> enter(s.copy(message = Message.empty), AdmissionFact.statusStarted),
  redelivered -> enter(s.copy(message = Message.redelivery), AdmissionFact.statusStarted)
    .because("the channel may deliver the message again")
)
```

A name is inert metadata (model/SEMANTICS.md, Named choices): the rows, fingerprints, Query
answers and Cases are the ones the same steps give written as an unnamed list. Each alternative is
one step written out, `enter(…)`, `stay(s)`, `List(Step(…))` or one of those with
`.because("…")`, or a call of a function that gives at most one step in each branch, such as a step
another action shares (`forged -> settle(s, Resolution.succeeded)`); where that
function gives no step, the alternative is not taken. The IR calls a copy of the function,
`<function>$<choice>`, whose every step carries the name, so the function's other calls keep their
unnamed steps. Each name is the simple name of its token's `val`. A choose of one alternative does
not compile; the IR generator refuses, at the alternative's line, a name given twice in one choose (one
token twice, or two tokens whose `val`s share a simple name), an alternative that is not one step
written out (`Nil` or `disabled`, two steps, an `if`), a called function that gives several steps
or a step it does not write out, a token no `val` declares, and an alternative kept in a `val`
rather than written in the call.

Every branching of a Model is a `choose`. The IR generator refuses, at its line, a step function's
several results written without one: a `List(Step(…), Step(…))` of two or more steps, and steps
joined with `++`, in a step function or any function it calls. A step function with one result
needs no `choose`.

A name is a token of its own, not a case of an enum: an enum used only for names would be lifted
as an IR type the Model does not otherwise need, and the names of different step functions are not
one closed set.

A Scenario without `starts` starts in its machine's one declared start, and a composition's in the
record of its members' starts; where there is not exactly one, it is refused. Evidence is optional.
A fact no line names is confirmed by evidence of its own name, so `evidence { case … }` lists only
the exceptions. A fact case with fields needs a line that covers all its values.

`query … in s` takes no given. A Property of the machine the Scenario's machine `refines` is read
through that refinement; a Property of an unrelated machine is refused at the Query's line.

### The reading order and its lint

A Scala object initializes its `val`s in the order they are written, and an object read while
another initializes is initialized then. A feature file read top to bottom is therefore held to its
order by a lint, which the IR generator runs over every source it reads before it lifts anything, in
the gate and in each of its own fixture tests (`model/irgen/Order.scala`). It refuses, each at its
line, as `lift: <file>:<line>: …`:

| Kind | Refused | Fix |
| --- | --- | --- |
| (a) | a `val` read while its object initializes, before the object declares it: it is still `null` there | declare it before the declaration that reads it |
| (b) | a cycle of objects, files' top levels and the objects nested in them, each read while the one before it initializes | read it in a `def`, a lambda or a lazy `val`, or move what is read into an object of its own, as the protocol's `implements` is |
| (c) | in a feature file, a declaration out of the order above: at the top level, or among a machine object's header and sections, or a Scenario after a Query in `queries` | move it |
| (d) | in a feature file, a declaration outside its place: a step function outside `effects`, vocabulary outside `states`, a refinement member outside `refinement`, a monitor outside `monitors`, a hand-written `action ~> step` in a machine object (outside `rebind`), a Property outside `properties`, capabilities outside `implements`, a Scenario or Query outside `queries`, any of them or a machine at the top level, an IR file outside `exports`, a section nested in a section or outside a machine, composition or file top level, two section members of the whole Model that would share a Definition ID, a Property, capabilities or Scenario over another object's machine, a Query over another object's Scenario; beside a feature file, a Model declaration in another file; and in a Model folder with no feature file, a Model declaration in a file not named after the folder | move it to the place the message names, or name the file after its folder |

A read inside a `def`, a lambda, a by-name argument or a lazy `val`, and an object declared but never
read while another initializes, initializes nothing and is not refused; a context function the DSL
applies at once is read where it is written, and so are a rule heading's rules and guard. The lint follows no call: a
`def` called during initialization that reads a later `val` is not caught. A feature file is a
source named after its folder in a package under `features` or `shared`; the other files of a
feature folder, its `Realization.scala` and tests, and the kit's files answer to (a) and (b) only. The lifter's refusal specimens (`*Rejects.scala` among its fixtures),
which read a `val` before it is declared on purpose, are left to the refusals they are specimens of. Each
kind has its refusal fixture, `model/irgen/testdata/initOrder/` and, for the sections,
`model/irgen/testdata/sectionOrder/`, and a misnamed feature file its own,
`model/irgen/testdata/misnamed/`.

The lint is the IR generator's own because neither of Scala's checkers serves (fn-126): `-Wsafe-init`
checks classes and not objects, as of Scala 3.9, and every owner here is an object or a file's top
level; `-Ysafe-init-global`, which checks objects, stops the compiler on a read of a ScalaPB gRPC
method descriptor (`WorkflowServiceGrpc.METHOD_*`), which every realization makes, through Scala
3.10.0-RC3. The first run found two reads of a citation before its declaration, `notFoundCode` in the
standalone activity and `jobsCode` in a lifter fixture; each was `null` at run time, where only the
lifter, which reads the trees, had read it.

### Starting a new feature

Copy the template, `model/irgen/testdata/layout/lamp/`, to `model/temporal/features/<feature>/`, and
name its feature file after the folder and its machines after the feature. A feature whose Models
include a refinement pair keeps its types, signature and `object exports` in that file, its Product
in `product/Product.scala`, and its System in `system/System.scala` with each zoom-in in a file of its
own beside it; a feature of one level keeps its machines in its feature file and has neither folder.
The structure lint (`model/irgen/Structure.scala`, fn-126 R20) refuses any other layout, and an
object in a machine object named other than `states`, `refinement`, `effects`, `monitors`, `rules`,
`syncs`, `properties`, `implements` or `queries`; its refusal fixtures are under
`model/irgen/testdata/layoutRefusals/`. It holds every feature under `model/temporal/`, and a lifter
fixture once it has a `product/` or `system/` folder. A file of a level folder reads in a feature
file's order, without `object exports`, which only the root feature file holds.

### Capabilities and their laws

A capability is what an entity can do, declared on its machine as a binding of a protocol's
parameters to the machine's own vocabulary: which statuses close it, which class pauses it, which
class hands its work to a worker. A law is a claim stated once for every entity that has a
capability, or a pair of them. A machine that declares its capabilities receives the laws the given
`Catalog` brings for each capability and for each pair it declares both of, without listing them,
as a Property, a Scenario and a Query named `<machine>.<law>`. A capability is not a machine: the
entity's machine stays the only one, and the laws are read on it.

The framework keeps the mechanism and names no capability (`model/umpire/Capabilities.scala`:
`CapabilityOf`, `capabilities`, `except`, `overriding`, `cited`; `model/umpire/Catalog.scala`:
`CapabilityKind`, `Law`, `Catalog`; `model/umpire/Compose.scala`: `through`, which a composition's
fields read a member with). Temporal's capability kinds, every law with its server
citations and the one `given Catalog` live in `model/temporal/capabilities`:

| Capability | Fields | Laws it brings |
| --- | --- | --- |
| `Closable` | `status`, `terminal`, `rejected` | `terminalStatesAreFinal`, `closedIsRejectedUniformly` |
| `Terminable` | `terminate`, `settled`, `reach`, `expect` | `terminateSettles` |
| `Cancelable` | `requestCancel`, `requested`, `reach`, `expect` | `cancelIsRequested` |
| `Pausable` with `Pollable` | `pause`, `unpause`, `paused`; `dispatch`, `running` | `pausedIsNotDispatched`, the law of the pair |
| `Describable` | `status`, the realization's fact-to-status table | none of its own: the generated finds' awaits read its table |

**The worked example.** The standalone activity declares its capabilities in two declarations, the
`implements` objects of its `ActivityProduct` and `ActivityProtocol` in
`model/temporal/features/standaloneactivity/StandaloneActivity.scala`, each `capabilities(limits)(…)`
of the machine it sits in:

```scala
import temporal.capabilities.{given, *}

// In ActivityProduct.implements:
val all = capabilities(limits = three)(
  Closable(status = states.phase, terminal = states.terminal, rejected = cited(Outcome.notFound, states.notFoundCode)),
  Pausable(pause = caller.control(Control.pause), unpause = caller.control(Control.unpause), paused = states.paused),
  Pollable(dispatch = worker.attemptStart, running = states.running)
)

// In ActivityProtocol.implements:
val all = capabilities(limits = three)(
  Terminable(terminate = caller.control(Control.terminate), settled = ProtocolFact.statusTerminated,
    reach = Seq(caller.start(), process.workerStop), expect = inconclusive(explanationsDisagree)),
  Cancelable(requestCancel = caller.control(Control.requestCancel), requested = ProtocolFact.statusCancelRequested,
    reach = Seq(caller.start(), process.workerStop), expect = inconclusive(explanationsDisagree)),
  Describable(status = ActivityRealization.activityStatus)
)
```

The first gives `activityProduct.terminalStatesAreFinal` and `activityProduct.closedIsRejectedUniformly`
(Closable), and `activityProduct.pausedIsNotDispatched`, because the product declares both Pausable
and Pollable; nobody lists the pair. Each is a transition Property, verified over the product's free
Scenario under `three`, and the protocol reads them through its refinement. The second gives
`activityProtocol.terminateSettles` and `activityProtocol.cancelIsRequested`, each a same-step
Property asked by a `find` that starts the activity, stops the worker and then takes the control; a
find has a realization, so they lower to the Cases `activity-activityProtocol.terminateSettles` and
`activity-activityProtocol.cancelIsRequested`, whose awaited status comes from the `Describable`
table. The functional laws sit on the protocol because a find lowers only through a realization,
which is the protocol's. Neither `terminalIsFinal` nor `pausedIsNotDispatched` is written in the
activity's own files any more; the admission designs and both composition families declare the same
three capabilities on their record, and the Nexus operation (`features/nexusoperation`) declares
Closable, Terminable, Cancelable and Describable.

**How a law is lifted.** Each law is the law's `apply` (or the def an
`overriding(law -> def, because = …)` names, which takes the law's parameters) folded with the model
and the fields of the capabilities that bring it, bound by parameter name. Every field that takes a
function names a def of the lifted sources, or a member's def read with `through` on a composition,
never a lambda. A law of one action class (`when`) is
asked by a `find` from the start through the capability's path to a live state (its field that lists
action classes, Terminable's `reach`) and that class, expecting of a server the Run its
`RunExpectation` field names; any other is verified over the free Scenario from the start under
`limits`. The Query's total is computed as [below](#counting-a-querys-total). A field of an action
class names an action the machine must bind. A capability is a case class extending `CapabilityOf`
whose companion extends `CapabilityKind`, which the catalog keys its laws by; the IR generator
refuses any other. A Query of the entity's own reads a generated Property by its law,
`declared.claim(pausedIsNotDispatched)`, as the activity's pinned paths do.

**`except` and `overriding`.** An entity the server answers otherwise waives the law with its reason,
citing the server code; both are refused without one. `except(law, because = …)` lifts nothing for
the law: the admission record answers a delivery after it closed, so its designs and compositions
declare `.except(closedIsRejectedUniformly, because = deliveryAfterClose)`.
`overriding(law -> ownDef, because = …)` lifts the entity's own def under the law's name: the Nexus
operation answers a repeated request id OK after it closed, so it declares
`.overriding(closedIsRejectedUniformly -> closedRejectsOrRepeats, because = repeatedRequestsAnswer)`.
A law lists the parameters where entities differ on purpose, `Law(…, parameters = Seq("rejected"))`,
and an entity backs each such binding with the server code that answers it so,
`rejected = cited(Outcome.notFound, "chasm/lib/activity/activity.go")`; `cited` changes no IR. Lint
forwards each waiver's reason into `<file>.lint.json` and reports a binding left uncited
([Running the gate](#running-the-gate)).

**How a new entity gets its laws.**

1. Give its machine object an `implements` object, in a feature file that imports
   `temporal.capabilities.{given, *}`, that declares
   `capabilities(limits)(…)` with the capabilities the machine has, each field a def or an action
   class of the Model, and each parameter its law lists under `parameters` written with `cited`.
2. Name the declaration as a root of the folder's `irFile`, as `exports.activity` names
   `ActivityProduct.implements.all`.
3. Run `make umpire-gen-model`: it writes the generated claims, the Cases of the generated finds and
   the law sidecar. A law the Model breaks shows as a counterexample of the Query `<machine>.<law>`.
   Where the server does not keep the law for this entity, waive it with `except` or `overriding`
   and the reason; otherwise fix the Model.
4. Read the laws on the machine's table: `go run ./tools/umpire/cmd/umpire-lint --tables
   model/ir/<file>.json` (below).

**The catalog and the two-entity rule.** The catalog is a Scala value, `given catalog` in
`model/temporal/capabilities/Catalog.scala`, built with `Catalog.single(kind)(law, …)`,
`Catalog.pair(kind, kind)(law, …)` and `++`; the IR generator folds it as data, and finds the pairs
among a declaration's kinds. A law enters the catalog only once two instantiating entities declare
the capabilities that bring it, an instantiating entity being a machine with its own state type: a
composition reading its members' capabilities through their projections does not count again, and
neither does a derived machine. `Catalog.test.scala` fails, naming the law, for a law fewer than two
declared machines instantiate, and lint's `law-with-one-instance` counts the same way across the
sidecars of `model/ir`. A claim with one instance stays the feature's own (`atMostOneActive`,
`startedByPollingWorker`).

**The law sidecar.** The IR has no text field on a Property, so the IR generator writes what it
expanded beside each IR file whose Models declare capabilities, as `<file>.laws.json`: each
generated claim with its law, the capabilities that brought it, the action class each of their
action fields names (`Pollable.dispatch`: `attemptStart`), its bindings and the citations of its
cited bindings, and the def that overrides it; each waiver with its reason and position; and the
catalog's laws with what they promise and do not promise, the parameters each instance must cite,
where the catalog brings them and the machines, one per state type, that instantiate them. It is a
checked-in output of the gate like the IR, not IR: lint, the table view and the accepted findings
read it, and a Go reader skips it (`IRPaths`). A generated Query whose verdict is a counterexample is
reported with the law, its capabilities and their bindings
(`rogueJob.pausedIsNotDispatched breaks the law pausedIsNotDispatched of Pausable and Pollable
(paused = …, running = …)`).

**The laws on the table.** `umpire-lint --tables` prints, after each machine's per-operation table,
the laws it is held to: per law its claim, what it promises and does not promise, and the modality
it pins (MUST NOT for a transition law, of its results where the cell is a MAY; MUST for a same-step
one, which a find asks on its path only) on the cells of its capabilities' actions, beside each
cell's own modality; a law whose
capabilities name no action, Closable's, pins its cells on every class. A product law the protocol
reads through its refinement is marked `inherited`, and `unchecked` where no Query over the
protocol's own Scenarios asks it. Then come the cells of the capabilities' actions that no law pins,
and the laws the machine waives with `except`. A law pins cells the step function wrote; it never
adds, removes or rewrites a row:

```text
laws model/ir/activity.json activityProduct
  activityProduct.pausedIsNotDispatched  pausedIsNotDispatched of Pausable and Pollable, MUST NOT  …/StandaloneActivity.scala:274
    promises: while an entity is paused no work is handed to a worker: no step from paused lands in running
    does not promise: what a pause of held work does (…), what a second pause or an unpause of a live entity answers, …
    attemptStart (Pollable.dispatch)    paused  MUST NOT  cell: ? s.phase != scheduled
    control-pause (Pausable.pause)      paused  MUST NOT  cell: ? !pausable(s)
    control-unpause (Pausable.unpause)  paused  MUST NOT of its results  cell: MAY accepted -> scheduled [statusScheduled]
  activityProduct.terminalStatesAreFinal  terminalStatesAreFinal of Closable, MUST NOT  …/StandaloneActivity.scala:269
    …
    every class  completed, failed, canceled, terminated, timedOut  MUST NOT
  no law pins
    attemptStart (Pollable.dispatch)    scheduled                 MAY  accepted -> started [statusStarted]
    control-pause (Pausable.pause)      scheduled, started        MAY  accepted -> paused [statusPaused]
    …
```

**Core and sugar.** `capabilities`, the capability types and kinds, `except`, `overriding`, `cited`,
`through`, `Catalog`, `Law` and the law objects are core: each introduces meaning the IR or the sidecar needs.
This surface has no sugar today; a convenience spelling of it would live in
`model/umpire/Syntax.scala` or `model/temporal/capabilities/Syntax.scala`, documented with its core
form, with its matching in `model/irgen/Syntax.scala` and a fixture requiring the core spelling's IR,
under the rule [above](#writing-a-model).

### Counting a Query's total

Every Query states `total n`, the number of its static combinations, which the author works out and
the Go reader recomputes. It is what a reviewer reads to see how large a question is before anything
runs. Count the Scenario machine's states, its whole state catalog: the product of its fields'
catalogs, an enum counting each case times its fields. Then:

- a pinned Scenario counts states times the scheduled slots within the step limit,
  `min(steps, scheduled actions)`;
- a free Scenario counts states times the machine's action classes times `steps`. Each bound action
  has one class per assignment of its inputs: an action without inputs is one class, and one with
  inputs the product of their catalogs. A composition counts its own state record, and as classes
  each member's classes that no sync takes plus, for each sync, the classes of its first action times
  those of its second.

`syncCompletion` above pins two actions within `limits two`. `ProtocolState` has a `Phase` of 8
cases, `attempts` of 0 to 2 and three two-valued `Timeout`s, so 8 × 3 × 2 × 2 × 2 = 192 states, and
the total is 192 × min(2, 2) = 384. A free Scenario of the same machine within three steps would
count 192 × 23 × 3 = 13248: `schedule` has 2 × 2 × 2 = 8 classes, `handlerReply` 4 + 2 = 6 (one of
its five replies carries a Boolean), `complete` 3, and the six actions without inputs one each.

The count is taken before anything is reached. Unreachable states, disabled steps, states the search
meets twice and a search that stops at its first witness all count, and a named choice's alternatives
are results of one class, not more classes. So the total bounds what the search could be asked to
look at; it does not predict the paths it visits, which the Query's answer reports, or what a Run
executes. A Query read through a refinement counts its own Scenario machine, `limits` with 0 steps
count 0, and a total changes no table, fingerprint, answer, Case or exploration identity.

A Query declared inside a `def` over a machine argument, such as a list of Queries applied to several
providers, takes the total of each Query whose count differs between instances as an `Int` parameter,
and each call supplies the literal: `providerQueries(lossyMatchingQueue, anyTotal = 3240)`. A literal
in the body is right only when every instance counts the same. The IR generator refuses a Query with no
total, a second `total` on one Query, a negative one, and a total that is not an integer literal or a
parameter supplied as one, each at its line. A total that is not the count is refused by the reader
at the Query's line with both numbers and the factors, for example
`query syncCompletion declares a total of 380, and its static combination count is 192 states × 2
scheduled slots (the least of 2 steps and 2 scheduled actions) = 384`. IR lifted before totals
existed has none and is still read; generating Cases from `model/ir` requires one on every Query.

Anything else, such as a `var` or a loop, stops the lift with its source line. The IR generator works on
the compiler's typed trees (TASTy) and not as a macro, because a macro sees a function's body only
inside its own compilation run and only after pattern matching has been compiled away. The cost is
that a refusal arrives from the lift step, a few seconds after compiling, and not as a compile
error.

Scala compilation rejects type errors such as binding a step to an action with different inputs.
Positional calls are typed per position, so a value of another type, or another number of values
than the action has inputs, does not compile.
The IR generator rejects constructs it cannot express in the IR. Go then reports Model problems such as
a start outside the state domain, a stuck state, a class bound twice or a failed refinement from
the IR at the Scala source line recorded by the IR generator. These semantic errors surface through
`make umpire-check-model`, rather than a Scala unit test.

A Query that should become a Case needs two more things: a realization on its machine, and
`.expect(RunExpectation(...))` on the Query, which states the model assessment a Run of it should
get. A missing or malformed expectation fails generation. What a realization may declare, and how
each declaration lowers, is in [SEMANTICS.md](SEMANTICS.md) under Realizations and Generated Case
expectations.

What every Temporal realization says alike is declared once, in the kit `model/temporal/realize`
(`Kit.scala`, over the vocabulary of its `Realize.scala`): the roles (`workflowService`,
`caseWorker`, `taskQueue`, `handlerTaskQueue`, `nexusEndpoint`) and the environment bindings a run
supplies for them, the correlation window, the controller script, the one form a read waits in
(`await`, which writes no interval and no deadline), the helpers that declare evidence from the Run's own record (`answered`, `answeredAs`,
`delivered`), the deadlines a request sets, and `temporalRealization`, which takes what a feature
says differently. The standalone activity and the Nexus caller realizations both use it. A feature
that reads a status back declares what each fact reads as once, `statusTable(fact -> value, …)`
beside its realization, and its `awaitStatus` reads the status it polls for from that table. The
table is the realization-side form of the `Describable` status map of
[.plans/SEMANTIC_PROTOCOLS.md](../.plans/SEMANTIC_PROTOCOLS.md); the IR generator reads it when it lifts,
and it adds nothing to the IR.

A realization whose system serves the feature only behind a flag says so,
`requiredSettings = Vector(RequiredSetting(key, value))`, with the kit's `RequiredSetting`, the
key and value as the server's dynamic configuration spells them (the standalone Nexus
operation's `nexusoperation.enableStandalone`). A Case lowered through it carries them,
Testpilot's preparation refuses a Profile whose dynamic configuration lacks one or sets it
otherwise, naming the setting, and the live suite applies each Case's settings to the server it
starts.

How Temporal's APIs behave between calls is declared once, in the kit's `Behavior.scala`, each hint
with a comment citing the server code it rests on, and `temporalRealization` attaches it to every
Temporal realization (.plans/API_BEHAVIOR_HINTS.md):

```scala
METHOD_PAUSE_ACTIVITY_EXECUTION.visibleTo(METHOD_DESCRIBE_ACTIVITY_EXECUTION, Visible.atOnce)
CauseKind.handlerReply.visibleTo(
  METHOD_DESCRIBE_WORKFLOW_EXECUTION,
  Visible.eventually(WaitBound(intervalMs = 250, atMostMs = 2000))
)
CauseKind.delivery.boundedBy(WaitBound(intervalMs = 250, atMostMs = 3000))
```

A realization names the steps no command performs and the kind of cause each is,
`serverSteps = Vector(ServerStep(worker.attemptStart, CauseKind.delivery),
ServerStep(deadline.scheduleToStart, CauseKind.timer, deadlineMs))`, a timer with the kit's deadline its request sets (an activity's
retry `backoff` names the kit's `firstRetryBackoffMs`, the server's default first retry interval). A hint names only
generated method constants, so one the API does not have does not compile, and the IR generator refuses a
method that is no generated constant at its line; the Go reader refuses a missing or non-positive
bound at the hint's line. A read written with no interval waits as the lowering derives from both:
once after a write of its own script visible at once, and otherwise within the declared bounds of
what it waits for; a read after a write with no declared visibility is refused, naming both methods
(SEMANTICS.md, Realizations). No Temporal realization writes an interval or a deadline of its own:
every read is written with `await` and its wait is derived. A poll that writes its own interval in a
realization that declares a behavior is lint's `explicit-wait` finding, kept only where
`model/ir/<file>.lint.json` accepts it with the reason no hint covers its wait. The waits that stay
explicit are no reads: the Driver's own awaits (`AwaitLearned`, `AwaitCommand`, its controls) and the
Nexus caller's closing long poll take the instruction defaults the kit declares
(.plans/API_BEHAVIOR_HINTS.md, "As built by task 5").

`temporalBehavior` also declares, once for every Temporal realization, what Testpilot and
conformance would otherwise assume about Temporal (fn-124.3):

- `attemptNumbering = Some(AttemptNumbering(first = 1, oneRun = true))`: the server numbers an
  activity's attempts from 1, every one of its one run. The lowering writes it on each activity
  entrypoint (`ActivityActivation.attempt_numbering`); Testpilot judges each reservation's attempt by
  it and refuses an activity entrypoint that declares none, and conformance holds one operation's
  evidence to one activity run only where it is declared.
- `instructionDefaults = Some(InstructionLimit(timeoutMs = 10000, attempts = 1))`: the limits of an
  instruction that writes none. The lowering writes them into the Program
  (`Program.instruction_defaults`); a Temporal Profile has none of its own.
- `runOrderIsCausal = true`: the Run's record order is the order of one operation's evidence across
  its sources. The lowering writes it into the Program (`Program.run_order_is_causal`), and only then
  does Testpilot name the operation's previous evidence from another source as a causal parent.

A history kind's recorded message is whatever the realization's own read that lifts history reads
(its method's response at its path), and its kind is a member of that message's oneof, so neither
the lowering nor Testpilot names a history message.

### Naming protobuf data in a Model

Actions name message types, and realizations use generated unary method constants and typed field
selectors. For example:

```scala
import io.temporal.api.workflowservice.v1.StartActivityExecutionRequest
import io.temporal.api.workflowservice.v1.WorkflowServiceGrpc.METHOD_START_ACTIVITY_EXECUTION
import umpire.*
import umpire.realize.*
import temporal.realize.*

val start = action("start", Party("caller")).schema[StartActivityExecutionRequest]
val startActivity = rpc(workflowService, METHOD_START_ACTIVITY_EXECUTION) {
  field(_.namespace) := workerNamespace
  field(_.activityId) := run
}
```

The call is the command `start-activity`, after its `val`. The scope `rpc` opens fixes the request
type, `StartActivityExecutionRequest`, so each line selects a field of it and no line repeats the
type. It stands for the record the IR generator writes, whose core form
`Instruction.rpc(role, method)(Vector(Assignment.typed(Field[Req, V](_.x), operand)), Vector.empty)`
still lifts to the same IR.

`Recorded.read` and `Recorded.single` keep the method's request type and the selected response
message type in an `Evidence.read` reference. `readUntil`, the kit's `await` and `Instruction.readUntil` take
that reference, so their assignments use the method's request type and their condition selects
fields of the projected message. `Field[Root, Value]` also names evidence fields, operation keys,
response reads and Run Event guards. Select an optional nested message with a generated `get...`
accessor, each repeated element with `.map`, and a oneof arm through its generated selector. A
dynamic Run Event payload starts from `Operand.Projected.as[InstructionOutcome]` so the selected
root is explicit.

Constant messages remain symbolic: `Proto[Payload](ProtoField.typed(...))` names a message and its
fields by type; `ProtoValue.mapping(ProtoEntry.typed("encoding", ProtoValue.utf8("json/plain")))`
writes `Payload.metadata` data with its `String` key and `ByteString` value. The generated API and
ScalaPB runtime are authoring and lifting dependencies only. Scala never builds or sends a Temporal
request; Go still checks the lifted IR against descriptors and executes the Case.

## Generated Cases

`make umpire-gen-model` and `make umpire-gen-cases` write `model/cases/*-case.json` and
`model/cases/manifest.json`. The manifest gives every Query of every checked-in IR file one
standing: `lowered`, `nothing-to-realize`, `no-realization`, or `unsupported` with located reasons.
The generator builds and validates a complete tree before it replaces anything, and the check
fails on a changed, missing or left-over file. Ordinary Go tests never rewrite the tree.

Two other trees pin Cases from this one, byte for byte:
`tests/testcore/testpilot/testdata/generated` for the functional tests
(`make umpire-gen-fixtures`, `make umpire-check-fixtures`) and `tools/canary/casebinding/testdata`
for the canary (`make canary-gen-case`, `make canary-check-case`). After a Model change the order
is `umpire-gen-model`, `umpire-gen-fixtures`, then `canary-gen-case`.

`TestTestpilotGeneratedCases` (`tests/testpilot_generated_test.go`) discovers every
lowered file, binds two independent namespaces and task queues, runs them concurrently twice under
each Nexus implementation, and compares the Contract Verdict and the declared assessment, live and
replayed. A Case that needs a capability the environment lacks is reported before any resource is
provisioned.

`.build/umpire-run --case model/cases/<file>` with `--grpc`, `--http`, `--namespace` and
`--task-queue` runs one Case against any deployment and reports its Verdict. A Case that needs
delivery control is skipped there with its preparation reason, before anything is dialed, and the
command exits with status 3.

A generated Case's Contract checks the authored evidence path, the terminal evidence and the events
that support them. It does not check exact payload equality, ID spellings, the complete SDK attempt
sequence, or isolation by cross-namespace reads. Closing one of those gaps means declaring an
observation in the Model, not writing a Go assertion for one Query.

## Bounded exploration and replay

A `find` Query can declare `.explore(Exploration(...))`: finite alternatives for named positions
of its Scenario, integer priorities, a Run budget and a bounded prefix-deletion sweep. These are IR
declarations. `tools/umpire/explore` enumerates their combinations and has the same reader and
lowering produce a Case for every candidate. Enumerating the candidates is exact model coverage;
running one is a single sampled execution. Neither is an exhaustive search of server schedules.

`make umpire-ir-bridge` builds `.build/umpire-ir-bridge`, which serves candidates from `model/ir`
to the campaign and replay commands. `make umpire-fuzz-run` and `make umpire-replay-run` build it
and run those commands; both need a deployment. Replay first requires two fresh Runs that
reproduce the original failure, then tries the declared deletions and keeps an edit only when two
Runs reproduce the same failure again. An incomplete or unreproduced failure produces no proposal.
`TestTestpilotExplorationDiscoversUnpinnedExecution` and
`TestTestpilotNexusControlReplaysThroughTheCommand` run both against an in-process server; set
`UMPIRE_EXPLORATION_DIR` to keep their Cases, Runs, reports and HTML traces.

The control machine `ForgedCompletion` (`forgedCompletion`, `model/temporal/features/nexuscaller/NexusCaller.scala`)
deliberately admits a forged success beside the real failed callback. It is a negative control,
marked `NegativeControl`, that shows a violated Verdict being found and replayed, not a server
defect.
