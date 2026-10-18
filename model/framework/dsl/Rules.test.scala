package framework.dsl

import framework.checker.Effects.*
import framework.dsl.Finite.given
// The object forms' runtime wiring, run as Scala: a machine object is the machine, its `rules` the
// one section it reads, lowered to one step function per action, preserving declaration order.

import scala.collection.mutable

// A dotted file name holds no top-level definitions, so the fixture Models sit in an object.
object RulesFixture:
  enum Lit:
    case off, on, broken

  final case class Lamp(light: Lit, presses: UpTo[2])

  enum Said:
    case ok, refused

  enum Knob:
    case up, down

  given Ok[Said] = Ok(Said.ok)

  object hand extends Actor:
    val press = action
    val turn = action.input[Knob](name = "knob")

  object clock extends Internal:
    val tick = timer

  // What initialized, in order, so the tests read when each object is constructed.
  object Trace:
    val initialized = mutable.ArrayBuffer.empty[String]

  object Switch extends RecordMachine[Lamp, Said, Nothing]:
    Trace.initialized += "Switch"
    val init = Lamp(Lit.off, UpTo(0))
    override def end(s: Lamp) = s.light == Lit.broken
    def presses(s: Lamp): UpTo[2] = UpTo((s.presses + 1).min(2))

    object effects:
      Trace.initialized += "Switch.effects"
      def light(s: Lamp) =
        enter[Lamp, Said, Nothing](s.copy(light = Lit.on, presses = presses(s)))
      def dark(s: Lamp) = enter[Lamp, Said, Nothing](s.copy(light = Lit.off))
      def refuse(s: Lamp, @scala.annotation.unused k: Knob) = List(
        Step[Lamp, Said, Nothing](Said.refused, s)
      )
      def turned(s: Lamp, k: Knob) =
        enter[Lamp, Said, Nothing](
          s.copy(light = if k == Knob.up then Lit.on else Lit.off)
        )
      def wear(s: Lamp) = enter[Lamp, Said, Nothing](s.copy(light = Lit.broken))

    object transitions extends Rules:
      Trace.initialized += "Switch.transitions"
      on(hand.press):
        where(_.light == Lit.off) ~> effects.light
        where(_.light == Lit.on) ~> effects.dark
      on(hand.turn(Knob.down)):
        where(_.light == Lit.on) ~> effects.dark
      on(hand.turn(Knob.up)):
        where(_.light == Lit.on) ~> effects.light
      on(hand.turn):
        where(_.light == Lit.off) ~> effects.turned
        where(_.light == Lit.broken) ~> effects.refuse
      on(clock.tick):
        where(s => s.presses == 2 && s.light != Lit.broken) ~> effects.wear

  // The switch whose press is its own effect wherever it fires, and whose tick never fires.
  object Stuck extends Derived(Switch.rebind(hand.press ~> Switch.effects.wear))

  // The switch whose press fires everywhere, under one rule.
  object Loose
      extends Derived(
        Switch.rebind(
          on(hand.press):
            always ~> Switch.effects.wear
        )
      )

  object Overlapping extends RecordMachine[Lamp, Said, Nothing]:
    val init = Lamp(Lit.off, UpTo(0))
    override def end(s: Lamp) = true

    object transitions extends Rules:
      on(hand.turn(Knob.up)):
        where(_.light.in(Lit.off, Lit.on)) ~> Switch.effects.light
      on(hand.turn):
        where(_.presses == 1) ~> Switch.effects.turned

  object Unfelt extends RecordMachine[Lamp, Said, Nothing]:
    val init = Lamp(Lit.off, UpTo(0))
    def end(s: Lamp) = true

    object transitions extends Rules:
      on(hand.press):
        always ~> Switch.effects.light
      disabled(clock.tick)

  // One lamp's rules twice: step functions bound in the core, and argument-less `Rules` reading
  // the lamp's light.
  object Projected extends RecordMachine[Lamp, Said, Nothing]:
    val init = Lamp(Lit.off, UpTo(0))
    def end(s: Lamp) = true

    object states:
      def dark(l: Lit) = l != Lit.on

    object transitions
        extends Bindings(
          hand.press ~> ((s: Lamp) =>
            if s.light == Lit.on then Switch.effects.dark(s)
            else if states.dark(s.light) then Switch.effects.light(s)
            else Nil
          ),
          hand.turn ~> ((s: Lamp, k: Knob) =>
            if List(Lit.off, Lit.broken).contains(s.light) then Switch.effects.refuse(s, k)
            else Nil
          )
        )

  object RuledLamp extends RecordMachine[Lamp, Said, Nothing]:
    val init = Lamp(Lit.off, UpTo(0))
    override def end(s: Lamp) = true

    object states:
      def dark(l: Lit) = l != Lit.on

    object transitions extends Rules:
      on(hand.press):
        where(_.light == Lit.on) ~> Switch.effects.dark
        where(s => states.dark(s.light)) ~> Switch.effects.light
      on(hand.turn):
        where(_.light.in(Lit.off, Lit.broken)) ~> Switch.effects.refuse

  // The lamp on the shared outcomes: a press of a lit lamp is rejected with the server's reason,
  // and of a broken one as not found.
  object Guarded extends RecordMachine[Lamp, outcomes.Outcome, Nothing]:
    import outcomes.Rejection
    val init = Lamp(Lit.off, UpTo(0))
    override def end(s: Lamp) = true

    object effects:
      def light(s: Lamp) = enter[Lamp, Outcome, Nothing](s.copy(light = Lit.on))

    object transitions extends Rules:
      on(hand.press):
        where(_.light == Lit.off) ~> effects.light
        where(_.light == Lit.on) ~> rejects(Rejection.failedPrecondition).because(
          "the lamp is lit"
        )
        where(_.light == Lit.broken) ~> rejects(Rejection.notFound)

  // The switch's sets of lights, which a case names with `where`.
  object lights:
    def unlit(l: Lit) = l != Lit.on

  // One machine in the grouped forms, `from` with qualified targets, `where` and an `on` of two
  // classes, and in the plain form, separate qualified `on` blocks, rule for rule.
  object Grouped extends RecordMachine[Lamp, Said, Nothing]:
    val init = Lamp(Lit.off, UpTo(0))
    override def end(s: Lamp) = s.light == Lit.broken

    object transitions extends Rules:
      from(hand):
        on(hand.press):
          where(_.light == Lit.off) ~> Switch.effects.light
          where(_.light == Lit.on).where(_.presses == 1) ~> Switch.effects.dark
        on(hand.turn(Knob.up), hand.turn(Knob.down)):
          where(s => lights.unlit(s.light)) ~> Switch.effects.wear
      from(clock):
        on(clock.tick):
          where(_.light.in(Lit.on, Lit.off)).where(_.presses == 2) ~> Switch.effects.wear

  object Plain extends RecordMachine[Lamp, Said, Nothing]:
    val init = Lamp(Lit.off, UpTo(0))
    override def end(s: Lamp) = s.light == Lit.broken

    object transitions extends Rules:
      on(hand.press):
        where(_.light == Lit.off) ~> Switch.effects.light
        where(_.light == Lit.on).where(_.presses == 1) ~> Switch.effects.dark
      on(hand.turn(Knob.up)):
        where(s => lights.unlit(s.light)) ~> Switch.effects.wear
      on(hand.turn(Knob.down)):
        where(s => lights.unlit(s.light)) ~> Switch.effects.wear
      on(clock.tick):
        where(_.light.in(Lit.on, Lit.off)).where(_.presses == 2) ~> Switch.effects.wear

  // One action in several blocks whose cases hold in no common state.
  object Spread extends RecordMachine[Lamp, Said, Nothing]:
    val init = Lamp(Lit.off, UpTo(0))
    override def end(s: Lamp) = true

    object transitions extends Rules:
      on(hand.press):
        where(_.light == Lit.off) ~> Switch.effects.light
      on(hand.press, clock.tick):
        where(_.light == Lit.broken) ~> Switch.effects.wear
      on(hand.press):
        where(_.light == Lit.on) ~> Switch.effects.dark

  // The rules each of these declares are refused as its rules construct.
  object Nested extends RecordMachine[Lamp, Said, Nothing]:
    val init = Lamp(Lit.off, UpTo(0))
    override def end(s: Lamp) = true

    object transitions extends Rules:
      on(hand.press):
        on(clock.tick):
          always ~> Switch.effects.wear

  object NestedFrom extends RecordMachine[Lamp, Said, Nothing]:
    val init = Lamp(Lit.off, UpTo(0))
    override def end(s: Lamp) = true

    object transitions extends Rules:
      from(hand):
        from(clock):
          on(clock.tick):
            always ~> Switch.effects.wear

  object Foreign extends RecordMachine[Lamp, Said, Nothing]:
    val init = Lamp(Lit.off, UpTo(0))
    override def end(s: Lamp) = true

    object transitions extends Rules:
      from(hand):
        on(clock.tick):
          always ~> Switch.effects.wear

  object NamedTwice extends RecordMachine[Lamp, Said, Nothing]:
    val init = Lamp(Lit.off, UpTo(0))
    override def end(s: Lamp) = true

    object transitions extends Rules:
      on(hand.press, clock.tick, hand.press):
        always ~> Switch.effects.wear

  object DisabledTarget extends RecordMachine[Lamp, Said, Nothing]:
    val init = Lamp(Lit.off, UpTo(0))
    override def end(s: Lamp) = true

    object transitions extends Rules:
      disabled(clock.tick)
      on(hand.press, clock.tick):
        always ~> Switch.effects.wear

  object AcrossBlocks extends RecordMachine[Lamp, Said, Nothing]:
    val init = Lamp(Lit.off, UpTo(0))
    override def end(s: Lamp) = true

    object transitions extends Rules:
      on(hand.turn):
        where(_.light == Lit.off) ~> Switch.effects.turned
      on(hand.turn(Knob.down)):
        where(_.light == Lit.on) ~> Switch.effects.dark
        where(s => lights.unlit(s.light)).where(
          _.presses == 0
        ) ~> Switch.effects.wear

  final case class Pair(left: Lamp, right: Lamp)

  object NativeSwitch extends Machine[Said, Nothing]:
    var light = Lit.off

    object transitions extends Transitions:
      on(hand.press):
        effects.lightUp()

    object effects:
      def lightUp() =
        light = Lit.on
      def wear() =
        light = Lit.broken

  object NativeUnfelt extends Machine[Said, Nothing]:
    var light = Lit.off

    object transitions extends Transitions:
      on(hand.press):
        effects.lightUp()

    object effects:
      def lightUp() =
        light = Lit.on

  object NativeStuck
      extends Derived(
        NativeSwitch.rebind(
          on(hand.press):
            NativeSwitch.effects.wear()
        )
      )

  object Twins extends Composition[Nothing]:
    val left = NativeSwitch
    val right = NativeUnfelt
    override def end = left.light == Lit.broken

    object syncs:
      val pressed = sync(left(hand.press), right(hand.press))

  object Odd extends Composition(Twins.withMember(left = NativeStuck))

