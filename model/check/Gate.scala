// The model gate. Scala declares and compiles the Models in temporal/; the IR generator reads their
// typed trees and emits ProtoJSON using ScalaPB. Go alone evaluates the IR, reports Model errors at
// Scala source lines, and checks tables, identities and fingerprints against goldens. It stops on error.
// The model's own vocabulary is held too, by a Go test outside model/ that every run names.
//
//   scala-cli run model/check                        lift, require every file of model/ir and
//                                                    model/cases to be current, lint, test
//   scala-cli run model/check -- --update            lift and rewrite model/ir and model/cases
//   scala-cli run model/check -- --skip-go-checks    either, without `go vet` and `go test`
//   scala-cli run model/check -- --generate-ir       package the IR's classes and stop; with
//                                                    --if-stale, only when their inputs changed
//   scala-cli run model/check -- --generate-api      package linked API classes; --if-stale reuses
//                                                    a jar whose descriptors and tools are current
//   scala-cli run model/check -- --check-syntax      hold the sugar to the Syntax.scala files and
//                                                    their `Core form:` docs (SyntaxRule.scala);
//                                                    make lint-model runs it
//   scala-cli run model/check -- --check-comments    hold every Scala file of model/ to `//`
//                                                    comments (CommentRule.scala); make lint-model
//                                                    runs it
//   scala-cli run model/check -- --check-block-form  hold every rule of model/temporal to the
//                                                    block form, `on(...) { ... }`
//                                                    (BlockFormRule.scala)
//
// The IR generator's fixtures under irgen/testdata are built and lifted too, by its own tests: the
// Models it must lift, compared with the IR in irgen/testdata/lifts/expected, which --update
// rewrites as well, and the declarations the build or the IR generator must refuse, at their lines.
package umpire.check

import java.io.PrintStream
import java.nio.file.{Files, Path, StandardCopyOption}
import java.security.MessageDigest
import scala.concurrent.{blocking, Await, Future}
import scala.concurrent.ExecutionContext.Implicits.global
import scala.concurrent.duration.Duration
import scala.jdk.CollectionConverters.*
import scala.util.Try

// A check of the gate that failed.
final class GateError(message: String) extends Exception(message)

