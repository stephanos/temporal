package framework.dsl

import framework.checker.Effects.*
import RulesFixture.*

class RuleSourceTest extends munit.FunSuite:
  for inFrom <- List(false, true); arity <- 1 to 5 do
    test(s"imported bare target form $arity refuses with from=$inFrom"):
      import hand.{press, turn}
      import clock.tick
      object Bare extends RecordMachine[Lamp, Said, Nothing]:
        val init = Lamp(Lit.off, UpTo(0))
        def end(s: Lamp) = true
        object transitions extends Rules:
          def body(): Unit = arity match
            case 1 =>
              on(press):
                always ~> Switch.effects.wear
            case 2 =>
              on(turn(Knob.up)):
                always ~> Switch.effects.wear
            case 3 =>
              on(hand.press, turn(Knob.up)):
                always ~> Switch.effects.wear
            case 4 =>
              on(hand.press, hand.turn(Knob.up), turn(Knob.down)):
                always ~> Switch.effects.wear
            case _ =>
              on(hand.press, hand.turn(Knob.up), hand.turn(Knob.down), tick):
                always ~> Switch.effects.wear
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
      object Qualified extends RecordMachine[Lamp, Said, Nothing]:
        val init = Lamp(Lit.off, UpTo(0))
        def end(s: Lamp) = true
        object transitions extends Rules:
          def body(): Unit = arity match
            case 2 =>
              on(buttons.first, buttons.second):
                always ~> Switch.effects.wear
            case 3 =>
              on(buttons.first, buttons.second, buttons.third):
                always ~> Switch.effects.wear
            case _ =>
              on(buttons.first, buttons.second, buttons.third, buttons.fourth):
                always ~> Switch.effects.wear
          if inFrom then from(buttons)(body()) else body()
      assertEquals(
        Qualified.bindings.map(_.decl),
        List(buttons.first, buttons.second, buttons.third, buttons.fourth).take(arity).map(_.decl)
      )

  test("a from import refuses even with an already qualified target"):
    for wildcard <- List(false, true) do
      object Imported extends RecordMachine[Lamp, Said, Nothing]:
        val init = Lamp(Lit.off, UpTo(0))
        def end(s: Lamp) = true
        object transitions extends Rules:
          if wildcard then
            from(hand):
              import hand.*
              on(press):
                always ~> Switch.effects.wear
          else
            from(hand):
              import hand.press
              val _ = press
              on(hand.press):
                always ~> Switch.effects.wear
      val refused = intercept[IllegalArgumentException](Imported.bindings)
      assert(refused.getMessage.contains("from(hand) imports members at RuleSource.test.scala:"))
      assert(refused.getMessage.contains("a from only groups"))
      assert(refused.getMessage.contains("on(hand.<action>)"))

  test("bare disabled and derived on targets refuse natively"):
    import clock.tick
    import hand.press
    for inFrom <- List(false, true) do
      object Off extends RecordMachine[Lamp, Said, Nothing]:
        val init = Lamp(Lit.off, UpTo(0))
        def end(s: Lamp) = true
        object transitions extends Rules:
          if inFrom then from(clock)(disabled(tick)) else disabled(tick)
      assert(
        intercept[IllegalArgumentException](Off.bindings).getMessage
          .contains("disabled(tick) names its action bare")
      )
    object Rebound
        extends Derived(
          Switch.rebind(
            on(press):
              always ~> Switch.effects.wear
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
