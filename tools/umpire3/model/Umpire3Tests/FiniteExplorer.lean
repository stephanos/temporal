import Temporal.Families.NexusCancellation.Targets.Finite
import Umpire3.Explorer

namespace Umpire3.Tests.FiniteExplorer

open Umpire3.Temporal.Targets.NexusCancellationFencing

def limits : Exact.Limits where
  maxDepth := 16
  maxStates := 256
  maxTransitions := 4096
  maxStateBytes := 16384

def soundSearch := Exact.explore soundFiniteView noStaleCompletion limits

def mutatedSearch := Exact.explore mutatedFiniteView noStaleCompletion limits

example : Exact.classify soundFiniteView noStaleCompletion soundSearch =
    .finiteExhaustive := by
  decide

example : Safety
    (Umpire3.Temporal.System.NexusCancellationFencing.behavior.at .smoke)
    (fun state => noStaleCompletion state = true) :=
  Exact.classify_finiteExhaustive_sound soundFiniteView noStaleCompletion soundSearch (by decide)

example {node} (member : node ∈ soundSearch.certificate.nodes) :
    Umpire3.Temporal.System.NexusCancellationFencing.behavior.Reachable
      .smoke node.state :=
  Exact.explore_nodes_reachable soundFiniteView noStaleCompletion limits member

example (state) :
    soundSearch.certificate.Contains state ↔
      Umpire3.Temporal.System.NexusCancellationFencing.behavior.Reachable .smoke state :=
  Exact.explore_finiteExhaustive_exact soundFiniteView noStaleCompletion limits
    (by decide) state

example : Exact.classify mutatedFiniteView noStaleCompletion mutatedSearch =
    .traceWitness := by
  decide

example : ∃ state,
    Umpire3.Temporal.System.NexusCancellationFencing.mutatedBehavior.Reachable .smoke state ∧
      noStaleCompletion state = false :=
  Exact.classify_traceWitness_sound mutatedFiniteView noStaleCompletion mutatedSearch (by decide)

example : ∃ witness, mutatedSearch.witness? = some witness ∧
    Umpire3.Temporal.System.NexusCancellationFencing.mutatedBehavior.Reachable
      .smoke witness.state ∧
    noStaleCompletion witness.state = false ∧
    ∀ state,
      Exact.ReachableBefore
        Umpire3.Temporal.System.NexusCancellationFencing.mutatedBehavior
        .smoke witness.depth state →
      noStaleCompletion state = true :=
  Exact.explore_traceWitness_shortest mutatedFiniteView noStaleCompletion limits (by decide)

example : mutatedSearch.witness?.map (fun witness => witness.actions) = some [
    .dispatchTask,
    .acquireOwnership,
    .returnSuccess,
    .persistSuccess,
  ] := by
  decide

def mutatedBeforeWitnessDepth := Exact.explore mutatedFiniteView noStaleCompletion
  { limits with maxDepth := 3 }

example : Exact.classify mutatedFiniteView noStaleCompletion mutatedBeforeWitnessDepth =
    .boundedSafe 3 := by
  decide

example : ∀ state,
    Exact.ReachableWithin
      Umpire3.Temporal.System.NexusCancellationFencing.mutatedBehavior .smoke 3 state →
      noStaleCompletion state = true :=
  Exact.classify_boundedSafe_sound mutatedFiniteView noStaleCompletion
    mutatedBeforeWitnessDepth 3 (by decide)

example (state)
    (reachable : Exact.ReachableWithin
      Umpire3.Temporal.System.NexusCancellationFencing.mutatedBehavior .smoke 3 state) :
    mutatedBeforeWitnessDepth.certificate.Contains state :=
  Exact.explore_boundedSafe_covers mutatedFiniteView noStaleCompletion
    { limits with maxDepth := 3 } 3 (by decide) state reachable

def corruptedSoundSearch := soundSearch.mapCertificate fun certificate =>
  { certificate with nodes := certificate.nodes.drop 1 }

example : Exact.classify soundFiniteView noStaleCompletion corruptedSoundSearch =
    .invalidCertificate := by
  decide

def missingSuccessorSearch := soundSearch.mapCertificate fun certificate =>
  { certificate with nodes := certificate.nodes.dropLast }

example : Exact.classify soundFiniteView noStaleCompletion missingSuccessorSearch =
    .invalidCertificate := by
  decide

example : Exact.classify soundFiniteView (fun _ => false) soundSearch =
    .invalidCertificate := by
  decide

def illegalTraceSearch := soundSearch.mapCertificate fun certificate =>
  match certificate.nodes with
  | [] => certificate
  | node :: nodes =>
      { certificate with nodes := { node with actions := [.persistSuccess] } :: nodes }

example : Exact.classify soundFiniteView noStaleCompletion illegalTraceSearch =
    .invalidCertificate := by
  decide

def collidedSoundView := soundFiniteView.withFingerprint fun _ => 0

