package framework.`export`

import java.nio.charset.StandardCharsets.{US_ASCII, UTF_8}
import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*
import framework.check.{Scratch, Tools}
import FixtureBuild.buildFixture

// The packaged Models the lifter's suites build their fixtures against and lift beside them:
// model/build/model-scala.jar and its classpath, as the gate packages them (or a FEATURES-narrowed
// build of them). A fixture lift admits only the sources its roots reach (Admission.scala), so a
// feature in the jar that no fixture reaches neither fails nor slows it.
//
// What each suite reads of the Models is its coverage (fn-166.11): a fixture suite reads its own
// fixture and the shared code it reaches; `Mini` suites also lift `features/activity_mini`; a test
// that needs a real feature is held for the owner's sweep (`SweepHeld`) rather than deleted.
private[framework] object ModelBuild:
  private val tools = Tools.here
  val directory: Path = tools.directory.resolve("model/build")
  val jar: Path = directory.resolve("model-scala.jar")

  // The packaged Models' classpath with the jar in place of the Models' build directory, which a
  // later compile of model/ may have changed, as a file `lift` reads and as entries to build a
  // fixture with `--classpath`.
  lazy val classpathEntries: String =
    val written = Files.readString(directory.resolve("model-scala.classpath")).trim
    s"$jar${java.io.File.pathSeparator}${written.substring(written.indexOf(java.io.File.pathSeparatorChar) + 1)}"
  lazy val classpath: Path =
    Files.writeString(Scratch.directory("model-scala.").resolve("classpath"), classpathEntries)

  // The `jar=prefix` argument of the packaged Models, after a fixture's own.
  def jarArgument: String = s"$jar=model/"

  // The framework alone supplies definitions to isolated fixtures without adding Model roots.
  lazy val frameworkJarArgument: String =
    val out = Scratch.directory("framework-core.").resolve("framework.jar")
    tools.buildFixture(Seq("--power", "package", "--server=false", "--library",
      "model/project.scala", "model/framework/dsl", "model/framework/realize", "-f", "-o", out.toString)).orFail(): Unit
    s"$out=model/"

  // A copy of a fixture's stored sources, without the build state an earlier build left beside them
  // (a stale `.scala-build` there can keep TASTy compiled against an older framework). Its
  // project.scala's relative jar paths are resolved; another source's directives are blanked to
  // the same length, since its project.scala states them too, so every offset of the copy is the
  // stored file's and a lift maps the copy's positions back to the stored files.
  def copied(stored: Path, to: Path): Path =
    val tree = Files.walk(stored)
    try
      for
        from <- tree.iterator.asScala.toList
        if Files.isRegularFile(from) && !stored.relativize(from).toString.startsWith(".scala-build")
      do
        val target = to.resolve(stored.relativize(from).toString)
        Files.createDirectories(target.getParent): Unit
        val text = Files.readString(from)
        Files.writeString(
          target,
          if from.getFileName.toString == "project.scala" then
            text
              .replaceAll(
                """(\.\./)+build/model-scala\.jar""",
                java.util.regex.Matcher.quoteReplacement(jar.toString)
              )
              .replaceAll(
                """(\.\./)+build/api-scalapb\.jar""",
                java.util.regex.Matcher
                  .quoteReplacement(directory.resolve("api-scalapb.jar").toString)
              )
          else
            text.linesWithSeparators
              .map(line =>
                if line.startsWith("//> using") then
                  "//" + line.drop(2).map(c => if c == '\n' then c else ' ')
                else line
              )
              .mkString
        ): Unit
    finally tree.close()
    to

  // The mini feature's IR file, lifted once per JVM from the packaged Models, with its waivers.
  lazy val mini: Path =
    val out = Scratch.directory("mini-lift.")
    val ran = InProcess.lift(
      tools.directory,
      Seq("--ir", jarArgument, classpath.toString, out.toString, "activity-mini")
    )
    ran.orFail(): Unit
    out

  // The files a lift of every IR file the packaged Models declare writes, IR and waivers.
  lazy val files: Seq[String] =
    val out = Scratch.directory("models-lift.")
    InProcess
      .lift(tools.directory, Seq("--ir", jarArgument, classpath.toString, out.toString))
      .orFail(): Unit
    val listed = Files.list(out)
    try listed.iterator.asScala.map(_.getFileName.toString).toList.sorted
    finally listed.close()

