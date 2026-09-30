# Review: cmp/nim

## Scores (1-5, 5 best) with one sentence of justification each

1. **Spec fidelity: 5.** Both Models are complete with every name, section, limit, query and set as SPEC.md lists them, the Lean comments carried over nearly verbatim, the revised Model 2 applied (`pauseRequested` maps to `started` at `standalone_activity.nim:431`, visible retry rows at `standalone_activity.nim:171-174`), and every pin present in `pins.nim`.
2. **Language plausibility: 4.** The colon-block macros, `macrocache` registries, `static`-parameter `admit`, and case-guarded variant construction inside `finite` are expert-grade Nim, but the files carry roughly twenty same-module name collisions where the README admits four, plus a `strVal` on a `DotExpr` and a `bindSym` ordering slip, so neither Model module compiles as written.
3. **Authoring readability: 4.** The declarative half reads as configuration and mirrors the Lean line for line; the costs are backticked keywords on every block and `Phase.`/`ReplyKind.` qualification through most of the activity step functions.
4. **Check story accuracy: 3.** Most rows of the README table are real and the five sample error messages match the code's format, but the two `verify` queries of product claims on protocol scenarios cannot type-check (no map in `search`), bare-identifier classes in `when:`/scenarios are not validated despite the claim, and the search-cost narrative describes a candidate enumeration the code does not perform.
5. **Framework realism: 4.** Finite enumeration and the refinement walk are implemented faithfully to the revised rule; search exists but only walks the scenario path and cannot bind a product property to a protocol scenario, and composition is an honest sketch that the pins nonetheless assert against.
6. **README honesty and library table: 4.** Costs are stated plainly (stropping, macro debugging, VM speed, hiring, thin protobuf/gRPC chain); all four spot-checked library rows match GitHub exactly, but the collision table undercounts what the sample itself relies on.

