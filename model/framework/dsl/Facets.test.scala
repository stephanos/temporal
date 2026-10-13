package framework.dsl

// Facets run as Scala: a machine assembled from facets binds what the same machine written as one
// object binds, an action bound by two facets is refused, and a facet's self-type makes a machine
// without the facet it depends on fail to compile.
object FacetsFixture:
  enum Lit(val status: Note) extends Recorded[Note]:
    case off extends Lit(Note.dark)
    case on extends Lit(Note.lit)
    case broken extends Lit(Note.gone), Terminated

  enum Note:
    case dark, lit, gone

  final case class Lamp(light: Lit, presses: UpTo[1])

  enum Said:
    case ok

  given Ok[Said] = Ok(Said.ok)

  object hand extends Actor:
    val press = action(this)
    val smash = action(this)

  // The facet the others step through: pressing.
  trait Pressing extends Facet[Lamp, Said, Note], Phased[Lamp, Lit]:

    object pressing:

      object effects:
        def toggle(s: Lamp) = enter[Lamp, Said, Note](
          s.copy(light = if s.light.in(Lit.on) then Lit.off else Lit.on, presses = UpTo[1](1))
        )

      object rules extends Rules:
        on(hand.press) {
          when(Lit.off, Lit.on) ~> effects.toggle
        }

  // A facet that depends on Pressing's step.
  trait Breaking extends Facet[Lamp, Said, Note], Phased[Lamp, Lit]:
    this: Pressing =>

    object breaking:

      object effects:
        val smashed = choice
        val toggled = choice
        def smash(s: Lamp) =
          choose(smashed -> enters(Lit.broken)(s), toggled -> pressing.effects.toggle(s))

      object rules extends Rules:
        on(hand.smash) {
          when(Lit.on) ~> effects.smash
          when(Lit.off) ~> enters(Lit.broken)
        }

  object Faceted
      extends Machine[Lamp, Said, Note],
        Phased[Lamp, Lit](_.light, (s, p) => s.copy(light = p)),
        Pressing,
        Breaking:
    val init = Lamp(Lit.off, UpTo(0))

    object rules extends Facets(pressing.rules, breaking.rules)

  object Whole
      extends Machine[Lamp, Said, Note],
        Phased[Lamp, Lit](_.light, (s, p) => s.copy(light = p)):
    val init = Lamp(Lit.off, UpTo(0))

    object effects:
      val smashed = choice
      val toggled = choice
      def toggle(s: Lamp) = enter[Lamp, Said, Note](
        s.copy(light = if s.light.in(Lit.on) then Lit.off else Lit.on, presses = UpTo[1](1))
      )
      def smash(s: Lamp) = choose(smashed -> enters(Lit.broken)(s), toggled -> toggle(s))

    object rules extends Rules:
      on(hand.press) {
        when(Lit.off, Lit.on) ~> effects.toggle
      }
      on(hand.smash) {
        when(Lit.on) ~> effects.smash
        when(Lit.off) ~> enters(Lit.broken)
      }

  // Two facets that both bind the press.
  trait Pressing2 extends Facet[Lamp, Said, Note], Phased[Lamp, Lit]:

    object pressingAgain:

      object rules extends Rules:
        on(hand.press) {
          when(Lit.broken) ~> enters(Lit.broken)
        }

  object Twice
      extends Machine[Lamp, Said, Note],
        Phased[Lamp, Lit](_.light, (s, p) => s.copy(light = p)),
        Pressing,
        Pressing2:
    val init = Lamp(Lit.off, UpTo(0))

    object rules extends Facets(pressing.rules, pressingAgain.rules)

