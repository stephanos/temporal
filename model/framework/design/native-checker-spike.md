# Native checker portability spike

## Claim sections: a compile-time replacement works on Bank

Task fn-209.1 establishes R1 on the unchanged `samples.Bank.Model`, with its existing eleven
Bank tests. A scratch compiler plugin captures each Unit-valued claim initializer as a typed
callable. The scratch checker evaluates those callables with its existing assertion context and
state restoration. Its class-byte reload path is deleted, and `NativeClaimSections.prepare`
unconditionally throws. The successful run therefore cannot fall back to byte rewriting.

This is a one-sample mechanism proof, not a production replacement or a portable checker build.
The next target is WebAssembly (.5), as the owner required on 2026-10-18. Native-image and Scala
Native are later live-sidecar fallbacks; this note makes no target go/no-go decision yet.

### Mechanism and semantic evidence

`ClaimPlugin` uses the same `StandardPlugin`/`PluginPhase` approach as the existing reify plugin.
Its phase runs after `PostTyper` and before pickling. For a Unit val in a native Model's
`invariants`, `postconditions`, or `queries` section, it replaces the initializer with:

```scala
CompiledClaims.capture(enclosingModel, sectionName, memberName)(originalBody)
```

The by-name block is retained without evaluating it during declaration discovery. Compilation
still expands the existing claim macros, retaining their member names, authored lines, and text.
The runtime dispatch uses the exact Model identity and section name. Each callable has its own
`Stopped` boundary and `HoleReached` handling; the existing branch evaluator handles alternatives.
No Scala compiler, Quotes interpreter, or class-file API executes these bodies at runtime.

The source proof runs all eleven existing `BankTest` cases unchanged, including sparse balance
updates, refusal outcomes, the query witness, and the leaky negative control. Two additional tests
exercise the replacement directly:

- Bank invariants see valid, inflated, then restored state; a second instance has a different
  holder and supply. Its postcondition distinguishes unchanged from incorrectly changed state on
  refusal. `Checker.verifyAll` completes with both actual Bank claims verified in the existing
  four-address, supply-two world.
- Repeated evaluation preserves failed-check short-circuiting before an unsafe helper, a naked
  guard hole as unknown, an independently verified sibling, and a successful alternative after a
  failed branch.

`javap` of the compiled Bank invariant constructor shows an ordinary `Function0` and a call to
`CompiledClaims.capture` with the enclosing Bank instance, `invariants`, and `totalSupplyInv`.
Its preserved output is `compile-time-v3/bank-invariants-javap.log` under the scratch root.

### Recipes and observations

The scratch root is `.flow/tmp/fn209/` in the task worktree. Each version has its own source copy,
workspace, exact shell-escaped command, source/input SHA-256 lists, output log, and numeric status.
The runner refuses to overwrite an existing version. `plugin.sh` similarly preserves each plugin
source/resource snapshot and output jar hash. Commands run serially with `--server=false` and a
600-second foreground timeout; observations come from joined exit codes, not scraped success text.
The conductor preserves the frozen root as `.flow/tmp/fn209-proof-v1/` before worktree removal,
checking `MANIFEST.sha256`. Original command files retain their original absolute paths; later
probes must use new workspaces and adapt those paths, not mutate the retained evidence.

Tools are pinned by the repository's mise configuration: Scala CLI 1.17.1, Scala 3.9.0, Temurin
27+35, ScalaPB runtime-grpc 0.11.20, and MUnit 1.2.0. Native builds retain `-Werror`, deprecation,
feature, unchecked, unused, value-discard, nonunit-statement, safe-init, implausible-pattern, retained
tree, and strict-equality options. No warning exemption was added. The plugin itself compiles
against `scala3-compiler:3.9.0` with those strict options.

The retained immutable IR dependency has SHA-256
`b9e0538afffad2b33b9fc8fc9eb2a3f4e63b10928fa38506b585747535e85c92`.
It is copied into each task-owned scratch tree. The owner-removed repository dependency remains
absent. These are scratch comparisons; none is canonical repository gate credit.

