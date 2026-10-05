/* The lifter: reads the typed trees (TASTy) of Scala Models compiled against the umpire framework,
 * and emits the Umpire IR they declare. It lifts what authors wrote, as written: the `machine`
 * blocks, the action chains, and the step functions' bodies, including native `match`, `if`,
 * `copy` and local `val`s. Unsupported constructs stop the lift at their Scala source positions.
 * Go checks Model semantics from the IR at the positions the lifter recorded.
 * The IR is built as the gate's generated ScalaPB classes and written as ProtoJSON.
 *
 * TASTy is read after compilation rather than by a macro during it: a macro sees a function's body
 * only for definitions of its own compilation run, and there only after pattern matching has been
 * compiled away; TASTy keeps the typed tree with every `match` intact.
 */
// The declarations checks read over the machines -- compositions, channels, monitors, assumptions,
// holes, Properties, Scenarios, Queries and progress claims -- are folded at lift time: one written
// through a helper function, such as a list of Queries per design, is lifted once per call, with the
// call's arguments bound.
//
// A realization is emitted as written: its declarations are data, so each constructor becomes the IR
// message of its name and each argument the field of its parameter's name.
package umpire.irgen

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

/** A construct the IR cannot express, at the position the author wrote it. */
final case class LiftError(position: String, message: String)
    extends Exception(s"$position: $message")

/** What one lifter run lifts: the roots named on its command line, or IR files a Model declares. */
enum Target:
  /** One Model of the declarations these fully qualified names name. */
  case Roots(roots: Seq[String])

  /** Every IR file the lifted sources declare with `irFile`, or the ones named. */
  case IrFiles(names: Seq[String])

class Lifter(target: Target, prefixes: Map[String, String]) extends Inspector:
  /** The Model of each IR file lifted, by the file's name; a lift of roots has one, named "". */
  val models = mutable.LinkedHashMap.empty[String, ir.Model]

  /** The law sidecar of each IR file lifted that declares capabilities, by the file's name. */
  val sidecars = mutable.LinkedHashMap.empty[String, org.json4s.JValue]

  /** Every refusal, with the IR file it was lifting: "" for a lift of roots, or for none. */
  val errors = mutable.ArrayBuffer.empty[(String, LiftError)]

  // A lift error is kept rather than thrown through the compiler, which would report it as a crash.
  def inspect(using Quotes)(tastys: List[Tasty[quotes.type]]): Unit =
    try
      val index = Index(tastys, prefixes)
      // The declaration-order lint reads every inspected source first: a Model it refuses would
      // be null or half made where it runs, so nothing of it is lifted.
      val order = Order(index).refusals
      if order.nonEmpty then errors ++= order.map("" -> _)
      else
        target match
          case Target.Roots(roots)  => liftFile(index, "", roots)
          case Target.IrFiles(only) =>
            val declared = Lifting(Context(index)).irFiles(errors += "" -> _)
            // A declaration the lifter cannot read may be the one asked for, so nothing is lifted.
            if errors.isEmpty then
              for name <- only.distinct.sorted if !declared.exists(_._1 == name) do
                errors += "" -> LiftError(
                  s"IR file $name",
                  "no irFile of the lifted sources declares it"
                )
              if declared.isEmpty then
                errors += "" -> LiftError("IR files", "the lifted sources declare no irFile")
              // Each file is lifted over the one index, with state of its own.
              val chosen = declared.filter((name, _) => only.isEmpty || only.contains(name))
              for (name, roots) <- chosen.sortBy(_._1) do liftFile(index, name, roots)
    catch case e: LiftError => errors += "" -> e

  private def liftFile(index: Index, file: String, roots: Seq[String]): Unit =
    try liftRoots(Context(index), file, roots)
    catch case e: LiftError => errors += file -> e

  private def liftRoots(ctx: Context, file: String, roots: Seq[String]): Unit =
    import ctx.*
    val concerns = Lifting(ctx)
    concerns.readFinites()

    // Every root is lifted, and one that fails is reported without stopping the others. Nothing is
    // written once one failed, so what a failed root left behind matters only as the functions it
    // was still lifting. A root that lifted is then held to a total on each Query it declared; one
    // that failed already reports no Query of its own as missing one.
    for root <- roots.distinct.sorted do
      val before = queries.keySet.toSet
      try
        concerns.liftRoot(root)
        for (name, q) <- queries if !before(name) && q.total.isEmpty do
          val at = q.getPosition
          errors += file -> LiftError(
            s"${at.file}:${at.line}",
            s"Query $name asserts no total: write `.total(n)` with n its static combination " +
              "count, which model/README.md shows how to compute"
          )
      catch
        case e: LiftError =>
          errors += file -> e
          lifting.clear()

    // What a machine is for, as its markers say, is held to what this file lifted with it.
    errors ++= concerns.markerRefusals().map(file -> _)

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
    for laws <- lawSidecar(ctx) do sidecars(file) = laws