class RulesTest extends munit.FunSuite:
  import RulesFixture.*
  type Press = Lamp => List[Step[Lamp, Said, Nothing]]
  type Turn = (Lamp, Knob) => List[Step[Lamp, Said, Nothing]]

  def step(m: RecordMachine[Lamp, Said, Nothing], a: Action[?]): AnyRef =
    m.bindings.find(_.decl == a.decl).get.function

  def press(m: RecordMachine[Lamp, Said, Nothing], s: Lamp): List[Lamp] =
    step(m, hand.press)
      .asInstanceOf[Press](s)
      .map(_.state) // scalafix:ok DisableSyntax.asInstanceOf

  def turn(
      m: RecordMachine[Lamp, Said, Nothing],
      s: Lamp,
      k: Knob
  ): List[Step[Lamp, Said, Nothing]] =
    step(m, hand.turn)
      .asInstanceOf[Turn](s, k) // scalafix:ok DisableSyntax.asInstanceOf

  val off = Lamp(Lit.off, UpTo(0))
  val lit = Lamp(Lit.on, UpTo(1))

  test(
    "grouped monitor references preserve lazy attachment and initialize one definition"
  ) {
    val initialized = scala.collection.mutable.ArrayBuffer.empty[String]
    object Watched extends RecordMachine[Lamp, Said, Nothing]:
      val init = Lamp(Lit.off, UpTo(0))
      def end(s: Lamp) = true
      object monitors:
        lazy val unchanged =
          initialized += "monitor": Unit
          monitor[Lamp, Said, Nothing, Boolean]("unchanged", false)((_, before, after) =>
            before.light == after.state.light
          )(identity)
      object transitions
          extends Bindings(
            hand.press ~> ((s: Lamp) => List(Step(Said.refused, s)))
          )
      object properties:
        object safety:
          lazy val observed =
            initialized += "reference": Unit
            monitors.unchanged
          lazy val alias = observed
        object helper:
          lazy val ignored =
            initialized += "helper": Unit
            monitors.unchanged
    assertEquals(Watched.init, off)
    assertEquals(initialized.toList, Nil)
    val file = IrFile("monitor-reference", Seq(Watched))
    file.construct(): Unit
    assertEquals(initialized.toList, List("reference", "monitor"))
    assert(Watched.properties.safety.observed eq Watched.monitors.unchanged)
    assert(Watched.properties.safety.alias eq Watched.monitors.unchanged)
    file.construct(): Unit
    assertEquals(initialized.toList, List("reference", "monitor"))
  }

  test(
    "a machine object is named after its object, and its sections initialize on first use"
  ) {
    assertEquals(Switch.name, "switch")
    assertEquals(Switch.init, off)
    assert(Trace.initialized.contains("Switch"))
    assert(
      !Trace.initialized.contains("Switch.transitions"),
      Trace.initialized.toString
    )
    val _ = Switch.bindings
    assertEquals(
      Trace.initialized.toList.filter(_.startsWith("Switch")).take(2),
      List("Switch", "Switch.transitions")
    )
  }

  test(
    "each action's rules lower to one step function: the first rule that fires, else no step"
  ) {
    assertEquals(
      Switch.bindings.map(_.decl),
      List(hand.press, hand.turn, clock.tick).map(_.decl)
    )
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
    for state <- Finite[Lamp].values do
      val expected = state.light match
        case Lit.off =>
          List(
            state.copy(
              light = Lit.on,
              presses = UpTo[2]((state.presses + 1).min(2))
            )
          )
        case Lit.on     => List(state.copy(light = Lit.off))
        case Lit.broken => Nil
      assertEquals(press(Switch, state), expected, state)
      for knob <- Finite[Knob].values do
        val expected = state.light match
          case Lit.broken =>
            List(Step[Lamp, Said, Nothing](Said.refused, state))
          case Lit.off =>
            enter[Lamp, Said, Nothing](
              state.copy(light = if knob == Knob.up then Lit.on
              else Lit.off)
            )
          case Lit.on =>
            enter[Lamp, Said, Nothing](
              if knob == Knob.up then state.copy(presses = UpTo[2]((state.presses + 1).min(2)))
              else state.copy(light = Lit.off)
            )
        assertEquals(turn(Switch, state, knob), expected, (state, knob))
  }

  test("disabled binds an action no state enables") {
    assertEquals(
      Unfelt.bindings.map(_.decl),
      List(hand.press, clock.tick).map(_.decl)
    )
    val tick =
      Unfelt.bindings.last.function
        .asInstanceOf[Press] // scalafix:ok DisableSyntax.asInstanceOf
    assertEquals(Finite[Lamp].values.toList.flatMap(tick), Nil)
  }

  test(
    "a bare binding of a derivation keeps its action's rules and replaces their effect"
  ) {
    assertEquals(Stuck.name, "stuck")
    assertEquals(press(Stuck, off).map(_.light), List(Lit.broken))
    assertEquals(press(Stuck, Lamp(Lit.broken, UpTo(0))), Nil)
    assertEquals(
      press(Loose, Lamp(Lit.broken, UpTo(0))).map(_.light),
      List(Lit.broken)
    )
    assertEquals(turn(Stuck, lit, Knob.down).map(_.state.light), List(Lit.off))
    for state <- Finite[Lamp].values do
      assertEquals(
        press(Stuck, state),
        if state.light == Lit.broken then Nil
        else List(state.copy(light = Lit.broken))
      )
      assertEquals(press(Loose, state), List(state.copy(light = Lit.broken)))
      for knob <- Finite[Knob].values do
        assertEquals(turn(Stuck, state, knob), turn(Switch, state, knob))
  }

  test(
    "overlapping rules retain their declaration metadata without checking states"
  ) {
    val rules = Overlapping.table.collect { case (_, Bound.Ruled(rs)) => rs }.flatten
    assertEquals(rules.map(_.index), Vector(1, 2))
    assertEquals(rules.map(_.heading), Vector("where", "where"))
    assertEquals(rules.map(_.values), Vector(Some(List(Knob.up)), None))
    assertEquals(Overlapping.bindings.size, 1)
  }

  test(
    "argument-less rules read the state as the core step functions read it"
  ) {
    val knobs = Knob.values.toList
    for s <- Finite[Lamp].values do
      assertEquals(press(RuledLamp, s), press(Projected, s), s)
      for k <- knobs do assertEquals(turn(RuledLamp, s, k), turn(Projected, s, k), (s, k))
    assertEquals(press(RuledLamp, off), List(Lamp(Lit.on, UpTo(1))))
    assertEquals(press(RuledLamp, lit), List(lit.copy(light = Lit.off)))
    assertEquals(
      turn(RuledLamp, off, Knob.up).map(_.outcome),
      List(Said.refused)
    )
    assertEquals(turn(RuledLamp, lit, Knob.up), Nil)
  }

  test("Rules takes no projection") {
    val refused = compileErrors(
      "object Old extends RecordMachine[Lamp, Said, Nothing]:\n" +
        "  val init = Lamp(Lit.off, UpTo(0))\n" +
        "  def end(s: Lamp) = true\n" +
        "  object transitions extends " + "Rules(_.light):\n" +
        "    on(hand.press)(where(_.light == Lit.on) ~> Switch.effects.dark)"
    )
    assert(refused.nonEmpty, "Rules still accepts a projection argument")
  }

  test("a derivation retains multiple rules without evaluating their overlap") {
    val derived = Switch.rebind(
      on(hand.press):
        always ~> Switch.effects.wear
        where(_.light == Lit.off) ~> Switch.effects.dark
    )
    val rules = derived.table.collect {
      case (decl, Bound.Ruled(rs)) if decl == hand.press.decl => rs
    }.flatten
    assertEquals(rules.map(_.index), Vector(1, 2))
    assertEquals(rules.map(_.heading), Vector("always", "where"))
  }

  test(
    "a composition object is its members, and an IR file constructs what it reaches"
  ) {
    assertEquals(Twins.name, "twins")
    assertEquals(Twins.members, Vector(NativeSwitch, NativeUnfelt))
    assertEquals(Odd.name, "odd")
    assertEquals(Odd.members, Vector(NativeStuck, NativeUnfelt))
    intercept[IllegalArgumentException](Twins.withMember(right = NativeStuck)): Unit
    // Built directly, so the IR files the gate holds model/ir to stay the Models' own.
    IrFile("rules-test", Seq(Odd, Switch)).construct(): Unit
  }

  test(
    "a rejects row keeps the state and answers rejected, with its reason where one is given"
  ) {
    import outcomes.{Outcome, Rejection}
    type Answered = Lamp => List[Step[Lamp, Outcome, Nothing]]
    val pressed =
      Guarded.bindings.head.function
        .asInstanceOf[Answered] // scalafix:ok DisableSyntax.asInstanceOf
    val broken = Lamp(Lit.broken, UpTo(0))
    assertEquals(
      pressed(off),
      List(
        Step[Lamp, Outcome, Nothing](
          Outcome.accepted,
          lit.copy(presses = UpTo(0))
        )
      )
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
      List(
        Step[Lamp, Outcome, Nothing](
          Outcome.rejected(Rejection.notFound),
          broken
        )
      )
    )
    for state <- Finite[Lamp].values do
      val expected = state.light match
        case Lit.off =>
          Step[Lamp, Outcome, Nothing](
            Outcome.accepted,
            state.copy(light = Lit.on)
          )
        case Lit.on =>
          Step[Lamp, Outcome, Nothing](
            Outcome.rejected(Rejection.failedPrecondition),
            state,
            Nil,
            "the lamp is lit"
          )
        case Lit.broken =>
          Step[Lamp, Outcome, Nothing](
            Outcome.rejected(Rejection.notFound),
            state
          )
      assertEquals(pressed(state), List(expected), state)
  }

  // Every step of a machine, by action, state and class, as the bindings give them.
  def steps(
      m: RecordMachine[Lamp, Said, Nothing]
  ): List[(String, Lamp, List[Any], List[Step[Lamp, Said, Nothing]])] =
    for
      b <- m.bindings.toList
      s <- Finite[Lamp].values.toList
      inputs <- classesOf(b.decl)
    yield
      // scalafix:off DisableSyntax.asInstanceOf
      val result = inputs match
        case Nil =>
          b.function.asInstanceOf[Lamp => List[Step[Lamp, Said, Nothing]]](s)
        case List(one) =>
          b.function
            .asInstanceOf[(Lamp, Any) => List[Step[Lamp, Said, Nothing]]](
              s,
              one
            )
        case _ => fail("the fixture's actions take at most one input")
      // scalafix:on DisableSyntax.asInstanceOf
      (b.decl.name, s, inputs, result)

  // An error in an object's initializer is fatal to munit's `intercept`, so it is caught here.
  def refusal(m: => RecordMachine[Lamp, Said, Nothing]): String =
    try
      val _ = m.bindings
      fail("the rules constructed")
    catch case e: ExceptionInInitializerError => e.getCause.getMessage

  // Imported bare controls are compiled independently in RuleSourceTest.
  test("an on target with no declarer is refused as the rules construct") {
    val tap = hand.press
    object Undeclared extends RecordMachine[Lamp, Said, Nothing]:
      val init = Lamp(Lit.off, UpTo(0))
      override def end(s: Lamp) = true

      object transitions extends Rules:
        on(tap):
          always ~> Switch.effects.wear
    val refused = intercept[IllegalArgumentException](Undeclared.bindings)
    assertEquals(
      refused.getMessage,
      "requirement failed: on(tap) names its action bare at Rules.test.scala:626:12: name it through the object that " +
        "declares it, `on(<declarer>.tap)`"
    )
  }

  test(
    "from, where, where(...).where and on(a, b) build the qualified table"
  ) {
    assertEquals(Grouped.bindings.map(_.decl), Plain.bindings.map(_.decl))
    assertEquals(steps(Grouped), steps(Plain))
    assert(steps(Grouped).exists((_, _, _, result) => result.nonEmpty))
  }

  test(
    "an action or class sits in several blocks whose cases hold in no common state"
  ) {
    assertEquals(
      Spread.bindings.map(_.decl),
      List(hand.press, clock.tick).map(_.decl)
    )
    assertEquals(press(Spread, off).map(_.light), List(Lit.on))
    assertEquals(press(Spread, lit).map(_.light), List(Lit.off))
    assertEquals(
      press(Spread, Lamp(Lit.broken, UpTo(0))).map(_.light),
      List(Lit.broken)
    )
    for state <- Finite[Lamp].values do
      val expected = state.light match
        case Lit.off =>
          state.copy(
            light = Lit.on,
            presses = UpTo[2]((state.presses + 1).min(2))
          )
        case Lit.on     => state.copy(light = Lit.off)
        case Lit.broken => state
      assertEquals(press(Spread, state), List(expected), state)
  }

  test(
    "a block in a block, a from in a from and an action its declarer does not declare"
  ) {
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

  test("rules spread across blocks keep declaration order") {
    val rules = AcrossBlocks.table.collect { case (_, Bound.Ruled(rs)) => rs }.flatten
    assertEquals(rules.map(_.index), Vector(1, 2, 3))
    assertEquals(AcrossBlocks.bindings.size, 1)
  }

  test("a rule names its action as written") {
    assertEquals(writtenAction("example.orders.clerk.ship"), "ship")
    assertEquals(
      writtenAction(
        "example.orders.buyer.change.apply(example.orders.Change.hold)"
      ),
      "change"
    )
  }

  test("historical assignment syntax still yields its written action name") {
    assertEquals(
      writtenAction(
        "framework.dsl.apply[(A, B)](example.orders.buyer.place)(x := y)"
      ),
      "place"
    )
  }

  // fn-136.4: a case on `phase.in[R]` fires in the phases with the role, the narrower roles'
  // included.
  test("a case on a role fires in exactly the phases with the role R") {
    import RoleRulesFixture.*
    val press =
      Gate.bindings.head.function
        .asInstanceOf[Gate.Press] // scalafix:ok DisableSyntax.asInstanceOf
    assertEquals(
      Stage.values.toList.filter(p => press(Door(p)).nonEmpty),
      List(Stage.queued, Stage.backingOff, Stage.done)
    )
    assertEquals(
      Stage.values.toList.filter(_.in[Waiting]),
      List(Stage.queued, Stage.backingOff)
    )
  }

object RoleRulesFixture:
  import RulesFixture.{hand, Said, given}

  enum Stage:
    case unstarted
    case queued extends Stage, Waiting
    case backingOff extends Stage, Retrying
    case running extends Stage, Held
    case done extends Stage, Succeeded

  final case class Door(stage: Stage)

  object Gate extends RecordMachine[Door, Said, Nothing]:
    type Press = Door => List[Step[Door, Said, Nothing]]
    val init = Door(Stage.unstarted)
    override def end(s: Door) = s.stage.in[Closed]

    object effects:
      def open(s: Door) = stay[Door, Said, Nothing](s)

    object transitions extends Rules:
      on(hand.press):
        where(_.stage.in[Waiting]) ~> effects.open
        where(_.stage.in[Closed]) ~> effects.open
