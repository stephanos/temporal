# Review: cmp/quint

## Scores (1-5, 5 best) with one sentence of justification each

1. **Spec fidelity: 4.** Both Models are complete with the Lean section order, every property/scenario/query/set/composition name from SPEC.md, the revised Model 2 (`pauseRequested -> Started` at `standalone_activity.qnt:413`, visible retry rows at `standalone_activity.qnt:119-120`) and all listed pins in `pins.qnt`; the only drift is forced (capitalized constructors, `ResolvedSucceeded`/`AttemptCompleted` renames, disclosed at `README.md:43-46`) plus two inert declarations (`handlerWorker` at `nexus_caller.qnt:726`, `activityWorker` at `standalone_activity.qnt:690`) that nothing reads.
2. **Language plausibility: 3.** Every construct used is real Quint (instances with `from`, record spread, `with`, `tuples`, tuple-destructuring lambdas, `then`/`expect`/`assert`, higher-order `pure def`s, closed-row `match`), but the files commit three name-collision errors (QNT101) that `quint parse` rejects, one of which is the very rule the README lectures about; see Red flags.
3. **Authoring readability: 3.** The step functions read like the Lean ones with different brackets and the properties/scenarios/queries are one-liners, but each machine costs two modules, a repeated eight-line `fire` and `last` bookkeeping, and the sets are string-typed records, so the declarative part reads as code rather than configuration.
4. **Check story accuracy: 4.** The README correctly says pins are `quint test` runtime assertions and not compile time (`README.md:63-64`), and I verified against Quint's source (`quint/src/types/specialConstraints.ts`, `matchConstraints`) that a wildcard-free `match` is typed as a closed variant so a missing arm is a typecheck error; small inaccuracies are the sample error text ("variant and record"), a low step-call estimate, and a stale "TypeScript is the default backend" framing (v0.33 defaults to Rust).
5. **Framework realism: 4.** Finite enumeration, reachability, stuck states, row counts and the mapped-states refinement are real executable pure code in `umpire.qnt:133-183`, and I hand-checked the pinned 158 reachable states, 1152 rows and 744 stutters and they are correct; search does not exist and is disclosed (`nexus_caller.qnt:638-642`), and Limits/Sets/restrict are inert data or convention.
6. **README honesty and library table: 4.** Costs are stated bluntly (`README.md:161-184`), and all five repositories I spot-checked match the table; the one factual miss is the backend default.

### Library spot-checks (`gh api repos/<owner>/<repo>`, 2026-09-29)

| Repo | README claim | gh api result | Match |
|---|---|---|---|
| quint-co/quint (redirect from informalsystems/quint) | pushed 2026-09-28, v0.33.0, maintained | pushed_at 2026-09-28T22:39:44Z, archived false, latest release v0.33.0 (2026-09-28) | yes |
| apalache-mc/apalache | pushed 2026-09-24, v0.62.2, maintained | pushed_at 2026-09-24T18:51:28Z, archived false, latest release v0.62.2 (2026-08-26) | yes |
| quint-co/quint-connect | pushed 2026-05-25, v0.1.1, maintained | pushed_at 2026-05-25T17:21:46Z, archived false, latest release v0.1.1 (2025-12-19) | yes |
| informalsystems/itf-go | 2023-11-10, not maintained | pushed_at 2023-11-10T08:43:08Z, archived false | yes |
| quint-co/quint-trace-explorer | 2026-01-30, maintained | pushed_at 2026-01-30T08:01:04Z, archived false | yes |

## Verbatim snippets (for side-by-side comparison; trim comments)

**`handlerReplyStep` of the Nexus product machine** (`nexus_caller.qnt:118-129`):
```quint
pure def handlerReplyStep(s: ProductState, reply: Reply): List[ProductStep] =
  if (s.phase != Scheduled) [] else
  match reply {
    | SyncSuccess => productStep(Succeeded, NexusOperationCompleted)
    | Async => productStep(Started, NexusOperationStarted)
    | OperationFailed => productStep(Failed, NexusOperationFailed)
    | OperationCanceled => productStep(Canceled, NexusOperationCanceled)
    | HandlerError(retryable) =>
      if (retryable) [] else productStep(Failed, NexusOperationFailed)
  }
```

