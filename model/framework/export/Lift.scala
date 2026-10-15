// The lifter: reads the typed trees (TASTy) of Scala Models compiled against the framework,
// and emits the Umpire IR they declare. It lifts what authors wrote, as written: the `machine`
// blocks, the action chains, and the step functions' bodies, including native `match`, `if`,
// `copy` and local `val`s. Unsupported constructs stop the lift at their Scala source positions.
// Go checks Model semantics from the IR at the positions the lifter recorded.
// The IR is built as the gate's generated ScalaPB classes and written as ProtoJSON.
//
// TASTy is read after compilation. A macro at the DSL's capture points could lift the same typed
// bodies during compilation (it sees every `match` intact and, with `-Yretain-trees`, bodies defined
// elsewhere); fn-141 evaluates that move (model/design/scala-macros-export.md).
// The declarations checks read over the machines -- compositions, channels, monitors, assumptions,
// holes, Properties, Scenarios, Queries and progress claims -- are folded at lift time: one written
// through a helper function, such as a list of Queries per design, is lifted once per call, with the
// call's arguments bound.
//
// A realization is emitted as written: its declarations are data, so each constructor becomes the IR
// message of its name and each argument the field of its parameter's name.
package framework.`export`

import com.fasterxml.jackson.core.JsonGenerator
import com.fasterxml.jackson.core.util.DefaultPrettyPrinter
import java.nio.file.{Files, Path, Paths}
import java.util.zip.ZipFile
import scala.collection.mutable
import scala.jdk.CollectionConverters.*
import scala.quoted.*
import scala.tasty.inspector.*
import io.temporal.server.api.umpire.v1 as ir
import org.json4s.jackson.JsonMethods
import scalapb.json4s.Printer
import framework.inspect.Index
import framework.lint.{unrefuted, Admission, GapAnnotations, Markers, Order, Refutation, Subset}

// What one lifter run lifts: the roots named on its command line, or IR files a Model declares.
enum Target:
  // One Model of the declarations these fully qualified names name.
  case Roots(roots: Seq[String])

  // Every IR file the lifted sources declare with `irFile`, or the ones named.
  case IrFiles(names: Seq[String])

