package umpire.paths

import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*

object OutputPaths:
  private def roots(root: Path): Vector[Path] =
    Vector("features", "foundations", "actors").map(kind => root.resolve(s"model/temporal/$kind"))

  private def logical(base: Path, owner: Path): String =
    base.relativize(owner).iterator.asScala.map(_.toString.replace('_', '-')).mkString("-")

  private def owners(root: Path): Vector[(String, Path)] =
    roots(root).flatMap: base =>
      if !Files.isDirectory(base) then Vector.empty
      else
        val stream = Files.walk(base)
        try
          stream.iterator.asScala
            .filter(path =>
              Files.isDirectory(path) && !path.equals(base) &&
                !base.relativize(path).iterator.asScala.exists(_.toString.equals("generated"))
            )
            .map(path => logical(base, path) -> path)
            .toVector
        finally stream.close()

  private def owner(root: Path, name: String): (String, Path) =
    require(name.matches("[a-zA-Z0-9]+(?:[-_][a-zA-Z0-9]+)*"), s"invalid logical model name: $name")
    owners(root)
      .filter((logical, _) => name.equals(logical) || name.startsWith(logical + "-"))
      .sortBy((logical, path) => (-logical.length, path.toString))
      .headOption
      .getOrElse(name -> root.resolve(s"model/temporal/features/${name.replace('-', '_')}"))

  def generated(root: Path, name: String): Path = owner(root, name)._2.resolve("generated")

  def ir(root: Path, name: String): Path =
    val (logical, directory) = owner(root, name)
    directory
      .resolve("generated")
      .resolve(
        if name.equals(logical) then "ir.json" else s"legacy/$name.json"
      )

  def lint(root: Path, name: String): Path =
    val file = ir(root, name)
    file.resolveSibling(file.getFileName.toString.stripSuffix(".json") + ".lint.json")

  def mainCases(root: Path, name: String): Path = generated(root, name)

  def legacyCases(root: Path, name: String): Path = generated(root, name).resolve("legacy")

  def inventory(root: Path): Vector[(String, Path)] =
    owners(root)
      .flatMap: (name, directory) =>
        val generated = directory.resolve("generated")
        val main = generated.resolve("ir.json")
        val legacy = generated.resolve("legacy")
        val supplemental =
          if !Files.isDirectory(legacy) then Vector.empty
          else
            val stream = Files.list(legacy)
            try
              stream.iterator.asScala
                .filter(path =>
                  Files.isRegularFile(path) && path.getFileName.toString.endsWith(".json") &&
                    !path.getFileName.toString.endsWith(".lint.json") &&
                    !path.getFileName.toString.endsWith(".waivers.json") &&
                    !path.getFileName.toString.endsWith("-case.json") &&
                    !path.getFileName.toString.endsWith(".case.json") &&
                    !path.getFileName.toString.equals("manifest.json")
                )
                .map(path => path.getFileName.toString.stripSuffix(".json") -> path)
                .toVector
            finally stream.close()
        Option.when(Files.isRegularFile(main))(name -> main).toVector ++ supplemental
      .sortBy(_._1)
