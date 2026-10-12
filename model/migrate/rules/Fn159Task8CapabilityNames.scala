package umpire.migrate

import scala.meta.*
import scalafix.v1.*

final class Fn159Task8CapabilityNames extends SyntacticRule("Fn159Task8CapabilityNames"):
  override def fix(using doc: SyntacticDocument): Patch =
    val owner = "temporal.capabilities.system"
    val renamed = Map("Retries" -> "Retryable", "Deadline" -> "Expirable")
    def topLevel(tree: Tree): Boolean = tree.parent.exists {
      case _: Source | _: Pkg | _: Pkg.Body => true
      case _                                => false
    }
    def qualified(n: Name): Boolean = n.parent.exists {
      case s: Term.Select => s.name.pos == n.pos && s.qual.syntax == owner
      case s: Type.Select => s.name.pos == n.pos && s.qual.syntax == owner
      case _              => false
    }
    val imports = doc.tree.collect { case i: Importer => i }
    def frameworkBase(name: String): Boolean = imports.exists(i =>
      i.ref.syntax == "framework" && i.importees.exists {
        case _: Importee.Wildcard => true
        case n: Importee.Name     => n.name.value == name
        case _                    => false
      }
    )
    val ownedImports = imports.filter(i =>
      i.ref.syntax == owner ||
        renamed.keys.exists(n => i.ref.syntax == s"$owner.$n")
    )
    val declaredOwner = doc.tree.collect { case p: Pkg => p.ref.syntax }.mkString(".") == owner
    val declarations = doc.tree.collect {
      case c: Defn.Class if declaredOwner && renamed.contains(c.name.value)  => c
      case o: Defn.Object if declaredOwner && renamed.contains(o.name.value) => o
    }
    val explicit = doc.tree.collect {
      case n: Name if renamed.contains(n.value) && qualified(n) => n
    }
    val relevant = ownedImports.exists(i =>
      i.ref.syntax != owner || i.importees.exists {
        case n: Importee.Name     => renamed.contains(n.name.value)
        case r: Importee.Rename   => renamed.contains(r.name.value)
        case _: Importee.Wildcard => true
        case _                    => false
      }
    ) || declarations.nonEmpty || explicit.nonEmpty
    if !relevant then Patch.empty
    else
      val imported = ownedImports
        .filter(_.ref.syntax == owner)
        .flatMap(_.importees.collect {
          case n: Importee.Name if renamed.contains(n.name.value) => n.name
        })
      val declarationNames = declarations.collect {
        case c: Defn.Class  => c.name
        case o: Defn.Object => o.name
      }
      val members = ownedImports
        .filter(_.ref.syntax != owner)
        .flatMap(_.ref.collect {
          case n: Name if qualified(n) => n
        })
      val known = (imported ++ declarationNames).map(_.value).toSet
      val declared = declarationNames.map(_.pos.start).toSet
      val references = doc.tree.collect {
        case n: Name if known(n.value) && !n.parent.exists {
              case s: Term.Select            => s.name.pos == n.pos
              case s: Type.Select            => s.name.pos == n.pos
              case _: Importee | _: Importer => true
              case _                         => false
            } =>
          n
      }
      val outsideImport = references.exists(n =>
        imported.exists(_.value == n.value) &&
          !ownedImports.exists(i =>
            i.ref.syntax == owner &&
              i.importees.exists {
                case a: Importee.Name => a.name.value == n.value
                case _                => false
              } && i.pos.end <= n.pos.start && i.parent
                .flatMap(_.parent)
                .exists(scope => scope.pos.start <= n.pos.start && n.pos.end <= scope.pos.end)
          )
      )
      val bindings = doc.tree.collect {
        case n: Pat.Var     => n.name
        case n: Term.Param  => n.name
        case n: Type.Param  => n.name
        case n: Defn.Class  => n.name
        case n: Defn.Trait  => n.name
        case n: Defn.Object => n.name
        case n: Defn.Type   => n.name
        case n: Defn.Def    => n.name
      }
      val badBinding = bindings.exists(n =>
        (known(n.value) && !declared(n.pos.start)) || renamed.values.toSet(n.value) ||
          (explicit.nonEmpty || members.nonEmpty) && n.value == "temporal" ||
          declarations.nonEmpty && Set("CapabilityOf", "CapabilityKind", "framework")(n.value)
      )
      val badDeclaration = declarations.exists {
        case c: Defn.Class =>
          !topLevel(c) || !c.templ.inits.exists(i =>
            frameworkBase("CapabilityOf") && i.tpe.syntax
              .startsWith("CapabilityOf[") || i.tpe.syntax.startsWith(
              "framework.CapabilityOf["
            )
          )
        case o: Defn.Object =>
          !topLevel(o) || !o.templ.inits.exists(i =>
            frameworkBase(
              "CapabilityKind"
            ) && i.tpe.syntax == "CapabilityKind" || i.tpe.syntax == "framework.CapabilityKind"
          )
        case _ => true
      } || declarationNames.groupBy(_.value).exists((_, ns) => ns.size != 2)
      val badImport = imports.exists { i =>
        val owned = ownedImports.contains(i)
        val competing = i.importees.exists {
          case n: Importee.Name =>
            known(n.name.value) || renamed.values.toSet(n.name.value) ||
            declarations.nonEmpty && i.ref.syntax != "framework" &&
            Set("CapabilityOf", "CapabilityKind")(n.name.value)
          case r: Importee.Rename =>
            known(r.rename.value) || renamed.values.toSet(r.rename.value) ||
            declarations.nonEmpty && Set("CapabilityOf", "CapabilityKind", "framework")(
              r.rename.value
            )
          case _: Importee.Wildcard => i.ref.syntax != "framework"
          case _                    => false
        }
        if owned then
          !i.parent.flatMap(_.parent).exists {
            case _: Source | _: Pkg | _: Pkg.Body => true
            case _                                => false
          } || i.importees.exists {
            case _: Importee.Wildcard | _: Importee.Rename | _: Importee.Unimport => true
            case _                                                                => false
          }
        else competing
      } || imported.groupBy(_.value).exists((_, ns) => ns.size != 1)
      if badBinding || badDeclaration || badImport || outsideImport then
        Patch.lint(
          Diagnostic(
            "unsupported-capability-rename",
            "ambiguous, shadowed or unsupported capability ownership: manually resolve every legacy reference to temporal.capabilities.system.Retryable/Expirable, inventory callers and rename the declaration files; preserve deadline bindings and unrelated symbols",
            (imported ++ declarationNames ++ explicit ++ members).headOption
              .fold(doc.tree.pos)(_.pos)
          )
        )
      else
        (imported ++ declarationNames ++ explicit ++ members ++ references)
          .groupBy(_.pos.start)
          .values
          .map(_.head)
          .toList
          .map(n => Patch.replaceTree(n, renamed(n.value)))
          .asPatch
          .atomic