/**
 * `lift <model.jar> <classpath file> <out.json> <source prefix> <root>...`: lift the machines named
 * by the roots, the fully qualified names of their `val`s, from the Temporal Models in the jar. The
 * prefix turns the sources' build-relative paths into repository-relative ones.
 */
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
  def usage(): Nothing =
    System.err.println(
      "usage: lift <jar=prefix>,... <classpath file> <out.json> <root>...\n" +
        "       lift --ir <jar=prefix>,... <classpath file> <out directory> [<IR file>...]"
    )
    sys.exit(2)
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
      System.err.println("lift: no roots: name the declarations to lift")
      sys.exit(1)
    case _ => ()
  val scratch = Files.createTempDirectory("umpire-lift")
  val prefixes = mutable.Map.empty[String, String]
  val tastys = specs.zipWithIndex.flatMap { case ((jar, prefix), i) =>
    val zip = ZipFile(jar)
    zip.entries.asScala
      .filter(e => lifted(e.getName) && e.getName.endsWith(".tasty"))
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
  val lifter = Lifter(target, prefixes.toMap)
  TastyInspector.inspectAllTastyFiles(tastys, Nil, classpath)(lifter)
  if lifter.errors.nonEmpty then
    if sys.env.contains("LIFT_DEBUG") then lifter.errors.foreach(_._2.printStackTrace())
    // A file's refusals follow a line that names it, as the gate named the file of each lift.
    for (file, refused) <- lifter.errors.groupBy(_._1).toSeq.sortBy(_._1) do
      if file.nonEmpty then System.err.println(s"lift: the roots of $file.json did not lift:")
      refused.foreach((_, e) => System.err.println(s"lift: ${e.getMessage}"))
    sys.exit(1)
  def json(value: org.json4s.JValue) =
    JsonMethods.mapper.writer(Pretty()).writeValueAsString(JsonMethods.asJsonNode(value)) + "\n"
  // Each IR file's law sidecar, where it declares capabilities, is written beside it as
  // `<file>.laws.json`.
  target match
    case Target.Roots(_) =>
      val model = lifter.models.get("").getOrElse(sys.error("lift: nothing was lifted"))
      Files.writeString(Paths.get(out), json(Printer().toJson(model)))
      for laws <- lifter.sidecars.get("") do
        Files.writeString(Paths.get(out.stripSuffix(".json") + ".laws.json"), json(laws))
    case Target.IrFiles(_) =>
      val directory = Files.createDirectories(Paths.get(out))
      for (file, model) <- lifter.models do
        Files.writeString(directory.resolve(s"$file.json"), json(Printer().toJson(model)))
      for (file, laws) <- lifter.sidecars do
        Files.writeString(directory.resolve(s"$file.laws.json"), json(laws))

/** Whether a jar entry is of the lifted sources: a Model's, never the framework's. */
private def lifted(entry: String): Boolean = !entry.startsWith("umpire/")

/** Jackson's indented layout, with a field's value after `": "` as ProtoJSON is usually written. */
final private class Pretty extends DefaultPrettyPrinter:
  override def createInstance(): DefaultPrettyPrinter = Pretty()
  override def writeObjectFieldValueSeparator(g: JsonGenerator): Unit = g.writeRaw(": ")
