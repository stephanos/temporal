// The model gate. Scala is only the authoring front end: the Models in temporal/ are compiled and
// tested, the lifter reads their typed trees and emits the IR, and Go interprets the IR and checks
// every table, identity and fingerprint against its goldens. It stops at the first failure.
// The model's own vocabulary is held too, by a Go test outside model/ that every run names.
//
//   scala-cli run model/gate                        lift, require every file of model/ir and
//                                                   model/cases to be current, test
//   scala-cli run model/gate -- --update            lift and rewrite model/ir and model/cases
//   scala-cli run model/gate -- --skip-go-checks    either, without `go vet` and `go test`
//   scala-cli run model/gate -- --generate-ir       package the IR's classes and stop; with
//                                                   --if-stale, only when their inputs changed
//
// The lifter's fixtures under lifter/testdata are built and lifted too, by the lifter's own tests:
// the Models it must lift, compared with the IR in lifter/testdata/lifts/expected, which --update
// rewrites as well, and the declarations the build or the lifter must refuse, at their lines.
package umpire.gate

import java.io.PrintStream
import java.nio.file.{Files, Path}
import java.security.MessageDigest
import scala.concurrent.{blocking, Await, Future}
import scala.concurrent.ExecutionContext.Implicits.global
import scala.concurrent.duration.Duration
import scala.jdk.CollectionConverters.*

/** A check of the gate that failed. */
final class GateError(message: String) extends Exception(message)

