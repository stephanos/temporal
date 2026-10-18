# fn-193.1 probe: plain-Scala compositions, run natively

The throwaway prototype behind [../plain-composition-prototype.md](../plain-composition-prototype.md).
Unlike the fn-190.1 Kettle probe it does not bring its own Machine: its members are the framework's
own plain `Machine`s (vars, `transitions`, `effects`, claim sections), and only the composition layer
is new. It is compiled with `model/framework/dsl` and `model/framework/realize`'s **main** sources
(not their tests) and changes none of them. Nothing in the Makefile or the framework gate compiles
this directory.

It lives in package `framework.dsl.compose`, a subpackage of `framework.dsl`, for two reasons: it
reaches the native runtime's `private[framework]` parts (`Vars`, `NativeClaims`, `NativeCheck`,
`effectOf`), which is the only framework hook it needs, and its `Composition` shadows the
record-state one in fixtures written in the same package.

Rerun from anywhere in the checkout (scala-cli comes from the repository's `mise.toml`):

```sh
bash "$PWD/model/design/plain-composition-prototype/run.sh"   # from the repository root
```

It compiles the probe under the framework's strict flags (`model/framework/dsl/project.scala`), runs
its suite, and prints the report. Logs land in `logs/`:

| Log | What it holds |
| --- | --- |
| `compile.log`, `compile.status` | the strict-flag compile, main and test scope |
| `test.log`, `test.status` | the munit suite (`ComposeSuite`) |
| `report.log`, `report.status` | members, syncs, reachable graphs with each composed step, verdicts, witnesses, the negative control, the run-time refusals |

## Layout

| Path | Role |
| --- | --- |
| `compose/Composition.scala` | author vocabulary: `Composition[F]`, members as vals, `sync`, `left(action)`, `after`, `records`, `before`, `find`, `withMember(right = ...)` |
| `compose/ComposeMacros.scala` | compile-time half: the member schema (vals, order, fact cases), the typed `before`, `find`'s target, and the refusals |
| `compose/ComposeRuntime.scala` | native runtime: composed state as the members' snapshots, own and synced steps, claim evaluation with every member restored, BFS, checks, witnesses |
| `fixtures/Joined.test.scala` | `Subject`/`Other` as plain Machines; `Together`, `Reordered`, `Apart`, `LateTogether` |
| `fixtures/Queue.test.scala` | SEMANTICS §17's `Queue` cut down: a sync of two actions each bound by one member |
| `fixtures/Refused.test.scala` | compositions the runtime refuses: shared vars, an ambiguous and an unbound sync |
| `fixtures/Probe.test.scala` | `checkAll`, `invariant`, `assertRefuted`, `refusal` |
| `fixtures/Compose.test.scala` | the suite, with the compile-time refusals through `compileErrors` |
| `fixtures/Report.test.scala` | `framework.dsl.compose.report`, what run.sh prints |

One compiler hazard turned up. An inline method that calls a `@publicInBinary` private
constructor, used in the same compilation run that defines it, sends the Scala 3.9.0 compiler into
an endless loop in its inlining phase (thread dump: `Inliner$InlineCopier.Apply` →
`Denotations.bringForward`, the second run of a unit suspended for a macro). The probe's first
`sync` did exactly that, and `Together` never finished compiling; `sync` now calls a public factory,
`Sync.declared`. Swapping the constructor call for the factory, all else equal, removes the loop
(main-scope fixtures then compile in seconds). The fixtures, suite and report are test scope, so they compile after the probe's DSL in
any case.