// A test held for the owner's sweep (fn-166.11): it reads a real feature, which the mini-only scope
// leaves behind, so it is reported as ignored under its name, the feature it reads and this tag, and
// kept, assertions and all, for the sweep to restore by dropping the hold.
private[framework] object SweepHeld:
  val tag = munit.Tag("sweep-held")

  def apply(name: String, reads: String)(using munit.Location): munit.TestOptions =
    munit.TestOptions(s"$name [sweep-held: reads $reads]").tag(tag).ignore

  // The fixture sources held for the sweep, by their path under testdata: each reads a real feature
  // and has no mini twin yet, so a fixture's build leaves it out, and the tests and expected lines
  // of its declarations are held with it.
  val sources: Set[String] = Set("lifts/Scripts.scala", "lifts/ScriptRejects.scala")

  // Whether a fixture's source is held.
  def source(fixture: String, file: String): Boolean = sources(s"$fixture/$file")

// A fixture's build in the tests' own JVM (fn-166.14), with scala-cli's command line and what it
// prints: `tools.buildFixture(arguments)` stands for `tools.scalaCli(arguments)`. A
// `--server=false` build used to start a scala-cli and a compiler JVM of its own, cold; here the
// compiler of the lifter's own classpath (scala3-compiler, beside the TASTy inspector) compiles the
// fixture, warm after the first build, and a build of the same sources, options and classpath is
// compiled once per run and shared by every test that asks for it, wherever its sources were
// copied.
//
// It reads the command lines the suites write: `package --library [--test] [--classpath <cp>]
// [--jar <jar>]... <directory> -f -o <jar>` and `compile` of the same, with `--server=false`, and
// the directives the fixtures state in one file (`scala 3.9.0`, `jvm`, `options`, `jar`, the
// Models' one `dep`). The sources and their scope are scala-cli's: the directory's `.scala` files
// outside hidden directories, the `.test.scala` ones only with `--test`; the compiler is given
// scala-cli's workspace as its `-sourceroot`, so the classes and TASTy are those scala-cli writes,
// and it prints the compiler's own report, as scala-cli's compiler JVM does, so a test reads a
// refusal as it read scala-cli's. Any other command line (a build through scala-cli's Bloop server,
// which reports in a form of its own, among them), directive or source kind is left to scala-cli.
object FixtureBuild:
  // `tools.buildFixture(arguments)` is `tools.scalaCli(arguments)`, built in this JVM where it can
  // be. A suite imports it, `import FixtureBuild.buildFixture`: a top-level extension here would
  // make a `ModelBuild.test` package object, whose name the classpath cannot hold.
  extension (tools: Tools)
    def buildFixture(arguments: Seq[String]): framework.check.Ran = scalaCli(tools, arguments)

  final private case class Request(
      command: String,
      scala: Boolean,
      test: Boolean,
      workspace: Path,
      sources: Seq[Path],
      options: Seq[String],
      classpath: Seq[String],
      out: Option[Path]
  )

  // A compile: its classes, whether it failed, and the compiler's report, which names the sources
  // under `workspace`.
  final private case class Compiled(
      classes: Path,
      failed: Boolean,
      report: String,
      workspace: Path
  ):
    lazy val jar: Path =
      val jar = classes.resolveSibling("fixture.jar")
      FixtureBuild.jar(classes, jar)
      jar

  // A compile shared by every test asking for it, run by the first to ask.
  final private class Shared(compile: () => Compiled):
    lazy val compiled: Compiled = compile()

  private val shared = java.util.concurrent.ConcurrentHashMap[String, Shared]()

  // The compiles run side by side in the tests' heap, beside their lifts (InProcess), so at most
  // `slots` run at once.
  private val slots = java.util.concurrent.Semaphore(3, true)

  private val version = "3.9.0"
  private val runtime = "com.thesamet.scalapb::scalapb-runtime-grpc:0.11.20"

  // The classpath of the Models' build, by what scala-cli adds for a directive: the Scala library
  // for every build, and the Models' one dependency, the scalapb runtime, with what it pulls in.
  private lazy val packaged: Seq[String] =
    ModelBuild.classpathEntries.split(java.io.File.pathSeparator).toSeq.drop(1)
  private lazy val library: Seq[String] = packaged.filter(entry =>
    val name = Path.of(entry).getFileName.toString
    name == s"scala3-library_3-$version.jar" || name == s"scala-library-$version.jar"
  )
  private lazy val dependency: Seq[String] = packaged.filterNot(entry =>
    library.contains(entry) || Path.of(entry).getFileName.toString == "api-scalapb.jar"
  )

  def scalaCli(tools: Tools, arguments: Seq[String]): framework.check.Ran =
    request(tools.directory, arguments) match
      case Some(asked) => built(tools.directory, arguments, asked)
      case None        => tools.scalaCli(arguments)

  // The command line's request, when it is one this build reads.
  private def request(directory: Path, arguments: Seq[String]): Option[Request] =
    val ignored = Set("--power", "--server=false", "--suppress-outdated-dependency-warning")
    arguments.filterNot(ignored) match
      case command +: rest
          if (command == "package" || command == "compile") && arguments.contains(
            "--server=false"
          ) =>
        flags(rest, Request(command, false, false, directory, Nil, Nil, Nil, None), Nil)
          .flatMap((asked, inputs) =>
            inputs match
              case Seq(input) =>
                val path = directory.resolve(input).toAbsolutePath.normalize
                val found =
                  if Files.isDirectory(path) then sources(path, asked.test).map((path, _))
                  else if Files.isRegularFile(path) && input.endsWith(".scala") then
                    Some((path.getParent, Seq(path)))
                  else None
                found.flatMap((workspace, found) =>
                  directives(found, asked.copy(workspace = workspace, sources = found))
                )
              case _ => None
          )
          .filter(asked => asked.scala && (asked.command == "compile" || asked.out.isDefined))
      case _ => None

  // The flags of a command line, which `request` reads in the directory it is run in.
  @scala.annotation.tailrec
  private def flags(
      rest: Seq[String],
      asked: Request,
      inputs: Seq[String]
  ): Option[(Request, Seq[String])] =
    rest match
      case Seq()                                             => Some((asked, inputs))
      case "--library" +: more if asked.command == "package" => flags(more, asked, inputs)
      case "-f" +: more if asked.command == "package"        => flags(more, asked, inputs)
      case "--test" +: more => flags(more, asked.copy(test = true), inputs)
      case "--scala" +: stated +: more if stated == version =>
        flags(more, asked.copy(scala = true), inputs)
      case "--jvm" +: _ +: more                              => flags(more, asked, inputs)
      case "-o" +: out +: more if asked.command == "package" =>
        flags(more, asked.copy(out = Some(asked.workspace.resolve(out))), inputs)
      case "--classpath" +: entries +: more =>
        flags(
          more,
          asked.copy(classpath =
            asked.classpath ++ entries.split(java.io.File.pathSeparator).filter(_.nonEmpty)
          ),
          inputs
        )
      case "--jar" +: jar +: more =>
        flags(
          more,
          asked.copy(classpath = asked.classpath :+ asked.workspace.resolve(jar).toString),
          inputs
        )
      case flag +: _ if flag.startsWith("-") => None
      case input +: more                     => flags(more, asked, inputs :+ input)

  // A directory's sources as scala-cli reads them, or none when it holds a kind this build leaves
  // to scala-cli (a script, a Java source, a `test` directory).
  private def sources(workspace: Path, test: Boolean): Option[Seq[Path]] =
    val tree = Files.walk(workspace)
    try
      val files = tree.iterator.asScala
        .filter(Files.isRegularFile(_))
        .map(workspace.relativize)
        .filterNot(_.iterator.asScala.exists(_.toString.startsWith(".")))
        .toList
      val names = files.map(_.toString)
      if files.exists(_.iterator.asScala.exists(_.toString == "test")) ||
        names.exists(name => name.endsWith(".sc") || name.endsWith(".java"))
      then None
      else
        Some(
          names
            .filter(name => name.endsWith(".scala") && (test || !name.endsWith(".test.scala")))
            .sorted
            .map(workspace.resolve)
        )
    finally tree.close()

  // The sources' directives, read into the request, or none when one is not a directive this build
  // reads.
  private def directives(sources: Seq[Path], asked: Request): Option[Request] =
    val stated = for
      source <- sources
      line <- Files.readAllLines(source).asScala
      if line.startsWith("//> using ")
    yield (source, words(line.stripPrefix("//> using ")))
    // scala-cli warns of directives in more than one file.
    if stated.map(_._1).distinct.size > 1 then None
    else
      stated.foldLeft(Option(asked)) {
        case (None, _)                           => None
        case (Some(so), (source, key +: values)) =>
          key match
            case "scala" if values == Seq(version) => Some(so.copy(scala = true))
            case "jvm"                             => Some(so)
            case "options" | "option"              => Some(so.copy(options = so.options ++ values))
            case "dep" | "deps" if values == Seq(runtime) =>
              Some(so.copy(classpath = so.classpath ++ dependency.filterNot(so.classpath.contains)))
            case "jar" | "jars" =>
              Some(
                so.copy(classpath =
                  so.classpath ++ values.map(source.getParent.resolve(_).normalize.toString)
                )
              )
            case _ => None
        case (Some(_), _) => None
      }

  // A directive's values: words apart, or a quoted one whole.
  private def words(text: String): Seq[String] =
    """"([^"]*)"|(\S+)""".r
      .findAllMatchIn(text)
      .map(m => Option(m.group(1)).getOrElse(m.group(2)))
      .toSeq

  private def built(directory: Path, arguments: Seq[String], asked: Request): framework.check.Ran =
    val classpath = (library ++ asked.classpath).distinct
    val key = digest(asked, classpath)
    val compiled =
      shared.computeIfAbsent(key, _ => Shared(() => compile(asked, classpath))).compiled
    val printed = compiled.report
      .replace(compiled.workspace.toString, asked.workspace.toString)
      .linesIterator
      .toSeq ++ (if compiled.failed then Seq("Compilation failed") else Nil)
    val wrote = asked.out.filter(_ => !compiled.failed).map { out =>
      Files.createDirectories(out.getParent): Unit
      Files.copy(compiled.jar, out, java.nio.file.StandardCopyOption.REPLACE_EXISTING): Unit
      s"Wrote $out"
    }
    framework.check.Ran(
      "scala-cli" +: arguments,
      directory,
      if compiled.failed then 1 else 0,
      (printed ++ wrote).mkString("", "\n", "\n")
    )

  // What a build depends on: its scope, options, classpath (each entry with its size and time) and
  // its sources by their path in the workspace, not by where the workspace was copied.
  private def digest(asked: Request, classpath: Seq[String]): String =
    val hash = java.security.MessageDigest.getInstance("SHA-256")
    def add(text: String): Unit = hash.update((text + "\u0000").getBytes(UTF_8))
    add(asked.test.toString)
    asked.options.foreach(add)
    add("classpath")
    for entry <- classpath do
      val path = Path.of(entry)
      add(entry)
      if Files.isRegularFile(path) then
        add(s"${Files.size(path)}:${Files.getLastModifiedTime(path).toMillis}")
    add("sources")
    for source <- asked.sources do
      add(asked.workspace.relativize(source).toString)
      hash.update(Files.readAllBytes(source))
      add("")
    java.util.HexFormat.of().formatHex(hash.digest())

  private def compile(asked: Request, classpath: Seq[String]): Compiled =
    val classes = Files.createDirectories(Scratch.directory("fixture-build.").resolve("classes"))
    val printed = java.io.StringWriter()
    val reporter =
      dotty.tools.dotc.reporting.ConsoleReporter(Console.in, java.io.PrintWriter(printed, true))
    val arguments = asked.options ++ Seq(
      "-sourceroot",
      asked.workspace.toString,
      "-classpath",
      classpath.mkString(java.io.File.pathSeparator),
      "-d",
      classes.toString
    ) ++ asked.sources.map(_.toString)
    slots.acquire()
    try dotty.tools.dotc.Driver().process(arguments.toArray, reporter): Unit
    finally slots.release()
    Compiled(
      classes,
      reporter.hasErrors,
      // The compiler JVM scala-cli starts prints in ASCII, a `?` for any other character.
      String(
        printed.toString.replaceAll("\u001b\\[[0-9;]*m", "").getBytes(US_ASCII),
        US_ASCII
      ),
      asked.workspace
    )

  // A library jar of the compiled classes, as `package --library` writes one.
  private def jar(classes: Path, jar: Path): Unit =
    val manifest = java.util.jar.Manifest()
    manifest.getMainAttributes.put(java.util.jar.Attributes.Name.MANIFEST_VERSION, "1.0"): Unit
    val written = java.util.jar.JarOutputStream(Files.newOutputStream(jar), manifest)
    val tree = Files.walk(classes)
    try
      for file <- tree.iterator.asScala.filter(Files.isRegularFile(_)).toList.sortBy(_.toString) do
        written.putNextEntry(java.util.jar.JarEntry(classes.relativize(file).toString))
        written.write(Files.readAllBytes(file))
        written.closeEntry()
    finally
      tree.close()
      written.close()
