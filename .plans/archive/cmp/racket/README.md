# Racket

The two Models of `../SPEC.md`, authored against a Racket macro library that also doubles as a
`#lang`. Nothing here has been run through a toolchain; the shapes are what a Racket author would
write, and the README says where each check would happen. Model 2 follows the spec as revised on
2026-09-29: `pauseRequested` maps to `started`, and the product sees a retryable failure.

| File | What it is |
| --- | --- |
| `umpire.rkt` | The framework surface: `syntax-parse` macros, the runtime structs, table building, refinement, the `#lang umpire` module-begin and reader. |
| `worker.rkt` | The worker entity and its `polling` machine (the Lean `Worker` module). |
| `nexus-caller.rkt` | Model 1, in the Lean file's section order. |
| `standalone-activity.rkt` | Model 2, same order. |
| `pins.rkt` | The `rackunit` equivalents of the Lean `#guard` pins. |

## The DSL mechanism

Every declaration is a macro built with `syntax-parse`: `entity`, `enum`, `record`, `action`,
`observation`, `machine`, `compose`, `property`, `scenario`, `limits`, `query`, `set`, and `cases`
(an exhaustiveness-checked `match` on an enum). Clauses are keywords, so a declaration reads like
the Lean one turned sideways:

```racket
(machine nexusProtocol
  #:for operation
  #:state ProtocolState
  #:refines nexusProduct
  #:map productOf
  #:starts (unscheduled)
  #:ends (succeeded failed canceled timedOut)
  #:timers (backoff scheduleToClose scheduleToStart startToClose)
  #:unobservable (backoff)
  #:evidence (...)
  #:steps ([schedule scheduleStep] [handlerReply protocolHandlerReplyStep] ...))
```

Each form binds its name twice: to a runtime value (`nexusProtocol-machine`, a `machine` struct
with its finite table) and, through `define-syntax`, to a compile-time record other forms inspect
with `syntax-local-value`. That is what lets `machine` ask "is `handlerRepy` a declared action?" of
the binding rather than of a string table, and report the answer at the token. Step functions are
plain Racket functions under `define/contract`, so the `S inputs... -> (listof step?)` signature is
enforced at every call the table builder makes, with blame on the Model module.

Nullary enum constructors are symbols (`'scheduled`) and constructors with fields are transparent
structs (`(handlerError #t)`). The choice is deliberate: Racket has one namespace per module where
Lean has one per enum, and a symbol claims no binding, so `ProductPhase` and `Phase` may both have
a `scheduled` in one file. The cost is that `'schedulde` in expression position is only caught where
a contract or `cases` sees it; `cases` refuses a pattern that is not a constructor of its enum,
where plain `match` would have read it as a variable that swallows every case.

`#lang umpire` is the same library installed as a module language: the reader is
`syntax/module-reader`, and the `#%module-begin` appends a `test` submodule that runs every Query.
A Model written that way needs no `require` and no `(module+ test ...)`:

```racket
#lang umpire
(entity worker #:key taskQueue)
(enum WorkerPhase polling stopped)
(record WorkerState [phase : WorkerPhase])
(action workerStop #:party worker)
(action serve #:party worker #:on worker)
(define/contract (stopStep state) (-> WorkerState? (listof step?))
  (cases WorkerPhase (WorkerState-phase state)
    [polling (list (step 'accepted (WorkerState 'stopped) '()))]
    [stopped '()]))
(machine polling #:for worker #:state WorkerState #:starts (polling) #:ends (polling stopped)
  #:steps ([workerStop stopStep] [serve serveStep]))
```

The sample files are `#lang racket` with an explicit `(require "umpire.rkt")` so the mechanism is
visible; the `#lang` form is the one a team would ship.

## When each check runs