// The gate over the repository `tools` runs in; `log` receives its progress.
final class Gate(tools: Tools, log: PrintStream):
  private val root = tools.directory
  private val model = root.resolve("model")
  private val build = model.resolve("build")
  private val schemaFile = "proto/internal/temporal/server/api/umpire/v1/ir.proto"
  private val schema = root.resolve(schemaFile)
  private val irJar = build.resolve("ir-scalapb.jar")
  private val apiDescriptor = root.resolve("proto/api.binpb")
  private val apiJar = build.resolve("api-scalapb.jar")
  private val apiOptions = "flat_package,scala3_sources,grpc"
  private val scalaPlugin = build.resolve("protoc-gen-scala")
  // ScalaPB's generator and its runtime are released together; this release is built for Scala 3
  // and its generator runs with the pinned protoc.
  private val scalapb = "0.11.20"
  // The lifter reads the ScalaPB classes' TASTy, so they are compiled by the lifter's Scala.
  private val scala = "3.9.0"
  private val modelJar = build.resolve("model-scala.jar")
  private val modelClasspath = build.resolve("model-scala.classpath")
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

  // A step started beside others: its title, its output or failure, and how long it took.
  private type Beside = (String, Future[(Try[String], Long)])

  // Its output is kept, so steps that run side by side do not interleave; `report` prints it.
  private def beside(title: String)(body: => String): Beside =
    title -> Future(blocking {
      val started = System.nanoTime()
      val result = Try(body)
      (result, System.nanoTime() - started)
    })

  // Every step is waited for, so none outlives the gate, and is reported in the order given, so the
  // log reads the same whichever ended first. Every one that failed is reported, after `failed`.
  private def report(steps: Seq[Beside], failed: Option[Throwable] = None): Unit =
    val failures = failed.toSeq ++ steps.flatMap: (title, running) =>
      val (result, took) = Await.result(running, Duration.Inf)
      log.println(s"== $title")
      result.foreach(output => if output.nonEmpty then log.println(output.stripLineEnd))
      log.println(s"   ${took / 1000000000}s")
      result.failed.toOption
    failures match
      case Seq()        => ()
      case Seq(failure) => throw failure
      case many         =>
        throw GateError(
          many
            .map {
              case e: (GateError | ToolError) => e.getMessage
              case e                          => e.toString
            }
            .mkString("\n")
        )

  // Scratch directories are made fresh under the ignored model/build/history and kept for
  // inspection: nothing is deleted, so another process's build state is never removed.
  private def scratch(name: String): Path =
    Files.createTempDirectory(Files.createDirectories(build.resolve("history")), s"$name.")

  // Packages the classes the lifter compiles against: model/build/ir-scalapb.jar, the IR's ScalaPB
  // classes, compiled against scalapb-runtime. The IR schema is the file
  // proto/internal/temporal/server/api/umpire/v1/ir.proto; its Go code is generated with every
  // other internal proto by `make protoc` into api/umpire/v1.
  //
  // The jar has a stamp beside it, the hash of the schema and the versions of the generator and of
  // Scala. With `ifStale`, the jar is packaged only when one of those changed since it was packaged.
  def generateIr(ifStale: Boolean): Unit =
    if !Files.isRegularFile(schema) then throw GateError(s"the IR schema $schema is missing")
    val hash =
      MessageDigest
        .getInstance("SHA-256")
        .digest(Files.readAllBytes(schema))
        .map("%02x".format(_))
        .mkString
    val stamp = s"$hash scalapb:$scalapb scala:$scala"
    val stampFile = build.resolve("ir.stamp")
    val current = Seq(irJar, stampFile).forall(Files.isRegularFile(_))
      && Files.readString(stampFile).trim == stamp
    if !(ifStale && current) then
      // An output of an earlier run is removed first, so one scala-cli did not write is missing
      // rather than taken for new. File times cannot tell: a mounted file system may stamp them with
      // another clock than the gate's.
      Seq(scalaPlugin, irJar).foreach(Files.deleteIfExists)
      val sources = scratch("schema")

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

      val classes = Files.createDirectories(sources.resolve("ir"))
      // flat_package leaves the file's name out of the package, so the classes sit in the schema's
      // java_package, io.temporal.server.api.umpire.v1, which the lifter imports.
      tools
        .run(
          "protoc",
          Seq(
            s"--plugin=protoc-gen-scala=$scalaPlugin",
            "--proto_path=proto/internal",
            s"--scala_out=flat_package,scala3_sources:$classes",
            schemaFile.stripPrefix("proto/internal/")
          )
        )
        .orFail()
      tools
        .scalaCli(
          Seq(
            "--power",
            "package",
            "--library",
            classes.toString,
            "--scala",
            scala,
            "--dep",
            s"com.thesamet.scalapb::scalapb-runtime:$scalapb",
            "-f",
            "-o",
            irJar.toString
          )
        )
        .orFail()

      val jar = root.relativize(irJar)
      if !Files.isRegularFile(irJar) then
        throw GateError(s"$jar is missing: the IR schema $schemaFile was not packaged")
      Files.writeString(stampFile, stamp + "\n")
      log.println(s"generated $jar")

  // Packages the linked Temporal API and Testpilot classes for Model authoring and lifting.
  def generateApi(ifStale: Boolean): Unit =
    if !Files.isRegularFile(apiDescriptor) then
      throw GateError("proto/api.binpb is missing; run make proto/api.binpb")
    val checked = tools.run("make", Seq("-q", "proto/api.binpb"))
    if checked.exit == 1 then throw GateError("proto/api.binpb is stale; run make proto/api.binpb")
    checked.orFail()
    generateIr(ifStale = true)

    val sources = scratch("api")
    val internal = sources.resolve("current-internal.binpb")
    val testpilot = root.resolve("proto/internal/temporal/server/api/testpilot")
    if !Files.isDirectory(testpilot) then
      throw GateError("Testpilot proto sources are missing; run make protoc")
    val stream = Files.walk(testpilot)
    val internalNames =
      try
        stream.iterator.asScala
          .filter(path => Files.isRegularFile(path) && path.toString.endsWith(".proto"))
          .map(root.resolve("proto/internal").relativize(_).toString)
          .toSeq
          .sorted :+ schemaFile.stripPrefix("proto/internal/")
      finally stream.close()
    if internalNames.size < 2 then
      throw GateError("Testpilot proto sources are missing; run make protoc")
    tools
      .run(
        "protoc",
        Seq(
          "--descriptor_set_in=proto/api.binpb",
          "--proto_path=proto/internal",
          s"--descriptor_set_out=$internal"
        ) ++ internalNames
      )
      .orFail()
    if !Files.isRegularFile(internal) then
      throw GateError("current internal descriptors are missing; run make protoc")
    val descriptor = sources.resolve("linked.binpb")
    val linked = tools
      .run(
        "go",
        Seq(
          "run",
          "./cmd/tools/getproto",
          "--linked-api",
          "proto/api.binpb",
          "--current-internal",
          internal.toString,
          "--out",
          descriptor.toString
        )
      )
      .orFail()
    val names = linked.output.linesIterator.filter(_.endsWith(".proto")).toSeq
    if !Files.isRegularFile(descriptor) ||
      !names.contains("temporal/api/workflowservice/v1/service.proto") ||
      !names.exists(_.startsWith("temporal/server/api/testpilot/"))
    then throw GateError("linked API descriptor is incomplete; run make proto/api.binpb")

    val digest = MessageDigest.getInstance("SHA-256")
    val inputs = Seq(
      apiDescriptor,
      descriptor,
      root.resolve("cmd/tools/getproto/main.go"),
      root.resolve("cmd/tools/getproto/files.go"),
      root.resolve("go.mod"),
      root.resolve("go.sum"),
      scalaPlugin,
      root.resolve("model/check/Gate.scala"),
      root.resolve("model/check/project.scala")
    )
    for source <- inputs do
      val bytes = Files.readAllBytes(source)
      digest.update(java.nio.ByteBuffer.allocate(8).putLong(bytes.length.toLong).array())
      digest.update(bytes)
    val protocVersion = tools.run("protoc", Seq("--version")).orFail().output.trim
    val cliVersion = tools.run("scala-cli", Seq("version")).orFail().output.trim
    digest.update(
      s"scalapb:$scalapb scala:$scala protoc:$protocVersion scala-cli:$cliVersion options:$apiOptions runtime-grpc:$scalapb"
        .getBytes(java.nio.charset.StandardCharsets.UTF_8)
    )
    val stamp = digest.digest().map("%02x".format(_)).mkString +
      s" options:$apiOptions scalapb:$scalapb scala:$scala"
    val stampFile = build.resolve("api.stamp")
    val current = Seq(apiJar, stampFile).forall(Files.isRegularFile(_)) &&
      Files.readString(stampFile).trim == stamp
    if !(ifStale && current) then
      Files.deleteIfExists(stampFile)
      val classes = Files.createDirectories(sources.resolve("classes"))
      tools
        .run(
          "protoc",
          Seq(
            s"--plugin=protoc-gen-scala=$scalaPlugin",
            s"--descriptor_set_in=$descriptor",
            s"--scala_out=$apiOptions:$classes"
          ) ++ names
        )
        .orFail()
      val packaged = sources.resolve("api-scalapb.jar")
      tools
        .scalaCli(
          Seq(
            "--power",
            "package",
            "--library",
            classes.toString,
            "--scala",
            scala,
            "--dep",
            s"com.thesamet.scalapb::scalapb-runtime-grpc:$scalapb",
            "-f",
            "-o",
            packaged.toString
          )
        )
        .orFail()
      if !Files.isRegularFile(packaged) then
        throw GateError(
          "model/build/api-scalapb.jar is missing: linked API classes were not packaged"
        )
      Files.move(
        packaged,
        apiJar,
        StandardCopyOption.ATOMIC_MOVE,
        StandardCopyOption.REPLACE_EXISTING
      )
      Files.writeString(stampFile, stamp + "\n")
      log.println("generated model/build/api-scalapb.jar")

  // The whole gate. An update rewrites the checked-in IR and Cases; a check writes neither.
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
        "./tools/umpire/ir"
      )
      val ran = tools.run("go", arguments).orFail()
      // A name that matches no test passes too, so the check is required to have run.
      if !ran.output.linesIterator.exists(_.startsWith(s"--- PASS: $vocabulary ")) then
        throw GateError(s"the vocabulary check $vocabulary did not run:\n${ran.diagnostics}")

    step("reject free-text protobuf names in Models"):
      ProtoLiterals.check(model.resolve("temporal"), root)

    // A check lowers the checked-in IR, which it never writes, so its Cases are checked beside the
    // build and the lifts. An update lowers the IR it has just written.
    val cases =
      if update then Nil
      else Seq(beside("check every lowered Case and the Query manifest")(lower(update)))
    report(cases, Try(build(update)).failed.toOption)
    if update then step("generate every lowered Case and the Query manifest")(lower(update)): Unit

    // Not one of the Go checks a run may skip: it is the gate's check of the IR it settled.
    // It fails on a finding no acceptance matches and on a stale acceptance, never on a count. The
    // gate has already held the accepted findings to the capabilities sections' waivers above.
    step("lint every IR file and print its coverage"):
      tools
        .run("go", Seq("run", "./tools/umpire/cmd/umpire-lint"), Output.Shown)
        .orFail(): Unit

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

  // An update's output is shown as it runs; a check's is kept, since it runs beside other steps.
  private def lower(update: Boolean): String =
    if update then
      tools
        .run("go", Seq("run", "./tools/umpire/cmd/umpire-gen-cases", "--update"), Output.Shown)
        .orFail()
      ""
    else tools.run("go", Seq("run", "./tools/umpire/cmd/umpire-gen-cases")).orFail().output

  // Generates and compiles what the lifter reads, then builds its fixtures and lifts the Models.
  private def build(update: Boolean): Unit =
    step("generate the IR's classes when their inputs changed"):
      generateIr(ifStale = true)
      val generated = root.resolve("api/umpire/v1/ir.pb.go")
      if !Files.isRegularFile(generated)
        || Files.getLastModifiedTime(generated).compareTo(Files.getLastModifiedTime(schema)) <= 0
      then
        throw GateError(
          s"${root.relativize(generated)} is older than the IR schema $schemaFile; run make protoc"
        )

    step("generate the linked API's classes when their inputs changed"):
      generateApi(ifStale = true)

    // The framework must build without the Temporal Models, so nothing in umpire/ reaches into them.
    step("compile the framework alone"):
      tools.scalaCli("compile" +: dsl).orFail()

    step("compile and test the framework and the Temporal Models"):
      tools.scalaCli(Seq("test", "--server=false") ++ models).orFail()

    step("package the Models' TASTy"):
      tools
        .scalaCli(
          Seq("--power", "package", "--server=false", "--library") ++ models ++ Seq(
            "-f",
            "-o",
            modelJar.toString
          )
        )
        .orFail()
      // Its standard error is not kept, so no printed error is read: the package above built these
      // sources and was read.
      val classpath =
        tools
          .scalaCli(
            Seq("compile", "--server=false", "--print-class-path") ++ models,
            Output.KeptApart
          )
          .orFail()
      Files.writeString(modelClasspath, classpath.output)

    // Compiled once here, so the steps below that run it side by side only read its classes.
    step("compile the lifter and its tests"):
      tools.scalaCli(Seq("compile", "--test", "model/irgen")).orFail()

    // Both read the packaged Models and the lifter, and neither reads what the other writes.
    val fixtures = beside(
      "build and lift the lifter's fixtures: the expected IR, and each refusal at its line"
    ):
      // Set either way, so a check never inherits an update from the environment it is run in.
      tools
        .withEnvironment("UMPIRE_LIFTER_UPDATE" -> (if update then "1" else ""))
        .scalaCli(Seq("test", "model/irgen"))
        .orFail()
      ""
    val lifted = scratch("ir")
    val modelLifts = beside("lift every IR file the Models declare"):
      // One lifter run reads the Models' TASTy once and writes every IR file they declare with
      // `irFile`, each lifted apart; its refusals name the file they were lifting.
      val arguments = Seq(
        "run",
        "model/irgen",
        "--main-class",
        "umpire.irgen.lift",
        "--",
        "--ir",
        s"$modelJar=model/",
        modelClasspath.toString,
        lifted.toString
      )
      val ran = tools.scalaCli(arguments)
      if ran.failed then throw GateError(s"the Models' IR files did not lift:\n${ran.diagnostics}")
      ""
    report(Seq(fixtures, modelLifts))
    // After both, so an update rewrites model/ir only once the lifter's fixtures passed too.
    Gate.settle(model.resolve("ir"), lifted, root, update)
    Gate.acceptWaivers(model.resolve("ir"), lifted, root, update)

