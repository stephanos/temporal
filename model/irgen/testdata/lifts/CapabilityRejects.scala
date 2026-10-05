// Capability declarations the lifter refuses, each at its line (fn-122.2, fn-122.5): an action the
// machine does not bind, a lambda for a function-valued field, two capabilities of one kind in one
// declaration or in two, a waiver of a law the catalog does not bring, a waiver with no reason, an
// overriding def with other parameters than the law's, a function-valued argument of a law that is
// a lambda, a `cited` with no citation or a computed one, a law citing a parameter it lacks, and a
// `through` whose selector is no field path or whose read is a lambda (fn-127.2).
package fixture.capabilityrejects

import umpire.*
import temporal.capabilities.{closedIsRejectedUniformly, terminalStatesAreFinal}
import temporal.capabilities.{given, *}
import fixture.capabilities.{job, kill, poll, three, Answer, Job, Jobs, Note, Phase}
import fixture.capabilities.{pair, pause, resume}

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

/** A binding whose companion is no capability kind, which no catalog keys a law by. */
final case class Unkinded[S](running: S => Boolean) extends CapabilityOf[S, Nothing, Nothing]

val unkinded = capabilities(job, limits = three)(Unkinded(running = Jobs.running))

/** Another kit's kind that shares Temporal's name, which Temporal's catalog brings no law. */
object otherKit:
  final case class Closable[S, P, O](status: S => P, terminal: P => Boolean, rejected: O)
      extends CapabilityOf[S, O, Nothing]
  object Closable extends CapabilityKind

val sameName = capabilities(job, limits = three)(
  otherKit.Closable(status = Jobs.phase, terminal = Jobs.terminal, rejected = Answer.gone)
).except(terminalStatesAreFinal, because = "another kit's Closable brings no Temporal law")

// A binding `cited` with no citation, and one whose citation is computed.
val uncited = capabilities(job, limits = three)(
  Closable(status = Jobs.phase, terminal = Jobs.terminal, rejected = cited(Answer.gone))
)

val computedCitation = capabilities(job, limits = three)(
  Closable(
    status = Jobs.phase,
    terminal = Jobs.terminal,
    rejected = cited(Answer.gone, Answer.gone.toString)
  )
)

/** A law that names among its cited parameters one its apply does not take. */
object citesNoParameter
    extends Law(
      cites = Seq("model/irgen/testdata/lifts/CapabilityRejects.scala"),
      promises = "a closed job keeps its status",
      doesNotPromise = "anything a fixture does not need",
      parameters = Seq("terminal", "rejected")
    ):
  def apply[S, P](m: Declares[S])(status: S => P, terminal: P => Boolean): Property[S] =
    m.property.once(s => terminal(status(s))).keeps(status)

val unknownParameter = capabilities(job, limits = three)(
  Closable(status = Jobs.phase, terminal = Jobs.terminal, rejected = Answer.gone)
)(using Catalog.single(Closable)(citesNoParameter))

// A member read with `through` by a computed selector, and with a lambda for its def.
val throughComputed = capabilities(pair, limits = three)(
  Pollable(
    dispatch = pair.own(_.left, poll),
    running = through(_.left.copy(phase = Phase.queued), Jobs.running)
  )
)

val throughLambda = capabilities(pair, limits = three)(
  Pausable(
    pause = pair.own(_.left, pause),
    unpause = pair.own(_.left, resume),
    paused = through(_.left, j => j.phase == Phase.paused)
  )
)
