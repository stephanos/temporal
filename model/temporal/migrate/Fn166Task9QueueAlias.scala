package temporal.migrate

import scala.meta.*
import scalafix.v1.*

final class Fn166Task9QueueAlias extends SyntacticRule("Fn166Task9QueueAlias"):
  override def fix(using doc: SyntacticDocument): Patch =
    val oldName = "foundationQueue"
    val newName = "taskQueueSource"
    require(oldName.length == newName.length)
    val owner = doc.tree.collect { case p: Pkg => p.ref.syntax }.mkString(".")
    val imports = doc.tree.collect {
      case i: Importer if i.ref.syntax == "foundations.taskqueue" =>
        i.importees.collect {
          case r: Importee.Rename if r.name.value == "queue" && r.rename.value == oldName => r.rename
        }
    }.flatten
    if owner != "temporal.features.activity.mini.protocol" || imports.isEmpty then Patch.empty
    else
      val names = doc.tree.collect { case n: Name if n.value == oldName => n }
      val qualifiers = doc.tree.collect {
        case Term.Select(n: Term.Name, _) if n.value == oldName => n
      }
      val allowed = (imports ++ qualifiers).map(_.pos.start).toSet
      val collision = doc.tree.collect { case n: Name if n.value == newName => n }.nonEmpty
      if imports.size != 1 || collision || names.exists(n => !allowed(n.pos.start)) then
        Patch.lint(Diagnostic("ambiguous-queue-alias",
          "ambiguous queue alias: expected one import, no shadowing or target collision, and qualified uses only",
          imports.head.pos))
      else names.map(n => Patch.replaceTree(n, newName)).asPatch.atomic