// `specs` is the Flow registry a gap's `until` is resolved against, read only where one names a spec.
class Lifter(
    target: Target,
    prefixes: Map[String, String],
    specs: () => SpecRegistry = () => SpecRegistry.flow(Path.of("")),
    stage: String => Unit = _ => (),
    authorLint: Boolean = false,
    runtimeLoader: ClassLoader = classOf[Lifter].getClassLoader
) extends Inspector:
  // The Model of each IR file lifted, by the file's name; a lift of roots has one, named "".
  val models = mutable.LinkedHashMap.empty[String, ir.Model]

  // The waivers of each IR file lifted with a `capabilities` section, by the file's name.
  val waivers = mutable.LinkedHashMap.empty[String, org.json4s.JValue]

  // Every refusal, with the IR file it was lifting: "" for a lift of roots, or for none.
  val errors = mutable.ArrayBuffer.empty[(String, LiftError)]

  // The phase projection each machine or composition object lifted reads, by its name, as its
  // source writes it: its `Phased` argument, or a derived object's source's. Nothing is written of
  // it; the lifter's tests read it.
  val phases = mutable.LinkedHashMap.empty[String, String]

  // The gaps of each IR file lifted, resolved (Gaps.scala), with its capability waivers as gaps. Nothing
  // is written of them: they are the lifter's own until the shared IR carries them (fn-167).
  private[`export`] val gaps = mutable.LinkedHashMap.empty[String, Seq[ResolvedGap]]

  // The provenance of each derived machine of each IR file lifted (fn-167.3), for the same consumer.
  private[`export`] val derivations = mutable.LinkedHashMap.empty[String, Seq[Provenance]]

  // The declarations a gap annotates that some IR file of the run lifted.
  private val placed = mutable.Set.empty[String]
  private lazy val registry = specs()

  // What each negative control of every file lifted asks of the run's Queries.
  private val refutations = mutable.ArrayBuffer.empty[Refutation]
  private val markerInputs = mutable.LinkedHashMap.empty[
    String,
    (Seq[(String, String)], Map[String, Set[String]])
  ]

  // A lift error is kept rather than thrown through the compiler, which would report it as a crash.
  def inspect(using Quotes)(tastys: List[Tasty[quotes.type]]): Unit =
    try
      val index = Index(tastys, prefixes)
      // The sources the lift admits (fn-166.11): those its roots reach, or every one for a lift of
      // every IR file the sources declare. The IR files are read here to find their roots; when the
      // lifter cannot read them, or a file asked for is declared by none, every source is admitted,
      // so the lint reports what may keep it from being declared, and they are read again after it.
      val admission = Admission(index, index.authorSource)
      val admitted = target match
        case Target.Roots(roots)                  => admission.of(roots)
        case Target.IrFiles(only) if only.isEmpty => admission.all
        case Target.IrFiles(only)                 =>
          scala.util.Try(Lifting(Context(index)).irFiles(_ => ())).toOption match
            case Some(declared)
                if only.forall(name => declared.exists(_._1 == name)) &&
                  !declared.forall((name, _) => only.contains(name)) =>
              admission.of(declared.filter((name, _) => only.contains(name)).flatMap(_._2))
            case _ => admission.all
      // The declaration-order lint reads every admitted source first: a Model it refuses would
      // be null or half made where it runs, so nothing of it is lifted.
      stage("lint admitted sources")
      val order = Order(index, admitted, Plugin.installed).refusals ++
        (if authorLint then Subset(index).refusals(file => !admitted(file)).map(_._3) else Nil)
      if order.nonEmpty then errors ++= order.map("" -> _)
      else
        stage("lift IR")
        target match
          case Target.Roots(roots)  => liftFile(index, "", roots)
          case Target.IrFiles(only) =>
            val declared = Lifting(Context(index)).irFiles(errors += "" -> _)
            // A declaration the lifter cannot read may be the one asked for, so nothing is lifted.
            if errors.isEmpty then
              for name <- only.distinct.sorted if !declared.exists(_._1 == name)
              do
                errors += "" -> LiftError(
                  s"IR file $name",
                  "no feature of the lifted sources declares it"
                )
              if declared.isEmpty then
                errors += "" -> LiftError(
                  "IR files",
                  "no feature of the lifted sources declares a Query"
                )
              // Each file is lifted over the one index, with state of its own.
              val chosen = declared.filter((name, _) => only.isEmpty || only.contains(name))
              for (name, roots) <- chosen.sortBy(_._1) do liftFile(index, name, roots)
              // A gap on a declaration no IR file lifts declares nothing the lint reads.
              if only.isEmpty && errors.isEmpty then errors ++= unplaced(index).map("" -> _)
        stage("check lifted markers")
        val markers = Markers(index)
        for (file, model) <- models do
          val (declarations, ruled) = markerInputs(file)
          val (marked, asked) = markers.refusals(model, declarations, ruled, file.nonEmpty)
          errors ++= marked.map(file -> _)
          refutations ++= asked
        errors ++= unrefuted(refutations.toSeq).map("" -> _)
        errors ++= derivedIdTwins(models.toSeq).map("" -> _)
    catch case e: LiftError => errors += "" -> e

  private def liftFile(index: Index, file: String, roots: Seq[String]): Unit =
    try liftRoots(Context(index, runtimeLoader), file, roots)
    catch case e: LiftError => errors += file -> e

  private def liftRoots(ctx: Context, file: String, roots: Seq[String]): Unit =
    import ctx.*
    val concerns = Lifting(ctx)
    concerns.readFinites()
    val firstError = errors.size
    val constructed = mutable.ArrayBuffer.empty[concerns.Pending]

    // Every root is lifted, and one that fails is reported without stopping the others. Nothing is
    // written once one failed, so what a failed root left behind matters only as the functions it
    // was still lifting. A root that lifted then gives each Query it declared that asserts no total
    // the static combination count the lifter computes (model/SEMANTICS.md, Query totals), which Go
    // holds every total to, as it holds an author's.
    for root <- roots.distinct.sorted do
      val before = queries.keySet.toSet
      try
        if concerns.constructedRoot(root) then constructed += concerns.prepareConstructed(root)
        else concerns.liftRoot(root)
        for (name, q) <- queries.toSeq if !before(name) && q.total.isEmpty do
          queries(name) = q.withTotal(concerns.countedTotal(q))
      catch
        case e: LiftError =>
          errors += file -> e
          lifting.clear()

    errors ++= duplicateProperties(functions, properties.values.toSeq).map(file -> _)
    if errors.size == firstError then
      errors ++= concerns.completeConstructed(constructed.toSeq).map(file -> _)

    def sorted[K: Ordering, V](m: collection.Map[K, V]): Seq[V] = m.toSeq.sortBy(_._1).map(_._2)
    models(file) = ir.Model(
      source = "model: " + roots.toList.sorted.mkString(", "),
      types = sorted(types),
      functions = sorted(functions),
      actions = sorted(actions),
      machines = machines.values.toSeq.sortBy(_.name),
      channels = sorted(channels),
      monitors = sorted(monitors),
      assumptions = sorted(assumptions),
      holes = sorted(holes),
      compositions = compositions.values.toSeq.sortBy(_.name),
      properties = sorted(properties),
      scenarios = sorted(scenarios),
      queries = sorted(queries),
      progress = sorted(progress),
      realizations = sorted(realizations)
    )
    markerInputs(file) = (
      machines.toSeq.map((key, m) => key -> m.name) ++
        compositions.toSeq.map((key, c) => key -> c.name),
      concerns.rulesOf.toMap.view.mapValues(_.keySet).toMap
    )
    phases ++= concerns.phases
    for stated <- sectionWaivers(ctx) do waivers(file) = stated
    // Resolved after the Model is made, so nothing the resolution reads changes what is written.
    val (resolved, refused, declarations) = concerns.resolveGaps(registry)
    gaps(file) = resolved
    derivations(file) = ctx.provenance.values.toSeq
    placed ++= declarations
    errors ++= refused.map(file -> _)

  // Each gap of the run whose declaration no IR file lifted.
  private def unplaced(index: Index): Seq[LiftError] =
    val annotations = GapAnnotations(index)
    annotations.declarations.filterNot(d => placed(annotations.name(d))).map { d =>
      LiftError(
        annotations.where(d),
        s"the gap of ${annotations.name(d)} annotates a declaration no IR file lifts: a gap " +
          "names a machine, composition, facet, realization, Property, Query or refinement an " +
          "IR file lifts"
      )
    }

