package umpire.irgen

// The concerns of one lift over its context. They call one another, so each is a trait of its own
// file and this class is where they meet.
final private[irgen] class Lifting(val ctx: Context)
    extends Types,
      Constants,
      Expressions,
      Declarations,
      Realizations,
      Compositions,
      Claims,
      Capabilities,
      Syntax,
      Markers,
      PhaseRoles:
  import ctx.*
  import ctx.quotes.reflect.*

  // A root: a machine, a composition, a Query, a list of Queries, a progress claim, a realization, a
  // `capabilities` section, which is its object's, or a
  // `queries` section, which roots every Query, list of Queries and progress claim it declares, in
  // the order written.
  def liftRoot(root: String): Unit =
    val sym = defs.keys
      .find(s => s.isValDef && s.fullName == root)
      .getOrElse(throw LiftError(s"root $root", "names no declaration of the lifted sources"))
    val d = valDef(sym, sym.tree, "a declaration")
    // A machine or composition object is a root by its object, `irFile(...)(ActivityProduct)`.
    if realizationObject(sym) then realizationObjectOf(sym, d): Unit
    else if objectForm(sym) && isMachine(d.tpt.tpe) then machineOf(sym, d)
    else if objectForm(sym) then compositionOf(sym, d)
    else if capabilitiesObject(sym) then fold(Ref(sym), Map.empty): Unit
    else if queriesSection(sym) then
      for q <- statements(objectBody(moduleClassOf(sym), d)) do
        q match
          case v: ValDef if claimRoot(v.tpt.tpe) => liftRoot(v.symbol.fullName)
          case v: ValDef if readAtRunTime(v)     =>
            fail(
              v,
              s"${v.name} is a ${v.tpt.tpe.widen.dealias.show}, which the IR file reads at run " +
                "time but the IR does not: a `queries` section declares a Query, a List or Vector " +
                "of Queries, or a progress claim"
            )
          case _ => ()
    else if isNamed(d.tpt.tpe, "umpire.Machine") then machineOf(sym, d)
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
          s"$root is a ${kind.show}; a root is a machine, a composition, a Query, a list of Queries, " +
            "a progress claim, a realization, or a machine's `capabilities` or `queries` section"
        )
      fold(Ref(sym), Map.empty)

  // Whether a symbol names the `queries` section of a machine or composition object.
  private def queriesSection(sym: Symbol): Boolean =
    val cls = moduleClassOf(sym)
    !cls.isNoSymbol && cls.name.stripSuffix("$") == "queries" && objectForm(cls.maybeOwner)

  // Whether the IR file reads a val of a `queries` section at run time (`IrFile.queriesOf`): a
  // public one that is a Query or a sequence. One the lifter does not root is refused, so the IR and
  // the run never disagree on a section's Queries.
  private def readAtRunTime(v: ValDef): Boolean =
    val kind = v.tpt.tpe.widen.dealias
    !v.symbol.flags.is(Flags.Private) && !v.symbol.flags.is(Flags.Protected) &&
    (kind.baseClasses.exists(_.fullName == "umpire.Query") ||
      kind.baseClasses.exists(_.fullName == "scala.collection.Seq"))

  // Whether a val of a `queries` section is a root: a Query, a list of them or a progress claim.
  private def claimRoot(t: TypeRepr): Boolean =
    val kind = t.widen.dealias
    val listed =
      isList(kind.typeSymbol) || kind.typeSymbol.fullName == "scala.collection.immutable.Vector"
    Set("umpire.Query", "umpire.Progress")(kind.typeSymbol.fullName) ||
    (listed && kind.typeArgs.headOption.exists(a => isNamed(a, "umpire.Query")))

  // The IR files the lifted sources declare, `val f = irFile("name")(root, ...)`: each file's name
  // and its roots' fully qualified names, in the order written. A root is a reference to a val, so
  // one that names nothing does not compile; the lift refuses it as it refuses a root of the command
  // line that is not a declaration. A declaration that cannot be read is given to `refused`, each one.
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
