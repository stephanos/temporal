# Draft confinement on Scala 3.9.0

fn-156.8's capture-checking report (R8), published by fn-156.7. Its evidence (`capture/…` below:
examples, receipts, unabridged diagnostics, verifier) stays in the isolated report checkout
`.worktrees/agent/fn156-capture/.flow/tmp/fn156/capture/`, which is ignored scratch, not a tracked
input. No source change follows from it: capture checking is not adopted.

Model authors do not gain Draft confinement by enabling capture checking on the current API. The pinned compiler accepted and executed three escapes from the unchanged extracted implementation. A scratch API using a scoped capability rejected all four requested escape forms and preserved valid effects, but adopting it would change framework types, Model accessors and the trust boundary of compiled callers. This report supplies a compiler finding, not an adoption change or a whole-Model proof.

## Baseline and reproducibility

The isolated checkout is `agent/fn156-capture` at `557ecca8d0a5e5588fb847dc49c8a7384fae54e8`. It includes committed fn-155.1–.5 under the owner's execution-order amendment. It does not inherit fn-155.6 provenance or passing gates. No tracked file changed and no commit was created.

| Pinned input | Value |
| --- | --- |
| Scala compiler | 3.9.0, verified by the actual compiler's `-version` |
| Compiler JAR SHA-256 | `3e2c09a5ca6cef17a55f2f66e123815f9c4ea061fce20d0e9d2371989405a61a` |
| scala-cli | 1.17.1 |
| JDK | Temurin 27+35, Linux aarch64 |
| `model/framework/Syntax.scala` SHA-256 | `17b848d4efc382844681c99f70daa617fee4891c5640b4f94187cf0b714e125d` |
| `model/project.scala` SHA-256 | `cfb8605d816a37d6b0765cd785f506b582143a4259d8289babb5ae17d1466038` |
| `model/irgen/project.scala` SHA-256 | `31bd4cbeae7d2c7537f1eb8a9cc2df288663d5b06ffe0ca6ff06a7514f1d1cff` |

Source manifest (`capture/baseline/manifest.json`), toolchain and downloaded release-source hashes (`capture/baseline/toolchain.json`), complete compiler receipt index (`capture/index.md`), and verification result (`capture/verification.json`) retain the exact inputs, commands, exit codes and UTC timestamps. Each receipt links unabridged stdout/stderr. The runner's own successful exit only means it wrote a receipt; the receipt's `exit_code` is the compiler observation.

The conservative start is the task claim at `2026-10-10T15:16:59.623944Z`. Actual compiler probes ran from `2026-10-10T15:20:25.797927Z` through `2026-10-10T15:26:28.487006Z`. The verification receipt records the report's final elapsed bound, well below two days. No validation reached one hour, timed out or remained running. There are no stuck-validation deferrals.

All inputs, Coursier and scala-cli caches, Java temporary files, outputs and receipts are under this capture directory. Installed scala-cli/JDK binaries were read in place. Every compiler process used `--server=false`, a separate workspace, two active processors and a 768 MiB Java heap ceiling. These small probes ran independently; no production gate, shared daemon operation or heavy-lock bypass occurred.

The task's disposable-compile Quick command governed this report. The ordinary compiler control and unchanged valid effect were green before testing capture behavior. `gate classify` refused tier-B because the tracked diff is empty; no docs-only or production-gate passing credit is claimed. Parent-spec lint, Model/Case generation and full gates remain with their source/integration tasks.

## What was compiled

Baseline.scala (`capture/examples/Baseline.scala`) contains an exact extraction of `Syntax.scala` lines 88–189, including `View`, `Fixed`, `Recorded`, `Draft`, `is`, `effect`, `record` and `reject`. Extraction metadata (`capture/baseline/extraction.json`) pins those bytes. The small support shell supplies `Ok`, a `Step` data record and a phantom `Owner` witness. It does not construct `Machine`, derive `Finite`, compile ScalaPB, or lift IR. These are minimal compiler reproductions of the scoped API, not full-framework builds.

