// The model gate. Scala declares and compiles the Models in temporal/; the IR generator reads their
// typed trees and emits ProtoJSON using ScalaPB. Go alone evaluates the IR, reports Model errors at
// Scala source lines, and checks tables, identities and fingerprints against goldens. It stops on error.
// The model's own vocabulary is held too, by a Go test outside model/ that every run names.
//
//   scala-cli run model/temporal/check
//       lift, require selected generated outputs to be current, lint, test
//   scala-cli run model/temporal/check -- --update
//       lift and rewrite selected generated outputs
//   scala-cli run model/temporal/check -- --skip-go-checks
//       either, without `go vet` and `go test`
//   scala-cli run model/temporal/check -- --generate-ir
//       package the IR's classes and stop; with --if-stale, only when their inputs changed
//   scala-cli run model/temporal/check -- --generate-api
//       package linked API classes; --if-stale reuses a jar whose descriptors and tools are
//       current
//   scala-cli run model/temporal/check -- --check-syntax
//       hold the sugar to the Syntax.scala files and their `Core form:` docs (SyntaxRule.scala);
//       make lint-model runs it
//   scala-cli run model/temporal/check -- --check-comments
//       hold every Scala file of model/ to `//` comments (CommentRule.scala); make lint-model runs
//       it
//   scala-cli run model/temporal/check -- --check-block-form
//       hold every rule of model/temporal to the block form, `on(...) { ... }`
//       (BlockFormRule.scala)
//   scala-cli run model/temporal/check -- --capture-baseline <dir> [--dumper <path>]
//       capture the step tables, IR, Cases and sources into <dir> (Equivalence.scala)
//   scala-cli run model/temporal/check -- --check-equivalence <dir> <map> [--steps-only] [--dumper <path>]
//       compare the tree with that baseline under the rename map <map>; --steps-only compares the
//       step tables alone (Batch 1d)
//
// The IR generator's fixtures under framework/export/testdata are built and lifted too, by its own
// tests: the Models it must lift, compared with the IR in testdata/lifts/expected, which --update
// rewrites as well, and the declarations the build or the IR generator must refuse, at their lines.
package temporal.check

import java.io.PrintStream
import java.nio.file.{Files, Path, StandardCopyOption}
import java.security.MessageDigest
import scala.concurrent.{blocking, Await, Future}
import scala.concurrent.ExecutionContext.Implicits.global
import scala.concurrent.duration.Duration
import scala.jdk.CollectionConverters.*
import scala.util.Try
import scala.util.matching.Regex
import umpire.paths.OutputPaths
import framework.check.*
import framework.check.equivalence.*
import framework.check.lint.*
import temporal.check.lint.ProtoLiterals

