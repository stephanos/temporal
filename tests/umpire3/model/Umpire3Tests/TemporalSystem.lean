import Temporal.Refinement.NexusTasks

namespace Umpire3.Temporal.System.NexusTasks.Tests

def scheduled : State := { initial with scheduled := true }
def dispatched : State := { scheduled with task := .dispatched, workerEpoch := some 0 }
def cancellationRequested : State := { dispatched with
  cancellation := { noProgress with attempted := true, applied := true }
  visible := .cancellationAccepted }
def cancellationCommitted : State := { cancellationRequested with
  cancellation := { cancellationRequested.cancellation with
    committed := true, persisted := true, observed := true }
  visible := .cancelled }
def ownershipChanged : State := { cancellationCommitted with
  ownerEpoch := 1
  staleWorkerEpoch := some 0 }
def retried : State := { ownershipChanged with
  task := .dispatched
  attempt := 1
  workerEpoch := some 1
  completionEpoch := none }
def staleReturned : State := { retried with
  task := .returned
  staleWorkerEpoch := none
  completionEpoch := some 0
  success := { noProgress with attempted := true, applied := true } }
def unsafePersisted : State := { staleReturned with
  success := { staleReturned.success with
    committed := true, persisted := true, observed := true }
  visible := .succeeded }

example : system.Step initial .ScheduleOperation scheduled := by
  apply (next_iff initial .ScheduleOperation scheduled).mp
  simp [next, transitions, scheduled, initial, stutterResult]

example : system.Step scheduled .DispatchTask dispatched := by
  apply (next_iff scheduled .DispatchTask dispatched).mp
  simp [next, transitions, dispatched, scheduled, initial, stutterResult]

example : system.Step dispatched .RequestCancellation cancellationRequested := by
  apply (next_iff dispatched .RequestCancellation cancellationRequested).mp
  simp [next, transitions, productResult, cancellationRequested, dispatched, scheduled, initial,
    noProgress]

example : system.Step cancellationRequested .CommitCancellation cancellationCommitted := by
  apply (next_iff cancellationRequested .CommitCancellation cancellationCommitted).mp
  simp [next, transitions, productResult, cancellationCommitted, cancellationRequested,
    dispatched, scheduled, initial, noProgress]

example : system.Step cancellationCommitted .AcquireOwnership ownershipChanged := by
  apply (next_iff cancellationCommitted .AcquireOwnership ownershipChanged).mp
  simp [next, transitions, stutterResult, ownershipChanged, cancellationCommitted,
    cancellationRequested, dispatched, scheduled, initial, noProgress]

example : system.Step ownershipChanged .RetryTask retried := by
  apply (next_iff ownershipChanged .RetryTask retried).mp
  simp [next, transitions, stutterResult, retried, ownershipChanged, cancellationCommitted,
    cancellationRequested, dispatched, scheduled, initial, noProgress]

example : system.Step retried .WorkerReturnsSuccess staleReturned := by
  apply (next_iff retried .WorkerReturnsSuccess staleReturned).mp
  simp [next, transitions, stutterResult, returnedEpoch, staleReturned, retried,
    ownershipChanged, cancellationCommitted, cancellationRequested, dispatched, scheduled,
    initial, noProgress]

example : next staleReturned .PersistSuccess = [] := by
  simp [next, transitions, staleReturned, retried, ownershipChanged, cancellationCommitted,
    cancellationRequested, dispatched, scheduled, initial, noProgress]

example : unsafeSystem.Step staleReturned .PersistSuccess unsafePersisted := by
  simp [unsafeStep, unsafeNext, unsafePersisted, staleReturned, retried, ownershipChanged,
    cancellationCommitted, cancellationRequested, dispatched, scheduled, initial, noProgress]

example : staleReturned.completionEpoch = some 0 := by rfl

example : staleReturned.ownerEpoch = 1 := by rfl

example : Projects initial Temporal.Product.Nexus.initial := rfl

example : Refinement system Temporal.Product.Nexus.product := nexusTasksRefinesProduct

example : StutteringAction .RetryTask := by
  intro state result member
  simp [transitions] at member
  rcases member with ⟨_, rfl⟩
  rfl

end Umpire3.Temporal.System.NexusTasks.Tests
