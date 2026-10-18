package framework.dsl

import scala.quoted.{Expr, Quotes}

// The feature a declaration belongs to: the package every Definition ID a realization kit forms
// hangs off, such as `<family>.evidence.<kind>`. A feature package supplies it once,
//
//   given Family = Family.here
//
// and a kit function that forms an ID takes `(using f: Family)` and reads `f.id`. A call with no
// Family in scope is the ordinary missing-given error. A family names a feature, so `Family.here`
// refuses at compile time inside the framework and inside a realization kit (a package named
// `realize`), naming the package.
final case class Family private[framework] (id: String)

object Family:
  inline def here: Family = ${ enclosing }

  private def enclosing(using q: Quotes): Expr[Family] =
    import q.reflect.*
    val pkg = Iterator
      .iterate(Symbol.spliceOwner)(_.owner)
      .find(s => s.isNoSymbol || s.isPackageDef)
      .filterNot(_.isNoSymbol)
      .fold("<empty>")(_.fullName)
    val segments = pkg.split('.').toList
    if pkg == "<empty>" || pkg == "<root>" then
      report.errorAndAbort("Family.here in the empty package: a family names a feature's package")
    else if segments.head == "framework" || segments.contains("realize") then
      report.errorAndAbort(
        s"Family.here in $pkg: a family names a feature, not the framework or a realization kit"
      )
    '{ Family(${ Expr(pkg) }) }
