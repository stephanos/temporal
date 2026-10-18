package temporal.migrate

import scala.meta.*
import scalafix.v1.*

// fn-169.1: the authoring DSL moved from package `framework` into `framework.dsl`, so a Model's
// `import framework.dsl.*` brings the authoring API alone into scope. Every name of the DSL a source
// writes out through `framework` gains `.dsl`: an import of `framework.*`, `framework.X` or
// `framework.{A, B}`, a reference `framework.X` or `_root_.framework.X`, and a package clause
// `package framework` (a test of the DSL's internals). The framework's other packages, `realize`,
// `export`, `check` and `migrate`, keep their names, and so do comments and literals; a moved test,
// or a source that imported `framework.*` and named one of them relatively, `realize.X`, names it
// from the root. An import that names one of them beside a DSL name,
// `import framework.{realize, Machine}`, refuses the document (`mixed-framework-import`).
final class Fn169Task1DslPackage extends SyntacticRule("Fn169Task1DslPackage"):
  override def fix(using doc: SyntacticDocument): Patch =
    import Fn169Task1DslPackage.{framework, kept}
    val others = kept - "dsl"
    def dsl(name: String) = !kept(name)
    def root(ref: Term.Ref): Option[Term.Name] = ref match
      case q: Term.Name if framework(q) => Some(q)
      case _                            => None
    val qualifiers = doc.tree.collect {
      case Term.Select(q: Term.Name, n) if framework(q) && dsl(n.value)                => q
      case Type.Select(q: Term.Name, n) if framework(q) && dsl(n.value)                => q
      case Term.Select(Term.Select(r: Term.Name, q: Term.Name), n)
          if r.value == "_root_" && framework(q) && dsl(n.value) =>
        q
      case Type.Select(Term.Select(r: Term.Name, q: Term.Name), n)
          if r.value == "_root_" && framework(q) && dsl(n.value) =>
        q
      case p: Pkg if root(p.ref).nonEmpty                                               => p.ref
      case Importer(q: Term.Name, importees) if framework(q) && importees.forall {
            case Importee.Name(n)      => dsl(n.value)
            case Importee.Rename(n, _) => dsl(n.value)
            case Importee.Unimport(n)  => dsl(n.value)
            case _                     => true
          } =>
        q
    }
    // A source in package `framework`, or one that imported `framework.*`, read the framework's
    // other packages relatively, `realize.X`; once it reads the DSL from `framework.dsl` it names
    // them from the root, `framework.realize.X`.
    val moved = doc.tree.collect { case p: Pkg => p.ref }.headOption.exists(root(_).nonEmpty)
    val wildcard = doc.tree.collect {
      case Importer(q: Term.Name, importees)
          if framework(q) && importees.exists {
            case _: Importee.Wildcard => true
            case _                    => false
          } =>
        q
    }.nonEmpty
    val relative =
      if !moved && !wildcard then Nil
      else
        doc.tree.collect {
          case Term.Select(q: Term.Name, _) if others(q.value) => q
          case Type.Select(q: Term.Name, _) if others(q.value) => q
          case Importer(q: Term.Name, _) if others(q.value)    => q
        }
    val mixed = doc.tree.collect {
      case i @ Importer(q: Term.Name, importees) if framework(q) && {
            val named = importees.collect {
              case Importee.Name(n)      => n.value
              case Importee.Rename(n, _) => n.value
              case Importee.Unimport(n)  => n.value
            }
            named.exists(kept) && (named.exists(dsl) || importees.exists {
              case _: Importee.Wildcard | _: Importee.Given | _: Importee.GivenAll => true
              case _                                                                => false
            })
          } =>
        i
    }
    mixed.headOption match
      case Some(i) =>
        Patch.lint(
          Diagnostic(
            "mixed-framework-import",
            "an import names a package of the framework beside a name of its DSL: split it, the " +
              "DSL's names from framework.dsl",
            i.pos
          )
        )
      case None =>
        (qualifiers.distinctBy(_.pos.start).map(Patch.addRight(_, ".dsl")) ++
          relative.distinctBy(_.pos.start).map(Patch.addLeft(_, "framework."))).asPatch.atomic

object Fn169Task1DslPackage:
  // The framework's packages besides the DSL, which keep their names.
  val kept: Set[String] = Set("dsl", "realize", "export", "check", "migrate")

  // The root `framework`, written alone, as the head of a path.
  def framework(name: Term.Name): Boolean = name.value == "framework"
