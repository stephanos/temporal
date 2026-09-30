# Review: cmp/racket

## Scores (1-5, 5 best) with one sentence of justification each

1. **Spec fidelity: 4.** Both Models are complete, every step arm matches the spec including the revised Model 2 (`pauseRequested -> started` at `standalone-activity.rkt:349`, visible retry rows at `standalone-activity.rkt:129-131`), section order follows the Lean file, and all listed pins exist in `pins.rkt`; the one deviation is the query `handlerError` bound as `handlerErrorQuery` with `#:named handlerError` (`nexus-caller.rkt:524`), which the spec's "names must match exactly" rule does not allow.
2. **Language plausibility: 4.** The macro layer is convincingly expert (static records via `syntax-local-value`, `prop:struct-info` on records, rename transformers for the shared `workerStop`), but the step-function contract `(->* (S?) #:rest (listof any/c) (listof step?))` (`nexus-caller.rkt:122`) requires a variadic procedure and would reject every fixed-arity step function at definition, and `check-memq` (`umpire.rkt:728`) is not a rackunit check.
3. **Authoring readability: 3.** The declarations read as configuration (keyword clauses, one per line), but the step functions and properties are accessor chains like `(eq? (ProtocolState-phase (step-state s)) 'succeeded)` where Lean has `step.state.phase == .succeeded`, quoted symbols litter every scenario, and Model 1 lacks the `phase-is`/`records` helpers Model 2 adds at `standalone-activity.rkt:393-394`.
4. **Check story accuracy: 4.** The README table (`README.md:75-85`) is careful and correct about expansion versus instantiation versus `raco test`, but the cross-machine query check it lists as expansion-time is a stub that always returns true (`umpire.rkt:533-535`), and `umpire.rkt:451-452` claims `#:holds` arity is checked at expansion when the code checks it at instantiation (`umpire.rkt:466-471`).
5. **Framework realism: 4.** Finite enumeration (`umpire.rkt:274-288`, `589-614`) and the refinement walk (`umpire.rkt:633-654`) are real and match the spec's mapped-state rule, but `build-machine` destructures its `refines+map` argument wrongly (`umpire.rkt:616-617`), search is elided to an empty trace list (`umpire.rkt:705-706`), and composition returns an empty machine (`umpire.rkt:664-667`).
6. **README honesty and library table: 5.** Costs are stated plainly (dynamic typing, one namespace per module, Lisp syntax for a Go team, no gRPC, no protobuf, no stateful property tester), and all five dates spot-checked with `gh api` match GitHub exactly.

Library spot-checks (2026-09-29):

| Repo | README claim | `gh api pushed_at` | Match |
| --- | --- | --- | --- |
| `racket/rackunit` | 2026-08-11 maintained | 2026-08-11 | yes |
| `Bogdanp/racket-protocol-buffers` | 2023-12-29 no-go | 2023-12-29 | yes |
| `michaelballantyne/syntax-spec` | 2025-10-14 maintained | 2025-10-14 | yes (borderline, 11.5 months) |
| `emina/rosette` | 2026-07-31 maintained | 2026-07-31 | yes |
| `Bogdanp/rackcheck` | 2024-04-26 no-go | 2024-04-26 | yes |

## Verbatim snippets (for side-by-side comparison; trim comments)

The `handlerReply` step function of the Nexus product machine (`nexus-caller.rkt:129-140`):

```racket
(define/contract (handlerReplyStep state reply) product-step/c
  (if (not (eq? (ProductState-phase state) 'scheduled))
      '()
      (cases Reply reply
        [syncSuccess (productStep 'succeeded 'nexusOperationCompleted)]
        [async (productStep 'started 'nexusOperationStarted)]
        [operationFailed (productStep 'failed 'nexusOperationFailed)]
        [operationCanceled (productStep 'canceled 'nexusOperationCanceled)]
        [(handlerError #t) '()]
        [(handlerError #f) (productStep 'failed 'nexusOperationFailed)])))
```

The `syncSucceeds` property (`nexus-caller.rkt:391-395`):

```racket
(property syncSucceeds
  #:machine nexusProtocol
  #:when [handlerReply 'syncSuccess]
  #:holds (λ (s) (and (eq? (ProtocolState-phase (step-state s)) 'succeeded)
                      (member 'nexusOperationCompleted (step-facts s)))))
```

The `syncReplied` scenario and the `syncCompletion` query (`nexus-caller.rkt:459-462`, `:519`):

```racket
(scenario syncReplied
  #:model nexusProtocol
  #:starts unscheduled
  #:actions ([schedule 'unset 'unset 'unset] [handlerReply 'syncSuccess]))

(query syncCompletion #:find syncSucceeds #:in syncReplied #:limits two)
```

The `nexusCaller` compose block (`nexus-caller.rkt:584-593`):

```racket
(struct NexusCallerState (operation worker) #:transparent)

(compose nexusCaller
  #:for (operation Worker.worker)
  #:state NexusCallerState
  #:members ([operation nexusProtocol] [worker handlerWorker])
  #:sync ([workerStop operation.workerStop worker.workerStop]
          [handlerReply operation.handlerReply worker.serve])
  #:starts (operation.unscheduled worker.polling)
  #:ends (operation.succeeded operation.failed operation.canceled operation.timedOut))
```

## Line counts

