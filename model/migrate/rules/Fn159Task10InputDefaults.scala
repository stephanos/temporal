package umpire.migrate

import scala.meta.*
import scalafix.v1.*

final class Fn159Task10InputDefaults extends SyntacticRule("Fn159Task10InputDefaults"):
  override def fix(using doc: SyntacticDocument): Patch =
    val owner = doc.tree.collect { case p: Pkg => p.ref.syntax }.mkString(".")
    val mini = "temporal.features.activity.mini"
    val inventory = Map(
      "scheduleToStart" -> ("Timeout", Some("Timeout.unset")),
      "startToClose" -> ("Timeout", Some("Timeout.unset")),
      "maxAttempts" -> ("MaxAttempts", Some("MaxAttempts.one"))
    )
    val required = owner == "temporal.features.activity"
    val known = if required then Map("failure" -> ("Failure", None)) else inventory
    val declarations = doc.tree.collect { case v @ Defn.Val(_, List(Pat.Var(name)), _, rhs) =>
      (v, name.value, rhs)
    }
    def refuse(at: Position): Patch = Patch.lint(
      Diagnostic(
        "unknown-input-default",
        "input default ownership or old value is unproven; manual inventory of every omitted input and its actual catalog is required before declaring defaults",
        at
      )
    )
    val imports = doc.tree.collect { case i: Importer => i }
    val aliases = imports
      .filter(_.ref.syntax == "framework")
      .flatMap(_.importees.collect {
        case r: Importee.Rename if r.name.value == "input" => r.rename.value
      })
      .toSet
    val calls = doc.tree.collect {
      case t: Term.ApplyType if t.fun.syntax == "input" || aliases(t.fun.syntax) || (t.fun match
            case Term.Select(_, Term.Name("input")) => true
            case _ => false) =>
        t
    }
    if calls.isEmpty then
      if Set(mini, "temporal.features.activity")(owner) && declarations.exists((_, name, _) =>
          known.contains(name)
        )
      then refuse(doc.tree.pos)
      else Patch.empty
    else
      val framework = imports.exists(i =>
        i.ref.syntax == "framework" && i.importees.exists {
          case _: Importee.Wildcard => true
          case Importee.Name(n)     => n.value == "input"
          case _                    => false
        }
      )
      val competing = imports.exists(i =>
        i.ref.syntax != "framework" && i.importees.exists {
          case _: Importee.Wildcard =>
            !Set("temporal.capabilities.product", "temporal.capabilities.system")(i.ref.syntax)
          case Importee.Name(n) =>
            Set("input", "Timeout", "MaxAttempts", "Failure")(n.value) &&
            !(i.ref.syntax == "temporal.features.activity" && Set("Timeout", "Failure")(n.value))
          case r: Importee.Rename =>
            Set("input", "Timeout", "MaxAttempts", "Failure")(r.rename.value)
          case _ => false
        }
      )
      val shadows = doc.tree.collect {
        case d: Defn.Def if d.name.value == "input" => d
        case v: Defn.Val if v.pats.exists {
              case Pat.Var(n) => Set("input", "Timeout", "MaxAttempts", "Failure")(n.value)
              case _          => false
            } =>
          v
      }
      val changes = calls.map { call =>
        val declaration = declarations.filter { (_, _, rhs) =>
          rhs == call || (rhs match
            case applied: Term.Apply => applied.fun == call
            case _                   => false)
        }
        declaration match
          case List((v, name, rhs))
              if call.fun.syntax == "input" && v.parent.exists(_.isInstanceOf[Pkg.Body]) && known
                .contains(name) &&
                call.targClause.values.map(_.syntax) == List(known(name)._1) =>
            val valid = rhs match
              case _: Term.ApplyType   => true
              case applied: Term.Apply =>
                applied.argClause.values match
                  case List(Term.Assign(Term.Name("default"), _)) => true
                  case _                                          => false
              case _ => false
            if !valid then None
            else
              Some(if rhs == call then
                known(name)._2.fold(Patch.empty)(value =>
                  Patch.addRight(call, s"(default = $value)")
                )
              else Patch.empty)
          case _ => None
      }
      val duplicate = declarations.map(_._2).groupBy(identity).exists((_, ds) => ds.size > 1)
      val unproven = declarations.exists { (_, name, rhs) =>
        known.contains(name) && !calls.exists(call =>
          rhs == call || (rhs match
            case applied: Term.Apply => applied.fun == call
            case _                   => false)
        )
      }
      if !Set(mini, "temporal.features.activity")(
          owner
        ) || !framework || competing || shadows.nonEmpty ||
        duplicate || unproven || changes.exists(_.isEmpty)
      then refuse(calls.head.pos)
      else changes.flatten.asPatch.atomic
