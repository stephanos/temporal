package framework.check
package lint

import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*

// The framework's boundary (fn-169.1): model/framework is self-contained and Temporal-free, so it can
// serve another system. A system's checks name the package roots that are its own (`systems`), and
// every Scala file under model/framework, its tests too but not the fixtures under testdata, is
// held to three rules:
//
//   - its code declares, imports and names no package of a system: no `package temporal...`, no
//     `import temporal...`, no `io.temporal.api.X` written out, outside comments and strings;
//   - its build directives take no jar but the IR's classes (`build/ir-scalapb.jar`), so no
//     system's API is on a framework build's classpath;
//   - its build directives take no file from outside model/framework but the shared
//     model/OutputPaths.scala, so no system's source is either.
//
// The framework's modules also compile on their own, without any system's half (`make
// check-model-boundary`). What the framework still names of Temporal as text, the lifter's kit and
// layout names, is recorded in model/README.md ("Residual Temporal coupling") for fn-169.2.
object Boundary:
  // The jars a framework build may take, by their path from the model's root.
  val jars: Set[String] = Set("build/ir-scalapb.jar")

  // The files outside model/framework a framework build may take, from the model's root.
  val shared: Set[String] = Set("OutputPaths.scala")

  private val directive = "^//> using (jar|file|files|resourceDir|resourceDirs) (.+)$".r

  private def sources(root: Path): Vector[Path] =
    val base = root.resolve("model/framework")
    if !Files.isDirectory(base) then Vector.empty
    else
      val skipped = Set("testdata", "build", ".scala-build", ".bsp")
      val stream = Files.walk(base)
      try
        stream.iterator.asScala
          .filter: path =>
            Files.isRegularFile(path) && path.getFileName.toString.endsWith(".scala") &&
            !base.relativize(path).iterator.asScala.toSeq.init.exists(p => skipped(p.toString))
          .toVector
          .sortBy(_.toString)
      finally stream.close()

  // What crosses the boundary in the repository at `root`, each as `file:line: reason`, in order.
  def findings(root: Path, systems: Seq[String]): Vector[String] =
    val model = root.resolve("model").normalize
    val framework = model.resolve("framework")
    val named = systems.map(java.util.regex.Pattern.quote).mkString("|")
    val reference = s"(?<![\\w.])(?:_root_\\.)?(?:$named)(?:\\.|\\b)".r
    sources(root).flatMap: path =>
      val file = root.relativize(path).iterator.asScala.mkString("/")
      val source = Files.readString(path)
      val code = Literals.code(source, Literals.literals(source))
      def lineAt(offset: Int) = source.substring(0, offset).count(_ == '\n') + 1
      val names =
        if systems.isEmpty then Vector.empty
        else
          reference
            .findAllMatchIn(code)
            .map: found =>
              s"$file:${lineAt(found.start)}: boundary: the framework names " +
                s"`${found.matched.stripSuffix(".")}`, a package of a system; it names no system"
            .toVector
            .distinct
      val directives = source.linesIterator.zipWithIndex.toVector.flatMap: (line, i) =>
        line.trim match
          case directive(kind, values) =>
            values.trim.split("\\s+").toVector.flatMap: value =>
              val target = path.getParent.resolve(value).normalize
              val inModel = target.startsWith(model)
              val relative = if inModel then model.relativize(target).toString else value
              val allowed =
                if kind == "jar" then inModel && jars(relative)
                else target.startsWith(framework) || (inModel && shared(relative))
              Option.unless(allowed)(
                s"$file:${i + 1}: boundary: the framework's build takes $value, from outside " +
                  "the framework; it takes no system's jar or source"
              )
          case _ => Vector.empty
      names ++ directives
