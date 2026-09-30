# Walkthrough: the standalone activity Model in Racket

## What this is

Umpire is model-based testing with three parts. A Model is the rulebook: it says which moves each
party may make and what the server must show afterwards. A run of the real system is a
playthrough. The referee reads the playthrough against the rulebook and issues a verdict. A
standalone activity is a Temporal activity started directly with `StartActivityExecution`, with no
workflow around it; it writes no history events, so everything the rulebook can check is a status
read through `DescribeActivityExecution` or a result read through `PollActivityExecution`. The file
`standalone-activity.rkt` is the rulebook for one such activity, written against the small
framework in `umpire.rkt`, and `pins.rkt` is a set of tests that pin what the rulebook says.

## The language in five minutes

Racket is a Lisp: every form is a parenthesized list whose first element is the operator, so
`(f x y)` calls `f` and `(if c a b)` is a conditional. A form starting with `#:` is a keyword
argument, used here to make declarations read like configuration:

```racket
(entity activity #:key activityId)
```

A quoted identifier is a symbol, an interned constant name. This Model uses symbols for phases and
facts, so `'scheduled` is the value "scheduled" and `(eq? phase 'scheduled)` compares by identity.

A struct is a record type. `(struct StandaloneActivityState (activity worker) #:transparent)`
defines a constructor of the same name, a predicate `StandaloneActivityState?`, and one accessor
per field, written `Type-field`. `#:transparent` makes two structs with equal fields `equal?`.
`struct-copy` builds a new struct with some fields replaced, as in
`(struct-copy ProtocolState state [phase 'started])`.

A macro is a function from syntax to syntax that runs when the file is compiled. `syntax-parse`
is Racket's macro-writing library: a macro states the grammar of its input once, and any input
that does not fit is rejected with an error pointing at the offending token. Every declaration in
this Model (`enum`, `record`, `action`, `machine`, `property`, ...) is such a macro. The one you
will see most is `cases`, a `match` on an enum that the macro checks for completeness.

A contract is a runtime type check attached to a function boundary. `define/contract` declares
a function and its contract together; `(-> A? B? R?)` says "two arguments satisfying `A?` and
`B?`, result satisfying `R?`". When a call breaks the contract, the error names the party to blame:

```racket
(define/contract (attemptResultStep state result)
  (-> ProductState? AttemptResult? (listof step?))
```

Finally, `(module+ test ...)` declares a submodule that `raco test` runs and ordinary loading
skips, and `(prefix-in Worker. "worker.rkt")` imports a file with every name prefixed.

## Vocabulary: entities, parties, actions, inputs

An entity is the thing a machine is about. The activity is named by the id the caller chose for
it, because no scheduled event exists to name it by.

`standalone-activity.rkt:21`
```racket
(entity activity #:key activityId)
```

A party is who performs an action. This Model uses `caller` and `worker`; `system` is reserved
for timers the machine owns. An action is a named side effect of a party, with typed finite
inputs. The caller's start request creates the activity and carries three deadlines:

`standalone-activity.rkt:42-48`
```racket
(action start
  #:party caller
  #:creates activity
  #:schema "temporal.api.workflowservice.v1.StartActivityExecutionRequest"
  #:input [scheduleToClose : Timeout]
          [scheduleToStart : Timeout]
          [startToClose : Timeout])
```

`#:schema` names the protobuf message the realization would send. The `action` macro checks at
compile time that `activity` is a declared entity and `Timeout` a declared enum.

The worker has two actions: `attemptStart`, its poll receiving the task, declared the same way
with no inputs, and `attemptResult`, which reports the attempt:

`standalone-activity.rkt:56-62`
```racket
(action attemptResult
  #:party worker
  #:on activity
  #:schema "RespondActivityTaskCompletedRequest | RespondActivityTaskFailedRequest | RespondActivityTaskCanceledRequest"
  #:input [result : AttemptResult]
  #:examples ([(failed #f)] -> "ApplicationFailure nonRetryable")
             ([(failed #t)] -> "ApplicationFailure retryable"))
```

An input domain is an enum. A constructor may carry finite fields, and each assignment of them is
a separate class:

`standalone-activity.rkt:30-34`
```racket
(enum AttemptResult completed (failed [retryable : Bool]) canceled)

(enum Delivery accepted notFound)

(enum Control pause unpause requestCancel terminate)
```

