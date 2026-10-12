package umpire.migrate

import java.nio.file.{Files, Path}
import scala.meta.*
import scalafix.v1.SyntacticDocument
import scalafix.internal.v1.Rules

object RewriteQualifiedPhasesMini:
  def main(args: Array[String]): Unit =
    require(args.length == 1, "provide the isolated copied-mini protocol directory")
    val directory = Path.of(args(0)).toRealPath()
    val original = Path.of("model/temporal/features/activity_mini/protocol").toRealPath()
    require(directory != original, "the source proof must not rewrite the authored mini")
    val path = directory.resolve("Dispatching.scala")
    val accepted = Files.readString(path)
    val legacy = accepted
      .replace("\n\nimport framework.*", "\nimport Phase.{scheduled as ready}\nimport framework.*")
      .replace("Phase.scheduled", "ready")
    require(
      legacy != accepted && legacy.contains("when(ready)"),
      "the copied mini must exercise phase qualification"
    )
    val rules = Rules(List(new Fn159Task3QualifiedPhases))
    def rewrite(source: String) =
      val tree = dialects.Scala3(Input.String(source)).parse[Source].get
      rules.syntacticPatch(SyntacticDocument.fromTree(tree), suppress = false)
    val result = rewrite(legacy)
    Files.writeString(directory.resolve("Dispatching.phases.accepted.txt"), accepted): Unit
    Files.writeString(directory.resolve("Dispatching.phases.legacy.txt"), legacy): Unit
    Files.writeString(directory.resolve("Dispatching.phases.qualified.txt"), result.fixed): Unit
    require(result.diagnostics.isEmpty, result.diagnostics.map(_.message).mkString("; "))
    require(
      result.fixed == accepted,
      "phase qualification must restore every accepted source coordinate and identity"
    )
    val again = rewrite(result.fixed)
    require(
      again.diagnostics.isEmpty && again.fixed == accepted,
      "phase qualification must converge"
    )
    val ambiguous = legacy.replace(
      "import Phase.{scheduled as ready}",
      "import Phase.{scheduled as ready}\nimport Other.ready"
    )
    val refused = rewrite(ambiguous)
    require(
      refused.diagnostics.nonEmpty && refused.fixed == ambiguous,
      "ambiguous phase imports must refuse without partial rewriting"
    )
    Files.writeString(path, result.fixed): Unit
    println(
      "qualified phases, idempotent and ambiguity-refused: Dispatching.scala; exact accepted source restored"
    )
