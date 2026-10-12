package umpire.irgen

import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*
import umpire.check.{Scratch, Tools}

// The packaged Models the lifter's suites build their fixtures against and lift beside them:
// model/build/model-scala.jar and its classpath, as the gate packages them (or a FEATURES-narrowed
// build of them). A fixture lift admits only the sources its roots reach (Admission.scala), so a
// feature in the jar that no fixture reaches neither fails nor slows it.
//
// What each suite reads of the Models is its coverage (fn-166.11): a fixture suite reads its own
// fixture and the shared code it reaches; `Mini` suites also lift `features/activity_mini`; a test
// that needs a real feature is held for the owner's sweep (`SweepHeld`) rather than deleted.
private[irgen] object ModelBuild:
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
private[irgen] object SweepHeld:
  val tag = munit.Tag("sweep-held")

  def apply(name: String, reads: String)(using munit.Location): munit.TestOptions =
    munit.TestOptions(s"$name [sweep-held: reads $reads]").tag(tag).ignore

  // The fixture sources held for the sweep, by their path under testdata: each reads a real feature
  // and has no mini twin yet, so a fixture's build leaves it out, and the tests and expected lines
  // of its declarations are held with it.
  val sources: Set[String] = Set("lifts/Scripts.scala", "lifts/ScriptRejects.scala")

  // Whether a fixture's source is held.
  def source(fixture: String, file: String): Boolean = sources(s"$fixture/$file")

// What the lifter's suites read of the Models (fn-166.11), as the registry of every suite's
// coverage declares it (model/check/Coverage.scala, fn-166.13): every suite reads the lifter, the
// framework and its own fixtures under testdata, and most read the shared code through the packaged
// Models; a fixture lift admits only what its roots reach (Admission.scala), so a feature no fixture
// reaches changes no suite's result. The suites below also read the mini feature,
// features/activity_mini: they assert on its lift, or build fixtures that import it. A test of a real
// feature is held for the sweep (`SweepHeld`), not listed.
private[irgen] object SuiteCoverage:
  val mini: Set[String] = umpire.check.Coverage.suites.collect {
    case (name, covers) if name.startsWith("umpire.irgen.") && covers.features("activity_mini") =>
      name
  }.toSet
