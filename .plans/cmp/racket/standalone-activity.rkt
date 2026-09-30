#lang racket
;; # The standalone activity Model
;;
;; A Temporal activity started directly through `StartActivityExecution`, with no workflow. Grounded
;; in `chasm/lib/activity/statemachine.go` and `proto/v1/activity_state.proto`. Reset is deferred
;; (like cancellation in the Nexus Model) and not modeled; the heartbeat timeout is not modeled.
;; Standalone activities write no history events, so every fact is a status read through
;; `DescribeActivityExecution` or a result read through `PollActivityExecution`.
;;
;; Read from top to bottom: vocabulary → the two machines → what they promise → what the set asks.

(require "umpire.rkt"
         (prefix-in Worker. "worker.rkt"))

(provide (all-defined-out))

;; ### Entities
;;
;; An activity is named by the id the caller chose for it; no scheduled event exists to name it by.

(entity activity #:key activityId)

;; ### The input domains
;;
;; `failed [retryable : Bool]` is one constructor and two classes, the granularity the examples are
;; written at: one per `ApplicationFailure` flavor.

(enum Timeout unset expires)

(enum AttemptResult completed (failed [retryable : Bool]) canceled)

(enum Delivery accepted notFound)

(enum Control pause unpause requestCancel terminate)

;; ### Actions
;;
;; The caller starts and controls the activity; the worker's poll receives the task and reports the
;; attempt's result. A fault is an ordinary action of a declared party, and a timer is `system`
;; behavior the machine owns.

(action start
  #:party caller
  #:creates activity
  #:schema "temporal.api.workflowservice.v1.StartActivityExecutionRequest"
  #:input [scheduleToClose : Timeout]
          [scheduleToStart : Timeout]
          [startToClose : Timeout])

;; The worker's poll receives the task (`PollActivityTaskQueue`).
(action attemptStart
  #:party worker
  #:on activity
  #:schema "temporal.api.workflowservice.v1.PollActivityTaskQueueResponse")

(action attemptResult
  #:party worker
  #:on activity
  #:schema "RespondActivityTaskCompletedRequest | RespondActivityTaskFailedRequest | RespondActivityTaskCanceledRequest"
  #:input [result : AttemptResult]
  #:examples ([(failed #f)] -> "ApplicationFailure nonRetryable")
             ([(failed #t)] -> "ApplicationFailure retryable"))

(action control
  #:party caller
  #:on activity
  #:schema "PauseActivityExecutionRequest | UnpauseActivityExecutionRequest | RequestCancelActivityExecutionRequest | TerminateActivityExecutionRequest"
  #:input [control : Control]
  #:results Delivery)

;; The worker stops polling: behavior no entity records, so the machines keep their state and record
;; nothing at it. The worker module's action, aliased so `machine` sees the declaration the
;; composition synchronizes on.
(define-syntax workerStop (make-rename-transformer #'Worker.workerStop))

;; ### The derived observation
;;
;; A retryable attempt failure writes nothing, so the attempt count is read back through
;; `DescribeActivityExecution`.

(observation attemptCount
  #:on activity
  #:read attempt)

;; ### The product machine
;;
;; What the caller sees through Describe, with no account of how. Every Property written against it
;; is carried to the protocol machine by the refinement declared there. Unlike the Nexus product,
;; this one sees a retry: Describe reads SCHEDULED again after a retryable failure, so the retry is a
;; visible row and only the backoff wait is the protocol's own.

(enum ProductPhase
  scheduled started paused cancelRequested completed failed canceled terminated timedOut)

(record ProductState [phase : ProductPhase])

(enum ProductOutcome accepted notFound)

(enum ProductFact
  statusScheduled statusStarted statusPaused statusCancelRequested statusCompleted statusFailed
  statusCanceled statusTerminated statusTimedOut)

(define (productStep phase recorded)
  (list (step 'accepted (ProductState phase) (list recorded))))

;; The five phases the product machine ends on.
(define (productTerminal state)
  (and (memq (ProductState-phase state) '(completed failed canceled terminated timedOut)) #t))

;; The worker's poll receives the task. Only a scheduled activity is dispatched.
(define/contract (attemptStartStep state)
  (-> ProductState? (listof step?))
  (if (eq? (ProductState-phase state) 'scheduled)
      (productStep 'started 'statusStarted)
      '()))

;; The worker reports the attempt. A retryable failure is visible here, unlike in the Nexus product:
;; Describe reads SCHEDULED again with a higher attempt count (`TransitionRescheduled`), and a
;; retryable failure while a cancel is requested settles the activity as canceled. A cancel result
;; is honored only when a cancel was requested.
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

;; The caller's control requests. A request against an activity that is over is not found and
;; changes nothing; `requestCancel` is idempotent while a cancel is already requested.
(define/contract (controlStep state control)
  (-> ProductState? Control? (listof step?))
  (define phase (ProductState-phase state))
  (if (productTerminal state)
      (list (step 'notFound state '()))
      (cases Control control
        [pause (if (memq phase '(scheduled started)) (productStep 'paused 'statusPaused) '())]
        [unpause (if (eq? phase 'paused) (productStep 'scheduled 'statusScheduled) '())]
        [requestCancel (if (memq phase '(scheduled started paused cancelRequested))
                           (productStep 'cancelRequested 'statusCancelRequested)
                           '())]
        [terminate (productStep 'terminated 'statusTerminated)])))

;; The worker stopping is a fault the Run records and the activity does not feel. The product
;; machine cannot see it: a step that kept the state and recorded nothing would be indistinguishable
;; from a stutter, and the refinement would read every stutter as this step.
(define/contract (workerStopStep _state)
  (-> ProductState? (listof step?)) '())

;; One of the activity's deadlines firing. Which deadline is the protocol's account of how, so the
;; product machine has one timer, and it fires while the activity runs.
(define/contract (timeoutStep state)
  (-> ProductState? (listof step?))
  (if (memq (ProductState-phase state) '(scheduled started cancelRequested paused))
      (productStep 'timedOut 'statusTimedOut)
      '()))

(machine activityProduct
  #:for activity
  #:state ProductState
  #:starts (scheduled)
  #:ends (completed failed canceled terminated timedOut)
  #:timers (timeout)
  #:evidence ([statusScheduled statusScheduled]
              [statusStarted statusStarted]
              [statusPaused statusPaused]
              [statusCancelRequested statusCancelRequested]
              [statusCompleted statusCompleted]
              [statusFailed statusFailed]
              [statusCanceled statusCanceled]
              [statusTerminated statusTerminated]
              [statusTimedOut statusTimedOut])
  #:steps ([attemptStart attemptStartStep]
           [attemptResult attemptResultStep]
           [control controlStep]
           [workerStop workerStopStep]
           [timeout timeoutStep]))

;; ### The protocol machine
;;
;; How the server gets there: the retry the product machine cannot see, the pause the worker has
;; not yet acknowledged, the three timers the start request sets, and the attempt count a dispatch
;; raises. Written against the same actions, so a Property proved on the product machine is carried
;; here by the refinement.
;;
;; The machine begins before the activity exists: `unstarted` is the "no instance yet" member, and
;; it is what makes the three deadline fields reachable at anything but their first value -- the
;; start request is what sets them.

(enum Phase
  unstarted scheduled backingOff started paused pauseRequested cancelRequested
  completed failed canceled terminated timedOut)

;; Which timer fired. Describe reports it, so a Contract that did not check it would pass a run that
;; timed out on the wrong deadline.
(enum TimeoutType scheduleToClose scheduleToStart startToClose)

;; The attempt count is bounded by the Limits in the design; nothing wires the Limits into a
;; machine's state, so the bound is written here and the saturating successor keeps a retry inside
;; it.
(define attemptBound 2)

(record ProtocolState
  [phase : Phase]
  [attempts : (Fin 3)]          ; 0 .. attemptBound
  [scheduleToClose : Timeout]
  [scheduleToStart : Timeout]
  [startToClose : Timeout])

(enum ProtocolOutcome accepted notFound)

;; As the product's facts, with the typed timeout replacing the untyped one, plus the derived count.
(enum ProtocolFact
  statusScheduled statusStarted statusPaused statusCancelRequested statusCompleted statusFailed
  statusCanceled statusTerminated
  (statusTimedOut [timeoutType : TimeoutType])
  attemptCount)

;; The five phases the design ends on. A control request that arrives after one of them is not
;; found.
(define (terminalPhase phase)
  (and (memq phase '(completed failed canceled terminated timedOut)) #t))

;; Started and not yet over: the phases the schedule-to-close deadline covers.
(define (running phase)
  (and (memq phase '(scheduled backingOff started paused pauseRequested cancelRequested)) #t))

(define (moves state phase recorded)
  (list (step 'accepted (struct-copy ProtocolState state [phase phase]) recorded)))

(define (bump state)
  (saturating-succ (ProtocolState-attempts state) attemptBound))

;; The caller's start request. It names the activity's three deadlines, and every one of them is a
;; state field because whether a timer fires is a question about the activity and not about the
;; request that started it.
(define/contract (startStep state scheduleToClose scheduleToStart startToClose)
  (-> ProtocolState? Timeout? Timeout? Timeout? (listof step?))
  (if (not (eq? (ProtocolState-phase state) 'unstarted))
      '()
      (list (step 'accepted
                  (ProtocolState 'scheduled 0 scheduleToClose scheduleToStart startToClose)
                  '(statusScheduled)))))

;; The worker's poll receives the task: the attempt count rises, and Describe reads both the status
;; and the count.
(define/contract (protocolAttemptStartStep state)
  (-> ProtocolState? (listof step?))
  (if (not (eq? (ProtocolState-phase state) 'scheduled))
      '()
      (list (step 'accepted
                  (struct-copy ProtocolState state [phase 'started] [attempts (bump state)])
                  '(statusStarted attemptCount)))))

;; The worker reports the attempt. What the product machine cannot see is the retryable arm, and
;; where it leads depends on what the caller asked for meanwhile: from `started` it backs off; from
;; `cancelRequested` it settles as canceled, faithful to statemachine.go where CANCEL_REQUESTED is a
;; source of Canceled; from `pauseRequested` it lands in `paused`
;; (TransitionAttemptFailedWhilePauseRequested). A cancel result without a cancel request has no
;; row.
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

;; The caller's control requests. A pause of a running attempt is only requested until the worker
;; reports (Describe reads PAUSE_REQUESTED; the fact is `statusPaused` for both, as the product
;; cannot tell them apart). Terminate settles any activity that exists and is not over.
(define/contract (protocolControlStep state control)
  (-> ProtocolState? Control? (listof step?))
  (define phase (ProtocolState-phase state))
  (cond
    [(terminalPhase phase) (list (step 'notFound state '()))]
    [(eq? phase 'unstarted) '()]
    [else
     (cases Control control
       [pause (cond [(memq phase '(scheduled backingOff)) (moves state 'paused '(statusPaused))]
                    [(eq? phase 'started) (moves state 'pauseRequested '(statusPaused))]
                    [else '()])]
       [unpause (cond [(eq? phase 'paused) (moves state 'scheduled '(statusScheduled))]
                      [(eq? phase 'pauseRequested) (moves state 'started '(statusStarted))]
                      [else '()])]
       [requestCancel (moves state 'cancelRequested '(statusCancelRequested))]
       [terminate (moves state 'terminated '(statusTerminated))])]))

;; The worker stopping is a fault the Run records and the activity does not feel, so the step keeps
;; the state and records nothing. On a path it is confirmed by the evidence of the step after it.
(define/contract (protocolWorkerStopStep state)
  (-> ProtocolState? (listof step?))
  (list (step 'accepted state '())))

;; The backoff timer. It is what makes `backingOff` a phase the activity leaves rather than a state
;; it is stuck in, and it records nothing: a retry writes nothing.
(define/contract (backoffStep state)
  (-> ProtocolState? (listof step?))
  (if (eq? (ProtocolState-phase state) 'backingOff) (moves state 'scheduled '()) '()))

;; The schedule-to-close deadline covers the whole activity, so it fires in every running phase --
;; and only when the start request set it.
(define/contract (scheduleToCloseStep state)
  (-> ProtocolState? (listof step?))
  (if (and (running (ProtocolState-phase state)) (eq? (ProtocolState-scheduleToClose state) 'expires))
      (moves state 'timedOut (list (statusTimedOut 'scheduleToClose)))
      '()))

;; The schedule-to-start deadline covers the wait for a worker to pick the task up, so it stops at
;; the dispatch.
(define/contract (scheduleToStartStep state)
  (-> ProtocolState? (listof step?))
  (if (and (memq (ProtocolState-phase state) '(scheduled backingOff))
           (eq? (ProtocolState-scheduleToStart state) 'expires))
      (moves state 'timedOut (list (statusTimedOut 'scheduleToStart)))
      '()))

;; The start-to-close deadline covers the attempt, so it begins at the dispatch and keeps running
;; while a pause or a cancel waits on the worker.
(define/contract (startToCloseStep state)
  (-> ProtocolState? (listof step?))
  (if (and (memq (ProtocolState-phase state) '(started pauseRequested cancelRequested))
           (eq? (ProtocolState-startToClose state) 'expires))
      (moves state 'timedOut (list (statusTimedOut 'startToClose)))
      '()))

;; How a protocol state reads as a product state. Backing off is still scheduled, because the
;; product does not see the wait; an activity not yet started reads as scheduled, because the
;; product begins there; a requested pause reads as started, because the worker still holds the
;; attempt: every answer it can give is a product row from `started`, and the request itself is a
;; stutter. Every other field is hidden.
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

(machine activityProtocol
  #:for activity
  #:state ProtocolState
  #:refines activityProduct
  #:map productOf
  #:starts (unstarted)
  #:ends (completed failed canceled terminated timedOut)
  #:timers (backoff scheduleToClose scheduleToStart startToClose)
  #:unobservable (backoff)
  #:evidence ([statusScheduled statusScheduled]
              [statusStarted statusStarted]
              [statusPaused statusPaused]
              [statusCancelRequested statusCancelRequested]
              [statusCompleted statusCompleted]
              [statusFailed statusFailed]
              [statusCanceled statusCanceled]
              [statusTerminated statusTerminated]
              [statusTimedOut statusTimedOut]
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

;; ### What the machines promise
;;
;; A same-step claim names the action it is about under `#:when` and holds of the step that action
;; produces; a transition claim holds of the step before and the step after. A functional Query
;; realizes a same-step claim; a transition claim is searched and verified, never realized.

(define (phase-is s phase) (eq? (ProtocolState-phase (step-state s)) phase))
(define (records s fact) (and (member fact (step-facts s)) #t))

;; Once an activity is over, no step changes its phase. Declared on the product machine and read on
;; the protocol machine through the map.
(property terminalIsFinal
  #:machine activityProduct
  #:holds (λ (before after)
            (or (not (productTerminal (step-state before)))
                (eq? (ProductState-phase (step-state after)) (ProductState-phase (step-state before))))))

;; A completed attempt settles the activity, and Describe reads COMPLETED.
(property completes
  #:machine activityProtocol
  #:when [attemptResult 'completed]
  #:holds (λ (s) (and (phase-is s 'completed) (records s 'statusCompleted))))

;; A non-retryable failure settles the activity as failed.
(property nonRetryableFails
  #:machine activityProtocol
  #:when [attemptResult (failed #f)]
  #:holds (λ (s) (and (phase-is s 'failed) (records s 'statusFailed))))

;; Completed on the second attempt of an activity with no deadline set. A claim fixes one state, so
;; every field is named; the second dispatch raised the count to two.
(define completedOnRetry (ProtocolState 'completed 2 'unset 'unset 'unset))

(property retryCompletes
  #:machine activityProtocol
  #:when [attemptResult 'completed]
  #:holds (λ (s) (and (equal? (step-state s) completedOnRetry) (records s 'statusCompleted))))

;; A cancel request against a running attempt is recorded as requested, not as canceled: the
;; worker has yet to report.
(property cancelRequestedWhileStarted
  #:machine activityProtocol
  #:when [control 'requestCancel]
  #:holds (λ (s) (and (phase-is s 'cancelRequested) (records s 'statusCancelRequested))))

;; The worker honoring the cancel settles the activity as canceled.
(property canceledByWorker
  #:machine activityProtocol
  #:when [attemptResult 'canceled]
  #:holds (λ (s) (and (phase-is s 'canceled) (records s 'statusCanceled))))

;; A terminate settles the activity whatever it was doing.
(property terminated
  #:machine activityProtocol
  #:when [control 'terminate]
  #:holds (λ (s) (and (phase-is s 'terminated) (records s 'statusTerminated))))

;; A paused activity is never dispatched: no step leaves `paused` for `started`. A transition claim
;; on the product, read on the protocol machine through the map; it holds over the whole product
;; table.
(property pausedIsNotDispatched
  #:machine activityProduct
  #:holds (λ (before after)
            (or (not (eq? (ProductState-phase (step-state before)) 'paused))
                (not (eq? (ProductState-phase (step-state after)) 'started)))))

;; The schedule-to-start deadline settles an activity no worker picked up as timed out, and Describe
;; reads which deadline it was.
(property scheduleToStartFires
  #:machine activityProtocol
  #:when scheduleToStart
  #:holds (λ (s) (and (phase-is s 'timedOut) (records s (statusTimedOut 'scheduleToStart)))))

;; The start-to-close deadline settles a dispatched attempt no worker reported as timed out.
(property startToCloseFires
  #:machine activityProtocol
  #:when startToClose
  #:holds (λ (s) (and (phase-is s 'timedOut) (records s (statusTimedOut 'startToClose)))))

;; ### The paths the Queries run
;;
;; Each path is one upstream functional test's shape: the start request with no deadline set, then
;; the side effects that settle the activity.

(scenario completed
  #:model activityProtocol
  #:starts unstarted
  #:actions ([start 'unset 'unset 'unset] attemptStart [attemptResult 'completed]))

(scenario nonRetryable
  #:model activityProtocol
  #:starts unstarted
  #:actions ([start 'unset 'unset 'unset] attemptStart [attemptResult (failed #f)]))

;; The retryable failure backs the activity off; the backoff timer fires and records nothing; the
;; second dispatch completes.
(scenario retriedThenCompleted
  #:model activityProtocol
  #:starts unstarted
  #:actions ([start 'unset 'unset 'unset] attemptStart [attemptResult (failed #t)] backoff
             attemptStart [attemptResult 'completed]))

(scenario cancelRequestedThenCanceled
  #:model activityProtocol
  #:starts unstarted
  #:actions ([start 'unset 'unset 'unset] attemptStart [control 'requestCancel]
             [attemptResult 'canceled]))

;; The worker stops before anything is dispatched; the caller terminates the scheduled activity.
(scenario terminatedWhileScheduled
  #:model activityProtocol
  #:starts unstarted
  #:actions ([start 'unset 'unset 'unset] workerStop [control 'terminate]))

(scenario pausedThenCompleted
  #:model activityProtocol
  #:starts unstarted
  #:actions ([start 'unset 'unset 'unset] [control 'pause] [control 'unpause] attemptStart
             [attemptResult 'completed]))

(scenario scheduleToStartExpires
  #:model activityProtocol
  #:starts unstarted
  #:actions ([start 'unset 'expires 'unset] workerStop scheduleToStart))

(scenario startToCloseExpires
  #:model activityProtocol
  #:starts unstarted
  #:actions ([start 'unset 'unset 'expires] attemptStart startToClose))

(limits three #:steps 3 #:actions 3 #:search 4096)
(limits four #:steps 4 #:actions 4 #:search 32768)
(limits six #:steps 6 #:actions 6 #:search 262144)

;; ### The Queries
;;
;; Eight find-Queries, one per side effect that settles or redirects the activity, and two verify
;; Queries for the product claims read on protocol paths.

(query completion #:find completes #:in completed #:limits three)
(query nonRetryableFailure #:find nonRetryableFails #:in nonRetryable #:limits three)
(query retry #:find retryCompletes #:in retriedThenCompleted #:limits six)
(query cancel #:find canceledByWorker #:in cancelRequestedThenCanceled #:limits four)
(query terminate #:find terminated #:in terminatedWhileScheduled #:limits three)
(query pauseResume #:find completes #:in pausedThenCompleted #:limits six)
(query scheduleToStartTimeout #:find scheduleToStartFires #:in scheduleToStartExpires #:limits three)
(query startToCloseTimeout #:find startToCloseFires #:in startToCloseExpires #:limits three)
(query terminalHolds #:verify terminalIsFinal #:in completed #:limits three)
(query pauseHolds #:verify pausedIsNotDispatched #:in pausedThenCompleted #:limits six)

;; ### The sets
;;
;; The functional set drives both parties and does not repeat: standalone activities are CHASM
;; only, so there is no implementation switch to run under twice.

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

(set standaloneActivityExploration
  #:purpose exploratory
  #:bind ([caller driven] [worker driven])
  #:machine activityProtocol
  #:cover (rows results classMembers)
  #:budget four)

;; ### The activity and its worker
;;
;; Composed with the worker of the activity's task queue, the stop is the worker's own phase change
;; and every dispatch is the worker serving, so a dispatch has a row only while the worker polls.

(machine activityWorker
  #:from Worker.polling
  #:restrict (workerStop Worker.serve))

(struct StandaloneActivityState (activity worker) #:transparent)

(compose standaloneActivity
  #:for (activity Worker.worker)
  #:state StandaloneActivityState
  #:members ([activity activityProtocol] [worker activityWorker])
  #:sync ([workerStop activity.workerStop worker.workerStop]
          [attemptStart activity.attemptStart worker.serve])
  #:starts (activity.unstarted worker.polling)
  #:ends (activity.completed activity.failed activity.canceled activity.terminated activity.timedOut))

;; Every dispatch leaves the worker polling: no task is picked up while the worker is stopped.
(property startedByPollingWorker
  #:machine standaloneActivity
  #:when attemptStart
  #:holds (λ (s) (eq? (Worker.WorkerState-phase (StandaloneActivityState-worker (step-state s))) 'polling)))

;; An attempt starts while the worker polls and fails retryably; the backoff fires, then the worker
;; stops, so the retry is never picked up and the schedule-to-start deadline fires. The path performs
;; `attemptStart` once, so the claim fires and is not verified vacuously, as the earlier
;; `stoppedBeforeDispatch` was.
(scenario stoppedBeforeRetry
  #:model standaloneActivity
  #:starts activity.unstarted
  #:actions ([activity.start 'unset 'expires 'unset] attemptStart [activity.attemptResult (failed #t)]
             activity.backoff workerStop activity.scheduleToStart))

(query stoppedWorkerStartsNothing
  #:verify startedByPollingWorker
  #:in stoppedBeforeRetry
  #:limits six)

;; ### Where the checks run
;;
;; The refinement ran when this module was instantiated. Search runs here, under
;; `raco test standalone-activity.rkt`; pins.rkt reads the same tables.

(module+ test
  (require rackunit)
  (for ([q (list completion nonRetryableFailure retry cancel terminate pauseResume
                 scheduleToStartTimeout startToCloseTimeout)])
    (check-eq? (query-result-outcome (run-query q)) 'found (format "~a" q)))
  (for ([q (list terminalHolds pauseHolds stoppedWorkerStartsNothing)])
    (check-eq? (query-result-outcome (run-query q)) 'verified (format "~a" q))))
