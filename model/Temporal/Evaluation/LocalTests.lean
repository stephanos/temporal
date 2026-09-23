import Temporal.Evaluation.Local

/-! The `local-ephemeral` Profile declares, renders the table the plan fixes, and keeps its pinned
identity: a change to the Profile is a change to this pin and to the rendered file Go embeds. -/

namespace Temporal.Evaluation.LocalTests

open Umpire.Evaluation Temporal.Evaluation.Local

#guard localEphemeral.toOption.isSome

#guard localEphemeral.toOption.map (·.reasons.map fun reason =>
    (reason.condition.name, reason.decision.name)) == some [
  ("verdict-violated", "rejected"),
  ("disposition-stopped", "rejected"),
  ("verdict-inconclusive", "incomplete"),
  ("disposition-incomplete", "incomplete"),
  ("cleanup-unclosed", "incomplete"),
  ("known-gap-blocking", "incomplete"),
  ("unsupported-rule", "incomplete")
]

#guard localEphemeral.toOption.map (·.blockingGaps.map (·.name)) ==
  some ["capability", "interpretation"]

#guard declared.map (·.toOption.map (·.name)) == [some "local-ephemeral"]

#guard localEphemeral.toOption.map Profile.identity == some "sha256:2803afa29ed404cf0a774ead9c0672de29a6d38f27fe54fd42014f0b2f78174c"

end Temporal.Evaluation.LocalTests