/** The gate over the repository `tools` runs in; `log` receives its progress. */
final class Gate(tools: Tools, log: PrintStream):
  private val root = tools.directory
  private val model = root.resolve("model")
  private val gen = model.resolve("gen")
  private val schemaFile = "proto/internal/temporal/server/api/umpire/v1/ir.proto"
  private val schema = root.resolve(schemaFile)
  private val irJar = gen.resolve("ir-proto.jar")
  private val scalaIrJar = gen.resolve("ir-scalapb.jar")
  private val scalaPlugin = gen.resolve("protoc-gen-scala")
  // The version of protobuf-java is the one the lifter runs with (lifter/project.scala).
  private val protobuf = "com.google.protobuf:protobuf-java:4.29.5"
  // ScalaPB's generator and its runtime are released together; this release is built for Scala 3
  // and its generator runs with the pinned protoc.
  private val scalapb = "0.11.20"
  // The lifter reads the ScalaPB classes' TASTy, so they are compiled by the lifter's Scala.
  private val scala = "3.9.0"
  private val modelJar = gen.resolve("model-scala.jar")
  private val modelClasspath = gen.resolve("model-scala.classpath")
  private val dsl = Seq("model/project.scala", "model/umpire")
  // The words the model does not use are spelled in this Go test and not here, so that the gate is
  // no use of them.
  private val vocabulary = "TestModelNamesNoRetiredFrontEnd"
  private val models = dsl :+ "model/temporal"

  private def step[A](title: String)(body: => A): A =
    log.println(s"== $title")
    val started = System.nanoTime()
    val result = body
    log.println(s"   ${(System.nanoTime() - started) / 1000000000}s")
    result

  // Scratch directories are made fresh under the ignored model/gen/history and kept for inspection:
  // nothing is deleted, so another process's build state is never removed.
  private def scratch(name: String): Path =
    Files.createTempDirectory(Files.createDirectories(gen.resolve("history")), s"$name.")

  /**
   * Packages the classes the lifter compiles against, both from the IR schema: the Java classes in
   * model/gen/ir-proto.jar and the ScalaPB classes in model/gen/ir-scalapb.jar. The IR schema is
   * proto/internal/temporal/server/api/umpire/v1/ir.proto; its Go code is generated with every other
   * internal proto by `make protoc` into api/umpire/v1.
   *
   * The jars have a stamp beside them, the hash of the schema and the versions of the generators.
   * With `ifStale`, the jars are packaged only when one of those changed since they were packaged.
   */
  def generateIr(ifStale: Boolean): Unit =
    if !Files.isRegularFile(schema) then throw GateError(s"the IR schema $schema is missing")
    val hash =
      MessageDigest
        .getInstance("SHA-256")
        .digest(Files.readAllBytes(schema))
        .map("%02x".format(_))
        .mkString
    val stamp = s"$hash $protobuf scalapb:$scalapb scala:$scala"
    val stampFile = gen.resolve("ir.stamp")
    val jars = Seq(irJar, scalaIrJar)
    val current = (stampFile +: jars).forall(Files.isRegularFile(_))
      && Files.readString(stampFile).trim == stamp
    if !(ifStale && current) then
      // Whole seconds, no finer than the file times of the file systems the gate writes to.
      val started = System.currentTimeMillis() / 1000 * 1000
      val sources = scratch("schema")
      def generate(out: String, plugin: Option[Path] = None) =
        val arguments = plugin.map(p => s"--plugin=protoc-gen-scala=$p").toSeq ++ Seq(
          "--proto_path=proto/internal",
          out,
          schemaFile.stripPrefix("proto/internal/")
        )
        tools.run("protoc", arguments).orFail()
      def packaged(directory: Path, dependency: String, jar: Path) =
        tools
          .scalaCli(
            Seq(
              "--power",
              "package",
              "--library",
              directory.toString,
              "--scala",
              scala,
              "--dep",
              dependency,
              "-f",
              "-o",
              jar.toString
            )
          )
          .orFail()

      val java = Files.createDirectories(sources.resolve("java"))
      generate(s"--java_out=$java")
      packaged(java, protobuf, irJar)

      // protoc runs a plugin as an executable, so ScalaPB's generator is packaged as a launcher of
      // its own; scala-cli fetches it, as it fetches every other dependency of the model. It has no
      // sources, and the empty directory it is packaged from keeps scala-cli's build state there.
      val launcher = Files.createDirectories(sources.resolve("plugin"))
      tools
        .scalaCli(
          Seq(
            "--power",
            "package",
            launcher.toString,
            "--scala",
            scala,
            "--dep",
            s"com.thesamet.scalapb::compilerplugin:$scalapb",
            "--main-class",
            "scalapb.ScalaPbCodeGenerator",
            "-f",
            "-o",
            scalaPlugin.toString
          )
        )
        .orFail()
      val plugin = root.relativize(scalaPlugin)
      if !Files.isRegularFile(scalaPlugin) || !Files.isExecutable(scalaPlugin) then
        throw GateError(s"the ScalaPB plugin $plugin is missing: scala-cli did not package it")
      if Files.getLastModifiedTime(scalaPlugin).toMillis < started then
        throw GateError(s"the ScalaPB plugin $plugin is stale: scala-cli did not package it")
      val scalaSources = Files.createDirectories(sources.resolve("scala"))
      // flat_package leaves the file's name out of the package, as java_multiple_files does for Java.
      // The ScalaPB classes therefore take the Java classes' names, so the lifter reads one jar.
      generate(s"--scala_out=flat_package,scala3_sources:$scalaSources", Some(scalaPlugin))
      packaged(scalaSources, s"com.thesamet.scalapb::scalapb-runtime:$scalapb", scalaIrJar)

      for jar <- jars do
        if !Files.isRegularFile(jar) then
          throw GateError(
            s"${root.relativize(jar)} is missing: the IR schema $schemaFile was not packaged"
          )
        if Files.getLastModifiedTime(jar).toMillis < started then
          throw GateError(
            s"${root.relativize(jar)} is stale: the IR schema $schemaFile was not packaged into it"
          )
      Files.writeString(stampFile, stamp + "\n")
      jars.foreach(jar => log.println(s"generated ${root.relativize(jar)}"))

  /** The whole gate. An update rewrites the checked-in IR and Cases; a check writes neither. */
  def run(update: Boolean, goChecks: Boolean): Unit =
    Seq("scala-cli", "protoc", "go").foreach(tools.find)

    // Not one of the Go checks a run may skip: it reads the model's sources, not the IR. It needs
    // nothing the gate builds, so it comes first and a mention fails in seconds.
    step("hold the model's sources and documents to its own vocabulary"):
      val arguments = Seq(
        "test",
        "-count=1",
        "-tags",
        "test_dep",
        "-v",
        "-run",
        s"^$vocabulary$$",
        "./tools/umpire/model"
      )
      val ran = tools.run("go", arguments).orFail()
      // A name that matches no test passes too, so the check is required to have run.
      if !ran.output.linesIterator.exists(_.startsWith(s"--- PASS: $vocabulary ")) then
        throw GateError(s"the vocabulary check $vocabulary did not run:\n${ran.diagnostics}")

    step("generate the IR's classes when their inputs changed"):
      generateIr(ifStale = true)
      val generated = root.resolve("api/umpire/v1/ir.pb.go")
      if !Files.isRegularFile(generated)
        || Files.getLastModifiedTime(generated).compareTo(Files.getLastModifiedTime(schema)) <= 0
      then
        throw GateError(
          s"${root.relativize(generated)} is older than the IR schema $schemaFile; run make protoc"
        )

    // The framework must build without the Temporal Models, so nothing in umpire/ reaches into them.
    step("compile the framework alone"):
      tools.scalaCli("compile" +: dsl).orFail()

    step("compile and test the framework and the Temporal Models"):
      tools.scalaCli("test" +: models).orFail()

    step("package the Models' TASTy"):
      tools
        .scalaCli(
          Seq("--power", "package", "--library") ++ models ++ Seq("-f", "-o", modelJar.toString)
        )
        .orFail()
      // Its standard error is not kept, so no printed error is read: the package above built these
      // sources and was read.
      val classpath =
        tools.scalaCli(Seq("compile", "--print-class-path") ++ models, Output.KeptApart).orFail()
      Files.writeString(modelClasspath, classpath.output)

    step("build and lift the lifter's fixtures: the expected IR, and each refusal at its line"):
      // Set either way, so a check never inherits an update from the environment it is run in.
      tools
        .withEnvironment("UMPIRE_LIFTER_UPDATE" -> (if update then "1" else ""))
        .scalaCli(Seq("test", "model/lifter"))
        .orFail()

    step("lift the Nexus caller Model, the activity Models and the Nexus close designs"):
      val lifted = scratch("ir")
      // Each lift is its own JVM, so they run side by side.
      val lifts = Roots.ir.map: (file, roots) =>
        // The lifter's own arguments follow `--`: an argument after it is a root, and the lifter
        // refuses a root that names nothing.
        val arguments = Seq(
          "run",
          "model/lifter",
          "--",
          s"$modelJar=model/",
          modelClasspath.toString,
          lifted.resolve(file).toString
        )
        file -> Future(blocking(tools.scalaCli(arguments ++ roots)))
      // Every lift is waited for, so none outlives the gate, and every one that failed is reported.
      val failures = lifts
        .map((file, lift) => file -> Await.result(lift, Duration.Inf))
        .collect:
          case (file, ran) if ran.failed =>
            s"the roots of model/ir/$file did not lift:\n${ran.diagnostics}"
      if failures.nonEmpty then throw GateError(failures.mkString("\n"))
      Gate.settle(model.resolve("ir"), lifted, root, update)

    step(if update then "generate every lowered Case and the Query manifest"
    else "check every lowered Case and the Query manifest"):
      val arguments =
        Seq("run", "./tools/umpire/cmd/umpire-gen-cases") ++ Option.when(update)("--update")
      tools.run("go", arguments, Output.Shown).orFail()

    // Combined verification may run the complete Go suite separately; the default gate includes it.
    if goChecks then
      step("interpret the IR in Go and hold it to its goldens"):
        tools
          .run("go", Seq("vet", "-tags", "test_dep", "./tools/umpire/..."), Output.Shown)
          .orFail()
        tools
          .run(
            "go",
            Seq("test", "-count=1", "-tags", "test_dep", "./tools/umpire/..."),
            Output.Shown
          )
          .orFail()
    log.println("== ok")

