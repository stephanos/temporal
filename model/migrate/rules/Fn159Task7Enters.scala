package umpire.migrate

import scala.meta.*
import scalafix.v1.*

final class Fn159Task7Enters(val sources: List[Source] = Nil)
    extends SyntacticRule("Fn159Task7Enters"):
  override def fix(using doc: SyntacticDocument): Patch =
    val roots = doc.tree :: sources
    val errors = scala.collection.mutable.ArrayBuffer.empty[Diagnostic]
    val edits = scala.collection.mutable.ArrayBuffer.empty[Patch]
    def refuse(at: Tree): Unit = errors += Diagnostic(
      "unsafe-phase-entry",
      "cannot prove a phase-only entry safely: manually resolve the concrete Phased projection/writer and enum status, inventory external callers, replace only matching one-step uses by enters(Type.case), then remove the unused effect",
      at.pos
    )
    def ancestor(t: Tree)(p: Tree => Boolean): Option[Tree] =
      t.parent.flatMap(a => if p(a) then Some(a) else ancestor(a)(p))
    def effects(t: Tree) = ancestor(t) {
      case o: Defn.Object => o.name.value == "effects"
      case _              => false
    }
    def subject(t: Tree) = ancestor(t) {
      case _: Defn.Object | _: Defn.Trait => true
      case _                              => false
    }
    def phased(init: Init): Option[(String, String, List[Term])] = init.tpe match
      case a: Type.Apply if a.tpe.syntax == "Phased" && a.argClause.values.size == 2 =>
        Some(
          (
            a.argClause.values.head.syntax,
            a.argClause.values(1).syntax,
            init.argClauses.flatMap(_.values).toList
          )
        )
      case _ => None
    def parents(t: Tree): List[Init] = t match
      case o: Defn.Object => o.templ.inits
      case t: Defn.Trait  => t.templ.inits
      case _              => Nil
    def remove(t: Tree): Patch =
      val lineStart = doc.input.text.lastIndexOf('\n', t.pos.start - 1) + 1
      val lineEnd = doc.input.text.indexOf('\n', t.pos.end)
      if lineEnd >= 0 && doc.input.text.substring(lineStart, t.pos.start).forall(_.isWhitespace) &&
        doc.input.text.substring(t.pos.end, lineEnd).forall(_.isWhitespace)
      then
        Patch.removeTokens(
          doc.tokens.filter(tok => tok.pos.start >= lineStart && tok.pos.end <= lineEnd + 1)
        )
      else Patch.replaceTree(t, "")
    val candidates = doc.tree
      .collect {
        case d: Defn.Def if effects(d).nonEmpty => d
      }
      .flatMap { d =>
        d.body match
          case a: Term.Apply if a.fun.syntax == "enter" =>
            a.argClause.values match
              case (copy: Term.Apply) :: facts if copy.argClause.values.size == 1 =>
                (copy.fun, copy.argClause.values.head) match
                  case (
                        Term.Select(Term.Name(s), Term.Name("copy")),
                        Term.Assign(Term.Name(field), phase: Term.Select)
                      ) =>
                    Some((d, s, field, phase, facts))
                  case _ => None
              case _ => None
          case _ => None
      }
    for (d, state, field, phase, facts) <- candidates do
      val container = effects(d).get
      val owner = subject(container).get
      val owners = owner :: subject(owner).toList
      val declarations = owners.flatMap(parents).flatMap(phased)
      val phaseType = phase.qual.syntax
      val types = declarations.filter(_._2 == phaseType).map(_._1).distinct
      val enums = roots.flatMap(_.collect { case e: Defn.Enum if e.name.value == phaseType => e })
      val records = roots.flatMap(_.collect {
        case c: Defn.Class if types.contains(c.name.value) => c
      })
      val stateField = records.size == 1 && records.head.ctor.paramClauses
        .flatMap(_.values)
        .exists(p => p.name.value == field && p.decltpe.exists(_.syntax == phaseType))
      val writers = roots
        .flatMap(_.collect {
          case o: Defn.Object if owners.exists {
                case t: Defn.Trait    => o.templ.inits.exists(_.tpe.syntax == t.name.value)
                case own: Defn.Object => own.pos == o.pos && own.syntax == o.syntax
                case _                => false
              } =>
            o
        })
        .flatMap(parents)
        .flatMap(phased)
        .filter(p => types.contains(p._1) && p._2 == phaseType)
      def writes(args: List[Term]): Boolean = args match
        case List(project, writer: Term.Function) if project.syntax == s"_.$field" =>
          val ps = writer.paramClause.values.map(_.name.value)
          ps.size == 2 && writer.body.syntax == s"${ps.head}.copy($field = ${ps(1)})"
        case _ => false
      val knownPlain = enums.size == 1 && enums.head.templ.inits.isEmpty
      val status = if enums.size == 1 then
        val e = enums.head
        if e.templ.inits.size == 1 && e.templ.inits.head.tpe.syntax.startsWith("Recorded[") then
          e.collect { case c: Defn.EnumCase if c.name.value == phase.name.value => c }
            .flatMap(
              _.inits.filter(_.tpe.syntax == phaseType).flatMap(_.argClauses.flatMap(_.values))
            )
            .headOption
            .map(_.syntax)
        else None
      else None
      val extraFact =
        facts.nonEmpty && (knownPlain || status.exists(s => facts.map(_.syntax) != List(s)))
      if !extraFact then
        val params = d.paramClauseGroups.flatMap(_.paramClauses.flatMap(_.values))
        val privateOwner = owners.exists {
          case o: Defn.Object => o.mods.exists(_.isInstanceOf[Mod.Private])
          case t: Defn.Trait  => t.mods.exists(_.isInstanceOf[Mod.Private])
          case _              => false
        } || d.mods.exists(_.isInstanceOf[Mod.Private])
        val frameworkSurface = roots.forall(root =>
          root.collect {
            case i: Importer
                if i.importees.exists(_.isInstanceOf[Importee.Wildcard]) &&
                  i.ref.syntax != "framework" =>
              i
            case t: Defn.Trait if Set("Phased", "Recorded")(t.name.value)                     => t
            case o: Defn.Object if Set("Phased", "Recorded", "enter", "enters")(o.name.value) => o
            case d: Defn.Def if Set("enter", "enters")(d.name.value)                          => d
            case i: Importer if i.ref.syntax != "framework" && i.importees.exists {
                  case n: Importee.Name =>
                    Set("Phased", "Recorded", "enter", "enters")(n.name.value)
                  case n: Importee.Rename =>
                    Set("Phased", "Recorded", "enter", "enters")(n.rename.value)
                  case _ => false
                } =>
              i
          }.isEmpty
        )
        val proven =
          (privateOwner || sources.nonEmpty) && frameworkSurface &&
            types.size == 1 && stateField && writers.nonEmpty && writers.forall(w =>
              writes(w._3)
            ) &&
            enums.size == 1 && (knownPlain || status.exists(s => facts.map(_.syntax) == List(s))) &&
            params.size == 1 && params.head.name.value == state && params.head.decltpe.exists(t =>
              types.contains(t.syntax)
            )
        if !proven then refuse(d)
        else
          val aliases = container.collect {
            case v: Defn.Val if v.rhs.syntax == d.name.value && v.pats.size == 1 => v
          }
          val names = d.name.value :: aliases.flatMap(_.pats.collect { case Pat.Var(n) => n.value })
          val definitions = d :: aliases
          val uses = doc.tree
            .collect { case n: Term.Name if names.contains(n.value) => n }
            .filterNot(n =>
              definitions.exists(t => t.pos.start <= n.pos.start && n.pos.end <= t.pos.end)
            )
          val changes = scala.collection.mutable.ArrayBuffer.empty[(Tree, String)]
          for n <- uses do
            val ref: Term = n.parent match
              case Some(s: Term.Select) if s.name.pos == n.pos => s
              case _                                           => n
            val within = ancestor(ref)(_.pos == owner.pos).nonEmpty
            val application = ref.parent.collect { case a: Term.Apply if a.fun.pos == ref.pos => a }
            val validRef = ref.syntax == s"effects.${n.value}" ||
              (ref == n && effects(ref).exists(_.pos == container.pos))
            application match
              case Some(a)
                  if within && validRef && a.argClause.values.size == 1 &&
                    a.argClause.values.head.syntax == "s" && effects(a).exists(
                      _.pos == container.pos
                    ) =>
                changes += a -> s"enters(${phase.syntax})(s)"
              case None if within && validRef && ancestor(ref) {
                    case i: Term.ApplyInfix =>
                      i.op.value == "~>" && i.argClause.values.exists(_.pos == ref.pos)
                    case _ => false
                  }.nonEmpty =>
                changes += ref -> s"enters(${phase.syntax})"
              case _ => refuse(ref)
          val external = sources.flatMap(_.collect {
            case s: Term.Select
                if names.contains(s.name.value) && s.qual.syntax.endsWith("effects") =>
              s
          })
          if uses.isEmpty || external.nonEmpty then refuse(d)
          if errors.isEmpty then
            edits ++= definitions.map(remove)
            edits ++= changes.map((t, replacement) => Patch.replaceTree(t, replacement))
    if errors.nonEmpty then errors.map(Patch.lint).toSeq.asPatch
    else edits.toSeq.asPatch.atomic
