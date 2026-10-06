package umpire.check

import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*

// Comments in the model are `//` lines: no `/* */` and no scaladoc `/** */`. `make lint-model` holds
// every Scala file under model/ to it, its tests and fixtures too (`gate --check-comments`). The
// source is read with ProtoLiterals' scanner, so a `/*` in a string literal is no comment.
private[check] object CommentRule:
  // The line each block comment of a source opens on.
  def blocks(source: String): Vector[Int] =
    ProtoLiterals
      .literals(source)
      .comments
      .collect:
        case (start, _) if source.startsWith("/*", start) =>
          source.substring(0, start).count(_ == '\n') + 1

  // Build output is no source, and model/build is large enough that the walk must not enter it.
  private val skipped = Set("build", ".scala-build", ".bsp")

  private def sources(directory: Path): Vector[Path] =
    val stream = Files.list(directory)
    val entries =
      try stream.iterator.asScala.toVector
      finally stream.close()
    entries
      .sortBy(_.toString)
      .flatMap: path =>
        if Files.isDirectory(path) then
          if skipped(path.getFileName.toString) then Vector.empty else sources(path)
        else if path.getFileName.toString.endsWith(".scala") then Vector(path)
        else Vector.empty

  // What breaks the rule in the repository at `root`, each as `file:line: reason`, in order.
  def findings(root: Path): Vector[String] =
    val model = root.resolve("model")
    if !Files.isDirectory(model) then Vector.empty
    else
      sources(model).flatMap: path =>
        val file = root.relativize(path).iterator.asScala.mkString("/")
        blocks(Files.readString(path)).map: line =>
          s"$file:$line: comment rule: write the comment as // lines, not /* */ or /** */"