// The refusal of two declarations of one package that derive one ID across the IR files of a run:
// two machines or compositions of one name (`<family>.target.<name>`), or two Queries of one name
// (`<family>.query.<name>`), the family the package their machine is declared in. Scala names no
// two declarations alike, but these IDs are derived from names, so two files of one package may
// declare `completion` twice, and one shared `def` may build `query("completion")` over two
// machines of one package. One declaration lifted into several files is one: the same position
// over the same machine.
private[`export`] def derivedIdTwins(models: Seq[(String, ir.Model)]): Seq[LiftError] =
  def at(p: Option[ir.Position]) = p.fold("")(p => s"${p.file}:${p.line}")
  // Each derived ID with the declaration that derives it: its position and what it is over.
  val declared = models.flatMap { (_, m) =>
    val family = (m.machines.map(x => x.name -> x.family) ++
      m.compositions.map(x => x.name -> x.family)).toMap
    m.machines.map(x => (s"${x.family}.target.${x.name}", (at(x.position), x.name))) ++
      m.compositions.map(x => (s"${x.family}.target.${x.name}", (at(x.position), x.name))) ++
      m.queries.flatMap(q =>
        val over = q.getScenario.machine
        family.get(over).map(f => (s"$f.query.${q.name}", (at(q.position), over)))
      )
  }
  declared
    .groupBy(_._1)
    .toSeq
    .sortBy(_._1)
    .flatMap { (id, ats) =>
      ats.map(_._2).distinct.sorted match
        case (first, firstOver) +: again if again.nonEmpty =>
          again.map((where, over) =>
            val which =
              if where == first then s"this declaration over $over and the one over $firstOver"
              else s"this declaration and the one at $first"
            LiftError(
              where,
              s"$which derive the ID $id: name them apart, since an ID derived from a name is " +
                "unique in its package"
            )
          )
        case _ => Nil
    }

