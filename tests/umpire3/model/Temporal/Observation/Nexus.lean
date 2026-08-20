import Umpire3.Observation

namespace Umpire3.Temporal.Observation.Nexus

open Umpire3.Observation

def cancellationAccepted : Program where
  identifier := "observation.nexus.cancellation-accepted"
  observation := "cancellation-accepted"
  operation := .exists
  matchers := [{ factType := "history-event", kind := "nexus-cancellation-accepted" }]

def cancellationWon : Program where
  identifier := "observation.nexus.cancellation-won"
  observation := "cancellation-won"
  operation := .exists
  matchers := [{ factType := "history-event", kind := "nexus-cancellation-committed" }]

def staleSuccessAbsent : Program where
  identifier := "observation.nexus.stale-success-absent"
  observation := "stale-success-absent"
  operation := .absentWhenClosed
  violations := [
    {
      factType := "history-event"
      kind := "nexus-success-recorded"
      cancellationCommitted := some true
    },
    {
      factType := "history-event"
      kind := "nexus-success-recorded"
      ownerEpochRelation := some .notEqual
    },
  ]
  closures := [{
    factType := "evidence-window"
    kind := "nexus-cancellation"
    closed := some true
  }]

def source (sequence : Nat) : Source where
  identity := "history/source"
  clockDomain := "history/sequence"
  sequence := sequence
  reference := "operation/1/fact"
  entityIdentity := "operation/1"
  lineage := ["namespace/1", "operation/1"]

def cancellationFact : Fact where
  identifier := "history/cancelled"
  source := source 2
  value := .history {
    eventType := "nexus-cancellation-committed"
    eventID := 2
    operationID := some "operation/1"
  }

def cancellationAcceptedFact : Fact where
  identifier := "history/cancellation-accepted"
  source := source 1
  value := .history {
    eventType := "nexus-cancellation-accepted"
    eventID := 1
    operationID := some "operation/1"
  }

def closedWindow : Fact where
  identifier := "window/closed"
  source := source 4
  value := .window {
    purpose := "nexus-cancellation"
    closed := true
    throughSequence := 4
  }

def openWindow : Fact where
  identifier := "window/closed"
  source := source 4
  value := .window {
    purpose := "nexus-cancellation"
    closed := false
    throughSequence := 4
  }

def staleSuccess : Fact where
  identifier := "history/stale-success"
  source := source 3
  value := .history {
    eventType := "nexus-success-recorded"
    eventID := 3
    operationID := some "operation/1"
    ownerEpoch := some 1
    currentOwnerEpoch := some 2
    cancellationCommitted := some true
  }

def fixtures : List Fixture := [
  {
    identifier := "nexus-cancellation-accepted"
    observation := "cancellation-accepted"
    facts := [cancellationAcceptedFact]
    expected := { value := .true, support := ["history/cancellation-accepted"] }
  },
  {
    identifier := "nexus-cancellation-won"
    observation := "cancellation-won"
    facts := [cancellationFact]
    expected := { value := .true, support := ["history/cancelled"] }
  },
  {
    identifier := "nexus-stale-success-missing-closure"
    observation := "stale-success-absent"
    facts := [cancellationFact]
    expected := { value := .unknown }
  },
  {
    identifier := "nexus-stale-success-absent"
    observation := "stale-success-absent"
    facts := [cancellationFact, closedWindow]
    expected := { value := .true, support := ["window/closed"] }
  },
  {
    identifier := "nexus-stale-success-visible"
    observation := "stale-success-absent"
    facts := [cancellationFact, staleSuccess, closedWindow]
    expected := { value := .false, support := ["history/stale-success"] }
  },
  {
    identifier := "nexus-window-conflict"
    observation := "stale-success-absent"
    facts := [closedWindow, openWindow]
    expected := { value := .conflict, support := ["window/closed"] }
  },
]

def programs : List Program := [cancellationAccepted, cancellationWon, staleSuccessAbsent]

example : cancellationWon.evaluate [cancellationFact] =
    { value := .true, support := ["history/cancelled"] } := by decide

example : staleSuccessAbsent.evaluate [cancellationFact] = { value := .unknown } := by decide

example : staleSuccessAbsent.evaluate [cancellationFact, closedWindow] =
    { value := .true, support := ["window/closed"] } := by decide

example : staleSuccessAbsent.evaluate [cancellationFact, staleSuccess, closedWindow] =
    { value := .false, support := ["history/stale-success"] } := by decide

example : staleSuccessAbsent.evaluate [closedWindow, openWindow] =
    { value := .conflict, support := ["window/closed"] } := by decide

end Umpire3.Temporal.Observation.Nexus
