package framework.dsl

import framework.checker.Effects.*
import framework.dsl.Finite.given

// A derivation of a Machine that declares its state as vars binds blocks written as its
// `transitions` are: plain control flow whose leaves call an effect, reject or are `()`.
object VarDerivationsFixture:
  enum Said:
    case ok, refused

  enum Coin:
    case heads, tails

  given Ok[Said] = Ok(Said.ok)

  object player extends Actor:
    val toss = action
    val pay = action.input[UpTo[3]](name = "amount")
    val flip = action.input[Coin](name = "coin")
    val bonus = action

  object Game extends Machine[Said, Nothing]:
    var score = 0
    var tossed = false

    object transitions extends Transitions:
      on(player.toss):
        if !tossed then effects.toss()
      on(player.pay): amount =>
        if amount <= score then effects.pay(amount) else reject(Said.refused)
      on(player.flip(Coin.heads)):
        if tossed then effects.win()
      on(player.flip(Coin.tails)):
        if tossed then effects.lose()

    object effects:
      def toss() =
        tossed = true
      def pay(amount: UpTo[3]) =
        score = score - amount
      def payTwice(amount: UpTo[3]) =
        score = score - 2 * amount
      def win() =
        score = score + 1
      def lose() =
        score = score - 1
      def jackpot() =
        score = score + 10

  object Generous
      extends Derived(
        Game.rebind(
          on(player.toss):
            if Game.score < 2 then Game.effects.toss()
          ,
          on(player.pay): amount =>
            Game.effects.payTwice(amount)
        )
      )

  object Bonus
      extends Derived(
        Game.extend(
          on(player.bonus):
            if Game.tossed then Game.effects.jackpot()
        )
      )

class VarDerivationsSuite extends munit.FunSuite:
  import VarDerivationsFixture.*

  private def run(machine: RecordMachine[State, Said, Nothing], state: State, cls: Class) =
    val binding = machine.bindings.find(_.decl eq cls.decl).get
    effectOf[State, Said, Nothing](binding.decl, binding.function)(state, cls)

  private def at(score: Int, tossed: Boolean): State =
    Game.init.copy(score = score, tossed = tossed)

  test("a rebound block of a Machine's vars fires its effect where its control flow reaches it"):
    assertEquals(run(Generous, at(0, false), player.toss()), List(Step(Said.ok, at(0, true))))
    assertEquals(run(Generous, at(2, false), player.toss()), Nil)
    assertEquals(run(Game, at(2, false), player.toss()), List(Step(Said.ok, at(2, true))))

  test("a rebound block with an input reads it as its parameter"):
    assertEquals(
      run(Generous, at(5, false), player.pay(UpTo[3](2))),
      List(Step(Said.ok, at(1, false)))
    )
    assertEquals(run(Game, at(5, false), player.pay(UpTo[3](2))), List(Step(Said.ok, at(3, false))))

  test("an extended block binds an action its source does not"):
    assertEquals(run(Bonus, at(0, true), player.bonus()), List(Step(Said.ok, at(10, true))))
    assertEquals(run(Bonus, at(0, false), player.bonus()), Nil)
    assertEquals(Game.bindings.exists(_.decl eq player.bonus.decl), false)

  test("a derivation's block refuses an assignment and a leaf that is no effect call"):
    val assigned = compileErrors(
      """Game.rebind(on(player.toss) { if Game.tossed then Game.score = 3 })"""
    )
    assert(assigned.contains("a transition says when; write the state change in effects"), assigned)
    val printed = compileErrors(
      """Game.rebind(on(player.toss) { if Game.tossed then println("x") })"""
    )
    assert(printed.contains("a transition ends in an effect call, a rejection or ()"), printed)