| Version / recipe arguments | Joined result | Meaning |
| --- | --- | --- |
| `run.sh baseline-v1 baseline` | 1; 760 passed, 13 initialization errors | Original broad comparison is red: MUnit discovers nested existing suite fixtures without public empty constructors. Bank 11, short-circuit 24, and claim-hole 12 cases passed inside that failed aggregate. |
| `run.sh baseline-selected-v1 baseline-selected` | 0; Bank 11 passed | Fresh explicitly selected semantic baseline. |
| `run.sh rewrite-disabled-red-v1 red` | 1; compiler errors | Inconclusive: the new test's wildcard import selected DSL `Checker`; corrected with explicit product qualification. |
| `run.sh rewrite-disabled-red-v2 red` | 1; 8 passed, 5 failed | Intended red: both new tests and three existing Bank tests reach the disabled-rewriter exception. |
| `plugin.sh plugin-v1` / `run.sh compile-time-v1 probe <v1 jar>` | Package 0; proof 1 | Package-only success; proof inconclusive on a compiler-generated symbol-less ValDef. Added the required symbol guard. |
| `plugin.sh plugin-v2` / `run.sh compile-time-v2 probe <v2 jar>` | Package 0; proof 1 | Safe-init rejects leaking the initializing section object. Registration changed to enclosing Model plus section name. Flags stayed strict. |
| `plugin.sh plugin-v3` / `run.sh compile-time-v3 probe <v3 jar>` | Both 0; 13 tests passed | Working source proof with rewriting disabled and no reload path. |
| `format.sh` | Both passes 0; identical second-pass hashes | Scalafmt 3.11.5 copied from the existing pinned cache into task scratch; only prototype sources formatted. |
| `plugin.sh plugin-final-v4` | 0 | Formatted plugin packaged; package-only observation. |
| `run.sh native-library-final-v1 compiled-library <v4 jar>` | 0 | Fresh native checker/DSL/realization library packaged with the replacement seam and throwing rewrite sentinel; package-only observation. |
| `run.sh separate-bank-client-final-v1 compiled-client <v4 jar> <native jar>` | 0; 13 tests passed | Fresh Bank author/test compile against that immutable library, without compiling checker/DSL source in the client command. |

The successful plugin jar hash is
`d8e9a73579ec1e444e0ac616db7b64da38878d52974c648c5e39f7ece7a6e969`.
The final formatted plugin hash is
`b9203d1237847f3a7e2b658347732dd2955872bd15ab0561f09318f881f0162d`; the separately
compiled native library hash is
`245e7ff0d35c7b9617186d83eac7efa33f270cf9465676a51e377136b0aef578`.
Each library/client recipe uses a fresh workspace; no local jar was overwritten or reused as a
mutable dependency. The native library has neither Bank classes nor a `ClaimEvaluation.Reload`
class. All compiler and test process handles were joined before handover.

Bank source and test source are byte-identical to the repository; their hashes are respectively
`ece6bd7d01c4cb798b7e691c63e002eb63c5269111053ea3cb2efc1394c727e3` and
`e11473d2da78d7990a507d2338bdb7c328248993bc74d36f04fdf701e3164894`.

The original broad baseline used `a651ad74ea7cc91b4fa4c288dae6f50e34ff4f93`. Before subsequent
snapshots, the clean worktree fast-forwarded to the independently committed ModelSuite admission
fix `ad087b4e70eab24c6e9606c00bd26912e722b3a2`, as the conductor directed. This does not grant
retrospective green credit to the original broad baseline.

### Limits and the WASM handoff

The prototype retains Java reflection for section/member discovery, var accessors, and native
runtime installation. Its small identity registry strongly retains captured owners. It does not
prove collection, general section helper initialization semantics, cross-target library portability, or the
entire claim-authoring language. Bank has no state-dependent helper fields in its claim sections;
generalizing beyond that shape needs explicit semantics and tests before any production adoption.

Compile-time capture removes the need to infer boundaries from JVM debug scopes, rewrite bytes,
and define a fresh class for each evaluation. It does not remove every JVM dependency from the
checker. A production replacement belongs in a separately admitted implementation task.

For .5, the required proof is the real checker/sample in the simulation's wasmtime/wasmtime-go
store beside another module, including GC and exception support, actual GC-heap/state restoration,
identical answers after restore, and 100,000 in-host calls. No WASM, host, snapshot, target binary,
or latency measurement ran in .1. Copying linear memory alone is not evidence of restored WasmGC
objects or host-side state.

The working recipes still use the JVM ScalaPB runtime-grpc dependency and retained generated IR
jar. They establish no Scala.js/Native ScalaPB artifact or generated-type compatibility.

The [Scala.js semantics documentation](https://www.scala-js.org/doc/semantics.html) limits
`java.lang.Class` support and excludes Java reflection. Its
[WebAssembly backend documentation](https://www.scala-js.org/doc/project/webassembly.html), read
on 2026-10-18, requires an ECMAScript 2022+ JavaScript host and says its output is not standalone
Wasm. Those are concrete follow-on surfaces, not reasons to skip a real .5 probe. The
[standard plugin documentation](https://www.scala-lang.org/api/3.7.4/docs/docs/reference/changed-features/compiler-plugins.html)
describes the phase mechanism; this experiment validates the
[actual Scala 3.9.0 compiler tree API](https://github.com/scala/scala3/blob/3.9.0/compiler/src/dotty/tools/dotc/ast/tpd.scala)
by compilation rather than assuming API compatibility from a different release.

The current parent amendment was read separately from the task's committed base and preserved as
`parent-live-20261018.md` (SHA-256
`b81ff4571487056c649cc6a78170e8df7f1176fb15feabea69a2096f47f19d1c`).
It makes WASM the next required target; R1 success supplies neither its host proof nor its snapshot
proof. No held model, export, schema, backend, runtime-Case, or dependent-spec implementation was run.
