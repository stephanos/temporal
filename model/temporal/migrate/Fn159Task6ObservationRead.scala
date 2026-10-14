package temporal.migrate

import scala.meta.*
import scalafix.v1.*

final class Fn159Task6ObservationRead extends SyntacticRule("Fn159Task6ObservationRead"):
  override def fix(using doc: SyntacticDocument): Patch =
    val imports = doc.tree.collect { case i: Importer => i }
    val shadowed = doc.tree.collect {
      case p: Pat.Var if Set("Observation", "framework")(p.name.value)     => p
      case p: Term.Param if Set("Observation", "framework")(p.name.value)  => p
      case d: Defn.Def if Set("Observation", "framework")(d.name.value)    => d
      case d: Defn.Object if Set("Observation", "framework")(d.name.value) => d
      case d: Defn.Class if d.name.value == "Observation"                  => d
    }
    val patches = scala.collection.mutable.ArrayBuffer.empty[Patch]
    val errors = scala.collection.mutable.ArrayBuffer.empty[Diagnostic]
    def refuse(at: Tree): Unit =
      errors += Diagnostic(
        "unsafe-observation-read",
        "resolve the framework constructor manually; retain name/on and typed response selectors, evaluate any effectful label explicitly before removing read, and add an accurate response-field hint",
        at.pos
      )
    def visible(i: Importer, at: Tree): Boolean =
      i.pos.end < at.pos.start && i.parent
        .flatMap(_.parent)
        .exists(p => p.pos.start <= at.pos.start && at.pos.end <= p.pos.end)
    def provides(i: Importer): Boolean = i.importees.exists {
      case n: Importee.Name     => n.name.value == "Observation"
      case _: Importee.Wildcard => true
      case n: Importee.Rename   => n.rename.value == "Observation"
      case _                    => false
    }
    val aliases = imports
      .flatMap(i =>
        i.importees.collect {
          case n: Importee.Rename if i.ref.syntax == "framework" && n.name.value == "Observation" =>
            n.rename.value
        }
      )
      .toSet
    val constructors = Set("Observation", "framework.Observation", "_root_.framework.Observation")
    def explicitApply(fun: Term): Boolean = fun match
      case s: Term.Select =>
        s.name.value == "apply" && (constructors(s.qual.syntax) || aliases(s.qual.syntax))
      case _ => false
    doc.tree
      .collect {
        case a: Term.Apply if aliases(a.fun.syntax)                                       => a
        case a: Term.Apply if explicitApply(a.fun)                                        => a
        case n: Term.New if constructors(n.init.tpe.syntax) || aliases(n.init.tpe.syntax) => n
      }
      .foreach(refuse)
    for call <- doc.tree.collect { case a: Term.Apply if constructors(a.fun.syntax) => a }
    do
      val args = call.argClause.values
      val namedRead = args.collect { case a: Term.Assign if a.lhs.syntax == "read" => a }
      val named = args.exists(_.isInstanceOf[Term.Assign])
      val label = if named then namedRead.headOption else args.lift(2)
      val legacy = label.nonEmpty || args.size > 2 || args.exists {
        case a: Term.Assign => !Set("name", "on")(a.lhs.syntax)
        case _              => false
      }
      if legacy then
        val rooted = call.fun.syntax == "_root_.framework.Observation"
        val packageShadow = imports.exists(i =>
          visible(i, call) && i.importees.exists {
            case n: Importee.Name     => n.name.value == "framework"
            case n: Importee.Rename   => n.rename.value == "framework"
            case _: Importee.Wildcard => i.ref.syntax != "framework"
            case _                    => false
          }
        )
        val resolved = rooted || (!packageShadow && call.fun.syntax == "framework.Observation") ||
          (imports.count(i =>
            visible(i, call) && provides(i) && i.ref.syntax == "framework" &&
              i.importees.exists {
                case n: Importee.Name     => n.name.value == "Observation"
                case _: Importee.Wildcard => true
                case _                    => false
              }
          ) == 1 &&
            !imports.exists(i =>
              visible(i, call) && i.ref.syntax == "framework" && i.importees.exists {
                case n: Importee.Rename => n.rename.value == "Observation"
                case _                  => false
              }
            ) &&
            !imports.exists(i => visible(i, call) && provides(i) && i.ref.syntax != "framework"))
        val validShape = if named then
          args.forall {
            case a: Term.Assign => Set("name", "on", "read")(a.lhs.syntax)
            case _              => false
          } && args.collect { case a: Term.Assign => a.lhs.syntax }.distinct.size == args.size &&
          args.exists {
            case a: Term.Assign => a.lhs.syntax == "on"; case _ => false
          } && namedRead.size == 1
        else args.size == 3
        val literal = label.exists {
          case a: Term.Assign => a.rhs.isInstanceOf[Lit.String]
          case _: Lit.String  => true
          case _              => false
        }
        if !resolved || (!rooted && (shadowed.nonEmpty || packageShadow)) || !validShape || !literal
        then refuse(call)
        else
          val removed = label.get
          val index = args.indexOf(removed)
          val begin = if index == 0 then removed.pos.start else args(index - 1).pos.end
          val end = if index == 0 then args(1).pos.start else removed.pos.end
          patches += Patch.removeTokens(
            doc.tokens.filter(t =>
              begin <= t.pos.start && t.pos.end <= end && !t.isInstanceOf[Token.Comment]
            )
          )
    if errors.nonEmpty then errors.map(Patch.lint).asPatch
    else patches.asPatch.atomic
