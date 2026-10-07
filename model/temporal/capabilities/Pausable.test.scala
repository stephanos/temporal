package umpire

import temporal.capabilities.{Pausable, Pollable}

object PausingFixture:
  enum Phase derives Finite:
    case waiting extends Phase, Waiting
    case paused extends Phase, Suspended
    case pausedWhileHeld extends Phase, Held
    case started extends Phase, Held
    case done extends Phase, Succeeded

  enum Answer derives Finite:
    case accepted

  given Ok[Answer] = Ok(Answer.accepted)

  final case class Snapshot(phase: Phase) derives Finite

  object user extends Actor:
    val pause = action(this)
    val unpause = action(this)

  object worker extends Actor:
    val poll = action(this)

  object Record extends Machine[Snapshot, Answer, Nothing], Phased[Snapshot, Phase](_.phase):
    val init = Snapshot(Phase.waiting)
    object effects:
      def keep(s: State) = stay[Snapshot, Answer, Nothing](s)
    object rules extends Rules:
      on(user.pause, user.unpause, worker.poll)(always ~> effects.keep)
    val pausable: Pausable[Snapshot, Phase] = Pausable(pause = user.pause, unpause = user.unpause)
    val pollable: Pollable[Snapshot, Phase] = Pollable(dispatch = worker.poll)
    val notDispatched = Pausable.pausedIsNotDispatched(this)

  object DerivedRecord extends Derived(Record.unmonitored):
    val notDispatched = Pausable.pausedIsNotDispatched(this)

class PausableSuite extends munit.FunSuite:
  import PausingFixture.*

  private def holds(property: Property[Snapshot], before: Phase, after: Phase): Boolean =
    val predicate = property.decl.holds2.get
      .asInstanceOf[
        (Snapshot, Step[Snapshot, Answer, Nothing]) => Boolean
      ] // scalafix:ok DisableSyntax.asInstanceOf
    predicate(Snapshot(before), Step(Answer.accepted, Snapshot(after)))

  test("an abstract phase cannot declare Pausable or Pollable without runtime type witnesses"):
    val pausableErrors = scala.compiletime.testing.typeCheckErrors("""
      import umpire.*
      import temporal.capabilities.Pausable
      def pausing[S, P](pause: ClassRef, unpause: ClassRef)(using Phasing[S, P], Finite[P]) =
        Pausable[S, P](pause, unpause)
    """)
    val pollableErrors = scala.compiletime.testing.typeCheckErrors("""
      import umpire.*
      import temporal.capabilities.Pollable
      def polling[S, P](dispatch: ClassRef)(using Phasing[S, P], Finite[P]) =
        Pollable[S, P](dispatch)
    """)
    assert(pausableErrors.exists(_.message.contains("ClassTag")), pausableErrors.toString)
    assert(pollableErrors.exists(_.message.contains("ClassTag")), pollableErrors.toString)

  test("Pausable and Pollable read only Suspended and Held through Phased"):
    for property <- Seq(Record.notDispatched, DerivedRecord.notDispatched) do
      assert(holds(property, Phase.waiting, Phase.started))
      assert(holds(property, Phase.paused, Phase.waiting))
      assert(!holds(property, Phase.paused, Phase.started))
      assert(!holds(property, Phase.paused, Phase.pausedWhileHeld))
      assert(holds(property, Phase.pausedWhileHeld, Phase.started))
