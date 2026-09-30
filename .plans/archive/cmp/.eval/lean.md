# Review: cmp/lean/

## Scores (1-5, 5 best)

1. **Spec fidelity: 3.** Model 1 is byte-identical to the real `Temporal/Feature/Nexus/Caller/Model.lean` (verified with `diff`), and Model 2 follows the revised spec closely (names, section order, `pauseRequested -> started` at `StandaloneActivity.lean:414`, visible retry rows at `StandaloneActivity.lean:189-191`), but `Pins.lean` is a verbatim copy of `Nexus/Caller/Tests.lean` and contains zero Activity pins, and no `umpire.lean` framework file is included in the sample directory.
2. **Language plausibility: 3.** Every command form used matches the real grammar in `Umpire/Command/Syntax.lean`, but two things could not elaborate as written: the nine `status*` evidence names would be rejected by the Temporal catalog check, and the protocol row for `attemptResult(failed true)` from `started` would be rejected by the real refinement checker (see Red flags).
3. **Authoring readability: 5.** The declarative blocks (`machine`, `property`, `scenario`, `query`, `set`, `compose`) read as indented configuration with no proof terms, tactics, or type-class ceremony; the only "code" is the step functions, which are short `if`/`match` bodies.
4. **Check story accuracy: 4.** The README's claims are confirmed in the elaborator: `enum` derives `BEq, DecidableEq, Repr, Finite` (`Syntax.lean:156-158`), `refines:` and `compose` prove by `decide +kernel` and reject `sorryAx` (`Syntax.lean:2835, 2875, 3065-3068`), `query` reports at the `find:`/`verify:` keyword (`Syntax.lean:1246-1256`), and `set_option maxRecDepth 65536`/`maxHeartbeats 1000000` are real (`Syntax.lean:2006-2007`); the README omits an example of the failed-query error, though one exists (`Nexus/Tests/Machines.lean:293`: "the search stopped at its declared bound after 1 traces; raise `limits` if the trace you mean is longer").
5. **Framework realism: 5 (by proxy).** The framework is the real 3,920-line elaborator with finite enumeration (`enumerationBound = 16384`, `Finite.lean:123`), a decided refinement witness, and a bounded search backend; nothing is hand-waved, but nothing is included in `cmp/lean/` either, so a reader must open the real repo.
6. **README honesty and library table: 4.** Costs are stated plainly (slow loop, big build, hand-bounded state). Spot-checks via `gh api`: verse-lab/veil pushed 2026-09-29, not archived (matches "maintained"); Lean-zh/protobuf pushed 2026-09-01, not archived (matches "pinned commit, small"); leanprover-community/batteries pushed 2026-09-29. Minor overstatements: "13 `declare_syntax_cat`" is 10 (9 in `Umpire/Command/Syntax.lean` + 1 in `Temporal/Case/Syntax.lean`), "about 25 `elab`s" is 19 (18 + 1), and the "4.6 GB build directory" measures 3.4 GB today (the 240 MB `umpire-lint` binary claim is exact). The two edit-to-feedback timings are not reproducible here.

## Verbatim snippets

**`handlerReply` step of the Nexus product machine** (`NexusCaller.lean:157-168`)
```lean
def handlerReplyStep (state : ProductState) (reply : Reply) :
    List (Step ProductState ProductOutcome ProductFact) :=
  if state.phase != .scheduled then [] else
  match reply with
  | .syncSuccess => productStep .succeeded .nexusOperationCompleted
  | .async => productStep .started .nexusOperationStarted
  | .operationFailed => productStep .failed .nexusOperationFailed
  | .operationCanceled => productStep .canceled .nexusOperationCanceled
  | .handlerError true => []
  | .handlerError false => productStep .failed .nexusOperationFailed
```

**`syncSucceeds`** (`NexusCaller.lean:475-479`)
```lean
property syncSucceeds
  machine: nexusProtocol
  when: handlerReply (syncSuccess)
  holds: fun step =>
    step.state.phase == .succeeded && step.facts.contains .nexusOperationCompleted
```

**`syncReplied` and `syncCompletion`** (`NexusCaller.lean:552-555, 614-617`)
```lean
scenario syncReplied
  model: nexusProtocol
  starts: unscheduled
  actions: [schedule (unset, unset, unset), handlerReply (syncSuccess)]

query syncCompletion
  find: syncSucceeds
  in: syncReplied
  limits: two
```

**`nexusCaller` compose block** (`NexusCaller.lean:744-754`)
```lean
compose nexusCaller
  for: [operation, Worker.worker]
  state: NexusCallerState
  members:
    operation: nexusProtocol
    worker: handlerWorker
  sync:
    workerStop: operation.workerStop ∥ worker.workerStop
    handlerReply: operation.handlerReply ∥ worker.serve
  starts: [operation.unscheduled, worker.polling]
  ends: [operation.succeeded, operation.failed, operation.canceled, operation.timedOut]
```

## Line counts

| File | Lines |
|---|---|
| NexusCaller.lean | 777 |
| StandaloneActivity.lean | 728 |
| Pins.lean | 474 |
| Worker.lean | 96 |
| README.md | 59 |
| **Two Model files (nexus + standalone)** | **1505** |

## Red flags

