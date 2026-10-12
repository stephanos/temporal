package umpire.migrate

import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*
import scala.meta.*
import scalafix.v1.SyntacticDocument
import scalafix.internal.v1.Rules

// fn-160.3 on a copied mini: the copy's accepted family names are spelled back to their legacy
// names and files, then the sweep's recipe runs, the compiled rule over every source and the file
// moves, and must restore every accepted source byte for byte (prove-mini.sh then lifts it again).
object RewriteFamilyNamesMini:
  def main(args: Array[String]): Unit =
    require(args.length == 1, "provide the isolated copied-mini protocol directory")
    val miniProtocol = Path.of(args(0)).toRealPath()
    require(
      miniProtocol != Path.of("model/temporal/features/activity_mini/protocol").toRealPath(),
      "the source proof must not rewrite the authored mini"
    )
    val temporal = miniProtocol.getParent.getParent.getParent
    val folders = List(miniProtocol, temporal.resolve("foundations/taskqueue/protocol"))
    val renames = Fn160Task3FamilyNames.shared ++ Fn160Task3FamilyNames.mini
    val back = renames.map(_.swap)
    def sources(folder: Path) =
      Files.list(folder).iterator.asScala.filter(_.toString.endsWith(".scala")).toList.sorted
    val accepted = folders.flatMap(sources).map(p => p -> Files.readString(p)).toMap
    // The legacy tree: old identifiers (comments and literals keep their text) at the old paths.
    val moves = Fn160Task3FamilyNames.moves.map(_.swap)
    for (path, text) <- accepted do
      val old = dialects.Scala3(Input.String(text)).tokenize.get.toList.map {
        case t: Token.Ident if back.contains(t.value) => back(t.value)
        case t                                        => t.syntax
      }.mkString
      Files.delete(path)
      Files.writeString(path.resolveSibling(moves.getOrElse(path.getFileName.toString, path.getFileName.toString)), old): Unit
    val legacy = folders.flatMap(sources)
    require(legacy.map(_.getFileName.toString).toSet.contains("Faults.scala"), "legacy layout not made")
    // The recipe: the rule over every source, then each file moved to its new name.
    val rules = Rules(List(new Fn160Task3FamilyNames))
    var changed = 0
    for path <- legacy do
      val text = Files.readString(path)
      val tree = dialects.Scala3(Input.String(text)).parse[Source].get
      val result = rules.syntacticPatch(SyntacticDocument.fromTree(tree), suppress = false)
      require(result.diagnostics.isEmpty, result.diagnostics.map(_.message).mkString("; "))
      if result.fixed != text then changed += 1
      val target = path.resolveSibling(
        Fn160Task3FamilyNames.moves.getOrElse(path.getFileName.toString, path.getFileName.toString)
      )
      Files.delete(path)
      Files.writeString(target, result.fixed): Unit
    val restored = folders.flatMap(sources).map(p => p -> Files.readString(p)).toMap
    require(restored.keySet == accepted.keySet, "the moves did not restore the accepted file set")
    for (path, text) <- accepted do require(restored(path) == text, s"$path differs from accepted")
    println(s"restored ${accepted.size} accepted sources byte for byte; the rule rewrote $changed, " +
      s"${Fn160Task3FamilyNames.moves.size} files moved")
