# model/scalav2

Scala as the authoring front end of a language-independent Umpire IR, with Go as the only thing
that runs it. The Models are model/scala's, unchanged; a lifter reads their compiled typed trees and
emits the IR, and a Go interpreter derives every table, identity and fingerprint from the IR alone.
The architecture is the one `SCALA.md` proposes ("Scala is syntax. The IR is the specification. Go
interprets the specification.").

```text
model/scala (Scala Models, Stainless kernel)
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
model/scalav2/run.sh           # lift, require ir/nexus-caller.json to be current, test
model/scalav2/run.sh --update  # lift and rewrite ir/nexus-caller.json
go test ./model/scalav2/...    # the Go side alone, from the checked-in IR, no JVM
```

scala-cli, the JDK and protoc come from the repository's `mise.toml`. After changing the schema, run
`make protoc` to regenerate `api/umpire/v1`.

## Layout

| Path | What it holds |
| --- | --- |
| `proto/internal/temporal/server/api/umpire/v1/ir.proto` | The IR schema, with the other internal protos: types and catalogs, functions as expression trees with `match`, actions, machines, source positions; its Go code is `api/umpire/v1`, from `make protoc` |
| `SEMANTICS.md` | The evaluation rules, which neither side defines |
| `ir/nexus-caller.json` | The Nexus caller and worker Models, lifted |
| `lifter/` | The TASTy lifter; `testdata/unsupported` is a Model it must refuse |
| `goir/` | Loader, validator and interpreter |
| `gen.sh` | Packages the IR's Java classes as `gen/ir-proto.jar` (gitignored) for the lifter |

## What is lifted

The lifter reads what an author wrote in model/scala, as written:

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
compilation over model/scala's classes. The cost is that a refusal arrives from the lift step, a few
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
