import Umpire.Target.Tests.Fixtures

/-! Successful baseline and alternate-model composition checks. -/

namespace Umpire.TargetTests

open Umpire

example : (composeTarget testTarget).isOk = true := by
  native_decide

def switchKernel : TransitionKernel Unit Bool Bool Bool Bool := {
  testKernel with
  metadata := {
    id := id "switch.kernel.transition"
    source := source "SwitchSemantic.lean"
  }
}

def switchProvider : CapabilityProvider TestLawStatement := {
  id := id "switch.provider.toggle"
  source := source "SwitchSemantic.lean"
  contract := {
    id := id "switch.capability.toggle"
    canonicalBehavior := "switch-toggle/v1"
    requiredLaws := [providerLaw]
  }
  meanings := [{
    definitionId := id "switch.action.toggle"
    kind := .action
    canonicalBehavior := "switch-action/v1"
  }]
  lawWitnesses := [witness providerLaw (by exact .inl rfl)]
}

def switchTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  id := id "switch.target.two-state"
  source := source "SwitchSemantic.lean"
  definitions := [
    metadata "switch.target.two-state" .target,
    metadata "switch.kernel.transition" .kernel,
    metadata "switch.capability.toggle" .capability,
    metadata "switch.provider.toggle" .provider,
    metadata "switch.action.toggle" .action,
    metadata "umpire.law.provider-sound" .law providerLaw.body
  ]
  requiredCapabilities := [id "switch.capability.toggle"]
  providers := [switchProvider]
  connectors := []
  resolvedSetups := [()]
  kernel := .checked switchKernel
}

-- A second model with unrelated vocabulary composes through the exact same public interface.
example : (composeTarget switchTarget).isOk = true := by
  native_decide

end Umpire.TargetTests
