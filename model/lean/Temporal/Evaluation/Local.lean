import Umpire.Evaluation

/-!
# The local Evaluation Profile

`local-ephemeral` assesses one closed Run of a Case recorded against an ephemeral local test
cluster. Its trust basis, `local-ephemeral-cluster`, is asserted by the Profile and not checked
against the recorded Driver identity. A violated claim is a failed claim, so a violated Verdict and
a Run its Monitor stopped reject. Absent verification is never proof and never a failure either,
so an inconclusive Verdict, an incomplete Run, an unclosed cleanup, a `capability` or
`interpretation` Known Gap, and a rule with no supporting event leave the subject incomplete.
-/

namespace Temporal.Evaluation.Local

open Umpire Umpire.Evaluation

/-- The authored `local-ephemeral` Profile, in precedence order. -/
def localEphemeralDeclaration : Declaration := {
  name := "local-ephemeral"
  claim := "The Case's Contract held for one closed Run of the Case against an ephemeral local test cluster, under the recorded Driver identity."
  trust := "local-ephemeral-cluster"
  blockingGaps := [.capability, .interpretation]
  reasons := [
    { name := "verdict-violated", condition := .verdictViolated, decision := .rejected },
    { name := "monitor-stopped", condition := .dispositionStopped, decision := .rejected },
    { name := "verdict-inconclusive", condition := .verdictInconclusive, decision := .incomplete },
    { name := "run-incomplete", condition := .dispositionIncomplete, decision := .incomplete },
    { name := "cleanup-unclosed", condition := .cleanupUnclosed, decision := .incomplete },
    { name := "known-gap-blocking", condition := .knownGapBlocking, decision := .incomplete },
    { name := "rule-unsupported", condition := .unsupportedRule, decision := .incomplete }
  ]
}

/-- The checked `local-ephemeral` Profile, or why it does not declare. -/
def localEphemeral : Except ProfileError Profile := Profile.declare localEphemeralDeclaration

/-- Every Profile this model declares, rendered by `umpire-evaluation-profiles`. -/
def declared : List (Except ProfileError Profile) := [localEphemeral]

end Temporal.Evaluation.Local
