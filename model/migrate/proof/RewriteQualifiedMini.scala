package umpire.migrate

import java.nio.file.{Files, Path}
import scala.meta.*
import scalafix.v1.SyntacticDocument
import scalafix.internal.v1.Rules

object RewriteQualifiedMini:
  def main(args: Array[String]): Unit =
    require(args.length == 1, "provide the isolated copied-mini protocol directory")
    val directory = Path.of(args(0)).toRealPath()
    val original = Path.of("model/temporal/features/activity_mini/protocol").toRealPath()
    require(directory != original, "the source proof must not rewrite the authored mini")
    val path = directory.resolve("Dispatching.scala")
    val accepted = Files.readString(path)
    val legacy = accepted
      .replace("\n\nimport framework.*", "\nimport activityWorker.poll\nimport framework.*")
      .replace("on(activityWorker.poll)", "on(poll)")
    require(
      legacy != accepted && legacy.contains("on(poll)"),
      "the copied mini must exercise qualification"
    )
    val rules = Rules(List(new Fn159Task1QualifiedActions))
    def rewrite(source: String) =
      val tree = dialects.Scala3(Input.String(source)).parse[Source].get
      rules.syntacticPatch(SyntacticDocument.fromTree(tree), suppress = false)
    val result = rewrite(legacy)
    Files.writeString(directory.resolve("Dispatching.accepted.txt"), accepted): Unit
    Files.writeString(directory.resolve("Dispatching.legacy.txt"), legacy): Unit
    Files.writeString(directory.resolve("Dispatching.qualified.txt"), result.fixed): Unit
    require(result.diagnostics.isEmpty, result.diagnostics.map(_.message).mkString("; "))
    require(
      result.fixed == accepted,
      "qualified copied-mini source must retain every original coordinate and identity"
    )
    val again = rewrite(result.fixed)
    require(again.diagnostics.isEmpty && again.fixed == accepted, "qualification must converge")
    val ambiguous =
      legacy.replace("import activityWorker.poll", "import activityWorker.poll\nimport other.poll")
    val refused = rewrite(ambiguous)
    require(
      refused.diagnostics.nonEmpty && refused.fixed == ambiguous,
      "ambiguous imports must refuse without partial rewriting"
    )
    Files.writeString(path, result.fixed): Unit
    println(
      "qualified, idempotent and ambiguity-refused: Dispatching.scala; exact accepted source restored"
    )
