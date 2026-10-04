package umpire.lift

/**
 * The concerns of one lift over its context. They call one another, so each is a trait of its own
 * file and this class is where they meet.
 */
final private[lift] class Lifting(val ctx: Context)
    extends Types,
      Constants,
      Expressions,
      Declarations,
      Realizations,
      Compositions,
      Claims,
      Syntax:
  import ctx.*
  import ctx.quotes.reflect.*

  /** A root: a machine, a composition, a Query, a list of Queries, a progress claim, or a realization. */
  def liftRoot(root: String): Unit =
    val sym = defs.keys
      .find(s => s.isValDef && s.fullName == root)
      .getOrElse(throw LiftError(s"root $root", "names no declaration of the lifted sources"))
    val d = valDef(sym, sym.tree, "a declaration")
    if isNamed(d.tpt.tpe, "umpire.Machine") then machineOf(sym, d)
    else if isNamed(d.tpt.tpe, "umpire.Composition") then compositionOf(sym, d)
    else if isNamed(d.tpt.tpe, "umpire.realize.Realization") then realizationOf(sym, d)
    else
      val kind = d.tpt.tpe.widen.dealias
      val claims = Set("umpire.Query", "umpire.Progress")
      val listed =
        isList(kind.typeSymbol) || kind.typeSymbol.fullName == "scala.collection.immutable.Vector"
      if !claims(kind.typeSymbol.fullName) && !(listed && kind.typeArgs.headOption
          .exists(a => isNamed(a, "umpire.Query")))
      then
        fail(
          d,
          s"$root is a ${kind.show}; a root is a machine, a composition, a Query, a list of Queries, a progress claim or a realization"
        )
      fold(Ref(sym), Map.empty)
