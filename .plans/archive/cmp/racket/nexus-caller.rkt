#lang racket
;; # The Nexus caller-side Model
;;
;; One workflow-scheduled Nexus operation, as the caller sees it: the product machine says what an
;; operation does, the protocol machine says how the server gets there and refines it, and the
;; functional set runs one Query per side effect that settles the operation, once per value of the
;; implementation switch. Step functions and predicates rather than rows, no cancellation (fn-79)
;; and no concurrency-limit setup parameter.
;;
;; Read from top to bottom: vocabulary → the two machines → what they promise → what the set asks.
;;
;; Racket note: a `#lang racket` Model requires the framework and shadows racket's own `compose`
;; and `set` with the DSL forms of the same name. The same file written as `#lang umpire` drops the
;; two `require`s and gains the `test` submodule that runs every Query (see README).

(require "umpire.rkt"
         (prefix-in Worker. "worker.rkt"))   ; Worker.polling, Worker.worker, Worker.serve

(provide (all-defined-out))

;; ### Entities
;;
;; An operation is scheduled by a caller workflow, and recorded data names one by its scheduled
;; event: every history event of the operation carries that event's id.

(entity workflow)

(entity operation
  #:refer ([caller workflow])
  #:key scheduledEvent)

;; ### The input domains
;;
;; A class is one member of a domain, and a constructor that carries finite fields contributes one
;; class per assignment of them: `handlerError [retryable : Bool]` is one constructor and two
;; classes, which is the granularity an example is written at and what mirrors a protobuf oneof.

(enum Timeout unset expires)

(enum Reply
  syncSuccess
  async
  operationFailed
  operationCanceled
  (handlerError [retryable : Bool]))

(enum Resolution succeeded failed canceled)

(enum Delivery accepted notFound)

;; ### Actions
;;
;; Parties are names the feature declares by using them: `caller`, `handler`, `network`, `worker`.
;; The reserved party `system` is the server. A fault is an ordinary action of a declared party,
;; and a timer is `system` behavior the machine owns, so neither is a separate kind.

(action schedule
  #:party caller
  #:creates operation
  #:schema "temporal.api.command.v1.ScheduleNexusOperationCommandAttributes"
  #:input [scheduleToClose : Timeout]
          [scheduleToStart : Timeout]
          [startToClose : Timeout])

(action handlerReply
  #:party handler
  #:on operation
  #:schema "temporal.api.nexus.v1.StartOperationResponse | temporal.api.nexus.v1.HandlerError"
  #:input [reply : Reply]
  #:examples ([(handlerError #f)] -> "BadRequest")
             ([(handlerError #t)] -> "Internal"))

;; The Nexus HTTP completion carries no protobuf message, so it declares no schema and its classes
;; are names the realization interprets.
(action complete
  #:party handler
  #:on operation
  #:input [resolution : Resolution]
  #:results Delivery)

(action transportFault
  #:party network
  #:on operation)

;; The handler's worker stops polling. An action that names no entity is behavior no entity
;; records: the Run records the fault, but nothing recorded names the operation, so the machines
;; keep their state and record nothing at it. It is the worker module's action, aliased here with a
;; rename transformer so `machine` sees the same declaration the composition synchronizes on.
(define-syntax workerStop (make-rename-transformer #'Worker.workerStop))

;; ### The derived observation
;;
;; A retryable attempt failure writes no history event, so the attempt count is read back through
;; `DescribeWorkflowExecution`. Every other evidence name resolves against the realization's
;; catalog, which is why only a derived observation is declared.

(observation pendingAttempts
  #:on operation
  #:read attempts)

;; ### The product machine
;;
;; What an operation does, with no account of how. Every Property written against it is carried to
;; the protocol machine by the refinement declared there.

(enum ProductPhase scheduled started succeeded failed canceled timedOut)

(record ProductState [phase : ProductPhase])

(enum ProductOutcome accepted notFound)

(enum ProductFact
  nexusOperationScheduled
  nexusOperationStarted
  nexusOperationCompleted
  nexusOperationFailed
  nexusOperationCanceled
  nexusOperationTimedOut)

;; Every step function carries its own `define/contract` signature, checked at each call the table
;; builder makes, with blame on this module.
(define (productStep phase recorded)
  (list (step 'accepted (ProductState phase) (list recorded))))

;; The handler's reply to the server's start request. An operation that has not started yet is the
;; only one a reply can move.
(define/contract (handlerReplyStep state reply)
  (-> ProductState? Reply? (listof step?))
  (if (not (eq? (ProductState-phase state) 'scheduled))
      '()
      (cases Reply reply
        [syncSuccess (productStep 'succeeded 'nexusOperationCompleted)]
        [async (productStep 'started 'nexusOperationStarted)]
        [operationFailed (productStep 'failed 'nexusOperationFailed)]
        [operationCanceled (productStep 'canceled 'nexusOperationCanceled)]
        ;; A retryable handler error leaves the operation where it is: the product machine does not
        ;; know about backing off, which is the whole of what the protocol machine adds.
        [(handlerError #t) '()]
        [(handlerError #f) (productStep 'failed 'nexusOperationFailed)])))

;; The four phases the product machine ends on.
(define (productTerminal state)
  (and (memq (ProductState-phase state) '(succeeded failed canceled timedOut)) #t))

;; An asynchronous completion. A completion that arrives after the operation is over is not found,
;; and changes nothing.
(define/contract (completeStep state resolution)
  (-> ProductState? Resolution? (listof step?))
  (if (productTerminal state)
      (list (step 'notFound state '()))
      (cases Resolution resolution
        [succeeded (productStep 'succeeded 'nexusOperationCompleted)]
        [failed (productStep 'failed 'nexusOperationFailed)]
        [canceled (productStep 'canceled 'nexusOperationCanceled)])))

;; A transport fault is an ordinary action of the network. The product machine cannot see one:
;; whether a delivery was retried is the protocol's account of how, not what.
(define/contract (transportFaultStep _state)
  (-> ProductState? (listof step?)) '())

;; The handler's worker stopping is a fault the Run records and the operation does not feel. The
;; product machine cannot see it, like the transport fault: a step that kept the state and recorded
;; nothing would be indistinguishable from a stutter, and the refinement would read every stutter
;; as this step.
(define/contract (workerStopStep _state)
  (-> ProductState? (listof step?)) '())

;; One of the operation's deadlines firing. Which deadline is the protocol's account of how, so the
;; product machine has one timer, and it fires while the operation runs.
(define/contract (timeoutStep state)
  (-> ProductState? (listof step?))
  (if (memq (ProductState-phase state) '(scheduled started))
      (productStep 'timedOut 'nexusOperationTimedOut)
      '()))

(machine nexusProduct
  #:for operation
  #:state ProductState
  #:starts (scheduled)
  #:ends (succeeded failed canceled timedOut)
  #:timers (timeout)
  #:evidence ([nexusOperationStarted nexusOperationStarted]
              [nexusOperationCompleted nexusOperationCompleted]
              [nexusOperationFailed nexusOperationFailed]
              [nexusOperationCanceled nexusOperationCanceled]
              [nexusOperationTimedOut nexusOperationTimedOut])
  #:steps ([handlerReply handlerReplyStep]
           [complete completeStep]
           [transportFault transportFaultStep]
           [workerStop workerStopStep]
           [timeout timeoutStep]))

;; ### The protocol machine
;;
;; How the server gets there: the retry the product machine cannot see, the three timers the
;; schedule command sets, and the attempt count a retryable failure raises. Written against the
;; same actions, so a Property proved on the product machine is carried here by the refinement.
;;
;; The machine begins before the operation exists: a state record has no "no instance yet" member,
;; so `unscheduled` is that member, and it is what makes the three deadline fields reachable at
;; anything but their first value -- the schedule command is what sets them.
;;
;; Not here, for reasons recorded rather than silent: the `cancel` field and its rows (fn-79), and
;; the concurrency-limit rejection. The limit exists, but a step function does not read the setup,
;; the key and value differ per switch value, and the rejection names no operation, so it is not
;; modeled until a Query needs it.

(enum Phase unscheduled scheduled backingOff started succeeded failed canceled timedOut)

;; Which timer fired. The history event records it, so a Contract that did not check it would pass
;; a run that timed out on the wrong deadline.
(enum TimeoutType scheduleToClose scheduleToStart startToClose)

;; The attempt count is bounded by the Limits in the design; nothing wires the Limits into a
;; machine's state, so the bound is written here and the saturating successor keeps a retry inside
;; it.
(define attemptBound 2)

(record ProtocolState
  [phase : Phase]
  [attempts : (Fin 3)]          ; 0 .. attemptBound; `(Fin (add1 attemptBound))` once `record` evaluates types
  [scheduleToClose : Timeout]
  [scheduleToStart : Timeout]
  [startToClose : Timeout])

(enum ProtocolOutcome accepted notFound)

(enum ProtocolFact
  nexusOperationScheduled
  nexusOperationStarted
  nexusOperationCompleted
  nexusOperationFailed
  nexusOperationCanceled
  (nexusOperationTimedOut [timeoutType : TimeoutType])
  pendingAttempts)

;; The four phases the design ends on. A completion that arrives after one of them is not found.
(define (terminalPhase phase)
  (and (memq phase '(succeeded failed canceled timedOut)) #t))

;; Scheduled and not yet over: the phases a completion resolves and a timer can fire in.
(define (running phase)
  (and (memq phase '(scheduled backingOff started)) #t))

(define (moves state phase recorded)
  (list (step 'accepted (struct-copy ProtocolState state [phase phase]) recorded)))

;; The caller's schedule command. It names the operation's three deadlines, and every one of them
;; is a state field because whether a timer fires is a question about the operation and not about
;; the command that started it.
(define/contract (scheduleStep state scheduleToClose scheduleToStart startToClose)
  (-> ProtocolState? Timeout? Timeout? Timeout? (listof step?))
  (if (not (eq? (ProtocolState-phase state) 'unscheduled))
      '()
      (list (step 'accepted
                  (ProtocolState 'scheduled 0 scheduleToClose scheduleToStart startToClose)
                  '(nexusOperationScheduled)))))

;; The handler's reply to the server's start request. What the product machine cannot see is the
;; last arm: a retryable failure backs the operation off and raises its attempt count, and the count
;; is read back through the `pendingAttempts` observation because no history event records it.
(define/contract (protocolHandlerReplyStep state reply)
  (-> ProtocolState? Reply? (listof step?))
  (if (not (eq? (ProtocolState-phase state) 'scheduled))
      '()
      (cases Reply reply
        [syncSuccess (moves state 'succeeded '(nexusOperationCompleted))]
        [async (moves state 'started '(nexusOperationStarted))]
        [operationFailed (moves state 'failed '(nexusOperationFailed))]
        [operationCanceled (moves state 'canceled '(nexusOperationCanceled))]
        [(handlerError #f) (moves state 'failed '(nexusOperationFailed))]
        [(handlerError #t)
         (list (step 'accepted
                     (struct-copy ProtocolState state
                                  [phase 'backingOff]
                                  [attempts (saturating-succ (ProtocolState-attempts state) attemptBound)])
                     '(pendingAttempts)))])))

;; A transport fault is the same failure arriving as a dropped delivery rather than as a reply.
(define/contract (protocolTransportFaultStep state)
  (-> ProtocolState? (listof step?))
  (if (not (eq? (ProtocolState-phase state) 'scheduled))
      '()
      (list (step 'accepted
                  (struct-copy ProtocolState state
                               [phase 'backingOff]
                               [attempts (saturating-succ (ProtocolState-attempts state) attemptBound)])
                  '(pendingAttempts)))))

;; The handler's worker stopping is a fault the Run records and the operation does not feel, so
;; the step keeps the state and records nothing. On a path it is confirmed by the evidence of the
;; step after it, and the Case says so in a Known Gap.
(define/contract (protocolWorkerStopStep state)
  (-> ProtocolState? (listof step?))
  (list (step 'accepted state '())))

;; An asynchronous completion. Before a start, the server records a Started event first, which is
;; why the evidence is two facts and not one -- and why the product machine, which has no
;; `backingOff` phase to have skipped, could write the completion alone.
(define/contract (protocolCompleteStep state resolution)
  (-> ProtocolState? Resolution? (listof step?))
  (define phase (ProtocolState-phase state))
  (cond
    [(terminalPhase phase) (list (step 'notFound state '()))]
    [(eq? phase 'unscheduled) '()]
    [else
     (define startedFirst (if (eq? phase 'started) '() '(nexusOperationStarted)))
     (cases Resolution resolution
       [succeeded (moves state 'succeeded (append startedFirst '(nexusOperationCompleted)))]
       [failed (moves state 'failed (append startedFirst '(nexusOperationFailed)))]
       [canceled (moves state 'canceled (append startedFirst '(nexusOperationCanceled)))])]))

;; The backoff timer. It is what makes `backingOff` a phase the operation leaves rather than a
;; state it is stuck in, and it records nothing: a retry writes no history event.
(define/contract (backoffStep state)
  (-> ProtocolState? (listof step?))
  (if (eq? (ProtocolState-phase state) 'backingOff) (moves state 'scheduled '()) '()))

;; The schedule-to-close deadline covers the whole operation, so it fires in every running phase --
;; and only when the schedule command set it.
(define/contract (scheduleToCloseStep state)
  (-> ProtocolState? (listof step?))
  (if (and (running (ProtocolState-phase state)) (eq? (ProtocolState-scheduleToClose state) 'expires))
      (moves state 'timedOut (list (nexusOperationTimedOut 'scheduleToClose)))
      '()))

;; The schedule-to-start deadline covers the wait for the handler to accept, so it stops at the
;; start.
(define/contract (scheduleToStartStep state)
  (-> ProtocolState? (listof step?))
  (if (and (memq (ProtocolState-phase state) '(scheduled backingOff))
           (eq? (ProtocolState-scheduleToStart state) 'expires))
      (moves state 'timedOut (list (nexusOperationTimedOut 'scheduleToStart)))
      '()))

;; The start-to-close deadline covers the handler's own work, so it begins at the start.
(define/contract (startToCloseStep state)
  (-> ProtocolState? (listof step?))
  (if (and (eq? (ProtocolState-phase state) 'started) (eq? (ProtocolState-startToClose state) 'expires))
      (moves state 'timedOut (list (nexusOperationTimedOut 'startToClose)))
      '()))

;; How a protocol state reads as a product state. A phase of the same name is that phase; backing
;; off is still scheduled, because the product machine cannot see a retry; and an operation not yet
;; scheduled reads as scheduled, because the product machine begins there. Every other field is
;; hidden, which is what a map that does not read it says.
(define/contract (productOf state) (-> ProtocolState? ProductState?)
  (ProductState
   (cases Phase (ProtocolState-phase state)
     [unscheduled 'scheduled] [scheduled 'scheduled] [backingOff 'scheduled]
     [started 'started]
     [succeeded 'succeeded]
     [failed 'failed]
     [canceled 'canceled]
     [timedOut 'timedOut])))

;; `#:refines` walks every row through `#:map` when this module is instantiated and raises
;; `exn:fail:umpire` at the first row that is neither a product step nor a stutter.
(machine nexusProtocol
  #:for operation
  #:state ProtocolState
  #:refines nexusProduct
  #:map productOf
  #:starts (unscheduled)
  #:ends (succeeded failed canceled timedOut)
  #:timers (backoff scheduleToClose scheduleToStart startToClose)
  #:unobservable (backoff)
  #:evidence ([nexusOperationScheduled nexusOperationScheduled]
              [nexusOperationStarted nexusOperationStarted]
              [nexusOperationCompleted nexusOperationCompleted]
              [nexusOperationFailed nexusOperationFailed]
              [nexusOperationCanceled nexusOperationCanceled]
              [nexusOperationTimedOut nexusOperationTimedOut]
              [pendingAttempts pendingAttempts])
  #:steps ([schedule scheduleStep]
           [handlerReply protocolHandlerReplyStep]
           [complete protocolCompleteStep]
           [transportFault protocolTransportFaultStep]
           [workerStop protocolWorkerStopStep]
           [backoff backoffStep]
           [scheduleToClose scheduleToCloseStep]
           [scheduleToStart scheduleToStartStep]
           [startToClose startToCloseStep]))

;; ### What the machines promise
;;
;; A same-step claim names the action it is about under `#:when` and holds of the step that action
;; produces; a transition claim holds of the step before and the step after. A functional Query
;; realizes a same-step claim, because the Case's Contract is the claim's clause triggered by the
;; action the Case performs; a transition claim is searched and verified, never realized.

;; Once an operation is over, no step changes its phase. Declared on the product machine and read
;; on the protocol machine through the map.
(property terminalIsFinal
  #:machine nexusProduct
  #:holds (λ (before after)
            (or (not (productTerminal (step-state before)))
                (eq? (ProductState-phase (step-state after)) (ProductState-phase (step-state before))))))

;; A synchronous reply settles the operation as succeeded, and the completed event records it.
(property syncSucceeds
  #:machine nexusProtocol
  #:when [handlerReply 'syncSuccess]
  #:holds (λ (s) (and (eq? (ProtocolState-phase (step-state s)) 'succeeded)
                      (member 'nexusOperationCompleted (step-facts s)))))

;; An asynchronous reply starts the operation, and the started event records it.
(property asyncStarts
  #:machine nexusProtocol
  #:when [handlerReply 'async]
  #:holds (λ (s) (and (eq? (ProtocolState-phase (step-state s)) 'started)
                      (member 'nexusOperationStarted (step-facts s)))))

;; A successful completion is recorded by the completed event. Neither the phase nor the outcome
;; is fixed: a completion resolves any running phase, and `accepted` is every earlier step's
;; outcome too, so a clause fixing it would be answered before the completion.
(property completionSucceeds
  #:machine nexusProtocol
  #:when [complete 'succeeded]
  #:holds (λ (s) (member 'nexusOperationCompleted (step-facts s))))

;; A failed completion is recorded by the failed event.
(property completionFails
  #:machine nexusProtocol
  #:when [complete 'failed]
  #:holds (λ (s) (member 'nexusOperationFailed (step-facts s))))

;; A non-retryable handler error settles the operation as failed, and the failed event records it.
(property handlerErrorFails
  #:machine nexusProtocol
  #:when [handlerReply (handlerError #f)]
  #:holds (λ (s) (and (eq? (ProtocolState-phase (step-state s)) 'failed)
                      (member 'nexusOperationFailed (step-facts s)))))

;; Succeeded on the second attempt of an operation with no deadline set. A claim fixes one state,
;; so every field is named; the record is transparent, so `equal?` compares them all.
(define succeededOnRetry (ProtocolState 'succeeded 1 'unset 'unset 'unset))

;; A synchronous reply to the retried attempt settles the operation as succeeded on its second
;; attempt: the count the retryable failure raised is still one, and the completed event records
;; the reply.
(property retrySucceeds
  #:machine nexusProtocol
  #:when [handlerReply 'syncSuccess]
  #:holds (λ (s) (and (equal? (step-state s) succeededOnRetry)
                      (member 'nexusOperationCompleted (step-facts s)))))

;; The schedule-to-start deadline settles an operation no handler started as timed out, and the
;; timed-out event records which deadline it was.
(property scheduleToStartFires
  #:machine nexusProtocol
  #:when scheduleToStart
  #:holds (λ (s) (and (eq? (ProtocolState-phase (step-state s)) 'timedOut)
                      (member (nexusOperationTimedOut 'scheduleToStart) (step-facts s)))))

;; The start-to-close deadline settles a started operation no handler completed as timed out.
(property startToCloseFires
  #:machine nexusProtocol
  #:when startToClose
  #:holds (λ (s) (and (eq? (ProtocolState-phase (step-state s)) 'timedOut)
                      (member (nexusOperationTimedOut 'startToClose) (step-facts s)))))

;; ### The paths the Queries run
;;
;; A protocol Scenario names its classed actions with their inputs and its start by its phase. Each
;; path below is one upstream functional test's shape: the schedule command with no deadline set,
;; then the side effects that settle the operation.

(scenario syncReplied
  #:model nexusProtocol
  #:starts unscheduled
  #:actions ([schedule 'unset 'unset 'unset] [handlerReply 'syncSuccess]))

(scenario asyncThenSucceeded
  #:model nexusProtocol
  #:starts unscheduled
  #:actions ([schedule 'unset 'unset 'unset] [handlerReply 'async] [complete 'succeeded]))

(scenario asyncThenFailed
  #:model nexusProtocol
  #:starts unscheduled
  #:actions ([schedule 'unset 'unset 'unset] [handlerReply 'async] [complete 'failed]))

(scenario nonRetryableError
  #:model nexusProtocol
  #:starts unscheduled
  #:actions ([schedule 'unset 'unset 'unset] [handlerReply (handlerError #f)]))

;; The retryable error backs the operation off; the backoff timer fires and records nothing; the
;; retried attempt is answered synchronously.
(scenario retriedThenSucceeded
  #:model nexusProtocol
  #:starts unscheduled
  #:actions ([schedule 'unset 'unset 'unset] [handlerReply (handlerError #t)] backoff
             [handlerReply 'syncSuccess]))

;; The schedule command sets the schedule-to-start deadline; the handler's worker stops, so nothing
;; answers the start request; the deadline fires. The worker stops after the schedule in the
;; operation's order, where the stop changes nothing; the realization stops it before the workflow
;; starts, where the stop cannot race the dispatch.
(scenario scheduleToStartExpires
  #:model nexusProtocol
  #:starts unscheduled
  #:actions ([schedule 'unset 'expires 'unset] workerStop scheduleToStart))

;; The schedule command sets the start-to-close deadline; the handler accepts asynchronously and
;; never completes; the deadline fires.
(scenario startToCloseExpires
  #:model nexusProtocol
  #:starts unscheduled
  #:actions ([schedule 'unset 'unset 'expires] [handlerReply 'async] startToClose))

;; Nine actions are enabled before the operation is scheduled and eleven once it is, so an exact
;; sequence of two is found among ninety-nine candidates, one of three among about a thousand and
;; one of four among about ten thousand.
(limits two #:steps 2 #:actions 2 #:search 512)
(limits three #:steps 3 #:actions 3 #:search 4096)
(limits four #:steps 4 #:actions 4 #:search 32768)

;; ### The Queries
;;
;; The design's seven: sync success, async reply then succeeded callback, async reply then failed
;; callback, non-retryable handler error, retryable handler error then sync success after one
;; backoff, schedule-to-start timeout with the handler's worker stopped, start-to-close timeout
;; after an asynchronous reply. Each finds its same-step claim on its path and is realized by the
;; set below. The product claim is verified over every trace of one path, outside the set, because
;; a `#:verify` Query realizes nothing.

(query syncCompletion #:find syncSucceeds #:in syncReplied #:limits two)
(query asyncCompletion #:find completionSucceeds #:in asyncThenSucceeded #:limits three)
(query asyncFailure #:find completionFails #:in asyncThenFailed #:limits three)
;; `handlerError` is the Reply constructor in this module, so the Query binds a longer name and
;; keeps the design's name as its own.
(query handlerErrorQuery #:named handlerError #:find handlerErrorFails #:in nonRetryableError #:limits two)
(query retry #:find retrySucceeds #:in retriedThenSucceeded #:limits four)
(query scheduleToStartTimeout #:find scheduleToStartFires #:in scheduleToStartExpires #:limits three)
(query startToCloseTimeout #:find startToCloseFires #:in startToCloseExpires #:limits three)
(query terminalHolds #:verify terminalIsFinal #:in asyncThenSucceeded #:limits three)

;; ### The functional set
;;
;; Every party but `system` is bound: the Case drives the caller, the handler and the worker, and
;; observes the network. The set repeats over the implementation switch, so each Query's Case runs
;; once under HSM and once under CHASM.

(set nexusCallerTests
  #:purpose functional
  #:bind ([caller driven] [handler driven] [network observed] [worker driven])
  #:repeat implementation
  #:queries (syncCompletion asyncCompletion asyncFailure handlerErrorQuery retry
             scheduleToStartTimeout startToCloseTimeout))

;; ### The canary set
;;
;; A canary runs a Query against a deployment that performs the handler's part itself: the handler
;; is `observed`, so the verifier reads which reply occurred and checks the machine allows it. A
;; path with a silent step -- the backoff, the worker stop -- is a capability gap no deployment
;; closes, so a canary naming it is rejected.

(set nexusCallerCanary
  #:purpose canary
  #:bind ([caller driven] [handler observed] [network observed] [worker driven])
  #:queries (syncCompletion asyncCompletion))

;; ### The exploratory set
;;
;; An exploration covers the protocol machine rather than listing Queries: the rows an exploration
;; within the budget's steps of a start can take, the results those rows reach and the members of
;; the classes their actions claim, each in the machine's catalog order and cut at the budget's
;; search count, so the enumeration is the same on every reading.

(set nexusCallerExploration
  #:purpose exploratory
  #:bind ([caller driven] [handler driven] [network observed] [worker driven])
  #:machine nexusProtocol
  #:cover (rows results classMembers)
  #:budget four)

;; ### The operation and the handler's worker
;;
;; The protocol machine's worker stop is a stutter row: the operation cannot see its handler's
;; worker, so the schedule-to-start Scenario orders the stop before the request by convention.
;; Composed with the worker of the handler's task queue, the stop is the worker's own phase change
;; and every reply is the worker serving, so a reply has a row only while the worker polls. No set
;; names the composition; it is what the cross-entity claim is verified over.

;; The caller's view of the handler's worker: it stops and it serves. It never resumes, because an
;; action no `#:sync` line names would stay executable on its own and admit a stop, a resume and
;; then a reply; the operation's timers settle every state a stop leaves.
(machine handlerWorker
  #:from Worker.polling
  #:restrict (workerStop Worker.serve))

(struct NexusCallerState (operation worker) #:transparent)

(compose nexusCaller
  #:for (operation Worker.worker)
  #:state NexusCallerState
  #:members ([operation nexusProtocol] [worker handlerWorker])
  #:sync ([workerStop operation.workerStop worker.workerStop]
          [handlerReply operation.handlerReply worker.serve])
  #:starts (operation.unscheduled worker.polling)
  #:ends (operation.succeeded operation.failed operation.canceled operation.timedOut))

;; Every reply, of any class, leaves the handler's worker polling: no handler replies while its
;; worker is stopped.
(property repliedByPollingWorker
  #:machine nexusCaller
  #:when handlerReply
  #:holds (λ (s) (eq? (Worker.WorkerState-phase (NexusCallerState-worker (step-state s))) 'polling)))

;; A retryable reply backs the operation off; the handler's worker then stops, so the retried
;; attempt is never answered and the schedule-to-start deadline fires.
(scenario repliedThenStopped
  #:model nexusCaller
  #:starts operation.unscheduled
  #:actions ([operation.schedule 'unset 'expires 'unset] [handlerReply (handlerError #t)]
             workerStop operation.scheduleToStart))

(query stoppedWorkerRepliesNothing
  #:verify repliedByPollingWorker
  #:in repliedThenStopped
  #:limits four)

;; ### Where the checks run
;;
;; The refinement already ran: instantiating this module built both tables and walked every
;; protocol row through `productOf`. Search runs here, under `raco test nexus-caller.rkt`; the
;; pins in pins.rkt read the same tables.

(module+ test
  (require rackunit)
  (for ([q (list syncCompletion asyncCompletion asyncFailure handlerErrorQuery retry
                 scheduleToStartTimeout startToCloseTimeout)])
    (check-eq? (query-result-outcome (run-query q)) 'found (format "~a" q)))
  (for ([q (list terminalHolds stoppedWorkerRepliesNothing)])
    (check-eq? (query-result-outcome (run-query q)) 'verified (format "~a" q))))
