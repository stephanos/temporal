package temporal.migrate

import scala.meta.*
import scalafix.v1.*

final class Fn159Task1QualifiedActions extends SyntacticRule("Fn159Task1QualifiedActions"):
  override def fix(using doc: SyntacticDocument): Patch =
    val imports = doc.tree.collect { case i: Import => i }
    val calls = doc.tree.collect {
      case a: Term.Apply if Set("on", "disabled")(a.fun.syntax) => a
    }
    def reference(t: Term): Term = t match
      case a: Term.Apply     => reference(a.fun)
      case a: Term.ApplyType => reference(a.fun)
      case other             => other
    def group(t: Tree): Option[String] = t.parent.flatMap {
      case a: Term.Apply =>
        a.fun match
          case f: Term.Apply if f.fun.syntax == "from" && f.argClause.values.size == 1 =>
            Some(f.argClause.values.head.syntax)
          case _ => group(a)
      case parent => group(parent)
    }
    def visible(imported: Import, target: Tree): Boolean =
      def inside(t: Tree): Boolean = imported.parent.exists(scope =>
        scope.pos.start <= t.pos.start && t.pos.end <= scope.pos.end
      )
      imported.pos.end <= target.pos.start && inside(target)
    val replacements = scala.collection.mutable.ArrayBuffer.empty[(Term.Name, String)]
    val removed = scala.collection.mutable.ArrayBuffer.empty[Import]
    val errors = scala.collection.mutable.ArrayBuffer.empty[Diagnostic]
    for call <- calls; target <- call.argClause.values do
      reference(target) match
        case name: Term.Name =>
          val candidates = imports.filter(visible(_, call)).flatMap { i =>
            i.importers.flatMap { importer =>
              importer.importees.collect {
                case n: Importee.Name if n.name.value == name.value =>
                  (i, importer.ref.syntax, name.value)
                case n: Importee.Rename if n.rename.value == name.value =>
                  (i, importer.ref.syntax, n.name.value)
                case _: Importee.Wildcard if group(call).contains(importer.ref.syntax) =>
                  (i, importer.ref.syntax, name.value)
              }
            }
          }
          val protectedNames = Set(name.value) ++ candidates.map(_._2.takeWhile(_ != '.'))
          val shadowed = doc.tree.collect {
            case p: Pat.Var if protectedNames(p.name.value)    => p
            case p: Term.Param if protectedNames(p.name.value) => p
            case d: Defn.Def if protectedNames(d.name.value)   => d
          }.nonEmpty
          if candidates.size != 1 || shadowed then
            errors += Diagnostic(
              "ambiguous-action-target",
              s"cannot resolve bare ${call.fun.syntax}(${name.value}): qualify through its declaring object manually",
              name.pos
            )
          else
            val (imported, owner, member) = candidates.head
            replacements += name -> s"$owner.$member"
            removed += imported
        case _: Term.Select => ()
        case other          =>
          errors += Diagnostic(
            "ambiguous-action-target",
            "unsupported action reference: qualify through its declaring object manually",
            other.pos
          )
    val rewritten = replacements.map(_._1.pos.start).toSet
    def bareUses(value: String) = doc.tree.collect {
      case name: Term.Name if name.value == value && !name.parent.exists {
            case select: Term.Select => select.name.pos.start == name.pos.start
            case _                   => false
          } =>
        name
    }
    val removable = removed.distinct.filter { imported =>
      val selectors = imported.importers.flatMap(_.importees)
      selectors.forall {
        case n: Importee.Name =>
          bareUses(n.name.value)
            .forall(name =>
              rewritten(
                name.pos.start
              ) || imported.pos.start <= name.pos.start && name.pos.end <= imported.pos.end
            )
        case n: Importee.Rename =>
          bareUses(n.rename.value)
            .forall(name =>
              rewritten(
                name.pos.start
              ) || imported.pos.start <= name.pos.start && name.pos.end <= imported.pos.end
            )
        case _: Importee.Wildcard => group(imported).isDefined
        case _                    => false
      }
    }
    for imported <- imports if group(imported).isDefined && !removable.contains(imported) do
      errors += Diagnostic(
        "ambiguous-action-import",
        "from imports cannot be removed safely: qualify its actions and remove the import manually",
        imported.pos
      )
    if errors.nonEmpty then errors.map(Patch.lint).asPatch
    else
      (replacements.map((name, qualified) => Patch.replaceTree(name, qualified)) ++
        removable.map(i => Patch.replaceTree(i, ""))).asPatch.atomic
