#lang racket
;; # The worker entity
;;
;; The worker of one task queue, as an entity of its own rather than a stutter row on the machines
;; of the work it serves. A polling worker serves its queue; the `worker` party stops it and resumes
;; it. The machine is the one a composition synchronizes with: a workflow or an operation whose
;; progress needs a worker names `serve` beside its own action, and a stopped worker has no row for
;; it.
;;
;; The module declares no set or Query: nothing here is realized on its own, and the Properties
;; about a worker are the cross-entity ones a composition states.

(require "umpire.rkt")

(provide worker workerStop workerResume serve polling
         (struct-out WorkerState) WorkerState-members WorkerPhase WorkerPhase-members)

;; ### Entities and domains

;; A worker is named by the task queue it polls: the handler's worker and the workflow's worker are
;; two instances of this entity, told apart by their queue.
(entity worker #:key taskQueue)

;; Racket has one namespace per module, so the enum is `WorkerPhase` rather than a second `Phase`:
;; the Lean file's namespace did that job. Its members are the symbols 'polling and 'stopped, so
;; the machine below may be called `polling` too.
(enum WorkerPhase polling stopped)

(record WorkerState [phase : WorkerPhase])

(enum WorkerOutcome accepted)

;; A worker records nothing of its own: its stop and resume are faults the Run records against no
;; entity, and what it serves is recorded by the work it serves.
(enum WorkerFact)

;; ### The actions
;;
;; The two faults are the `worker` party's and name no entity. The serve action is the worker's own
;; and takes no input, so a composition may synchronize it with an action of any class.

(action workerStop #:party worker)

(action workerResume #:party worker)

(action serve #:party worker #:on worker)

;; ### The machine

(define worker-step/c (-> WorkerState? (listof step?)))

;; A polling worker stops; a stopped one has nothing to stop.
(define/contract (stopStep state) worker-step/c
  (cases WorkerPhase (WorkerState-phase state)
    [polling (list (step 'accepted (WorkerState 'stopped) '()))]
    [stopped '()]))

;; A stopped worker resumes polling; a polling one has nothing to resume.
(define/contract (resumeStep state) worker-step/c
  (cases WorkerPhase (WorkerState-phase state)
    [stopped (list (step 'accepted (WorkerState 'polling) '()))]
    [polling '()]))

;; A polling worker serves and keeps polling; a stopped one serves nothing.
(define/contract (serveStep state) worker-step/c
  (cases WorkerPhase (WorkerState-phase state)
    [polling (list (step 'accepted state '()))]
    [stopped '()]))

;; A worker has no natural end: it may be left polling or stopped.
(machine polling
  #:for worker
  #:state WorkerState
  #:starts (polling)
  #:ends (polling stopped)
  #:steps ([workerStop stopStep]
           [workerResume resumeStep]
           [serve serveStep]))
