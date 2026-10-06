package umpire
// The object forms' runtime wiring, run as Scala: a machine object is the machine, its `rules` the
// one section it reads, lowered to one step function per action, and an overlap is refused as the
// rules object is constructed.

import scala.collection.mutable

// A dotted file name holds no top-level definitions, so the fixture Models sit in an object.
object RulesFixture:
  enum Lit derives Finite:
    case off, on, broken

  final case class Lamp(light: Lit, presses: UpTo[2]) derives Finite

  enum Said derives Finite:
    case ok, refused

  enum Knob derives Finite:
    case up, down

  given Ok[Said] = Ok(Said.ok)

  object hand extends Actor:
    val press = action(this)
    val turn = action(this).input[Knob]("knob")

  object clock:
    val tick = timer

  // What initialized, in order, so the tests read when each object is constructed.
  object Trace:
    val initialized = mutable.ArrayBuffer.empty[String]

  object Switch extends Machine[Lamp, Said, Nothing]:
    Trace.initialized += "Switch"
    val init = Lamp(Lit.off, UpTo(0))
    def end(s: Lamp) = s.light == Lit.broken
    def presses(s: Lamp): UpTo[2] = UpTo((s.presses + 1).min(2))

    object effects:
      Trace.initialized += "Switch.effects"
      def light(s: Lamp) = enter[Lamp, Said, Nothing](s.copy(light = Lit.on, presses = presses(s)))
      def dark(s: Lamp) = enter[Lamp, Said, Nothing](s.copy(light = Lit.off))
      def refuse(s: Lamp, @scala.annotation.unused k: Knob) = List(
        Step[Lamp, Said, Nothing](Said.refused, s)
      )
      def turned(s: Lamp, k: Knob) =
        enter[Lamp, Said, Nothing](s.copy(light = if k == Knob.up then Lit.on else Lit.off))
      def wear(s: Lamp) = enter[Lamp, Said, Nothing](s.copy(light = Lit.broken))

    object rules extends Rules(_.light):
      Trace.initialized += "Switch.rules"
      on(hand.press) {
        in(Lit.off) ~> effects.light
        in(Lit.on) ~> effects.dark
      }
      on(hand.turn(Knob.down))(in(Lit.on) ~> effects.dark)
      on(hand.turn(Knob.up))(in(Lit.on) ~> effects.light)
      on(hand.turn) {
        in(Lit.off) ~> effects.turned
        in(Lit.broken) ~> effects.refuse
      }
      on(clock.tick)(where(s => s.presses == 2 && s.light != Lit.broken) ~> effects.wear)

  // The switch whose press is its own effect wherever it fires, and whose tick never fires.
  object Stuck extends Derived(Switch.rebind(hand.press ~> Switch.effects.wear))

  // The switch whose press fires everywhere, under one rule.
  object Loose extends Derived(Switch.rebind(on(hand.press)(always ~> Switch.effects.wear)))

  object Overlapping extends Machine[Lamp, Said, Nothing]:
    val init = Lamp(Lit.off, UpTo(0))
    def end(s: Lamp) = true
    object rules extends Rules(_.light):
      on(hand.turn(Knob.up))(in(Lit.off, Lit.on) ~> Switch.effects.light)
      on(hand.turn)(where(_.presses == 1) ~> Switch.effects.turned)

  object Unfelt extends Machine[Lamp, Said, Nothing]:
    val init = Lamp(Lit.off, UpTo(0))
    def end(s: Lamp) = true
    object rules extends Rules:
      on(hand.press)(always ~> Switch.effects.light)
      disabled(clock.tick)

  final case class Pair(left: Lamp, right: Lamp)

  object Twins extends Composition[Pair](_.left -> Switch, _.right -> Unfelt):
    def end(s: Pair) = Switch.end(s.left)
    object syncs extends Syncs:
      sync(_.left -> hand.press, _.right -> hand.press)

  object Odd extends Composition(Twins.withMember(_.right -> Stuck))

