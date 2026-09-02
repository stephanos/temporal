import Umpire.Exploration.Tests.Engine

/-! Process-local one-candidate Exploration session transitions. -/

namespace Umpire.ExplorationTests

open Umpire

private def sessionResult :=
  beginSession (engineRequest .exhaustive 4) engineKernel

private def session := sessionResult.toOption.get (by native_decide)

private def selectedCandidates : List ExplorationCandidate :=
  (engineRun .exhaustive 4).toOption.map ExplorationResult.exploratory |>.getD []

private def firstCandidate := selectedCandidates.head?.get (by native_decide)

private def secondCandidate := (selectedCandidates.drop 1).head?.get (by native_decide)

private def firstStep := session.next.get (by native_decide)

private def firstOutstanding := firstStep.2

private def firstBinding := firstCandidate.experimentSpec.artifactBinding

private def secondBinding := secondCandidate.experimentSpec.artifactBinding

private def afterFirst := firstOutstanding.observe [firstBinding]

private def secondStep := afterFirst.bind ExplorationSession.next

private def secondOutstanding := secondStep.map Prod.snd |>.get (by native_decide)

private def advancesFromFirst (candidate : ExplorationSession) : Bool :=
  (candidate.observe [firstBinding]).bind ExplorationSession.next |>.any fun step =>
    step.1.identity == secondCandidate.identity

private def pinnedOnlySession : ExplorationSession :=
  (beginSession (engineRequest .exhaustive 1
    (selectedCandidates.map ExplorationCandidate.experimentSpec)) engineKernel).toOption.get
      (by native_decide)

private def pinnedOverlapSession : ExplorationSession :=
  (beginSession (engineRequest .exhaustive 1 [firstCandidate.experimentSpec])
    engineKernel).toOption.get (by native_decide)

/-! `next` preserves the checked selection order and cannot overlap outstanding candidates. -/
example :
    firstStep.1.identity == firstCandidate.identity &&
      firstOutstanding.next.isNone &&
      secondStep.any (fun step => step.1.identity == secondCandidate.identity) = true := by
  native_decide

/-!
Missing, extra, duplicate, crossed, and stale bindings produce no successor and leave the original
session available for the same exact admission.
-/
example :
    let stale := { firstBinding with formatVersion := "umpire-experiment/v1" }
    ([
      ([] : List ArtifactBinding),
      [firstBinding, secondBinding],
      [firstBinding, firstBinding],
      [secondBinding],
      [stale]
    ].all fun bindings =>
      (firstOutstanding.observe bindings).isNone && advancesFromFirst firstOutstanding) = true := by
  native_decide

/-! Duplicate and stale observations remain atomic after a successful admission and advance. -/
example :
    afterFirst.any (fun admitted => (admitted.observe [firstBinding]).isNone) &&
      (secondOutstanding.observe [firstBinding]).isNone &&
      (secondOutstanding.observe [secondBinding]).isSome = true := by
  native_decide

/-! Pinned-only sessions preserve their canonical order and exact admission binding. -/
example :
    pinnedOnlySession.next.any (fun first =>
      first.1.identity == firstCandidate.identity &&
        ((first.2.observe [firstBinding]).bind ExplorationSession.next |>.any (fun second =>
          second.1.identity == secondCandidate.identity))) = true := by
  native_decide

/-! Pinned overlap is yielded first, followed by the selected non-overlapping candidate. -/
example :
    pinnedOverlapSession.next.any (fun pinned =>
      pinned.1.identity == firstCandidate.identity &&
        ((pinned.2.observe [firstBinding]).bind ExplorationSession.next |>.any (fun exploratory =>
          exploratory.1.identity == secondCandidate.identity))) = true := by
  native_decide

end Umpire.ExplorationTests
