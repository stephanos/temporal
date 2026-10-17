# fn-190.1 probe: plain-Scala Models, run natively

The throwaway prototype behind [../plain-scala-models-prototype.md](../plain-scala-models-prototype.md).
It has its own minimal DSL (package `plain`) and shares no source with `model/framework`. Nothing
in the Makefile or the framework gate compiles this directory.

Rerun from anywhere in the checkout (scala-cli comes from the repository's `mise.toml`):

```sh
bash "$PWD/model/design/plain-scala-prototype/run.sh"   # from the repository root
```

It compiles the probe under the native strict flags, runs its suites, prints the report and checks
that a non-exhaustive phase `match` fails the build. Logs land in `logs/`:

| Log | What it holds |
| --- | --- |
| `compile.log`, `compile.status` | the strict-flag compile |
| `test.log`, `test.status` | the munit suites (Kettle, top-level Kettle, Lamps, refusals) |
| `report.log`, `report.status` | schema, transition arms, claim verdicts, progress answers, reload cost |
| `non-exhaustive.log`, `.status` | `negative/NonExhaustive.scala.txt` compiled alone, expected to fail |

## Layout

| Path | Role |
| --- | --- |
| `project.scala` | Scala 3.9.0, JVM 27, the flags of `model/framework/dsl/project.scala` |
| `dsl/Dsl.scala` | author vocabulary: `Machine`/`Facet`, `Actor`/`Internal`, `on`, `check`, `after`, `before`, `choose`, `reject`, `oneOf`, `leadsTo`, `only`, `fair`, `scenario.starts`, `UpTo` |
| `dsl/Macros.scala` | compile-time half: declaration names, the var schema, the typed `before` record, `check` positions, the `on` leaf rewrite and refusals |
| `dsl/Runtime.scala` | native runtime: snapshot/restore of the vars, steps with choice replay, reachable graph, claim evaluation by class reload, progress |
| `dsl/Controls.test.scala` | `assertRefuted` for negative controls |
| `kettle/Kettle.scala` | Kettle in the spec's form (nested `object Kettle`, `object Model extends Machine`) |
| `kettle/Kettle.test.scala` | NativeHistory's and Expect's Kettle cases, default checks, progress |
| `kettle/Refusals.test.scala` | compile-time refusals through `compileErrors` |
| `kettle/Report.scala` | `proto.report`, what run.sh prints |
| `toplevel/Kettle.scala` | the same Kettle as top-level definitions in `package samples.kettle`, Machine named `Kettle` |
| `lamp/Lamps.scala` | the Facets fixture: phase var, facets, `match` transitions, `choose` blocks, `UpTo[3]` |
| `negative/NonExhaustive.scala.txt` | a phase match missing a case (kept out of the main build by its suffix) |
