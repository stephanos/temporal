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
ir/nexus-caller.json  (ProtoJSON of the IR schema; checked in)
      │
goir/  ── validate, interpret, derive tables → model/go's umpire.Table
      │
parity with the Lean dumps: tables, IDs, refinement rows, target fingerprint
```

## Run

```sh
model/scalav2/run.sh           # build and test scala/, lift, require ir/nexus-caller.json to be current, test
model/scalav2/run.sh --update  # lift and rewrite ir/nexus-caller.json
go test ./model/scalav2/...    # the Go side alone, from the checked-in IR, no JVM
```

scala-cli, the JDK and protoc come from the repository's `mise.toml`. After changing the schema, run
`make protoc` to regenerate `api/modelir/v1`.

## Layout

| Path | What it holds |
| --- | --- |
| `proto/internal/temporal/server/api/modelir/v1/ir.proto` | The IR schema, with the other internal protos: types and catalogs, functions as expression trees with `match`, actions, machines, source positions; its Go code is `api/modelir/v1`, from `make protoc` |
| `SEMANTICS.md` | The evaluation rules, which neither side defines |
| `ir/nexus-caller.json` | The Nexus caller and worker Models, lifted |
| `scala/` | The Scala authoring project: the `umpire` framework and Case producer, the Temporal Models, and their munit tests |
| `lifter/` | The TASTy lifter; `testdata/unsupported` is a Model it must refuse |
| `goir/` | Loader, validator and interpreter |
| `gen.sh` | Packages the IR's and the Testpilot protos' Java classes as `gen/ir-proto.jar` and `gen/testpilot-proto.jar` (gitignored) for the lifter and `scala/` |
| `scala.sh` | scala-cli with an exit code that fails on any error it prints |
| [specimens/](specimens/README.md) | fn-107's two reviewed authoring sketches, standalone activity admission and Nexus close/reset, with their trace oracles, proposed extensions, Testpilot gaps and authoring measurements; hand-reviewed, and built by no gate |

## Independence from model/scala

`scala/` began as a copy of model/scala's framework, Case producer, Nexus caller, worker and
standalone activity Models and their tests; its views and Stainless proofs were not taken. Nothing
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

Not lifted yet: Properties, Scenarios and Queries, whose predicates are Scala lambdas; the
composition; the standalone activity. The IR has the expression forms Properties need, so lifting
them, and with them Cases, is the next step.

Of the standalone activity, `activityProduct` lifts today and `activityProtocol` stops at its
varargs `moves` (`scala/temporal/standaloneactivity/Model.scala:306`). The specimens record
the other constructs they found outside the subset in `specimens/README.md` (findings F2, F5-F7).