| Check | When | Where |
| --- | --- | --- |
| Clause shape (missing `#:for`, duplicate `#:steps`, `#:refines` without `#:map`) | expansion | `syntax-parse` `~once`/`~optional` with `#:description` |
| Step names an undeclared action; scenario names an action the machine has no step for; restriction names an unknown action; query pairs a property and scenario of different machines | expansion | `syntax-local-value` + `raise-syntax-error` |
| `cases` exhaustiveness and constructor spelling | expansion | `cases` macro |
| Step function signature (`S inputs... -> (listof step?)`) | each call, at module instantiation | `define/contract` |
| Finite table (all states × all action classes) | module instantiation | `build-machine` |
| Refinement (every protocol row is a stutter or some product transition between its mapped states) | module instantiation; raises `exn:fail:umpire` | `check-refinement` |
| Query search (`find` found, `verify` verified) | `raco test` | `module+ test` in each Model, or the submodule `#lang umpire` appends |
| Pins (state counts, class counts, specific rows) | `raco test pins.rkt` | `rackunit` |
| `#:holds` arity matches `#:when` presence | module instantiation | check emitted by `property` |

"Module instantiation" means `racket nexus-caller.rkt`, `raco make`, or any `require` of the file:
a rejected refinement stops the module from loading at all, so it is one step short of a compile
error in practice, and the same step for `raco test`.

## What an author's mistake looks like

A step naming an undeclared action, pinned to the token by `raise-syntax-error`:

```
nexus-caller.rkt:214:12: machine: step names an undeclared action
  at: handlerRepy
  in: (machine nexusProduct #:for operation ...)
```

A non-exhaustive `cases`, or a misspelled constructor that plain `match` would have accepted as a
variable:

```
nexus-caller.rkt:168:6: cases: non-exhaustive cases on Reply: missing (operationCanceled)
nexus-caller.rkt:170:9: cases: not a constructor of Reply
  at: syncSucess
```

A missing clause, in `syntax-parse`'s own words using the `#:name` given to the clause:

```
nexus-caller.rkt:205:0: machine: missing required occurrence of #:state clause
```

A refinement the map cannot explain, at load:

```
nexusProtocol: refinement of nexusProduct rejected at row backingOff-1-unset-unset-unset-backoff
  context...: build-machine
```

A failed query, from `raco test`:

```
--------------------
nexus-caller.rkt > test
FAILURE
name:       check-eq?
location:   nexus-caller.rkt:412:4
message:    "#<query retry>"
actual:     'not-found
expected:   'found
--------------------
```

A step function returning a bare step instead of a list, from the contract:

```
handlerReplyStep: broke its own contract
  promised: (listof step?)
  produced: (step 'accepted (ProductState 'succeeded) '(nexusOperationCompleted))
  in: the range of (-> ProductState? Reply? (listof step?))
  blaming: nexus-caller.rkt
```

## Toolchain and feedback loop

Racket 8.x, `raco make` for compilation, `raco test` for the submodules, DrRacket or the
`racket-langserver` for editors (VS Code and Emacs have working LSP clients; expansion-time errors
show inline with the exact span). Expansion of a file this size is under a second; building the
192- and 288-state tables and walking the refinement is well under a second; the search for the
`six` limits (262144 candidates) is the only part that would be felt, and it runs only under
`raco test`. The edit-to-first-error loop is the expander's, so a misnamed action is reported in
about the time a Go build reports a type error.

## Honest notes

- **Two namespaces short.** Lean gives every enum its own namespace; Racket gives every module one.
  Symbols for nullary constructors absorb most of the collisions, but not all: the Query the spec
  calls `handlerError` collides with the `Reply` constructor and is bound as `handlerErrorQuery`
  with `#:named handlerError`, and the worker's `Phase` is `WorkerPhase`. Model 2 shares `Timeout`
  with Model 1 by redeclaring it, which is what a Racket author would do rather than import one enum.
- **`(Fin 3)` is a literal.** The record field for the attempt count is written `(Fin 3)` where
  Lean writes `Fin (attemptBound + 1)`: the `record` macro reads the type at expansion, and
  `attemptBound` is a runtime value. A `define-for-syntax` would close the gap at the price of a
  second definition.
- **Dynamic typing.** A step function that puts a `Phase` symbol into a `ProductState` is caught
  only if a contract or the table builder compares it; the finite enumeration does not know the
  symbol is foreign. Typed Racket would close this and cost the macros a typed surface, which is
  real work; it is not in the sample.
- **Exhaustiveness is opt-in.** `cases` checks; `match` and `cond` do not. Every step function in
  the two Models uses `cases` on its enum argument and `cond`/`memq` on phase sets, as the Lean file
  uses `match` and `||`, and the `cond` arms have no exhaustiveness check at all.
