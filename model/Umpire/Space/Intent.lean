import Umpire.Artifact
import Umpire.Space.Language

/-! Checked projection from selected Space semantics into the existing Artifact intent fields. -/

namespace Umpire

/-- Candidate semantic intent produced while lowering one checked Space point. -/
structure ArtifactIntentDeclaration where
  selectedChoices : List ModelValue
  selectedVariants : List RoleBinding
  requestedFaults : List ArtifactFaultIntent
  additionalCapabilityRequirementDefinitionIds : List DefinitionId
  deriving BEq, DecidableEq, Repr

private def valueLe (left right : ModelValue) : Bool :=
  decide (left.definitionId.value < right.definitionId.value) ||
    (left.definitionId == right.definitionId && decide (left.value ≤ right.value))

private def bindingLe (left right : RoleBinding) : Bool :=
  decide (left.role.value < right.role.value) ||
    (left.role == right.role && valueLe left.value right.value)

private def faultLe (left right : ArtifactFaultIntent) : Bool :=
  decide (left.definitionId.value ≤ right.definitionId.value)

private def idLe (left right : DefinitionId) : Bool :=
  decide (left.value ≤ right.value)

/-- Check and canonically bind one intent declaration to the exact Query closure it targets. -/
def checkArtifactIntent
    (query : CheckedQuery LawStatement)
    (declaration : ArtifactIntentDeclaration) : Except ArtifactIntentError ArtifactIntent := do
  let intent : ArtifactIntent := {
    queryDefinitionId := query.id
    queryBehaviorFingerprint := query.behaviorFingerprint
    behaviorDefinitionId := query.behavior.id
    behaviorFingerprint := query.behavior.behaviorFingerprint
    targetDefinitionId := query.target.id
    targetBehaviorFingerprint := query.target.behaviorFingerprint
    kernelDefinitionId := query.target.kernel.metadata.id
    kernelBehaviorFingerprint := query.target.behaviorFingerprint
    selectedChoices := declaration.selectedChoices.mergeSort valueLe
    selectedVariants := declaration.selectedVariants.mergeSort bindingLe
    requestedFaults := declaration.requestedFaults.mergeSort faultLe
    additionalCapabilityRequirementDefinitionIds :=
      (declaration.additionalCapabilityRequirementDefinitionIds ++
        declaration.requestedFaults.map ArtifactFaultIntent.capabilityDefinitionId).mergeSort idLe
  }
  intent.validateFor query
  pure {
    intent with
    additionalCapabilityRequirementDefinitionIds :=
      intent.additionalCapabilityRequirementDefinitionIds.eraseDups
  }

end Umpire
