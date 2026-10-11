package umpire.check

import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*

final private[check] class ModelSources(root: Path, selection: Option[String]):
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
  private def hasFeatureFile(path: Path): Boolean = scalaFiles(path).exists: file =>
    val name = file.getFileName.toString.stripSuffix(".scala")
    val folder = path.getFileName.toString
    !file.toString.endsWith(".test.scala") &&
    name.equalsIgnoreCase(folder + path.getParent.getFileName.toString)
  val discovered: Vector[String] = children(featureRoot)
    .filter(Files.isDirectory(_))
    .flatMap: kind =>
      val forms = children(kind).filter(p => Files.isDirectory(p) && hasFeatureFile(p))
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
  private val ancestors = selectedPaths.map(_.getParent).filterNot(_.equals(featureRoot)).distinct
  private val shared = ancestors
    .flatMap(children)
    .filter(p =>
      Files.isDirectory(p) && !p.getFileName.toString.startsWith(".") && !discovered
        .exists(f => featureRoot.resolve(f).equals(p))
    )
  private val modelPaths = scalaFiles(temporal) ++
    Seq("foundations", "actors", "capabilities", "realize").map(temporal.resolve(_)) ++
    ancestors.flatMap(scalaFiles) ++ shared ++ selectedPaths
  val sources: Seq[String] = Seq("model/project.scala", "model/framework") ++
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

  def states(held: Path, lifted: Path): Vector[(String, String)] = features.map: feature =>
    val name = feature.replace('/', '-') + ".json"
    val before = held.resolve(name)
    val after = lifted.resolve(name)
    val state =
      if !Files.isRegularFile(before) then "missing"
      else if Files.isRegularFile(after) && java.util.Arrays
          .equals(Files.readAllBytes(before), Files.readAllBytes(after))
      then "current"
      else "behind"
    feature -> state