An action class is one action with one assignment of its inputs. `attemptResult` has four
classes: `completed`, `failed #f`, `failed #t`, `canceled`. `control` has four: one per `Control`
member. `start` has eight, one per assignment of three two-valued deadlines. The `enum` macro
(`umpire.rkt:199-226`) represents this domain by making each nullary constructor a symbol and each
constructor with fields a transparent struct, so `'completed` and `(failed #t)` are both
`AttemptResult?` values, and it defines `AttemptResult-members`, the list of all four classes in
declaration order, by taking the cartesian product of each constructor's field domains.

A fault is an ordinary action of a declared party, not a separate kind. The worker stopping is
the worker module's action, aliased here so the machine and the composition see one declaration.
A rename transformer is a compile-time alias: every use of `workerStop` means `Worker.workerStop`.

`standalone-activity.rkt:74`
```racket
(define-syntax workerStop (make-rename-transformer #'Worker.workerStop))
```

An observation is a derived read that is evidence without being a history event. The attempt
count is one: `(observation attemptCount #:on activity #:read attempt)` at
`standalone-activity.rkt:81-83`.

## State

There are two state types. The product state is only a phase:

`standalone-activity.rkt:92-95`
```racket
(enum ProductPhase
  scheduled started paused cancelRequested completed failed canceled terminated timedOut)

(record ProductState [phase : ProductPhase])
```

The protocol state adds the attempt count and the three deadline fields:

`standalone-activity.rkt:206-213`
```racket
(define attemptBound 2)

(record ProtocolState
  [phase : Phase]
  [attempts : (Fin 3)]          ; 0 .. attemptBound
  [scheduleToClose : Timeout]
  [scheduleToStart : Timeout]
  [startToClose : Timeout])
```

Every field is finite because the framework builds a table with one row per state and action
class, and a table needs a finite list of states. `record` is the macro that makes this list. It
reads each field type at compile time, asks the enum for its members (`Bool` is `(#t #f)`,
`(Fin 3)` is `(0 1 2)`), and defines `Name-members` as the cartesian product of the fields:

`umpire.rkt:282-286`
```racket
     #'(begin
         (struct name (f.name ...) #:transparent #:name info #:constructor-name make)
         (define members
           (for/list ([args (in-list (apply cartesian-product (list fmembers ...)))])
             (apply make args)))
```

So `ProductState-members` has 9 states and `ProtocolState-members` has 12 × 3 × 2 × 2 × 2 = 288.
The attempt count is written `(Fin 3)` rather than in terms of `attemptBound`, because the macro
reads the type before `attemptBound` exists as a value. The count is kept inside its bound by a
saturating successor:

`standalone-activity.rkt:236-237`
```racket
(define (bump state)
  (saturating-succ (ProtocolState-attempts state) attemptBound))
```

`saturating-succ` (`umpire.rkt:577-579`) is `(min (add1 n) bound)`: a third retry keeps the
count at two rather than leaving the finite domain.

## Step functions

A step function takes a state and the action's inputs and returns a list of steps. A step is an
outcome, the next state, and the facts the system must show. The empty list means the action is
not enabled in that state. The `step` struct (`umpire.rkt:56`) has exactly those three fields.

The product `attemptResultStep`, arm by arm:

`standalone-activity.rkt:121-132`
```racket
(define/contract (attemptResultStep state result)
  (-> ProductState? AttemptResult? (listof step?))
  (define phase (ProductState-phase state))
  (if (not (memq phase '(started cancelRequested)))
      '()
      (cases AttemptResult result
        [completed (productStep 'completed 'statusCompleted)]
        [(failed #f) (productStep 'failed 'statusFailed)]
        [(failed #t) (if (eq? phase 'started)
                         (productStep 'scheduled 'statusScheduled)
                         (productStep 'canceled 'statusCanceled))]
        [canceled (if (eq? phase 'cancelRequested) (productStep 'canceled 'statusCanceled) '())])))
```

Unless the activity is `started` or `cancelRequested`, no result is accepted. A completed
result settles it as completed and Describe reads COMPLETED. A non-retryable failure settles it as
failed. A retryable failure from `started` puts it back to `scheduled`, and Describe reads
SCHEDULED again: unlike the Nexus product, this one sees the retry, matching CHASM's
`TransitionRescheduled`. A retryable failure while a cancel is requested settles it as canceled. A
cancel result is honored only if a cancel was requested. `productStep` is a one-line helper that
builds the single accepted step.

