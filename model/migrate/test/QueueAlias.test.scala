package umpire.migrate

import java.nio.file.{Files, Path}
import scala.meta.*
import scalafix.v1.SyntacticDocument
import scalafix.internal.v1.Rules

class QueueAliasTest extends munit.FunSuite:
  private val rules = Rules(List(new Fn166Task9QueueAlias))
  private def rewrite(source: String) =
    val tree = dialects.Scala3(Input.String(source)).parse[Source].get
    rules.syntacticPatch(SyntacticDocument.fromTree(tree), suppress = false)

  private val imported = "import foundations.taskqueue.{QueueView, Outstanding, queue as foundationQueue}"
  private val prefix = "package temporal\npackage features.activity\npackage mini\npackage protocol\n\n"

  test("both committed mini imports and references rename exactly and converge"):
    for name <- List("Queue.scala", "ScheduledQueue.scala") do
      val path = Path.of("model/temporal/features/activity_mini/protocol", name)
      val source = Files.readString(path)
      val result = rewrite(source)
      assertEquals(result.diagnostics, Nil)
      assertEquals(result.fixed, source.replace("foundationQueue", "taskQueueSource"))
      assertNotEquals(result.fixed, source)
      assertEquals(result.fixed.length, source.length)
      assertEquals(rewrite(result.fixed).fixed, result.fixed)

  test("ambiguous aliases refuse the whole document without rewriting"):
    val cases = List(
      "val taskQueueSource = 1\nval x = foundationQueue.deliver",
      "def f(foundationQueue: Any) = foundationQueue.deliver",
      "val x = foundationQueue",
      "import other.{queue as foundationQueue}\nval x = foundationQueue.deliver"
    )
    for body <- cases do
      val source = prefix + imported + "\n" + body
      val result = rewrite(source)
      assertEquals(result.fixed, source)
      assertEquals(result.diagnostics.size, 1)
      assert(result.diagnostics.head.message.contains("ambiguous queue alias"))

  test("unrelated scopes, literals and comments are retained"):
    val source = prefix + imported + "\n// foundationQueue\nval text = \"foundationQueue\"\nval x = foundationQueue.deliver"
    val result = rewrite(source)
    assertEquals(result.fixed, prefix + imported.replace("foundationQueue", "taskQueueSource") +
      "\n// foundationQueue\nval text = \"foundationQueue\"\nval x = taskQueueSource.deliver")
    for unrelated <- List(source.replace("package mini", "package standalone"),
        source.replace("foundations.taskqueue", "other.taskqueue")) do
      assertEquals(rewrite(unrelated).fixed, unrelated)
