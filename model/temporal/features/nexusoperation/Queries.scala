/* The bounds and run expectations of the standalone Nexus operation's Queries, which its
 * capabilities generate.
 */
package temporal
package features.nexusoperation

import umpire.*
import umpire.realize.{Conformance, Outcome as RunOutcome, RunExpectation}

val three = Limits(steps = 3, actions = 3, search = 4096)

/**
 * A Run explains an unobserved control of a closed operation too, which records nothing, so the
 * claim's explanations disagree.
 */
val explanationsDisagree = "the executions that explain the evidence disagree"
def inconclusive(reason: String): RunExpectation =
  RunExpectation(Conformance.conformant, RunOutcome.inconclusive, reason)
