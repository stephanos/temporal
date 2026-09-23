import Umpire.Evaluation

/-!
# The canary Evaluation Profiles

`production-canary` assesses one serial Run of the pinned canary Case against the dedicated
production-canary namespace and route. Its trust basis, `dedicated-production-canary`, is asserted
by the Profile; the canary's own provenance says how the Run was authorized, fenced and cleaned up.
It blocks on every Known Gap kind, and its table is `local-ephemeral`'s except that a rule the
Verdict names at a terminal state with no supporting event rejects: in production, a claim nothing
supports is a failed claim, not a missing one.

`canary-harness` is the same policy under the trust basis `test-cluster-harness`. Only the
canary's harness build embeds it, so a harness receipt is never a production receipt.

Neither Profile carries a coordinate, a credential or a canary policy value: those stay under
`tools/canary`.
-/

namespace Temporal.Evaluation.Canary

open Umpire Umpire.Evaluation

/-- The reason table both canary Profiles share, in `local-ephemeral`'s order. -/
def canaryReasons : List Reason := [
  { name := "verdict-violated", condition := .verdictViolated, decision := .rejected },
  { name := "monitor-stopped", condition := .dispositionStopped, decision := .rejected },
  { name := "verdict-inconclusive", condition := .verdictInconclusive, decision := .incomplete },
  { name := "run-incomplete", condition := .dispositionIncomplete, decision := .incomplete },
  { name := "cleanup-unclosed", condition := .cleanupUnclosed, decision := .incomplete },
  { name := "known-gap-blocking", condition := .knownGapBlocking, decision := .incomplete },
  { name := "rule-unsupported", condition := .unsupportedRule, decision := .rejected }
]

/-- Every Known Gap kind blocks a canary. -/
def canaryBlockingGaps : List KnownGapKind := [.capability, .input, .interpretation, .claim]

/-- The authored `production-canary` Profile. -/
def productionCanaryDeclaration : Declaration := {
  name := "production-canary"
  claim := "The Case's Contract held for one serial Run of the pinned canary Case against the dedicated production-canary namespace and route, under the recorded Driver identity."
  trust := "dedicated-production-canary"
  blockingGaps := canaryBlockingGaps
  reasons := canaryReasons
}

/-- The authored `canary-harness` Profile: the production policy under the test cluster's trust. -/
def canaryHarnessDeclaration : Declaration := {
  name := "canary-harness"
  claim := "The Case's Contract held for one serial Run of the pinned canary Case against a test cluster standing in for the production-canary namespace and route."
  trust := "test-cluster-harness"
  blockingGaps := canaryBlockingGaps
  reasons := canaryReasons
}

/-- The checked `production-canary` Profile, or why it does not declare. -/
def productionCanary : Except ProfileError Profile := Profile.declare productionCanaryDeclaration

/-- The checked `canary-harness` Profile, or why it does not declare. -/
def canaryHarness : Except ProfileError Profile := Profile.declare canaryHarnessDeclaration

end Temporal.Evaluation.Canary