**`syncSucceeds`** (`nexus_caller.qnt:545-546`):
```quint
val syncSucceeds = claim(fired(HandlerReply(SyncSuccess)),
  state.phase == Succeeded and recorded(NexusOperationCompleted))
```

**`syncReplied` and `syncCompletion`** (`nexus_caller.qnt:606-607`, `:654`):
```quint
run syncReplied =
  init.then(schedule(Unset, Unset, Unset)).then(handlerReply(SyncSuccess))

run syncCompletion = syncReplied.expect(found(syncSucceeds))
```

**`nexusCaller` compose block** (`nexus_caller.qnt:714-782`, trimmed):
```quint
module nexusCaller {
  import umpire.* from "./umpire"
  import nexusCallerVocabulary.*
  import nexusProtocolTable.*
  import nexusProtocol as operation
  import Worker(taskQueue = "handler") as worker from "./umpire"

  pure val handlerWorker: Set[worker::Action] = Set(worker::WorkerStop, worker::Serve)

  action init = all { operation::init, worker::init }
  action workerStop = all { operation::workerStop, worker::workerStop }
  action handlerReply(reply: Reply): bool = all { operation::handlerReply(reply), worker::serve }

  pure val unsynced: Set[Action] = actionClasses.filter(a =>
    match a {
      | HandlerReply(_) => false
      | WorkerStop => false
      | _ => true
    })

  action step = any {
    { nondet a = oneOf(unsynced) operation::fire(a) },
    { nondet r = oneOf(replies) handlerReply(r) },
    workerStop,
  }

  val ended = terminalPhase(operation::state.phase)
  def firedReply = match operation::last {
    | None => false
    | Some(l) => match l.action { | HandlerReply(_) => true | _ => false }
  }
  val repliedByPollingWorker = claim(firedReply, worker::state.phase == worker::Polling)

  run repliedThenStopped =
    init.then(operation::schedule(Unset, Expires, Unset)).then(handlerReply(HandlerError(true)))
      .then(workerStop).then(operation::scheduleToStart)

  val pollingWorkerReplies = asInvariant(repliedByPollingWorker)
  run stoppedWorkerRepliesNothing =
    init.then(operation::schedule(Unset, Expires, Unset)).expect(pollingWorkerReplies)
      .then(handlerReply(HandlerError(true))).expect(pollingWorkerReplies)
      .then(workerStop).expect(pollingWorkerReplies)
      .then(operation::scheduleToStart).expect(pollingWorkerReplies)
}
```

## Line counts

| File | Lines |
|---|---|
| `README.md` | 194 |
| `umpire.qnt` | 294 |
| `pins.qnt` | 246 |
| `nexus_caller.qnt` | 782 |
| `standalone_activity.qnt` | 732 |
| Total | 2248 |
| Two Model files alone (nexus + standalone) | 1514 |

## Red flags

