import Temporal.Product.Nexus

namespace Umpire3.Temporal.Product.Nexus.Tests

example : product.Initial initial := rfl

example : product.Step initial .acceptCancellation cancellationAccepted := by
  simp [step, initial, cancellationAccepted]

example : product.Step cancellationAccepted .completeSuccess succeeded := by
  simp [step, cancellationAccepted, succeeded]

example : product.Step cancellationAccepted .winCancellation cancelled := by
  simp [step, cancellationAccepted, cancelled]

example : ¬product.Step cancelled .completeSuccess succeeded := by
  simp [step, cancelled, succeeded]

example : Terminal cancelled := by simp [Terminal, cancelled]

example : CancellationWon cancelled := by simp [CancellationWon, cancelled]

example {state action next} (terminal : Terminal state) (step : product.Step state action next) :
    state = next := terminal_stable terminal step

example (state action next) :
    next ∈ executable.next state action ↔ product.Step state action next :=
  executable.next_iff state action next

end Umpire3.Temporal.Product.Nexus.Tests
