import Temporal.Evaluation.Canary
import Temporal.Evaluation.Local

/-! The canary Profiles declare, keep the table the plan fixes, and keep their pinned identities:
a change to either is a change to this pin and to the rendered file the canary embeds. -/

namespace Temporal.Evaluation.CanaryTests

open Umpire.Evaluation Temporal.Evaluation.Canary

#guard productionCanary.toOption.isSome
#guard canaryHarness.toOption.isSome

#guard productionCanary.toOption.map (·.reasons.map fun reason =>
    (reason.condition.name, reason.decision.name)) == some [
  ("verdict-violated", "rejected"),
  ("disposition-stopped", "rejected"),
  ("verdict-inconclusive", "incomplete"),
  ("disposition-incomplete", "incomplete"),
  ("cleanup-unclosed", "incomplete"),
  ("known-gap-blocking", "incomplete"),
  ("unsupported-rule", "rejected")
]

#guard productionCanary.toOption.map (·.blockingGaps.map (·.name)) ==
  some ["capability", "input", "interpretation", "claim"]

-- The harness Profile is the production policy under another trust basis, never the same Profile.
#guard (productionCanary.toOption.map (·.reasons), productionCanary.toOption.map (·.blockingGaps)) ==
  (canaryHarness.toOption.map (·.reasons), canaryHarness.toOption.map (·.blockingGaps))
#guard canaryHarness.toOption.map (·.trust) == some "test-cluster-harness"
#guard productionCanary.toOption.map Profile.identity != canaryHarness.toOption.map Profile.identity
#guard productionCanary.toOption.map Profile.identity !=
  Temporal.Evaluation.Local.localEphemeral.toOption.map Profile.identity

#guard productionCanary.toOption.map Profile.identity == some "sha256:3da213bcca87cf29ab7b1e9bcb264aa5f10f850008b6a94949200e484c1944a5"
#guard canaryHarness.toOption.map Profile.identity == some "sha256:cfb6675934e5c89b50748e5d6b37c17fa7608a6bc63b1ca9108b710a654fc860"

end Temporal.Evaluation.CanaryTests
