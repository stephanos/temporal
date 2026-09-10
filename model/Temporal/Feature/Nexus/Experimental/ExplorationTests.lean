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
      "sha256:7ff8dacb18fd8f55e5016ef460130a9bfbcecbc16a6ea989b79af68332253ec3",
      "sha256:818c7b0a09e697820434c5a7c327e018dcb4ed24adc2393060a98bdcf06cb584",
      "sha256:90a3b85369ec24b853900e2e220dbc009f2b3b970d0d8f69e5a223f09cde8f78",
      "sha256:e970ae25481e122ed053944ac5b9c37941a147ac46eb11efc8f522f96ddf683a"
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

private def session : CandidateCursor :=
  (startSession .exhaustive 4).toOption.get (by native_decide)

private def firstStep := session.next.get (by native_decide)

private def firstOutstanding : CandidateCursor := firstStep.2

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
      (afterFirst.bind CandidateCursor.next |>.any (fun step =>
        step.1.identity == secondCandidate.identity)) = true := by
  native_decide


end Temporal.Feature.Nexus.Experimental.ExplorationTests