- **Lisp syntax for a Go team.** The declarations read well; the step functions read as Racket,
  which is to say `(struct-copy ProtocolState state [phase 'backingOff])` where Lean has
  `{ state with phase := .backingOff }`. Paredit-style editing is a skill the team would acquire or
  suffer without.
- **Tooling is thin but real.** One language server, one IDE, `raco test`, no coverage or
  benchmarking culture to speak of. Error messages from `syntax-parse` are excellent; error
  messages from a runtime `match` failure inside a macro-generated body are not.
- **What was easy.** The DSL itself: the twelve forms in `umpire.rkt` are a few hundred lines, the
  grammars are the documentation, the pinned errors came for free from `raise-syntax-error`, and
  `#lang umpire` was a two-line reader plus a module-begin. Racket is the only language in this
  comparison where "the framework is a language" is literally true rather than a figure of speech.

## Libraries to leverage

Maintenance checked on 2026-09-29 with `gh api repos/<owner>/<repo>`; anything without a push in
the last twelve months is marked no-go. Everything in the first group ships with the Racket
distribution or is one `raco pkg install` away.

| Library | Repo | Last push | Status | Would replace |
| --- | --- | --- | --- | --- |
| `syntax/parse`, `racket/contract`, `racket/match`, `json`, `file/sha1` (SHA-1/SHA-256 in core) | `racket/racket` | 2026-09-29 | maintained | The whole DSL layer, the step-function contracts, JSON output and the fingerprint hash. Already what `umpire.rkt` is built on. |
| `rackunit` | `racket/rackunit` | 2026-08-11 | maintained | The pin and query test harness (`pins.rkt`, the `test` submodules). |
| Typed Racket | `racket/typed-racket` | 2026-09-29 | maintained | Static typing of state records and step functions; would catch a `Phase` symbol placed in a `ProductState`. Costs a typed surface for the macros. |
| Rosette | `emina/rosette` | 2026-07-31 | maintained | Explicit-state search and refinement checking as solver queries: a step function lifted over symbolic state gives "is there a row the map cannot explain" and "is there a path where `holds` fails" in one `verify` each, instead of the hand-written table walk and bounded search. |
| Redex | `racket/redex` | 2026-09-23 | maintained | Machines as reduction relations: `reduction-relation` for steps, `apply-reduction-relation*` for reachable states, `redex-check` for random path generation. Closest thing in the ecosystem to a stateful property-based tester. |
| `syntax-spec` | `michaelballantyne/syntax-spec` | 2025-10-14 | maintained | The compile-time records and name checks in `umpire.rkt`: declares a DSL's binding structure once and derives the scope checking and error reporting the sample hand-rolls with `syntax-local-value`. |
| `crypto` | `rmculpepper/crypto` | 2026-09-04 | maintained | Hashing beyond SHA-2 for the Behavior Fingerprint, if the core `file/sha1` set is not enough. |
| `http-easy` | `Bogdanp/racket-http-easy` | 2026-02-19 | maintained | An HTTP client for the Nexus completion callback and Temporal's HTTP API, in the absence of gRPC. |
| `rackcheck` | `Bogdanp/rackcheck` | 2024-04-26 | not maintained, no-go | Property-based testing. |
| `quickcheck` | `ifigueroap/racket-quickcheck` | 2024-07-30 | not maintained, no-go | Property-based testing. |
| `protocol-buffers` | `Bogdanp/racket-protocol-buffers` | 2023-12-29 | not maintained, no-go | Protobuf encoding for the Case format. |
| `sha` | `greghendershott/sha` | 2022-11-25 | not maintained, no-go | SHA hashing; superseded by the core `file/sha1`. |

Two gaps have no maintained answer. There is no gRPC implementation for Racket at all, and the
only protobuf package is stale, so the Case format would be emitted as JSON and converted by the
Go side, or the Go runtime would own serialization outright. There is also no stateful
property-based tester in the style of Go's `rapid` or Erlang's PropEr; Redex's `redex-check` over a
reduction relation is the nearest substitute, and Rosette covers the bounded-verification half of
what such a tester would do. Neither `rackcheck` nor `quickcheck` offers stateful commands even
when they were current. JSON canonicalization is a few lines over `jsexpr->bytes` with sorted
hash keys; no package is needed.