The protocol version dispatches on the phase first, then on the result:

`standalone-activity.rkt:266-290`
```racket
(define/contract (protocolAttemptResultStep state result)
  (-> ProtocolState? AttemptResult? (listof step?))
  (define phase (ProtocolState-phase state))
  (define (settle-completed) (moves state 'completed '(statusCompleted)))
  (define (settle-failed) (moves state 'failed '(statusFailed)))
  (cases Phase phase
    [started
     (cases AttemptResult result
       [completed (settle-completed)]
       [(failed #f) (settle-failed)]
       [(failed #t) (list (step 'accepted (struct-copy ProtocolState state [phase 'backingOff]) '(attemptCount)))]
       [canceled '()])]
    [cancelRequested
     (cases AttemptResult result
       [completed (settle-completed)]
       [(failed #f) (settle-failed)]
       [(failed #t) (moves state 'canceled '(statusCanceled))]
       [canceled (moves state 'canceled '(statusCanceled))])]
    [pauseRequested
     (cases AttemptResult result
       [completed (settle-completed)]
       [(failed #f) (settle-failed)]
       [(failed #t) (moves state 'paused '(statusPaused))]
       [canceled '()])]
    [_ '()]))
```

It dispatches on phase because the three phases in which a worker can hold an attempt react
differently to a retryable failure. From `started` the activity backs off and only the attempt
count is evidence, since nothing else changed that Describe would show. From `cancelRequested` the
failure settles the activity as canceled, following `statemachine.go`, where CANCEL_REQUESTED is a
source of Canceled. From `pauseRequested` it lands in `paused`, which is CHASM's
`TransitionAttemptFailedWhilePauseRequested`. Completed and non-retryable results settle the
activity the same way from all three.

Exhaustiveness is enforced by the `cases` macro at compile time. It looks up the enum's
constructor list and refuses the form if a constructor has no clause and there is no `_`
catch-all. It also refuses a pattern that is not a constructor, which matters in a Lisp: plain
`match` would read a misspelled `completd` as a variable that matches everything. The error is
pinned to the token (`umpire.rkt:238-273`):

```
standalone-activity.rkt:274:8: cases: not a constructor of AttemptResult
  at: completd
```

The `cond` forms over phase sets in `protocolControlStep` have no such check; a phase left out of
every branch falls to the `else`.

## The machine and its table

A machine ties the state type to its step functions and says where runs start and end:

`standalone-activity.rkt:368-376`
```racket
(machine activityProtocol
  #:for activity
  #:state ProtocolState
  #:refines activityProduct
  #:map productOf
  #:starts (unstarted)
  #:ends (completed failed canceled terminated timedOut)
  #:timers (backoff scheduleToClose scheduleToStart startToClose)
  #:unobservable (backoff)
```

`#:starts` and `#:ends` name phases; the framework widens each to every state at that phase, so
the protocol machine has 5 × 3 × 8 = 120 end states, not 5. `#:timers` are `system` actions the
machine owns; they need no `action` declaration. `#:unobservable` marks the backoff timer as
recording nothing, so a run cannot be asked to show evidence for it.

`standalone-activity.rkt:377-378`, `386-395`
```racket
  #:evidence ([statusScheduled statusScheduled]
              [statusStarted statusStarted]
              ...
              [attemptCount attemptCount])
  #:steps ([start startStep]
           [attemptStart protocolAttemptStartStep]
           [attemptResult protocolAttemptResultStep]
           [control protocolControlStep]
           [workerStop protocolWorkerStopStep]
           [backoff backoffStep]
           [scheduleToClose scheduleToCloseStep]
           [scheduleToStart scheduleToStartStep]
           [startToClose startToCloseStep]))
```

Each `#:evidence` line maps a fact to the name the referee looks up in the recorded run. For a
workflow the right side would be a history event type. A standalone activity writes no history, so
every right side is a status value Describe returns or the derived `attemptCount` observation. The
two columns happen to be equal here because the facts were named after what Describe shows.

The `machine` macro checks at compile time that every `#:steps` key is a declared action or a
listed timer (`umpire.rkt:374-379`). `syntax-local-value` reads the compile-time record an `action` declaration bound to its name, so a
typo such as `attemptResutl` is an error at that token, before anything runs.

The table is built at load time, when the module is instantiated. `build-machine` enumerates
every action class, calls each step function on every state, and keeps the non-empty answers:

