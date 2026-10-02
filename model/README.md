# model/scalav2

Scala as the authoring front end of a language-independent Umpire IR, with Go as the only thing
that runs it. The framework and Models live in `scala/`; a lifter reads their compiled typed trees
and emits the IR, and a Go interpreter derives every table, identity and fingerprint from the IR alone.
The architecture is the one `SCALA.md` proposes ("Scala is syntax. The IR is the specification. Go
interprets the specification.").

```text
scala/ (umpire framework, Temporal Models, native tests)
      │  scala-cli compile → TASTy
      ▼
lifter/Lift.scala  ── typed trees → IR, refusing anything outside the subset at its line
      │
ir/*.json  (ProtoJSON of the IR schema; checked in)
      │    nexus-caller.json     the Nexus caller and worker machines, its functional Queries and their realization
      │    activity.json         the standalone activity Model, as model/go/standaloneactivity has it, and its realization
      │    activity-system.json  the standalone activity's system contract and its dispatch queue
      │    activity-race.json    the held race and one bounded admission response loss
      │    nexus-close.json      the Nexus caller close and reset designs
      │
goir/  ── validate, interpret, derive tables → model/go's umpire.Table
      │
      ├── parity with the Lean dumps: tables, IDs, refinement rows, target fingerprint
      │
goir/testpilot/  ── a find Query's witness through its realization → a Testpilot Case
      │
backends/  ── the same IR exported to Quint, and one monitor to P, each held to goir's reading
```

## Run

```sh
make umpire-check-model       # the gate (gate/): build and test umpire/ and temporal/, lift, require every ir/*.json and case to be current, test
make umpire-gen-model         # the gate with --update: lift and rewrite every ir/*.json, the lifter's expected IR and the Cases
go test ./model/scalav2/...    # the Go side alone, from the checked-in IR, no JVM
model/scalav2/backends/run.sh  # Quint and P against Go; needs the tools backends/README.md names
```

scala-cli, the JDK and protoc come from the repository's `mise.toml`. After changing the schema, run
`make protoc` to regenerate `api/umpire/v1`.

## Layout

| Path | What it holds |
| --- | --- |
| `proto/internal/temporal/server/api/umpire/v1/ir.proto` | The IR schema, with the other internal protos: types and catalogs, functions as expression trees with `match`, actions, machines, source positions; its Go code is `api/umpire/v1`, from `make protoc` |
| `SEMANTICS.md` | The evaluation rules, which neither side defines |
| `ir/nexus-caller.json` | The Nexus caller and worker Models, lifted with the functional Queries and the realization that runs them |
| `ir/activity.json` | The standalone activity Model, lifted with the claims `model/go/standaloneactivity` declares and the realization that runs its find Queries |
| `ir/activity-system.json` | The standalone activity's system contract, lifted: the admission designs, the dispatch queue's providers and their compositions |
| `ir/activity-race.json` | The held race and bounded admission response loss: Queries and realizations that hold a dispatch, observe durable admission, and test stale-delivery rejection or a lost committed answer |
| `ir/nexus-close.json` | The Nexus caller close and reset designs, lifted with their monitors, Queries and progress claims |
| `scala/` | The Scala authoring project: the `umpire` framework with the realization declarations in `umpire/realize`, the Temporal Models, and their munit tests |
| `lifter/` | The TASTy lifter; `testdata/unsupported` is a Model it must refuse |
| `goir/` | Loader, validator and interpreter |
| `goir/testpilot/` | Lowers a find Query's witness through the realization its machine declares into a Testpilot Case, and names what a realization declares that Testpilot cannot run yet |
| [backends/](backends/README.md) | Exports every machine and composition of the four IR files to Quint and one monitor to P, runs both tools, and compares each with `goir`: transitions, monitors and Properties by Quint, bounded event traces by P. Its tool runs are skipped by `go test` and required by `backends/run.sh` |
| `gate/` | The gate, one Scala program run by `make umpire-check-model`: builds and tests the Models, packages the IR's Java classes as `gen/ir-proto.jar` (gitignored) for the lifter, runs the lifter's tests, lifts, holds `ir/` to what it lifted and runs the Go checks. Every tool is run through `Tools.scala`, where a scala-cli run that prints an error fails whatever it exits with |
| [specimens/](specimens/README.md) | fn-107's two reviewed authoring sketches, standalone activity admission and Nexus close/reset, with their trace oracles, proposed extensions, Testpilot gaps and authoring measurements; hand-reviewed, and built by no gate |

## Independence from model/scala

`scala/` began as a copy of model/scala's framework, Case producer, Nexus caller, worker and
standalone activity Models and their tests; its views and Stainless proofs were not taken, and the
Case producer has since been retired: Scala declares a realization and Go builds every Case. Nothing
here reads model/scala, which stays an independent baseline: the gate, `make lint-model` and the Go
tests build, lift and check without it. Source positions in the IR resolve inside model/scalav2, and
`goir` rejects one that does not.

`make lint-model` formats and lints `scala/` like the lifter. A construct the lint rules forbid where
no rewrite keeps the behaviour, such as a cast to an erased type parameter or the declaration
scope's assignable fields, carries a line-scoped `// scalafix:ok <rule>`.

## What is lifted

The lifter reads what an author wrote in `scala/`, as written:

- **Declarations:** `machine[S, O, F](family, name) { … }` blocks, with `forEntity`, `starts`,
  `ends`, `evidence`, `unobservable`, `refines` and `steps(a ~> f, …)`; `restrict`; action chains
  (`action`, `timer`, `on`, `creates`, `input[T]`, `schema`, `results`, `example`).
- **Types:** enums with and without case fields, case classes, and integer fields bounded by the
  `given Finite[Int] = Finite.upTo(…)` beside the state's `Finite`.
- **Step functions:** `if`, native `match` with case, alternative, binding and wildcard patterns,
  local `val`s, `copy`, constructors, comparisons, arithmetic, list literals and `++`, calls of other
  functions, and the kernel prelude's `step`, `one`, `none`, `facts0`, `facts1`. `require` becomes the
  function's precondition; `ensuring` is dropped, since Stainless is what proves it.

Anything else, such as a `var` or a loop, stops the lift with its source line.

## Why TASTy and not a macro

A macro sees a function's body only for definitions of its own compilation run, and there only after
pattern matching has been compiled away into internal trees; for a definition on the classpath it
sees no body at all. TASTy keeps the typed tree with every `match` intact, so the lifter runs after
compilation over `scala/`'s classes. The cost is that a refusal arrives from the lift step, a few
seconds after compiling, rather than as a compile error.

## Results

| Check | Result |
| --- | --- |
| Tables of nexusProduct, nexusProtocol, polling, handlerWorker | Equal to Lean, all 1,152 protocol rows |
| Definition IDs of the same four | Equal |
| Refinement rows | Equal, 1,152 |
| nexusProtocol target fingerprint | `sha256:b3864750…af14`, Lean's |
| Stuck states | None |
| Loader diagnostics | Every problem reported, at its Scala line |
| Lifter refusal | `var` refused at its line in the fixture |

The comparisons with Lean read the dumps under `model/go/parity/testdata/lean`, which are
git-ignored; a checkout without them skips those comparisons.

Properties, Scenarios, Queries, monitors, compositions and progress claims lift, and `goir.Check`
answers them through the generic Go checker, one result per claim (see [SEMANTICS.md](SEMANTICS.md),
Results). The standalone activity lifts whole: `ir/activity.json` is compared with
`model/go/standaloneactivity` on every table row and every Property, and `ir/activity-system.json`
and `ir/nexus-close.json` carry the two design slices with their faulty controls.

A realization lifts too: the roles, learned values, observations, kinds of evidence, controls and
scripts a Model declares for running its find Queries, written with the declarations of
`scala/umpire/realize` (see [SEMANTICS.md](SEMANTICS.md), Realizations). Scala builds no Case.
`goir/testpilot` lowers one Query's witness through the realization into a Testpilot Case that
ordinary `testpilot.Prepare` admits, the same bytes for the same inputs. The Nexus caller's seven
functional Queries lower that way from `scala/temporal/nexuscaller/Realization.scala`.

What a realization declares that Testpilot has no primitive for is named instead of lowered around,
as a limit of the prototype: a durable commit read back through an RPC or from history, and a control
that holds the deliveries of a channel. A control that holds what a step dispatched to a task queue
lowers to the Driver's delivery controls, and the commit its release observes to the release's own
record (`ir/activity-race.json`); the authored monitors of the machine a realization runs are read by
the prepared assessment, beside the Case's Contract. Evidence
that is the Run's own record under a guard, evidence read from one message of a response, the fields
a kind keeps and an attempt answered as canceled lower to Testpilot's own declarations. A class a
path takes more than once is confirmed step by step: a kind of evidence may name the steps of a path
one piece of it confirms. An activity's worker script lowers to the attempts of an activity
entrypoint.

The standalone activity Model declares its realization in
`scala/temporal/standaloneactivity/Realization.scala`, which polls no state the activity passes
through on its own: a call's answer and an attempt's start are the Run's own record, each attempt's
record declared the record of that attempt of the activity's script, and only the paused and the
terminal statuses are read from DescribeActivityExecution. A path that pauses keeps
its Case's worker from polling from before the start until the release, since a pause is read back
only of an activity no worker has taken. Six of its nine find
Queries lower to Cases `testpilot.Prepare` admits: `completion`, `nonRetryableFailure`, `retry`,
`terminate`, `pauseResume` and `scheduleToStartTimeout`. Three name a limit of the prototype that no
task owns. `startToCloseTimeout` starts an attempt it gives no answer, and no instruction of an
activity entrypoint waits. `cancel` and `cancelRequest` request the cancellation while the attempt is
held, and a Run records an attempt once it is answered, so the attempt's record would reach the Run
after the cancel request's answer, out of the path's order. What a worker reports of an activation
is admitted as the record of a named attempt alone, and a Case that would run two activities is not
lowered, since a Run's record of an attempt names no script. The six lowered activity Cases run
against the in-process server and replay to the same Verdict and assessment
(`tests/testpilot_scala_generated_test.go`). Offline played-Driver coverage remains in
`goir/conformance/played_test.go`. Apart from the held race and the bounded admission-response-loss slice, the activity's system designs and the
Nexus close designs declare no realization yet, so their find Queries have the standing
`no-realization`; a verify Query realizes nothing.

Every Case of the Nexus caller carries the five history kinds, which its realization declares
exhaustive, whatever its path records. `syncCompletion` therefore concludes on its witness Run: a
history the closing read shows to hold no started event is the history of a synchronous completion.
The other six Properties stay inconclusive on their witness Runs, for reasons a closed history does
not touch (`goir/conformance/nexus_test.go`).

Quint and P read the same IR. `backends/run.sh` has Quint 0.33.0 evaluate all 29 machines of the
four files and the 6 compositions `goir` builds, and agree with `goir` on 42,506 state and class
pairs, 2,378 monitor product steps and 52,772 Property readings, and has P 3.1.0 agree with Go's
`terminalFinality` monitor on 1,654 bounded event traces. Apalache checks the two activity designs'
monitors and does not take the Nexus close module. Two compositions have no table in Go, because
`goir` rejects their replacement, and are not exported. Refinements, Queries and progress claims are
not exported, and no P module refinement is claimed ([backends/README.md](backends/README.md)).

The specimens record the constructs they found outside the subset in `specimens/README.md` (findings
F2, F5-F7).

## Generated live Cases

`make umpire-gen-model` lifts the Models and writes `cases/*-case.json` plus the versioned
`cases/manifest.json`. The manifest accounts for every checked-in IR Query: lowered,
nothing-to-realize, no-realization, or unsupported with located reasons. The generator builds and
validates a complete temporary tree before replacing it; `make umpire-check-model` compares the
whole inventory and fails on changed, missing, or obsolete files. Ordinary Go tests never rewrite it.

A lowerable Query declares `.expect(RunExpectation(...))` in Scala: trace conformance, the Query
Property's outcome and reason, and each additional monitor's outcome and reason. These are live
assessment expectations, separate from the model-search answer and the Case Contract. Missing or
malformed expectations fail admission/generation. The held race's admission Property is satisfied;
its active-attempt monitor stays inconclusive because rejected admission never reaches that
monitor's `readAfter(attemptAdmitted)` evaluation point.

`TestTestpilotScalaGeneratedCases` discovers every lowered file, binds two independent namespaces
and queues, runs them concurrently twice, and compares the Contract Verdict and declared assessment
both live and replayed. It exercises both Nexus implementations and the in-process delivery control.
The canary boundary test reads the same generated completion file. An unsupported capability is
reported before provisioning Case resources. `umpire-run --case model/scalav2/cases/<file>` consumes
these same Cases on external endpoints; a Case requiring delivery control is skipped with its
located preparation reason before dialing or provisioning. As other preparation failures do, it
returns exit code 3.

The removed per-Query tests also asserted details the current Contracts do not fully express.
Recorded gaps are exact returned activity payload equality (`done`), UUID spelling and unique
attempt-delivery IDs, the complete SDK attempt/response sequence beyond correlated evidence,
learned server-ID isolation by cross-namespace NotFound reads, raw Nexus scheduled endpoint/history
attribute equality, and workflow-backed versus standalone activity parity. These are not claims of
the generated runner. The Contract still checks the authored correlated evidence path (including
pause before unpause before the completed attempt), terminal evidence, and its supporting events;
the runner preserves independent resource bindings, unique Run IDs, every realized declared fault,
and the generic removal-of-durable-evidence ambiguity control. Closing these gaps requires Scala
observations/Contracts or a separate adapter parity test, not Query-specific Go assertions.

The generated `activity-race-admissionResponseLoss.committed-case.json` realizes one
`ADMISSION_RESPONSE_LOSS` after holding the Run's dispatch. Its Scala Model has a one-loss budget
and permits a committed update or a failed update behind the missing answer. The successful control
replaces one successful `RecordActivityTaskStarted` response with `Unavailable`, then waits for a
retry with the same activity execution, delivery stamp, and request ID before reporting success.
Its outcome preserves the first handler's durable admission, including the activity Run and attempt;
the retry's answer never changes that decision. The generic runner checks the declared satisfied
assessment, live/offline agreement, missing-durable-evidence ambiguity, independent concurrent Runs,
cleanup, and exactly one loss event (plus the separately declared hold event).

This cut supports the in-process history client's same-request retry path only. It does not claim
physical task redelivery, failed-commit injection, persistence failure, or a remote/canary actuator.
An unobserved retry, cancellation, refusal, or incomplete injection records no successful loss event.

## Bounded discovery and replay

A find Query can declare `.explore(Exploration(...))` with finite alternatives replacing named
positions of its Scenario's prefix, integer priorities, a Run budget, and a bounded prefix-deletion
sweep. These are IR declarations; `explore/` enumerates their Cartesian product and asks the same
Go checker and Case producer to realize every candidate. `nexusDeadlines` varies the synchronous
call's deadline classes. The highest priority candidate has a start-to-close deadline absent from
pinned Scenarios. Enumerating three candidates is exact finite model coverage; running the first
candidate is one sampled runtime execution. Neither implies an exhaustive server schedule search.

Build `./tools/umpire/cmd/umpire-ir-bridge` and pass that executable to the existing `umpire-fuzz`
or `umpire-replay` command with `--model-root model/scalav2`. The bridge implements their existing
initialize/admit, next, observe and finish protocols. It exchanges whole checked Cases. The
`nexusControl` exploration in `scala/temporal/nexuscaller/Control.scala` deliberately admits a
forged success alongside the real failed callback. The runtime sends an actual asynchronous failed
callback; this negative control is a demonstration of the mechanism, not a platform regression.

Replay first requires two fresh Runs with the original failure key. The generic replay reducer
then tries the declared prefix deletions in reverse order, re-answering and re-lowering the Query
before every attempt and retaining an edit only after two Runs reproduce the same key. Removing
required scheduling or learned callback authority is rejected by that checked path. An incomplete,
indeterminate, or unreproduced failure produces no regression proposal. The deterministic proposal
contains the original IR, source Query, selected alternative, accepted edits, and exact Case bytes
and identities. `umpire-ir-bridge proposal <regression.json>` verifies the recipe and writes its
Case, which can be passed to `umpire-run`. Physical namespaces and endpoints remain execution
bindings. Proposals are written exclusively by the existing replay proposal writer.

`TestTestpilotScalaExplorationDiscoversUnpinnedExecution` and
`TestTestpilotNexusControlReplaysThroughTheCommand` run the discovery and reduction against an
in-process server. Set `UMPIRE_EXPLORATION_DIR` to retain their Cases, recorded Runs, finite versus
sampled coverage, reduction report, replayable proposal, and local HTML traces. Traces place the
checked model witness and abstract product projection beside whole recorded Testpilot events,
monitor state, evidence, fault decisions, holes, and source links. A predicted internal step is
labelled as an expectation; the display does not claim unobserved server commitments occurred.

The fn-107 authoring exercise added a 16-line feature file under
`scala/temporal/nexuscaller`, declaring its own Scenario and using the existing Property and
realization to explore two deadline alternatives. It changed no framework or Go code. A subsequent
feature-only search-policy edit reversed the alternatives' priorities: the checked Go selection
changed from `bounded` (20 versus 10) to `default` (30 versus 0). On the local warm Scala toolchain,
a deliberately mistyped variation index was diagnosed at its source line in 0.598 seconds;
correcting it compiled in 0.475 seconds, lifted in 2.409 seconds, and enumerated and lowered
both candidates in Go in 3.081 seconds. This records an agent performing the Go-developer
workflow, not a human usability study or a cold-build benchmark. The task evidence retains the
invalid/valid and changed-policy source, exact commands, diagnostics and monotonic timings. The
temporary exercise declaration is not a new production Query.
