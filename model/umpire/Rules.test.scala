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

  // One lamp's rules twice: its projection named by the rules, `Rules(_.light)`, and mixed into the
  // machine, `Phased[Lamp, Lit](_.light)`, which argument-less `Rules` reads.
  object Projected extends Machine[Lamp, Said, Nothing]:
    val init = Lamp(Lit.off, UpTo(0))
    def end(s: Lamp) = true
    object states:
      def dark(l: Lit) = l != Lit.on
    object rules extends Rules(_.light):
      on(hand.press) {
        in(Lit.on) ~> Switch.effects.dark
        in(states.dark) ~> Switch.effects.light
      }
      on(hand.turn)(in(Lit.off, Lit.broken) ~> Switch.effects.refuse)

  object PhasedLamp extends Machine[Lamp, Said, Nothing], Phased[Lamp, Lit](_.light):
    val init = Lamp(Lit.off, UpTo(0))
    def end(s: Lamp) = true
    object states:
      def dark(l: Lit) = l != Lit.on
    object rules extends Rules:
      on(hand.press) {
        in(Lit.on) ~> Switch.effects.dark
        in(states.dark) ~> Switch.effects.light
      }
      on(hand.turn)(in(Lit.off, Lit.broken) ~> Switch.effects.refuse)

  object OverlappingPhased extends Machine[Lamp, Said, Nothing], Phased[Lamp, Lit](_.light):
    val init = Lamp(Lit.off, UpTo(0))
    def end(s: Lamp) = true
    object rules extends Rules:
      on(hand.turn(Knob.up))(in(Lit.off, Lit.on) ~> Switch.effects.light)
      on(hand.turn)(where(_.presses == 1) ~> Switch.effects.turned)

  // A derivation of a `Phased` machine reads its phase, with its phase type, and so does a
  // derivation of that.
  object PhasedStuck extends Derived(PhasedLamp.rebind(hand.press ~> Switch.effects.wear)):
    val phase = summon[Phasing[Lamp, Lit]]

  object PhasedStucker extends Derived(PhasedStuck.restrict(hand.press)):
    val phase = summon[Phasing[Lamp, Lit]]

  // A derived machine that mixes in a projection of its own.
  object Rephased
      extends Derived(PhasedLamp.rebind(hand.press ~> Switch.effects.wear)),
        Phased[Lamp, Lit](_.light)
  // The lamp on the shared outcomes: a press of a lit lamp is rejected with the server's reason,
  // and of a broken one as not found.
  object Guarded extends Machine[Lamp, outcomes.Outcome, Nothing]:
    import outcomes.Rejection
    val init = Lamp(Lit.off, UpTo(0))
    def end(s: Lamp) = true
    object effects:
      def light(s: Lamp) = enter[Lamp, Outcome, Nothing](s.copy(light = Lit.on))
    object rules extends Rules(_.light):
      on(hand.press) {
        in(Lit.off) ~> effects.light
        in(Lit.on) ~> rejects(Rejection.failedPrecondition).because("the lamp is lit")
        in(Lit.broken) ~> rejects(Rejection.notFound)
      }

  // The switch's sets of lights, which a case names with `when(set)`.
  object lights:
    def unlit(l: Lit) = l != Lit.on

  // One machine in the grouped forms, `from` with its import, `when` and an `on` of two classes,
  // and in the plain forms, `on` and `in`, rule for rule.
  object Grouped extends Machine[Lamp, Said, Nothing]:
    val init = Lamp(Lit.off, UpTo(0))
    def end(s: Lamp) = s.light == Lit.broken
    object rules extends Rules(_.light):
      from(hand) {
        import hand.*
        on(press) {
          when(Lit.off) ~> Switch.effects.light
          when(Lit.on).where(_.presses == 1) ~> Switch.effects.dark
        }
        on(turn(Knob.up), turn(Knob.down)) {
          when(lights.unlit) ~> Switch.effects.wear
        }
      }
      from(clock) {
        import clock.*
        on(tick)(when(Lit.on, Lit.off).where(_.presses == 2) ~> Switch.effects.wear)
      }

  object Plain extends Machine[Lamp, Said, Nothing]:
    val init = Lamp(Lit.off, UpTo(0))
    def end(s: Lamp) = s.light == Lit.broken
    object rules extends Rules(_.light):
      on(hand.press) {
        in(Lit.off) ~> Switch.effects.light
        in(Lit.on).where(_.presses == 1) ~> Switch.effects.dark
      }
      on(hand.turn(Knob.up))(in(lights.unlit) ~> Switch.effects.wear)
      on(hand.turn(Knob.down))(in(lights.unlit) ~> Switch.effects.wear)
      on(clock.tick)(in(Lit.on, Lit.off).where(_.presses == 2) ~> Switch.effects.wear)

  // One action in several blocks whose cases hold in no common state.
  object Spread extends Machine[Lamp, Said, Nothing]:
    val init = Lamp(Lit.off, UpTo(0))
    def end(s: Lamp) = true
    object rules extends Rules(_.light):
      on(hand.press)(when(Lit.off) ~> Switch.effects.light)
      on(hand.press, clock.tick)(when(Lit.broken) ~> Switch.effects.wear)
      on(hand.press)(when(Lit.on) ~> Switch.effects.dark)

  // The rules each of these declares are refused as its rules construct.
  object Nested extends Machine[Lamp, Said, Nothing]:
    val init = Lamp(Lit.off, UpTo(0))
    def end(s: Lamp) = true
    object rules extends Rules(_.light):
      on(hand.press) {
        on(clock.tick)(always ~> Switch.effects.wear)
      }

  object NestedFrom extends Machine[Lamp, Said, Nothing]:
    val init = Lamp(Lit.off, UpTo(0))
    def end(s: Lamp) = true
    object rules extends Rules(_.light):
      from(hand) {
        from(clock) {
          on(clock.tick)(always ~> Switch.effects.wear)
        }
      }

  object Foreign extends Machine[Lamp, Said, Nothing]:
    val init = Lamp(Lit.off, UpTo(0))
    def end(s: Lamp) = true
    object rules extends Rules(_.light):
      from(hand) {
        on(clock.tick)(always ~> Switch.effects.wear)
      }

  object NamedTwice extends Machine[Lamp, Said, Nothing]:
    val init = Lamp(Lit.off, UpTo(0))
    def end(s: Lamp) = true
    object rules extends Rules(_.light):
      on(hand.press, clock.tick, hand.press)(always ~> Switch.effects.wear)

  object DisabledTarget extends Machine[Lamp, Said, Nothing]:
    val init = Lamp(Lit.off, UpTo(0))
    def end(s: Lamp) = true
    object rules extends Rules(_.light):
      disabled(clock.tick)
      on(hand.press, clock.tick)(always ~> Switch.effects.wear)

  object AcrossBlocks extends Machine[Lamp, Said, Nothing]:
    val init = Lamp(Lit.off, UpTo(0))
    def end(s: Lamp) = true
    object rules extends Rules(_.light):
      on(hand.turn)(when(Lit.off) ~> Switch.effects.turned)
      on(hand.turn(Knob.down)) {
        when(Lit.on) ~> Switch.effects.dark
        when(lights.unlit).where(_.presses == 0) ~> Switch.effects.wear
      }

  final case class Pair(left: Lamp, right: Lamp)

  object Twins extends Composition[Pair](_.left -> Switch, _.right -> Unfelt):
    def end(s: Pair) = Switch.end(s.left)
    object syncs extends Syncs:
      sync(_.left -> hand.press, _.right -> hand.press)

  object Odd extends Composition(Twins.withMember(_.right -> Stuck))

  // A derived composition that mixes in a projection of its own.
  object Reprojected
      extends Composition(Twins.withMember(_.right -> Stuck)),
        Phased[Pair, Lit](_.left.light)

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

  test(
    "argument-less rules read the machine's Phased projection as Rules(projection) reads its own"
  ) {
    val knobs = Knob.values.toList
    for s <- Finite[Lamp].values do
      assertEquals(press(PhasedLamp, s), press(Projected, s), s)
      for k <- knobs do assertEquals(turn(PhasedLamp, s, k), turn(Projected, s, k), (s, k))
    assertEquals(press(PhasedLamp, off), List(Lamp(Lit.on, UpTo(1))))
    assertEquals(press(PhasedLamp, lit), List(lit.copy(light = Lit.off)))
    assertEquals(turn(PhasedLamp, off, Knob.up).map(_.outcome), List(Said.refused))
    assertEquals(turn(PhasedLamp, lit, Knob.up), Nil)
  }

  test("an overlap through the Phased projection is refused as the rules construct") {
    val refused =
      try
        OverlappingPhased.bindings: Unit
        fail("the overlapping rules constructed")
      catch case e: ExceptionInInitializerError => e.getCause
    assertEquals(
      refused.getMessage,
      "overlappingPhased fires turn-up by two rules in Lamp(off,1): rule 1, in(off, on), and " +
        "rule 2, where: the rules of one action class hold in no common state, so write " +
        "alternatives as one effect that names each with `choose`"
    )
  }

  test("in and when name no phase in the rules of a machine that is not Phased") {
    val fix =
      "mix the projection the phases are of into the machine, `Phased[State, Phase](_.phase)`"
    val listed = compileErrors(
      "object Plain extends Machine[Lamp, Said, Nothing]:\n" +
        "  val init = Lamp(Lit.off, UpTo(0))\n" +
        "  def end(s: Lamp) = true\n" +
        "  object rules extends Rules:\n" +
        "    on(hand.press)(in(Lit.on) ~> Switch.effects.dark)"
    )
    assert(listed.contains(fix), listed)
    val named = compileErrors(
      "object Plain extends Machine[Lamp, Said, Nothing]:\n" +
        "  val init = Lamp(Lit.off, UpTo(0))\n" +
        "  def end(s: Lamp) = true\n" +
        "  def dark(l: Lit) = l != Lit.on\n" +
        "  object rules extends Rules:\n" +
        "    on(hand.press)(in(dark) ~> Switch.effects.light)"
    )
    assert(named.contains(fix), named)
    val whenListed = compileErrors(
      "object Plain extends Machine[Lamp, Said, Nothing]:\n" +
        "  val init = Lamp(Lit.off, UpTo(0))\n" +
        "  def end(s: Lamp) = true\n" +
        "  object rules extends Rules:\n" +
        "    on(hand.press)(when(Lit.on) ~> Switch.effects.dark)"
    )
    assert(whenListed.contains(fix), whenListed)
    val whenNamed = compileErrors(
      "object Plain extends Machine[Lamp, Said, Nothing]:\n" +
        "  val init = Lamp(Lit.off, UpTo(0))\n" +
        "  def end(s: Lamp) = true\n" +
        "  def dark(l: Lit) = l != Lit.on\n" +
        "  object rules extends Rules:\n" +
        "    on(hand.press)(when(dark) ~> Switch.effects.light)"
    )
    assert(whenNamed.contains(fix), whenNamed)
  }

  test("a derived machine reads its source's phase projection, with its phase type") {
    assertEquals(PhasedStuck.phase.projection(lit), Lit.on)
    assertEquals(PhasedStucker.phase.projection(off), Lit.off)
  }

  test("a derived machine or composition that mixes in Phased is refused as it initializes") {
    for (derived, message) <- List(
        (() => Rephased.name, "rephased"),
        (() => Reprojected.name, "reprojected")
      )
    do
      val refused =
        try
          derived(): Unit
          fail(s"$message initialized")
        catch case e: ExceptionInInitializerError => e.getCause
      assertEquals(
        refused.getMessage,
        s"requirement failed: $message is derived and keeps its source's phase: it mixes in no " +
          "Phased of its own"
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

  test("a rejects row keeps the state and answers rejected, with its reason where one is given") {
    import outcomes.{Outcome, Rejection}
    type Answered = Lamp => List[Step[Lamp, Outcome, Nothing]]
    val pressed =
      Guarded.bindings.head.function
        .asInstanceOf[Answered] // scalafix:ok DisableSyntax.asInstanceOf
    val broken = Lamp(Lit.broken, UpTo(0))
    assertEquals(
      pressed(off),
      List(Step[Lamp, Outcome, Nothing](Outcome.accepted, lit.copy(presses = UpTo(0))))
    )
    assertEquals(
      pressed(lit),
      List(
        Step[Lamp, Outcome, Nothing](
          Outcome.rejected(Rejection.failedPrecondition),
          lit,
          Nil,
          "the lamp is lit"
        )
      )
    )
    assertEquals(
      pressed(broken),
      List(Step[Lamp, Outcome, Nothing](Outcome.rejected(Rejection.notFound), broken))
    )
  }

  // Every step of a machine, by action, state and class, as the bindings give them.
  def steps(m: Machine[Lamp, Said, Nothing]): List[(String, Lamp, List[Any], Any)] =
    for
      b <- m.bindings.toList
      s <- Finite[Lamp].values.toList
      inputs <- classesOf(b.decl)
    yield
      // scalafix:off DisableSyntax.asInstanceOf
      val result = inputs match
        case Nil       => b.function.asInstanceOf[Lamp => Any](s)
        case List(one) => b.function.asInstanceOf[(Lamp, Any) => Any](s, one)
        case _         => fail("the fixture's actions take at most one input")
      // scalafix:on DisableSyntax.asInstanceOf
      (b.decl.name, s, inputs, result)

  // An error in an object's initializer is fatal to munit's `intercept`, so it is caught here.
  def refusal(m: => Machine[Lamp, Said, Nothing]): String =
    try
      m.bindings: Unit
      fail("the rules constructed")
    catch case e: ExceptionInInitializerError => e.getCause.getMessage

  test("from, when, when(set), when(...).where and on(a, b) build the table on and in build") {
    assertEquals(Grouped.bindings.map(_.decl), Plain.bindings.map(_.decl))
    assertEquals(steps(Grouped), steps(Plain))
    assert(steps(Grouped).exists((_, _, _, result) => result != Nil))
  }

  test("an action or class sits in several blocks whose cases hold in no common state") {
    assertEquals(Spread.bindings.map(_.decl), List(hand.press, clock.tick).map(_.decl))
    assertEquals(press(Spread, off).map(_.light), List(Lit.on))
    assertEquals(press(Spread, lit).map(_.light), List(Lit.off))
    assertEquals(press(Spread, Lamp(Lit.broken, UpTo(0))).map(_.light), List(Lit.broken))
  }

  test("a block in a block, a from in a from and an action its declarer does not declare") {
    assertEquals(
      refusal(Nested),
      "requirement failed: on(tick) sits in on(press): a block holds cases alone"
    )
    assertEquals(
      refusal(NestedFrom),
      "requirement failed: from(clock) sits in from(hand): a from holds on blocks alone"
    )
    assertEquals(
      refusal(Foreign),
      "requirement failed: on(tick) sits in from(hand), and hand declares no tick: a from holds " +
        "the blocks of the actions its declarer declares"
    )
  }

  test("an on names each target once, none of them disabled") {
    assertEquals(
      refusal(NamedTwice),
      "requirement failed: press is named twice in one on: name each action, or class of it, once"
    )
    assertEquals(
      refusal(DisabledTarget),
      "requirement failed: tick is disabled and fired by a rule"
    )
  }

  test("two blocks of one class whose cases hold in one state are refused") {
    assertEquals(
      refusal(AcrossBlocks),
      "acrossBlocks fires turn-down by two rules in Lamp(off,0): rule 1, when(off), and rule 3, " +
        "when((l: umpire.RulesFixture.Lit) => umpire.RulesFixture.lights.unlit(l)).where: the " +
        "rules of one action class hold in no common state, so write alternatives as one effect " +
        "that names each with `choose`"
    )
  }

  test("a rule names its action as written") {
    assertEquals(writtenAction("example.orders.clerk.ship"), "ship")
    assertEquals(
      writtenAction("example.orders.buyer.change.apply(example.orders.Change.hold)"),
      "change"
    )
    assertEquals(writtenAction("umpire.apply[(A, B)](example.orders.buyer.place)(x := y)"), "place")
  }

  // fn-136.4: `when[R]` fires in the phases with the role, the narrower roles' included, and
  // `phase.in[R]` holds in them; rules that declare no projection name no role.
  test("when[R] fires in exactly the phases with the role R") {
    import RoleRulesFixture.*
    val press =
      Gate.bindings.head.function.asInstanceOf[Gate.Press] // scalafix:ok DisableSyntax.asInstanceOf
    assertEquals(
      Stage.values.toList.filter(p => press(Door(p)).nonEmpty),
      List(Stage.queued, Stage.backingOff, Stage.done)
    )
    assertEquals(
      Stage.values.toList.filter(_.in[Waiting]),
      List(Stage.queued, Stage.backingOff)
    )
    val refused = compileErrors(
      "object NoProjection extends Machine[Door, Said, Nothing] {\n" +
        "  val init = Door(Stage.unstarted)\n  def end(s: Door) = true\n" +
        "  object rules extends Rules { on(hand.press)(when[Closed] ~> Gate.effects.open) }\n}"
    )
    assert(
      refused.contains(
        "when names the phases of a role, and these rules declare no projection: declare the " +
          "projection the phases are of, `object rules extends Rules(_.phase)`"
      ),
      refused
    )
  }

object RoleRulesFixture:
  import RulesFixture.{hand, Said, given}

  enum Stage derives Finite:
    case unstarted
    case queued extends Stage, Waiting
    case backingOff extends Stage, Retrying
    case running extends Stage, Held
    case done extends Stage, Succeeded

  final case class Door(stage: Stage) derives Finite

  object Gate extends Machine[Door, Said, Nothing]:
    type Press = Door => List[Step[Door, Said, Nothing]]
    val init = Door(Stage.unstarted)
    def end(s: Door) = s.stage.in[Closed]
    object effects:
      def open(s: Door) = stay[Door, Said, Nothing](s)
    object rules extends Rules(_.stage):
      on(hand.press) {
        when[Waiting] ~> effects.open
        when[Closed] ~> effects.open
      }
