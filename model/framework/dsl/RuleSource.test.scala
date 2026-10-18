package framework.dsl

import framework.checker.Effects.*
import RuleSourceFixture.*

object RuleSourceFixture:
  enum Knob:
    case up, down
  enum Said:
    case ok
  given Ok[Said] = Ok(Said.ok)

  object hand extends Actor:
    val press = action
    val turn = action.input[Knob](name = "knob")

  object clock extends Actor:
    val tick = action
  abstract class FixtureMachine extends Machine[Said, Nothing]:
    var ready = false

    object effects:
      def wear(): Unit =
        ready = !ready

  object Switch extends FixtureMachine:

    object transitions extends Transitions:
      on(hand.press):
        effects.wear()
      on(hand.turn): _ =>
        effects.wear()
      on(clock.tick):
        effects.wear()

class RuleSourceTest extends munit.FunSuite:
  for inFrom <- List(false, true); arity <- 1 to 5 do
    test(s"imported bare target form $arity refuses with from=$inFrom"):
      import hand.{press, turn}
      import clock.tick
      object Bare extends FixtureMachine:
        object transitions extends Transitions:
          def body(): Unit = arity match
            case 1 =>
              on(press):
                effects.wear()
            case 2 =>
              on(turn(Knob.up)):
                effects.wear()
            case 3 =>
              on(hand.press, turn(Knob.up)):
                effects.wear()
            case 4 =>
              on(hand.press, hand.turn(Knob.up), turn(Knob.down)):
                effects.wear()
            case _ =>
              on(hand.press, hand.turn(Knob.up), hand.turn(Knob.down), tick):
                effects.wear()
          if inFrom then from(hand)(body()) else body()
      val refused = intercept[IllegalArgumentException](Bare.bindings)
      val action = arity match
        case 1 => "press"
        case 5 => "tick"
        case _ => "turn"
      assert(refused.getMessage.contains(s"on($action) names its action bare"), refused.getMessage)
      val declarer = if arity == 5 then "clock" else "hand"
      assert(refused.getMessage.contains(s"on($declarer.$action)`"), refused.getMessage)
      assert(
        refused.getMessage.matches("(?s).*RuleSource\\.test\\.scala:[0-9]+:[0-9]+:.*"),
        refused.getMessage
      )

  test("qualified two-to-four targets retain native ownership and binding order"):
    object buttons:
      val actor = Actor("buttons")
      val first = action(actor)
      val second = action(actor)
      val third = action(actor)
      val fourth = action(actor)
    for inFrom <- List(false, true); arity <- 2 to 4 do
      object Qualified extends FixtureMachine:
        object transitions extends Transitions:
          def body(): Unit = arity match
            case 2 =>
              on(buttons.first, buttons.second):
                effects.wear()
            case 3 =>
              on(buttons.first, buttons.second, buttons.third):
                effects.wear()
            case _ =>
              on(buttons.first, buttons.second, buttons.third, buttons.fourth):
                effects.wear()
          if inFrom then from(buttons)(body()) else body()
      assertEquals(
        Qualified.bindings.map(_.decl),
        List(buttons.first, buttons.second, buttons.third, buttons.fourth).take(arity).map(_.decl)
      )

  test("a from import refuses even with an already qualified target"):
    for wildcard <- List(false, true) do
      object Imported extends FixtureMachine:
        object transitions extends Transitions:
          if wildcard then
            from(hand):
              import hand.*
              on(press):
                effects.wear()
          else
            from(hand):
              import hand.press
              val _ = press
              on(hand.press):
                effects.wear()
      val refused = intercept[IllegalArgumentException](Imported.bindings)
      assert(refused.getMessage.contains("from(hand) imports members at RuleSource.test.scala:"))
      assert(refused.getMessage.contains("a from only groups"))
      assert(refused.getMessage.contains("on(hand.<action>)"))

  test("bare disabled and derived on targets refuse natively"):
    import clock.tick
    import hand.press
    for inFrom <- List(false, true) do
      object Off extends FixtureMachine:
        object transitions extends Transitions:
          if inFrom then from(clock)(disabled(tick)) else disabled(tick)
      assert(
        intercept[IllegalArgumentException](Off.bindings).getMessage
          .contains("disabled(tick) names its action bare")
      )
    object Rebound
        extends Derived(
          Switch.rebind(
            on(press):
              Switch.effects.wear()
          )
        )
    val refused = intercept[IllegalArgumentException](Rebound.bindings)
    assert(refused.getMessage.contains("on(press) names its action bare"))

  test("class argument and comment dots never qualify the bare action"):
    for source <- List("turn(Knob.up)", "/* hand.press */ press", "press // hand.press") do
      val refused = intercept[IllegalArgumentException](
        RuleSource.validate(source, "control.scala:1:1", "on", "hand")
      )
      assert(refused.getMessage.contains("names its action bare at control.scala:1:1"))
