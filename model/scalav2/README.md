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
model/scalav2/run.sh           # build and test scala/, lift, require every ir/*.json to be current, test
model/scalav2/run.sh --update  # lift and rewrite ir/nexus-caller.json, ir/activity.json, ir/activity-system.json and ir/nexus-close.json
go test ./model/scalav2/...    # the Go side alone, from the checked-in IR, no JVM
model/scalav2/backends/run.sh  # Quint and P against Go; needs the tools backends/README.md names
```

scala-cli, the JDK and protoc come from the repository's `mise.toml`. After changing the schema, run
`make protoc` to regenerate `api/modelir/v1`.

## Layout

| Path | What it holds |
| --- | --- |
| `proto/internal/temporal/server/api/modelir/v1/ir.proto` | The IR schema, with the other internal protos: types and catalogs, functions as expression trees with `match`, actions, machines, source positions; its Go code is `api/modelir/v1`, from `make protoc` |
| `SEMANTICS.md` | The evaluation rules, which neither side defines |
| `ir/nexus-caller.json` | The Nexus caller and worker Models, lifted with the functional Queries and the realization that runs them |
| `ir/activity.json` | The standalone activity Model, lifted with the claims `model/go/standaloneactivity` declares and the realization that runs its find Queries |
| `ir/activity-system.json` | The standalone activity's system contract, lifted: the admission designs, the dispatch queue's providers and their compositions |
| `ir/nexus-close.json` | The Nexus caller close and reset designs, lifted with their monitors, Queries and progress claims |
| `scala/` | The Scala authoring project: the `umpire` framework with the realization declarations in `umpire/realize`, the Temporal Models, and their munit tests |
| `lifter/` | The TASTy lifter; `testdata/unsupported` is a Model it must refuse |
| `goir/` | Loader, validator and interpreter |
| `goir/testpilot/` | Lowers a find Query's witness through the realization its machine declares into a Testpilot Case, and names what a realization declares that Testpilot cannot run yet |
| [backends/](backends/README.md) | Exports every machine and composition of the four IR files to Quint and one monitor to P, runs both tools, and compares each with `goir`: transitions, monitors and Properties by Quint, bounded event traces by P. Its tool runs are skipped by `go test` and required by `backends/run.sh` |
| `gen.sh` | Packages the IR's Java classes as `gen/ir-proto.jar` (gitignored) for the lifter |
| `scala.sh` | scala-cli with an exit code that fails on any error it prints |
| [specimens/](specimens/README.md) | fn-107's two reviewed authoring sketches, standalone activity admission and Nexus close/reset, with their trace oracles, proposed extensions, Testpilot gaps and authoring measurements; hand-reviewed, and built by no gate |

## Independence from model/scala

`scala/` began as a copy of model/scala's framework, Case producer, Nexus caller, worker and
standalone activity Models and their tests; its views and Stainless proofs were not taken, and the
Case producer has since been retired: Scala declares a realization and Go builds every Case. Nothing
here reads model/scala, which stays an independent baseline: `run.sh`, `make lint-scala` and the Go
tests build, lift and check without it. Source positions in the IR resolve inside model/scalav2, and
`goir` rejects one that does not.

`make lint-scala` formats and lints `scala/` like the lifter. A construct the lint rules forbid where
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
with the fn-107 task that owns the primitive: a durable-commit observation and a hold-delivery
control (task 10), and the authored monitors of the machine a realization runs (task 12). Evidence
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
after the cancel request's answer, out of the path's order. No lowered activity Case has run against
a server: the six run live through Testpilot's executor against a Driver that plays their paths, and
their Runs replay to the same Verdict and assessment (`goir/conformance/played_test.go`). The activity's system designs and the Nexus close designs declare no realization yet, so
their find Queries have the standing `no-realization`; a verify Query realizes nothing.

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
