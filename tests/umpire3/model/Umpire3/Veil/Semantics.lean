import Umpire3.FirstOrderView

namespace Umpire3.Veil

structure SemanticRelation (artifact : FirstOrderArtifact) where
  State : Type
  Label : Type
  initial : State → Prop
  next : State → Label → State → Prop
  property : State → Prop
  encodeState : State → FirstOrderState
  encodeLabel : Label → String
  initial_iff : ∀ state,
    initial state ↔ artifact.initial.eval (encodeState state) = true
  next_iff : ∀ state label nextState,
    next state label nextState ↔
      artifact.next (encodeState state) (encodeLabel label) = some (encodeState nextState)
  property_iff : ∀ state,
    property state ↔ artifact.invariant.eval (encodeState state) = true
  state_injective : Function.Injective encodeState
  label_injective : Function.Injective encodeLabel
  label_total : ∀ label, encodeLabel label ∈ artifact.actionIdentifiers
  label_complete : ∀ identifier, identifier ∈ artifact.actionIdentifiers →
    ∃ label, encodeLabel label = identifier

structure SemanticBinding (artifact : FirstOrderArtifact) where
  symbolic : SemanticRelation artifact
  concrete : SemanticRelation artifact

end Umpire3.Veil
