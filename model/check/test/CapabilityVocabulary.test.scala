package umpire.check

import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*

// The framework and the lifter keep the capability mechanism and name no capability of a feature
// kit's (fn-122.8): Temporal's kinds and their Properties live in model/temporal/capabilities. The
// lifter's fixtures, which lift capability Properties, are not its sources.
class CapabilityVocabularySuite extends munit.FunSuite:
  private val kinds =
    Seq(
      "Closable",
      "Terminable",
      "Pausable",
      "Cancelable",
      "Pollable",
      "Describable",
      "Retries",
      "Deadline"
    )

  private def scala(directory: Path, recursive: Boolean): Vector[Path] =
    val stream = if recursive then Files.walk(directory) else Files.list(directory)
    try stream.iterator.asScala.filter(_.toString.endsWith(".scala")).toVector.sorted
    finally stream.close()

  test("model/umpire and the lifter's sources name no Temporal capability kind") {
    val root = Tools.here.directory
    val files = scala(root.resolve("model/umpire"), recursive = true) ++
      scala(root.resolve("model/irgen"), recursive = false)
    val named = for
      file <- files
      (line, i) <- Files.readAllLines(file).asScala.zipWithIndex
      kind <- kinds if s"\\b$kind\\b".r.findFirstIn(line).isDefined
    yield s"${root.relativize(file)}:${i + 1}: $kind"
    assertEquals(named, Vector.empty)
  }
