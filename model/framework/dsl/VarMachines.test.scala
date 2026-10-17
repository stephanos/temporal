package framework.dsl

import framework.dsl.Finite.given

// A lamp whose state is two vars: its light and how often it was pressed. Its effects assign them,
// its guards and claims read them by name, and the native runtime restores a state into them
// before each step and claim.
object VarMachinesFixture:
  enum Lit:
    case off, on, broken

  enum Said:
    case ok, broken

  enum Note:
    case lit, darkened

  given Ok[Said] = Ok(Said.ok)

  object hand extends Actor:
    val press = action
    val smash = action
    val repair = action
    val set = action.input[Lit](name = "light")
    val flip = action

  object Lamp extends Machine[Said, Note]:
    var light = Lit.off
    var presses: UpTo[3] = UpTo(0)

    // a run stops once the lamp is broken
    override def end = light == Lit.broken

    object transitions extends Rules:
      on(hand.press):
        where(light != Lit.broken && presses < 3) ~> effects.press()
        where(light == Lit.broken) ~> effects.refuse()
      on(hand.smash):
        always ~> effects.smash()
      on(hand.repair):
        where(light == Lit.broken) ~> effects.repair()
      on(hand.set):
        where((to: Lit) => to != light && light != Lit.broken) ~> effects.set
      on(hand.flip):
        where(light != Lit.broken) ~> effects.flip()

    object effects:
      def press(): Unit =
        presses = UpTo(presses + 1)
        if light == Lit.on then
          light = Lit.off
          record(Note.darkened)
        else
          light = Lit.on
          record(Note.lit)
      def refuse(): Unit = reject(Said.broken)
      def smash(): Unit = light = Lit.broken
      // a read after a write in the same effect sees the write
      def repair(): Unit =
        presses = UpTo(0)
        light = if presses == 0 then Lit.off else Lit.on
      def set(to: Lit): Unit = light = to
      val lit = choice
      val darkened = choice
      // either way, recording which
      def flip(): Unit =
        choose(
          lit -> {
            light = Lit.on
            record(Note.lit)
          },
          darkened -> {
            light = Lit.off
            record(Note.darkened)
          }
        )

    object invariants:
      val brokenStaysUnpressed = property.expect:
        if light == Lit.broken then check(presses <= 3)
      val darkWhenUnpressed = property.expect:
        if presses == 0 then check(light != Lit.on)

    object postconditions:
      val pressCounts = property
        .after(hand.press)
        .across
        .expect:
          if after.outcome == Said.ok then check(presses == before.presses + 1)
      val pressToggles = property.across.expect:
        if after.records(Note.lit) then check(before.light == Lit.off && light == Lit.on)

  // The lamp whose press smashes it.
  object Brittle extends Derived(Lamp.rebind(hand.press ~> Lamp.effects.smash()))

  // The lamp whose light is set on whatever it is asked.
  object Stubborn extends Derived(Lamp.rebind(hand.set ~> ((_: Lit) => Lamp.effects.set(Lit.on))))

  // The lamp that is only pressed.
  object Pressed extends Derived(Lamp.restrict(hand.press)):
    import Lamp.light

    object progress:
      val lights = Pressed.leadsTo("lights")(light == Lit.off, light == Lit.on, within = 1)

