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
      "sha256:1272a5861a0a491cca53cd228dc8d72c1bcc970fd0c77b8eeee4eadcb3f2641e",
      "sha256:873abde394db383b2a9a750c6ecddebe1dedcf20faaea9ed75c9e623f3bca4cb",
      "sha256:ac24d6bac3eb7ac03b1bee26374c8a84bc06b211163d5c3f0169f7e2a6aa9ed4",
      "sha256:b38b568baaaea2bf7e1d9aa6ca1b24e26f161c65c3fa3234e4f52ab741efc720"
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
