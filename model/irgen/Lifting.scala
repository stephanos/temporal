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
    else if objectForm(sym) && isMachine(d.tpt.tpe) then machineOf(sym, d): Unit
    else if objectForm(sym) then compositionOf(sym, d): Unit
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
    else if isNamed(d.tpt.tpe, "framework.Machine") then machineOf(sym, d): Unit
    else if isNamed(d.tpt.tpe, "framework.Composition") then compositionOf(sym, d): Unit
    else if isNamed(d.tpt.tpe, "framework.realize.Realization") then realizationOf(sym, d): Unit
    else
      val kind = d.tpt.tpe.widen.dealias
      val claims = Set("framework.Query", "framework.Witness", "framework.Progress")
      val listed =
        isList(
          kind.typeSymbol
        ) || kind.typeSymbol.fullName == "scala.collection.immutable.Vector"
      if !claims(kind.typeSymbol.fullName) && !(listed && kind.typeArgs.headOption
          .exists(a => Seq("framework.Query", "framework.Witness").exists(isNamed(a, _))))
      then
        fail(
          d,
          s"$root is a ${kind.show}; a root is a machine, a composition, a Query, a list of Queries, " +
            "a progress claim, a realization, or a machine's `capabilities` or `queries` section"
        )
      fold(Ref(sym), Map.empty): Unit

  // Whether a symbol names the `queries` section of a machine or composition object.
  private[irgen] def queriesSection(sym: Symbol): Boolean =
    val cls = moduleClassOf(sym)
    // A facet's `queries` are its machine's (fn-161).
    !cls.isNoSymbol && cls.name.stripSuffix("$") == "queries" &&
    (objectForm(cls.maybeOwner) || facetObject(cls.maybeOwner))

  // Whether the IR file reads a val of a `queries` section at run time (`IrFile.queriesOf`): a
  // public one that is a Query or a sequence. One the lifter does not root is refused, so the IR and
  // the run never disagree on a section's Queries.
  private def readAtRunTime(v: ValDef): Boolean =
    val kind = v.tpt.tpe.widen.dealias
    !v.symbol.flags.is(Flags.Private) && !v.symbol.flags.is(Flags.Protected) &&
    (kind.baseClasses.exists(
      _.fullName == "framework.Query"
    ) ||
      kind.baseClasses.exists(
        _.fullName == "scala.collection.Seq"
      ))

  // Whether a val of a `queries` section is a root: a Query, a list of them or a progress claim.
  private[irgen] def claimRoot(t: TypeRepr): Boolean =
    val kind = t.widen.dealias
    val listed =
      isList(
        kind.typeSymbol
      ) || kind.typeSymbol.fullName == "scala.collection.immutable.Vector"
    Set("framework.Query", "framework.Witness", "framework.Progress")(kind.typeSymbol.fullName) ||
    (listed && kind.typeArgs.headOption.exists(a =>
      Seq("framework.Query", "framework.Witness").exists(isNamed(a, _))
    ))

  // fn-160 R2, R10: one IR file per feature, collected from its package. A feature is a form under a
  // kind (`temporal.features.activity.standalone`, file `activity-standalone`), a kind's own package
  // (`temporal.features.nexus`, `nexus`), or a package under `foundations` or `actors`
  // (`temporal.foundations.taskqueue`, `taskqueue`), under any root package. Its roots are every machine and composition
  // object, `queries` section (a machine's or a facet's), machine `capabilities` section, progress
  // claim and realization declared in the package and its sub-packages, but for a kind's forms,
  // which are features of their own; everything they reach is lifted with them, so a foundation
  // machine a feature composes is lifted into its file, its Queries into the foundation's own. A
  // kind's package, foundation or actor that declares no Query, progress claim or realization writes
  // no file. Nothing is listed, so nothing can be left out by omission.
  def irFiles(refused: LiftError => Unit): Seq[(String, Seq[String])] =
    val _ = refused
    def packageOf(sym: Symbol): String =
      Iterator
        .iterate(sym)(_.maybeOwner)
        .find(o => o.isNoSymbol || o.isPackageDef)
        .filter(_.isPackageDef)
        .fold("")(_.fullName)
    // The feature a package belongs to, its file's name, and whether it is a form, which writes a
    // file for anything it declares; a kind's own package, a foundation or an actor writes one only
    // where it declares a Query, a progress claim or a realization (R10).
    def featureOf(pkg: String): Option[(String, String, Boolean)] =
      val parts = pkg.split('.').toList
      val at = parts.indexWhere(Set("features", "foundations", "actors"))
      if at < 0 || parts.sizeIs <= at + 1 then None
      else
        val root = parts.take(at + 2).mkString(".")
        parts.drop(at) match
          case "features" :: kind :: form :: _ if Structure.formFolders(form) =>
            Some((s"$root.$form", s"$kind-$form", true))
          case _ :: feature :: _ => Some((root, feature, false))
          case _                 => None
    def progress(v: ValDef) = isNamed(v.tpt.tpe, "framework.Progress")
    def coreRealization(v: ValDef) =
      !v.symbol.flags.is(Flags.Module) && isNamed(v.tpt.tpe, "framework.realize.Realization")
    val roots = defs.values.toSeq.collect {
      case v: ValDef
          if objectForm(v.symbol) || realizationObject(v.symbol) || queriesSection(v.symbol) ||
            (capabilitiesObject(v.symbol) && objectForm(moduleClassOf(v.symbol).maybeOwner)) ||
            (progress(v) && !v.symbol.flags.is(Flags.Module)) || coreRealization(v) =>
        v
    }
    val byFeature = roots
      .flatMap(v => featureOf(packageOf(v.symbol)).map(f => f -> v))
      .groupBy(_._1)
    val asked = (v: ValDef) =>
      queriesSection(v.symbol) || realizationObject(v.symbol) || progress(v) || coreRealization(v)
    byFeature.toSeq
      .filter { case ((_, _, form), vs) => form || vs.exists((_, v) => asked(v)) }
      .map { case ((_, file, _), vs) =>
        file -> vs.map(_._2).sortBy(v => (pos(v).file, pos(v).line)).map(_.symbol.fullName).distinct
      }
      .sortBy(_._1)