object Gate:
  val usage =
    "usage: gate [--update] [--skip-go-checks] | gate --generate-ir|--generate-api [--if-stale]" +
      " | gate --check-syntax | gate --check-comments | gate --check-block-form"

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

  // The accepted lint findings beside an IR file, `<file>.lint.json`: an author writes them and the
  // lifter does not, and umpire-lint fails on one beside no IR file.
  private val acceptedSuffix = ".lint.json"

  // The waivers the `capabilities` sections of an IR file state, which the lifter writes beside
  // it, `<file>.waivers.json`: never checked in, they are written into its accepted findings.
  private val waiversSuffix = ".waivers.json"

  // Holds the checked-in `tree` to the `produced` directory, file for file: the tree is the files
  // that were produced and no others, apart from the accepted lint findings an author writes. A
  // check writes nothing and fails on every file that is stale, missing or produced by nothing. An
  // update writes the files that changed, and only once every file was produced and none is left
  // over.
  def settle(tree: Path, produced: Path, root: Path, update: Boolean): Unit =
    def name(file: Path) = root.relativize(file)
    val (held, lifted) = (
      files(tree).filterNot(_.endsWith(acceptedSuffix)),
      files(produced).filterNot(_.endsWith(waiversSuffix))
    )
    val orphans = (held -- lifted).toSeq.sorted.map(file =>
      s"${name(tree.resolve(file))} is checked in and nothing produces it: remove it or declare it with irFile"
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

  // Holds the accepted findings beside each IR file of `tree` to the waivers its `capabilities`
  // sections state, as the lift into `produced` wrote them: each waiver accepted under its key,
  // `<machine>.<property>`, for its reason (`withWaivers`). An update writes the findings that
  // differ; a check writes nothing and fails on each file missing a waiver's acceptance or holding
  // a stale one. A file that would accept nothing is not created.
  def acceptWaivers(tree: Path, produced: Path, root: Path, update: Boolean): Unit =
    val differing = files(produced).filter(_.endsWith(waiversSuffix)).toSeq.sorted.flatMap { file =>
      val (machines, waivers) = Accepted.waivers(Files.readString(produced.resolve(file)))
      val target = tree.resolve(file.stripSuffix(waiversSuffix) + acceptedSuffix)
      val held =
        if Files.isRegularFile(target) then Accepted.read(Files.readString(target)) else Vector()
      val accepted = Accepted.withWaivers(held, machines, waivers)
      if accepted == held then None
      else if update then
        Files.writeString(target, Accepted.encode(accepted)): Unit
        None
      else
        val (was, now) = (held.toSet, accepted.toSet)
        val missing = accepted.filterNot(was).flatMap(_.subjects)
        val stale = held.filterNot(now).flatMap(_.subjects)
        Some(
          s"${root.relativize(target)} does not carry the waivers its capabilities sections " +
            s"state: missing ${Some(missing).filter(_.nonEmpty).fold("none")(_.mkString(", "))}, " +
            s"stale ${Some(stale).filter(_.nonEmpty).fold("none")(_.mkString(", "))}"
        )
    }
    if differing.nonEmpty then
      throw GateError((differing :+ "rerun with --update (make umpire-gen-model)").mkString("\n"))

  // Runs the gate as its command line says and answers its exit status.
  def main(arguments: Seq[String], tools: => Tools, out: PrintStream, err: PrintStream): Int =
    val known = Set(
      "--update",
      "--skip-go-checks",
      "--generate-ir",
      "--generate-api",
      "--if-stale",
      "--check-syntax",
      "--check-comments",
      "--check-block-form"
    )
    val flags = arguments.toSet
    val generate = flags("--generate-ir") || flags("--generate-api")
    val valid = flags.subsetOf(known) &&
      (if flags("--check-syntax") then flags == Set("--check-syntax")
       else if flags("--check-comments") then flags == Set("--check-comments")
       else if flags("--check-block-form") then flags == Set("--check-block-form")
       else if generate then
         !flags("--update") && !flags("--skip-go-checks") &&
         !(flags("--generate-ir") && flags("--generate-api"))
       else !flags("--if-stale"))
    if !valid then
      err.println(usage)
      2
    else
      try
        if flags("--check-syntax") || flags("--check-comments") || flags("--check-block-form") then
          // A finding is a line of its own, `file:line: reason`, so an editor opens it.
          val findings =
            if flags("--check-syntax") then SyntaxRule.findings(tools.directory)
            else if flags("--check-comments") then CommentRule.findings(tools.directory)
            else BlockFormRule.findings(tools.directory)
          findings.foreach(err.println)
          if findings.isEmpty then 0 else 1
        else
          val gate = Gate(tools, out)
          if flags("--generate-ir") then gate.generateIr(flags("--if-stale"))
          else if flags("--generate-api") then gate.generateApi(flags("--if-stale"))
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

// An accepted lint finding, as umpire-lint reads and writes one (tools/umpire/lint/accept.go).
final case class Acceptance(kind: String, owner: String, subjects: Vector[String], because: String)

// The accepted findings beside an IR file, `<file>.lint.json`, read and written as umpire-lint's
// encoder writes them, and the waivers the lifter writes beside the IR file it lifted a
// `capabilities` section into.
object Accepted:
  // The kind an acceptance of a waiver's reason has, keyed `<machine>.<property>`.
  val waived = "waived-law"

  // `accepted`, holding the waivers `waivers`, `(machine, <machine>.<property>, reason)`, that the
  // `capabilities` sections of `machines` state: an acceptance of one keeps its place with its
  // reason, an acceptance of a waiver of those machines that none states any more is dropped, and
  // a waiver not accepted yet follows the others, in the order given. Every other acceptance keeps
  // its place, an author's and those another forward wrote for other machines alike.
  def withWaivers(
      accepted: Vector[Acceptance],
      machines: Set[String],
      waivers: Seq[(String, String, String)]
  ): Vector[Acceptance] =
    val stated = waivers.map((machine, subject, because) =>
      subject -> Acceptance(waived, machine, Vector(subject), because)
    )
    def ours(a: Acceptance) = a.kind == waived && machines(a.owner)
    val kept = accepted
      .flatMap(a =>
        if !ours(a) then Seq(a) else a.subjects.flatMap(s => stated.find(_._1 == s)).map(_._2)
      )
      .distinct
    val placed = kept.filter(ours).flatMap(_.subjects).toSet
    kept ++ stated.collect { case (subject, a) if !placed(subject) => a }

  // The machines and waivers a lifter's `<file>.waivers.json` holds.
  def waivers(text: String): (Set[String], Seq[(String, String, String)]) =
    val fields = Json.fields(Json.parse(text), "the waivers")
    val machines =
      Json.items(fields.getOrElse("machines", Vector()), "machines").map(Json.text(_, "a machine"))
    val waivers = Json.items(fields.getOrElse("waivers", Vector()), "waivers").map { w =>
      val f = Json.fields(w, "a waiver")
      def text(name: String) = Json.text(f.getOrElse(name, ""), s"a waiver's $name")
      (text("machine"), text("subject"), text("because"))
    }
    (machines.toSet, waivers)

  // The acceptances of a `<file>.lint.json`.
  def read(text: String): Vector[Acceptance] =
    val fields = Json.fields(Json.parse(text), "the accepted findings")
    Json.items(fields.getOrElse("accepted", Vector()), "accepted").map { a =>
      val f = Json.fields(a, "an acceptance")
      def text(name: String) = Json.text(f.getOrElse(name, ""), s"an acceptance's $name")
      Acceptance(
        text("kind"),
        text("owner"),
        Json.items(f.getOrElse("subjects", Vector()), "subjects").map(Json.text(_, "a subject")),
        text("because")
      )
    }

  // The acceptances as Go's encoder writes them, indented by two spaces, with HTML left unescaped.
  def encode(accepted: Vector[Acceptance]): String =
    def text(s: String) = Json.quoted(s)
    def list(items: Vector[String], indent: String) =
      if items.isEmpty then "[]"
      else items.map(i => s"$indent  ${text(i)}").mkString("[\n", ",\n", s"\n$indent]")
    val entries = accepted.map { a =>
      s"""    {
         |      "kind": ${text(a.kind)},
         |      "owner": ${text(a.owner)},
         |      "subjects": ${list(a.subjects, "      ")},
         |      "because": ${text(a.because)}
         |    }""".stripMargin
    }
    val body = if entries.isEmpty then "[]" else entries.mkString("[\n", ",\n", "\n  ]")
    s"{\n  \"accepted\": $body\n}\n"

// The JSON the gate reads with the standard library alone: a value is a Map of its fields, a Vector
// of its items, a String, a BigDecimal, a Boolean or None for null.
private[check] object Json:
  def parse(text: String): Any =
    val (v, end) = value(text, space(text, 0))
    if space(text, end) != text.length then fail(end, "text after the value")
    v

  def fields(v: Any, what: String): Map[String, Any] = v match
    case m: Map[?, ?] => m.collect { case (k: String, x) => k -> x }
    case _            => throw GateError(s"$what is no JSON object")
  def items(v: Any, what: String): Vector[Any] = v match
    case xs: Vector[?] => xs.toVector
    case _             => throw GateError(s"$what is no JSON array")
  def text(v: Any, what: String): String = v match
    case s: String => s
    case _         => throw GateError(s"$what is no JSON string")

  // A string as Go's encoder writes it with HTML left unescaped.
  def quoted(s: String): String =
    s.flatMap {
      case '"'                                            => "\\\""
      case '\\'                                           => "\\\\"
      case '\n'                                           => "\\n"
      case '\r'                                           => "\\r"
      case '\t'                                           => "\\t"
      case '\b'                                           => "\\b"
      case '\f'                                           => "\\f"
      case c if c < ' ' || c == '\u2028' || c == '\u2029' => f"\\u${c.toInt}%04x"
      case c                                              => c.toString
    }.mkString("\"", "", "\"")

  private def fail(at: Int, what: String): Nothing =
    throw GateError(s"not JSON at offset $at: $what")

  @scala.annotation.tailrec
  private def space(text: String, i: Int): Int =
    if i < text.length && " \t\r\n".contains(text(i)) then space(text, i + 1) else i

  private def value(text: String, i: Int): (Any, Int) =
    if i >= text.length then fail(i, "a value is missing")
    text(i) match
      case '{' => members(text, space(text, i + 1), Vector())
      case '[' => elements(text, space(text, i + 1), Vector())
      case '"' => string(text, i + 1, StringBuilder())
      case _   =>
        val end = Iterator.from(i).find(j => j >= text.length || ",]} \t\r\n".contains(text(j))).get
        text.substring(i, end) match
          case "true"  => (true, end)
          case "false" => (false, end)
          case "null"  => (None, end)
          case n => (scala.util.Try(BigDecimal(n)).getOrElse(fail(i, s"not a value: $n")), end)

  @scala.annotation.tailrec
  private def members(text: String, i: Int, done: Vector[(String, Any)]): (Any, Int) =
    if i < text.length && text(i) == '}' && done.isEmpty then (done.toMap, i + 1)
    else
      if i >= text.length || text(i) != '"' then fail(i, "a field name is missing")
      val (key, afterKey) = string(text, i + 1, StringBuilder())
      val colon = space(text, afterKey)
      if colon >= text.length || text(colon) != ':' then fail(colon, "a colon is missing")
      val (v, afterValue) = value(text, space(text, colon + 1))
      val next = space(text, afterValue)
      val all = done :+ (key.toString -> v)
      if next < text.length && text(next) == ',' then members(text, space(text, next + 1), all)
      else if next < text.length && text(next) == '}' then (all.toMap, next + 1)
      else fail(next, "a comma or a brace is missing")

  @scala.annotation.tailrec
  private def elements(text: String, i: Int, done: Vector[Any]): (Any, Int) =
    if i < text.length && text(i) == ']' && done.isEmpty then (done, i + 1)
    else
      val (v, afterValue) = value(text, i)
      val next = space(text, afterValue)
      if next < text.length && text(next) == ',' then
        elements(text, space(text, next + 1), done :+ v)
      else if next < text.length && text(next) == ']' then (done :+ v, next + 1)
      else fail(next, "a comma or a bracket is missing")

  @scala.annotation.tailrec
  private def string(text: String, i: Int, out: StringBuilder): (String, Int) =
    if i >= text.length then fail(i, "a string is not closed")
    text(i) match
      case '"'                         => (out.toString, i + 1)
      case '\\' if i + 1 < text.length =>
        text(i + 1) match
          case 'u' if i + 5 < text.length =>
            out += Integer.parseInt(text.substring(i + 2, i + 6), 16).toChar
            string(text, i + 6, out)
          case c =>
            out += Map('n' -> '\n', 't' -> '\t', 'r' -> '\r', 'b' -> '\b', 'f' -> '\f')
              .getOrElse(c, c)
            string(text, i + 2, out)
      case c =>
        out += c
        string(text, i + 1, out)

@main def run(arguments: String*): Unit =
  sys.exit(Gate.main(arguments, Tools.here, System.out, System.err))
