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
      "sha256:34b04499cc9163e1a5abb39963d9ce215f82fe41e355967ceeb33b07419dd1be",
      "sha256:79aa2d3b71be2ea76aadb3692e5e60258e1c590c6628a8c5cc0acda4f8fae555",
      "sha256:8a7e1f0f32b24b3b30024e3c6efe06274abf1726f4bf422f46dfb4112803186d",
      "sha256:916a2cc537c6c35066ad8a147cf139e00c661612594e80a59d90e1fc585d6221"
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
    let pinned := firstCandidate.plan
    let limited := (run .exhaustive 2).toOption
    let retained := (run .exhaustive 3 [pinned]).toOption
    limited.any (fun result =>
        result.pinned.isEmpty && result.exploratory.length == 2 &&
          result.completion == .limitReached) &&
      retained.any (fun result =>
        result.pinned.map (fun candidate => candidate.plan.artifactChecksum) ==
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

private def firstBinding := firstCandidate.plan.artifactBinding

private def secondBinding := secondCandidate.plan.artifactBinding

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