class VarMachinesTest extends munit.FunSuite:
  import VarMachinesFixture.*

  private def run(machine: RecordMachine[State, Said, Note], state: State, cls: Class) =
    val binding = machine.bindings.find(_.decl eq cls.decl).get
    effectOf[State, Said, Note](binding.decl, binding.function)(state, cls)

  private def holds(property: Property, step: Step[State, Said, Note]): Boolean =
    property.decl.holds.get.asInstanceOf[Step[State, Said, Note] => Boolean](step)

  private def holds(property: Property, before: State, step: Step[State, Said, Note]): Boolean =
    property.decl.holds2.get.asInstanceOf[(State, Step[State, Said, Note]) => Boolean](before, step)

  private val off = Lamp.init

  private def lamp(light: Lit, presses: Int) = off.copy(light = light, presses = UpTo[3](presses))

  test("a Machine's vars are its state, in declaration order, with their declared types"):
    assertEquals(Lamp.vars.map(_.name), Vector("light", "presses"))
    assertEquals(Lamp.vars.map(_.declared), Vector("Lit", "UpTo[3]"))
    assertEquals(Lamp.vars.map(_.bound), Vector(None, Some(3)))
    assertEquals(off.toString, "State(light = off, presses = 0)")

  test("init is what the vars start in, and a state is built from it by checked named updates"):
    assertEquals(off, lamp(Lit.off, 0))
    assertEquals(Lamp.at(off.copy(light = Lit.on))(Lamp.light), Lit.on)
    assertNotEquals(off.copy(light = Lit.on), off)
    assertEquals(Data[State].compare(off, lamp(Lit.on, 0)), Data[Lit].compare(Lit.off, Lit.on))
    val unknown = intercept[IllegalArgumentException](off.copy(lamp = Lit.on))
    assert(unknown.getMessage.contains("lamp is not a state var of lamp"), unknown.getMessage)
    val mistyped = intercept[IllegalArgumentException](off.copy(light = 3))
    assert(mistyped.getMessage.contains("light: 3 is not a Lit"), mistyped.getMessage)
    val foreign = intercept[IllegalArgumentException](Data[State].compare(off, Thermometer.init))
    assert(foreign.getMessage.contains("a state of another machine"), foreign.getMessage)

  test("a step runs on the vars restored from its state, and the vars read as init after it"):
    assertEquals(
      run(Lamp, off, hand.press()),
      List(Step(Said.ok, lamp(Lit.on, 1), List(Note.lit)))
    )
    assertEquals(
      run(Lamp, lamp(Lit.on, 1), hand.press()),
      List(Step(Said.ok, lamp(Lit.off, 2), List(Note.darkened)))
    )
    assertEquals((Lamp.light, Lamp.presses: Int), (Lit.off, 0))
    assertEquals(Lamp.init, off)

  test("a guard that does not hold disables the step, and a rejection keeps the start state"):
    assertEquals(run(Lamp, lamp(Lit.on, 3), hand.press()), Nil)
    val broken = lamp(Lit.broken, 2)
    assertEquals(run(Lamp, broken, hand.press()), List(Step(Said.broken, broken)))
    assertEquals(run(Lamp, broken, hand.smash()), List(Step(Said.ok, broken)))
    assertEquals(run(Lamp, off, hand.repair()), Nil)
    assertEquals(run(Lamp, broken, hand.repair()), List(Step(Said.ok, lamp(Lit.off, 0))))

  test("an effect of an input takes it as its parameter, and an input guard reads it"):
    assertEquals(run(Lamp, off, hand.set(Lit.on)), List(Step(Said.ok, lamp(Lit.on, 0))))
    assertEquals(run(Lamp, off, hand.set(Lit.off)), Nil)
    assertEquals(run(Lamp, lamp(Lit.broken, 0), hand.set(Lit.on)), Nil)

  test("an effect's choose takes one step per alternative, each from the state it started in"):
    assertEquals(
      run(Lamp, lamp(Lit.on, 2), hand.flip()),
      List(
        Step(Said.ok, lamp(Lit.on, 2), List(Note.lit)),
        Step(Said.ok, lamp(Lit.off, 2), List(Note.darkened))
      )
    )
    assertEquals(run(Lamp, lamp(Lit.broken, 2), hand.flip()), Nil)
    assertEquals(Lamp.light, Lit.off)
    val outside = intercept[IllegalStateException](Lamp.effects.flip())
    assert(outside.getMessage.contains("choose is a statement of an effect"), outside.getMessage)

  test("end reads the vars of the state it is asked of"):
    assert(Lamp.end(lamp(Lit.broken, 1)))
    assert(!Lamp.end(off))

  test("claims read the vars of the state a step went to, and before is the typed State record"):
    assert(holds(Lamp.invariants.darkWhenUnpressed, Step(Said.ok, off)))
    assert(!holds(Lamp.invariants.darkWhenUnpressed, Step(Said.ok, lamp(Lit.on, 0))))
    assert(holds(Lamp.invariants.brokenStaysUnpressed, Step(Said.ok, lamp(Lit.broken, 3))))
    val pressed = Step(Said.ok, lamp(Lit.on, 1), List(Note.lit))
    assert(holds(Lamp.postconditions.pressCounts, off, pressed))
    assert(!holds(Lamp.postconditions.pressCounts, lamp(Lit.off, 1), pressed))
    assert(holds(Lamp.postconditions.pressCounts, off, Step(Said.broken, off)))
    assert(holds(Lamp.postconditions.pressToggles, off, pressed))
    assert(!holds(Lamp.postconditions.pressToggles, lamp(Lit.on, 0), pressed))
    assertEquals(Lamp.postconditions.pressCounts.decl.when, Some(hand.press))
    assertEquals((Lamp.light, Lamp.presses: Int), (Lit.off, 0))

  test("at evaluates on a state's vars, nests, and puts back what it found"):
    val on = lamp(Lit.on, 2)
    assertEquals(Lamp.at(on)(Lamp.light), Lit.on)
    assertEquals(Lamp.at(on)((Lamp.at(off)(Lamp.light), Lamp.light)), (Lit.off, Lit.on))
    assertEquals(Lamp.light, Lit.off)
    val error = intercept[IllegalStateException](Lamp.at(on)(throw IllegalStateException("x")))
    assertEquals(error.getMessage, "x")
    assertEquals(Lamp.light, Lit.off)

  test("a statement of an effect outside one is refused"):
    val error = intercept[IllegalStateException](Lamp.effects.refuse())
    assert(error.getMessage.contains("a statement of an effect"), error.getMessage)

  test("derivations share the vars of their source and bind its effects"):
    assertEquals(Brittle.init, off)
    assertEquals(run(Brittle, off, hand.press()), List(Step(Said.ok, lamp(Lit.broken, 0))))
    assertEquals(run(Stubborn, off, hand.set(Lit.off)), Nil)
    assertEquals(
      run(Stubborn, lamp(Lit.on, 0), hand.set(Lit.off)),
      List(Step(Said.ok, lamp(Lit.on, 0)))
    )
    assertEquals(Pressed.bindings.map(_.decl), List(hand.press.decl))
    val lights = Pressed.progress.lights
    assert(lights.from(off) && !lights.to(off))
    assert(lights.to(lamp(Lit.on, 1)))

  test("a Machine without vars, and a var whose type has no schema, are refused"):
    val empty = compileErrors(
      "object Empty extends Machine[VarMachinesFixture.Said, Nothing]"
    )
    assert(empty.contains("a Machine declares at least one state var"), empty)
    val unschematic = compileErrors(
      "object Hot extends Machine[VarMachinesFixture.Said, Nothing]:\n  var temperature: Double = 0.0\n"
    )
    assert(
      unschematic.contains(
        "temperature: Double has no IR schema; use Int, a bounded alias, Boolean, an enum, a " +
          "record, Option, Set, Map, List or Inbox"
      ),
      unschematic
    )

  test("before reads the vars a postcondition's Machine declares, typed"):
    val missing = compileErrors(
      "object M extends Machine[VarMachinesFixture.Said, Nothing]:\n  var x = 1\n  object postconditions:\n    val a = property.across.expect:\n      check(before.y == 1)\n"
    )
    assert(missing.contains("value y is not a member of framework.dsl.State"), missing)
    val mistyped = compileErrors(
      "object M extends Machine[VarMachinesFixture.Said, Nothing]:\n  var x = 1\n  object postconditions:\n    val a = property.across.expect:\n      check(before.x)\n"
    )
    assert(mistyped.contains("Required: Boolean"), mistyped)
    val sameStep = compileErrors(
      "object M extends Machine[VarMachinesFixture.Said, Nothing]:\n  var x = 1\n  object invariants:\n    val a = property.expect:\n      check(before.x == 1)\n"
    )
    assert(
      sameStep.contains("before is the state a step started from; use it in a postcondition"),
      sameStep
    )

object Thermometer extends Machine[VarMachinesFixture.Said, Nothing]:
  var temperature = 20

  object transitions extends Rules
