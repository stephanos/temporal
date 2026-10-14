package temporal.migrate

import scala.meta.*
import scalafix.v1.SyntacticDocument
import scalafix.internal.v1.Rules

class CapabilityNamesTest extends munit.FunSuite:
  private val rules = Rules(List(new Fn159Task8CapabilityNames))
  private def rewrite(source: String) =
    val tree = dialects.Scala3(Input.String(source)).parse[Source].get
    rules.syntacticPatch(SyntacticDocument.fromTree(tree), suppress = false)

  test("owned imports, constructors, types, overrides and bounds migrate and converge"):
    val source = """package example
import temporal.capabilities.system.{Retries, Deadline}
val retryable: Retries[S, P, 2] = Retries[S, P, 2](failure)
val scheduleDeadline = Deadline[S, P, Waiting, Fact](timer)
val waiver = overriding(Retries.failureReturnsToWaiting[S, P] -> own, because = "Retries Deadline")
val budgets = capabilities.bound(three, Deadline.firesInWindow[S, P, Waiting] -> two)
// Retries Deadline scheduleDeadline
val text = "Retries Deadline"
val other = unrelated.Deadline
"""
    val expected = """package example
import temporal.capabilities.system.{Retryable, Expirable}
val retryable: Retryable[S, P, 2] = Retryable[S, P, 2](failure)
val scheduleDeadline = Expirable[S, P, Waiting, Fact](timer)
val waiver = overriding(Retryable.failureReturnsToWaiting[S, P] -> own, because = "Retries Deadline")
val budgets = capabilities.bound(three, Expirable.firesInWindow[S, P, Waiting] -> two)
// Retries Deadline scheduleDeadline
val text = "Retries Deadline"
val other = unrelated.Deadline
"""
    val result = rewrite(source)
    assertEquals(result.diagnostics, Nil)
    assertEquals(result.fixed, expected)
    assertEquals(rewrite(expected).fixed, expected)
    assertEquals(rewrite(expected).diagnostics, Nil)

  test("explicit owned declarations and fully qualified member imports migrate"):
    val source = """package temporal.capabilities.system
import framework.*
final case class Retries[S](field: S) extends CapabilityOf[S, Nothing, Nothing]
object Retries extends CapabilityKind:
  def law = Retries(1)
final case class Deadline[S](field: S) extends CapabilityOf[S, Nothing, Nothing]
object Deadline extends CapabilityKind
"""
    val expected = source.replace("Retries", "Retryable").replace("Deadline", "Expirable")
    assertEquals(rewrite(source).fixed, expected)
    val imported = """package example
import temporal.capabilities.system.Retries.failureEndsFailed
val law = temporal.capabilities.system.Deadline.firesInWindow[S, P, Waiting]
"""
    assertEquals(
      rewrite(imported).fixed,
      imported.replace("Retries", "Retryable").replace("Deadline", "Expirable")
    )

  test("ambiguity, shadowing and unsupported ownership refuse atomically with manual remedy"):
    val prefix = "package example\nimport temporal.capabilities.system.{Retries, Deadline}\n"
    val bodies = List(
      "import other.Retries\nval x = Deadline(timer)",
      "def f(Retries: Int) = Retries\nval x = Deadline(timer)",
      "type Deadline = Int\nval x = Retries(failure)",
      "val Retryable = 1\nval x = Deadline(timer)",
      "import other.*\nval x = Retries(failure)",
      "object nested { import temporal.capabilities.system.Retries; val x = Retries(failure) }"
    )
    val unsupported = List(
      "package example\nimport temporal.capabilities.system.*\nval x = Retries(failure)",
      "package example\nimport temporal.capabilities.system.{Retries as R}\nval x = R(failure)",
      "package temporal.capabilities.system\nclass Retries\nobject Retries",
      "package temporal.capabilities.system\nimport other.{CapabilityOf, CapabilityKind}\n" +
        "case class Retries[S](s: S) extends CapabilityOf[S, Nothing, Nothing]\nobject Retries extends CapabilityKind",
      "package temporal.capabilities.system\nimport framework.*\nimport other.{Base as CapabilityKind}\n" +
        "case class Retries[S](s: S) extends CapabilityOf[S, Nothing, Nothing]\nobject Retries extends CapabilityKind",
      "package example\nval temporal = other\nval x = temporal.capabilities.system.Retries(failure)"
    )
    for source <- bodies.map(prefix + _) ++ unsupported do
      val result = rewrite(source)
      assertEquals(result.fixed, source)
      assertEquals(result.diagnostics.size, 1, source)
      assert(result.diagnostics.head.message.contains("manually resolve"))

  test("unrelated declarations and imports, comments, strings and deadline vocabulary stay intact"):
    val source = """package unrelated
import other.{Retries, Deadline}
object Deadline
val x: Retries = Deadline
val scheduleDeadline = deadline.scheduleToStart
val timeout: Timeout = Timeout.expires
// Retries Deadline
val text = "Retries Deadline"
"""
    assertEquals(rewrite(source).fixed, source)
    assertEquals(rewrite(source).diagnostics, Nil)

  test("an import cannot establish ownership before itself or outside its lexical package"):
    val sources = List(
      "package example\nval earlier = Retries(failure)\nimport temporal.capabilities.system.Retries\nval later = Retries(failure)",
      "package first { import temporal.capabilities.system.Retries; val x = Retries(failure) }\n" +
        "package second { val y = Retries(failure) }"
    )
    for source <- sources do
      val result = rewrite(source)
      assertEquals(result.fixed, source)
      assertEquals(result.diagnostics.size, 1)