`umpire.rkt:615-621`
```racket
  (define rows
    (for*/list ([st (in-list members)]
                [c (in-list classes)]
                [out (in-value (apply (hash-ref fn-of (first c)) st (second c)))]
                #:when (pair? out))
      (list st (class-key (first c) (second c)) out)))
  (define tbl (table members classes rows))
```

For the protocol machine that is 288 states × 21 classes (8 for `start`, 1, 4, 4, 1, and 4
timers), each call checked by the step function's contract. A row is a state, a class key such as
`"attemptResult-failed-true"`, and the steps. This table is what everything downstream reads.

## Two levels and the refinement

Why two machines? The product machine says what an activity does, as the caller sees it through
Describe: nine phases, one timer, no attempt count. The protocol machine says how the server gets
there: the backoff wait, the pause the worker has not yet acknowledged, which of three deadlines
fired, how many attempts ran. Properties are easiest to state on the product and are carried to the
protocol by a refinement, so one claim covers both.

The abstraction function maps a protocol state to a product state:

`standalone-activity.rkt:355-366`
```racket
(define/contract (productOf state) (-> ProtocolState? ProductState?)
  (ProductState
   (cases Phase (ProtocolState-phase state)
     [unstarted 'scheduled] [scheduled 'scheduled] [backingOff 'scheduled]
     [started 'started] [pauseRequested 'started]
     [paused 'paused]
     [cancelRequested 'cancelRequested]
     [completed 'completed]
     [failed 'failed]
     [canceled 'canceled]
     [terminated 'terminated]
     [timedOut 'timedOut])))
```

Backing off reads as scheduled because the product does not see the wait. Not-yet-started reads as
scheduled because the product begins there. `pauseRequested` reads as `started`, not `paused`,
because the worker still holds the attempt: every answer it can give is a product row from
`started`, and the request itself is a stutter. The first draft of the spec mapped it to `paused`,
and the refinement could not hold.

