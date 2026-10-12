package umpire.migrate

import java.nio.file.{Files, Path}
import scala.meta.*
import scalafix.v1.SyntacticDocument
import scalafix.internal.v1.Rules

object RewriteCentralBoundsMini:
  def main(args: Array[String]): Unit =
    require(args.length == 1, "provide the isolated copied-mini protocol directory")
    val directory = Path.of(args(0)).toRealPath()
    require(
      directory != Path.of("model/temporal/features/activity_mini/protocol").toRealPath(),
      "the source proof must not rewrite the authored mini"
    )
    val path = directory.resolve("Queue.scala")
    val accepted = Files.readString(path)
    val target = "query verify properties.safety.deliveredWasScheduled in delivery limits three"
    val legacy = accepted.replace(
      target,
      "{ val local = Limits(steps = 3, actions = 3, search = 4096); " +
        target.replace("limits three", "limits local") + " }"
    )
    require(
      legacy != accepted && legacy.contains("limits local"),
      "exercise actual local Bounds migration"
    )
    val rules = Rules(List(new Fn159Task5CentralBounds))
    def rewrite(source: String) =
      val tree = dialects.Scala3(Input.String(source)).parse[Source].get
      rules.syntacticPatch(SyntacticDocument.fromTree(tree), suppress = false)
    val result = rewrite(legacy)
    Files.writeString(directory.resolve("Queue.bounds.accepted.txt"), accepted): Unit
    Files.writeString(directory.resolve("Queue.bounds.legacy.txt"), legacy): Unit
    Files.writeString(directory.resolve("Queue.bounds.central.txt"), result.fixed): Unit
    require(result.diagnostics.isEmpty, result.diagnostics.map(_.message).mkString("; "))
    require(result.fixed == accepted, "restore complete accepted source bytes and coordinates")
    val again = rewrite(result.fixed)
    require(
      again.diagnostics.isEmpty && again.fixed == accepted,
      "central Bounds migration must converge"
    )
    val unknown = legacy.replace("search = 4096", "search = 4097")
    val refused = rewrite(unknown)
    require(
      refused.diagnostics.nonEmpty && refused.fixed == unknown,
      "unknown budgets must refuse atomically"
    )
    Files.writeString(path, result.fixed): Unit
    println(
      "central Bounds: transformed local declaration/use, converged and refused unknown budget; exact Queue.scala restored"
    )
