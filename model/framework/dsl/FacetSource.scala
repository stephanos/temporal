package framework.dsl

import scala.quoted.*

final private[framework] class FacetSource(
    val facet: Option[String],
    val section: String,
    val identity: Option[FacetSource.Identity] = None
)

object FacetSource:
  final private[framework] case class Identity(
      source: Machine[?, ?, ?],
      namespace: AnyRef,
      traitClass: java.lang.Class[?],
      traitName: String,
      namespaceName: String,
      rulesName: String
  )

  inline given declared: FacetSource = ${ declaredImpl }

  private[framework] def make(facet: Option[String], section: String): FacetSource =
    new FacetSource(facet, section)

  private def declaredImpl(using Quotes): Expr[FacetSource] =
    import quotes.reflect.*
    val section = Iterator
      .iterate(Symbol.spliceOwner)(_.owner)
      .takeWhile(_.exists)
      .find(s => s.isClassDef && s.flags.is(Flags.Module))
    val namespace = section.map(_.owner)
    val facet = namespace
      .filter(s => s.isClassDef && s.flags.is(Flags.Module))
      .filter(s =>
        s.owner.isClassDef && s.owner.flags.is(Flags.Trait) &&
          s.owner.typeRef.derivesFrom(Symbol.requiredClass("framework.dsl.Facet"))
      )
      .map(_.name.stripSuffix("$"))
    val named =
      section.fold("<rules>")(s => s.owner.name.stripSuffix("$") + "." + s.name.stripSuffix("$"))
    '{ make(${ Expr(facet) }, ${ Expr(named) }) }