// `lift <model.jar> <classpath file> <out.json> <source prefix> <root>...`: lift the machines named
// by the roots, the fully qualified names of their `val`s, from the Temporal Models in the jar. The
// prefix turns the sources' build-relative paths into repository-relative ones.
// `lift <jar=prefix>,... <classpath file> <out.json> <root>...` reads several jars, each with the
// prefix of its own sources. Either way a root may also name a composition, a Query, a list of Queries
// or a progress claim; every jar's TASTy but the framework's is read; and every root is lifted and
// every refusal reported before anything is written.
//
// `lift --ir <jar=prefix>,... <classpath file> <out directory> [<IR file>...]` lifts the IR files
// the jars declare with `irFile` instead, each into <out directory>/<name>.json: every one, or the
// ones named. The TASTy is read once for all of them, each is lifted with state of its own, and a
// refusal is reported under the file it was lifting; nothing is written once any file failed.
@main def lift(args: String*): Unit =
  val status = LiftRun.run(args, System.err, Switches.fromEnvironment)
  if status != 0 then sys.exit(status)

@main def checkAndLift(args: String*): Unit =
  val status = LiftRun.run(
    args,
    System.err,
    Switches.fromEnvironment,
    stage => System.err.println(s"== $stage"),
    authorLint = true
  )
  if status != 0 then sys.exit(status)

object LiftRun:
  // A lift's end before it writes anything, with the exit status `lift` ends its JVM with.
  final private class Ended(val status: Int) extends Exception(null, null, false, false)

  // One lift as `lift` runs it, with its arguments, in this JVM: its refusals and usage go to
  // `err`, and its exit status is returned rather than ending the JVM, so the lifter's tests lift
  // in-process. `rules` are the switched lifter rules turned on (Switches). Its scratch is deleted
  // when it returns.
  def run(
      args: Seq[String],
      err: java.io.PrintStream,
      rules: Set[String],
      stage: String => Unit = _ => (),
      authorLint: Boolean = false
  ): Int =
    try
      Switches.within(rules)(lifting(args, err, stage, authorLint))
      0
    catch case e: Ended => e.status

  private def lifting(
      args: Seq[String],
      err: java.io.PrintStream,
      stage: String => Unit,
      authorLint: Boolean
  ): Unit =
    def usage(): Nothing =
      err.println(
        "usage: lift <jar=prefix>,... <classpath file> <out.json> <root>...\n" +
          "       lift --ir <jar=prefix>,... <classpath file> <out directory> [<IR file>...]"
      )
      throw Ended(2)
    def jarsOf(jars: String) = jars
      .split(",")
      .toList
      .map(_.split("=", 2) match
        case Array(jar, prefix) => (jar, prefix)
        case Array(jar)         => (jar, ""))
    val (specs, classpathFile, out, target) = args.toList match
      case "--ir" :: jars :: classpath :: out :: names =>
        (jarsOf(jars), classpath, out, Target.IrFiles(names))
      case jars :: classpath :: out :: roots if jars.contains("=") =>
        (jarsOf(jars), classpath, out, Target.Roots(roots))
      case jar :: classpath :: out :: prefix :: roots =>
        (List(jar -> prefix), classpath, out, Target.Roots(roots))
      case _ => usage()
    target match
      case Target.Roots(Nil) =>
        err.println("lift: no roots: name the declarations to lift")
        throw Ended(1)
      case _ => ()
    // The TASTy is read from a scratch directory of its own, deleted when the lift returns.
    def inScratch(scratch: Path): Unit =
      val prefixes = mutable.Map.empty[String, String]
      val tastys = specs.zipWithIndex.flatMap { case ((jar, prefix), i) =>
        val zip = ZipFile(jar)
        zip.entries.asScala
          .filter(e => e.getName.endsWith(".tasty"))
          .map { e =>
            val entry = s"$i/${e.getName}"
            val p = scratch.resolve(entry)
            Files.createDirectories(p.getParent)
            Files.copy(zip.getInputStream(e), p)
            prefixes(entry) = prefix
            p.toString
          }
          .toList
      }.sorted
      val classpath = specs.map(_._1) ++ Files
        .readString(Path.of(classpathFile))
        .trim
        .split(java.io.File.pathSeparator)
        .toList
      val loader = new java.net.URLClassLoader(
        classpath.map(p => Path.of(p).toUri.toURL).toArray,
        classOf[Lifter].getClassLoader
      )
      val lifter = Lifter(
        target,
        prefixes.toMap,
        stage = stage,
        authorLint = authorLint,
        runtimeLoader = loader
      )
      try TastyInspector.inspectAllTastyFiles(tastys, Nil, classpath)(lifter): Unit
      finally loader.close()
      if lifter.errors.nonEmpty then
        if sys.env.contains("LIFT_DEBUG") then lifter.errors.foreach(_._2.printStackTrace())
        // A file's refusals follow a line that names it, as the gate named the file of each lift.
        for (file, refused) <- lifter.errors.groupBy(_._1).toSeq.sortBy(_._1) do
          if file.nonEmpty then err.println(s"lift: the roots of $file.json did not lift:")
          refused.foreach((_, e) => err.println(s"lift: ${e.getMessage}"))
        throw Ended(1)
      def json(value: org.json4s.JValue) =
        JsonMethods.mapper.writer(Pretty()).writeValueAsString(JsonMethods.asJsonNode(value)) + "\n"
      // The waivers of each IR file's `capabilities` sections are written as `<file>.waivers.json`,
      // which the model gate reads and never checks in.
      target match
        case Target.Roots(_) =>
          val model = lifter.models.get("").getOrElse(sys.error("lift: nothing was lifted"))
          Files.writeString(Paths.get(out), json(Printer().toJson(model)))
          for stated <- lifter.waivers.get("") do
            Files.writeString(Paths.get(out.stripSuffix(".json") + ".waivers.json"), json(stated))
        case Target.IrFiles(_) =>
          val directory = Files.createDirectories(Paths.get(out))
          for (file, model) <- lifter.models do
            Files.writeString(directory.resolve(s"$file.json"), json(Printer().toJson(model)))
          for (file, stated) <- lifter.waivers do
            Files.writeString(directory.resolve(s"$file.waivers.json"), json(stated))
    val scratch = Files.createTempDirectory("umpire-lift")
    try inScratch(scratch)
    finally framework.check.Scratch.delete(scratch)

