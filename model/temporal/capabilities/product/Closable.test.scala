package framework.dsl

import framework.dsl.Finite.given

import temporal.capabilities.product.Closable

object ClosingFixture:
  enum Phase:
    case open extends Phase, Waiting
    case done extends Phase, Succeeded
    case failed extends Phase, Failed

  enum Answer:
    case accepted, rejected

  final case class Snapshot(phase: Phase)

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
      import framework.*
      import temporal.capabilities.product.Closable
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
    for
      before <- Finite[Snapshot].values
      after <- Finite[Snapshot].values
      answer <- Finite[Answer].values
    do
      val step = Step[Snapshot, Answer, Nothing](answer, after)
      assertEquals(
        finality(before, step),
        before.phase == Phase.open || before.phase == after.phase,
        s"$before -> $step"
      )
      assertEquals(
        rejection(before, step),
        before.phase == Phase.open || (answer == Answer.rejected && before == after),
        s"$before -> $step"
      )
