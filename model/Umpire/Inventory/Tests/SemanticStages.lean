import Umpire.ImplementationLink.Application
import Umpire.Evidence.PropertyStatus

/-! Semantic-stage constructor catalogs retain their owner-local vocabularies. -/

namespace Umpire.InventoryTests.SemanticStages

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

example : OutcomeConstructorClassifiers.names Evidence.PropertyStatus.constructorClassifiers =
    ["satisfied", "violated", "unknown", "conflict", "unsupported"] := by
  native_decide

example : OutcomeConstructorClassifiers.ExactlyOne Evidence.PropertyStatus.constructorClassifiers :=
  Evidence.PropertyStatus.constructorClassifiers_exactlyOne

example : OutcomeConstructorClassifiers.names QueryStatus.constructorClassifiers =
    ["satisfied", "violated", "incomplete"] := by
  native_decide

example : OutcomeConstructorClassifiers.ExactlyOne QueryStatus.constructorClassifiers :=
  QueryStatus.constructorClassifiers_exactlyOne

example :
    [ObservationStatus.accepted, .unknown, .conflict, .unsupported].map ObservationStatus.name =
      OutcomeConstructorClassifiers.names ObservationStatus.constructorClassifiers ∧
    [ImplementationLinkStatus.applied, .invalid, .unknown, .conflict, .unsupported].map
        ImplementationLinkStatus.name =
      OutcomeConstructorClassifiers.names ImplementationLinkStatus.constructorClassifiers ∧
    [Evidence.PropertyStatus.satisfied, .violated, .unknown, .conflict, .unsupported].map
        Evidence.PropertyStatus.name =
      OutcomeConstructorClassifiers.names Evidence.PropertyStatus.constructorClassifiers ∧
    [QueryStatus.satisfied, .violated, .incomplete].map QueryStatus.name =
      OutcomeConstructorClassifiers.names QueryStatus.constructorClassifiers := by
  native_decide

private def qualifiedNames
    (family : String)
    (classifiers : List (OutcomeConstructorClassifier Outcome)) : List (String × String) :=
  (OutcomeConstructorClassifiers.names classifiers).map fun name => (family, name)

private def semanticStageConstructorRows : List (String × String) :=
  qualifiedNames "observation" ObservationStatus.constructorClassifiers ++
  qualifiedNames "implementation-link" ImplementationLinkStatus.constructorClassifiers ++
  qualifiedNames "semantic-property" Evidence.PropertyStatus.constructorClassifiers ++
  qualifiedNames "strict-query" QueryStatus.constructorClassifiers

example : semanticStageConstructorRows.filter (fun row => row.2 == "unknown") = [
    ("observation", "unknown"),
    ("implementation-link", "unknown"),
    ("semantic-property", "unknown")
  ] := by
  native_decide

example :
    ImplementationLinkStatus.stageNotRunMarker = {
      id := "implementation-link.not-evaluated"
      owner := "Implementation Link"
      name := "not-evaluated"
      description := "The optional Implementation Link stage was not evaluated."
    } ∧
    (OutcomeConstructorClassifiers.names ImplementationLinkStatus.constructorClassifiers).contains
        ImplementationLinkStatus.stageNotRunMarker.name = false := by
  native_decide

end Umpire.InventoryTests.SemanticStages
