package temporal.migrate

import java.nio.file.{Files, Path}
import scala.meta.*
import scalafix.v1.SyntacticDocument
import scalafix.internal.v1.Rules

object RewriteMini:
  def main(args: Array[String]): Unit =
    require(args.length == 1, "provide the isolated copied-mini protocol directory")
    val directory = Path.of(args(0)).toRealPath()
    val original = Path.of("model/temporal/features/activity_mini/protocol").toRealPath()
    require(directory != original, "the source proof must not rewrite the authored mini")
    val rules = Rules(List(new Fn166Task9QueueAlias))
    val results = List("Queue.scala", "QueueScheduled.scala").map { name =>
      val path = directory.resolve(name)
      val source = Files.readString(path)
      val tree = dialects.Scala3(Input.String(source)).parse[Source].get
      val result = rules.syntacticPatch(SyntacticDocument.fromTree(tree), suppress = false)
      require(result.diagnostics.isEmpty, result.diagnostics.map(_.message).mkString("; "))
      require(result.fixed.length == source.length, "source coordinates must be preserved")
      require(result.fixed != source, s"$name did not exercise the rename")
      val second = dialects.Scala3(Input.String(result.fixed)).parse[Source].get
      val again = rules.syntacticPatch(SyntacticDocument.fromTree(second), suppress = false)
      require(again.diagnostics.isEmpty && again.fixed == result.fixed, "the rule must converge")
      path -> result.fixed
    }
    results.foreach { (path, source) =>
      Files.writeString(path, source): Unit
      println(s"renamed and idempotent: ${path.getFileName}")
    }
