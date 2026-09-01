import Umpire.ImplementationLink.Application
import Umpire.Observation.Verdict

/-! Semantic-stage constructor catalogs retain their owner-local vocabularies. -/

namespace Umpire.SemanticInventoryTests.SemanticStages

open Umpire

example : OutcomeConstructorClassifiers.names ObservationStatus.constructorClassifiers =
    ["accepted", "unknown", "conflict", "unsupported"] := by
  native_decide

example : OutcomeConstructorClassifiers.ExactlyOne ObservationStatus.constructorClassifiers :=
  ObservationStatus.constructorClassifiers_exactlyOne

example : OutcomeConstructorClassifiers.names ImplementationLinkStatus.constructorClassifiers =
    ["applied", "invalid", "unknown", "conflict", "unsupported"] := by
  native_decide

example :
    OutcomeConstructorClassifiers.ExactlyOne ImplementationLinkStatus.constructorClassifiers :=
  ImplementationLinkStatus.constructorClassifiers_exactlyOne

example : OutcomeConstructorClassifiers.names SemanticVerdictStatus.constructorClassifiers =
    ["satisfied", "violated", "unknown", "conflict", "unsupported"] := by
  native_decide

example : OutcomeConstructorClassifiers.ExactlyOne SemanticVerdictStatus.constructorClassifiers :=
  SemanticVerdictStatus.constructorClassifiers_exactlyOne

example : OutcomeConstructorClassifiers.names StrictQueryStatus.constructorClassifiers =
    ["satisfied", "violated", "incomplete"] := by
  native_decide

example : OutcomeConstructorClassifiers.ExactlyOne StrictQueryStatus.constructorClassifiers :=
  StrictQueryStatus.constructorClassifiers_exactlyOne

example :
    [ObservationStatus.accepted, .unknown, .conflict, .unsupported].map ObservationStatus.name =
      OutcomeConstructorClassifiers.names ObservationStatus.constructorClassifiers ∧
    [ImplementationLinkStatus.applied, .invalid, .unknown, .conflict, .unsupported].map
        ImplementationLinkStatus.name =
      OutcomeConstructorClassifiers.names ImplementationLinkStatus.constructorClassifiers ∧
    [SemanticVerdictStatus.satisfied, .violated, .unknown, .conflict, .unsupported].map
        SemanticVerdictStatus.name =
      OutcomeConstructorClassifiers.names SemanticVerdictStatus.constructorClassifiers ∧
    [StrictQueryStatus.satisfied, .violated, .incomplete].map StrictQueryStatus.name =
      OutcomeConstructorClassifiers.names StrictQueryStatus.constructorClassifiers := by
  native_decide

private def qualifiedNames
    (family : String)
    (classifiers : List (OutcomeConstructorClassifier Outcome)) : List (String × String) :=
  (OutcomeConstructorClassifiers.names classifiers).map fun name => (family, name)

private def semanticStageConstructorRows : List (String × String) :=
  qualifiedNames "observation" ObservationStatus.constructorClassifiers ++
  qualifiedNames "implementation-link" ImplementationLinkStatus.constructorClassifiers ++
  qualifiedNames "semantic-property" SemanticVerdictStatus.constructorClassifiers ++
  qualifiedNames "strict-query" StrictQueryStatus.constructorClassifiers

example : semanticStageConstructorRows.filter (fun row => row.2 == "unknown") = [
    ("observation", "unknown"),
    ("implementation-link", "unknown"),
    ("semantic-property", "unknown")
  ] := by
  native_decide

example :
    ImplementationLinkStatus.notEvaluatedProjectionSentinel = {
      id := "implementation-link.not-evaluated"
      owner := "Implementation Link"
      name := "not-evaluated"
      description := "The optional Implementation Link stage was not evaluated."
    } ∧
    (OutcomeConstructorClassifiers.names ImplementationLinkStatus.constructorClassifiers).contains
        ImplementationLinkStatus.notEvaluatedProjectionSentinel.name = false := by
  native_decide

end Umpire.SemanticInventoryTests.SemanticStages