The first valid run used the baseline warning options `-Werror -deprecation -feature -unchecked -Wunused:imports`. Both a source import and `-language:experimental.captureChecking` worked with the actual Scala 3.9.0 release. No nightly, compiler upgrade or `-experimental` override was needed for these probes. The [pinned enabling documentation](https://raw.githubusercontent.com/scala/scala3/3.9.0/docs/_docs/reference/experimental/capture-checking/how-to-use.md) recommends nightly use; the receipts establish what this release accepted.

Similar-code search found the production scope in `Syntax.scala` and the valid accessor/status examples in `Effects.test.scala`. The experiment reused their forms. It created no alternate production implementation.

## Unchanged API with capture checking

| Case | Compiler observation | Evidence |
| --- | --- | --- |
| Valid local effect | Accepted and executed. State update, fact recording and rejection assertions passed. | 08 (`capture/runs/08-local-valid/receipt.json`), source (`capture/examples/local/Valid.scala`) |
| Returned value | Accepted and executed. A Draft assigned into `S = Any` escaped as the returned `Step.state`. | 09 (`capture/runs/09-local-return/receipt.json`), source (`capture/examples/local/Return.scala`) |
| Retained closure | Accepted and executed. The outer closure invoked `Draft.set` after the effect completed. | 10 (`capture/runs/10-local-closure/receipt.json`), source (`capture/examples/local/Closure.scala`) |
| Object field | Accepted and executed. A holder retained `Option[Draft]` and mutated it after completion. | 11 (`capture/runs/11-local-field/receipt.json`), source (`capture/examples/local/Field.scala`) |
| Retained nested effect using outer Draft | Rejected with E007 because the returned function's scoped root capability could not enter the outer variable. | 12 (`capture/runs/12-local-nested/receipt.json`), diagnostic (`capture/runs/12-local-nested/diagnostics.log`) |
| Retained nested effect capturing no Draft | Also rejected with E007. The broad function result type produces the same scope conflict. | 25 (`capture/runs/25-baseline-nested-independent/receipt.json`), diagnostic (`capture/runs/25-baseline-nested-independent/diagnostics.log`) |

The last control prevents interpreting the nested rejection as a precise proof that the current Draft is confined. It rejects a safe independent effect too. The return example uses `Any` solely to expose a generic Scala escape; it is not a valid finite Model state or evidence that the lifter accepts that Model. Closure and holder examples likewise exceed the lifter's documented straight-line effect grammar. Compiler confinement and the lifter's accepted subset are separate protections.

The initial object-field examples also hit migration errors before their escape behavior could be judged. Runs 03–07 preserve them. In particular, the unchanged `S => List[Step[...]]` result makes inferred effect fields require explicit types. Moving only the probe declarations into a method isolated those incidental errors without changing Draft or effect.

## Sufficient scratch changes and their cost

Candidate.scala (`capture/examples/Candidate.scala`), generated by candidate.py (`capture/candidate.py`), keeps the extracted implementation and changes its types. Draft extends `scala.caps.ExclusiveCapability`; `effect` takes `Draft[S,O,F]^ ?-> Unit` and returns `S -> List[Step[S,O,F]]`; `View.get` and both setters take capture-free functions; and `record`/`reject` explicitly accept a capturing Draft. This is a demonstrated sufficient combination, not a claim that every edit is individually necessary.

The release's [capability definitions](https://raw.githubusercontent.com/scala/scala3/3.9.0/library/src/scala/caps/package.scala) make `Capability` sealed. `ExclusiveCapability` supplies the scoped marker used here. `Mutable` also inherits `Unscoped`, so substituting it would change the intended lifetime restriction. The [capture-type documentation](https://raw.githubusercontent.com/scala/scala3/3.9.0/docs/_docs/reference/experimental/capture-checking/basics.md) defines `->` and `?->` as capture-free arrows. That concerns retained capabilities, not a guarantee that a method has no mutation or observable effects.

| Case against generic candidate | Result and located reason |
| --- | --- |
| Returned Draft in `Step.state` | 20 rejected (`capture/runs/20-candidate-return/diagnostics.log`). Setter function captures `draft` where `Any -> Any` permits no capture. |
| Retained closure | 21 rejected (`capture/runs/21-candidate-closure/diagnostics.log`). Closure cannot charge the inner Draft to the outer variable's capture set. |
| Object field | 22 rejected (`capture/runs/22-candidate-field/diagnostics.log`). Callback Draft cannot flow into `holder.any`. |
| Nested effect retaining outer Draft | 23 rejected (`capture/runs/23-candidate-nested/diagnostics.log`). Nested pure context function cannot capture `outer`. |
| Valid effect fields, state/facts/rejection | 27 accepted and executed under the baseline warning flags (`capture/runs/27-candidate-strict-valid/receipt.json`). |
| Independent nested effect | 26 accepted and executed (`capture/runs/26-candidate-nested-independent/receipt.json`). A retained effect that captures no Draft remains usable. |
| Existing getter/setter form | 28 rejected (`capture/runs/28-candidate-accessors/diagnostics.log`). A capturing Draft cannot supply bare `View[State]`. |
| Getter changed to `View[State]^` | 29 accepted and executed (`capture/runs/29-candidate-accessors-captured/receipt.json`). It preserves explicit-fact-before-status ordering. |

The accessor pair makes the migration cost concrete. Framework annotations alone do not preserve the existing Model getter signatures. AccessorsCaptured.scala (`capture/examples/AccessorsCaptured.scala`) changes that one signature and keeps the status setter's existing shape. The API also excludes ambient capabilities from effect bodies and updater functions. Deliberately using an outer Draft inside a nested effect would require a more expressive dependent-capture contract; the candidate's pure callback rejects that use even if it runs immediately.

A separate scalar scope fixture (`capture/examples/Scoped.scala`) isolates the general lifetime mechanism. Runs 14–17 reject direct return, closure, holder and inner-to-outer escapes. Run 18 (`capture/runs/18-scoped-nested-valid/receipt.json`) permits using an outer Draft inside a shorter inner scope and returns an immutable snapshot. These results agree with the release's [scoped-capability rules](https://raw.githubusercontent.com/scala/scala3/3.9.0/docs/_docs/reference/experimental/capture-checking/scoped-capabilities.md). The generic production `effect` returns `Unit` from its callback, so merely ending a callback with a Draft expression would discard it and would not test return escape.

## Limits and publication hold

The compiler enforces the demonstrated confinement boundary in these checked examples. This spike does not prove arbitrary Scala programs safe. Release-tagged [Setup.scala](https://raw.githubusercontent.com/scala/scala3/3.9.0/compiler/src/dotty/tools/dotc/cc/Setup.scala) and [CaptureSet.scala](https://raw.githubusercontent.com/scala/scala3/3.9.0/compiler/src/dotty/tools/dotc/cc/CaptureSet.scala) deliberately relax capture constraints when interoperating with code compiled without capture checking. [Safe-mode documentation](https://raw.githubusercontent.com/scala/scala3/3.9.0/docs/_docs/reference/experimental/capture-checking/safe.md) separately addresses casts, reflection, unsafe globals and programmer-assumed safety. This spike neither enabled safe mode nor audited those trust boundaries. Confinement also proves no thread safety, exactly-once invocation, rollback or Model semantics.

Experimental APIs propagate experimental requirements to consumers and promise no stable source/binary compatibility. The [pinned experimental-definition rules](https://raw.githubusercontent.com/scala/scala3/3.9.0/docs/_docs/reference/other-new-features/experimental-defs.md) and research notes (`capture/docs/research.md`) record that maintenance cost. Adding `Stateful`/`update` for mutation permissions would be another investigation; the compiled candidate only uses `ExclusiveCapability` for lifetime tracking.

All four requested escape categories have compiler evidence. No category is silently deferred. Remaining unanswered adoption questions are the full framework/Models' annotation inventory, separate-compilation and unchecked-library behavior, lifter/TASTy compatibility, exact IR/Case byte identity, and the complete build/lint/gate cost. They need a separately authorized adoption task and cannot be inferred from these small probes. The existing integration task owns its ordinary gates; this report provides no R9 credit.

Task .7 must compare the eventual closed fn-155 baseline's Draft and compiler/directive hashes with the pinned manifest (`capture/baseline/manifest.json`). If those inputs differ, rerun affected probes from that closed baseline and retain both receipts before publication. Source-lane changes must receive the same assessment. That reconciliation is an explicit publication hold. fn-155.6's missing provenance/gates, deferred fn-157 resource work and unrelated Activity semantics remain uncredited.

To reproduce one observation, run run.py (`capture/run.py`) from this workspace with a fresh receipt name and the same inputs/options recorded in its JSON. For example:

```sh
python3 .flow/tmp/fn156/capture/run.py repeat-return compile examples/Candidate.scala examples/local/Return.scala -- --scalac-option -language:experimental.captureChecking
```

The runner refuses to overwrite an existing receipt directory. Preserve this ignored subtree, including examples, downloaded source snapshots and logs, when handing the report to .7. verify.py (`capture/verify.py`) validates the original 30 observations and their hashes without rerunning the compiler. No source adoption is proposed by this handoff.

## Reconciliation at publication (fn-156.7)

The hold above is discharged against the source lane's final tree rather than fn-155's closure,
which stays an owner-recorded hold (fn-155.6). The extracted `Syntax.scala` lines 88–189, `View`
through `reject`, hash to the pinned `ed6b7160…` at the source lane's head: fn-156 changed
`Syntax.scala` only below them (`Once.keeps`, input-slot identity, rule-table comparisons). The
compiler stays Scala 3.9.0 with scala-cli 1.17.1. The compiler options changed: the Model build
gained `-Wunused:all -Wvalue-discard -Wnonunit-statement -Wsafe-init -Wimplausible-patterns
-language:strictEquality`. Copies of the examples were recompiled with the new warning options
and gave the original results: candidate valid effects accepted and executed, candidate returned
Draft rejected with E007, the unchanged API's valid effect accepted and its returned Draft accepted
and executed. With `-language:strictEquality` added, the examples' own assertions (`==` between
the support shell's `Step` lists, which have no `CanEqual`) fail before any capture result; that is
the probe harness, not a capture finding, and adoption would have to give `Step` its evidence. The
rerun's receipts are in the source lane's `.flow/tmp/fn156/capture-recheck/receipts.txt`.