Library spot-checks via `gh api` (all match the README's dates and status):

| Repo | README says | GitHub |
| --- | --- | --- |
| nitely/nim-grpc | 2026-06-13, maintained, 25 stars | pushed 2026-06-13, not archived, 25 stars |
| status-im/nim-protobuf-serialization | 2026-09-11, maintained | pushed 2026-09-11, not archived |
| PMunch/protobuf-nim | 2023-10-17, no-go | pushed 2023-10-17, not archived |
| nim-lang/fusion | 2025-11-24, low activity | pushed 2025-11-24, not archived |

## Verbatim snippets (for side-by-side comparison; trim comments)

`handlerReplyStep`, `nexus_caller.nim:163-173`:

```nim
proc handlerReplyStep*(state: ProductState, reply: Reply): seq[ProductStep] =
  if state.phase != scheduled: return @[]
  case reply.kind
  of ReplyKind.syncSuccess: productStep(succeeded, nexusOperationCompleted)
  of ReplyKind.async: productStep(started, nexusOperationStarted)
  of ReplyKind.operationFailed: productStep(failed, nexusOperationFailed)
  of ReplyKind.operationCanceled: productStep(canceled, nexusOperationCanceled)
  of ReplyKind.handlerError:
    if reply.retryable: @[] else: productStep(failed, nexusOperationFailed)
```

`syncSucceeds`, `nexus_caller.nim:457-460`:

```nim
property syncSucceeds:
  machine: nexusProtocol
  `when`: handlerReply(syncSuccess)
  holds: step => step.state.phase == succeeded and nexusOperationCompleted in step.facts
```

`syncReplied` and `syncCompletion`, `nexus_caller.nim:522-525` and `:596-599`:

```nim
scenario syncReplied:
  model: nexusProtocol
  starts: unscheduled
  actions: [schedule(unset, unset, unset), handlerReply(syncSuccess)]

query syncCompletion:
  find: syncSucceeds
  `in`: syncReplied
  limits: two
```

`nexusCaller`, `nexus_caller.nim:711-721`:

```nim
compose nexusCaller:
  `for`: [operation, Worker.worker]
  state: NexusCallerState
  members:
    operation: nexusProtocol
    worker: handlerWorker
  sync:
    workerStop: operation.workerStop || worker.workerStop
    handlerReply: operation.handlerReply || worker.serve
  starts: [operation.unscheduled, worker.polling]
  ends: [operation.succeeded, operation.failed, operation.canceled, operation.timedOut]
```

## Line counts

| File | Lines |
| --- | --- |
| nexus_caller.nim | 741 |
| standalone_activity.nim | 755 |
| umpire.nim | 676 |
| pins.nim | 133 |
| worker.nim | 86 |
| README.md | 237 |
| **Two Model files** | **1496** |

## Red flags

- **Name collisions are undercounted by about five times.** The README table (`README.md:132-137`) lists four. Also colliding in one module: the `polling` machine const versus the `Phase.polling` enum field (`worker.nim:22-24` vs `:78`), the five protocol fact consts versus the non-pure `ProductFact` fields of the same names (`nexus_caller.nim:146-152` vs `:293-299`), the eight `status*` consts versus `ProductFact` (`standalone_activity.nim:136-145` vs `:293-302`), and the `completed`/`canceled` consts (`standalone_activity.nim:49-51`) versus the later `ProductPhase` and `Phase` fields. The README's claim that pure kind enums "already dodge this" (`README.md:139`) misses that the product enums are not pure. The sample even uses `Worker.polling` for the machine at `nexus_caller.nim:702` and the phase at `:728`.
- **The two product-claim verifies cannot run.** `query` passes the property's machine to `search` (`umpire.nim:597, 605-606`), so `terminalHolds` becomes `search(nexusProduct, terminalIsFinal, asyncThenSucceeded, ...)` with `Scenario[ProtocolState]` against `Machine[ProductState]`, a generic mismatch; there is no map-through anywhere in `search` (`umpire.nim:246-286`). The README lists verify as a compile-time VM check (`README.md:72`) and the pins assert both pass (`pins.nim:104-105, 130-131`). Same for `pauseHolds`.
- **Search is a path walk, not a search.** `search` follows only the scenario's listed actions (`umpire.nim:256-266`); `limits.actions` is never read and `limits.search` is practically unreachable. The copied Lean comment about ninety-nine candidates (`nexus_caller.nim:566-568`) and the README's "262144-candidate search" (`README.md:198`) describe a different engine.
- **Bare-identifier classes are unchecked.** `actionClass` discards the lookup for an `nnkIdent` and `lookupOrTimer` never errors (`umpire.nim:330-334, 343-345`), so a misspelled timer in `when:` or a scenario passes the macro and surfaces only as "no row ... on the path". `README.md:69` claims the compiler catches every wrong class.
- **Composition cannot expand as written**, beyond the acknowledged sketch. `nameOf` calls `strVal` on the `Worker.polling` dotted node (`umpire.nim:306, 432-435`), `actionClass` does the same on `operation.schedule(...)` (`:347`), `startState` routes through `phaseStates` which needs a `.phase` field on the composed state (`:411, :669-671`), and `product` returns nothing (`:660-667`). The `stoppedWorkerRepliesNothing` and `stoppedWorkerStartsNothing` pins (`pins.nim:107, 133`) therefore pin nothing.
- **Ambiguous references in pins.nim.** With both Model modules imported, `retry`, `scheduleToStartTimeout`, `startToCloseTimeout` (`pins.nim:99-100`), `terminalHolds` (`:104-105`) and `attemptBound` (`:72`) are unqualified while the same names are qualified at `:126-127, :130`.
- **Small framework slips.** `bindSym"phaseStates"` (`umpire.nim:409`) precedes the declaration it binds (`:411`); the `set` macro's comment says it rejects non-`find` queries (`:612-613`) but only checks existence (`:620-622`).

## Strengths

- **Refinement matches the revised rule exactly.** `refinementOf` (`umpire.nim:170-196`) treats equal mapped states as stutters and otherwise accepts any product class between the mapped states, preferring the same class, and the pins probe precisely the interesting rows (`pins.nim:92-95`).
- **Finite enumeration is genuinely derived.** The `finite` macro (`umpire.nim:92-143`) builds the cartesian product per variant branch using case-guarded object construction, and `key` falls out of `fieldPairs` visiting only the active branch (`:83-90`), so "handlerError-true" costs one generic proc.
- **Node-pinned admission is a real Nim technique.** `admit` with a `static Refinement` parameter (`umpire.nim:364-374`) lets a VM-computed rejection report at the author's `refines:` or `find:` line, and the README explains it in one paragraph (`README.md:77-80`).
- **Undeclared-step detection is implemented, not asserted.** The `machine` macro consults the cross-module `macrocache` registry and the machine's own `timers:` (`umpire.nim:455-462`), which is what lets `nexus_caller.nim` step on the `workerStop` that `worker.nim` declared.
- **The activity revision is done carefully.** Product retry rows (`standalone_activity.nim:166-180`), `pauseRequested` reading as `started` (`:428-438`), and the `cancelRequested` retryable-failure-cancels arm (`:354-360`) each carry a comment tying them to `statemachine.go`.

## One-paragraph verdict

The Nim sample shows that Nim's parser and compile-time VM can host this model layer with almost no DSL machinery: the colon-block form gives a Lean-shaped surface for free, plain procs build the tables and walk the refinement inside a `const`, and macro errors land on the author's own line. The declarative half reads as well as any sample could while keeping the spec's spelling, and the framework's enumeration and refinement are real code a reviewer can trace. The single biggest reservation is that the sample's correctness story is weaker than its README says: the two `verify` queries that carry a product claim to a protocol scenario, which are the spec's headline mechanism, have no implementation and would not type-check, and the same-module name collisions the spec's flat vocabulary induces are several times more numerous than the four the README lists. Both are fixable (a map argument to `search`, `{.pure.}` on the product enums or a module split), but a decision maker reading the README alone would believe more is checked at compile time than the code can deliver.
