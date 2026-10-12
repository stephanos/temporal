package umpire.migrate

import java.nio.file.{Files, Path}
import scala.meta.*
import scalafix.v1.SyntacticDocument
import scalafix.internal.v1.Rules

class FamilyNamesTest extends munit.FunSuite:
  private val rules = Rules(List(new Fn160Task3FamilyNames))
  private def rewrite(source: String) =
    val tree = dialects.Scala3(Input.String(source)).parse[Source].get
    rules.syntacticPatch(SyntacticDocument.fromTree(tree), suppress = false)
  // The legacy source: each identifier token of a new name spelled with its old one; comments and
  // literals keep their text (the sweep updates those by hand).
  private def legacy(source: String) =
    val back = (Fn160Task3FamilyNames.shared ++ Fn160Task3FamilyNames.mini).map(_.swap)
    dialects.Scala3(Input.String(source)).tokenize.get.toList.map {
      case t: Token.Ident if back.contains(t.value) => back(t.value)
      case t                                        => t.syntax
    }.mkString
  private val mini = Path.of("model/temporal/features/activity_mini/protocol")
  private val queue = Path.of("model/temporal/foundations/taskqueue/protocol")

  test("the committed renamed sources are restored from their legacy names and converge"):
    val files = List("AnswerTrusting.scala", "QueueScheduled.scala", "DispatchingProgress.scala",
      "Queue.scala").map(mini.resolve) ++
      List("QueueForgetful.scala", "QueueLossyMatching.scala", "QueueVolatile.scala").map(queue.resolve)
    for path <- files do
      val accepted = Files.readString(path)
      val old = legacy(accepted)
      assertNotEquals(old, accepted, path.toString)
      val result = rewrite(old)
      assertEquals(result.diagnostics, Nil, path.toString)
      assertEquals(result.fixed, accepted, path.toString)
      assertEquals(rewrite(accepted).fixed, accepted, path.toString)

  test("comments and strings keep their text; the mini's names stay outside its package"):
    val source = "package temporal.features.activity.standalone.protocol\n\n" +
      "// ScheduledQueue and ForgetfulQueue\nval text = \"ForgetfulQueue\"\n" +
      "val a = ScheduledQueue\nval b = ForgetfulQueue\n"
    assertEquals(
      rewrite(source).fixed,
      source.replace("val b = ForgetfulQueue", "val b = QueueForgetful")
    )

  test("an import of a shared provider renames with its uses, as a real feature's does"):
    val source = "package temporal.features.activity.standalone.protocol\n\n" +
      "import foundations.taskqueue.protocol.ForgetfulQueue\n" +
      "import foundations.taskqueue.protocol.{LossyMatchingQueue, VolatileQueue as Volatile}\n" +
      "val a = ForgetfulQueue\nval b = LossyMatchingQueue\nval c = Volatile\n"
    val result = rewrite(source)
    assertEquals(result.diagnostics, Nil)
    assertEquals(
      result.fixed,
      source.replace("ForgetfulQueue", "QueueForgetful").replace("LossyMatchingQueue", "QueueLossyMatching")
        .replace("VolatileQueue", "QueueVolatile")
    )

  test("a name already in use refuses the whole document"):
    val source = "package temporal.features.activity.mini.protocol\n\n" +
      "val QueueScheduled = 1\nval a = ScheduledQueue\n"
    val result = rewrite(source)
    assertEquals(result.fixed, source)
    assertEquals(result.diagnostics.size, 1)
    assert(result.diagnostics.head.message.contains("QueueScheduled is already named here"))
