package temporal.migrate

import scala.meta.*
import scalafix.v1.*

final class Fn159Task3QualifiedPhases extends SyntacticRule("Fn159Task3QualifiedPhases"):
  override def fix(using doc: SyntacticDocument): Patch =
    val phaseTypes = doc.tree.collect {
      case t: Type.Apply if t.tpe.syntax == "Phased" && t.argClause.values.size == 2 =>
        t.argClause.values(1).syntax
    }.toSet
    val imports = doc.tree.collect { case i: Import => i }
    val phaseImports = imports.filter(_.importers.exists(i => phaseTypes(i.ref.syntax)))
    val replacements = scala.collection.mutable.ArrayBuffer.empty[(Term.Name, String)]
    val errors = scala.collection.mutable.ArrayBuffer.empty[Diagnostic]
    def refuse(at: Tree): Unit =
      errors += Diagnostic(
        "ambiguous-phase-import",
        "cannot resolve phase members safely: qualify every phase through its Phased type and remove the import manually",
        at.pos
      )
    def visible(imported: Import, target: Tree): Boolean =
      imported.pos.end <= target.pos.start && imported.parent.exists(scope =>
        scope.pos.start <= target.pos.start && target.pos.end <= scope.pos.end
      )
    def binding(importer: Importer): List[(String, String)] =
      importer.importees.flatMap {
        case n: Importee.Name     => List(n.name.value -> n.name.value)
        case n: Importee.Rename   => List(n.rename.value -> n.name.value)
        case _: Importee.Wildcard =>
          val enums = doc.tree.collect {
            case e: Defn.Enum if e.name.value == importer.ref.syntax =>
              e.templ.body.stats.collect {
                case c: Defn.RepeatedEnumCase => c.cases.map(n => n.value -> n.value)
                case c: Defn.EnumCase         => List(c.name.value -> c.name.value)
              }.flatten
          }
          if enums.size != 1 || enums.head.isEmpty then
            refuse(importer)
            Nil
          else enums.head
        case other =>
          refuse(other)
          Nil
      }
    val bindings = phaseImports.flatMap(i =>
      i.importers
        .filter(j => phaseTypes(j.ref.syntax))
        .flatMap(j => binding(j).map((local, member) => (i, local, j.ref.syntax + "." + member)))
    )
    phaseImports.filter(_.importers.exists(j => !phaseTypes(j.ref.syntax))).foreach(refuse)
    val names = bindings.map(_._2).toSet
    val qualifiers = phaseTypes.map(_.takeWhile(_ != '.'))
    val shadowed = doc.tree.collect {
      case p: Pat.Var if names(p.name.value) || qualifiers(p.name.value)    => p
      case p: Term.Param if names(p.name.value) || qualifiers(p.name.value) => p
      case d: Defn.Def if names(d.name.value) || qualifiers(d.name.value)   => d
      case o: Defn.Object if qualifiers(o.name.value)                       => o
    }
    shadowed.foreach(refuse)
    for name <- doc.tree.collect { case n: Term.Name if names(n.value) => n } do
      val imported = imports.exists(i => i.pos.start <= name.pos.start && name.pos.end <= i.pos.end)
      val selected = name.parent.exists {
        case s: Term.Select                              => s.name.pos.start == name.pos.start
        case _: Defn.EnumCase | _: Defn.RepeatedEnumCase => true
        case _                                           => false
      }
      if !imported && !selected then
        val candidates = bindings.filter((i, local, _) => local == name.value && visible(i, name))
        val competing = imports
          .filter(visible(_, name))
          .exists(
            _.importers.exists(j =>
              !phaseTypes(j.ref.syntax) && j.ref.syntax != "framework" && j.importees.exists {
                case n: Importee.Name     => n.name.value == name.value
                case n: Importee.Rename   => n.rename.value == name.value
                case _: Importee.Wildcard => true
                case _                    => false
              }
            )
          )
        if candidates.size != 1 || competing then refuse(name)
        else replacements += name -> candidates.head._3
    if errors.nonEmpty then errors.map(Patch.lint).asPatch
    else
      (replacements.map((name, qualified) => Patch.replaceTree(name, qualified)) ++
        phaseImports.map(i => Patch.replaceTree(i, ""))).asPatch.atomic
