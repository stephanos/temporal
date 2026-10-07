package umpire

import temporal.capabilities.Closable

object ClosingFixture:
  enum Phase derives Finite:
    case open extends Phase, Waiting
    case done extends Phase, Succeeded
    case failed extends Phase, Failed

  enum Answer derives Finite:
    case accepted, rejected

  final case class Snapshot(phase: Phase) derives Finite

  object Record extends Machine[Snapshot, Answer, Nothing], Phased[Snapshot, Phase](_.phase):
    val init = Snapshot(Phase.open)
    object rules extends Rules
    val closing = Closable(rejected = Answer.rejected)
    val finality = Closable.terminalStatesAreFinal(this)
    val rejection = Closable.closedIsRejectedUniformly(this)(Answer.rejected)

class ClosableSuite extends munit.FunSuite:
  import ClosingFixture.*

  test("an abstract phase cannot declare Closable without its runtime type witness"):
    val errors = scala.compiletime.testing.typeCheckErrors("""
      import umpire.*
      import temporal.capabilities.Closable
      def closing[S, P](using Phasing[S, P], Finite[P]) = Closable[S, P, String]("rejected")
    """)
    assert(errors.exists(_.message.contains("ClassTag")), errors.toString)

  test("Closable reads Closed from Phased, preserving the phase and rejecting mutations"):
    val finality = Record.finality.decl.holds2.get
      .asInstanceOf[
        (Snapshot, Step[Snapshot, Answer, Nothing]) => Boolean
      ] // scalafix:ok DisableSyntax.asInstanceOf
    val rejection = Record.rejection.decl.holds2.get
      .asInstanceOf[
        (Snapshot, Step[Snapshot, Answer, Nothing]) => Boolean
      ] // scalafix:ok DisableSyntax.asInstanceOf
    val open = Snapshot(Phase.open)
    val done = Snapshot(Phase.done)
    val failed = Snapshot(Phase.failed)
    assert(finality(open, Step(Answer.accepted, done)))
    assert(finality(done, Step(Answer.rejected, done)))
    assert(!finality(done, Step(Answer.rejected, failed)))
    assert(rejection(open, Step(Answer.accepted, done)))
    assert(rejection(done, Step(Answer.rejected, done)))
    assert(!rejection(done, Step(Answer.accepted, done)))