// fn-162 R3: each Property is declared once. An author's Property of a machine whose `when` and
// predicate equal another's of the same machine, up to names and positions, is refused, naming
// both; a capability's generated Properties keep their own identities and are not compared.
private def duplicateProperties(
    functions: collection.Map[String, ir.Function],
    properties: Seq[ir.Property]
): Seq[LiftError] =
  def where(p: ir.Property) = p.position.fold("")(at => s"${at.file}:${at.line}")
  def body(p: ir.Property): String =
    functions.get(p.holds).fold(p.holds) { f =>
      val text = f.body.fold("")(_.toProtoString).replaceAll("(?s)position \\{[^{}]*\\}", "")
      f.params.zipWithIndex.foldLeft(text) { case (t, (param, i)) =>
        t.replace("\"" + param.name + "\"", "\"$" + i + "\"")
      }
    }
  def key(p: ir.Property) =
    (p.machine, p.when.whenClass.map(_.toProtoString), p.when.whenAction, p.transition, body(p))
  properties
    .filter(_.origin.isEmpty)
    .groupBy(key)
    .values
    .filter(_.sizeIs > 1)
    .toSeq
    .flatMap { same =>
      val ordered = same.sortBy(where)
      ordered.tail.map(p =>
        LiftError(
          where(p),
          s"${p.name} states the promise ${ordered.head.name} states at ${where(ordered.head)}, of " +
            s"${p.machine}: declare each Property once, in the facet whose promise it states, and " +
            "ask it from there"
        )
      )
    }

// Jackson's indented layout, with a field's value after `": "` as ProtoJSON is usually written.
final private class Pretty extends DefaultPrettyPrinter:
  override def createInstance(): DefaultPrettyPrinter = Pretty()
  override def writeObjectFieldValueSeparator(g: JsonGenerator): Unit = g.writeRaw(": ")
