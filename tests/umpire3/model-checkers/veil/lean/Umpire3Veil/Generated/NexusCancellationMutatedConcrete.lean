import Veil

set_option veil.__modelCheckCompileMode true

veil module NexusCancellationStaleCompletionGuardRemovedConcrete

enum Lifecycle = {Open, CancellationAccepted, Cancelled, Succeeded}
enum TaskStage = {Idle, Dispatched, Returned}
enum Epoch = {None, Epoch0, Epoch1}

individual lifecycle : Lifecycle
individual task : TaskStage
individual ownerEpoch : Epoch
individual workerEpoch : Epoch
individual completionEpoch : Epoch

#gen_state

after_init {
  lifecycle := *
  task := *
  ownerEpoch := *
  workerEpoch := *
  completionEpoch := *
  assume (((((lifecycle = Open) ∧ (task = Idle)) ∧ (ownerEpoch = Epoch0)) ∧ (workerEpoch = None)) ∧ (completionEpoch = None))
}

action DispatchTask {
  let preOwnerEpoch := ownerEpoch
  require ((task = Idle) ∧ (lifecycle = Open))
  task := Dispatched
  workerEpoch := preOwnerEpoch
}

action RequestCancellation {
  require (lifecycle = Open)
  lifecycle := CancellationAccepted
}

action AcquireOwnership {
  require (((lifecycle = Open) ∨ (lifecycle = CancellationAccepted)) ∧ (ownerEpoch = Epoch0))
  ownerEpoch := Epoch1
}

action CommitCancellation {
  require (lifecycle = CancellationAccepted)
  lifecycle := Cancelled
}

action WorkerReturnsSuccess {
  let preWorkerEpoch := workerEpoch
  require (task = Dispatched)
  task := Returned
  completionEpoch := preWorkerEpoch
}

action PersistSuccess {
  require ((task = Returned) ∧ (¬ (completionEpoch = None)))
  lifecycle := Succeeded
}

safety [NexusCancellationWonExcludesSuccess] ((¬ (lifecycle = Succeeded)) ∨ (completionEpoch = ownerEpoch))

