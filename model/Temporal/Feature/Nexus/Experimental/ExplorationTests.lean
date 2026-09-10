import Temporal.Feature.Nexus.Experimental.Exploration

namespace Temporal.Feature.Nexus.Experimental.ExplorationTests

open Umpire
open Temporal.Feature.Nexus.Experimental.Exploration

private def exhaustiveResult : ExplorationResult :=
  (run .exhaustive 4).toOption.get (by native_decide)

private def exhaustiveCandidates : List ExplorationCandidate :=
  exhaustiveResult.exploratory

private def firstCandidate : ExplorationCandidate :=
  exhaustiveCandidates.head?.get (by native_decide)

private def secondCandidate : ExplorationCandidate :=
  (exhaustiveCandidates.drop 1).head?.get (by native_decide)

/-!
The checked four-point Nexus Space has one stable exhaustive identity order, while coordinate
guidance selects from that same universe and reports its bounded outcome separately.
-/
example :
    exhaustiveCandidates.map (ArtifactChecksum.render ∘ ExplorationCandidate.identity) == [
      "sha256:08ca605e2d5b556230d7593b9a7e088b00970f153e9145963f0873983b5b634b",
      "sha256:0beae8011517812fbf284afdbe70eb6b329b338f3efcc52a7ba20fab9c9de7b8",
      "sha256:23b6bac41bfee2637d1ba16737c50c9f3df83ba5c520b1c9e72bb6257c1e2bbc",
      "sha256:868d5978e85783eabb8af534b375be1d50c22f44456355898e8a3949d12f3607"
    ] &&
      exhaustiveResult.completion == .exhausted &&
      (run (.uncoveredCoordinate (.fact 1 1)) 1).toOption.any (fun result =>
        result.exploratory.map ExplorationCandidate.identity == [firstCandidate.identity] &&
          result.coordinateOutcome == some .coordinateSelected &&
          result.completion == .limitReached) = true := by
  native_decide

/-!
Pinned candidates precede and disappear from the exploratory partition without consuming its
Limit; only the complete eligible partition reports exhaustion.
-/
example :
    let pinned := firstCandidate.experimentSpec
    let limited := (run .exhaustive 2).toOption
    let retained := (run .exhaustive 3 [pinned]).toOption
    limited.any (fun result =>
        result.pinned.isEmpty && result.exploratory.length == 2 &&
          result.completion == .limitReached) &&
      retained.any (fun result =>
        result.pinned.map (fun candidate => candidate.experimentSpec.artifactChecksum) ==
            [firstCandidate.identity] &&
          result.exploratory.length == 3 &&
          !(result.exploratory.map ExplorationCandidate.identity).contains
            firstCandidate.identity &&
          result.omissions == [{
            identity := firstCandidate.identity
            reason := .pinnedPrecedence
          }] &&
          result.completion == .exhausted) = true := by
  native_decide

private def session : ExplorationSession :=
  (startSession .exhaustive 4).toOption.get (by native_decide)

private def firstStep := session.next.get (by native_decide)

private def firstOutstanding : ExplorationSession := firstStep.2

private def firstBinding := firstCandidate.experimentSpec.artifactBinding

private def secondBinding := secondCandidate.experimentSpec.artifactBinding

/-!
The Nexus session preserves its fixed order, permits only one outstanding candidate, and advances
only after the exact checked binding; crossed and stale observations remain atomic failures.
-/
example :
    let stale := { firstBinding with formatVersion := "unsupported-format" }
    let afterFirst := firstOutstanding.observe [firstBinding]
    firstStep.1.identity == firstCandidate.identity &&
      firstOutstanding.next.isNone &&
      (firstOutstanding.observe [secondBinding]).isNone &&
      (firstOutstanding.observe [stale]).isNone &&
      (afterFirst.bind ExplorationSession.next |>.any (fun step =>
        step.1.identity == secondCandidate.identity)) = true := by
  native_decide

end Temporal.Feature.Nexus.Experimental.ExplorationTests
