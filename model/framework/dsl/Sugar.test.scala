package framework.dsl

class SugarNativeTest extends munit.FunSuite:
  final case class Pair(left: Boolean, right: Boolean)
  enum Fact:
    case changed
    case repeated(count: Int)

  private val left: Pair => Boolean = _.left

  test("never from retains its named argument and native transition meaning"):
    object Checked extends RecordMachine[Boolean, Boolean, Nothing]:
      val init = false
      def end(s: Boolean) = false
      object rules extends Bindings[Boolean, Boolean, Nothing]()
    val property = Checked.property.never(_.state).from(before = identity)
    val holds = property.decl.holds2.get.asInstanceOf[
      (Boolean, Step[Boolean, Boolean, Nothing]) => Boolean
    ] // scalafix:ok DisableSyntax.asInstanceOf
    assertEquals(holds(false, Step(true, true)), true)
    assertEquals(holds(true, Step(true, true)), false)
    assertEquals(holds(true, Step(true, false)), true)
    val press = action(Actor("fixture"))
    val refused = intercept[IllegalArgumentException](Checked.property.when(press).stays(identity))
    assert(refused.getMessage.contains("takes no when"), refused.getMessage)

  test("composition records uses the selected member even when member states and facts coincide"):
    val after = Step(true, Pair(false, false), List("left_changed"))
    assert(after.records(_.left, Fact.changed))
    assert(!after.records(_.right, Fact.changed))
    assert(!after.records(_.left, "other"))
    val repeated = after.copy(facts = List("left_repeated-2"))
    assert(repeated.records(_.left, Fact.repeated(2)))
    assert(!repeated.records(_.right, Fact.repeated(2)))

  test("implication keeps its right side lazy and enter keeps fact order"):
    var reads = 0 // scalafix:ok DisableSyntax.var
    def reached: Boolean =
      reads += 1
      true
    assert(false implies reached)
    assertEquals(reads, 0)
    assert(true implies reached)
    assertEquals(reads, 1)
    given Ok[Boolean] = Ok(true)
    assertEquals(
      enter(false, "second", "first", "second").head.facts,
      List("second", "first", "second")
    )

  @sugar private def forwarded(
      after: Step[Pair, Boolean, String],
      member: Pair => Boolean
  ): Boolean =
    after.records(member, Fact.changed)

  test("plain marked definitions retain strict argument order and by-name laziness"):
    @sugar def ignore(value: Boolean): Boolean = true
    @sugar def conditional(value: Boolean): Boolean = false && value
    @sugar def lazyConditional(value: => Boolean): Boolean = false && value
    val missing = hole
    assertEquals(intercept[HoleReached](ignore(missing.reached)).hole, missing)
    assertEquals(intercept[HoleReached](conditional(missing.reached)).hole, missing)
    assert(!lazyConditional(missing.reached))
    var order = List.empty[Int] // scalafix:ok DisableSyntax.var
    def read(n: Int): Boolean =
      order = order :+ n
      true
    @sugar def swapped(first: Boolean, second: Boolean): Boolean = second && first && first
    assert(swapped(read(1), read(2)))
    assertEquals(order, List(1, 2))

  test("a selector opaque at a plain helper boundary is refused at run time"):
    val after = Step(true, Pair(false, false), List("left_changed"))
    val refused = intercept[IllegalArgumentException](forwarded(after, _.left))
    assert(refused.getMessage.contains("selector capture"), refused.getMessage)
    val local: Pair => Boolean = _.left
    for read <- Seq(
        () => after.records(local, Fact.changed),
        () => after.records(left, Fact.changed)
      )
    do
      val refused = intercept[IllegalArgumentException](read())
      assert(refused.getMessage.contains("selector capture"), refused.getMessage)
