import Umpire.Exploration.Tests.Engine

/-! Pinned Regression precedence and exploration-budget independence. -/

namespace Umpire.ExplorationTests

open Umpire

private def unpinnedCandidates : List ExplorationCandidate :=
  (engineRun .exhaustive 4).toOption.map ExplorationResult.exploratory |>.getD []

private def firstPinned : ExperimentSpec :=
  (unpinnedCandidates.head?.get (by native_decide)).experimentSpec

private def secondPinned : ExperimentSpec :=
  ((unpinnedCandidates.drop 1).head?.get (by native_decide)).experimentSpec

/-!
Pinned Regressions precede exploratory candidates, consume no exploration budget, and explain each
overlap through the closed pinned-precedence reason.
-/
example :
    let result := (engineRun .exhaustive 3 [firstPinned]).toOption
    result.any (fun result =>
      result.pinned.length == 1 && result.exploratory.length == 3 &&
        result.selectedIdentities.head? == some firstPinned.artifactChecksum &&
        !(result.exploratory.map ExplorationCandidate.identity).contains
          firstPinned.artifactChecksum &&
        result.omissions == [({
          identity := firstPinned.artifactChecksum
          reason := .pinnedPrecedence
        } : ExplorationOmission)] &&
        result.omissions.map (ExplorationOmissionReason.name ∘ ExplorationOmission.reason) ==
          ["pinned-precedence"] &&
        result.completion == .exhausted) := by
  native_decide

/-! Pinned selection stays outside the Limit when the exploratory partition remains partial. -/
example : (engineRun .exhaustive 1 [firstPinned]).toOption.map (fun result =>
    (result.pinned.length, result.exploratory.length, result.selectedIdentities.length,
      result.completion)) = some (1, 1, 2, .limitReached) := by
  native_decide

/-! A compatible pinned Regression outside this Space's candidate identities creates no omission. -/
example :
    let result :=
      (engineRun .exhaustive 1 [Umpire.Examples.Switch.compiledArtifact]).toOption
    result.any (fun result =>
      result.pinned.length == 1 && result.exploratory.length == 1 &&
        result.omissions.isEmpty && result.completion == .limitReached) := by
  native_decide

/-! Guided selection also removes pinned overlap before applying its exploration Limit. -/
example :
    let result :=
      (engineRun (.uncoveredCoordinate (.observation 1 1)) 1 [firstPinned]).toOption
    result.any (fun result =>
      result.pinned.length == 1 && result.exploratory.length == 1 &&
        !(result.exploratory.map ExplorationCandidate.identity).contains
          firstPinned.artifactChecksum &&
        result.coordinateOutcome == some .coordinateSelected &&
        result.completion == .limitReached) := by
  native_decide

/-! Reversed pinned input is canonicalized independently of the exploratory identity order. -/
example :
    let result := (engineRun .exhaustive 2 [secondPinned, firstPinned]).toOption
    result.any (fun result =>
      let pinnedIdentities := result.pinned.map fun pinned =>
        pinned.experimentSpec.artifactChecksum
      pinnedIdentities == pinnedIdentities.mergeSort
          (fun left right => decide (left.render ≤ right.render)) &&
        result.exploratory.length == 2 && result.omissions.length == 2 &&
        result.completion == .exhausted) := by
  native_decide

end Umpire.ExplorationTests
