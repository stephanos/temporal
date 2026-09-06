import Temporal.Feature.Nexus2.Authoring

/-! An ordinary transition edit changes only typed table data and is readmitted without proof or
support-code changes. Direct single-run elaboration measured 0.53s on the task checkout; editor
completion, hover, navigation, recovery, repeated-run variance, scaling, and human usability remain
unmeasured. -/

namespace Temporal.Feature.Nexus2.AuthoringEditProbe

open Umpire
open Temporal.Feature.Nexus2

def editedTable : FiniteTable Race.Setup Race.State Race.Action Race.Outcome Race.Fact := {
  Race.table with
  transitions := Race.table.transitions ++ [{
    key := "resolve-before-request"
    source := .started
    action := .resolve
    results := [Race.succeededResult]
  }]
}

private def editedTransitionFingerprintDiffers : Option Bool := do
  let original ← Race.targetResult.toOption
  let edited ← editedTable.checkModelTarget Race.identity Race.targetDefinition Race.targetComposition
    |>.toOption
  pure (edited.id == original.id &&
    (edited.kernel.steps (ModelValue.named Race.operationStateId "started")
      (ModelValue.named Race.resolveActionId "resolve")).length == 1 &&
    edited.behaviorFingerprint != original.behaviorFingerprint)

#guard editedTransitionFingerprintDiffers == some true

end Temporal.Feature.Nexus2.AuthoringEditProbe