| File | Lines |
| --- | --- |
| `nexus-caller.rkt` | 627 |
| `standalone-activity.rkt` | 610 |
| `umpire.rkt` | 737 |
| `worker.rkt` | 78 |
| `pins.rkt` | 191 |
| `README.md` | 215 |
| **Two Model files (nexus + standalone)** | **1237** |

## Red flags

- **Contract arity misuse across both Models.** `product-step/c` and `protocol-step/c` use `->*` with `#:rest` (`nexus-caller.rkt:122`, `:234`; `standalone-activity.rkt:103`, `:221`; `umpire.rkt:568`). Racket's `->*` with a rest contract requires the procedure to accept arbitrarily many arguments, so every `define/contract` step function of fixed arity would fail with "expected a procedure that accepts 1 non-keyword argument and arbitrarily many more". `worker.rkt:50` uses the correct `(-> WorkerState? (listof step?))` form. The README's sample contract error at `README.md:144` quotes the wrong contract as if it worked.
- **`build-machine` destructures `refines+map` wrongly.** The macro passes `(list (cons refines map))` (`umpire.rkt:400`), but the runtime takes `(car refines+map)` as the target and `(cdr refines+map)` as the abstraction (`umpire.rkt:616-617`). The target is therefore always a truthy pair, so even non-refining machines enter the refinement branch and the abstraction is `'()`. The header comment calls this part "the real thing" (`umpire.rkt:18-19`).
- **`#lang umpire` provide conflict.** `(except-out (all-from-out racket) compose set)` together with `(rename-out [umpire-module-begin #%module-begin])` (`umpire.rkt:716-717`) exports two different `#%module-begin` bindings. The exception list needs `#%module-begin` too.
- **Invented rackunit check.** `check-memq` (`umpire.rkt:728`) does not exist in rackunit.
- **Claimed expansion-time check that never fires.** `refines?` is stubbed to `#t` (`umpire.rkt:535`), so the "property and scenario are about different machines" error at `umpire.rkt:524-527` is unreachable, while `README.md:78` lists it under expansion.
- **Search and composition are empty.** `search-traces` returns `'()` and `claim-holds?` returns `#t` (`umpire.rkt:705-706`), so every find-query reports `'not-found` and every verify-query reports `'verified` vacuously. `build-composition` returns a machine with no states and an empty table (`umpire.rkt:667`), so the compose blocks and cross-entity queries bind to nothing. Both are marked "sketched" or "elided", so this is honest hand-waving, but a decision maker should know the pins in `pins.rkt:131-137` and `:184-191` cannot pass as written.
- **Spec name deviation.** The query `handlerError` is bound as `handlerErrorQuery` (`nexus-caller.rkt:524`) and the set lists the long name (`nexus-caller.rkt:540`). Documented in `README.md:160-162`, but it breaks side-by-side comparison.
- **Missing imports.** `extract-struct-info` (`umpire.rkt:288`) needs `racket/struct-info` at phase 1, and `quote-srcloc` (`umpire.rkt:470`) needs `syntax/srcloc`; neither is required.
- **Restriction leaves stale rows.** `restrict-machine` filters the step table but not `machine-table` (`umpire.rkt:660-662`), so `handlerWorker` still carries `workerResume` rows.
- **Borderline maintenance claim.** `syntax-spec` is marked maintained on a 2025-10-14 push (`README.md:200`), eleven and a half months before the stated twelve-month cutoff. It matches GitHub, but it is one push away from no-go.

No semantic drift was found in the step functions: every enabled phase, target phase, and fact list in both Models matches the spec and the Lean file.

## Strengths

- **Binding-based static checks.** Every declaration binds a compile-time record, so `machine` asks whether a step key is a declared action of the binding (`umpire.rkt:374-378`), and the shared `workerStop` is a rename transformer (`nexus-caller.rkt:89`) that `free-identifier=?` sees as the Worker module's action. This is how real Racket DSLs are built.
- **`cases` closes the exhaustiveness gap honestly.** The macro refuses non-constructor patterns and reports missing constructors at expansion (`umpire.rkt:237-269`), and both Models use it consistently on every enum argument.
- **Refinement algorithm matches the spec's rule.** Stutter when mapped states are equal, otherwise any product class between the mapped states (`umpire.rkt:633-654`), and the pins inspect which product class explained specific rows (`pins.rkt:118-127`).
- **Revised Model 2 applied precisely.** Visible retry from started and from cancelRequested (`standalone-activity.rkt:129-131`), `pauseRequested` mapped to started with the spec's rationale in the comment (`standalone-activity.rkt:340-356`), and the `TransitionAttemptFailedWhilePauseRequested` arm (`standalone-activity.rkt:284`).
- **Library table verified.** All five spot-checked pushes match GitHub to the day, no-go libraries are named as such, and the two ecosystem gaps (gRPC, stateful property testing) are stated without spin.

## One-paragraph verdict

This sample shows Racket at its strongest for the Umpire model layer: the framework literally is a language, expansion-time checks on bindings give token-pinned errors for undeclared actions and non-exhaustive matches, and the declarative parts of both Models read like configuration in the Lean file's order with its comments intact. The single biggest reservation is that everything below the macro layer rests on dynamic typing plus hand-written contracts, and the sample's own contracts are misused in a way that would reject every step function at load, which no static check would have caught. For a Go team that mostly reads, the Lisp accessor chains in step functions and properties are the second cost, and they do not go away with more macros.
