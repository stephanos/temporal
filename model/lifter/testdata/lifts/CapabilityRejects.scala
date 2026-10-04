// Capability declarations the lifter refuses, each at its line (fn-122.2): an action the machine
// does not bind, a lambda for a function-valued field, two capabilities of one kind in one
// declaration or in two, a waiver of a
// law the catalog does not bring, a waiver with no reason, an overriding def with other parameters
// than the law's, and a function-valued argument of a law that is a lambda.
package fixture.capabilityrejects

import umpire.*
import umpire.laws.{closedIsRejectedUniformly, terminalStatesAreFinal}
import temporal.laws.{terminateSettles, given}
import fixture.capabilities.{job, kill, poll, three, Answer, Job, Jobs, Note, Phase}

given Family = Family("fixture.capabilityrejects")

/** Answers like the law, with its parameters in another order. */
def reordered[S, P](m: Declares[S])(
    terminal: P => Boolean,
    status: S => P,
    rejected: m.Outcome
): Property[S] =
  closedIsRejectedUniformly(m)(status, terminal, rejected)

// A machine that binds poll and nothing else.
val pollOnly = machine[Job, Answer, Note] {
  starts(Job(Phase.queued))
  ends(Jobs.ends)
  steps(poll ~> Jobs.poll)
}

val unboundAction = capabilities(pollOnly, limits = three)(
  Terminable(
    terminate = kill,
    settled = Note.killedNote,
    reach = Seq(poll),
    expect = fixture.capabilities.settles
  )
)

val lambdaField = capabilities(job, limits = three)(
  Closable(status = (j: Job) => j.phase, terminal = Jobs.terminal, rejected = Answer.gone)
)

val declaredTwice = capabilities(job, limits = three)(
  Pollable(dispatch = poll, running = Jobs.running),
  Pollable(dispatch = poll, running = Jobs.paused)
)

val notBrought = capabilities(job, limits = three)(
  Pollable(dispatch = poll, running = Jobs.running)
).except(terminateSettles, because = "a law no capability here brings")

val noReason = capabilities(job, limits = three)(
  Closable(status = Jobs.phase, terminal = Jobs.terminal, rejected = Answer.gone)
).except(terminalStatesAreFinal, because = " ")

val otherSignature = capabilities(job, limits = three)(
  Closable(status = Jobs.phase, terminal = Jobs.terminal, rejected = Answer.gone)
).overriding(closedIsRejectedUniformly -> reordered, because = "its parameters differ")

// A machine that declares Pollable in two declarations, the second refused.
val againJob = machine[Job, Answer, Note] {
  starts(Job(Phase.queued))
  ends(Jobs.ends)
  steps(poll ~> Jobs.poll)
}
val againFirst =
  capabilities(againJob, limits = three)(Pollable(dispatch = poll, running = Jobs.running))
val againSecond =
  capabilities(againJob, limits = three)(Pollable(dispatch = poll, running = Jobs.paused))

// A Query of a law the declaration waives, which generates no Property for it.
val waivedClaim = query find fixture.capabilities.legacyCapabilities.claim(
  terminalStatesAreFinal
) in pollOnly.scenario.actions(poll) limits three total 5

// A law called directly with a lambda for a function-valued parameter.
val lambdaArgument = terminalStatesAreFinal(job)((j: Job) => j.phase, Jobs.terminal)
val lambdaQuery =
  query verify lambdaArgument in job.scenario("lambdaAny").free limits three total 90
