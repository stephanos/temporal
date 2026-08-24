import NexusAutoClose
import Temporal.Experiment.Compiler

namespace Temporal.Experiment.NexusCallerClosure

open NexusAutoClose

def targetId : ModelId := ⟨"nexus-caller-closure"⟩
def regressionId : RegressionId := ⟨"nexus-caller-closure-upgrade"⟩
def clashResourceId : ResourceId := ⟨"caller-closure-clash"⟩
def forceCloseActionId : ActionId := ⟨"caller-force-close"⟩
def honoredDeliveryPropertyId : PropertyId := ⟨"honored-delivery"⟩
def cancellationUniquenessPropertyId : PropertyId := ⟨"cancellation-uniqueness"⟩

def targetDeclaration : String :=
  "NexusAutoClose.wClash|NexusAutoClose.autoClose:upgrade"

def clashSetupValue : String := reprStr wClash

def upgradedOutcomeValue : String := reprStr (autoClose .upgrade wClash)

def clashSetup : ResolvedSetup := ⟨[
  { id := clashResourceId, value := clashSetupValue }
]⟩

private def checkedObservation {property : Prop}
    (id : PropertyId)
    (contract : String)
    (_ : property) : PropertyObservation :=
  { id, contract }

def honoredDeliveryObservation : PropertyObservation :=
  checkedObservation honoredDeliveryPropertyId
    "NexusAutoClose.upgrade_honors_delivery(NexusAutoClose.wClash)"
    (upgrade_honors_delivery wClash)

def cancellationUniquenessObservation : PropertyObservation :=
  checkedObservation cancellationUniquenessPropertyId
    ("NexusAutoClose.upgrade_preserves_uniqueness" ++
      "(NexusAutoClose.wClash,NexusAutoClose.wClash_reachable(upgrade))")
    (upgrade_preserves_uniqueness wClash (wClash_reachable .upgrade))

def projectForceClose (setup : ResolvedSetup) : Option ModelOutcome :=
  if setup == clashSetup then
    some ⟨upgradedOutcomeValue⟩
  else
    none

def target : ModelTarget := {
  id := targetId
  declaration := targetDeclaration
  resources := [{ id := clashResourceId, value := clashSetupValue }]
  actionProjections := [{ id := forceCloseActionId, project := projectForceClose }]
  propertyObservations := [
    honoredDeliveryObservation,
    cancellationUniquenessObservation
  ]
  provenance := { source := "NexusAutoClose", compiler := "lean-regression" }
}

def regression : Regression := {
  id := regressionId
  target := targetId
  resources := [clashResourceId]
  actionAttempts := [forceCloseActionId]
  ordering := []
  expectedProperties := ⟨[
    honoredDeliveryPropertyId,
    cancellationUniquenessPropertyId
  ]⟩
  bounds := { resources := 1, actions := 1, precedenceEdges := 0 }
  omissions := ["runtime-execution", "state-exploration"]
}

def compiled : Except CompileError ExperimentSpec := compile target regression

end Temporal.Experiment.NexusCallerClosure
