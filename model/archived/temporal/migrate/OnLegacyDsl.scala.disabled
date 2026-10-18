package temporal.migrate

import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*
import scala.meta.*
import scalafix.v1.SyntacticDocument
import scalafix.internal.v1.Rules

// A rule written before fn-169.1 reads the legacy spelling of the DSL, `import framework.*`, as the
// real features still have it; the sweep runs it before Fn169Task1DslPackage. Its proof on a copied
// mini (framework/migrate/prove.sh) runs in that order: `INNER=<runner> prove.sh
// temporal.migrate.OnLegacyDsl` spells the copy's Model sources back to the legacy package, runs the
// inner runner on the target, then applies Fn169Task1DslPackage to every Model source of the copy,
// so the lift after compares with the lift before as the inner runner's proof requires.
object OnLegacyDsl:
  // Every Scala source under `temporal`, sorted.
  def sources(temporal: Path): List[Path] =
    val stream = Files.walk(temporal)
    try stream.iterator.asScala.filter(_.toString.endsWith(".scala")).toList.sorted
    finally stream.close()

  // The legacy spelling: `framework.dsl` written as `framework` (comments and literals keep theirs).
  def legacy(text: String): String =
    val tokens = dialects.Scala3(Input.String(text)).tokenize.get.toVector
    tokens.indices
      .filterNot: i =>
        def at(j: Int, value: String) = tokens.lift(j).exists(t => t.syntax == value)
        (at(i, ".") && at(i - 1, "framework") && at(i + 1, "dsl")) ||
        (at(i, "dsl") && at(i - 1, ".") && at(i - 2, "framework"))
      .map(tokens(_).syntax)
      .mkString

  private val rules = Rules(List(new Fn169Task1DslPackage))

  // `text` as Fn169Task1DslPackage rewrites it; a refusal fails the proof.
  def forward(path: Path, text: String): String =
    val tree = dialects.Scala3(Input.String(text)).parse[Source].get
    val result = rules.syntacticPatch(SyntacticDocument.fromTree(tree), suppress = false)
    val refusals = result.diagnostics.filterNot(_.message.startsWith("Unused Scalafix suppression"))
    require(refusals.isEmpty, s"$path: ${refusals.map(_.message).mkString("; ")}")
    result.fixed

  def main(args: Array[String]): Unit =
    require(args.length == 1, "provide the isolated copied-mini protocol directory")
    val inner = sys.env.getOrElse("INNER", sys.error("name the inner runner in INNER"))
    val protocol = Path.of(args(0)).toRealPath()
    val temporal = protocol.getParent.getParent.getParent
    require(temporal.getFileName.toString == "temporal", s"$temporal is no copied model/temporal")
    val before = sources(temporal)
    for path <- before do Files.writeString(path, legacy(Files.readString(path))): Unit
    Class
      .forName(inner)
      .getMethod("main", classOf[Array[String]])
      .invoke(null, Array(args(0))): Unit
    val after = sources(temporal)
    for path <- after do Files.writeString(path, forward(path, Files.readString(path))): Unit
    println(
      s"$inner ran on the legacy spelling of ${before.size} Model sources; Fn169Task1DslPackage " +
        s"then spelled ${after.size} back to framework.dsl"
    )
