package temporal.`export`

import java.nio.file.Path
import scala.jdk.CollectionConverters.*
import framework.`export`.Plugin
import framework.lint.Structure

// Temporal's plug-in to the generic lifter (framework/export/Plugin.scala): the module map of the
// Temporal Models, which the lifter's import lint holds them to (R4, Conventions.scala), and the
// module of a test under model/temporal, which its folders name. Registered as a Java service in
// resources/META-INF/services.
final class TemporalPlugin extends Plugin:
  val root = "temporal"

  private val models = Path.of("model/temporal").toAbsolutePath.normalize

  private def within(pkg: String, parent: String): Boolean =
    pkg == parent || pkg.startsWith(parent + ".")
  private def category(pkg: String): String =
    pkg.split('.').lift(1).getOrElse("")
  private def feature(pkg: String): String =
    pkg.split('.').take(3).mkString(".")
  private def form(pkg: String): Option[String] =
    pkg.split('.').lift(3).filter(Structure.formFolders)

  // A feature's layers, which depend one way, realization -> protocol -> product (fn-160 R9).
  private def layer(pkg: String): Option[Int] =
    pkg.split('.').collectFirst {
      case "product"             => 0
      case "protocol" | "system" => 1
      case "realization"         => 2
    }

  // Within a feature, a layer imports only the layers below it, and only the realization layer
  // reads the Temporal API; the framework reads none of it, and a Model does not read the IR.
  def permitted(from: String, to: String, test: Boolean): Option[Boolean] =
    if from == "umpire.paths" && within(to, root) then Some(false)
    else if !test && within(from, "temporal.features") && layer(from).forall(
        _ < 2
      ) &&
      (within(to, "io.temporal.api") || within(to, "temporal.server.api"))
    then Some(false)
    else if !test && within(from, "temporal.features") && feature(
        to
      ) == feature(from) &&
      layer(from).exists(f => layer(to).exists(_ > f))
    then Some(false)
    else if within(to, "io.temporal.server.api.umpire") then
      Some(
        within(from, "framework.dsl") || within(from, "framework.realize") ||
          within(from, "temporal.realize")
      )
    else if within(to, "temporal.server.api") then Some(!within(from, "framework"))
    else if within(from, "framework") && within(to, "io.temporal") then Some(false)
    else if within(from, root) &&
      (within(to, "framework") || within(to, root) || within(to, "umpire"))
    then
      Some(
        if within(to, "framework") then true
        else if !within(to, root) then false
        else if test then true
        else
          category(from) match
            case "capabilities" | "realize" =>
              category(to) == category(from) || to == root
            case "actors" | "foundations" =>
              feature(to) == feature(from) || to == root
            case "features" =>
              Set("capabilities", "realize", "actors", "foundations")(
                category(to)
              ) ||
              to == root ||
              (feature(to) == feature(from) && (form(to).isEmpty || form(
                to
              ) == form(from)))
            case _ => to == root
      )
    else None

  // A test under model/temporal is held to the module of its folder, whatever its package.
  def testModule(file: Path): Option[String] =
    val path = file.toAbsolutePath.normalize
    Option.when(path.startsWith(models)):
      (root +: models
        .relativize(path.getParent)
        .iterator
        .asScala
        .map(_.toString)
        .filter(_.nonEmpty)
        .toList).mkString(".")
