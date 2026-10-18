package temporal.migrate

import java.nio.file.{Files, Path}
import scala.meta.*
import scalafix.v1.SyntacticDocument
import scalafix.internal.v1.Rules

object RewriteInputDefaultsMini:
  def main(args: Array[String]): Unit =
    require(args.length == 1, "provide the isolated copied-mini protocol directory")
    val directory = Path.of(args(0)).toRealPath().getParent
    require(directory != Path.of("model/temporal/features/activity_mini").toRealPath())
    val path = directory.resolve("MiniActivity.scala")
    val accepted = Files.readString(path)
    val legacy = accepted
      .replace("(default = Timeout.unset)", "")
      .replace("(default = MaxAttempts.one)", "")
    require(legacy != accepted && legacy.contains("val scheduleToStart = input[Timeout]"))
    val rules = Rules(List(new Fn159Task10InputDefaults))
    def rewrite(source: String) =
      val tree = dialects.Scala3(Input.String(source)).parse[Source].get
      rules.syntacticPatch(SyntacticDocument.fromTree(tree), suppress = false)
    val result = rewrite(legacy)
    Files.writeString(directory.resolve("MiniActivity.defaults.legacy.txt"), legacy): Unit
    Files.writeString(directory.resolve("MiniActivity.defaults.accepted.txt"), accepted): Unit
    Files.writeString(directory.resolve("MiniActivity.defaults.fixed.txt"), result.fixed): Unit
    require(result.diagnostics.isEmpty, result.diagnostics.map(_.message).mkString("; "))
    require(result.fixed == accepted, "actual mini declarations restore exact accepted source")
    val again = rewrite(result.fixed)
    require(again.diagnostics.isEmpty && again.fixed == accepted, "migration must converge")
    val unsafe = legacy + "\nval unknown = input[Timeout]\n"
    val refused = rewrite(unsafe)
    require(refused.diagnostics.nonEmpty && refused.fixed == unsafe, "refusal must be atomic")
    Files.writeString(path, result.fixed): Unit
    println(
      "parsed legacy mini input declarations migrated by the explicit omission inventory; accepted source restored, convergent and atomic refusal"
    )