object Gate:
  val usage = "usage: gate [--update] [--skip-go-checks] | gate --generate-ir [--if-stale]"

  private def same(checkedIn: Path, produced: Path) =
    Files.isRegularFile(checkedIn) && Files.mismatch(checkedIn, produced) == -1L

  private def files(directory: Path): Set[String] =
    val stream = Files.list(directory)
    try stream.iterator.asScala.filter(Files.isRegularFile(_)).map(_.getFileName.toString).toSet
    finally stream.close()

  // Where two texts first differ, which is enough to see what moved; both files are kept whole.
  private def difference(checkedIn: String, lifted: String): String =
    val (was, now) = (checkedIn.linesIterator.toVector, lifted.linesIterator.toVector)
    val at = was.map(Some(_)).zipAll(now.map(Some(_)), None, None).indexWhere(_ != _)
    def line(text: Vector[String]) = text.lift(at).fold("nothing")(l => s"`${l.take(200)}`")
    if at < 0 then "its lines are the same and its line endings differ"
    else s"line ${at + 1} is ${line(was)} and lifts to ${line(now)}"

  /**
   * Holds the checked-in `tree` to the `produced` directory, file for file: the tree is the files
   * that were produced and no others. A check writes nothing and fails on every file that is stale,
   * missing or produced by nothing. An update writes the files that changed, and only once every file
   * was produced and none is left over.
   */
  def settle(tree: Path, produced: Path, root: Path, update: Boolean): Unit =
    def name(file: Path) = root.relativize(file)
    val (held, lifted) = (files(tree), files(produced))
    val orphans = (held -- lifted).toSeq.sorted.map(file =>
      s"${name(tree.resolve(file))} is checked in and nothing produces it: remove it or name its roots"
    )
    if update then
      if orphans.nonEmpty then throw GateError(orphans.mkString("\n"))
      for file <- lifted.toSeq.sorted do
        val (target, source) = (tree.resolve(file), produced.resolve(file))
        if !same(target, source) then Files.write(target, Files.readAllBytes(source)): Unit
    else
      val stale = lifted.toSeq.sorted.flatMap: file =>
        val (target, source) = (tree.resolve(file), produced.resolve(file))
        if !Files.isRegularFile(target) then Some(s"${name(target)} is missing")
        else if same(target, source) then None
        else
          val differs = difference(Files.readString(target), Files.readString(source))
          Some(s"${name(target)} is stale: $differs")
      if stale.nonEmpty || orphans.nonEmpty then
        val remedy =
          s"rerun with --update (make umpire-gen-model); the lifted files are in $produced"
        throw GateError((stale ++ orphans :+ remedy).mkString("\n"))

  /** Runs the gate as its command line says and answers its exit status. */
  def main(arguments: Seq[String], tools: => Tools, out: PrintStream, err: PrintStream): Int =
    val known = Set("--update", "--skip-go-checks", "--generate-ir", "--if-stale")
    val flags = arguments.toSet
    val generate = flags("--generate-ir")
    val valid = flags.subsetOf(known) &&
      (if generate then !flags("--update") && !flags("--skip-go-checks") else !flags("--if-stale"))
    if !valid then
      err.println(usage)
      2
    else
      try
        val gate = Gate(tools, out)
        if generate then gate.generateIr(flags("--if-stale"))
        else gate.run(flags("--update"), !flags("--skip-go-checks"))
        0
      catch
        case e: (GateError | ToolError) =>
          err.println(s"gate: ${e.getMessage}")
          1
        // A file the gate reads or writes: the exception names the path and what went wrong.
        case e: java.io.IOException =>
          err.println(s"gate: $e")
          1

@main def run(arguments: String*): Unit =
  sys.exit(Gate.main(arguments, Tools.here, System.out, System.err))