- **Constructor label collision inside one module, in both Models.** `nexus_caller.qnt:276` declares `type TimeoutType = ScheduleToClose | ScheduleToStart | StartToClose` and `nexus_caller.qnt:308-310` reuses the same three labels in `Action` of the same module; `standalone_activity.qnt:263` and `:298-300` repeat this. Quint generates a module-level operator per variant label (`quint/src/parsing/ToIrListener.ts`, "Generate all the variant constructors implied by a variant type definition"), so this is a QNT101 "Conflicting definitions" error from the name collector. The README states this exact rule at `README.md:43-46` and `:177-178`, which makes the miss more misleading.
- **Local definition shadowing a star import.** `nexusProduct` imports `nexusProductTable.*` (`nexus_caller.qnt:204`) which brings `terminalIsFinal` (`:197`), then defines its own `val terminalIsFinal` (`:239`). Same in `activityProduct` for `terminalIsFinal` and `pausedIsNotDispatched` (`standalone_activity.qnt:198`, `:228`, `:233` vs `:187`, `:191`). The language manual says "imports are not allowed to introduce name collisions", and `quint/src/names/collector.ts` flags same-depth same-name entries with different ids.
- **Type name versus constructor name.** `umpire.qnt:41` defines `type Timeout`; both product tables import `umpire.*` and declare an `Action` constructor `Timeout` (`nexus_caller.qnt:111`, `standalone_activity.qnt:96`). Quint's collector stores type and value definitions in one name table and defaults depth to 0 for both, so this also reads as a QNT101 conflict. I am less certain of this one than the two above.
- **Stale backend claim.** `README.md:118-119` and `:136` present TypeScript as the default with `--backend rust` optional and say Rust lacks `--seed`/`--mbt`. Quint v0.33 `quint/src/cli.ts` defaults `--backend` to `rust` for run, test and REPL and forwards `--mbt` to the Rust backend (`cliCommands.ts`); the `evaluator/README.md` checklist the sample cites still says TypeScript is the default and is stale.
- **Exact numbers presented without being run.** `README.md:10` says nothing was run, yet `pins.qnt:61`, `:67`, `:123` pin 1152 rows, 158 reachable states and 744 stutters. I recomputed all three by hand and they are right, but a reader should know they are the author's arithmetic, not tool output.
- **Minor overclaims.** The sample typecheck output at `README.md:76` says "variant and record"; the real unification error for a missing arm is variant against variant. The refinement pin cost at `README.md:66` counts only the outer 6,336 protocol step calls and omits the inner product-row scans (`rowsFrom` over 11 product classes per non-stutter row). The `quint test` output shape at `README.md:89-95` is the older "ok/passed N test(s)" format; the current CLI guide shows `[PASS]`/`[FAIL]` lines.
- **Inert restriction and limits.** `handlerWorker`/`activityWorker` and the `two`/`three`/`four`/`six` Limits are declared but consumed by nothing; restriction is enforced only by which actions `step` names. Disclosed in comments, but a reader of the Model files alone could think they bind.

## Strengths

- **Step functions are the Lean ones.** The protocol `steps` dispatch, `moves`, `productOf` and `terminalPhase` (`nexus_caller.qnt:313-427`) are near-transliterations, and the closed-row `match` in `steps` and `productOf` really does give a typecheck error on a missing arm.
- **The refinement is genuinely executable and correct.** `rejectedRows`/`refines` in `umpire.qnt:171-183` implement the mapped-states rule exactly as SPEC.md describes, are reused by both Models, and the same row predicate doubles as a `quint verify` invariant (`nexus_caller.qnt:521-527`). I walked every non-stutter protocol row of Model 2 through `productOf` and each has a product row, including the three the revision note names.
- **The `last` evidence record is a good idea.** Recording action, before-state, outcome and facts in one `var` (`umpire.qnt:70`) makes same-step claims, transition claims, refinement and the future ITF adapter all read one place, and the README explains why `--mbt` alone is not enough.
- **Pins are richer than required.** Beyond SPEC.md's counts, `pins.qnt` checks stuck states, reachable counts, row counts, stutter classification and stutter invariance (`pins.qnt:117-134`), with comments that explain why each number is what it is.
- **The library table is accurate and blunt.** All five repositories checked with `gh api` match the stated push dates and status, and the no-go rows are called no-go.

## One-paragraph verdict

This sample shows that Quint fits the Umpire model layer unusually well where it matters: the machines, the finite tables, the refinement check and the scenarios are short, real and checkable with existing tooling, and nothing had to be invented beyond a `Step` record and a `last` history variable. What Quint cannot hold is also clear and honestly stated: sets, limits, schemas, parties, evidence and realization are strings a Go adapter must interpret, and there is no search. The single biggest reservation is that the files as written would not pass `quint parse`: three kinds of name collisions, one being the exact module-global-constructor rule the README explains at length, mean a decision maker should read the "typechecker catches it" story as a promise about the language, not as a property this sample has demonstrated on itself.