// The gate over the repository `tools` runs in; `log` receives its progress.
final class Gate(tools: Tools, log: PrintStream):
  private val root = tools.directory
  private val model = root.resolve("model")
  private val build = model.resolve("build")
  private val schemaFile =
    "proto/internal/temporal/server/api/umpire/v1/ir.proto"
  private val schema = root.resolve(schemaFile)
  // The schema imports the rest of the IR from this directory, under proto/internal.
  private val schemaDirectory = "temporal/server/api/umpire/"
  private val schemaImport: Regex =
    """^\s*import\s+(?:public\s+|weak\s+)?"([^"]+\.proto)"\s*;""".r.unanchored
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
  // Temporal's lifter: the framework's half, whose project.scala is the build's own, and Temporal's.
  private val lifter = Seq("model/framework/export", "model/temporal/export")
  private val dsl = ModelSources.framework
  // The words the model does not use are spelled in this Go test and not here, so that the gate is
  // no use of them.
  private val vocabulary = "TestModelNamesNoRetiredFrontEnd"
  private val selection = ModelSources(root, tools.modelFeatures)
  private val models = selection.sources ++ selection.options

  private def step[A](title: String)(body: => A): A =
    log.println(s"== $title")
    val started = System.nanoTime()
    try body
    finally log.println(s"   ${(System.nanoTime() - started) / 1000000000}s")

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
  private def report(
      steps: Seq[Beside],
      failed: Option[Throwable] = None
  ): Unit =
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
    Files.createTempDirectory(
      Files.createDirectories(build.resolve("history")),
      s"$name."
    )

  // The IR schema's files by their paths under proto/internal, sorted: the root,
  // proto/internal/temporal/server/api/umpire/v1/ir.proto, and every file of the IR it imports,
  // directly or not. The well-known types it imports are ScalaPB's runtime's, not the IR's.
  private def schemaClosure(): Seq[String] =
    if !Files.isRegularFile(schema) then throw GateError(s"the IR schema $schema is missing")
    val internal = root.resolve("proto/internal")
    @annotation.tailrec
    def visit(pending: List[String], seen: Set[String]): Set[String] =
      pending match
        case Nil          => seen
        case file :: rest =>
          val imported = Files
            .readAllLines(internal.resolve(file))
            .asScala
            .flatMap(line => schemaImport.findFirstMatchIn(line).map(_.group(1)))
            .filter(name => name.startsWith(schemaDirectory) && !seen.contains(name))
            .distinct
            .toList
          for name <- imported if !Files.isRegularFile(internal.resolve(name))
          do
            throw GateError(
              s"the IR schema proto/internal/$file imports $name, which is missing"
            )
          visit(imported ++ rest, seen ++ imported)
    val rootName = schemaFile.stripPrefix("proto/internal/")
    visit(List(rootName), Set(rootName)).toSeq.sorted

  // The IR jar's stamp: the hash of the schema's files and the versions of the generator and Scala.
  private def irStamp(closure: Seq[String]): String =
    val digest = MessageDigest.getInstance("SHA-256")
    // Each file's path and length is hashed before its content, so no two closures hash alike.
    for name <- closure do
      val bytes =
        Files.readAllBytes(root.resolve("proto/internal").resolve(name))
      digest.update(
        (name + "\n").getBytes(java.nio.charset.StandardCharsets.UTF_8)
      )
      digest.update(
        java.nio.ByteBuffer.allocate(8).putLong(bytes.length.toLong).array()
      )
      digest.update(bytes)
    val hash = digest.digest().map("%02x".format(_)).mkString
    s"$hash scalapb:$scalapb scala:$scala"

  // Packages the classes the lifter compiles against: model/build/ir-scalapb.jar, the IR's ScalaPB
  // classes, compiled against scalapb-runtime. The IR schema is the file
  // proto/internal/temporal/server/api/umpire/v1/ir.proto and the files of the IR it imports
  // (`schemaClosure`); their Go code is generated with every other internal proto by `make protoc`
  // into api/umpire/v1.
  //
  // The jar has a stamp beside it, the hash of the schema's files and the versions of the generator
  // and of Scala. With `ifStale`, the jar is packaged only when one of those changed since it was
  // packaged.
  def generateIr(ifStale: Boolean): Unit =
    val closure = schemaClosure()
    val stamp = irStamp(closure)
    val stampFile = build.resolve("ir.stamp")
    val current = Seq(irJar, stampFile).forall(Files.isRegularFile(_))
      && Files
        .readString(stampFile)
        .trim == stamp
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
      if !Files.isRegularFile(scalaPlugin) || !Files.isExecutable(scalaPlugin)
      then
        throw GateError(
          s"the ScalaPB plugin $plugin is missing: scala-cli did not package it"
        )

      val classes = Files.createDirectories(sources.resolve("ir"))
      // flat_package leaves the file's name out of the package, so the classes sit in the schema's
      // java_package, io.temporal.server.api.umpire.v1, which the lifter imports.
      tools
        .run(
          "protoc",
          Seq(
            s"--plugin=protoc-gen-scala=$scalaPlugin",
            "--proto_path=proto/internal",
            s"--scala_out=flat_package,scala3_sources:$classes"
          ) ++ closure
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
        throw GateError(
          s"$jar is missing: the IR schema $schemaFile was not packaged"
        )
      Files.writeString(stampFile, stamp + "\n")
      log.println(s"generated $jar")

  // Packages the linked Temporal API and Testpilot classes for Model authoring and lifting.
  def generateApi(ifStale: Boolean): Unit =
    if !Files.isRegularFile(apiDescriptor) then
      throw GateError("proto/api.binpb is missing; run make proto/api.binpb")
    val checked = tools.run("make", Seq("-q", "proto/api.binpb"))
    if checked.exit == 1 then
      throw GateError(
        "proto/api.binpb is stale; run make proto/api.binpb"
      )
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
          .sorted ++ schemaClosure()
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
      throw GateError(
        "current internal descriptors are missing; run make protoc"
      )
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
    then
      throw GateError(
        "linked API descriptor is incomplete; run make proto/api.binpb"
      )

    val digest = MessageDigest.getInstance("SHA-256")
    val inputs = Seq(
      apiDescriptor,
      descriptor,
      root.resolve("cmd/tools/getproto/main.go"),
      root.resolve("cmd/tools/getproto/files.go"),
      root.resolve("go.mod"),
      root.resolve("go.sum"),
      scalaPlugin,
      root.resolve("model/temporal/check/Gate.scala"),
      root.resolve("model/temporal/check/project.scala")
    )
    for source <- inputs do
      val bytes = Files.readAllBytes(source)
      digest.update(
        java.nio.ByteBuffer.allocate(8).putLong(bytes.length.toLong).array()
      )
      digest.update(bytes)
    val protocVersion =
      tools.run("protoc", Seq("--version")).orFail().output.trim
    val cliVersion = tools.run("scala-cli", Seq("version")).orFail().output.trim
    digest.update(
      s"scalapb:$scalapb scala:$scala protoc:$protocVersion scala-cli:$cliVersion options:$apiOptions runtime-grpc:$scalapb"
        .getBytes(java.nio.charset.StandardCharsets.UTF_8)
    )
    val stamp = digest.digest().map("%02x".format(_)).mkString +
      s" options:$apiOptions scalapb:$scalapb scala:$scala"
    val stampFile = build.resolve("api.stamp")
    val current = Seq(apiJar, stampFile).forall(Files.isRegularFile(_)) &&
      Files
        .readString(stampFile)
        .trim == stamp
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
      // The IR's classes are ir-scalapb.jar's; the API jar holds none of them as well.
      val irClasses = classes.resolve("io/temporal/server/api/umpire")
      if Files.exists(irClasses) then
        throw GateError(
          "linked API classes include the IR schema's io.temporal.server.api.umpire classes, which " +
            "model/build/ir-scalapb.jar holds; cmd/tools/getproto must leave the IR's files out"
        )
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
      if !ran.output.linesIterator.exists(
          _.startsWith(s"--- PASS: $vocabulary ")
        )
      then
        throw GateError(
          s"the vocabulary check $vocabulary did not run:\n${ran.diagnostics}"
        )

    step("reject free-text protobuf names in Models"):
      ProtoLiterals.check(model.resolve("temporal"), root, selection.includes)

    // A check lowers the checked-in IR, which it never writes, so its Cases are checked beside the
    // build and the lifts. An update lowers the IR it has just written.
    val cases =
      if update then Nil
      else
        Seq(
          beside("check every lowered Case and the Query manifest")(
            lower(update)
          )
        )
    report(cases, Try(build(update)).failed.toOption)
    if update then
      step("generate every lowered Case and the Query manifest")(
        lower(update)
      ): Unit

    // Not one of the Go checks a run may skip: it is the gate's check of the IR it settled.
    // It fails on a finding no acceptance matches and on a stale acceptance, never on a count. The
    // gate has already held the accepted findings to the capabilities sections' waivers above.
    step("lint every IR file and print its coverage"):
      tools
        .run(
          "go",
          Seq("run", "./tools/umpire/cmd/umpire-lint") ++ selectedIr.map(_.toString),
          Output.Shown
        )
        .orFail(): Unit

    // Combined verification may run the complete Go suite separately; the default gate includes it.
    if goChecks && !selection.narrowed then
      step("interpret the IR in Go and hold it to its goldens"):
        tools
          .run(
            "go",
            Seq("vet", "-tags", "test_dep", "./tools/umpire/..."),
            Output.Shown
          )
          .orFail()
        tools
          .run(
            "go",
            Seq("test", "-count=1", "-tags", "test_dep", "./tools/umpire/..."),
            Output.Shown
          )
          .orFail()
    log.println("== ok")

  // The step tables of the Models as they are now, written into `dir` (model/framework/dsl/StepTables.scala).
  private def writeStepTables(dir: Path): Unit =
    val ran = tools
      .scalaCli(
        Seq("run", "--server=false") ++ models ++ Seq(
          "--main-class",
          "framework.writeStepTables",
          "--",
          "temporal",
          dir.toString
        )
      )
      .orFail()
    ran.output.linesIterator
      .filter(_.startsWith("wrote "))
      .foreach(l => log.println(s"   $l"))

  private val tablePins = "model/testdata/table-pins/replay.py"

  private def equivalenceOwned(name: String): Boolean =
    !selection.narrowed || selection.features.exists(feature =>
      OutputPaths.generated(root, logicalFeature(feature)).equals(OutputPaths.generated(root, name))
    )

  private def featureKeyPrefixes(features: Seq[String]): Seq[String] = features.flatMap(feature =>
    Seq(feature.replace('/', '.'), feature.replace('/', '.').replace('_', '.')).distinct
      .map("temporal.features." + _ + ".")
  )

  private def unselectedTable(key: String): Boolean =
    featureKeyPrefixes(selection.discovered.filterNot(selection.features.contains))
      .exists(key.startsWith(_))

  private[check] def equivalenceOutputs(out: Path): Unit =
    val ir = Files.createDirectories(out.resolve("ir"))
    val cases = Files.createDirectories(out.resolve("cases"))
    def leaf(name: String): String =
      if !Path.of(name).getFileName.toString.equals(name) || name.equals(".") || name.equals("..")
      then throw GateError(s"$name is no artifact file name")
      name
    def copy(from: Path, to: Path): Unit =
      if Files.exists(to) then
        if !Files.readAllBytes(from).sameElements(Files.readAllBytes(to)) then
          throw GateError(s"conflicting logical artifact $to from $from")
      else Files.copy(from, to, StandardCopyOption.COPY_ATTRIBUTES): Unit
    val inventory = OutputPaths.inventory(root).filter((name, _) => equivalenceOwned(name))
    for (name, file) <- inventory do
      copy(file, ir.resolve(leaf(name + ".json")))
      val lint = OutputPaths.lint(root, name)
      if Files.isRegularFile(lint) then copy(lint, ir.resolve(leaf(name + ".lint.json")))
    val manifests = inventory
      .map(_._2.getParent.resolve("manifest.json"))
      .distinct
      .filter(Files.isRegularFile(_))
      .sortBy(_.toString)
    val metadata = manifests.map(file =>
      Json.fields(Json.parse(Files.readString(file)), file.toString) - "queries"
    )
    if metadata.map(Equivalence.canonical).distinct.sizeIs > 1 then
      throw GateError("relocated manifests have conflicting metadata")
    val queries = manifests.flatMap: manifest =>
      val fields = Json.fields(Json.parse(Files.readString(manifest)), manifest.toString)
      Json
        .items(fields.getOrElse("queries", Vector.empty), s"$manifest queries")
        .map: value =>
          val row = Json.fields(value, s"$manifest query")
          row
            .get("file")
            .fold(row): value =>
              val file = leaf(Json.text(value, "Case file"))
              val logical =
                if file.endsWith(".case.json") then
                  val name = Json.text(row("model"), "Case Model").stripSuffix(".json")
                  val query = Json.fields(row("query"), "Case Query")
                  leaf(name + "-" + Json.text(query("Name"), "Query name") + "-case.json")
                else file
              copy(manifest.getParent.resolve(file), cases.resolve(logical))
              row.updated("file", logical)
    for directory <- inventory.map(_._2.getParent).distinct do
      val stream = Files.list(directory)
      try
        for file <- stream.iterator.asScala if Files.isRegularFile(file) do
          val name = file.getFileName.toString
          if name.endsWith("-case.json") then copy(file, cases.resolve(name))
          else if name.endsWith(".case.json") then
            val model = inventory
              .find((_, ir) =>
                ir.getParent.equals(directory) && ir.getFileName.toString.equals("ir.json")
              )
              .getOrElse(throw GateError(s"$file has no owning primary IR"))
              ._1
            copy(
              file,
              cases.resolve(leaf(model + "-" + name.stripSuffix(".case.json") + "-case.json"))
            )
      finally stream.close()
    if manifests.nonEmpty then
      Files.writeString(
        cases.resolve("manifest.json"),
        Equivalence.canonical(metadata.head.updated("queries", queries)) + "\n"
      ): Unit

  private[check] def equivalenceBaseline(
      base: Path,
      out: Path,
      map: RenameMap
  ): Unit =
    val scopeMap = RenameMap(map.renames, map.removals)
    val prefixes = featureKeyPrefixes(selection.features)
    val entries = Equivalence
      .index(base.resolve(Equivalence.stepTables))
      .filter((key, _) =>
        val renamed = scopeMap(key)
        !renamed.startsWith("temporal.features.") || prefixes.exists(renamed.startsWith(_))
      )
    val tables = Files.createDirectories(out.resolve(Equivalence.stepTables))
    for (_, (digest, _)) <- entries.distinctBy(_._2._1) do
      Files.copy(
        base.resolve(Equivalence.stepTables).resolve(digest + ".steps.gz"),
        tables.resolve(digest + ".steps.gz"),
        StandardCopyOption.COPY_ATTRIBUTES
      ): Unit
    Files.writeString(
      tables.resolve("index.tsv"),
      entries.map((key, info) => s"$key\t${info._1}\t${info._2}\n").mkString
    ): Unit
    val ir = Files.createDirectories(out.resolve("ir"))
    val cases = Files.createDirectories(out.resolve("cases"))
    val selectedNames = _root_.scala.collection.mutable.Set.empty[String]
    if Files.isDirectory(base.resolve("ir")) then
      val stream = Files.list(base.resolve("ir"))
      try
        for file <- stream.iterator.asScala if Files.isRegularFile(file) do
          val filename = file.getFileName.toString
          val name = filename.stripSuffix(".lint.json").stripSuffix(".json")
          if filename.endsWith(".json") && equivalenceOwned(scopeMap(name)) then
            selectedNames += name: Unit
            Files.copy(file, ir.resolve(filename), StandardCopyOption.COPY_ATTRIBUTES): Unit
      finally stream.close()

    if Files.isDirectory(base.resolve("cases")) then
      val stream = Files.list(base.resolve("cases"))
      try
        for file <- stream.iterator.asScala if Files.isRegularFile(file) do
          val name = file.getFileName.toString
          if name.endsWith("-case.json") && selectedNames
              .exists(stem => name.startsWith(stem + "-"))
          then Files.copy(file, cases.resolve(name), StandardCopyOption.COPY_ATTRIBUTES): Unit
      finally stream.close()
      val manifest = base.resolve("cases/manifest.json")
      if Files.isRegularFile(manifest) then
        val fields = Json.fields(Json.parse(Files.readString(manifest)), manifest.toString)
        val queries = Json
          .items(fields.getOrElse("queries", Vector.empty), "baseline queries")
          .filter(row =>
            val model = Json.text(Json.fields(row, "baseline query")("model"), "baseline Model")
            equivalenceOwned(scopeMap(model.stripSuffix(".json")))
          )
        Files.writeString(
          cases.resolve("manifest.json"),
          Equivalence.canonical(fields.updated("queries", queries)) + "\n"
        ): Unit
    if Files.isDirectory(base.resolve("native")) then
      val stream = Files.list(base.resolve("native"))
      try
        for directory <- stream.iterator.asScala if Files.isDirectory(directory) do
          val name = directory.getFileName.toString
          if equivalenceOwned(scopeMap(name)) then
            copyTree(directory, out.resolve("native").resolve(name))
      finally stream.close()

  private[check] def equivalencePins(out: Path): Path =
    val original = root.resolve("model/testdata/table-pins/pins.json")
    if !selection.narrowed then original
    else
      val fields = Json.fields(Json.parse(Files.readString(original)), original.toString)
      val artifacts = Json
        .items(fields("artifacts"), "pin artifacts")
        .filter(pin =>
          val artifact = Json.text(Json.fields(pin, "pin artifact")("artifact"), "artifact")
          equivalenceOwned(artifact.stripSuffix(".json"))
        )
      val selected = out.resolve("native-pins.json")
      Files.writeString(
        selected,
        Equivalence.canonical(fields.updated("artifacts", artifacts)) + "\n"
      ): Unit
      selected

  // Captures a baseline of the tree into `out`, which must be absent or empty (Equivalence.scala).
  // With `dumper`, the interpreter reads the legacy logical view of relocated IR too, after
  // the dumper and the IR are held to the table pins (model/testdata/table-pins).
  def captureBaseline(out: Path, dumper: Option[Path]): Unit =
    if Files.exists(out) && Files.list(out).findAny().isPresent then
      throw GateError(
        s"$out is not empty: capture a baseline into a new directory"
      )
    Files.createDirectories(out): Unit
    val started = System.currentTimeMillis()
    val commit =
      Try(tools.run("git", Seq("rev-parse", "HEAD")).orFail().output.trim)
        .getOrElse("unknown")
    val dirty = Try(
      tools
        .run("git", Seq("status", "--porcelain", "--", "model"))
        .orFail()
        .output
        .stripTrailing
    )
      .getOrElse("unknown")
    step("write the step tables"):
      writeStepTables(out.resolve(Equivalence.stepTables))
    step("copy generated IR, Cases and the Models' sources"):
      equivalenceOutputs(out)
      for source <- dsl.filter(_.startsWith("model/framework")) do
        copyTree(root.resolve(source), out.resolve("sources").resolve(source.stripPrefix("model/")))
      for source <- selection.sources if source.startsWith("model/temporal") do
        val from = root.resolve(source)
        val to = out.resolve("sources").resolve(model.relativize(from))
        if Files.isDirectory(from) then copyTree(from, to)
        else
          Files.createDirectories(to.getParent): Unit
          Files.copy(from, to, StandardCopyOption.COPY_ATTRIBUTES): Unit
    dumper.foreach: d =>
      step("replay the interpreter tables of generated IR against the table pins"):
        val pins = equivalencePins(out)
        tools
          .run(
            "python3",
            Seq(
              tablePins,
              "--pins",
              pins.toString,
              "--ir",
              out.resolve("ir").toString,
              "--dumper",
              d.toString,
              "--out",
              out.resolve("native").toString,
              "--receipt",
              out.resolve("native-receipt.json").toString
            ),
            Output.Shown
          )
          .orFail(): Unit
    val tables = Equivalence.index(out.resolve(Equivalence.stepTables))
    Files.writeString(
      out.resolve("baseline.txt"),
      Seq(
        s"commit\t$commit",
        s"model\t${if dirty.isEmpty then "clean" else "changed:\n" + dirty}",
        s"captured\t${java.time.Instant.now()}",
        s"step tables\t${tables.size} (${tables.map(_._2._1).distinct.size} distinct, ${tables.map(_._2._2).sum} lines)",
        s"native streams\t${if dumper.isDefined then "native/" else "not captured"}",
        s"seconds\t${(System.currentTimeMillis() - started) / 1000}"
      ).mkString("", "\n", "\n")
    ): Unit
    if dirty.nonEmpty then log.println(s"   model/ has changes; baseline.txt records them:\n$dirty")
    log.println(s"== captured $commit into $out")

  // Compares the tree with the baseline in `base` under `map` (Equivalence.scala), and fails on
  // every difference the map does not explain and every entry of the map that explains nothing.
  def checkEquivalence(
      base: Path,
      mapFile: Path,
      stepsOnly: Boolean,
      dumper: Option[Path]
  ): Unit =
    val map = RenameMap.read(mapFile)
    val ledger = StepLedger
      .forComparison(mapFile, stepsOnly)
      .map(value =>
        value.copy(
          orders = value.orders.filterNot(order => unselectedTable(order.currentKey)),
          additions = value.additions.filterNot(addition => unselectedTable(addition.currentKey))
        )
      )
    if !Files.isDirectory(base.resolve(Equivalence.stepTables)) then
      throw GateError(
        s"$base is no baseline: it has no ${Equivalence.stepTables}"
      )
    val now = scratch("steps")
    if !stepsOnly then equivalenceOutputs(now)
    step("write the step tables"):
      writeStepTables(now.resolve(Equivalence.stepTables))
    val comparisonBase =
      if !selection.narrowed then base
      else
        val scoped = now.resolve("baseline")
        equivalenceBaseline(base, scoped, map)
        scoped
    val tables = step("compare the step tables with the baseline's"):
      Equivalence.compareTables(
        comparisonBase.resolve(Equivalence.stepTables),
        now.resolve(Equivalence.stepTables),
        map,
        ledger
      )
    val projected =
      if stepsOnly then Vector.empty
      else
        step(
          "compare the IR, its lint findings, the Cases and the manifest with the baseline's"
        ):
          Equivalence.compareProjection(
            comparisonBase.resolve("ir"),
            comparisonBase.resolve("cases"),
            now.resolve("ir"),
            now.resolve("cases"),
            map
          )
    val (native, nativeUsed, nativeRemovals) = dumper match
      case Some(d) if !stepsOnly =>
        if !Files.isDirectory(comparisonBase.resolve("native")) then
          throw GateError(
            s"$base holds no interpreter streams: capture it with --dumper"
          )
        step("compare the interpreter tables of generated IR with the baseline's"):
          val receipt = now.resolve("native-receipt.json")
          val ran = tools.run(
            "python3",
            Seq(
              tablePins,
              "--ir",
              now.resolve("ir").toString,
              "--dumper",
              d.toString,
              "--out",
              now.resolve("native").toString,
              "--receipt",
              receipt.toString,
              "--baseline",
              comparisonBase.resolve("native").toString,
              "--map",
              mapFile.toString
            )
          )
          if !Files.isRegularFile(receipt) then
            throw GateError(
              s"the interpreter tables were not compared:\n${ran.diagnostics}"
            )
          val fields =
            Json.fields(Json.parse(Files.readString(receipt)), receipt.toString)
          def texts(name: String) =
            Json
              .items(fields.getOrElse(name, Vector()), name)
              .map(Json.text(_, name))
          (
            texts("differences").map("interpreter " + _),
            texts("used").toSet,
            texts("used_removals").toSet
          )
      case _ => (Vector.empty, Set.empty[String], Set.empty[String])
    map.used ++= nativeUsed
    val (unused, deferred) =
      Equivalence.unused(
        map,
        tables.removalsUsed ++ nativeRemovals,
        !stepsOnly,
        comparisonBase
      )
    deferred.foreach(d => log.println(s"   $d"))
    val all = tables.differences ++ projected ++ native ++ unused
    if all.nonEmpty then
      throw GateError(
        (all :+ s"${all.size} differences from the baseline $base; scratch tables in $now")
          .mkString("\n")
      )
    log.println(s"== equivalent to the baseline $base")

  private def copyTree(from: Path, to: Path): Unit =
    if Files.isDirectory(from) then
      val stream = Files.walk(from)
      try
        stream.iterator.asScala.foreach: p =>
          val target = to.resolve(from.relativize(p).toString)
          if Files.isDirectory(p) then Files.createDirectories(target): Unit
          else Files.copy(p, target, StandardCopyOption.COPY_ATTRIBUTES): Unit
      finally stream.close()

  // An update's output is shown as it runs; a check's is kept, since it runs beside other steps.
  private def lower(update: Boolean): String =
    selection.features
      .map: feature =>
        val directory = OutputPaths.generated(root, logicalFeature(feature))
        tools
          .inDirectory(directory)
          .run(
            "go",
            Seq("run", root.resolve("tools/umpire/cmd/umpire-gen-cases").toString) ++
              (if update then Seq("--update") else Seq.empty),
            if update then Output.Shown else Output.Kept
          )
          .orFail()
          .output
      .mkString("\n")

  private def logicalFeature(feature: String): String = feature.replace('/', '-').replace('_', '-')

  private def selectedIr: Seq[Path] =
    selection.features.map(f => OutputPaths.ir(root, logicalFeature(f)))

  def modelLoop(skipLint: Boolean = false): Unit =
    val lifted = build(update = false, fast = true)
    val fresh =
      selection.features.map(f => lifted.resolve(logicalFeature(f) + ".json"))
    // The Go checks read the IR through the schema's Go code; while that is stale (its generation,
    // `make protoc`, is owner-held) they are skipped and named, never reported as passing.
    val held = goIrHeld()
    val skipped = Vector.newBuilder[String]
    def goStep(title: String)(body: => Unit): Unit = held match
      case Some(reason) =>
        log.println(s"== skipped $title (held: $reason)")
        skipped += title
      case None => step(title)(body)
    goStep("check selected lifted IR in Go"):
      val checked = scratch("go-check")
      for feature <- selection.features do
        val directory = Files.createDirectories(
          checked
            .resolve("model/temporal/features")
            .resolve(feature)
            .resolve("generated")
        )
        Files.copy(
          lifted.resolve(logicalFeature(feature) + ".json"),
          directory.resolve("ir.json")
        )
        tools
          .inDirectory(directory)
          .run(
            "go",
            Seq(
              "run",
              root.resolve("tools/umpire/cmd/umpire-gen-cases").toString,
              "--update"
            ),
            Output.Shown
          )
          .orFail(): Unit
    if skipLint then
      log.println("== skipped selected IR lint (--skip-lint)")
      log.println("== skipped selected Scala syntax/style (--skip-lint)")
    else
      goStep("lint selected lifted IR"):
        for feature <- selection.features do
          val accepted = OutputPaths.lint(root, logicalFeature(feature))
          if Files.isRegularFile(accepted) then
            Files.copy(
              accepted,
              lifted.resolve(logicalFeature(feature) + ".lint.json")
            ): Unit
        tools
          .run(
            "go",
            Seq("run", "./tools/umpire/cmd/umpire-lint") ++ fresh.map(
              _.toString
            ),
            Output.Shown
          )
          .orFail(): Unit
      step("check selected Scala syntax"):
        val findings = SyntaxRule.findings(root, selection.includes) ++
          CommentRule.findings(root, selection.includes) ++ BlockFormRule
            .findings(
              root,
              selection.includes
            )
        if findings.nonEmpty then throw GateError(findings.mkString("\n"))
    step("compare selected IR with checked-in outputs"):
      tools
        .run(
          "git",
          Seq("diff", "--stat", "--") ++
            selection.features
              .map(f =>
                root
                  .relativize(OutputPaths.generated(root, logicalFeature(f)))
                  .toString
              ),
          Output.Shown
        )
        .orFail(): Unit
      Gate.settleFeatures(root, lifted, selection.features, update = false)
    goStep("check selected Cases in Go"):
      lower(update = false): Unit
    val notRun = skipped.result()
    if notRun.nonEmpty then
      log.println(
        s"== Scala stages passed; NOT run (held Go checks): ${notRun.mkString("; ")}"
      )

  // Generates and compiles what the lifter reads, then builds its fixtures and lifts the Models.
  private def build(update: Boolean, fast: Boolean = false): Path =
    val lifted = scratch("ir")
    def report(completed: Boolean): Unit =
      selection
        .states(lifted)
        .foreach: (feature, state) =>
          val reported = if !completed && state == "current" then "behind" else state
          log.println(s"feature $feature: $reported")
    try buildSelected(update, fast, lifted)
    catch
      case failed: Throwable =>
        report(completed = false)
        throw failed
    try
      if !fast then Gate.settleFeatures(root, lifted, selection.features, update)
      lifted
    finally report(completed = true)

  // Why the IR schema's Go code cannot be trusted, if it cannot: each file of the schema has its Go
  // code beside the others', as `make protoc` writes it, newer than the file.
  private def goIrHeld(): Option[String] =
    schemaClosure().iterator
      .flatMap: name =>
        val file = s"proto/internal/$name"
        val generated =
          root
            .resolve("api")
            .resolve(
              name
                .stripPrefix("temporal/server/api/")
                .stripSuffix(".proto") + ".pb.go"
            )
        if !Files.isRegularFile(generated) then
          Some(s"${root.relativize(generated)} is missing for the IR schema $file; run make protoc")
        else if Files
            .getLastModifiedTime(generated)
            .compareTo(Files.getLastModifiedTime(root.resolve(file))) <= 0
        then
          Some(s"${root.relativize(generated)} is older than the IR schema $file; run make protoc")
        else None
      .nextOption()

  // The linked API's generation runs protoc and Go over the API descriptors, which is owner-held,
  // so the loop never runs it: it reads the jars `make gen-model-api` packaged, and refuses a
  // missing jar or an IR jar older than the IR schema, naming that target.
  private def useCachedClasses(): Unit =
    for jar <- Seq(irJar, apiJar) if !Files.isRegularFile(jar) do
      throw GateError(
        s"${root.relativize(jar)} is missing; run make gen-model-api once (the held linked-API generation)"
      )
    val stampFile = build.resolve("ir.stamp")
    if !Files.isRegularFile(stampFile) ||
      Files.readString(stampFile).trim != irStamp(schemaClosure())
    then
      throw GateError(
        s"${root.relativize(irJar)} is older than the IR schema; run make gen-model-api"
      )
    log.println(
      s"   cached ${root.relativize(irJar)} and ${root.relativize(apiJar)}, not regenerated"
    )

  private def buildSelected(update: Boolean, fast: Boolean, lifted: Path): Unit =
    // The loop reads the cached jars; only the canonical gate regenerates them.
    if fast then
      step("use the cached IR and API classes"):
        useCachedClasses()
    else
      step("generate the IR's classes when their inputs changed"):
        generateIr(ifStale = true)
        goIrHeld().foreach(reason => throw GateError(reason))
      step("generate the linked API's classes when their inputs changed"):
        generateApi(ifStale = true)

    // The framework must build without the Temporal Models, so nothing in framework/ reaches into them.
    if !fast then
      step("compile the framework alone"):
        tools.scalaCli("compile" +: dsl).orFail()

    step("compile each selected feature independently"):
      val failures = selection.features.flatMap: feature =>
        val sources = ModelSources(root, Some(feature))
        val ran =
          tools.scalaCli(Seq("compile", "--server=false") ++ sources.sources ++ sources.options)
        if ran.failed then
          log.println(s"feature $feature: behind (compile)")
          Some(s"feature $feature is behind: ${ran.diagnostics}")
        else None
      if failures.nonEmpty then throw GateError(failures.mkString("\n"))

    if !fast then
      step("compile and test the framework and the Temporal Models"):
        tools.scalaCli(Seq("test", "--server=false") ++ models).orFail()

    step("package the Models' TASTy"):
      tools
        .scalaCli(
          Seq(
            "--power",
            "package",
            "--server=false",
            "--library"
          ) ++ models ++ Seq(
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
      // make's record of the selection it packaged is no longer the jar's, so its next package runs.
      Files.deleteIfExists(build.resolve("model-sources")): Unit

    // Compiled once here, so the steps below that run it side by side only read its classes.
    if !fast && !selection.narrowed then
      step("compile the lifter and its tests"):
        tools.scalaCli(Seq("compile", "--test") ++ lifter).orFail()
    if !fast && selection.narrowed then
      log.println("== skipped complete lifter fixture suite (narrowed FEATURES; run the full-set gate)")

    // Both read the packaged Models and the lifter, and neither reads what the other writes.
    val fixtures =
      if fast || selection.narrowed then Seq.empty
      else
        Seq(
          beside(
            "build and lift the lifter's fixtures: the expected IR, and each refusal at its line"
          ):
            tools
              .withEnvironment("UMPIRE_LIFTER_UPDATE" -> (if update then "1" else ""))
              .scalaCli("test" +: lifter)
              .orFail()
            ""
        )
    val modelLifts = beside("lift each selected feature's IR"):
      // One lifter run reads the Models' TASTy once and writes one IR file per feature, collected
      // from its package, each lifted apart; its refusals name the file they were lifting.
      val arguments = ("run" +: lifter) ++ Seq(
        "--main-class",
        "framework.export.lift",
        "--",
        "--ir",
        s"$modelJar=model/",
        modelClasspath.toString,
        lifted.toString
      ) ++ selection.features.map(_.replace('/', '-').replace('_', '-'))
      val ran = tools.scalaCli(arguments)
      if ran.failed then
        throw GateError(
          s"the Models' IR files did not lift:\n${ran.diagnostics}"
        )
      ""
    report(fixtures :+ modelLifts)

object Gate:
  def settleFeatures(root: Path, produced: Path, features: Seq[String], update: Boolean): Unit =
    val selected = features.map(_.replace('/', '-').replace('_', '-'))
    val expected = selected.map(_ + ".json").toSet
    val actual = files(produced).filterNot(file =>
      file.endsWith(waiversSuffix) || file.endsWith(acceptedSuffix)
    )
    val extra = actual -- expected
    if extra.nonEmpty then
      throw GateError(s"unselected IR outputs: ${extra.toSeq.sorted.mkString(", ")}")
    val errors = selected.flatMap: feature =>
      val target = OutputPaths.ir(root, feature)
      val source = produced.resolve(feature + ".json")
      if !Files.isRegularFile(source) then Some(s"feature $feature: missing lifted IR $source")
      else if !Files.isRegularFile(target) && !update then
        Some(s"feature $feature: missing $target")
      else if !same(target, source) && !update then
        Some(
          s"feature $feature: behind (${difference(Files.readString(target), Files.readString(source))})"
        )
      else None
    if errors.nonEmpty then
      throw GateError(
        (errors :+ s"rerun with --update; lifted files are in $produced").mkString("\n")
      )
    for feature <- selected do
      val target = OutputPaths.ir(root, feature)
      val source = produced.resolve(feature + ".json")
      if update && !same(target, source) then
        Files.createDirectories(target.getParent)
        Files.write(target, Files.readAllBytes(source)): Unit
      val waiver = produced.resolve(feature + waiversSuffix)
      if Files.isRegularFile(waiver) then
        val (machines, waivers) = Accepted.waivers(Files.readString(waiver))
        val acceptedFile = OutputPaths.lint(root, feature)
        val held = if Files.isRegularFile(acceptedFile) then
          Accepted.read(Files.readString(acceptedFile))
        else Vector()
        val accepted = Accepted.withWaivers(held, machines, waivers)
        if accepted != held then
          if update then Files.writeString(acceptedFile, Accepted.encode(accepted)): Unit
          else
            throw GateError(
              s"feature $feature: behind ($acceptedFile does not carry lifted capability waivers)"
            )

  val usage =
    "usage: gate [--update] [--skip-go-checks] | gate --generate-ir|--generate-api [--if-stale]" +
      " | gate --check-syntax | gate --check-comments | gate --check-block-form" +
      " | gate --capture-baseline <dir> [--dumper <path>]" +
      " | gate --check-equivalence <dir> <map> [--steps-only] [--dumper <path>]"

  // The equivalence commands' arguments: the command, its paths, and the flags after them.
  private def equivalence(
      arguments: Seq[String]
  ): Option[(String, Seq[Path], Boolean, Option[Path])] =
    def flags(rest: Seq[String]): Option[(Boolean, Option[Path])] = rest match
      case Seq()                  => Some((false, None))
      case "--steps-only" +: more =>
        flags(more).filter(!_._1).map((_, d) => (true, d))
      case "--dumper" +: path +: more if !path.startsWith("--") =>
        flags(more).filter(_._2.isEmpty).map((s, _) => (s, Some(Path.of(path))))
      case _ => None
    arguments match
      case "--capture-baseline" +: out +: rest if !out.startsWith("--") =>
        flags(rest)
          .filter(!_._1)
          .map((s, d) => ("capture", Seq(Path.of(out)), s, d))
      case "--check-equivalence" +: base +: map +: rest
          if !base.startsWith("--") && !map.startsWith("--") =>
        flags(rest).map((s, d) => ("check", Seq(Path.of(base), Path.of(map)), s, d))
      case _ => None

  private def same(checkedIn: Path, produced: Path) =
    Files.isRegularFile(checkedIn) && Files.mismatch(
      checkedIn,
      produced
    ) == -1L

  private def files(directory: Path): Set[String] =
    val stream = Files.list(directory)
    try
      stream.iterator.asScala
        .filter(Files.isRegularFile(_))
        .map(_.getFileName.toString)
        .toSet
    finally stream.close()

  // Where two texts first differ, which is enough to see what moved; both files are kept whole.
  private def difference(checkedIn: String, lifted: String): String =
    val (was, now) =
      (checkedIn.linesIterator.toVector, lifted.linesIterator.toVector)
    val at = was
      .map(Some(_))
      .zipAll(now.map(Some(_)), None, None)
      .indexWhere(
        _ != _
      )
    def line(text: Vector[String]) =
      text.lift(at).fold("nothing")(l => s"`${l.take(200)}`")
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
  def settle(
      tree: Path,
      produced: Path,
      root: Path,
      update: Boolean,
      selected: Boolean = false
  ): Unit =
    def name(file: Path) = root.relativize(file)
    val (held, lifted) = (
      files(tree).filterNot(_.endsWith(acceptedSuffix)),
      files(produced).filterNot(_.endsWith(waiversSuffix))
    )
    val orphans = (if selected then Set.empty[String] else held -- lifted).toSeq.sorted.map(file =>
      s"${name(tree.resolve(file))} is checked in and nothing produces it: remove it, or declare its feature's Queries"
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
          val differs =
            difference(Files.readString(target), Files.readString(source))
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
  def acceptWaivers(
      tree: Path,
      produced: Path,
      root: Path,
      update: Boolean
  ): Unit =
    val differing =
      files(produced).filter(_.endsWith(waiversSuffix)).toSeq.sorted.flatMap { file =>
        val (machines, waivers) =
          Accepted.waivers(Files.readString(produced.resolve(file)))
        val target =
          tree.resolve(file.stripSuffix(waiversSuffix) + acceptedSuffix)
        val held =
          if Files.isRegularFile(target) then Accepted.read(Files.readString(target))
          else Vector()
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
      throw GateError(
        (differing :+ "rerun with --update (make umpire-gen-model)").mkString(
          "\n"
        )
      )

  // Runs the gate as its command line says and answers its exit status.
  def main(
      arguments: Seq[String],
      tools: => Tools,
      out: PrintStream,
      err: PrintStream
  ): Int =
    if arguments.headOption.exists(a =>
        a.equals("--capture-baseline") || a.equals("--check-equivalence")
      )
    then
      equivalence(arguments) match
        case None =>
          err.println(usage)
          2
        case Some((command, paths, stepsOnly, dumper)) =>
          try
            val gate = Gate(tools, out)
            val absolute = paths.map(p => tools.directory.resolve(p))
            if command.equals("capture") then gate.captureBaseline(absolute.head, dumper)
            else gate.checkEquivalence(absolute(0), absolute(1), stepsOnly, dumper)
            0
          catch
            case e: (GateError | ToolError) =>
              err.println(s"gate: ${e.getMessage}")
              1
            case e: java.io.IOException =>
              err.println(s"gate: $e")
              1
    else run(arguments, tools, out, err)

  private def run(
      arguments: Seq[String],
      tools: => Tools,
      out: PrintStream,
      err: PrintStream
  ): Int =
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
      (if flags("--check-syntax") then
         flags == Set(
           "--check-syntax"
         )
       else if flags("--check-comments") then
         flags == Set(
           "--check-comments"
         )
       else if flags("--check-block-form") then
         flags == Set(
           "--check-block-form"
         )
       else if generate then
         !flags("--update") && !flags("--skip-go-checks") &&
         !(flags("--generate-ir") && flags("--generate-api"))
       else !flags("--if-stale"))
    if !valid then
      err.println(usage)
      2
    else
      try
        if flags("--check-syntax") || flags("--check-comments") || flags(
            "--check-block-form"
          )
        then
          // A finding is a line of its own, `file:line: reason`, so an editor opens it.
          val selection = ModelSources(tools.directory, tools.modelFeatures)
          val findings =
            if flags("--check-syntax") then SyntaxRule.findings(tools.directory, selection.includes)
            else if flags("--check-comments") then
              CommentRule.findings(tools.directory, selection.includes)
            else BlockFormRule.findings(tools.directory, selection.includes)
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
final case class Acceptance(
    kind: String,
    owner: String,
    subjects: Vector[String],
    because: String
) derives CanEqual

// The accepted findings beside an IR file, `<file>.lint.json`, read and written as umpire-lint's
// encoder writes them, and the waivers the lifter writes beside the IR file it lifted a
// `capabilities` section into.
object Accepted:
  // The kind an acceptance of a waiver's reason has, keyed `<machine>.<property>`.
  val waived = "capability-waiver"

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
    def ours(a: Acceptance) = a.kind == waived && machines(
      a.owner
    )
    val kept = accepted
      .flatMap(a =>
        if !ours(a) then Seq(a)
        else
          a.subjects
            .flatMap(s => stated.find(_._1 == s))
            .map(_._2)
      )
      .distinct
    val placed = kept.filter(ours).flatMap(_.subjects).toSet
    kept ++ stated.collect { case (subject, a) if !placed(subject) => a }

  // The machines and waivers a lifter's `<file>.waivers.json` holds.
  def waivers(text: String): (Set[String], Seq[(String, String, String)]) =
    val fields = Json.fields(Json.parse(text), "the waivers")
    val machines =
      Json
        .items(fields.getOrElse("machines", Vector()), "machines")
        .map(Json.text(_, "a machine"))
    val waivers =
      Json.items(fields.getOrElse("waivers", Vector()), "waivers").map { w =>
        val f = Json.fields(w, "a waiver")
        def text(name: String) =
          Json.text(f.getOrElse(name, ""), s"a waiver's $name")
        (text("machine"), text("subject"), text("because"))
      }
    (machines.toSet, waivers)

  // The acceptances of a `<file>.lint.json`.
  def read(text: String): Vector[Acceptance] =
    val fields = Json.fields(Json.parse(text), "the accepted findings")
    Json.items(fields.getOrElse("accepted", Vector()), "accepted").map { a =>
      val f = Json.fields(a, "an acceptance")
      def text(name: String) =
        Json.text(f.getOrElse(name, ""), s"an acceptance's $name")
      Acceptance(
        text("kind"),
        text("owner"),
        Json
          .items(f.getOrElse("subjects", Vector()), "subjects")
          .map(Json.text(_, "a subject")),
        text("because")
      )
    }

  // The acceptances as Go's encoder writes them, indented by two spaces, with HTML left unescaped.
  def encode(accepted: Vector[Acceptance]): String =
    def text(s: String) = Json.quoted(s)
    def list(items: Vector[String], indent: String) =
      if items.isEmpty then "[]"
      else
        items
          .map(i => s"$indent  ${text(i)}")
          .mkString("[\n", ",\n", s"\n$indent]")
    val entries = accepted.map { a =>
      s"""    {
         |      "kind": ${text(a.kind)},
         |      "owner": ${text(a.owner)},
         |      "subjects": ${list(a.subjects, "      ")},
         |      "because": ${text(a.because)}
         |    }""".stripMargin
    }
    val body =
      if entries.isEmpty then "[]" else entries.mkString("[\n", ",\n", "\n  ]")
    s"{\n  \"accepted\": $body\n}\n"

@main def run(arguments: String*): Unit =
  sys.exit(Gate.main(arguments, Tools.here, System.out, System.err))

private[check] def modelLoopStatus(
    arguments: Seq[String],
    tools: => Tools,
    out: PrintStream,
    err: PrintStream
): Int =
  if arguments.exists(_ != "--skip-lint") || arguments.size > 1 then
    err.println("usage: modelLoop [--skip-lint]")
    2
  else
    try
      Gate(tools, out).modelLoop(skipLint = arguments.contains("--skip-lint"))
      0
    catch
      case e: (GateError | ToolError | java.io.IOException) =>
        err.println(s"model: ${e.getMessage}")
        1

@main def modelLoop(arguments: String*): Unit =
  sys.exit(modelLoopStatus(arguments, Tools.here, System.out, System.err))

// The change-scoped check over Temporal's coverage registry: `make model-check-changed [BASE=<rev>]`.
@main def changed(arguments: String*): Unit =
  sys.exit(ChangeCheck.main(arguments, Tools.here, System.out, System.err, Coverage))