The rule, as implemented here: for every protocol row (s, a, s'), either `productOf s` equals
`productOf s'` (a stutter), or the product table has some row from `productOf s` to
`productOf s'` under any action class. The check is a function over the two tables:

`umpire.rkt:648-661`
```racket
  (define-values (rows rejected)
    (for*/fold ([rows '()] [rejected #f])
               ([r (in-list (table-rows (machine-table protocol)))]
                [s (in-list (third r))]
                #:break rejected)
      (define before (abstraction (first r)))
      (define after (abstraction (step-state s)))
      (define key (format "~a-~a" (state-key (first r)) (second r)))
      (cond
        [(equal? before after) (values (cons (cons key #f) rows) #f)]
        [(hash-ref product-moves (cons before after) #f)
         => (λ (product-key) (values (cons (cons key product-key) rows) #f))]
        [else (values rows key)])))
  (refinement (reverse rows) rejected))
```

`product-moves` is a hash from (before . after) product states to the first product class that
moves between them. `build-machine` runs this when the module loads (`umpire.rkt:627-633`) and raises
`exn:fail:umpire` naming the first unexplained row, so a broken refinement stops the file from
loading at all, under `racket`, `raco make` and `raco test` alike.

One row by hand. Protocol state `started`, attempts 1, all deadlines unset; action class
`attemptResult-failed-true`. The step function returns one step to `backingOff` with facts
`(attemptCount)`. `productOf` maps the before state to `started` and the after state to
`scheduled`. They differ, so this is not a stutter. The product table has a row from `started` to
`scheduled`: `attemptResultStep` on `failed #t` from `started`, which we read above. So the row is
explained, and the refinement records it as `"started-1-unset-unset-unset-attemptResult-failed-true"
-> "attemptResult-failed-true"`.

The real Lean checker is stricter. It also requires the matching product row to have the same
outcome and its facts to be among the protocol row's facts, compared by evidence name. Under that
rule this row would fail: the product row records `statusScheduled` and the protocol row records
only `attemptCount`. The spec's second revision note says the Lean sample adds `statusScheduled`
to the protocol step and that the other samples implement the mapped-states rule as stated. This
sample does the latter, so the protocol step at line 276 stays as written.

## Properties

A property is a claim about the table. A same-step claim names an action class under `#:when`
and holds of the step that class produces. `completes` says a completed result lands in
`completed` and Describe reads COMPLETED:

`standalone-activity.rkt:403-404`, `415-418`
```racket
(define (phase-is s phase) (eq? (ProtocolState-phase (step-state s)) phase))
(define (records s fact) (and (member fact (step-facts s)) #t))

(property completes
  #:machine activityProtocol
  #:when [attemptResult 'completed]
  #:holds (λ (s) (and (phase-is s 'completed) (records s 'statusCompleted))))
```

`retryCompletes` fixes the whole state, so every field is named. It is the same `#:when` as
`completes`, and only a path with one retry can satisfy it:

`standalone-activity.rkt:428-433`
```racket
(define completedOnRetry (ProtocolState 'completed 2 'unset 'unset 'unset))

(property retryCompletes
  #:machine activityProtocol
  #:when [attemptResult 'completed]
  #:holds (λ (s) (and (equal? (step-state s) completedOnRetry) (records s 'statusCompleted))))
```

A transition claim has no `#:when` and takes two steps, the one before and the one after. Both
transition claims here are on the product machine and are read on the protocol machine through
the map. `terminalIsFinal` (`standalone-activity.rkt:408-412`) says that if the phase before is
terminal, the phase after is the same. `pausedIsNotDispatched` says a paused activity never
becomes started in one step:

`standalone-activity.rkt:457-461`
```racket
(property pausedIsNotDispatched
  #:machine activityProduct
  #:holds (λ (before after)
            (or (not (eq? (ProductState-phase (step-state before)) 'paused))
                (not (eq? (ProductState-phase (step-state after)) 'started)))))
```

The `property` macro checks at compile time that `#:machine` is a declared machine. Whether the
lambda takes one argument or two is checked when the module loads, since a lambda's arity is a
runtime fact in Racket; the error carries the lambda's source location.

## Scenarios and limits

A scenario is a path: a start phase and a list of classed actions in order. A classed action is
spelled as the action name followed by its inputs as values, in brackets; an action with no inputs
is bare.

`standalone-activity.rkt:493-497`
```racket
(scenario retriedThenCompleted
  #:model activityProtocol
  #:starts unstarted
  #:actions ([start 'unset 'unset 'unset] attemptStart [attemptResult (failed #t)] backoff
             attemptStart [attemptResult 'completed]))
```

`standalone-activity.rkt:511-515`
```racket
(scenario pausedThenCompleted
  #:model activityProtocol
  #:starts unstarted
  #:actions ([start 'unset 'unset 'unset] [control 'pause] [control 'unpause] attemptStart
             [attemptResult 'completed]))
```

The macro turns each action into a class key such as `"attemptResult-failed-true"` and checks that
every action name is one the machine has a step for. The timer `backoff` is listed where it fires.

Limits bound the search: how many steps a trace may have, how many actions, and how many
candidates to examine. `(limits six #:steps 6 #:actions 6 #:search 262144)` at
`standalone-activity.rkt:529` is the one the retry path needs; `three` and `four` are declared
beside it.

## Queries

A query pairs a property with a scenario and limits. `#:find` asks for a trace of the scenario
on which the property's `#:when` step satisfies `#:holds`; it is what a set later realizes as a
test. `#:verify` asks that the property hold on every trace of the scenario; it is searched, never
realized.

`standalone-activity.rkt:538`, `545`
```racket
(query retry #:find retryCompletes #:in retriedThenCompleted #:limits six)
(query pauseHolds #:verify pausedIsNotDispatched #:in pausedThenCompleted #:limits six)
```

The `query` macro checks at compile time that the property's machine is the scenario's machine or
one it refines, which is what lets `pauseHolds` read a product property on a protocol scenario.

The search reads only the table. `run-query` is real in shape and elided in body:

`umpire.rkt:713-718`
```racket
  (match mode
    ['find (query-result (if (ormap (λ (t) (claim-holds? prop t)) witness) 'found 'not-found) witness)]
    ['verify (query-result (if (andmap (λ (t) (claim-holds? prop t)) witness) 'verified 'refuted) witness)]))

(define (search-traces m scen lim) '())        ; elided
(define (claim-holds? prop trace) #t)          ; elided
```

`search-traces` would walk the table from the scenario's start state, following the scenario's
class keys in order, and return each trace as a list of steps. `claim-holds?` would apply
`#:holds` to the step at the `#:when` position, or to each adjacent pair for a transition claim.
As written, every find query reports `'not-found` and every verify query reports `'verified`
vacuously.

Queries run under `raco test`, in the Model's own test submodule
(`standalone-activity.rkt:617-623`), which checks that each find query reports `'found` and each
verify query `'verified`. A failed query looks like any rackunit failure: the check's location, the message naming the
query, `actual: 'not-found`, `expected: 'found`. Nothing in the source is underlined, because by
this point the mistake is in the table, not in the syntax.

## Sets

A set groups queries for one purpose and says how each party is bound. `driven` means the test
harness performs that party's actions; `observed` means the harness reads which action occurred
and checks the machine allows it.

`standalone-activity.rkt:552-562`
```racket
(set standaloneActivityTests
  #:purpose functional
  #:bind ([caller driven] [worker driven])
  #:queries (completion nonRetryableFailure retry cancel terminate pauseResume
             scheduleToStartTimeout startToCloseTimeout))

;; The canary observes the worker: a deployment performs the worker's part itself.
(set standaloneActivityCanary
  #:purpose canary
  #:bind ([caller driven] [worker observed])
  #:queries (completion cancel))
```

The functional set has no `#:repeat implementation` line. The Nexus Model repeats each query under
the HSM and CHASM implementations; standalone activities exist only in CHASM, so there is nothing to
switch. An exploratory set (`standaloneActivityExploration`, `standalone-activity.rkt:564-569`)
names a machine and a coverage target instead of queries: `#:machine activityProtocol`,
`#:cover (rows results classMembers)`, `#:budget four`. The `set` macro refuses an exploratory set that lists queries and a functional set that lists
none.

## Composition with the worker

The protocol machine's `workerStop` is a stutter: the activity cannot see its worker. To state
anything about the worker, the Model composes with the worker entity from `worker.rkt`. Its
machine `polling` (`worker.rkt:71-78`) has two phases, `polling` and `stopped`, and three steps:
`workerStop` moves polling to stopped, `workerResume` moves back, and `serve` keeps polling and
has no row from stopped.

The activity's view of its worker is that machine restricted to stopping and serving. It never
resumes, because an action no `#:sync` line names would stay executable on its own and admit a
stop, a resume and then a dispatch:

`standalone-activity.rkt:576-578`
```racket
(machine activityWorker
  #:from Worker.polling
  #:restrict (workerStop Worker.serve))
```

The composition is a product of the two machines. A `#:sync` line makes two member actions fire as
one: the worker's stop is the activity's stop, and every dispatch is the worker serving.

`standalone-activity.rkt:580-589`
```racket
(struct StandaloneActivityState (activity worker) #:transparent)

(compose standaloneActivity
  #:for (activity Worker.worker)
  #:state StandaloneActivityState
  #:members ([activity activityProtocol] [worker activityWorker])
  #:sync ([workerStop activity.workerStop worker.workerStop]
          [attemptStart activity.attemptStart worker.serve])
  #:starts (activity.unstarted worker.polling)
  #:ends (activity.completed activity.failed activity.canceled activity.terminated activity.timedOut))
```

`activity.workerStop` is one identifier; the `compose` macro splits it at the dot and checks that
`activity` is a member. The cross-entity claim:

`standalone-activity.rkt:592-595`
```racket
(property startedByPollingWorker
  #:machine standaloneActivity
  #:when attemptStart
  #:holds (λ (s) (eq? (Worker.WorkerState-phase (StandaloneActivityState-worker (step-state s))) 'polling)))
```

It is verified by the query `stoppedWorkerStartsNothing` over the scenario `stoppedBeforeRetry`
(`standalone-activity.rkt:601-610`): start with the schedule-to-start deadline set, `attemptStart`
while the worker polls, a retryable failure, `activity.backoff`, `workerStop`, then
`activity.scheduleToStart`, under limits `six`. The path performs `attemptStart` once, so the claim
fires and is exercised rather than verified vacuously, which is what the spec's third revision note
fixed.

What the composed claim proves: on the protocol machine alone, `workerStop` changes nothing, so
the scenario's ordering of the stop before the deadline is a convention. In the composition,
`attemptStart` has a row only while the worker's phase is `polling`, because `serve` has no row
from `stopped`. `startedByPollingWorker` verified over the path says exactly that: no dispatch
happens after the stop. In this sample `build-composition` is a stub that returns an empty machine,
so the composed table is not actually built.

## Pins

`pins.rkt` fixes what the tables say, so a step arm that changes its answer fails a test.

`pins.rkt:156-158`
```racket
(test-case "activityProtocol: 288 states, 120 ends"
  (check-equal? (length (table-states (machine-table act:activityProtocol))) (* 12 3 8))
  (check-equal? (length (machine-ends act:activityProtocol)) (* 5 3 8)))
```

This guards the state enumeration: a field added to `ProtocolState` or a phase added to `Phase`
changes the count, and the author must update the pin knowingly.

`pins.rkt:161-162`
```racket
(test-case "activityProtocol: canceled from started is []"
  (check-equal? (act:protocolAttemptResultStep (act-at 'started #:attempts 1) 'canceled) '()))
```

This guards one arm: a cancel result without a cancel request has no row.

`pins.rkt:175-177`
```racket
(test-case "activityProduct: a retryable failure reads as scheduled again"
  (check-equal? (act:attemptResultStep (act:ProductState 'started) (act:failed #t))
                (list (step 'accepted (act:ProductState 'scheduled) '(statusScheduled)))))
```

This guards the revised product rule that makes the retry visible. A fourth pin
(`pins.rkt:179-180`) checks the refinement was not rejected; it is nearly redundant, since a
rejected refinement would have stopped the module from loading, but it names the claim where a
reader looks for it.

## From model to running test

Nothing in this directory runs against a Temporal server. After the Model, the pipeline would be:
Case lowering, which turns each find query's witness trace into a Case with the actions to perform
and the evidence to expect at each step; a realization, which binds each action class to a concrete
request (there is no realization for standalone activities yet, in any language); the Go Testpilot
runtime, which drives the `driven` parties, records the run, and reads the `observed` ones; and a
verdict per Case, from the referee comparing the recorded run to the table. None of that layer is
in the Racket sample, which stops at the checked Model, its tables, and the pins.

## Gaps and gradual growth

The Model says "not modeled" in several places, on purpose. Reset is deferred, like cancellation in
the Nexus Model. The heartbeat timeout is not a timer here. `workerStop` is a stutter on the
protocol machine and an empty step on the product machine, so the composition is where it means
anything. Every step function returns `'()` for the states where an action is not enabled, and
`protocolAttemptResultStep` returns `'()` for all phases but three.

This is what lets the Model grow one action at a time. Adding, say, a heartbeat timeout means: one
`Timeout` field on `ProtocolState`, one input on `start`, one timer in `#:timers`, one step
function of the same shape as `startToCloseStep`, one `productOf` case if a new phase appears, and
the refinement check tells you at load whether the product needs a row for it. The pins tell you
what changed. Nothing else in the file needs to move.

## Mental model recap

- An action is a party's side effect with finite inputs; a class is one assignment of them. Faults
  and timers are actions too.
- A step function is `state, inputs -> list of (outcome, next state, facts)`. Empty means not
  enabled.
- State is a record of finite fields; the framework enumerates every state and calls every step
  function on each to build the table, at load time.
- Two machines: the product says what Describe shows, the protocol says how the server gets there.
  `productOf` maps protocol states to product states.
- Refinement: every protocol row is a stutter under the map or matches some product row between
  the mapped states. Checked at load; a failure stops the module.
- Properties are same-step (`#:when` an action class) or transition (two steps). Scenarios are
  paths. Queries pair them under limits. Sets group queries with party bindings.
- Compile time catches shape and name mistakes; load time catches contracts and refinement; `raco
  test` runs queries and pins.

## Where this implementation is weak

- **The search and the composition are stubs.** `search-traces` returns no traces and
  `claim-holds?` returns true, so every find query reports `'not-found` and every verify query
  passes vacuously; `build-composition` returns an empty machine. The query pins and the composed
  claim cannot pass as written. The macro layer and the table and refinement code are real; the
  layer below them is a sketch.
- **Nothing has been compiled.** The reviewer found step-function contracts that would have
  rejected every fixed-arity function at load, a mis-destructured `refines+map` argument, a
  `#%module-begin` provide conflict, an invented rackunit check, and two missing imports. All are
  fixed in the current files, but by reading, not by `raco make`; more of the kind may remain.
- **Dynamic typing below the macros.** A `Phase` symbol placed in a `ProductState` is caught only
  if a contract or the table builder happens to compare it. `cases` checks enums; `cond` over
  phase sets checks nothing.
- **Accessor chains.** Properties read as `(eq? (ProtocolState-phase (step-state s)) 'completed)`
  where Lean reads `step.state.phase == .completed`. The `phase-is` and `records` helpers hide
  some of it; the scenarios' quoted symbols do not go away.
