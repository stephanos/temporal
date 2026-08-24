import Temporal.Families.NexusCancellation.Refinement
import Temporal.Families.NexusCancellation.Targets.Finite
import Umpire3.Mutation

namespace Umpire3.Temporal.Mutations.NexusCancellationFencing

open Umpire3.Temporal.Targets.NexusCancellationFencing

def limits : Exact.Limits where
  maxDepth := 16
  maxStates := 256
  maxTransitions := 4096
  maxStateBytes := 16384

def exactSearch := Exact.explore mutatedFiniteView noStaleCompletion limits

theorem exactMutationWitness :
    ExactMutationDetected (Exact.classify mutatedFiniteView noStaleCompletion exactSearch) := by
  change Exact.classify mutatedFiniteView noStaleCompletion exactSearch = .traceWitness
  decide

def refinementManifest : SemanticProofManifest where
  identifier := "nexus-cancellation-mutation-rejection-v1"
  proof := resolved_mutation_rejection%
    Umpire3.Temporal.Refinement.NexusCancellationFencing.mutationBreaksDeclaredSimulation

def exactManifest : SemanticProofManifest where
  identifier := "nexus-cancellation-exact-witness-v1"
  proof := resolved_exact_mutation% exactMutationWitness

end Umpire3.Temporal.Mutations.NexusCancellationFencing
