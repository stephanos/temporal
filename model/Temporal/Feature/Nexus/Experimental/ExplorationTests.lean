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
      "sha256:0c7c1802a9a680e5418de51a0bad8b546cf4186c85e72468086bcc2114f867be",
      "sha256:29b441158e81bc6ed56ef4d9644c4b17dbc6887c5137f2f0ee7bb0f8c3635e35",
      "sha256:e4c43fb50dfc533e683e900472c9241a638707f1b1e91862e2f592e4ed13d3e1",
      "sha256:f96937ff66ce229d722bd1c5de5b723785058f930ebde6495779d5805f7defc6"
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
