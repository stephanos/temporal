package framework.dsl

import framework.checker.Effects.*
import framework.dsl.Finite.given

// Facets run as Scala: a Machine assembled from facets binds what the same Machine written as one
// object binds, an action bound by two facets is refused, a facet's self-type makes a Machine
// without the facet it depends on fail to compile, and a var a facet declares that the Machine does
// not define is Scala's abstract-member error.
object FacetsFixture:
  enum Lit(val status: Note) extends Recorded[Note]:
    case off extends Lit(Note.dark)
    case on extends Lit(Note.lit)
    case broken extends Lit(Note.gone), Terminated

  enum Note:
    case dark, lit, gone

  enum Said:
    case ok

  given Ok[Said] = Ok(Said.ok)

  object hand extends Actor:
    val press = action
    val smash = action

  // The facet the others step through: pressing. It declares the vars it reads and assigns.
  trait Pressing extends Facet[Said, Note]:
    var light: Lit
    var presses: UpTo[1]

    object pressing:

      object transitions extends Transitions:
        on(hand.press):
          light match
            case Lit.off | Lit.on => effects.toggle()
            case _                => ()

      object effects:
        def toggle() =
          light = if light == Lit.on then Lit.off else Lit.on
          presses = UpTo(1)

  // A facet that depends on Pressing's step, and reads the light through it.
  trait Breaking extends Facet[Said, Note]:
    this: Pressing =>

    object breaking:

      object transitions extends Transitions:
        on(hand.smash):
          light match
            case Lit.on  => effects.smash()
            case Lit.off => effects.breakDown()
            case _       => ()

      object effects:
        val smashed = choice
        val toggled = choice
        def breakDown() =
          light = Lit.broken
          record(light.status)
        def smash() =
          choose(smashed -> breakDown(), toggled -> pressing.effects.toggle())

  object Faceted extends Machine[Said, Note], Pressing, Breaking:
    var light = Lit.off
    var presses: UpTo[1] = UpTo(0)

    override def end: Boolean = light.in[Closed]

    object transitions extends Facets(pressing.transitions, breaking.transitions)

  object Whole extends Machine[Said, Note]:
    var light = Lit.off
    var presses: UpTo[1] = UpTo(0)

    override def end: Boolean = light.in[Closed]

    object transitions extends Transitions:
      on(hand.press):
        light match
          case Lit.off | Lit.on => effects.toggle()
          case _                => ()
      on(hand.smash):
        light match
          case Lit.on  => effects.smash()
          case Lit.off => effects.breakDown()
          case _       => ()

    object effects:
      val smashed = choice
      val toggled = choice
      def toggle() =
        light = if light == Lit.on then Lit.off else Lit.on
        presses = UpTo(1)
      def breakDown() =
        light = Lit.broken
        record(light.status)
      def smash() =
        choose(smashed -> breakDown(), toggled -> toggle())

  // Two facets that both bind the press.
  trait Pressing2 extends Facet[Said, Note]:
    var light: Lit

    object pressingAgain:

      object transitions extends Transitions:
        on(hand.press):
          light match
            case Lit.broken => effects.stayBroken()
            case _          => ()

      object effects:
        def stayBroken() =
          light = Lit.broken
          record(light.status)

  object Twice extends Machine[Said, Note], Pressing, Pressing2:
    var light = Lit.off
    var presses: UpTo[1] = UpTo(0)

    object transitions extends Facets(pressing.transitions, pressingAgain.transitions)

