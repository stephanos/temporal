// Discovers the framework's one-file samples and checks the complete source-to-answer path. A
// sample is its source filename and an object of the same name whose `queries` section is the lift
// root, with every progress claim the object declares as its own member (`val name: Progress[S] =
// ...`), since no list of Queries holds one; adding a file therefore adds it to this check without
// another registry.
package framework.check

import java.nio.file.{Files, Path}
import java.security.MessageDigest
import scala.jdk.CollectionConverters.*

final case class Sample(
    name: String,
    source: Path,
    root: String,
    ir: Path,
    progress: Vector[String] = Vector.empty
)

object Samples:
  // Each sample owns an independent namespace. Keep its lift and temporary Go adaptation together
  // in one JVM, publishing each successful sample before starting the next.
  // The held Go IR consumer still names Step before the DSL package move. Keep the exported IR
  // current, and adapt only the temporary input that consumer reads until its regeneration.
  private val lifter =
    """import framework.`export`.{LiftRun, Switches}
      |import java.nio.file.{Files, Path}
      |
      |@main def liftSamples(jar: String, classpath: String, args: String*): Unit =
      |  def lift(pending: List[String]): Unit = pending match
      |    case out :: count :: remaining =>
      |      val (roots, tail) = remaining.splitAt(count.toInt)
      |      val status = LiftRun.run(Seq(jar, classpath, out) ++ roots, System.err, Switches.fromEnvironment)
      |      if status != 0 then sys.exit(status)
      |      val ir = Path.of(out)
      |      if Files.isRegularFile(ir) then
      |        val checked = ir.resolveSibling(ir.getFileName.toString.stripSuffix(".json") + ".check.json")
      |        Files.writeString(checked, Files.readString(ir).replace("\"framework.dsl.Step\"", "\"framework.Step\"")): Unit
      |        lift(tail)
      |    case Nil => ()
      |    case _ => throw IllegalArgumentException("incomplete sample lift arguments")
      |  lift(args.toList)
      |""".stripMargin

  private val checker =
    """package main
      |
      |import (
      |  "fmt"
      |  "os"
      |  "strings"
      |
      |  umpirespb "MODULE/api/umpire/v1"
      |  "MODULE/tools/umpire/check"
      |  "MODULE/tools/umpire/ir"
      |)
      |
      |func main() {
      |  failed := false
      |  for _, path := range os.Args[1:] {
      |    model, err := ir.Load(path)
      |    if err != nil {
      |      fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
      |      failed = true
      |      continue
      |    }
      |    report := check.Check(model, check.DefaultScope)
      |    checked, refuted := 0, 0
      |    for _, query := range model.GetQueries() {
      |      want := check.Found
      |      if query.GetForm() == umpirespb.Query_FORM_VERIFY {
      |        want = check.Verified
      |      }
      |      if strings.HasPrefix(query.GetName(), "refutes") {
      |        want = check.Counterexample
      |        refuted++
      |      }
      |      var answer *check.Receipt
      |      for index := range report.Receipts {
      |        receipt := &report.Receipts[index]
      |        if receipt.Subject == check.QuerySubject && receipt.Key.Name == query.GetName() {
      |          answer = receipt
      |          break
      |        }
      |      }
      |      if answer == nil {
      |        fmt.Fprintf(os.Stderr, "%s: Query %s produced no receipt\n", path, query.GetName())
      |        failed = true
      |      } else if answer.Kind != want {
      |        fmt.Fprintf(os.Stderr, "%s: Query %s: got %s, want %s (%s)\n",
      |          path, query.GetName(), answer.Kind, want, answer.Explanation)
      |        failed = true
      |      }
      |      checked++
      |    }
      |    progress := map[string]bool{}
      |    for _, receipt := range report.Receipts {
      |      if receipt.Subject != check.ProgressSubject {
      |        continue
      |      }
      |      progress[receipt.Key.Owner+"."+receipt.Key.Name] = true
      |      if receipt.Kind != check.Verified {
      |        fmt.Fprintf(os.Stderr, "%s: progress %s %s: got %s, want %s (%s)\n",
      |          path, receipt.Key.Owner, receipt.Key.Name, receipt.Kind, check.Verified,
      |          receipt.Explanation)
      |        failed = true
      |      }
      |    }
      |    if checked == 0 {
      |      fmt.Fprintf(os.Stderr, "%s: no Query to check\n", path)
      |      failed = true
      |    } else {
      |      fmt.Printf("%s: checked %d Queries (%d negative controls) and %d progress claims\n",
      |        path, checked, refuted, len(progress))
      |    }
      |  }
      |  if failed { os.Exit(1) }
      |}
      |""".stripMargin

  private def discovered(root: Path, output: Path): Vector[Sample] =
    val directory = root.resolve("model/framework/samples")
    if !Files.isDirectory(directory) then
      throw GateError(s"the framework samples directory is missing: $directory")
    val stream = Files.list(directory)
    val sources =
      try
        stream.iterator.asScala
          .filter(path =>
            Files.isRegularFile(path) && path.toString.endsWith(".scala") &&
              !path.toString.endsWith(".test.scala")
          )
          .toVector
          .sortBy(_.getFileName.toString)
      finally stream.close()
    if sources.isEmpty then throw GateError(s"no Scala sample in $directory")
    sources.map: source =>
      val name = source.getFileName.toString.stripSuffix(".scala")
      if !name.matches("[A-Z][A-Za-z0-9]*") then
        throw GateError(s"sample ${source.getFileName} does not name a Scala object")
      Sample(
        name,
        source,
        s"samples.$name$$.queries",
        output.resolve(name + ".json"),
        progressClaims(Files.readString(source)).map(claim => s"samples.$name$$.$claim")
      )

  // The progress claims a sample object declares as its own members, at its body's indentation.
  private[check] def progressClaims(source: String): Vector[String] =
    """(?m)^  val (\w+): Progress\[""".r.findAllMatchIn(source).map(_.group(1)).toVector

  private def files(root: Path, inputs: Seq[String]): Vector[Path] = inputs
    .flatMap: input =>
      val path = root.resolve(input)
      if Files.isDirectory(path) then
        val stream = Files.walk(path)
        try
          stream.iterator.asScala
            .filter: file =>
              Files.isRegularFile(file) && file.toString.endsWith(".scala") &&
                !root
                  .relativize(file)
                  .iterator
                  .asScala
                  .exists(p => p.toString.startsWith(".") || p.toString == "testdata")
            .toVector
        finally stream.close()
      else Vector(path)
    .distinct
    .sortBy(_.toString)
    .toVector

  private def fingerprint(paths: Seq[Path], settings: Seq[String] = Seq.empty): String =
    val digest = MessageDigest.getInstance("SHA-256")
    def add(bytes: Array[Byte]): Unit =
      digest.update(java.nio.ByteBuffer.allocate(8).putLong(bytes.length.toLong).array())
      digest.update(bytes)
    settings.foreach(value => add(value.getBytes(java.nio.charset.StandardCharsets.UTF_8)))
    for path <- paths do
      add(path.toString.getBytes(java.nio.charset.StandardCharsets.UTF_8))
      add(if Files.isRegularFile(path) then Files.readAllBytes(path) else Array.emptyByteArray)
    digest.digest().map("%02x".format(_)).mkString

  private def cached(directory: Path, key: String, outputs: Seq[Path])(build: => Unit): Unit =
    val stamp = directory.resolve("stamp")
    def complete = outputs.forall(p => Files.isRegularFile(p) && Files.size(p) > 0)
    val current = complete && Files.isRegularFile(stamp) &&
      Files.readString(stamp) == s"$key\n${fingerprint(outputs)}\n"
    if !current then
      Files.createDirectories(directory): Unit
      Files.deleteIfExists(stamp): Unit
      outputs.foreach(p => Files.deleteIfExists(p): Unit)
      build
      if !complete then throw GateError(s"sample build wrote incomplete outputs in $directory")
      Files.writeString(stamp, s"$key\n${fingerprint(outputs)}\n"): Unit

  private def classpathExists(classpath: Path): Boolean =
    if !Files.isRegularFile(classpath) then false
    else
      val printed = Files.readString(classpath).trim
      printed.nonEmpty && printed
        .split(java.io.File.pathSeparator)
        .forall(p => Files.exists(Path.of(p)))

  private def withBuilds(tools: Tools)(
      check: (Path, Path, Path) => Vector[Sample]
  ): Vector[Sample] =
    val root = tools.directory
    val cache = Files.createDirectories(root.resolve("model/build/sample-check"))
    val channel = java.nio.channels.FileChannel.open(
      cache.resolve("lock"),
      java.nio.file.StandardOpenOption.CREATE,
      java.nio.file.StandardOpenOption.WRITE
    )
    try
      val lock = channel.lock()
      try buildAndCheck(tools, cache)(check)
      finally lock.release()
    finally channel.close()

  private def buildAndCheck(tools: Tools, cache: Path)(
      check: (Path, Path, Path) => Vector[Sample]
  ): Vector[Sample] =
    val root = tools.directory
    val versions = tools.buildEnvironment ++ Seq(
      tools.find("scala-cli").toString,
      tools.run("scala-cli", Seq("version")).orFail().output,
      tools.find("java").toString,
      tools.run("java", Seq("-version")).orFail().output
    )
    val samplesBuild = cache.resolve("samples")
    val jar = samplesBuild.resolve("samples.jar")
    val classpath = samplesBuild.resolve("samples.classpath")
    val inputs = ModelSources.framework :+ "model/framework/samples"
    val compile =
      Seq("compile", "--print-class-path", "--compilation-output", jar.toString) ++ inputs
    val sampleKey = fingerprint(
      files(
        root,
        inputs ++ Seq("model/OutputPaths.scala", "model/build/api-scalapb.jar", "mise.toml")
      ),
      versions ++ compile
    )
    if !classpathExists(classpath)
    then Files.deleteIfExists(samplesBuild.resolve("stamp")): Unit
    cached(samplesBuild, sampleKey, Seq(jar, classpath)):
      val compiled = tools.scalaCli(compile, Output.KeptApart).orFail()
      Files.writeString(classpath, compiled.output): Unit
      if !classpathExists(classpath) then
        throw GateError(s"sample compilation wrote an invalid classpath in $classpath")
    val lifterBuild = Files.createDirectories(cache.resolve("lifter"))
    val runner = lifterBuild.resolve("lift-samples.scala")
    val executable = lifterBuild.resolve("lift-samples.jar")
    val pack = Seq(
      "--power",
      "package",
      "model/framework/export",
      runner.toString,
      "--assembly",
      "--preamble=false",
      "--main-class",
      "liftSamples",
      "-f",
      "-o",
      executable.toString
    )
    val liftKey = fingerprint(
      files(
        root,
        Seq(
          "model/framework/export",
          "model/framework/inspect",
          "model/framework/lint",
          "model/framework/check/Tools.scala",
          "model/framework/check/Retaining.test.scala",
          "model/OutputPaths.scala",
          "model/build/ir-scalapb.jar",
          "mise.toml"
        )
      ),
      versions ++ pack :+ lifter
    )
    cached(lifterBuild, liftKey, Seq(executable)):
      Files.writeString(runner, lifter): Unit
      tools.scalaCli(pack).orFail(): Unit
    check(jar, classpath, executable)

  def check(tools: Tools, output: Path): Vector[Sample] = synchronized:
    val root = tools.directory
    Files.createDirectories(output): Unit
    val samples = discovered(root, output)
    for sample <- samples do
      Files.deleteIfExists(sample.ir): Unit
      Files.deleteIfExists(output.resolve(sample.name + ".check.json")): Unit
    withBuilds(tools): (jar, classpath, executable) =>
      tools
        .run(
          "java",
          Seq(
            "-Xmx3g",
            "-jar",
            executable.toString,
            s"$jar=model/",
            classpath.toString
          ) ++ samples.flatMap(sample =>
            Seq(
              sample.ir.toString,
              (sample.progress.size + 1).toString,
              sample.root
            ) ++ sample.progress
          )
        )
        .orFail(): Unit
      for sample <- samples do
        if !Files.isRegularFile(sample.ir) then
          throw GateError(s"the lift of ${sample.name} wrote no ${sample.ir}")
      val runner = output.resolve("check-samples.go")
      val module = "go." + "tem" + "poral.io/server"
      Files.writeString(runner, checker.replace("MODULE", module)): Unit
      tools
        .run(
          "go",
          Seq("run", runner.toString) ++ samples
            .map(sample => output.resolve(sample.name + ".check.json").toString)
        )
        .orFail(): Unit
      samples
