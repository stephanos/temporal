package umpire.migrate

import java.nio.file.{Files, Path}
import scala.meta.*
import scalafix.v1.SyntacticDocument
import scalafix.internal.v1.Rules

object RewriteCapabilityNamesMini:
  def main(args: Array[String]): Unit =
    require(args.length == 1, "provide the isolated copied-mini protocol directory")
    val directory = Path.of(args(0)).toRealPath()
    require(
      directory != Path.of("model/temporal/features/activity_mini/protocol").toRealPath(),
      "the proof rewrites copied sources only"
    )
    val path = directory.resolve("Completion.scala")
    val accepted = Files.readString(path)
    val legacy = accepted.replace("Retryable", "Retries").replace("Expirable", "Deadline")
    require(
      legacy != accepted && legacy.contains("overriding(Retries.") &&
        legacy.contains("val scheduleDeadline: Capability = Deadline[")
    )
    val rules = Rules(List(new Fn159Task8CapabilityNames))
    def rewrite(source: String) =
      val tree = dialects.Scala3(Input.String(source)).parse[Source].get
      rules.syntacticPatch(SyntacticDocument.fromTree(tree), suppress = false)
    val result = rewrite(legacy)
    Files.writeString(directory.resolve("Completion.capability.accepted.txt"), accepted): Unit
    Files.writeString(directory.resolve("Completion.capability.legacy.txt"), legacy): Unit
    Files.writeString(directory.resolve("Completion.capability.fixed.txt"), result.fixed): Unit
    require(result.diagnostics.isEmpty, result.diagnostics.map(_.message).mkString("; "))
    require(
      result.fixed == accepted,
      "actual mini capability migration restores exact source bytes"
    )
    val again = rewrite(result.fixed)
    require(again.diagnostics.isEmpty && again.fixed == accepted, "migration must converge")
    val ambiguous = legacy + "\nimport other.Retries\n"
    val refused = rewrite(ambiguous)
    require(refused.diagnostics.nonEmpty && refused.fixed == ambiguous, "refusal must be atomic")
    Files.writeString(path, result.fixed): Unit
    println(
      "actual Completion imports, bindings and retry override migrated; exact accepted source restored; convergent and atomic refusal"
    )
