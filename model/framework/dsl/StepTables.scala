package framework

import java.lang.reflect.InvocationTargetException
import java.nio.file.{Files, Path}
import scala.collection.mutable
import scala.jdk.CollectionConverters.*

// The step tables of every machine and composition the Models declare, written as text so a
// meaning-preserving change can be checked against a baseline taken before it, without the IR
// (fn-160.5). The tables read the machines as Scala runs them, so they hold while the checked-in IR
// and Cases are stale and while the IR generator does not build. The model gate's
// `--capture-baseline` writes them and its `--check-equivalence` compares them under a rename map
// (model/framework/check/equivalence/Equivalence.scala).
//
// A machine's table is a header and one row per state and class of each bound action, disabled
// pairs included, so a change that enables or disables a pair shows as well as one that changes a
// result:
//
//   init       <state>
//   bindings   <actor.action> ...                in binding order
//   refines    <machine>                         where it declares a refinement
//   state      <state>  end=<bool>  [product <image>]
//   step       <actor.action>[<inputs>]  <state>  { <outcome> | <state> | [<facts>] | <because> ; ... }
//
// A composition's table is its members, the composition it derives from, its syncs, the members
// it replaces, and its end over its members' states. At run time a composition is its members,
// whose own tables hold its steps.
object StepTables:
  // A table file: `<name>.steps`.
  val suffix = ".steps"

  // The ends of a composition are tabled over at most this many composed states.
  private val endLimit = 1_000_000

  // Every machine and composition the IR files of `packagePrefix` declare (IrFile.discover) reach,
  // the declared ones each by its key, and the derivations made in place, such as `m.restrict(...)`
  // passed as a member, in the order found: the discovery the table pins use
  // (model/temporal/MachineTablePins.test.scala).
  def discover(packagePrefix: String): (Vector[(String, Model)], Vector[Model]) =
    val seen = java.util.IdentityHashMap[Model, java.lang.Boolean]()
    val found = IrFile
      .discover(packagePrefix)
      .flatMap(_.reached())
      .filter(m => seen.put(m, java.lang.Boolean.TRUE) == null) // scalafix:ok DisableSyntax.null
    val declared = found.filter(isDeclared).map(m => key(m) -> m)
    val clashes = declared.groupBy(_._1).filter(_._2.sizeIs > 1)
    require(
      clashes.isEmpty,
      "declared models share a key: " + clashes.keys.toVector.sorted.mkString(", ")
    )
    (declared.sortBy(_._1), found.filterNot(isDeclared))

  // A value as the tables and the table pins spell it: a case by its name, followed by its fields,
  // joined by `-`; an option as `None` or `Some-<value>`; a list in brackets.
  def spell(value: Any): String = value match
    case _: None.type                 => "None"
    case Some(member)                 => "Some-" + spell(member)
    case values: List[?]              => values.map(spell).mkString("[", ",", "]")
    case enumCase: scala.reflect.Enum =>
      val product = enumCase.asInstanceOf[Product] // scalafix:ok DisableSyntax.asInstanceOf
      (product.productPrefix +: product.productIterator.map(spell).toSeq).mkString("-")
    case record: Product => record.productIterator.map(spell).mkString("-")
    case other           => other.toString

  // A model's key: its family, the package or object that declares it, and its name,
  // `temporal.features.nexus.workflow.protocol.nexusProtocol`, as the IR names it. A derivation made in
  // place takes its source's.
  private def key(m: Model): String = m match
    case d: DerivedFrom[?] if !isDeclared(m) =>
      d.source match
        case source: Model => key(source)
        case _             => m.name
    case _ => IrFile.identity(m).split('.').dropRight(1).mkString(".") + "." + m.name

  // A machine or composition an object of its own declares, rather than a derivation made in place.
  private def isDeclared(m: Model): Boolean = m match
    case _: Built[?, ?, ?, ?] => false
    case c: Composition[?]    => !c.getClass.getName.contains("$anon")
    case _                    => true

  private def unwrap(what: String)(read: => AnyRef): AnyRef =
    try read
    catch
      case e: InvocationTargetException =>
        throw IllegalStateException(s"$what failed: ${e.getCause}", e.getCause)
      case e: ExceptionInInitializerError =>
        throw IllegalStateException(s"$what failed to initialize: ${e.getCause}", e.getCause)

  // A model's table, as lines.
  def lines(model: Model): Iterator[String] = model match
    case m: Machine[?, ?, ?] => machineLines(m)
    case c: Composition[?]   => compositionLines(c)
    case other => throw IllegalArgumentException(s"${other.name} is no machine or composition")

  // Each bound action's name, `actor.action`, in binding order: as its rules write it, or as it is
  // declared.
  private def actionNames[S, O, F](machine: Machine[S, O, F]): Vector[String] =
    machine.table.map:
      case (decl, Bound.Ruled(rules)) if rules.nonEmpty =>
        s"${decl.actor.name}.${rules.head.action}"
      case (decl, _) => declaredName(decl, machine)

  // An action's name, `actor.action`: its own, or the `val` of its actor or of `also` that declares it.
  private def declaredName(decl: ActionDecl, also: AnyRef*): String =
    val action =
      if decl.name.nonEmpty then decl.name
      else
        (decl.actor +: also)
          .flatMap: owner =>
            owner.getClass.getDeclaredMethods.toVector
              .filter(m =>
                m.getParameterCount.equals(0) && !m.getName.contains('$') &&
                  classOf[Action[?]].isAssignableFrom(m.getReturnType)
              )
              .flatMap(m =>
                unwrap(s"${owner.getClass.getName}.${m.getName}")(m.invoke(owner)) match
                  case a: Action[?] if a.decl eq decl => Some(m.getName)
                  case _                              => None
              )
          .sorted
          .headOption
          .getOrElse("")
    s"${decl.actor.name}.$action"

  private def machineLines[S, O, F](machine: Machine[S, O, F]): Iterator[String] =
    val refinement = machine.refinementEvidence.map(
      _.asInstanceOf[Refinement[S, Any]]
    ) // scalafix:ok DisableSyntax.asInstanceOf
    val bindings = machine.bindings.toVector
    val names = actionNames(machine)
    val header = Vector(
      s"init\t${spell(machine.init)}",
      s"bindings\t${names.mkString(" ")}"
    ) ++ refinement.map(r => s"refines\t${r.of.name}")
    val states = machine.fs.values.iterator.map: state =>
      val image = refinement.fold("")(r => s"\tproduct ${spell(r.toProduct(state))}")
      s"state\t${spell(state)}\tend=${machine.end(state)}$image"
    val steps =
      for
        (binding, name) <- bindings.iterator.zip(names)
        run = effectOf[S, O, F](binding.decl, binding.function)
        inputs <- classesOf(binding.decl).iterator
        state <- machine.fs.values.iterator
      yield
        val result =
          try
            run(state, inputs)
              .map: step =>
                s"${spell(step.outcome)} | ${spell(step.state)} | ${spell(step.facts)} | ${step.because}"
              .mkString("{ ", " ; ", " }")
          catch case HoleReached(hole) => s"hole ${hole.name}"
        s"step\t$name${spell(inputs)}\t${spell(state)}\t$result"
    header.iterator ++ states ++ steps

  private def compositionLines[S <: Product](composition: Composition[S]): Iterator[String] =
    val fields =
      composition.mirror.fromProduct(Composition.Empty).productElementNames.toVector
    val members = composition.members
    // A derivation made in place has no name: it is read as its source with the member it replaces.
    def derivation(c: Composition[S]): String = c.shape match
      case Composition.Shape.Members(_)         => c.name
      case Composition.Shape.Of(source)         => derivation(source)
      case Composition.Shape.With(base, member) =>
        s"${derivation(base)} with ${Composition.selected(member, c.mirror).name}"
    val shape = composition.shape match
      case Composition.Shape.Members(_) => Vector.empty
      case _ => Vector(s"derives\t${derivation(composition).stripPrefix(composition.name + " ")}")
    // A member action as its member's table names it.
    val bound = members.collect { case m: Machine[?, ?, ?] =>
      m.table.map(_._1).zip(actionNames(m))
    }.flatten
    def moved(decl: ActionDecl) = bound
      .collectFirst { case (d, n) if d eq decl => n }
      .getOrElse(declaredName(decl))
    val syncs = Syncs
      .declaredBy(composition)
      .toVector
      .flatMap(_.recorded)
      .map:
        case Syncs.Statement.Sync(name, first, second) =>
          val named = if name.isEmpty then "" else s"$name: "
          s"sync\t$named${moved(first)} ${moved(second)}"
        case Syncs.Statement.Replaces(opaque) => s"replaces\t${opaque.name}"
    (Vector(s"members\t${fields.zip(members).map((f, m) => s"$f=${m.name}").mkString(" ")}") ++
      shape ++ syncs ++ ends(composition, members)).iterator

  // The composed states the composition may end in, over every combination of its members'
  // states, where its members are machines and it declares `end`.
  private def ends[S <: Product](
      composition: Composition[S],
      members: Vector[Model]
  ): Vector[String] =
    // A derived composition keeps its source's end.
    def owner(c: Composition[S]): Composition[S] =
      val declares =
        c.getClass.getMethods.exists(m => m.getName.equals("end") && m.getParameterCount.equals(1))
      c.shape match
        case _ if declares                   => c
        case Composition.Shape.Of(source)    => owner(source)
        case Composition.Shape.With(base, _) => owner(base)
        case Composition.Shape.Members(_)    => c
    val endOwner = owner(composition)
    val declared = endOwner.getClass.getMethods.find(m =>
      m.getName.equals("end") && m.getParameterCount.equals(1)
    )
    val spaces = members.collect { case m: Machine[?, ?, ?] => m.fs.values.toVector }
    val size = spaces.foldLeft(BigInt(1))(_ * _.size)
    declared match
      case None                                   => Vector("end\tsource's")
      case Some(_) if spaces.size != members.size =>
        Vector("end\tnot tabled: a member is a composition")
      case Some(_) if size > endLimit => Vector(s"end\tnot tabled: $size composed states")
      case Some(method)               =>
        val combinations = spaces.foldLeft(Vector(Vector.empty[Any]))((prefixes, space) =>
          for prefix <- prefixes; v <- space yield prefix :+ v
        )
        val ending = combinations.filter: values =>
          val state = composition.mirror.fromProduct(Tuple.fromArray(values.toArray[Any]))
          unwrap(s"${composition.name}.end")(method.invoke(endOwner, state)) match
            case b: java.lang.Boolean => b.booleanValue
            case other => throw IllegalStateException(s"${composition.name}.end gave $other")
        Vector(s"ends\t${ending.size} of ${combinations.size}") ++
          ending.map(values => s"end\t${spell(values.toList)}")

  // Writes the tables of every model the package declares into `dir`, which must hold none: the
  // index, `index.tsv`, one `<key>\t<digest>\t<lines>` line per model in key order, and each
  // distinct table once, gzipped, as `<digest>.steps.gz`, its digest the SHA-256 of its text. A
  // derivation made in place whose table no other model has is keyed `<source's key>~<n>`. Gives
  // the index's lines.
  def write(packagePrefix: String, dir: Path): Vector[String] =
    Files.createDirectories(dir): Unit
    val existing = Files.list(dir)
    try
      require(
        !existing.iterator.asScala.exists(_.getFileName.toString.endsWith(suffix)) &&
          !Files.exists(dir.resolve(index)),
        s"$dir already holds step tables: write them into an empty directory"
      )
    finally existing.close()
    val (declared, anonymous) = discover(packagePrefix)
    val written = mutable.LinkedHashMap.empty[String, Long]
    // Writes one table, and gives its digest and line count.
    def table(model: Model): (String, Long) =
      val scratch = Files.createTempFile(dir, "table", ".tmp")
      val digest = java.security.MessageDigest.getInstance("SHA-256")
      var count = 0L // scalafix:ok DisableSyntax.var
      val out = java.io.BufferedWriter(
        java.io.OutputStreamWriter(
          java.util.zip.GZIPOutputStream(Files.newOutputStream(scratch), 1 << 16),
          java.nio.charset.StandardCharsets.UTF_8
        ),
        1 << 16
      )
      try
        for line <- lines(model) do
          val bytes = (line + "\n").getBytes(java.nio.charset.StandardCharsets.UTF_8)
          digest.update(bytes)
          out.write(line)
          out.write('\n')
          count += 1
      finally out.close()
      val hex = digest.digest().map(b => f"${b & 0xff}%02x").mkString
      if written.contains(hex) then Files.delete(scratch)
      else
        Files.move(scratch, dir.resolve(hex + suffix + ".gz")): Unit
        written.update(hex, count)
      (hex, count)
    val keyed = declared.map((k, m) => k -> table(m))
    val seen = mutable.Set.from(keyed.map(_._2._1))
    val made = mutable.Map.empty[String, Int]
    val extra = anonymous.flatMap: m =>
      val (hex, count) = table(m)
      if seen(hex) then None
      else
        seen += hex
        val n = made.getOrElse(key(m), 0) + 1
        made.update(key(m), n)
        Some(s"${key(m)}~$n" -> (hex, count))
    val rows = (keyed ++ extra).sortBy(_._1).map((k, t) => s"$k\t${t._1}\t${t._2}")
    Files.write(dir.resolve(index), rows.asJava): Unit
    rows

  // The index of a directory of tables.
  val index = "index.tsv"

// Writes the step tables of the Models: `writeStepTables <package> <directory>`. The model gate
// runs it.
@main def writeStepTables(packagePrefix: String, directory: String): Unit =
  val started = System.nanoTime()
  val written = StepTables.write(packagePrefix, Path.of(directory))
  val runtime = Runtime.getRuntime
  println(
    s"wrote ${written.size} step tables to $directory in ${(System.nanoTime() - started) / 1000000} ms" +
      s" (heap used ${(runtime.totalMemory - runtime.freeMemory) >> 20} MiB," +
      s" max ${runtime.maxMemory >> 20} MiB)"
  )
