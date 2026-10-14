package framework.lint

import java.nio.file.{Files, Path}
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import framework.check.{Ran, RetainedScratch, Tools}
import framework.`export`.{FixtureBuild, InProcess, ModelBuild}
import FixtureBuild.buildFixture

private[lint] trait LintFixtures extends munit.FunSuite with RetainedScratch:
  override val munitTimeout: Duration = 20.minutes
  protected val tools = Tools.here
  protected val root = tools.directory
  protected val build = root.resolve("model/build")
  protected val testdata = root.resolve("model/framework/export/testdata")
  protected val modelJar = ModelBuild.jar
  protected val modelClasspath = ModelBuild.classpath
  protected lazy val scratch = retained(build.resolve("history"), "lint.")
  protected def stored(fixture: String) = s"model/framework/export/testdata/$fixture/"
  protected def materialize(fixture: String): Path =
    ModelBuild.copied(testdata.resolve(fixture), scratch.resolve(fixture))
  protected def materializeTree(
      fixture: String,
      name: String = "",
      excluding: Set[String] = Set.empty
  ): Path =
    LayoutSources.materialize(
      testdata.resolve(fixture),
      scratch.resolve(if name.isEmpty then fixture else name),
      modelJar,
      excluding
    )
  protected def packaged(name: String, sources: Path): Path =
    val jar = scratch.resolve(s"$name.jar")
    tools
      .buildFixture(
        Seq(
          "--power",
          "package",
          "--server=false",
          "--library",
          sources.toString,
          "-f",
          "-o",
          jar.toString
        )
      )
      .orFail(): Unit
    jar
  protected def lift(arguments: String*): Ran =
    InProcess.lift(tools.directory, arguments)
  protected def liftIr(out: Path, jars: String, names: String*): Ran =
    lift((Seq("--ir", jars, modelClasspath.toString, out.toString) ++ names)*)
  protected def refused(ran: Ran): Seq[String] =
    ran.output.linesIterator.filter(_.startsWith("lift:")).toSeq
  protected def listed(directory: Path): Seq[String] =
    if !Files.isDirectory(directory) then Nil
    else
      val stream = Files.list(directory)
      try stream.iterator.asScala.map(_.getFileName.toString).toList.sorted
      finally stream.close()

private[framework] object LayoutSources:
  def materialize(
      from: Path,
      to: Path,
      modelJar: Path,
      excluding: Set[String]
  ): Path =
    Files.createDirectories(to)
    val jar = java.util.regex.Matcher.quoteReplacement(modelJar.toString)
    val stream = Files.walk(from)
    try
      for
        source <- stream.iterator.asScala.toList
        if source.toString.endsWith(".scala")
        if !excluding.exists(from.relativize(source).startsWith(_))
      do
        val copy = to.resolve(from.relativize(source).toString)
        Files.createDirectories(copy.getParent)
        Files.writeString(
          copy,
          Files
            .readString(source)
            .replaceAll("""(\.\./)+build/model-scala\.jar""", jar)
        )
    finally stream.close()
    to
