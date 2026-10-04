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
      Capabilities,
      Syntax:
  import ctx.*
  import ctx.quotes.reflect.*

  /**
   * A root: a machine, a composition, a Query, a list of Queries, a progress claim, a realization, or
   * a capability declaration.
   */
  def liftRoot(root: String): Unit =
    val sym = defs.keys
      .find(s => s.isValDef && s.fullName == root)
      .getOrElse(throw LiftError(s"root $root", "names no declaration of the lifted sources"))
    val d = valDef(sym, sym.tree, "a declaration")
    if isNamed(d.tpt.tpe, "umpire.Machine") then machineOf(sym, d)
    else if isNamed(d.tpt.tpe, "umpire.Composition") then compositionOf(sym, d)
    else if isNamed(d.tpt.tpe, "umpire.realize.Realization") then realizationOf(sym, d)
    else if isNamed(d.tpt.tpe, "umpire.Capabilities") then fold(Ref(sym), Map.empty): Unit
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

  /**
   * The IR files the lifted sources declare, `val f = irFile("name")(root, ...)`: each file's name
   * and its roots' fully qualified names, in the order written. A root is a reference to a val, so
   * one that names nothing does not compile; the lift refuses it as it refuses a root of the command
   * line that is not a declaration. A declaration that cannot be read is given to `refused`, each one.
   */
  def irFiles(refused: LiftError => Unit): Seq[(String, Seq[String])] =
    val declared = defs.values.toSeq
      .collect { case v: ValDef if v.rhs.nonEmpty && isNamed(v.tpt.tpe, "umpire.IrFile") => v }
      .sortBy(v => (pos(v).file, pos(v).line))
    val files = declared.flatMap: v =>
      try
        call(v.rhs.get) match
          case Some(("irFile", List(List(name), roots))) =>
            val file = name match
              case Literal(StringConstant(n)) if n.nonEmpty && !n.contains('/') => n
              case other                                                        =>
                fail(other, "an IR file is named by a nonempty string literal without a `/`")
            val named = roots
              .flatMap(varargs)
              .map:
                case r: Ref if r.symbol.isValDef => r.symbol.fullName
                case other                       =>
                  fail(other, s"$file names its roots one by one, each a val that declares one")
            Some((v, file, named))
          case _ => fail(v, s"${v.name} declares an IR file with `irFile(name)(root, ...)`")
      catch
        case e: LiftError =>
          refused(e)
          None
    val twice =
      files.groupBy(_._2).collect { case (file, group) if group.size > 1 => file -> group }
    for (file, group) <- twice.toSeq.sortBy(_._1); again <- group.tail do
      refused(
        LiftError(
          where(again._1),
          s"$file is declared twice, at ${where(group.head._1)} and here: an IR file is declared once"
        )
      )
    files.collect { case (_, file, roots) if !twice.contains(file) => file -> roots }