- **Invented realization.** `Temporal.Case.Realization.standaloneActivity` (`StandaloneActivity.lean:671, 675`) does not exist. The real namespace holds only `asyncNexus`, `workflowStart`, `workflowOutage` and `unaryRpc` (`Temporal/Case/Realization/{Nexus,Workflow,Rpc}.lean`). The one-string-argument shape is also a guess: `asyncNexus` takes two strings and `workflowOutage` takes none (`Outage/Model.lean:145`). The real `case` elaborator is Nexus-specific (`"unsupported Nexus evidence line"`, `Temporal/Case/Syntax.lean:239`).
- **Refinement would be rejected as written.** The real checker requires a matching product row to have the same outcome and all of the product's facts among the protocol row's facts (`Umpire/Command/Refinement.lean:112-113`: `carried.state == mappedTo && some carried.outcome == mappedOutcome && carried.facts.all mappedFacts.contains`), not just matching mapped states as SPEC.md describes. Protocol `attemptResult(failed true)` from `started` records `[.attemptCount]` (`StandaloneActivity.lean:351`) while the product row records `[statusScheduled]` (`StandaloneActivity.lean:191`), so the 24 `started-*-attemptResult-failed-true` rows are neither a product step nor a stutter. The spec's own pin "the refinement check passes" would fail; the fix is one line (`[.statusScheduled, .attemptCount]`). Note for all samples: SPEC.md's framework-vocabulary description of the refinement rule understates the real checker (outcome and facts also have to match).
- **Evidence names would not resolve.** `Temporal.Case.Syntax` imports `Temporal.Case.Catalog`, whose `initialize` installs a check that admits only history event kinds, Testpilot Run Event kinds, `ReadKind` bindings (`pendingAttempts`, `scheduledEvent`), or declared `observation`s (`Temporal/Case/Catalog.lean:69-81`). The nine `status*` right-hand names in both `evidence:` blocks (`StandaloneActivity.lean:234-242, 433-441`) are none of those. The file's own header says every fact is a Describe read (`:14-16`), so it needed nine `observation` commands, not one (`attemptCount`, `:122-124`).
- **No Activity pins.** `Pins.lean` is `Temporal/Feature/Nexus/Caller/Tests.lean` unchanged; the spec's Activity pins (9/5 states, 288/120 states/ends, `attemptResult(canceled)` from `started` is `[]`, refinement passes, `terminalHolds` and `pauseHolds` verify) are absent. Had they been written, the two issues above would have surfaced.
- **README counts overstated** (10 syntax categories, not 13; 19 `elab`s, not 25; 3.4 GB build, not 4.6 GB). Cosmetic, but the README presents them as measured.
- **Search budgets copied, not derived.** The `search:` values (`StandaloneActivity.lean:569-582`) reuse the Nexus numbers plus 262144 for `six` with no branching-factor comment like the Nexus one at `NexusCaller.lean:592-594`; the activity has more enabled actions per state (four controls), so `4096` for three-step paths is unverified.

## Strengths

- **Model 1 is the genuine article**, not a sketch: it is the file the drift test, `AUTHORING.md` regions, and 470 lines of `#guard` pins are run against, so every claim about it is verifiable.
- **Model 2 follows the real grammar exactly.** Multi-line `schema:` with `|` continuation (`StandaloneActivity.lean:88-90, 102-105`) is legal: the rule is `"schema:" ident ("|" ident)*` with no column constraint (`Syntax.lean:1322`), and all nine activity request/response message types exist in `Temporal/API/Types.lean`, so the installed schema check (`Temporal/Case/Schema.lean:87`) would accept them. (The lead's question named `enum`; `schema:` is a key of `action`, not `enum`. `enum` itself supports binder constructors such as `failed (retryable : Bool)` per `Syntax.lean:145-146`.) `from:`/`restrict:`, composed `field.value` starts, and bare sync names on a composition property all match real forms.
- **Semantic drift is essentially nil against the spec text.** Every step function, `productOf` arm, property predicate, scenario action list, limit, query, set binding and compose sync was checked line by line against SPEC.md's revised Model 2 and matched, including the subtle `canceled -> []` from `started` and `failed(true)` from `pauseRequested -> paused`. Every other protocol row was walked through the real refinement rule by hand and passes; only the one row class above fails.
- **Comments are preserved and adapted, not dropped**: product-versus-protocol (`:128-132, :252-257`), why faults are ordinary actions (`:67-68, :110-111`), stutter rows (`:377-378`), and the revised `pauseRequested` mapping rationale (`:407-410`).
- **The README states real costs with real numbers** the reader can check: the `set_option` escalations, the enumeration ceiling, the Node/npm prerequisite for Veil, and the pinned commits in `lakefile.lean`.

## One-paragraph verdict

This sample demonstrates that Lean 4's command elaborators give the Umpire model layer the cleanest declarative surface of the eight, with finiteness, exhaustiveness, refinement and bounded search all genuinely decided at compile time by a real 3,920-line framework, and that a new Model can be written in that grammar in about 730 lines that read like the original. The single biggest reservation is that "reads like the original" is not "elaborates": the uncompiled `StandaloneActivity.lean` would fail at its first `evidence:` line and again at the refinement witness, and the sample ships no Activity pins to catch either, which is exactly the failure mode a 3-minute edit-to-feedback loop and a spec whose refinement rule understates the real checker make expensive to discover.