invariant [CanonicalReachableEnvelope] (((lifecycle = Open) ∧ (task = Idle) ∧ (ownerEpoch = Epoch0) ∧ (workerEpoch = None) ∧ (completionEpoch = None)) ∨ ((lifecycle = Open) ∧ (task = Dispatched) ∧ (ownerEpoch = Epoch0) ∧ (workerEpoch = Epoch0) ∧ (completionEpoch = None)) ∨ ((lifecycle = CancellationAccepted) ∧ (task = Idle) ∧ (ownerEpoch = Epoch0) ∧ (workerEpoch = None) ∧ (completionEpoch = None)) ∨ ((lifecycle = Open) ∧ (task = Idle) ∧ (ownerEpoch = Epoch1) ∧ (workerEpoch = None) ∧ (completionEpoch = None)) ∨ ((lifecycle = CancellationAccepted) ∧ (task = Dispatched) ∧ (ownerEpoch = Epoch0) ∧ (workerEpoch = Epoch0) ∧ (completionEpoch = None)) ∨ ((lifecycle = Open) ∧ (task = Dispatched) ∧ (ownerEpoch = Epoch1) ∧ (workerEpoch = Epoch0) ∧ (completionEpoch = None)) ∨ ((lifecycle = Open) ∧ (task = Returned) ∧ (ownerEpoch = Epoch0) ∧ (workerEpoch = Epoch0) ∧ (completionEpoch = Epoch0)) ∨ ((lifecycle = CancellationAccepted) ∧ (task = Idle) ∧ (ownerEpoch = Epoch1) ∧ (workerEpoch = None) ∧ (completionEpoch = None)) ∨ ((lifecycle = Cancelled) ∧ (task = Idle) ∧ (ownerEpoch = Epoch0) ∧ (workerEpoch = None) ∧ (completionEpoch = None)) ∨ ((lifecycle = Open) ∧ (task = Dispatched) ∧ (ownerEpoch = Epoch1) ∧ (workerEpoch = Epoch1) ∧ (completionEpoch = None)) ∨ ((lifecycle = CancellationAccepted) ∧ (task = Dispatched) ∧ (ownerEpoch = Epoch1) ∧ (workerEpoch = Epoch0) ∧ (completionEpoch = None)) ∨ ((lifecycle = Cancelled) ∧ (task = Dispatched) ∧ (ownerEpoch = Epoch0) ∧ (workerEpoch = Epoch0) ∧ (completionEpoch = None)) ∨ ((lifecycle = CancellationAccepted) ∧ (task = Returned) ∧ (ownerEpoch = Epoch0) ∧ (workerEpoch = Epoch0) ∧ (completionEpoch = Epoch0)) ∨ ((lifecycle = Open) ∧ (task = Returned) ∧ (ownerEpoch = Epoch1) ∧ (workerEpoch = Epoch0) ∧ (completionEpoch = Epoch0)) ∨ ((lifecycle = Succeeded) ∧ (task = Returned) ∧ (ownerEpoch = Epoch0) ∧ (workerEpoch = Epoch0) ∧ (completionEpoch = Epoch0)) ∨ ((lifecycle = Cancelled) ∧ (task = Idle) ∧ (ownerEpoch = Epoch1) ∧ (workerEpoch = None) ∧ (completionEpoch = None)) ∨ ((lifecycle = CancellationAccepted) ∧ (task = Dispatched) ∧ (ownerEpoch = Epoch1) ∧ (workerEpoch = Epoch1) ∧ (completionEpoch = None)) ∨ ((lifecycle = Open) ∧ (task = Returned) ∧ (ownerEpoch = Epoch1) ∧ (workerEpoch = Epoch1) ∧ (completionEpoch = Epoch1)) ∨ ((lifecycle = Cancelled) ∧ (task = Dispatched) ∧ (ownerEpoch = Epoch1) ∧ (workerEpoch = Epoch0) ∧ (completionEpoch = None)) ∨ ((lifecycle = CancellationAccepted) ∧ (task = Returned) ∧ (ownerEpoch = Epoch1) ∧ (workerEpoch = Epoch0) ∧ (completionEpoch = Epoch0)) ∨ ((lifecycle = Cancelled) ∧ (task = Returned) ∧ (ownerEpoch = Epoch0) ∧ (workerEpoch = Epoch0) ∧ (completionEpoch = Epoch0)) ∨ ((lifecycle = Succeeded) ∧ (task = Returned) ∧ (ownerEpoch = Epoch1) ∧ (workerEpoch = Epoch0) ∧ (completionEpoch = Epoch0)) ∨ ((lifecycle = Cancelled) ∧ (task = Dispatched) ∧ (ownerEpoch = Epoch1) ∧ (workerEpoch = Epoch1) ∧ (completionEpoch = None)) ∨ ((lifecycle = CancellationAccepted) ∧ (task = Returned) ∧ (ownerEpoch = Epoch1) ∧ (workerEpoch = Epoch1) ∧ (completionEpoch = Epoch1)) ∨ ((lifecycle = Succeeded) ∧ (task = Returned) ∧ (ownerEpoch = Epoch1) ∧ (workerEpoch = Epoch1) ∧ (completionEpoch = Epoch1)) ∨ ((lifecycle = Cancelled) ∧ (task = Returned) ∧ (ownerEpoch = Epoch1) ∧ (workerEpoch = Epoch0) ∧ (completionEpoch = Epoch0)) ∨ ((lifecycle = Cancelled) ∧ (task = Returned) ∧ (ownerEpoch = Epoch1) ∧ (workerEpoch = Epoch1) ∧ (completionEpoch = Epoch1)))

#gen_spec

#model_check {  } { } (sequential := true)
