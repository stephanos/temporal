package umpire.migrate

import scala.meta.*
import scalafix.v1.SyntacticDocument
import scalafix.internal.v1.Rules

class BoundedIntegersTest extends munit.FunSuite:
  private val rules = Rules(List(new Fn159Task9BoundedIntegers))
  private def rewrite(source: String) =
    val tree = dialects.Scala3(Input.String(source)).parse[Source].get
    rules.syntacticPatch(SyntacticDocument.fromTree(tree), suppress = false)

  val legacy = """package fixture.probe
import framework.*
import framework.Finite.given
final case class State(n: Int)
given Finite[State] =
  given Finite[Int] = Finite.upTo(2)
  Finite.derived
val start = State(0)
def next(s: State) = s.copy(n = (s.n + 1).min(2))
"""
  val accepted = """package fixture.probe
import framework.*
import framework.Finite.given
final case class State(n: UpTo[2])
given Finite[State] =
  Finite.derived
val start = State(UpTo[2](0))
def next(s: State) = s.copy(n = UpTo[2]((s.n + 1).min(2)))
"""

  test("bounded field, constructors, copy writes and retired local given migrate together"):
    val result = rewrite(legacy)
    assertEquals(result.diagnostics, Nil)
    assertEquals(result.fixed, accepted)
    assertEquals(rewrite(result.fixed).fixed, accepted)
    assertEquals(rewrite(result.fixed).diagnostics, Nil)

  test("a bounded channel, its Inbox and sent integer migrate together"):
    val source = """import framework.*
val wire: Channel[Int] = channel[Int](capacity = 1, order = Order.fifo, loss = Loss.reliable)(using Finite.upTo(3))
final case class Mail(inbox: Inbox[Int])
given Finite[Mail] =
  given Finite[Inbox[Int]] = wire.contents
  Finite.derived
def send(s: Mail) = s.copy(inbox = s.inbox.send(2))
"""
    val result = rewrite(source)
    assertEquals(result.diagnostics, Nil)
    assert(result.fixed.contains("Channel[UpTo[3]]"))
    assert(result.fixed.contains("channel[UpTo[3]]"))
    assert(!result.fixed.contains("using Finite.upTo"))
    assert(result.fixed.contains("Inbox[UpTo[3]]"))
    assert(result.fixed.contains("send(UpTo[3](2))"))
    assertEquals(rewrite(result.fixed).fixed, result.fixed)

  test("unknown bounds, ambiguous scopes and unsafe writes refuse the whole document"):
    for source <- List(
      legacy.replace("Finite.upTo(2)", "Finite.upTo(limit)"),
      legacy.replace("Finite.upTo(2)", "Finite.upTo(-1)"),
      legacy.replace("import framework.*", "import framework.*\nimport other.*"),
      legacy.replace("val start", "object UpTo {}\nval start"),
      legacy.replace("(s.n + 1).min(2)", "unsafe(s.n)"),
      legacy + "val escaped = State.apply _\n",
      legacy.replace("given Finite[Int]", "given a: Finite[Int]") .replace("  Finite.derived", "  given b: Finite[Int] = Finite.upTo(3)\n  Finite.derived")
    ) do
      val result = rewrite(source)
      assertEquals(result.fixed, source)
      assert(result.diagnostics.nonEmpty)
      assert(result.diagnostics.forall(_.message.contains("manually")))

  test("named opaque ranges, helper Ints and already current source stay intact"):
    val source = """import framework.*
object bounded:
  opaque type Retries = Int
  object Retries:
    given Finite[Retries] = Finite.upTo(2)
def widen(n: Int): Int = n
final case class Current(n: UpTo[2])
"""
    assertEquals(rewrite(source).fixed, source)
    assertEquals(rewrite(source).diagnostics, Nil)
