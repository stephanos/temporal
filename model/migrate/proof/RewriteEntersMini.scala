package umpire.migrate

import java.nio.file.{Files, Path}
import scala.meta.*
import scalafix.v1.SyntacticDocument
import scalafix.internal.v1.Rules

object RewriteEntersMini:
  def main(args: Array[String]): Unit =
    require(args.length == 1, "provide the isolated copied-mini protocol directory")
    val directory = Path.of(args(0)).toRealPath()
    require(
      directory != Path.of("model/temporal/features/activity_mini/protocol").toRealPath(),
      "the proof rewrites copied sources only"
    )
    def parse(s: String) = dialects.Scala3(Input.String(s)).parse[Source].get
    val context = List("Faults.scala", "AnswerDispatch.scala").map(n =>
      parse(Files.readString(directory.resolve(n)))
    )
    val path = directory.resolve("Answering.scala")
    val accepted = Files.readString(path)
    val legacy = accepted
      .replace(
        "      def receive",
        "      def backOff(s: AnswerState) = enter(s.copy(phase = AnswerPhase.backingOff))\n      def resume(s: AnswerState) = enter(s.copy(phase = AnswerPhase.waiting))\n      def receive"
      )
      .replace("~> enters(AnswerPhase.backingOff)", "~> effects.backOff")
      .replace("~> enters(AnswerPhase.waiting)", "~> effects.resume")
    require(
      legacy != accepted && legacy.contains("def backOff") && legacy.contains("effects.resume")
    )
    val rules = Rules(List(new Fn159Task7Enters(context)))
    def rewrite(source: String) =
      rules.syntacticPatch(SyntacticDocument.fromTree(parse(source)), suppress = false)
    val result = rewrite(legacy)
    Files.writeString(directory.resolve("Answering.enters.accepted.txt"), accepted): Unit
    Files.writeString(directory.resolve("Answering.enters.legacy.txt"), legacy): Unit
    Files.writeString(directory.resolve("Answering.enters.fixed.txt"), result.fixed): Unit
    require(result.diagnostics.isEmpty, result.diagnostics.map(_.message).mkString("; "))
    require(
      result.fixed == accepted,
      "the real copied mini transformation restores exact accepted bytes"
    )
    require(rewrite(result.fixed).fixed == accepted && rewrite(result.fixed).diagnostics.isEmpty)
    val unsafe = legacy + "\nval exported = AnswerDispatch.answering.effects.backOff\n"
    require(rewrite(unsafe).fixed == unsafe && rewrite(unsafe).diagnostics.nonEmpty)
    Files.writeString(path, result.fixed): Unit
    println(
      "actual Answering backOff/resume transform, convergence and atomic escape refusal; exact accepted source restored"
    )
