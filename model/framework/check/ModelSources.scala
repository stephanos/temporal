package framework.check

import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*
import umpire.paths.OutputPaths

final class ModelSources(root: Path, selection: Option[String]):
  private val temporal = root.resolve("model/temporal")
  private val featureRoot = temporal.resolve("features")
  private def children(path: Path): Vector[Path] =
    if !Files.isDirectory(path) then Vector.empty
    else
      val stream = Files.list(path)
      try stream.iterator.asScala.toVector.sortBy(_.toString)
      finally stream.close()
  private def scalaFiles(path: Path): Vector[Path] = children(path)
    .filter(p => Files.isRegularFile(p) && p.toString.endsWith(".scala"))
  private def featureFolder(path: Path): Boolean =
    val name = path.getFileName.toString.toLowerCase(java.util.Locale.ROOT)
    Files.isDirectory(path) && !name.startsWith(".") &&
    !Set("shared", "product", "protocol", "generated").contains(name)
  private def hasFeatureFile(path: Path): Boolean = scalaFiles(path).exists: file =>
    def normalized(name: String): String =
      name.replace("_", "").toLowerCase(java.util.Locale.ROOT)
    val name = normalized(file.getFileName.toString.stripSuffix(".scala"))
    val folder = path.getFileName.toString
    val parts = folder.split("_", 2)
    !file.toString.endsWith(".test.scala") &&
    (name == normalized(folder) ||
      (path.getParent.equals(featureRoot) && parts.length == 2 &&
        name == normalized(parts(1) + parts(0))) ||
      (!path.getParent.equals(featureRoot) &&
        name == normalized(folder + path.getParent.getFileName.toString)))
  val discovered: Vector[String] = children(featureRoot)
    .filter(featureFolder)
    .flatMap: kind =>
      val forms = children(kind).filter(p => featureFolder(p) && hasFeatureFile(p))
      (if forms.nonEmpty then forms else Vector(kind).filter(hasFeatureFile))
        .map(p => featureRoot.relativize(p).toString)
  val features: Vector[String] =
    selection.fold(discovered)(_.split("\\s+").filter(_.nonEmpty).toVector.distinct)
  for feature <- features if !discovered.contains(feature) do
    throw GateError(
      s"feature $feature does not exist or has no feature file under model/temporal/features"
    )
  val narrowed: Boolean = features.toSet != discovered.toSet
  private val selectedPaths = features.map(featureRoot.resolve(_))
  private val flatKinds = features
    .filterNot(_.contains('/'))
    .flatMap: feature =>
      feature
        .split('_')
        .headOption
        .filter(_ != feature)
        .map(featureRoot.resolve(_))
        .filter(Files.isDirectory(_))
  private val ancestors =
    (selectedPaths.map(_.getParent).filterNot(_.equals(featureRoot)) ++ flatKinds).distinct
  private val shared = ancestors
    .flatMap(children)
    .filter(p =>
      Files.isDirectory(p) && !p.getFileName.toString.startsWith(".") && !discovered
        .exists(f => featureRoot.resolve(f).equals(p))
    )
  private val modelPaths = scalaFiles(temporal) ++
    Seq("foundations", "actors", "capabilities", "realize").map(temporal.resolve(_)) ++
    ancestors.flatMap(scalaFiles) ++ shared ++ selectedPaths
  val sources: Seq[String] = ModelSources.framework ++
    modelPaths.distinct.map(root.relativize(_).toString)
  private val unlisted =
    discovered.filterNot(features.contains).map(f => "temporal.features." + f.replace('/', '.'))
  private val candidateTests = modelPaths.flatMap: path =>
    if Files.isRegularFile(path) then
      if path.toString.endsWith(".test.scala") then Vector(path) else Vector.empty
    else if Files.isDirectory(path) then
      val stream = Files.walk(path)
      try
        stream.iterator.asScala
          .filter(p => Files.isRegularFile(p) && p.toString.endsWith(".test.scala"))
          .toVector
      finally stream.close()
    else Vector.empty
  private val excluded =
    candidateTests.distinct.filter(p => unlisted.exists(Files.readString(p).contains))
  val options: Seq[String] = excluded.flatMap(p => Seq("--exclude", p.toAbsolutePath.toString))

  def includes(path: Path): Boolean =
    val absolute = path.toAbsolutePath.normalize
    !excluded.exists(_.toAbsolutePath.normalize.equals(absolute)) &&
    (!absolute.startsWith(featureRoot.toAbsolutePath.normalize) ||
      modelPaths.exists(p => absolute.startsWith(p.toAbsolutePath.normalize)))

  def states(lifted: Path): Vector[(String, String)] = features.map: feature =>
    val logical = feature.replace('/', '-').replace('_', '-')
    val name = logical + ".json"
    val before = OutputPaths.ir(root, logical)
    val after = lifted.resolve(name)
    val state =
      if !Files.isRegularFile(before) then "missing"
      else if Files.isRegularFile(after) && java.util.Arrays
          .equals(Files.readAllBytes(before), Files.readAllBytes(after))
      then "current"
      else "behind"
    feature -> state

object ModelSources:
  // The tools beside the Models, under model/framework and model/temporal: the lifter (export), the
  // checks and the migration harness. They are no Model source, so the Models' rules skip them.
  val tools: Seq[String] = Seq("export", "check", "migrate")

  // Whether a path, relative to the repository, is in one of the tools' directories.
  def tool(path: String): Boolean =
    tools.exists(t => path.startsWith(s"model/framework/$t/") || path.startsWith(s"model/temporal/$t/"))

  // The build settings and the framework's sources the Models compile against: the DSL and the
  // generic realization core, not the tools beside them (export, check, migrate).
  val framework: Seq[String] =
    Seq("model/project.scala", "model/framework/dsl", "model/framework/realize")