class RulesTest extends munit.FunSuite:
  import RulesFixture.*
  type Press = Lamp => List[Step[Lamp, Said, Nothing]]
  type Turn = (Lamp, Knob) => List[Step[Lamp, Said, Nothing]]

  def step(m: Machine[Lamp, Said, Nothing], a: Action[?]): AnyRef =
    m.bindings.find(_.decl == a.decl).get.function

  def press(m: Machine[Lamp, Said, Nothing], s: Lamp): List[Lamp] =
    step(m, hand.press)
      .asInstanceOf[Press](s)
      .map(_.state) // scalafix:ok DisableSyntax.asInstanceOf

  def turn(m: Machine[Lamp, Said, Nothing], s: Lamp, k: Knob): List[Step[Lamp, Said, Nothing]] =
    step(m, hand.turn).asInstanceOf[Turn](s, k) // scalafix:ok DisableSyntax.asInstanceOf

  val off = Lamp(Lit.off, UpTo(0))
  val lit = Lamp(Lit.on, UpTo(1))

  test("a machine object is named after its object, and its sections initialize on first use") {
    assertEquals(Switch.name, "switch")
    assertEquals(Switch.init, off)
    assert(Trace.initialized.contains("Switch"))
    assert(!Trace.initialized.contains("Switch.rules"), Trace.initialized.toString)
    Switch.bindings: Unit
    assertEquals(
      Trace.initialized.toList.filter(_.startsWith("Switch")).take(2),
      List("Switch", "Switch.rules")
    )
  }

  test("each action's rules lower to one step function: the first rule that fires, else no step") {
    assertEquals(Switch.bindings.map(_.decl), List(hand.press, hand.turn, clock.tick).map(_.decl))
    assertEquals(press(Switch, off), List(Lamp(Lit.on, UpTo(1))))
    assertEquals(press(Switch, lit), List(lit.copy(light = Lit.off)))
    assertEquals(press(Switch, Lamp(Lit.broken, UpTo(0))), Nil)
    // One class's rule, or the whole action's: down darkens a lit lamp, up keeps it lit.
    assertEquals(turn(Switch, lit, Knob.down).map(_.state.light), List(Lit.off))
    assertEquals(turn(Switch, lit, Knob.up).map(_.state.light), List(Lit.on))
    assertEquals(
      turn(Switch, Lamp(Lit.broken, UpTo(0)), Knob.up).map(_.outcome),
      List(Said.refused)
    )
  }

  test("disabled binds an action no state enables") {
    assertEquals(Unfelt.bindings.map(_.decl), List(hand.press, clock.tick).map(_.decl))
    val tick =
      Unfelt.bindings.last.function.asInstanceOf[Press] // scalafix:ok DisableSyntax.asInstanceOf
    assertEquals(Finite[Lamp].values.toList.flatMap(tick), Nil)
  }

  test("a bare binding of a derivation keeps its action's rules and replaces their effect") {
    assertEquals(Stuck.name, "stuck")
    assertEquals(press(Stuck, off).map(_.light), List(Lit.broken))
    assertEquals(press(Stuck, Lamp(Lit.broken, UpTo(0))), Nil)
    assertEquals(press(Loose, Lamp(Lit.broken, UpTo(0))).map(_.light), List(Lit.broken))
    assertEquals(turn(Stuck, lit, Knob.down).map(_.state.light), List(Lit.off))
  }

  test(
    "two rules of one action class that hold in one state are refused when the rules construct"
  ) {
    // An error in an object's initializer is fatal to munit's `intercept`, so it is caught here.
    val refused =
      try
        Overlapping.bindings: Unit
        fail("the overlapping rules constructed")
      catch case e: ExceptionInInitializerError => e.getCause
    assertEquals(
      refused.getMessage,
      "overlapping fires turn-up by two rules in Lamp(off,1): rule 1, in(off, on), and rule 2, " +
        "where: the rules of one action class hold in no common state, so write alternatives as " +
        "one effect that names each with `choose`"
    )
  }

  test("a derivation's rules are checked as it binds them") {
    val refused = intercept[IllegalArgumentException](
      Switch.rebind(on(hand.press) {
        always ~> Switch.effects.wear
        where(_.light == Lit.off) ~> Switch.effects.dark
      })
    )
    assertEquals(
      refused.getMessage,
      "a derivation of switch fires press by two rules in Lamp(off,0): rule 1, always, and rule " +
        "2, where: the rules of one action class hold in no common state, so write alternatives " +
        "as one effect that names each with `choose`"
    )
  }

  test("a composition object is its members, and an IR file constructs what it reaches") {
    assertEquals(Twins.name, "twins")
    assertEquals(Twins.members, Vector(Switch, Unfelt))
    assertEquals(Odd.name, "odd")
    assertEquals(Odd.members, Vector(Switch, Unfelt, Stuck))
    // Built directly, so the IR files the gate holds model/ir to stay the Models' own.
    IrFile("rules-test", Seq(Odd, Switch)).construct(): Unit
  }

  test("a rule names its action as written") {
    assertEquals(writtenAction("example.orders.clerk.ship"), "ship")
    assertEquals(
      writtenAction("example.orders.buyer.change.apply(example.orders.Change.hold)"),
      "change"
    )
    assertEquals(writtenAction("umpire.apply[(A, B)](example.orders.buyer.place)(x := y)"), "place")
  }
