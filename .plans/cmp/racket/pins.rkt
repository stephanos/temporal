#lang racket
;; # What the two Models say
;;
;; The claims below are about the tables, because the tables are what Search, the Behavior
;; Fingerprint and Contract lowering read. A `cases` arm that stopped saying what it says would fail
;; here. Run with `raco test pins.rkt`; instantiating either Model already built its tables and
;; checked its refinement, so a rejected refinement fails before the first `check`.
;;
;; Where Lean writes `#guard` and the compiler answers, Racket writes `check-equal?` and `raco test`
;; answers; the claim is the same, the moment is one step later.

(require rackunit
         "umpire.rkt"
         (prefix-in nexus: "nexus-caller.rkt")
         (prefix-in act: "standalone-activity.rkt"))

;; A protocol state written the way a reader names one: the phase, and whichever fields are not at
;; the value the operation begins with.
(define (nexus-at phase #:attempts [attempts 0] #:scheduleToClose [stc 'unset]
                  #:scheduleToStart [sts 'unset] #:startToClose [stcl 'unset])
  (nexus:ProtocolState phase attempts stc sts stcl))

(define (act-at phase #:attempts [attempts 0] #:scheduleToClose [stc 'unset]
                #:scheduleToStart [sts 'unset] #:startToClose [stcl 'unset])
  (act:ProtocolState phase attempts stc sts stcl))

(define (found? q) (eq? (query-result-outcome (run-query q)) 'found))
(define (verified? q) (eq? (query-result-outcome (run-query q)) 'verified))

;; ### Nexus: the product machine

(test-case "nexusProduct: six phases, and the four the design ends on"
  (check-equal? (length (table-states (machine-table nexus:nexusProduct))) 6)
  (check-equal? (length (machine-ends nexus:nexusProduct)) 4))

;; Every action class the machine steps on: six replies, three resolutions, the two faults it cannot
;; see, and the one timer.
(test-case "nexusProduct: twelve action classes"
  (check-equal? (length (machine-action-keys nexus:nexusProduct)) 12))

;; A retryable handler error is invisible here: it is the protocol machine that backs off.
(test-case "nexusProduct: handlerError true from scheduled is []"
  (check-equal? (nexus:handlerReplyStep (nexus:ProductState 'scheduled) (nexus:handlerError #t)) '()))

;; What the Model actually reaches: every phase.
(test-case "nexusProduct: every phase is reachable"
  (check-equal? (length (reachable-from (machine-starts nexus:nexusProduct)
                                        (machine-transitions nexus:nexusProduct)))
                6))

(test-case "nexusProduct: nothing is stuck"
  (check-false (machine-stuck nexus:nexusProduct)))

;; ### Nexus: the protocol machine

;; Eight phases, three attempt counts and three deadlines, and the four phases the design ends on.
(test-case "nexusProtocol: 192 states, 96 ends"
  (check-equal? (length (table-states (machine-table nexus:nexusProtocol))) (* 8 3 2 2 2))
  (check-equal? (length (machine-ends nexus:nexusProtocol)) (* 4 3 2 2 2)))

;; Eight schedule commands, six replies, three resolutions, the two faults and the four timers. The
;; catalog is in canonical order, so it opens on the backoff timer rather than on `schedule`.
(test-case "nexusProtocol: 23 action classes in catalog order"
  (check-equal? (length (machine-action-keys nexus:nexusProtocol)) (+ 8 6 3 1 1 4))
  (check-equal? (take (machine-action-keys nexus:nexusProtocol) 2) '("backoff" "complete-canceled")))

;; A retryable handler error backs the operation off and raises the attempt count. No history event
;; records that, which is why its evidence is the derived `pendingAttempts` observation.
(test-case "nexusProtocol: retryable error backs off"
  (check-equal? (nexus:protocolHandlerReplyStep (nexus-at 'scheduled) (nexus:handlerError #t))
                (list (step 'accepted (nexus-at 'backingOff #:attempts 1) '(pendingAttempts)))))

;; The count saturates rather than wrapping.
(test-case "nexusProtocol: the attempt count saturates"
  (check-equal? (map (compose1 nexus:ProtocolState-attempts step-state)
                     (nexus:protocolHandlerReplyStep (nexus-at 'scheduled #:attempts nexus:attemptBound)
                                                     (nexus:handlerError #t)))
                (list nexus:attemptBound)))

;; A completion before the start records the Started event first; one after it does not.
(test-case "nexusProtocol: completion records Started first when needed"
  (check-equal? (append-map step-facts (nexus:protocolCompleteStep (nexus-at 'backingOff #:attempts 1) 'succeeded))
                '(nexusOperationStarted nexusOperationCompleted))
  (check-equal? (append-map step-facts (nexus:protocolCompleteStep (nexus-at 'started) 'succeeded))
                '(nexusOperationCompleted)))

;; A completion after the operation is over is not found and changes nothing.
(test-case "nexusProtocol: completion after the end is notFound"
  (check-equal? (nexus:protocolCompleteStep (nexus-at 'timedOut) 'succeeded)
                (list (step 'notFound (nexus-at 'timedOut) '()))))

;; A timer fires only when the schedule command set it, and each covers its own span; which timer
;; fired is recorded.
(test-case "nexusProtocol: timers cover their spans"
  (check-equal? (nexus:startToCloseStep (nexus-at 'scheduled #:startToClose 'expires)) '())
  (check-equal? (map (compose1 nexus:ProtocolState-phase step-state)
                     (nexus:startToCloseStep (nexus-at 'started #:startToClose 'expires)))
                '(timedOut))
  (check-equal? (nexus:scheduleToCloseStep (nexus-at 'started)) '())
  (check-equal? (append-map step-facts (nexus:scheduleToStartStep (nexus-at 'scheduled #:scheduleToStart 'expires)))
                (list (nexus:nexusOperationTimedOut 'scheduleToStart))))

;; The worker stop keeps the state and records nothing on the protocol; the product does not see it.
(test-case "workerStop: stutter on the protocol, invisible on the product"
  (check-equal? (nexus:protocolWorkerStopStep (nexus-at 'scheduled #:scheduleToStart 'expires))
                (list (step 'accepted (nexus-at 'scheduled #:scheduleToStart 'expires) '())))
  (check-equal? (nexus:workerStopStep (nexus:ProductState 'scheduled)) '()))

(test-case "nexusProtocol: nothing is stuck"
  (check-false (machine-stuck nexus:nexusProtocol)))

;; ### Nexus: the refinement
;;
;; `#:refines nexusProduct` with `#:map productOf` walked every protocol row through the map. A row
;; whose mapped states are equal is a stutter; otherwise some product row must move between them,
;; under any class, and `refinement-rows` names the class that did.

(test-case "nexusProtocol refines nexusProduct"
  (define r (machine-refinement nexus:nexusProtocol))
  (check-false (refinement-rejected r))
  (check-equal? (dict-ref (refinement-rows r) "scheduled-0-unset-unset-unset-handlerReply-async")
                "handlerReply-async")
  (check-equal? (dict-ref (refinement-rows r) "scheduled-0-unset-unset-unset-handlerReply-handlerError-true")
                #f)
  (check-equal? (dict-ref (refinement-rows r) "backingOff-1-unset-unset-unset-backoff") #f)
  ;; a deadline firing is the product's one timer, whichever deadline it was
  (check-equal? (dict-ref (refinement-rows r) "started-0-unset-unset-expires-startToClose") "timeout"))

;; ### Nexus: the Queries

(test-case "nexus: each functional Query finds its claim on its path"
  (for ([q (list nexus:syncCompletion nexus:asyncCompletion nexus:asyncFailure nexus:handlerErrorQuery
                 nexus:retry nexus:scheduleToStartTimeout nexus:startToCloseTimeout)])
    (check-true (found? q) (format "~a" (query-name q)))))

(test-case "nexus: terminalIsFinal verifies on the async path"
  (check-true (verified? nexus:terminalHolds)))

;; A protocol Scenario names its classed actions with their inputs, and its start by its phase.
(test-case "nexus: scenario occurrences are class keys"
  (check-equal? (scenario-occurrences nexus:asyncThenSucceeded)
                '("schedule-unset-unset-unset" "handlerReply-async" "complete-succeeded"))
  (check-equal? (scenario-occurrences nexus:retriedThenSucceeded)
                '("schedule-unset-unset-unset" "handlerReply-handlerError-true" "backoff"
                  "handlerReply-syncSuccess")))

;; ### Activity: the product machine

(test-case "activityProduct: nine phases, five ends"
  (check-equal? (length (table-states (machine-table act:activityProduct))) 9)
  (check-equal? (length (machine-ends act:activityProduct)) 5))

;; ### Activity: the protocol machine

;; Twelve phases, three attempt counts and three deadlines, and the five phases the design ends on.
(test-case "activityProtocol: 288 states, 120 ends"
  (check-equal? (length (table-states (machine-table act:activityProtocol))) (* 12 3 8))
  (check-equal? (length (machine-ends act:activityProtocol)) (* 5 3 8)))

;; A cancel result without a cancel request has no row.
(test-case "activityProtocol: canceled from started is []"
  (check-equal? (act:protocolAttemptResultStep (act-at 'started #:attempts 1) 'canceled) '()))

;; The second dispatch raises the count to two, and it is what `retryCompletes` fixes.
(test-case "activityProtocol: a retry raises the count"
  (check-equal? (map (compose1 act:ProtocolState-attempts step-state)
                     (act:protocolAttemptStartStep (act-at 'scheduled #:attempts 1)))
                '(2)))

;; A requested pause reads as started: the worker still holds the attempt.
(test-case "activityProtocol: pauseRequested reads as started"
  (check-equal? (act:productOf (act-at 'pauseRequested #:attempts 1)) (act:ProductState 'started)))

;; The product sees the retry, unlike the Nexus product.
(test-case "activityProduct: a retryable failure reads as scheduled again"
  (check-equal? (act:attemptResultStep (act:ProductState 'started) (act:failed #t))
                (list (step 'accepted (act:ProductState 'scheduled) '(statusScheduled)))))

(test-case "activityProtocol refines activityProduct"
  (check-false (refinement-rejected (machine-refinement act:activityProtocol))))

;; ### Activity: the Queries

(test-case "activity: each functional Query finds its claim on its path"
  (for ([q (list act:completion act:nonRetryableFailure act:retry act:cancel act:terminate
                 act:pauseResume act:scheduleToStartTimeout act:startToCloseTimeout)])
    (check-true (found? q) (format "~a" (query-name q)))))

(test-case "activity: the product claims verify on protocol paths"
  (check-true (verified? act:terminalHolds))
  (check-true (verified? act:pauseHolds)))
