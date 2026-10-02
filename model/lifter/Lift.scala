/* The lifter: reads the typed trees (TASTy) of Scala Models compiled against the umpire framework,
 * and emits the Umpire IR they declare. It lifts what authors wrote, as written: the `machine`
 * blocks, the action chains, and the step functions' bodies, including native `match`, `if`,
 * `copy` and local `val`s. Anything outside the subset stops the lift with the source position of
 * the construct, so a Model that cannot become IR is reported where it was written.
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
package umpire.lift

import com.google.protobuf.util.JsonFormat
import java.nio.file.{Files, Path, Paths}
import java.util.zip.ZipFile
import scala.collection.mutable
import scala.jdk.CollectionConverters.*
import scala.quoted.*
import scala.tasty.inspector.*
import io.temporal.server.api.umpire.v1 as ir

/** A construct the IR cannot express, at the position the author wrote it. */
final case class LiftError(position: String, message: String)
    extends Exception(s"$position: $message")

class Lifter(roots: Seq[String], prefixes: Map[String, String]) extends Inspector:
  val models = mutable.ArrayBuffer.empty[ir.Model]
  val errors = mutable.ArrayBuffer.empty[LiftError]

  // A lift error is kept rather than thrown through the compiler, which would report it as a crash.
  def inspect(using Quotes)(tastys: List[Tasty[quotes.type]]): Unit =
    try liftAll(tastys)
    catch case e: LiftError => errors += e

  private def liftAll(using Quotes)(tastys: List[Tasty[quotes.type]]): Unit =
    val ctx = Context(tastys, prefixes)
    import ctx.*
    val concerns = Lifting(ctx)
    concerns.readFinites()

    // Every root is lifted, and one that fails is reported without stopping the others. Nothing is
    // written once one failed, so what a failed root left behind matters only as the functions it
    // was still lifting.
    for root <- roots.distinct.sorted do
      try concerns.liftRoot(root)
      catch
        case e: LiftError =>
          errors += e
          lifting.clear()

    val m = ir.Model.newBuilder().setSource("model: " + roots.toList.sorted.mkString(", "))
    types.toList.sortBy(_._1).foreach((_, t) => m.addTypes(t))
    functions.toList.sortBy(_._1).foreach((_, f) => m.addFunctions(f))
    actions.toList.sortBy(_._1).foreach((_, a) => m.addActions(a))
    machines.values.toList.sortBy(_.getName).foreach(m.addMachines)
    channels.toList.sortBy(_._1).foreach((_, c) => m.addChannels(c))
    monitors.toList.sortBy(_._1).foreach((_, x) => m.addMonitors(x))
    assumptions.toList.sortBy(_._1).foreach((_, a) => m.addAssumptions(a))
    holes.toList.sortBy(_._1).foreach((_, h) => m.addHoles(h))
    compositions.values.toList.sortBy(_.getName).foreach(m.addCompositions)
    properties.toList.sortBy(_._1).foreach((_, p) => m.addProperties(p))
    scenarios.toList.sortBy(_._1).foreach((_, s) => m.addScenarios(s))
    queries.toList.sortBy(_._1).foreach((_, q) => m.addQueries(q))
    progress.toList.sortBy(_._1).foreach((_, p) => m.addProgress(p))
    realizations.toList.sortBy(_._1).foreach((_, r) => m.addRealizations(r))
    models += m.build()

/**
 * `lift <model.jar> <classpath file> <out.json> <source prefix> <root>...`: lift the machines named
 * by the roots, the fully qualified names of their `val`s, from the Temporal Models in the jar. The
 * prefix turns the sources' build-relative paths into repository-relative ones.
 */
// `lift <jar=prefix>,... <classpath file> <out.json> <root>...` reads several jars, each with the
// prefix of its own sources. Either way a root may also name a composition, a Query, a list of Queries
// or a progress claim; every jar's TASTy but the framework's is read; and every root is lifted and
// every refusal reported before anything is written.
@main def lift(args: String*): Unit =
  val (specs, classpathFile, out, roots) = args.toList match
    case jars :: classpath :: out :: roots if jars.contains("=") =>
      val specs = jars
        .split(",")
        .toList
        .map(_.split("=", 2) match
          case Array(jar, prefix) => (jar, prefix)
          case Array(jar)         => (jar, ""))
      (specs, classpath, out, roots)
    case jar :: classpath :: out :: prefix :: roots => (List(jar -> prefix), classpath, out, roots)
    case _                                          =>
      System.err.println("usage: lift <jar=prefix>,... <classpath file> <out.json> <root>...")
      sys.exit(2)
  val scratch = Files.createTempDirectory("umpire-lift")
  val prefixes = mutable.Map.empty[String, String]
  val tastys = specs.zipWithIndex.flatMap { case ((jar, prefix), i) =>
    val zip = ZipFile(jar)
    zip.entries.asScala
      .filter(e => !e.getName.startsWith("umpire/") && e.getName.endsWith(".tasty"))
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
  if roots.isEmpty then
    System.err.println("lift: no roots: name the declarations to lift")
    sys.exit(1)
  val lifter = Lifter(roots, prefixes.toMap)
  TastyInspector.inspectAllTastyFiles(tastys, Nil, classpath)(lifter)
  if lifter.errors.nonEmpty then
    if sys.env.contains("LIFT_DEBUG") then lifter.errors.foreach(_.printStackTrace())
    lifter.errors.foreach(e => System.err.println(s"lift: ${e.getMessage}"))
    sys.exit(1)
  val json = JsonFormat
    .printer()
    .print(lifter.models.headOption.getOrElse(sys.error("lift: nothing was lifted")))
  Files.writeString(Paths.get(out), json + "\n")