class FacetsTest extends munit.FunSuite:
  import FacetsFixture.*

  def steps(m: Machine[Lamp, Said, Note]): List[(String, Lamp, List[Step[Lamp, Said, Note]])] =
    for
      b <- m.bindings
      s <- Finite[Lamp].values.toList
    yield
      val f = b.function.asInstanceOf[Lamp => List[
        Step[Lamp, Said, Note]
      ]] // scalafix:ok DisableSyntax.asInstanceOf
      (b.decl.toString, s, f(s))

  test("a machine assembled from facets binds what the machine written as one object binds") {
    assertEquals(Faceted.bindings.map(_.decl), Whole.bindings.map(_.decl))
    assertEquals(steps(Faceted), steps(Whole))
    assert(steps(Faceted).exists(_._3.nonEmpty))
  }

  test("an action bound by two facets is refused as the rules construct, naming both") {
    val refused = intercept[IllegalArgumentException](Twice.bindings)
    assertEquals(
      refused.getMessage,
      "press is bound by the facets pressing and pressingAgain: an action belongs to one facet, " +
        "which holds all its rules"
    )
  }

  test("a machine that mixes in a facet without the facet it depends on does not compile") {
    val errors = compileErrors(
      "object Lone extends Machine[Lamp, Said, Note], Phased[Lamp, Lit](_.light), Breaking:\n" +
        "  val init = Lamp(Lit.off, UpTo(0))\n" +
        "  object rules extends Facets(breaking.rules)"
    )
    assert(errors.contains("Pressing"), errors)
  }

  for kind <- List("ordinary", "timer", "fault", "disabled") do
    test(s"duplicate $kind owners name the actual action and both facets"):
      object targets:
        val ordinary = action(Actor("caller"))
        val tick = timer
        val fault = action(Actor("fault"))
      trait First extends Facet[Lamp, Said, Note], Phased[Lamp, Lit]:
        object first:
          object rules extends Rules:
            kind match
              case "ordinary" => on(targets.ordinary)(when(Lit.off) ~> enters(Lit.on))
              case "timer"    => on(targets.tick)(when(Lit.off) ~> enters(Lit.on))
              case "fault"    => on(targets.fault)(when(Lit.off) ~> enters(Lit.on))
              case _          => disabled(targets.ordinary)
      trait Second extends Facet[Lamp, Said, Note], Phased[Lamp, Lit]:
        object second:
          object rules extends Rules:
            kind match
              case "ordinary" => on(targets.ordinary)(when(Lit.on) ~> enters(Lit.off))
              case "timer"    => on(targets.tick)(when(Lit.on) ~> enters(Lit.off))
              case "fault"    => on(targets.fault)(when(Lit.on) ~> enters(Lit.off))
              case _          => disabled(targets.ordinary)
      object Duplicate
          extends Machine[Lamp, Said, Note],
            Phased[Lamp, Lit](_.light, (s, p) => s.copy(light = p)),
            First,
            Second:
        val init = Lamp(Lit.off, UpTo(0))
        object rules extends Facets(first.rules, second.rules)
      val actionName =
        if kind == "timer" then "tick" else if kind == "fault" then "fault" else "ordinary"
      val error = intercept[IllegalArgumentException](Duplicate.bindings)
      assertEquals(
        error.getMessage,
        s"$actionName is bound by the facets first and second: an action belongs to one facet, which holds all its rules"
      )

  for kind <- List("ordinary", "timer", "fault", "disabled") do
    test(s"Facets refuses unowned $kind rules despite a facet-looking object name"):
      object targets:
        val ordinary = action(Actor("caller"))
        val tick = timer
        val fault = action(Actor("fault"))
      object Unowned extends Machine[Lamp, Said, Note], Phased[Lamp, Lit](_.light):
        val init = Lamp(Lit.off, UpTo(0))
        object pressing:
          object rules extends Rules:
            kind match
              case "ordinary" =>
                on(targets.ordinary)(always ~> ((s: Lamp) => stay[Lamp, Said, Note](s)))
              case "timer" => on(targets.tick)(always ~> ((s: Lamp) => stay[Lamp, Said, Note](s)))
              case "fault" => on(targets.fault)(always ~> ((s: Lamp) => stay[Lamp, Said, Note](s)))
              case _       => disabled(targets.ordinary)
        object rules extends Facets(pressing.rules)
      val error = intercept[IllegalArgumentException](Unowned.bindings)
      assert(
        error.getMessage.contains("pressing.rules") && error.getMessage.contains("not a facet"),
        error.getMessage
      )
      val expected =
        if kind == "timer" then "tick" else if kind == "fault" then "fault" else "ordinary"
      assert(error.getMessage.contains(s"$expected has no owning facet"), error.getMessage)

  test("distinct namespaces may repeat an effect name; a shared exposed namespace collides"):
    val valid = compileErrors("""trait A extends Facet[Lamp, Said, Note]:
  object a:
    object effects:
      def settle(s: Lamp) = enter[Lamp, Said, Note](s)
trait B extends Facet[Lamp, Said, Note]:
  object b:
    object effects:
      def settle(s: Lamp) = enter[Lamp, Said, Note](s)
object Both extends Machine[Lamp, Said, Note], A, B:
  val init = Lamp(Lit.off, UpTo(0))
  def end(s: State) = true
  object rules extends Facets()
val _ = Both.a.effects.settle(Both.init)
val _ = Both.b.effects.settle(Both.init)
""")
    assertEquals(valid, "")
    val collided = compileErrors("""trait A extends Facet[Lamp, Said, Note]:
  object shared:
    object effects:
      def settle(s: Lamp) = enter[Lamp, Said, Note](s)
trait B extends Facet[Lamp, Said, Note]:
  object shared:
    object effects:
      def settle(s: Lamp) = enter[Lamp, Said, Note](s)
object Both extends Machine[Lamp, Said, Note], A, B:
  val init = Lamp(Lit.off, UpTo(0))
  def end(s: State) = true
  object rules extends Facets()
val _ = Both.shared.effects.settle(Both.init)
""")
    assert(
      collided.contains("object shared") && collided.contains(
        "cannot override final member object shared"
      ),
      collided
    )