def collidedSoundSearch := Exact.explore collidedSoundView noStaleCompletion limits

example : Exact.classify collidedSoundView noStaleCompletion collidedSoundSearch =
    .finiteExhaustive := by
  decide

example : collidedSoundSearch.statistics.visited = soundSearch.statistics.visited := by
  decide

example : collidedSoundSearch.statistics.collisions > 0 := by
  decide

def depthLimits : Exact.Limits := { limits with maxDepth := 2 }

def depthSearch := Exact.explore soundFiniteView noStaleCompletion depthLimits

example : Exact.classify soundFiniteView noStaleCompletion depthSearch =
    .boundedSafe 2 := by
  decide

example : ∀ state,
    Exact.ReachableWithin
      Umpire3.Temporal.System.NexusCancellationFencing.behavior .smoke 2 state →
      noStaleCompletion state = true :=
  Exact.classify_boundedSafe_sound soundFiniteView noStaleCompletion depthSearch 2 (by decide)

def stateLimitedSearch := Exact.explore soundFiniteView noStaleCompletion
  { limits with maxStates := 1 }

example : Exact.classify soundFiniteView noStaleCompletion stateLimitedSearch =
    .resourceLimited .states := by
  decide

def transitionLimitedSearch := Exact.explore soundFiniteView noStaleCompletion
  { limits with maxTransitions := 0 }

example : Exact.classify soundFiniteView noStaleCompletion transitionLimitedSearch =
    .resourceLimited .transitions := by
  decide

def memoryLimitedSearch := Exact.explore soundFiniteView noStaleCompletion
  { limits with maxStateBytes := 47 }

example : Exact.classify soundFiniteView noStaleCompletion memoryLimitedSearch =
    .resourceLimited .stateBytes := by
  decide

def workLimitedSearch := Exact.explore soundFiniteView noStaleCompletion
  { limits with maxWork := 1 }

example : Exact.classify soundFiniteView noStaleCompletion workLimitedSearch =
    .resourceLimited .work := by
  decide

example : soundSearch.statistics.coverage.contains "persist-success" := by
  set_option maxRecDepth 100000 in
    decide

example : !soundSearch.statistics.deadlocks.isEmpty := by
  decide

inductive GraphState where
  | leftInitial
  | rightInitial
  | merged
  | deadlocked
  deriving DecidableEq, Repr

inductive GraphAction where
  | advance
  | loop
  | finish
  deriving DecidableEq, Repr

def graphSuccessors : GraphState → List (GraphAction × GraphState)
  | .leftInitial => [(.advance, .merged), (.advance, .deadlocked)]
  | .rightInitial => [(.advance, .merged)]
  | .merged => [(.loop, .merged), (.finish, .deadlocked)]
  | .deadlocked => []

abbrev graphBehavior : Behavior Unit where
  State := fun _ => GraphState
  Action := fun _ => GraphAction
  Initial := fun _ state => state = .leftInitial ∨ state = .rightInitial
  Step := fun _ state action nextState => (action, nextState) ∈ graphSuccessors state

def graphExecutable : ExecutableView graphBehavior where
  initials := fun _ => [.leftInitial, .rightInitial]
  successors := fun _ => graphSuccessors
  initials_exact := by
    intro world state
    cases world
    cases state <;> decide
  successors_exact := by
    intro _ _ _ _
    rfl

def graphIdentity : StateIdentity GraphState where
  Code := GraphState
  codeDecidableEq := inferInstance
  encode := id
  encode_injective := fun _ _ equality => equality
  fingerprint := fun
    | .leftInitial => 0
    | .rightInitial => 1
    | .merged => 2
    | .deadlocked => 3
  encodedSize := fun _ => 1

def graphView : FiniteView graphBehavior () where
  executable := graphExecutable
  identity := graphIdentity
  actionDecidableEq := inferInstance
  actionName := fun
    | .advance => "advance"
    | .loop => "loop"
    | .finish => "finish"
  actionName_injective := by
    intro left right
    cases left <;> cases right <;> simp_all

def graphLimits : Exact.Limits where
  maxDepth := 8
  maxStates := 16
  maxTransitions := 32
  maxStateBytes := 16

def graphSearch := Exact.explore graphView (fun _ => true) graphLimits

example : Exact.classify graphView (fun _ => true) graphSearch = .finiteExhaustive := by
  decide

example : graphSearch.statistics.visited = 4 := by
  decide

example : graphSearch.statistics.transitions = 5 := by
  decide

example : graphSearch.statistics.deadlocks = [.deadlocked] := by
  decide

example : graphSearch.statistics.coverage = ["advance", "loop", "finish"] := by
  decide

def shortestGraphSearch := Exact.explore graphView
  (fun state => decide (state ≠ .deadlocked)) graphLimits

example : shortestGraphSearch.witness?.map (fun witness => witness.actions) =
    some [.advance] := by
  decide

end Umpire3.Tests.FiniteExplorer