class FacetsTest extends munit.FunSuite:
  import FacetsFixture.*

  // Every state of the lamp: each light with each number of presses.
  private def lamps(m: Machine[Said, Note]): List[State] =
    for
      light <- Lit.values.toList
      presses <- (0 to 1).toList
    yield m.init.copy(light = light, presses = UpTo[1](presses))

  private def steps(m: Machine[Said, Note]) =
    for
      b <- m.bindings
      s <- lamps(m)
      cls <- inputClassesOf(b.decl)
    yield
      val taken = effectOf[State, Said, Note](b.decl, b.function)(s, cls)
      (b.decl.toString, s.values, taken.map(step => (step.outcome, step.state.values, step.facts)))

  test("a Machine assembled from facets binds what the Machine written as one object binds"):
    assertEquals(Faceted.bindings.map(_.decl), Whole.bindings.map(_.decl))
    assertEquals(Faceted.vars.map(_.toString), Whole.vars.map(_.toString))
    assertEquals(steps(Faceted), steps(Whole))
    assert(steps(Faceted).exists(_._3.nonEmpty))
    assert(steps(Faceted).exists(_._3.exists(_._3 == List(Note.gone))))
    assertEquals(lamps(Faceted).map(Faceted.end), lamps(Whole).map(Whole.end))
    assertEquals(lamps(Faceted).count(Faceted.end), 2)

  test("assigning a Recorded phase records nothing; record(light.status) records its status"):
    def taken(action: Action[EmptyTuple]) =
      val b = Faceted.bindings.find(_.decl eq action.decl).get
      effectOf[State, Said, Note](b.decl, b.function)(Faceted.init, action())
        .map(step => (step.state.selectDynamic("light"), step.facts))
    assertEquals(taken(hand.press), List((Lit.on, Nil)))
    assertEquals(taken(hand.smash), List((Lit.broken, List(Note.gone))))

  test("an action bound by two facets is refused as the transitions construct, naming both"):
    val refused = intercept[IllegalArgumentException](Twice.bindings)
    assertEquals(
      refused.getMessage,
      "press is bound by the facets pressing and pressingAgain: an action belongs to one facet, " +
        "which holds all its transitions"
    )

  test("a Machine that mixes in a facet without the facet it depends on does not compile"):
    val errors = compileErrors(
      "object Lone extends Machine[Said, Note], Breaking:\n" +
        "  var light = Lit.off\n" +
        "  object transitions extends Facets(breaking.transitions)"
    )
    assert(errors.contains("Pressing"), errors)

  test("a var a facet declares that the Machine does not define is Scala's abstract-member error"):
    val errors = compileErrors(
      "object Partial extends Machine[Said, Note], Pressing:\n" +
        "  var light = Lit.off\n" +
        "  object transitions extends Facets(pressing.transitions)"
    )
    assert(errors.contains("presses") && errors.contains("is not defined"), errors)

  test("a facet declares its vars abstractly: one it defines itself is refused"):
    val errors = compileErrors(
      "trait Counting extends Facet[Said, Note]:\n" +
        "  var count: UpTo[1] = UpTo(0)\n" +
        "object Counted extends Machine[Said, Note], Counting:\n" +
        "  var light = Lit.off\n" +
        "  object transitions extends Facets()"
    )
    assert(
      errors.contains(
        "count: Counting defines a var, which a facet declares abstractly and the Machine defines"
      ),
      errors
    )

  for kind <- List("ordinary", "timer", "fault", "disabled") do
    test(s"duplicate $kind owners name the actual action and both facets"):
      object targets extends Internal:
        val ordinary = action(Actor("caller"))
        val tick = timer
        val fault = action(Actor("fault")).crash
      trait First extends Facet[Said, Note]:
        var light: Lit
        object first:
          object transitions extends Transitions:
            kind match
              case "ordinary" =>
                on(targets.ordinary):
                  light match
                    case Lit.off => effects.lightUp()
                    case _       => ()
              case "timer" =>
                on(targets.tick):
                  light match
                    case Lit.off => effects.lightUp()
                    case _       => ()
              case "fault" =>
                on(targets.fault):
                  light match
                    case Lit.off => effects.lightUp()
                    case _       => ()
              case _ => disabled(targets.ordinary)
          object effects:
            def lightUp() =
              light = Lit.on
      trait Second extends Facet[Said, Note]:
        var light: Lit
        object second:
          object transitions extends Transitions:
            kind match
              case "ordinary" =>
                on(targets.ordinary):
                  light match
                    case Lit.on => effects.darken()
                    case _      => ()
              case "timer" =>
                on(targets.tick):
                  light match
                    case Lit.on => effects.darken()
                    case _      => ()
              case "fault" =>
                on(targets.fault):
                  light match
                    case Lit.on => effects.darken()
                    case _      => ()
              case _ => disabled(targets.ordinary)
          object effects:
            def darken() =
              light = Lit.off
      object Duplicate extends Machine[Said, Note], First, Second:
        var light = Lit.off
        object transitions extends Facets(first.transitions, second.transitions)
      val actionName =
        if kind == "timer" then "tick" else if kind == "fault" then "fault" else "ordinary"
      val error = intercept[IllegalArgumentException](Duplicate.bindings)
      assertEquals(
        error.getMessage,
        s"$actionName is bound by the facets first and second: an action belongs to one facet, which holds all its transitions"
      )

  for kind <- List("ordinary", "timer", "fault", "disabled") do
    test(s"Facets refuses unowned $kind transitions despite a facet-looking object name"):
      object targets extends Internal:
        val ordinary = action(Actor("caller"))
        val tick = timer
        val fault = action(Actor("fault")).crash
      object Unowned extends Machine[Said, Note]:
        var light = Lit.off
        object pressing:
          object transitions extends Transitions:
            kind match
              case "ordinary" =>
                on(targets.ordinary):
                  light match
                    case _ => effects.stay()
              case "timer" =>
                on(targets.tick):
                  light match
                    case _ => effects.stay()
              case "fault" =>
                on(targets.fault):
                  light match
                    case _ => effects.stay()
              case _ => disabled(targets.ordinary)
        object effects:
          def stay() =
            ()
        object transitions extends Facets(pressing.transitions)
      val error = intercept[IllegalArgumentException](Unowned.bindings)
      assert(
        error.getMessage.contains("pressing.transitions") && error.getMessage.contains(
          "not a facet"
        ),
        error.getMessage
      )
      val expected =
        if kind == "timer" then "tick" else if kind == "fault" then "fault" else "ordinary"
      assert(error.getMessage.contains(s"$expected has no owning facet"), error.getMessage)

  test("distinct namespaces may repeat an effect name; a shared exposed namespace collides"):
    val valid = compileErrors("""trait A extends Facet[Said, Note]:
  object a:
    object effects:
      def settle() = ()
trait B extends Facet[Said, Note]:
  object b:
    object effects:
      def settle() = ()
object Both extends Machine[Said, Note], A, B:
  var light = Lit.off
  object transitions extends Facets()
val _ = Both.a.effects.settle()
val _ = Both.b.effects.settle()
""")
    assertEquals(valid, "")
    val collided = compileErrors("""trait A extends Facet[Said, Note]:
  object shared:
    object effects:
      def settle() = ()
trait B extends Facet[Said, Note]:
  object shared:
    object effects:
      def settle() = ()
object Both extends Machine[Said, Note], A, B:
  var light = Lit.off
  object transitions extends Facets()
val _ = Both.shared.effects.settle()
""")
    assert(
      collided.contains("object shared") && collided.contains(
        "cannot override final member object shared"
      ),
      collided
    )
