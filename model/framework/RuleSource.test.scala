package framework

import RulesFixture.*

class RuleSourceTest extends munit.FunSuite:
  for grouped <- List(false, true); arity <- 1 to 5 do
    test(s"imported bare target form $arity refuses with from=$grouped"):
      import hand.{press, turn}
      object Bare extends Machine[Lamp, Said, Nothing]:
        val init = Lamp(Lit.off, UpTo(0))
        def end(s: Lamp) = true
        object rules extends Rules:
          def body(): Unit = arity match
            case 1 => on(press)(always ~> Switch.effects.wear)
            case 2 => on(turn(Knob.up))(always ~> Switch.effects.wear)
            case 3 => on(press, hand.turn(Knob.up))(always ~> Switch.effects.wear)
            case 4 => on(press, hand.turn(Knob.up), hand.turn(Knob.down))(always ~> Switch.effects.wear)
            case _ => on(press, hand.turn(Knob.up), hand.turn(Knob.down), clock.tick)(always ~> Switch.effects.wear)
          if grouped then from(hand) { body() } else body()
      val refused = intercept[IllegalArgumentException](Bare.bindings)
      val action = if arity == 2 then "turn" else "press"
      assert(refused.getMessage.contains(s"on($action) names its action bare"), refused.getMessage)
      assert(refused.getMessage.contains(s".$action)`"), refused.getMessage)
