package temporal.migrate

import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*
import scala.meta.*
import scalafix.v1.SyntacticDocument
import scalafix.internal.v1.Rules

// fn-169.1 on a copied mini (framework/migrate/prove.sh): every Model source of the copy, the mini's
// and the shared code's, is spelled back to the legacy `framework` package, `framework.dsl.X` to
// `framework.X`; then the rule runs over each and must restore every accepted source byte for
// byte, converge, and leave the framework's other packages and the copy's framework alone.
object RewriteDslPackageMini:
  def main(args: Array[String]): Unit =
    require(args.length == 1, "provide the isolated copied-mini protocol directory")
    val protocol = Path.of(args(0)).toRealPath()
    require(
      protocol != Path.of("model/temporal/features/activity_mini/protocol").toRealPath(),
      "the source proof must not rewrite the authored mini"
    )
    val temporal = protocol.getParent.getParent.getParent
    require(temporal.getFileName.toString == "temporal", s"$temporal is no copied model/temporal")
    val sources = OnLegacyDsl.sources(temporal)
    val accepted = sources.map(p => p -> Files.readString(p)).toMap
    val rules = Rules(List(new Fn169Task1DslPackage))
    var spelled = 0
    for (path, text) <- accepted do
      val old = OnLegacyDsl.legacy(text)
      if old != text then spelled += 1
      val tree = dialects.Scala3(Input.String(old)).parse[Source].get
      val result = rules.syntacticPatch(SyntacticDocument.fromTree(tree), suppress = false)
      val refusals = result.diagnostics.filterNot(_.message.startsWith("Unused Scalafix suppression"))
      require(refusals.isEmpty, s"$path: ${refusals.map(_.message).mkString("; ")}")
      require(result.fixed == text, s"$path: the rule does not restore the accepted source")
      val again = rules.syntacticPatch(
        SyntacticDocument.fromTree(dialects.Scala3(Input.String(result.fixed)).parse[Source].get),
        suppress = false
      )
      require(again.fixed == result.fixed, s"$path: the rule does not converge")
      Files.writeString(path, result.fixed): Unit
    require(spelled > 0, "no source of the copy names the DSL through framework")
    println(
      s"spelled $spelled of ${accepted.size} Model sources back to package framework; the rule " +
        "restored every one byte for byte and converges"
    )
