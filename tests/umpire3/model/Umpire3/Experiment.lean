import Umpire3.Executable

namespace Umpire3

structure SemanticResource where
  identifier : String
  kind : String
  deriving DecidableEq, Repr

structure SemanticAction where
  identifier : String
  kind : String
  requiredCapabilities : List String
  preCheckpoint : Option String
  postCheckpoint : Option String
  deriving DecidableEq, Repr

structure SemanticCheckpoint where
  identifier : String
  observation : String
  ordering : String
  omissionPolicy : String
  deriving DecidableEq, Repr

structure SemanticExperiment where
  identifier : String
  modelModules : List String
  propertyIdentifier : String
  scope : ExplorationScope
  resources : List SemanticResource
  actions : List SemanticAction
  checkpoints : List SemanticCheckpoint
  provenance : String
  deriving DecidableEq, Repr

def SemanticExperiment.WellFormed (experiment : SemanticExperiment) : Prop :=
  experiment.identifier ≠ "" ∧
    experiment.modelModules ≠ [] ∧
    experiment.propertyIdentifier ≠ "" ∧
    experiment.actions ≠ [] ∧
    experiment.checkpoints ≠ [] ∧
    ∀ action ∈ experiment.actions,
      (∀ checkpoint, action.preCheckpoint = some checkpoint →
        ∃ candidate ∈ experiment.checkpoints, candidate.identifier = checkpoint) ∧
      (∀ checkpoint, action.postCheckpoint = some checkpoint →
        ∃ candidate ∈ experiment.checkpoints, candidate.identifier = checkpoint)

end Umpire3
