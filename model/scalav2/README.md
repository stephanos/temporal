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
      │    activity.json         the standalone activity Model, as model/go/standaloneactivity has it
      │    activity-system.json  the standalone activity's system contract and its dispatch queue
      │    nexus-close.json      the Nexus caller close and reset designs
      │
goir/  ── validate, interpret, derive tables → model/go's umpire.Table
      │
      ├── parity with the Lean dumps: tables, IDs, refinement rows, target fingerprint
      │
goir/testpilot/  ── a find Query's witness through its realization → a Testpilot Case
```

## Run

```sh
model/scalav2/run.sh           # build and test scala/, lift, require every ir/*.json to be current, test
model/scalav2/run.sh --update  # lift and rewrite ir/nexus-caller.json, ir/activity.json, ir/activity-system.json and ir/nexus-close.json
go test ./model/scalav2/...    # the Go side alone, from the checked-in IR, no JVM
```

scala-cli, the JDK and protoc come from the repository's `mise.toml`. After changing the schema, run
`make protoc` to regenerate `api/modelir/v1`.

## Layout

| Path | What it holds |
| --- | --- |
| `proto/internal/temporal/server/api/modelir/v1/ir.proto` | The IR schema, with the other internal protos: types and catalogs, functions as expression trees with `match`, actions, machines, source positions; its Go code is `api/modelir/v1`, from `make protoc` |
| `SEMANTICS.md` | The evaluation rules, which neither side defines |
| `ir/nexus-caller.json` | The Nexus caller and worker Models, lifted with the functional Queries and the realization that runs them |
| `ir/activity.json` | The standalone activity Model, lifted with the claims `model/go/standaloneactivity` declares |
| `ir/activity-system.json` | The standalone activity's system contract, lifted: the admission designs, the dispatch queue's providers and their compositions |
| `ir/nexus-close.json` | The Nexus caller close and reset designs, lifted with their monitors, Queries and progress claims |
| `scala/` | The Scala authoring project: the `umpire` framework with the realization declarations in `umpire/realize`, the Temporal Models, and their munit tests |
| `lifter/` | The TASTy lifter; `testdata/unsupported` is a Model it must refuse |
| `goir/` | Loader, validator and interpreter |
| `goir/testpilot/` | Lowers a find Query's witness through the realization its machine declares into a Testpilot Case, and names what a realization declares that Testpilot cannot run yet |
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
with the fn-107 task that owns the primitive: an activity's worker script (task 13), a durable-commit
observation and a hold-delivery control (task 10), and the authored monitors of the machine a
realization runs (task 12). The activity Models and the Nexus close designs declare no realization
yet, so their find Queries have the standing `no-realization`; a verify Query realizes nothing.

The specimens record the constructs they found outside the subset in `specimens/README.md` (findings
F2, F5-F7).
