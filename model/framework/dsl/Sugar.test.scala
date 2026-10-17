package framework.dsl

class SugarNativeTest extends munit.FunSuite:
  final case class Pair(left: Boolean, right: Boolean)
  enum Fact:
    case changed
    case repeated(count: Int)

  test("a composition declares member fact cases and refuses an untagged member fact"):
    assertEquals(
      compileErrors("""object Tagged extends Composition[Pair]() {
        enum Fact { case left(value: SugarNativeTest.this.Fact); case right(value: SugarNativeTest.this.Fact) }
        val claim = witness(own(_.left, action("move", Actor("tester")))).records(Fact.left(SugarNativeTest.this.Fact.changed))
      }"""),
      ""
    )

  test("a derived composition retains the source's tagged fact type"):
    assertEquals(
      compileErrors("""enum Note { case changed }
      object Member extends RecordMachine[Boolean, Boolean, Note] {
        def init = false
        def end(state: Boolean) = state
        object transitions extends Rules
      }
      object Source extends Composition[Pair](_.left -> Member, _.right -> Member) {
        enum Fact { case left(value: Note); case right(value: Note) }
      }
      object Derived extends DerivedComposition(Source.withMember(_.left -> Member)) {
        val claim = witness(own(_.left, action("move", Actor("tester")))).records(Source.Fact.left(Note.changed))
      }
      val replacement = Source.withMember(_.left -> Member)
      val claim = replacement.witness(replacement.own(_.left, action("move", Actor("tester")))).records(Source.Fact.left(Note.changed))
      """),
      ""
    )

  test("never from retains its named argument and native transition meaning"):
    object Checked extends RecordMachine[Boolean, Boolean, Nothing]:
      val init = false
      def end(s: Boolean) = false
      object transitions extends Bindings[Boolean, Boolean, Nothing]()
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

  test("tagged member facts retain member identity and fact payloads"):
    enum Tagged:
      case left(value: Fact)
      case right(value: Fact)
    val after = Step(true, Pair(false, false), List(Tagged.left(Fact.changed)))
    assert(after.records(Tagged.left(Fact.changed)))
    assert(!after.records(Tagged.right(Fact.changed)))
    val repeated = after.copy(facts = List(Tagged.left(Fact.repeated(2))))
    assert(repeated.records(Tagged.left(Fact.repeated(2))))
    assert(!repeated.records(Tagged.left(Fact.repeated(3))))
    assert(!repeated.records(Tagged.right(Fact.repeated(2))))
    assert(compileErrors("after.records(Fact.changed)").nonEmpty)

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

  test("plain marked definitions retain strict argument order and by-name laziness"):
    inline def ignore(value: Boolean): Boolean = true
    inline def conditional(value: Boolean): Boolean = false && value
    inline def lazyConditional(value: => Boolean): Boolean = false && value
    val missing = hole
    assertEquals(intercept[HoleReached](ignore(missing.reached)).hole, missing)
    assertEquals(intercept[HoleReached](conditional(missing.reached)).hole, missing)
    assert(!lazyConditional(missing.reached))
    var order = List.empty[Int] // scalafix:ok DisableSyntax.var
    def read(n: Int): Boolean =
      order = order :+ n
      true
    inline def swapped(first: Boolean, second: Boolean): Boolean = second && first && first
    assert(swapped(read(1), read(2)))
    assertEquals(order, List(1, 2))
